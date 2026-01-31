package purchaseService

import (
	"encoding/json"
	"errors"
	goodsReceiveService "prime-erp-core/external/goods-receive-service"
	"prime-erp-core/internal/models"
	purchaseRepository "prime-erp-core/internal/repositories/purchase"
	prePurchaseService "prime-erp-core/internal/services/pre-purchase-service"
	"time"

	"github.com/gin-gonic/gin"
)

func GetPO(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := models.GetPurchaseRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	purchases, total, page, pageSize, totalPage, err := purchaseRepository.GetPurchaseList(
		req.PurchaseCodes,
		req.SupplierCodes,
		req.StatusApprove,
		req.StatusPayment,
		req.StatusPaymentIncomplete,
		req.ProductCodes,
		req.PurchaseType,
		req.DocRef,
		req.TradingRef,
		req.CompanyCode,
		req.SiteCode,
		req.Page,
		req.PageSize,
	)
	if err != nil {
		return nil, errors.New("failed to get purchase list: " + err.Error())
	}

	result := models.GetPurchaseResponse{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPage,
	}

	if len(purchases) == 0 {
		result.DataList = []models.PurchaseResponse{}
		return result, nil
	}

	purchaseCodes := []string{}
	prePurchaseCodes := []string{}
	for _, purchase := range purchases {
		purchaseCodes = append(purchaseCodes, purchase.PurchaseCode)

		if purchase.PurchaseType == "PRE" && purchase.DocRef != nil {
			prePurchaseCodes = append(prePurchaseCodes, *purchase.DocRef)
		}
	}

	// Get Approvals
	approvalsResp, err := prePurchaseService.GetPOApproval(ctx, purchaseCodes)
	if err != nil {
		return nil, errors.New("failed to get purchase approvals: " + err.Error())
	}

	mapStatusApprove := map[string]string{}
	for _, approval := range approvalsResp {
		mapStatusApprove[approval.DocumentCode] = approval.Status
	}

	// Get PrePurchase
	prePurchaseReq := models.GetPOBigLotListRequest{
		PrePurchaseCodes: prePurchaseCodes,
		CompanyCode:      req.CompanyCode,
		SiteCode:         req.SiteCode,
		Page:             1,
		PageSize:         len(prePurchaseCodes),
	}

	mapPrePurchase, err := GetRelatedPrePurchase(ctx, prePurchaseReq)
	if err != nil {
		return nil, errors.New("failed to get pre purchase list: " + err.Error())
	}

	// Create Result
	for _, purchase := range purchases {
		purchaseResponse := MapPurchaseModelToPurchaseResponse(purchase)

		statusApprove, ok := mapStatusApprove[purchase.PurchaseCode]
		if ok {
			purchaseResponse.StatusApprove = statusApprove
		}

		if purchase.PurchaseType == "PRE" && purchase.DocRef != nil {
			if prePurchase, ok := mapPrePurchase[*purchase.DocRef]; ok {
				purchaseResponse.RefBigLot = &prePurchase
			}
		}

		var subtotalExclDiscountExclVat float64
		items := make([]models.PurchaseItemResponse, 0, len(purchase.PurchaseItems))
		for _, item := range purchase.PurchaseItems {
			itemResp := MapPurchaseItemModelToPurchaseItemResponse(item)
			subtotalExclDiscountExclVat += item.TotalCost
			items = append(items, itemResp)
		}

		purchaseResponse.Items = items
		purchaseResponse.SubtotalExclDiscountExclVat = subtotalExclDiscountExclVat
		result.DataList = append(result.DataList, purchaseResponse)
	}

	return result, nil
}

