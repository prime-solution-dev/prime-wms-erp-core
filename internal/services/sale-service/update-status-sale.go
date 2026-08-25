package saleService

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UpdateStatusSaleRequest struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
}

type UpdateStatusSaleResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// หน้าจอมีปุ่มที่ยิงเส้นนี้อยู่สองปุ่มคือ Cancel SO กับ Close SO
// เดิมไม่มีด่านนี้เลย ส่ง status อะไรเข้ามาก็เขียนลง column ตรงๆ
var allowedSaleStatuses = map[string]bool{
	"CANCELED":  true,
	"COMPLETED": true,
}

func UpdateStatusSale(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := UpdateStatusSaleRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	if req.ID == uuid.Nil {
		return nil, fmt.Errorf("sale ID is required")
	}

	if req.Status == "" {
		return nil, fmt.Errorf("status is required")
	}

	if !allowedSaleStatuses[req.Status] {
		return nil, fmt.Errorf("invalid status: %s (expected CANCELED or COMPLETED)", req.Status)
	}

	gormx, err := db.ConnectGORM(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	var sale models.Sale
	if err := gormx.Where("id = ?", req.ID).Take(&sale).Error; err != nil {
		return nil, fmt.Errorf("sale not found: %v", err)
	}

	// เดิมไม่เช็คสถานะเดิมเลย สั่งซ้ำกี่รอบก็เขียนทับ และดันใบที่ยกเลิกไปแล้ว
	// กลับมาเป็น COMPLETED ได้ด้วย
	if sale.Status == req.Status {
		return UpdateStatusSaleResponse{
			Status:  "success",
			Message: fmt.Sprintf("Sale is already %s", req.Status),
		}, nil
	}

	if sale.Status == "CANCELED" {
		return nil, fmt.Errorf("sale %s is already canceled", sale.SaleCode)
	}

	if req.Status == "CANCELED" {
		// ใบจองคิวส่งอ้าง sale_code ไว้ที่ document_ref และฝั่งคลังอ้างต่อจากใบจองอีกที
		// ยกเลิก SO ทิ้งทั้งที่ใบจองยังเดินอยู่ จะเหลือ order ฝั่ง WMS ชี้มาที่ใบที่ตายแล้ว
		// ด่านเดียวกับที่ UpdateStatusDelivery กันไม่ให้ยกเลิกใบที่คลังเริ่มทำงานแล้ว
		openDeliveries, err := openDeliveriesOfSale(gormx, sale.SaleCode)
		if err != nil {
			return nil, err
		}

		if len(openDeliveries) > 0 {
			return nil, fmt.Errorf(
				"sale %s still has active delivery booking (%s), cancel those first",
				sale.SaleCode, strings.Join(openDeliveries, ", "))
		}
	}

	user := ctx.GetString("user")
	if user == "" {
		user = `system` // fallback
	}
	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	updateFields := map[string]interface{}{
		"status":      req.Status,
		"update_date": nowDateOnly,
		"update_by":   user,
	}

	// เดิมอัปเดตหัวใบอย่างเดียวไม่มี transaction ยกเลิกใบแล้ว item ยังเดินอยู่ทั้งใบ
	err = gormx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Sale{}).
			Where("id = ?", req.ID).
			Updates(updateFields).Error; err != nil {
			return fmt.Errorf("failed to update sale status: %v", err)
		}

		// ปิดใบด้วยมือ (COMPLETED) ไม่แตะ item เพราะ sale_item.status = COMPLETED
		// แปลว่าส่งของครบจริง ซึ่ง UpdateSaleItemStatus เป็นคนตั้งตามความคืบหน้าฝั่งคลัง
		// เขียนทับตรงนี้จะกลายเป็นบอกว่าส่งครบแล้วทั้งที่ยังไม่ได้ส่ง
		if req.Status != "CANCELED" {
			return nil
		}

		if err := tx.Model(&models.SaleItem{}).
			Where("sale_id = ?", req.ID).
			Updates(updateFields).Error; err != nil {
			return fmt.Errorf("failed to update sale items status: %v", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return UpdateStatusSaleResponse{
		Status:  "success",
		Message: "Sale status updated successfully",
	}, nil
}

// openDeliveriesOfSale คืนรหัสใบจองคิวส่งของ sale ใบนี้ที่ยังไม่ถูกยกเลิก
// ใบที่ส่งของไปแล้ว (COMPLETED) ก็นับว่ายังเดินอยู่ ยกเลิก SO ทีหลังไม่ได้เหมือนกัน
func openDeliveriesOfSale(gormx *gorm.DB, saleCode string) ([]string, error) {
	deliveryCodes := []string{}

	if err := gormx.Model(&models.Delivery{}).
		Where("document_ref = ? AND status <> ?", saleCode, "CANCELED").
		Pluck("delivery_code", &deliveryCodes).Error; err != nil {
		return nil, fmt.Errorf("failed to read delivery bookings of sale %s: %v", saleCode, err)
	}

	return deliveryCodes, nil
}
