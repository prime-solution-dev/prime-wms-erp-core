package quotationService

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	verifyService "prime-erp-core/internal/services/verify-service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ReviseQuotation ออกใบ revision ใหม่จากใบเดิมในทีเดียว
//
// เดิมหน้าบ้านยิง 3 เส้นเรียงกันเอง (CancelQuotation -> CreateQuotation ->
// EditQuotation) โดยไม่มี transaction คร่อม ถ้าเส้นไหนล้มกลางทางจะได้สภาพครึ่งๆ
// เคสที่เจอจริงคือเส้นที่ 3 ยิงด้วย quotation_code ว่าง ใบใหม่เลยค้างเป็น
// status_approve = COMPLETED ทั้งที่ยังไม่เคยผ่านการอนุมัติ และแก้ไขอะไรไม่ได้
//
// เส้นนี้รวมทั้งสามขั้นไว้ใน transaction เดียว และบังคับสถานะอนุมัติของใบใหม่
// เป็น PENDING ตั้งแต่ตอนสร้าง จึงไม่ต้องมีขั้นตอนดันสถานะกลับตามหลังอีก
//
// ตั้งใจไม่แตะ CancelQuotation / CreateQuotation / EditQuotation ของเดิม
// เพราะเส้นอื่นยังเรียกใช้อยู่
type ReviseQuotationRequest struct {
	// ID ของใบเดิมที่จะถูกยกเลิกและออก revision ต่อ
	ID uuid.UUID `json:"id"`
	// เนื้อใบใหม่ ใช้ payload ชุดเดียวกับ CreateQuotation
	// ฟิลด์ที่เกี่ยวกับเลขที่เอกสารและสถานะอนุมัติจะถูก server เขียนทับทั้งหมด
	Quotation QuotationDocument `json:"quotation"`
}

type ReviseQuotationResponse struct {
	QuotationCode string  `json:"quotation_code"`
	Revision      float64 `json:"revision"`
}

