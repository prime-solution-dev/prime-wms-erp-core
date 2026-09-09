package saleService

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

type UpdateSaleItemStatusRequest struct {
	SaleItem []string `json:"sale_item"`
	Status   string   `json:"status"`
}

type UpdateSaleItemStatusResponse struct {
	Status         string   `json:"status"`
	Message        string   `json:"message"`
	UpdatedItems   []string `json:"updated_items"`
	UpdatedSales   []string `json:"updated_sales,omitempty"`
	CompletedSales []string `json:"completed_sales,omitempty"`
}

// closedSaleItemStatusList คือสถานะของบรรทัดขายที่ถือว่างานบรรทัดนั้นจบแล้ว
// ยกเลิกก็นับว่าจบ ไม่งั้นใบที่มีบรรทัดถูกยกเลิกจะไม่มีวันปิดหัวใบ
//
// ประกาศเป็น slice ที่เดียวแล้วแปลงเป็น map ให้ predicate ใช้ เพื่อให้เงื่อนไข SQL
// (NOT IN) กับ IsSaleItemClosed อ้างลิสต์เดียวกันจริงๆ เพิ่มวิธีสะกดใหม่แล้วไม่หลุดที่ใดที่หนึ่ง
var closedSaleItemStatusList = []string{"COMPLETED", "CANCELED", "CANCELLED"}

var closedSaleItemStatuses = func() map[string]bool {
	statuses := map[string]bool{}
	for _, status := range closedSaleItemStatusList {
		statuses[status] = true
	}

	return statuses
}()

// IsSaleItemClosed บอกว่าบรรทัดขายบรรทัดเดียวจบงานแล้วหรือยัง
// export เพราะ delivery-service ต้องใช้กติกาเดียวกันตอนเลือกบรรทัดที่จะปิด
func IsSaleItemClosed(status string) bool {
	return closedSaleItemStatuses[status]
}

// allSaleItemsClosed บอกว่าทุกบรรทัดของใบนั้นจบงานแล้วหรือยัง
func allSaleItemsClosed(statuses []string) bool {
	if len(statuses) == 0 {
		return false
	}

	for _, status := range statuses {
		if !IsSaleItemClosed(status) {
			return false
		}
	}

	return true
}

// MarkSaleItemsCompleted ตั้งสถานะบรรทัดขายที่ระบุเป็นค่าอะไรก็ได้ แล้วปิดหัวใบที่บรรทัดจบครบทุกบรรทัด
//
// ตัวนี้ไว้ให้ "เส้นมือ" (POST /sale/UpdateSaleItemStatus) ซึ่งหน้าที่ของมันคือตั้งสถานะ
// ตามที่ผู้ใช้สั่ง จึงต้องเขียนทับบรรทัดที่ปิดไปแล้วได้ด้วย (แก้ที่ตั้งผิดกลับ)
// เส้นอัตโนมัติต้องใช้ MarkOpenSaleItemsCompleted ที่มีการ์ดสถานะแทน
func MarkSaleItemsCompleted(tx *gorm.DB, saleItemCodes []string, status string, user string, now time.Time) ([]string, []string, error) {
	return markSaleItems(tx, saleItemCodes, status, user, now, false)
}

// MarkOpenSaleItemsCompleted ปิดเฉพาะบรรทัดที่ "ยังเปิดอยู่" เป็น COMPLETED
//
// ใช้กับเส้นอัตโนมัติ (ปิดตอนใบจองส่งของครบ ดูที่ delivery-service/close-sale-on-delivered.go)
// ซึ่งอ่านสถานะบรรทัดมาก่อนแล้วค่อยเขียน ระหว่างนั้นอาจมีคนยกเลิกบรรทัดนั้นไปแล้ว
// หรือใบจองอีกใบของ SO เดียวกันปิดไปก่อน ถ้าไม่มีการ์ดจะเขียนทับ CANCELED เป็น COMPLETED
// การ์ดนี้ทำให้เรียกซ้ำกี่รอบก็ได้ผลเท่าเดิม (idempotent) ตามที่เส้น hook ต้องการ
func MarkOpenSaleItemsCompleted(tx *gorm.DB, saleItemCodes []string, user string, now time.Time) ([]string, []string, error) {
	return markSaleItems(tx, saleItemCodes, "COMPLETED", user, now, true)
}

