package prePurchaseService

import (
	"encoding/json"
	"errors"
	"fmt"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	approvalService "prime-erp-core/internal/services/approval-service"

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

	userCode := ""
	if ctx != nil {
		userCode = ctx.GetString("user")
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	prePurchases := []models.PrePurchase{}
	for idx, r := range req {
		id := uuid.New()
		prePurchase := MapBigLotRequestToPrePurchaseModel(r)
		prePurchase.PrePurchaseCode = prePurchaseCodes[idx]
		prePurchase.ID = id
		prePurchase.CreateBy = userCode
		prePurchase.UpdateBy = userCode
		prePurchase.CreateDtm = time.Now().UTC()
		prePurchase.UpdateDtm = time.Now().UTC()

		items := []models.PrePurchaseItem{}
		for _, itemReq := range r.Items {
			preItem := fmt.Sprintf("%s-%s", prePurchaseCodes[idx], time.Now().Format("150405"))
			item := MapBigLotRequestToPrePurchaseItemsModel(itemReq, id, prePurchase.CreateBy, time.Now().UTC(), preItem)
			items = append(items, item)
		}

		if prePurchase.Status == "PENDING" {
			// Check auto approval for PENDING status
			autoApprovalReq := approvalService.CheckAutoApprovalRequest{
				RequestUserCode: userCode,
				ModuleCode:      "CUSTOMIZE",
				TopicCode:       "CUSTOMIZE",
				MdItemCode:      "CTM-CTM3",
				CondRangeMin:    prePurchase.TotalAmount,
				// TODO: Add module_code, topic_code, md_item_code
			}

			autoApprovalRes, err := approvalService.CheckAutoApproval(gormx, autoApprovalReq, userCode)
			if err != nil {
				return nil, err
			}

			if autoApprovalRes.IsAutoApproved {
				prePurchase.IsApproved = true
				prePurchase.StatusApprove = "COMPLETED"
			} else {
				prePurchase.IsApproved = false
				prePurchase.StatusApprove = "PROCESS"
			}
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
