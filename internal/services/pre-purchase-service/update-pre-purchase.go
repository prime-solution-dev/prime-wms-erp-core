package prePurchaseService

import (
	"encoding/json"
	"errors"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	prePurchaseRepository "prime-erp-core/internal/repositories/prePurchase"
	approvalService "prime-erp-core/internal/services/approval-service"
	"time"

	"github.com/gin-gonic/gin"
)

func UpdatePOBigLot(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := []models.UpdatePOBigLotRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
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

	for _, r := range req {
		prePurchase := MapUpdatePOBigLotRequestToPrePurchase(r)

		for _, itemReq := range r.PrePurchaseItems {
			item := MapUpdatePOBigLotRequestToPrePurchaseItem(itemReq, prePurchase.UpdateBy, time.Now().UTC(), prePurchase.PrePurchaseCode)
			prePurchase.PrePurchaseItems = append(prePurchase.PrePurchaseItems, item)
		}

		prePurchase.UpdateBy = userCode
		prePurchase.UpdateDtm = time.Now().UTC()

		if r.Status == "PENDING" {
			// Check auto approval for PENDING status
			autoApprovalReq := approvalService.CheckAutoApprovalRequest{
				RequestUserCode: userCode,
				ModuleCode:      "CUSTOMIZE",
				TopicCode:       "CUSTOMIZE",
				MdItemCode:      "CTM-CTM3",
				CondRangeMin:    r.TotalAmount,
				// TODO: Add module_code, topic_code, md_item_code
			}

			autoApprovalRes, err := approvalService.CheckAutoApproval(gormx, autoApprovalReq, userCode)
			if err != nil {
				return nil, err
			}

			if autoApprovalRes.IsAutoApproved {
				r.IsApproved = true
				r.StatusApprove = "COMPLETED"
			} else {
				r.IsApproved = false
				r.StatusApprove = "PROCESS"
			}
		}

		prePurchases = append(prePurchases, prePurchase)
	}

	if err := prePurchaseRepository.UpdatePOBigLot(gormx, prePurchases); err != nil {
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
