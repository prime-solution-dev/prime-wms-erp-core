package saleService

import (
	"encoding/json"
	"errors"
	"fmt"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	systemConfigService "prime-erp-core/internal/services/system-config"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CreateSaleRequest struct {
	IsVerifyPrice      bool   `json:"is_verify_price"`       // true = verify, if not verified can't create
	IsVerifyCredit     bool   `json:"is_verify_credit"`      // true = verify, if not verified can't create
	IsVerifyExpiryDate bool   `json:"is_verify_expiry_date"` // true = verify, if not verified can't create
	IsVerifyInventory  bool   `json:"is_verify_inventory"`
	QuotationID        string `json:"quotation_id"`
	Status             string `json:"status"` // Status ที่หน้าบ้านส่งมา (APPROVED หรือ WAIT_FOR_APPROVED)
	Sales              []SaleDocument
}

type SaleDocument struct {
	models.Sale
	Items []models.SaleItem
}

type CreateSaleResponse struct {
	IsPass           bool   `json:"is_pass"`
	IsPassPrice      bool   `json:"is_pass_price"`
	IsPassCredit     bool   `json:"is_pass_credit"`
	IsPassInventory  bool   `json:"is_pass_inventory"`
	IsPassExpiryDate bool   `json:"is_pass_expiry_date"`
	SaleCode         string `json:"sale_code"`
	Status           string `json:"status"`  // Status ของ Sale ที่สร้างแล้ว
	Message          string `json:"message"` // ข้อความแจ้งผลลัพธ์
}

func CreateSale(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := CreateSaleRequest{}
	res := []CreateSaleResponse{}

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

	user := `system` // TODO: get from ctx
	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	createSales := []models.Sale{}
	createSaleItems := []models.SaleItem{}

	// Generate all sale codes first
	saleCodes, err := generateSaleCodes(ctx, len(req.Sales))
	if err != nil {
		return nil, err
	}

	// ใช้ status ที่หน้าบ้านส่งมา
	statusApprove := "PENDING"
	isApproved := false
	status := "PENDING"
	if req.Status == "APPROVED" {
		statusApprove = "COMPLETED"
		isApproved = true
		status = "COMPLETED"
	}

	for i, saleReq := range req.Sales {
		tempSale := saleReq.Sale
		tempSale.ID = uuid.New()

		// Use pre-generated sale code
		saleCode := saleCodes[i]

		if tempSale.SaleCode == "" {
			tempSale.SaleCode = saleCode
		}

		tempSale.CreateDate = &nowDateOnly
		tempSale.CreateBy = user
		tempSale.UpdateDate = &nowDateOnly
		tempSale.UpdateBy = user
		// ใช้ status จากหน้าบ้าน
		tempSale.Status = status
		tempSale.StatusApprove = statusApprove
		tempSale.IsApproved = isApproved

		// ตั้งค่า validation flags เป็นค่าเริ่มต้น (หน้าบ้านได้ validate แล้ว)
		tempSale.PassPriceList = "Y"
		tempSale.PassPriceExpire = "Y"
		tempSale.PassCreditLimit = "Y"
		tempSale.PassAtpCheck = "Y"

		createSales = append(createSales, tempSale)

		for _, item := range saleReq.Items {
			item.ID = uuid.New()
			item.SaleID = tempSale.ID

			saleItem := uuid.New().String()

			if item.SaleItem == "" {
				item.SaleItem = saleItem
			}

			item.CreateDate = &nowDateOnly
			item.CreateBy = user
			item.UpdateDate = &nowDateOnly
			item.UpdateBy = user

			createSaleItems = append(createSaleItems, item)
		}

		// เพิ่มข้อมูลใน response
		statusMessage := "Sale Order สร้างสำเร็จ"
		if req.Status == "WAIT_FOR_APPROVED" {
			statusMessage = "Sale Order สร้างสำเร็จ - รออนุมัติ"
		}

		res = append(res, CreateSaleResponse{
			IsPass:           true, // หน้าบ้านได้ validate แล้วถึงส่งมา
			IsPassPrice:      true,
			IsPassCredit:     true,
			IsPassInventory:  true,
			IsPassExpiryDate: true,
			SaleCode:         saleCode,
			Status:           req.Status,
			Message:          statusMessage,
		})
	}

	// check duplicate sale codes
	var existCount int64
	codes := make([]string, 0, len(createSales))
	for _, s := range createSales {
		codes = append(codes, s.SaleCode)
	}

	tx := gormx.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if len(codes) > 0 {
		if err := tx.Model(&models.Sale{}).
			Where("sale_code IN ?", codes).
			Count(&existCount).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		if existCount > 0 {
			tx.Rollback()
			return nil, errors.New("duplicate sale code detected")
		}
	}

	// Insert sales
	if len(createSales) > 0 {
		if err := tx.Create(&createSales).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Insert sale items
	if len(createSaleItems) > 0 {
		if err := tx.Create(&createSaleItems).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Update quotation status to COMPLETED if QuotationID is provided
	if req.QuotationID != "" {
		// Parse QuotationID to UUID
		quotationUUID, err := uuid.Parse(req.QuotationID)
		if err != nil {
			tx.Rollback()
			return nil, errors.New("invalid quotation_id format: " + err.Error())
		}

		// Update quotation status
		if err := tx.Model(&models.Quotation{}).
			Where("id = ?", quotationUUID).
			Updates(map[string]interface{}{
				"status":      "COMPLETED",
				"update_date": &nowDateOnly,
				"update_by":   user,
			}).Error; err != nil {
			tx.Rollback()
			return nil, errors.New("failed to update quotation status: " + err.Error())
		}

		// Update quotation items status
		if err := tx.Model(&models.QuotationItem{}).
			Where("quotation_id = ?", quotationUUID).
			Updates(map[string]interface{}{
				"status":      "COMPLETED",
				"update_date": &nowDateOnly,
				"update_by":   user,
			}).Error; err != nil {
			tx.Rollback()
			return nil, errors.New("failed to update quotation items status: " + err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Update running number after successful creation
	if err := updateSaleRunningConfig(ctx, len(createSales)); err != nil {
		// Log error but don't fail the transaction as sales are already created
		fmt.Printf("Warning: failed to update running config: %v\n", err)
	}

	return res, nil
}

// updateSaleRunningConfig updates the running number configuration for sales
func updateSaleRunningConfig(ctx *gin.Context, count int) error {
	if count <= 0 {
		return nil // No sales created, nothing to update
	}

	updateReq := systemConfigService.UpdateRunningSystemConfigRequest{
		ConfigCode: "RUNNING_SO",
		Count:      count,
	}

	reqJSON, err := json.Marshal(updateReq)
	if err != nil {
		return fmt.Errorf("failed to marshal update request: %v", err)
	}

	_, err = systemConfigService.UpdateRunningSystemConfig(ctx, string(reqJSON))
	if err != nil {
		return fmt.Errorf("failed to update running config: %v", err)
	}

	return nil
}

// generateSaleCodes generates sale codes using system config
func generateSaleCodes(ctx *gin.Context, count int) ([]string, error) {
	if count <= 0 {
		return []string{}, nil // No sales to generate codes for
	}

	getReq := systemConfigService.GetRunningSystemConfigRequest{
		ConfigCode: "RUNNING_SO",
		Count:      count,
	}

	reqJSON, err := json.Marshal(getReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get request: %v", err)
	}

	saleCodeResponse, err := systemConfigService.GetRunningSystemConfig(ctx, string(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to generate sale codes: %v", err)
	}

	saleResult, ok := saleCodeResponse.(systemConfigService.GetRunningSystemConfigResponse)
	if !ok || len(saleResult.Data) != count {
		return nil, errors.New("failed to get correct number of sale codes from system config")
	}

	return saleResult.Data, nil
}
