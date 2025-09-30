package quotationService

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	approvalService "prime-erp-core/internal/services/approval-service"
	verifyService "prime-erp-core/internal/services/verify-service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UpdateStatusApproveQuotationRequest struct {
	ID         uuid.UUID `json:"id"`
	ApprovalID uuid.UUID `json:"approval_id"`
	Status     string    `json:"status"` // REVIEW, REJECT, COMPLETED
	Remark     string    `json:"remark"`
}

func UpdateStatusApproveQuotation(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := UpdateStatusApproveQuotationRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	sqlx, err := db.ConnectSqlx(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer sqlx.Close()

	gormx, err := db.ConnectGORM(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	updateApprovalReq := []struct {
		ID     uuid.UUID `json:"id"`
		Status string    `json:"status"`
		Remark string    `json:"remark"`
	}{
		{
			ID:     req.ApprovalID,
			Status: req.Status,
			Remark: req.Remark,
		},
	}

	updateApprovalPayload, _ := json.Marshal(updateApprovalReq)
	approvalResult, err := approvalService.UpdateApproval(ctx, string(updateApprovalPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to update approval: %v", err)
	}

	// Update quotation based on approval status
	var quotationStatus, quotationStatusApprove string

	switch req.Status {
	case "REVIEW":
		quotationStatus = "PENDING"
		quotationStatusApprove = "REVIEW"
	case "REJECT":
		quotationStatus = "CANCELED"
		quotationStatusApprove = "REJECT"
	case "COMPLETED":
		quotationStatus = "PENDING"
		quotationStatusApprove = "COMPLETED"
	default:
		return nil, fmt.Errorf("invalid status: %s", req.Status)
	}

	updateFields := map[string]interface{}{
		"status":          quotationStatus,
		"status_approve":  quotationStatusApprove,
		"remark_approval": req.Remark,
		"update_date":     gormx.NowFunc(),
	}
	if req.Status == "COMPLETED" {
		// Query existing quotation to check expire_price_date
		var existingQuotation models.Quotation
		if err := gormx.Where("id = ?", req.ID).First(&existingQuotation).Error; err != nil {
			return nil, fmt.Errorf("failed to get existing quotation: %v", err)
		}

		// Check if expire_price_date has expired
		now := time.Now()
		isPriceExpired := existingQuotation.ExpirePriceDate == nil || existingQuotation.ExpirePriceDate.Before(now)

		if isPriceExpired {
			//get config
			topic := `PRICE`
			configCodes := []string{`EXPIRY_PRICE_DAYS`}
			configMap, err := verifyService.GetConfigSystem(gormx, topic, configCodes)
			if err != nil {
				return nil, err
			}

			expiryDaysConfig, exists := configMap[fmt.Sprintf(`%s|%s`, topic, `EXPIRY_PRICE_DAYS`)]
			if !exists {
				return nil, errors.New("missing configuration for expiry price days")
			}

			expiryDays, err := strconv.ParseInt(expiryDaysConfig.Value, 10, 64)
			if err != nil {
				return nil, errors.New("failed to convert expiry days to int64: " + err.Error())
			}

			expiryDate := time.Now().AddDate(0, 0, int(expiryDays))

			updateFields["effective_date_price"] = gormx.NowFunc()
			updateFields["expire_price_day"] = int(expiryDays)
			updateFields["expire_price_date"] = &expiryDate
		}
		updateFields["is_approved"] = true
	}

	if err := gormx.Model(&models.Quotation{}).
		Where("id = ?", req.ID).
		Updates(updateFields).Error; err != nil {
		return nil, fmt.Errorf("failed to update quotation status: %v", err)
	}

	// If status is REJECT, also update quotation_item status to CANCELED
	if req.Status == "REJECT" {
		if err := gormx.Model(&models.QuotationItem{}).
			Where("quotation_id = ?", req.ID).
			Updates(map[string]interface{}{
				"status":      "CANCELED",
				"update_date": gormx.NowFunc(),
			}).Error; err != nil {
			return nil, fmt.Errorf("failed to update quotation items status: %v", err)
		}
	}

	return map[string]interface{}{
		"external_response": approvalResult,
		"status":            "success",
		"message":           "Approval updated successfully",
	}, nil
}