// markSaleItems ตั้งสถานะบรรทัดขายที่ระบุ แล้วปิดหัวใบที่บรรทัดจบครบทุกบรรทัด
//
// ทำงานบน tx ที่ผู้เรียกเปิดมา เพื่อให้เส้นอัตโนมัติกับเส้นมือใช้กติกาเดียวกันจริงๆ
// ไม่ใช่เขียนซ้ำสองที่
//
// openOnly = true จะแตะแค่บรรทัดที่ยังไม่จบงาน (ดู closedSaleItemStatusList)
func markSaleItems(tx *gorm.DB, saleItemCodes []string, status string, user string, now time.Time, openOnly bool) ([]string, []string, error) {
	updatedSaleCodes := []string{}
	completedSaleCodes := []string{}

	if len(saleItemCodes) == 0 {
		return updatedSaleCodes, completedSaleCodes, nil
	}

	updateFields := map[string]interface{}{
		"status":      status,
		"update_date": now,
		"update_by":   user,
	}

	updateQuery := tx.Model(&models.SaleItem{}).Where("sale_item IN ?", saleItemCodes)
	if openOnly {
		// status IS NULL ต้องนับเป็น "ยังเปิด" ด้วย ไม่งั้น NOT IN จะตัดแถวนั้นออกเงียบๆ
		updateQuery = updateQuery.Where("(status IS NULL OR status NOT IN ?)", closedSaleItemStatusList)
	}

	if err := updateQuery.Updates(updateFields).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to update sale items status: %v", err)
	}

	var affectedSaleIDs []uuid.UUID
	if err := tx.Model(&models.SaleItem{}).
		Where("sale_item IN ?", saleItemCodes).
		Distinct("sale_id").
		Pluck("sale_id", &affectedSaleIDs).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to get affected sale IDs: %v", err)
	}

	for _, saleID := range affectedSaleIDs {
		var sale models.Sale
		if err := tx.Where("id = ?", saleID).First(&sale).Error; err != nil {
			return nil, nil, fmt.Errorf("failed to get sale for ID %v: %v", saleID, err)
		}
		updatedSaleCodes = append(updatedSaleCodes, sale.SaleCode)

		var saleItems []models.SaleItem
		if err := tx.Where("sale_id = ?", saleID).Find(&saleItems).Error; err != nil {
			return nil, nil, fmt.Errorf("failed to get sale items for sale ID %v: %v", saleID, err)
		}

		statuses := make([]string, 0, len(saleItems))
		for _, item := range saleItems {
			statuses = append(statuses, item.Status)
		}

		if !allSaleItemsClosed(statuses) {
			continue
		}

		// ใบที่ยกเลิกหรือปิดไปแล้ว ห้ามถูกเขียนทับ
		// ใช้ predicate ตัวเดียวกับบรรทัด จะได้รู้จัก CANCELLED สองแอลด้วย
		if IsSaleItemClosed(sale.Status) {
			continue
		}

		if err := tx.Model(&models.Sale{}).
			Where("id = ?", saleID).
			Updates(map[string]interface{}{
				"status":      "COMPLETED",
				"update_date": now,
				"update_by":   user,
			}).Error; err != nil {
			return nil, nil, fmt.Errorf("failed to update sale status to completed for sale ID %v: %v", saleID, err)
		}

		completedSaleCodes = append(completedSaleCodes, sale.SaleCode)
	}

	return updatedSaleCodes, completedSaleCodes, nil
}

func UpdateSaleItemStatus(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := UpdateSaleItemStatusRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	if len(req.SaleItem) == 0 {
		return nil, fmt.Errorf("sale items are required")
	}

	if req.Status == "" {
		return nil, fmt.Errorf("status is required")
	}

	gormx, err := db.ConnectGORM(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	user := `system` // TODO: get from ctx
	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Start transaction
	tx := gormx.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	updatedSaleCodes, completedSaleCodes, err := MarkSaleItemsCompleted(tx, req.SaleItem, req.Status, user, nowDateOnly)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	response := UpdateSaleItemStatusResponse{
		Status:       "success",
		Message:      fmt.Sprintf("Updated %d sale items status to %s", len(req.SaleItem), req.Status),
		UpdatedItems: req.SaleItem,
		UpdatedSales: updatedSaleCodes,
	}

	if len(completedSaleCodes) > 0 {
		response.CompletedSales = completedSaleCodes
		response.Message += fmt.Sprintf(", and completed %d sales", len(completedSaleCodes))
	}

	return response, nil
}
