package quotationService

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CancelQuotationRequest struct {
	ID uuid.UUID `json:"id"`
}

func CancelQuotation(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := CancelQuotationRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	var quotation models.Quotation
	if err := gormx.Where("id = ?", req.ID).Take(&quotation).Error; err != nil {
		return nil, fmt.Errorf("quotation not found: %v", err)
	}

	// เดิมไม่เช็คสถานะเลย ใบที่แปลงเป็น sale order ไปแล้ว (status = COMPLETED)
	// หรือใบที่ยกเลิกไปแล้ว ก็สั่งยกเลิกซ้ำได้ ทั้งที่ SO ยังเดินอยู่
	switch quotation.Status {
	case "COMPLETED":
		return nil, errors.New("quotation already converted to a sale order and cannot be canceled")
	case "CANCELED":
		return nil, errors.New("quotation is already canceled")
	}

	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// เดิมยิง 2 update แยกกันไม่มี transaction ตัวที่สองพังจะเหลือหัวใบ CANCELED แต่ item ยังเดินอยู่
	err = gormx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Quotation{}).
			Where("id = ?", req.ID).
			Updates(map[string]interface{}{
				"status": "CANCELED",
				// เดิมไม่แตะ status_approve เลย ใบที่ยกเลิกแล้วจึงยังค้างเป็น COMPLETED
				// ทำให้หน้าจอที่อ่าน status_approve คิดว่ายังอนุมัติอยู่
				"status_approve": "CANCELED",
				"update_date":    nowDateOnly,
			}).Error; err != nil {
			return fmt.Errorf("failed to update quotation status: %v", err)
		}

		if err := tx.Model(&models.QuotationItem{}).
			Where("quotation_id = ?", req.ID).
			Updates(map[string]interface{}{
				"status":      "CANCELED",
				"update_date": nowDateOnly,
			}).Error; err != nil {
			return fmt.Errorf("failed to update quotation items status: %v", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status":  "success",
		"message": "Quotation canceled successfully",
	}, nil
}