func GetPOItem(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := models.GetPurchaseItemRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal request: " + err.Error())
	}

	// Get Purchase Items
	filterPurchases, _, _, _, _, err := purchaseRepository.GetPurchaseListByGRFilter(
		req.SupplierCodes,
		req.PurchaseCodes,
		req.PurchaseItemCodes,
		req.POStatusApprove,
		req.POItemStatus,
		req.ProductCodes,
		req.NotItems,
		req.CompanyCode,
		req.SiteCode,
		0,
		0,
	)

	if err != nil {
		return nil, errors.New("failed to get purchase list: " + err.Error())
	}

	purchaseCodes := []string{}
	purchaseItemCodes := []string{}
	for _, purchase := range filterPurchases {
		purchaseCodes = append(purchaseCodes, purchase.PurchaseCode)

		for _, item := range purchase.PurchaseItems {
			purchaseItemCodes = append(purchaseItemCodes, item.PurchaseItem)
		}
	}

	// Calculate Used qty from Inbound
	reqInboundFilter := goodsReceiveService.InboundFilter{
		InboundItemDocumentRefItem: purchaseItemCodes,
	}
	inbounds, err := goodsReceiveService.GetInbounds(reqInboundFilter)
	if err != nil {
		return nil, errors.New("failed to get used qty from inbound: " + err.Error())
	}

	inboundCodes := []string{}
	inboundCodesCheck := make(map[string]bool)
	for _, inbound := range inbounds.InboundRes {
		if _, exists := inboundCodesCheck[inbound.InboundCode]; !exists {
			inboundCodes = append(inboundCodes, inbound.InboundCode)
			inboundCodesCheck[inbound.InboundCode] = true
		}
	}

	type GoodsReceive struct {
		InboundCode  string
		InboundItem  string
		ConfirmedQty float64
	}

	//Get goods receive
	if len(inboundCodes) > 0 {
		reqGoodsReceiveFilter := goodsReceiveService.GoodsReceiveFilter{
			ReferenceNo: inboundCodes,
		}
		resGoodsReceive, err := goodsReceiveService.GetGoodsReceives(reqGoodsReceiveFilter)
		if err != nil {
			return nil, errors.New("failed to get goods receive: " + err.Error())
		}

		// map inboundCode|inboundItem to confirmedQty
		grConfirmMap := map[string]float64{}
		for _, gr := range resGoodsReceive.GoodsReceive {
			if gr.Status != "COMPLETED" {
				continue
			}

			for _, grItem := range gr.GoodsReceiveItem {
				for _, grConfirm := range grItem.GoodsReceiveConfirm {
					key := gr.DocumentRef + "|" + grItem.DocumentRefItem
					grConfirmMap[key] += grConfirm.BaseQty
				}
			}
		}

		// map inbound item confirmed qty
		for i, inbound := range inbounds.InboundRes {
			if inbound.Status != "COMPLETED" {
				continue
			}

			for ii, inboundItem := range inbound.InboundItemRes {
				key := inbound.InboundCode + "|" + inboundItem.InboundItem
				if confirmedQty, ok := grConfirmMap[key]; ok {
					inbounds.InboundRes[i].InboundItemRes[ii].Qty = confirmedQty
				}
			}
		}
	}

	// map purchase item code to used qty
	inboundMap := make(map[string]float64)
	for _, inbound := range inbounds.InboundRes {
		for _, inboundItem := range inbound.InboundItemRes {
			inboundMap[inboundItem.DocumentRefItem] += inboundItem.Qty
		}
	}

	newReq := models.GetPurchaseItemRequest{
		NotItems:          req.NotItems,
		SupplierCodes:     req.SupplierCodes,
		PurchaseCodes:     req.PurchaseCodes,
		PurchaseItemCodes: req.PurchaseItemCodes,
		POStatusApprove:   req.POStatusApprove,
		POItemStatus:      req.POItemStatus,
		ProductCodes:      req.ProductCodes,
		CompanyCode:       req.CompanyCode,
		SiteCode:          req.SiteCode,
		Page:              req.Page,
		PageSize:          req.PageSize,
	}
	notItems := []models.ExceptPurchaseAndPurchaseItemRequest{}
	for _, p := range filterPurchases {
		notPOItemCodes := []string{}
		for _, pItem := range p.PurchaseItems {
			usedQty, ok := inboundMap[pItem.PurchaseItem]
			if ok {
				if pItem.Qty-usedQty <= 0 {
					notPOItemCodes = append(notPOItemCodes, pItem.PurchaseItem)
				}
			}
		}

		if len(notPOItemCodes) > 0 {
			notItems = append(notItems, models.ExceptPurchaseAndPurchaseItemRequest{
				PurchaseCode:      p.PurchaseCode,
				PurchaseItemCodes: notPOItemCodes,
			})
		}
	}

	if len(notItems) > 0 {
		newReq.NotItems = append(newReq.NotItems, notItems...)
	}

	purchases, total, page, pageSize, totalPage, err := purchaseRepository.GetPurchaseListByGRFilter(
		newReq.SupplierCodes,
		newReq.PurchaseCodes,
		newReq.PurchaseItemCodes,
		newReq.POStatusApprove,
		newReq.POItemStatus,
		newReq.ProductCodes,
		newReq.NotItems,
		newReq.CompanyCode,
		newReq.SiteCode,
		newReq.Page,
		newReq.PageSize,
	)

	if err != nil {
		return nil, errors.New("failed to get purchase list: " + err.Error())
	}

	purchaseCodes = []string{}
	purchaseItemCodes = []string{}
	prePurchaseCodes := []string{}
	for _, purchase := range purchases {
		purchaseCodes = append(purchaseCodes, purchase.PurchaseCode)

		if purchase.PurchaseType == "PRE" && purchase.DocRef != nil {
			prePurchaseCodes = append(prePurchaseCodes, *purchase.DocRef)
		}

		for _, item := range purchase.PurchaseItems {
			purchaseItemCodes = append(purchaseItemCodes, item.PurchaseItem)
		}
	}

	result := models.GetPurchaseItemListResponse{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPage,
	}

	if len(purchases) == 0 {
		result.DataList = []models.GetPurchaseItemResponse{}
		return result, nil
	}

	// Get Approvals
	approvalsResp, err := prePurchaseService.GetPOApproval(ctx, purchaseCodes)
	if err != nil {
		return nil, errors.New("failed to get purchase approvals: " + err.Error())
	}

	mapStatusApprove := map[string]string{}
	for _, approval := range approvalsResp {
		mapStatusApprove[approval.DocumentCode] = approval.Status
	}

	// Get PrePurchase
	prePurchaseReq := models.GetPOBigLotListRequest{
		PrePurchaseCodes: prePurchaseCodes,
		CompanyCode:      req.CompanyCode,
		SiteCode:         req.SiteCode,
		Page:             1,
		PageSize:         len(prePurchaseCodes),
	}

	mapPrePurchase, err := GetRelatedPrePurchase(ctx, prePurchaseReq)
	if err != nil {
		return nil, errors.New("failed to get pre purchase list: " + err.Error())
	}

	// Calculate Used Qty and Weight from Invoice
	usedMap, err := GetUsedQtyAndWeight(req.CompanyCode, req.SiteCode, purchaseCodes, purchaseItemCodes)
	if err != nil {
		return nil, errors.New("failed to get used qty and weight from invoice: " + err.Error())
	}

	// Create Result
	itemsResp := []models.GetPurchaseItemResponse{}
	for _, purchase := range purchases {
		statusApprove := mapStatusApprove[purchase.PurchaseCode]

		var refBigLot *models.GetPOBigLotResponse
		if purchase.PurchaseType == "PRE" && purchase.DocRef != nil {
			if prePurchase, ok := mapPrePurchase[*purchase.DocRef]; ok {
				refBigLot = &prePurchase
			}
		}

		for _, item := range purchase.PurchaseItems {
			itemResp := models.GetPurchaseItemResponse{
				ID:                   item.ID.String(),
				PurchaseCode:         purchase.PurchaseCode,
				PurchaseType:         purchase.PurchaseType,
				CompanyCode:          purchase.CompanyCode,
				SiteCode:             purchase.SiteCode,
				DocRefType:           purchase.DocRefType,
				DocRef:               purchase.DocRef,
				TradingRef:           purchase.TradingRef,
				SupplierCode:         purchase.SupplierCode,
				SupplierName:         purchase.SupplierName,
				SupplierAddress:      purchase.SupplierAddress,
				SupplierPhone:        purchase.SupplierPhone,
				SupplierEmail:        purchase.SupplierEmail,
				PurchaseItem:         item.PurchaseItem,
				DocRefItem:           item.DocRefItem,
				ProductCode:          item.ProductCode,
				ProductDesc:          item.ProductDesc,
				ProductGroupOneCode:  item.ProductGroupCode,
				ProductGroupOneName:  item.ProductGroupName,
				Qty:                  item.Qty,
				Unit:                 item.Unit,
				PurchaseQty:          item.PurchaseQty,
				PurchaseUnit:         item.PurchaseUnit,
				PurchaseUnitType:     item.PurchaseUnitType,
				PriceUnit:            item.PriceUnit,
				TotalDiscount:        item.TotalDiscount,
				TotalAmount:          item.TotalAmount,
				UnitUom:              item.UnitUom,
				TotalCost:            item.TotalCost,
				TotalDiscountPercent: item.TotalDiscountPercent,
				DiscountType:         item.DiscountType,
				TotalVat:             item.TotalVat,
				SubtotalExclVat:      item.SubtotalExclVat,
				WeightUnit:           item.WeightUnit,
				TotalWeight:          item.TotalWeight,
				Status:               item.Status,
				StatusPayment:        item.StatusPayment,
				IsApproved:           purchase.IsApproved,
				StatusApprove:        statusApprove,
				Remark:               item.Remark,
				CreditTerm:           purchase.CreditTerm,
				CreateDtm:            item.CreateDtm.Format(time.RFC3339),
				CreateBy:             item.CreateBy,
				UpdateDtm:            item.UpdateDtm.Format(time.RFC3339),
				UpdateBy:             item.UpdateBy,
				RefBigLot:            refBigLot,
				RemainQty:            item.Qty,
				RemainWeight:         item.TotalWeight,
			}

			if used, ok := usedMap[item.PurchaseItem]; ok {
				itemResp.RemainQty = item.Qty - used.Qty
				itemResp.RemainWeight = item.TotalWeight - used.Weight
			}

			itemResp.InboundRemainQty = item.Qty
			if usedInbound, ok := inboundMap[item.PurchaseItem]; ok {
				itemResp.InboundRemainQty = item.Qty - usedInbound
			}

			itemsResp = append(itemsResp, itemResp)
		}
	}
	result.DataList = itemsResp

	return result, nil
}