func ReviseQuotation(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := ReviseQuotationRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	if req.ID == uuid.Nil {
		return nil, errors.New("id of the quotation being revised is required")
	}

	gormx, err := db.ConnectGORM(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	user := ctx.GetString("user")
	if user == "" {
		user = `system` // fallback
	}

	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	expiryDays, err := getExpiryPriceDays(gormx)
	if err != nil {
		return nil, err
	}

	expiryDateTemp := now.AddDate(0, 0, int(expiryDays))
	expiryDate := time.Date(expiryDateTemp.Year(), expiryDateTemp.Month(), expiryDateTemp.Day(), 0, 0, 0, 0, expiryDateTemp.Location())

	res := ReviseQuotationResponse{}

	err = gormx.Transaction(func(tx *gorm.DB) error {
		var previous models.Quotation
		if err := tx.Where("id = ?", req.ID).Take(&previous).Error; err != nil {
			return fmt.Errorf("quotation not found: %v", err)
		}

		// ด่านเดียวกับ CancelQuotation / EditQuotation ใบที่แปลงเป็น sale order
		// ไปแล้วหรือยกเลิกไปแล้ว ห้ามออก revision ต่อ
		switch previous.Status {
		case "COMPLETED":
			return errors.New("quotation already converted to a sale order and cannot be revised")
		case "CANCELED":
			return errors.New("quotation is canceled and cannot be revised")
		}

		// เลข revision คิดจากใบเดิมที่อ่านมาสดๆ ใน transaction ไม่ใช่จากหน้าบ้าน
		// เดิมหน้าบ้านต่อสตริง `-R{revision+1}` เองโดยไม่มีใครยืนยันฝั่ง server
		codeRef := previous.QuotationCodeRef
		if codeRef == "" {
			codeRef = previous.QuotationCode
		}
		newRevision := previous.Revision + 1
		newCode := fmt.Sprintf("%s-R%d", codeRef, int(newRevision))

		var existCount int64
		if err := tx.Model(&models.Quotation{}).
			Where("quotation_code = ?", newCode).
			Count(&existCount).Error; err != nil {
			return err
		}
		if existCount > 0 {
			return fmt.Errorf("quotation %s already exists", newCode)
		}

		// ยกเลิกใบเดิม (หัว + item) แบบเดียวกับ CancelQuotation
		if err := tx.Model(&models.Quotation{}).
			Where("id = ?", previous.ID).
			Updates(map[string]interface{}{
				"status":         "CANCELED",
				"status_approve": "CANCELED",
				"update_date":    nowDateOnly,
				"update_by":      user,
			}).Error; err != nil {
			return fmt.Errorf("failed to cancel the previous quotation: %v", err)
		}

		if err := tx.Model(&models.QuotationItem{}).
			Where("quotation_id = ?", previous.ID).
			Updates(map[string]interface{}{
				"status":      "CANCELED",
				"update_date": nowDateOnly,
				"update_by":   user,
			}).Error; err != nil {
			return fmt.Errorf("failed to cancel the previous quotation items: %v", err)
		}

		newQuotation := req.Quotation.Quotation
		newQuotation.ID = uuid.New()
		newQuotation.QuotationCode = newCode
		newQuotation.QuotationCodeRef = codeRef
		newQuotation.Revision = newRevision

		// ใบ revision ต้องเริ่มที่ยังไม่อนุมัติเสมอ ไม่ผ่าน CheckAutoApproval และ
		// ไม่ผ่าน VerifyApproveLogic เหมือนเส้น create ปกติ เพราะผู้ใช้เพิ่งกดขอแก้ไข
		// ใบนี้ ยังไงก็ต้องกลับไปเข้าคิวอนุมัติใหม่หลังแก้เสร็จ
		newQuotation.Status = "PENDING"
		newQuotation.StatusApprove = "PENDING"
		newQuotation.IsApproved = false

		newQuotation.CreateDate = &nowDateOnly
		newQuotation.CreateBy = user
		newQuotation.UpdateDate = &nowDateOnly
		newQuotation.UpdateBy = user
		newQuotation.EffectiveDatePrice = &nowDateOnly
		newQuotation.ExpirePriceDate = &expiryDate
		newQuotation.ExpirePriceDay = int(expiryDays)

		if err := tx.Create(&newQuotation).Error; err != nil {
			return fmt.Errorf("failed to create the new revision: %v", err)
		}

		newItems := make([]models.QuotationItem, 0, len(req.Quotation.Items))
		for _, item := range req.Quotation.Items {
			item.ID = uuid.New()
			item.QuotationID = newQuotation.ID

			if item.QuotationItem == "" {
				item.QuotationItem = uuid.New().String()
			}

			item.CreateDate = &nowDateOnly
			item.CreateBy = user
			item.UpdateDate = &nowDateOnly
			item.UpdateBy = user

			newItems = append(newItems, item)
		}

		if len(newItems) > 0 {
			if err := tx.Create(&newItems).Error; err != nil {
				return fmt.Errorf("failed to create the new revision items: %v", err)
			}
		}

		res.QuotationCode = newCode
		res.Revision = newRevision

		return nil
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

// getExpiryPriceDays อ่านอายุราคาจาก system config ชุดเดียวกับที่ CreateQuotation ใช้
func getExpiryPriceDays(gormx *gorm.DB) (int64, error) {
	topic := `PRICE`
	configCodes := []string{`EXPIRY_PRICE_DAYS`}

	configMap, err := verifyService.GetConfigSystem(gormx, topic, configCodes)
	if err != nil {
		return 0, err
	}

	expiryDaysConfig, exists := configMap[fmt.Sprintf(`%s|%s`, topic, `EXPIRY_PRICE_DAYS`)]
	if !exists {
		return 0, errors.New("missing configuration for expiry price days")
	}

	expiryDays, err := strconv.ParseInt(expiryDaysConfig.Value, 10, 64)
	if err != nil {
		return 0, errors.New("failed to convert expiry days to int64: " + err.Error())
	}

	return expiryDays, nil
}
