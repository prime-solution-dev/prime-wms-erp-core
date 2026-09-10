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
	"gorm.io/gorm"
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

	var sale models.Sale
	if err := gormx.Where("id = ?", req.ID).Take(&sale).Error; err != nil {
		return nil, fmt.Errorf("sale not found: %v", err)
	}

	// เดิมไม่เช็คสถานะเลย กด REJECT ใบที่ปิดไปแล้วหรือใบที่ยกเลิกไปแล้วได้
	// ซึ่งจะดัน status กลับเป็น CANCELED ทั้งที่ของอาจส่งออกไปหมดแล้ว
	switch sale.Status {
	case "COMPLETED":
		return nil, errors.New("sale order is already closed and its approval cannot be changed")
	case "CANCELED":
		return nil, errors.New("sale order is canceled and its approval cannot be changed")
	}

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

	// อนุมัติผ่านแล้วให้ราคาที่ใช้จริงกลับไปเป็นราคาจาก quotation
	// บรรทัดที่ไม่มีราคาจาก quotation (0) ไม่แตะ เพราะไม่มีอะไรให้ยึด
	if shouldAdoptQuotationPrice(req.Status) {
		user := ctx.GetString("user")
		if user == "" {
			user = `system`
		}

		if err := gormx.Model(&models.SaleItem{}).
			Where("sale_id = ? AND old_price_list_unit > 0", req.ID).
			Updates(map[string]interface{}{
				"price_list_unit": gorm.Expr("old_price_list_unit"),
				"update_date":     nowDateOnly,
				"update_by":       user,
			}).Error; err != nil {
			return nil, fmt.Errorf("failed to adopt quotation price for sale %v: %v", req.ID, err)
		}
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

// shouldAdoptQuotationPrice บอกว่าผลอนุมัตินี้ทำให้ใบยึดราคาจาก quotation หรือยัง
// อนุมัติผ่าน = ผู้อนุมัติยอมรับราคาที่ห่างจาก price list master แล้ว (SA เคาะ 2026-09-10)
func shouldAdoptQuotationPrice(status string) bool {
	return status == "COMPLETED"
}
