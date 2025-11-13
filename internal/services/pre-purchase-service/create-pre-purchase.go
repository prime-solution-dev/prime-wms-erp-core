package prePurchaseService

import (
	"encoding/json"
	"errors"
	"fmt"
	"prime-erp-core/internal/models"

	prePurchaseRepository "prime-erp-core/internal/repositories/prePurchase"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreatePOBigLot(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := []models.CreatePOBigLotRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	prePurchaseCodeCount := len(req)
	prePurchaseCodes, err := GeneratePrePurchaseCodes(ctx, prePurchaseCodeCount)
	if err != nil {
		return nil, errors.New("failed to generate pre-purchase order codes: " + err.Error())
	}

	prePurchases := []models.PrePurchase{}
	for idx, r := range req {
		id := uuid.New()
		prePurchase := MapBigLotRequestToPrePurchaseModel(r)
		prePurchase.PrePurchaseCode = prePurchaseCodes[idx]
		prePurchase.ID = id

		items := []models.PrePurchaseItem{}
		for _, itemReq := range r.Items {
			preItem := fmt.Sprintf("%s-%s", prePurchaseCodes[idx], time.Now().Format("150405"))
			item := MapBigLotRequestToPrePurchaseItemsModel(itemReq, id, prePurchase.CreateBy, time.Now().UTC(), preItem)
			items = append(items, item)
		}

		prePurchase.PrePurchaseItems = items
		prePurchases = append(prePurchases, prePurchase)
	}

	if err := prePurchaseRepository.CreatePOBigLot(prePurchases); err != nil {
		return nil, errors.New("failed to create big lot: " + err.Error())
	}

	if err := CreateBigLotToApproval(ctx, prePurchases); err != nil {
		return nil, errors.New("failed to create approval: " + err.Error())
	}

	return prePurchaseCodes, nil
}
