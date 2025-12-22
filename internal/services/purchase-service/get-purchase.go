package purchaseService

import (
	"encoding/json"
	"errors"
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

		purchaseResponse.StatusApprove = mapStatusApprove[purchase.PurchaseCode]

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
	purchases, total, page, pageSize, totalPage, err := purchaseRepository.GetPurchaseListByGRFilter(
		req.SupplierCodes,
		req.POStatusApprove,
		req.POItemStatus,
		req.ProductCodes,
		req.NotItems,
		req.CompanyCode,
		req.SiteCode,
		req.Page,
		req.PageSize,
	)

	if err != nil {
		return nil, errors.New("failed to get purchase list: " + err.Error())
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

	purchaseCodes := []string{}
	purchaseItemCodes := []string{}
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

	itemsResp := []models.GetPurchaseItemResponse{}
	// Create Result
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

			itemsResp = append(itemsResp, itemResp)
		}
	}
	result.DataList = itemsResp

	return result, nil
}
