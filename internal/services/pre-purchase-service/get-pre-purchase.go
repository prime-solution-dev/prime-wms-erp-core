package prePurchaseService

import (
	"encoding/json"
	"errors"
	"prime-erp-core/internal/models"
	prePurchaseRepository "prime-erp-core/internal/repositories/prePurchase"

	"github.com/gin-gonic/gin"
)

func GetPOBigLot(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := models.GetPOBigLotListRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	prePurchaseList, total, page, pageSize, totalPage, err := prePurchaseRepository.GetPOBigLotList(req.PrePurchaseCodes, req.SupplierCodes, req.ProductGroupCodes, req.StatusApprove, req.CompanyCode, req.SiteCode, req.Page, req.PageSize)
	if err != nil {
		return nil, errors.New("failed to get big lot list: " + err.Error())
	}

	prePurchaseCodes := []string{}
	supplierReq := models.GetSupplierListRequest{}
	for _, prePurchase := range prePurchaseList {
		supplierReq.SupplierCodes = append(supplierReq.SupplierCodes, prePurchase.SupplierCode)
		prePurchaseCodes = append(prePurchaseCodes, prePurchase.PrePurchaseCode)
	}

	mapSupplier, err := GetSupplierByCode(supplierReq)
	if err != nil {
		return nil, err
	}

	// Get status approves
	approvalsResp, err := GetPOApproval(ctx, prePurchaseCodes)
	if err != nil {
		return nil, err
	}

	mapStatusApprove := map[string]string{}
	for _, approval := range approvalsResp {
		mapStatusApprove[approval.DocumentCode] = approval.Status
	}

	result := models.GetPOBigLotListResponse{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPage,
	}

	for _, prePurchase := range prePurchaseList {
		bigLotResponse := MapPrePurchasesModelToBigLotsResponse(prePurchase)

		bigLotResponse.SupplierName = mapSupplier[prePurchase.SupplierCode].SupplierName
		bigLotResponse.StatusApprove = mapStatusApprove[prePurchase.PrePurchaseCode]

		result.BigLotList = append(result.BigLotList, bigLotResponse)
	}

	return result, nil
}
