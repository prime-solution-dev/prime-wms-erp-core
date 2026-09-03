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

	prePurchaseList, total, page, pageSize, totalPage, err := prePurchaseRepository.GetPOBigLotList(req)
	if err != nil {
		return nil, errors.New("failed to get big lot list: " + err.Error())
	}

	result := models.GetPOBigLotListResponse{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPage,
	}

	if len(prePurchaseList) == 0 {
		result.BigLotList = []models.GetPOBigLotResponse{}
		return result, nil
	}

	prePurchaseCodes := []string{}
	for _, prePurchase := range prePurchaseList {
		prePurchaseCodes = append(prePurchaseCodes, prePurchase.PrePurchaseCode)
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

	for _, prePurchase := range prePurchaseList {
		bigLotResponse := MapPrePurchasesModelToBigLotsResponse(prePurchase)

		bigLotResponse.StatusApprove = mapStatusApprove[prePurchase.PrePurchaseCode]

		result.BigLotList = append(result.BigLotList, bigLotResponse)
	}

	return result, nil
}
