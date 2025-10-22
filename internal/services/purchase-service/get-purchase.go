package purchaseService

import (
	"encoding/json"
	"errors"
	"prime-erp-core/internal/models"
	purchaseRepository "prime-erp-core/internal/repositories/purchase"
	prePurchaseService "prime-erp-core/internal/services/pre-purchase-service"

	"github.com/gin-gonic/gin"
)

func GetPO(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := models.GetPurchaseRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	purchases, total, page, pageSize, totalPage, err := purchaseRepository.GetPurchaseList(
		req.PurchaseCodes, //
		req.SupplierCodes,
		req.StatusApprove,
		req.StatusPayment,
		req.StatusPaymentIncomplete,
		req.ProductCodes,
		req.PurchaseType,
		req.DocRef,
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
	supplierReq := models.GetSupplierListRequest{}
	productCodes := []string{}
	for _, purchase := range purchases {
		supplierReq.SupplierCodes = append(supplierReq.SupplierCodes, purchase.SupplierCode)
		purchaseCodes = append(purchaseCodes, purchase.PurchaseCode)

		if purchase.PurchaseType == "PRE" && purchase.DocRef != nil {
			prePurchaseCodes = append(prePurchaseCodes, *purchase.DocRef)
		}

		for _, item := range purchase.PurchaseItems {
			productCodes = append(productCodes, item.ProductCode)
		}
	}

	// Get suppliers
	mapSupplier, err := prePurchaseService.GetSupplierByCode(supplierReq)
	if err != nil {
		return nil, errors.New("failed to get supplier list: " + err.Error())
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

	// Get Products
	productReq := models.GetProductRequest{
		ProductCode: productCodes,
		SiteCode:    []string{req.SiteCode},
		CompanyCode: []string{req.CompanyCode},
	}

	mapProduct, err := GetProductByCode(productReq)
	if err != nil {
		return nil, errors.New("failed to get product list: " + err.Error())
	}

	// Get Product Group One
	productGroupOne := models.GetGroupRequest{
		GroupCodes: []string{"PRODUCT_GROUP1"},
	}

	mapProductGroupOne, err := GetProductGroup(productGroupOne)
	if err != nil {
		return nil, errors.New("failed to get product group one list: " + err.Error())
	}

	// Create Result
	for _, purchase := range purchases {
		purchaseResponse := MapPurchaseModelToPurchaseResponse(purchase)

		if supplier, ok := mapSupplier[purchase.SupplierCode]; ok {
			purchaseResponse.SupplierName = supplier.SupplierName
		}

		purchaseResponse.StatusApprove = mapStatusApprove[purchase.PurchaseCode]

		if purchase.PurchaseType == "PRE" && purchase.DocRef != nil {
			if prePurchase, ok := mapPrePurchase[*purchase.DocRef]; ok {
				purchaseResponse.RefBigLot = &prePurchase
			}
		}

		items := make([]models.PurchaseItemResponse, 0, len(purchase.PurchaseItems))
		for _, item := range purchase.PurchaseItems {
			itemResp := MapPurchaseItemModelToPurchaseItemResponse(item)
			if productDetail, ok := mapProduct[item.ProductCode]; ok {
				itemResp.ProductName = productDetail.ProductName

				// Set Product Group One
				if len(productDetail.ProductGroups) == 1 {
					groupCode := productDetail.ProductGroups[0].GroupValue
					if group, ok := mapProductGroupOne[groupCode]; ok {
						itemResp.ProductGroupOneCode = groupCode
						itemResp.ProductGroupOneName = group
					}
				} else if len(productDetail.ProductGroups) > 1 {
					itemResp.ProductGroupOneCode = "Multi"
					itemResp.ProductGroupOneName = "Multi"
				}
			} else {
				itemResp.ProductName = "Unknown"
			}

			items = append(items, itemResp)
		}

		purchaseResponse.Items = items
		result.DataList = append(result.DataList, purchaseResponse)
	}

	return result, nil
}
