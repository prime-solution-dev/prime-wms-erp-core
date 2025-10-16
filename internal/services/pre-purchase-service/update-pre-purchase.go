package prePurchaseService

import (
	"encoding/json"
	"errors"
	"prime-erp-core/internal/models"
	prePurchaseRepository "prime-erp-core/internal/repositories/prePurchase"
	"time"

	"github.com/gin-gonic/gin"
)

func UpdatePOBigLot(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := []models.UpdatePOBigLotRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	prePurchases := []models.PrePurchase{}

	for _, r := range req {
		prePurchase := MapUpdatePOBigLotRequestToPrePurchase(r)

		for _, itemReq := range r.PrePurchaseItems {
			item := MapUpdatePOBigLotRequestToPrePurchaseItem(itemReq, prePurchase.UpdateBy, time.Now().UTC(), prePurchase.PrePurchaseCode)
			prePurchase.PrePurchaseItems = append(prePurchase.PrePurchaseItems, item)
		}

		prePurchases = append(prePurchases, prePurchase)
	}

	if err := prePurchaseRepository.UpdatePOBigLot(prePurchases); err != nil {
		return nil, errors.New("fail to update big lot: " + err.Error())
	}

	return nil, nil
}

func UpdateStatusApprovePOBigLot(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := []models.UpdateStatusApprovePOBigLotRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	// Update approval status
	if err := UpdateBigLotToApproval(ctx, req); err != nil {
		return nil, err
	}

	if err := prePurchaseRepository.UpdateStatusApprovePOBigLot(req); err != nil {
		return nil, errors.New("failed to update pre purchase status approve: " + err.Error())
	}

	return nil, nil
}
