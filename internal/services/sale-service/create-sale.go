package saleService

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"prime-erp-core/internal/requestcontext"
	approvalService "prime-erp-core/internal/services/approval-service"
	systemConfigService "prime-erp-core/internal/services/system-config"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CreateSaleRequest struct {
	IsVerifyPrice      bool   `json:"is_verify_price"`       // true = verify, if not verified can't create
	IsVerifyCredit     bool   `json:"is_verify_credit"`      // true = verify, if not verified can't create
	IsVerifyExpiryDate bool   `json:"is_verify_expiry_date"` // true = verify, if not verified can't create
	IsVerifyInventory  bool   `json:"is_verify_inventory"`
	QuotationID        string `json:"quotation_id"`
	User               string `json:"user"`
	Status             string `json:"status"` // Status ที่หน้าบ้านส่งมา (APPROVED หรือ WAIT_FOR_APPROVED)
	Sales              []SaleDocument
}

type SaleDocument struct {
	models.Sale
	Items       []models.SaleItem
	SaleDeposit []models.SaleDeposit
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

func CreateSale(ctx context.Context, jsonPayload string) (interface{}, error) {
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

	// user คือชื่อที่ client ส่งมาใน body ใช้ได้เฉพาะด่านอนุมัติ (RequestUserCode / CheckAutoApproval)
	// ซึ่งเป็นเรื่องสิทธิ์ที่เจ้าของงานเป็นคนกำหนด ไม่ใช่เรื่องของ refactor นี้
	//
	// ส่วนชื่อที่บันทึกลง DB ต้องมาจาก token ของคนที่กดจริง ไม่ใช่จาก body
	// ไม่งั้นใครยิง API ตรงๆ ก็เขียนชื่อคนอื่นลง create_by ได้ (กติกาข้อ 3 ของ spec)
	// ถ้าไม่มี token มาด้วยจริงๆ ค่อยใช้ค่าจาก body เป็นตัวสำรอง จะได้ไม่เขียนค่าว่างทับของเดิม
	user := req.User

	auditUser := auditUserFrom(ctx, user)
	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	createSales := []models.Sale{}
	createSaleItems := []models.SaleItem{}
	createSaleDeposits := []models.SaleDeposit{}

	// Generate all sale codes first
	saleCodes, err := generateSaleCodes(gormx, len(req.Sales))
	if err != nil {
		return nil, err
	}

	// ใช้ status ที่หน้าบ้านส่งมา
	statusApprove := "PROCESS"
	isApproved := false
	status := "PENDING"
	if req.Status == "APPROVED" {
		statusApprove = "COMPLETED"
		isApproved = true
		status = "PENDING"
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
		tempSale.CreateBy = auditUser
		tempSale.UpdateDate = &nowDateOnly
		tempSale.UpdateBy = auditUser
		// ใช้ status จากหน้าบ้าน
		tempSale.Status = status
		tempSale.StatusApprove = statusApprove
		tempSale.IsApproved = isApproved

		// Check auto approval for PENDING status
		if status == "PENDING" && statusApprove != "COMPLETED" {
			autoApprovalReq := approvalService.CheckAutoApprovalRequest{
				RequestUserCode: user,
				ModuleCode:      "CUSTOMIZE",
				TopicCode:       "CUSTOMIZE",
				MdItemCode:      "CTM-CTM5",
				CondRangeMin:    saleReq.TotalAmount,
			}

			autoApprovalRes, err := approvalService.CheckAutoApproval(ctx, gormx, autoApprovalReq, user)
			if err != nil {
				return nil, err
			}

			if autoApprovalRes.IsAutoApproved {
				tempSale.IsApproved = true
				tempSale.StatusApprove = "COMPLETED"
			} else {
				tempSale.IsApproved = false
				tempSale.StatusApprove = "PENDING"
			}
		}

		// ผลตรวจมาจากหน้าบ้าน (ยิง ValidateSaleOrder ก่อน convert) ห้ามเขียนทับเป็น Y
		// ไม่งั้นใบที่ราคาไม่ผ่านจะดูเหมือนผ่าน และจอรออนุมัติจะหาใบไม่เจอ
		tempSale.PassPriceList = normalizePassFlag(tempSale.PassPriceList)
		tempSale.PassPriceExpire = normalizePassFlag(tempSale.PassPriceExpire)
		tempSale.PassCreditLimit = normalizePassFlag(tempSale.PassCreditLimit)
		tempSale.PassAtpCheck = normalizePassFlag(tempSale.PassAtpCheck)

		createSales = append(createSales, tempSale)

		for _, item := range saleReq.Items {
			item.ID = uuid.New()
			item.SaleID = tempSale.ID

			saleItem := uuid.New().String()

			if item.SaleItem == "" {
				item.SaleItem = saleItem
			}

			item.CreateDate = &nowDateOnly
			item.CreateBy = auditUser
			item.UpdateDate = &nowDateOnly
			item.UpdateBy = auditUser

			createSaleItems = append(createSaleItems, item)
		}

		for _, deposit := range saleReq.SaleDeposit {
			deposit.ID = uuid.New()
			deposit.SaleID = tempSale.ID

			createSaleDeposits = append(createSaleDeposits, deposit)
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
				"update_by":   auditUser,
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
				"update_by":   auditUser,
			}).Error; err != nil {
			tx.Rollback()
			return nil, errors.New("failed to update quotation items status: " + err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// ถ้า status เป็น WAIT_FOR_APPROVED ให้ส่ง sale id ไปสร้าง RequestApproveSale
	//
	// ทำงานหลัง tx.Commit() แล้ว และเป็น best-effort (พังแล้ว log ทิ้ง ไม่ทำให้ CreateSale ทั้งก้อนพัง)
	// ต้องใช้ postCommitContext ไม่งั้น caller ตัดสายกลางทาง (เช่น timeout ฝั่งเว็บ) จะทำให้
	// approval request ไม่ถูกสร้างเงียบๆ ทั้งที่ sale ถูกสร้างไปแล้วจริง
	if req.Status == "WAIT_FOR_APPROVED" {
		postCommitCtx := postCommitContext(ctx)
		for _, sale := range createSales {
			requestApproveReq := RequestApproveSaleRequest{
				ID: sale.ID,
			}
			approvePayload, err := json.Marshal(requestApproveReq)
			if err != nil {
				fmt.Printf("Warning: failed to marshal request approve sale: %v\n", err)
				continue
			}

			_, err = RequestApproveSale(postCommitCtx, string(approvePayload))
			if err != nil {
				fmt.Printf("Warning: failed to create approval request for sale %s: %v\n", sale.SaleCode, err)
			}
		}
	}

	return res, nil
}

// postCommitContext คืน context สำหรับงานที่ทำหลัง commit
// เก็บ user/token ไว้ครบ แต่ตัดการยกเลิกทิ้ง ไม่งั้นพอ caller หมดเวลาแล้วตัดสาย
// งานที่เหลือจะไม่เกิดขึ้นเลยและเงียบด้วย
func postCommitContext(ctx context.Context) context.Context {
	return context.WithoutCancel(ctx)
}

// generateSaleCodes จองเลขที่เอกสารแบบ atomic (ล็อกแถว config จนกว่าจะเดินเลขเสร็จ)
// เดิมแยกเป็นอ่านเลขตอนนี้ แล้วค่อยเดินเลขหลัง insert ซึ่งทำให้สองคนที่กดพร้อมกันได้เลขเดียวกัน
func generateSaleCodes(gormx *gorm.DB, count int) ([]string, error) {
	if count <= 0 {
		return []string{}, nil
	}

	codes, err := systemConfigService.ReserveRunningCodes(
		gormx, "RUNNING_SO", count, "", systemConfigService.StandardRunningPeriod())
	if err != nil {
		return nil, fmt.Errorf("failed to generate sale codes: %v", err)
	}

	if len(codes) != count {
		return nil, errors.New("failed to get correct number of sale codes from system config")
	}

	return codes, nil
}

// auditUserFrom เลือกชื่อที่จะเขียนลง create_by / update_by
//
// token ของคนที่กดมาก่อนเสมอ ค่าจาก body เป็นแค่ตัวสำรองของเส้นที่ยังไม่ส่ง token มา
func auditUserFrom(ctx context.Context, bodyUser string) string {
	if user := requestcontext.GetUserOrDefault(ctx); user != "" {
		return user
	}

	return bodyUser
}
