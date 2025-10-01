package purchaseService

import (
	"encoding/json"
	"errors"
	"fmt"
	"prime-erp-core/internal/models"
	purchaseRepository "prime-erp-core/internal/repositories/purchase"
	systemConfigRepository "prime-erp-core/internal/repositories/systemConfig"
	"time"

	"github.com/gin-gonic/gin"
)

func CreatePOBigLot(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := []models.CreatePOBigLotRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	prePurchases := []models.PrePurchase{}
	prePurchaseConfig, err := GetPrePurchaseCodeConfig()
	if err != nil {
		return nil, err
	}

	latestNum, err := ConvertConfigToLatestPrePurchaseNumber(*prePurchaseConfig)
	if err != nil {
		return nil, err
	}

	for idx, r := range req {
		prePurchase := MapBigLotRequestToPrePurchaseModel(r)
		saveValue := fmt.Sprintf(
			"%s-%04d",
			time.Now().Format("20060102"),
			latestNum+idx+1,
		)
		prePurchaseCode := fmt.Sprintf(
			"%s-%s%s",
			prePurchaseConfig.TopicCode,
			prePurchaseConfig.ConfigCode,
			saveValue,
		)
		prePurchase.PrePurchaseCode = prePurchaseCode
		prePurchases = append(prePurchases, prePurchase)

		if idx == len(req)-1 {
			prePurchaseConfig.Value = saveValue
		}
	}

	if err := purchaseRepository.CreatePOBigLot(prePurchases); err != nil {
		return nil, errors.New("failed to create big lot: " + err.Error())
	}

	if err := CreateBigLotToApproval(ctx, prePurchases); err != nil {
		return nil, errors.New("failed to create approval: " + err.Error())
	}

	updateSystemConfigReq := []models.SystemConfig{}
	updateSystemConfigReq = append(updateSystemConfigReq, *prePurchaseConfig)
	if err := systemConfigRepository.UpdateSystemConfig(updateSystemConfigReq); err != nil {
		return nil, err
	}
	return nil, nil
}
