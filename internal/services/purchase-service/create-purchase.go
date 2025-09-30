package purchaseService

import (
	"encoding/json"
	"errors"
	"prime-erp-core/internal/models"
	purchaseRepository "prime-erp-core/internal/repositories/purchase"

	"github.com/gin-gonic/gin"
)

func CreatePOBigLot(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := []models.CreatePOBigLotRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	prePurchases := []models.PrePurchase{}

	for _, r := range req {
		prePurchase := MapBigLotRequestToPrePurchaseModel(r)
		prePurchases = append(prePurchases, prePurchase)
	}

	if err := purchaseRepository.CreatePOBigLot(prePurchases); err != nil {
		return nil, errors.New("failed to create big lot: " + err.Error())
	}

	if err := CreateBigLotToApproval(ctx, prePurchases); err != nil {
		return nil, errors.New("failed to create approval: " + err.Error())
	}

	return nil, nil
}
