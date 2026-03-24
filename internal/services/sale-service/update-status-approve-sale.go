package saleService

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	approvalService "prime-erp-core/internal/services/approval-service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UpdateStatusApproveSaleRequest struct {
	ID         uuid.UUID `json:"id"`
	ApprovalID uuid.UUID `json:"approval_id"`
	Status     string    `json:"status"` // REVIEW, REJECT, COMPLETED
	Remark     string    `json:"remark"`
}

func UpdateStatusApproveSale(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := UpdateStatusApproveSaleRequest{}

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

	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	updateFields := map[string]interface{}{
		"status":          quotationStatus,
		"status_approve":  quotationStatusApprove,
		"remark_approval": req.Remark,
		"update_date":     nowDateOnly,
	}

	// เพิ่ม is_approved = true เมื่อ status เป็น COMPLETED
	if req.Status == "COMPLETED" {
		updateFields["is_approved"] = true
	}

	if err := gormx.Model(&models.Sale{}).
		Where("id = ?", req.ID).
		Updates(updateFields).Error; err != nil {
		return nil, fmt.Errorf("failed to update sale status: %v", err)
	}

	// If status is REJECT, also update sale_item status to CANCELED
	if req.Status == "REJECT" {
		if err := gormx.Model(&models.SaleItem{}).
			Where("sale_id = ?", req.ID).
			Updates(map[string]interface{}{
				"status":      "CANCELED",
				"update_date": nowDateOnly,
			}).Error; err != nil {
			return nil, fmt.Errorf("failed to update sale items status: %v", err)
		}
	}

	return map[string]interface{}{
		"external_response": approvalResult,
		"status":            "success",
		"message":           "Approval updated successfully",
	}, nil
}
