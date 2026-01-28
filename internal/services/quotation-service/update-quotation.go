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
)

type UpdateQuotationRequest struct {
	IsVerifyPrice bool                `json:"is_verify_price"` // true = verify, if not verified can't create
	Quotations    []QuotationDocument `json:"quotations"`
}

type UpdateQuotationResponse struct {
	IsPass        bool   `json:"is_pass"`
	QuotationCode string `json:"quotation_code"`
}

func UpdateQuotation(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := UpdateQuotationRequest{}
	res := []UpdateQuotationResponse{}

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
	nowTruc := now.Truncate(24 * time.Hour)
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

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
	expiryDateTemp := time.Now().AddDate(0, 0, int(expiryDays))
	expiryDate := time.Date(expiryDateTemp.Year(), expiryDateTemp.Month(), expiryDateTemp.Day(), 0, 0, 0, 0, expiryDateTemp.Location())

	updateQuotations := []models.Quotation{}
	updateQuotationItems := []models.QuotationItem{}
	verifyReqMap := map[string]verifyService.VerifyApproveRequest{}

	for _, quotationReq := range req.Quotations {
		tempQuotation := quotationReq.Quotation

		if quotationReq.EffectiveDatePrice == nil {
			return nil, fmt.Errorf("effective date is required for quotation %s", quotationReq.QuotationCode)
		}

		if tempQuotation.ID == uuid.Nil {
			return nil, fmt.Errorf("quotation ID is required for update")
		}

		if tempQuotation.QuotationCode == "" {
			return nil, fmt.Errorf("quotation code is required for update")
		}

		// Only update timestamp and user for update
		tempQuotation.UpdateDate = &nowDateOnly
		tempQuotation.UpdateBy = user

		if quotationReq.Status != "TEMP" {
			tempQuotation.ExpirePriceDate = &expiryDate
			tempQuotation.ExpirePriceDay = int(expiryDays)
		}

		if tempQuotation.DeliveryDate != nil {
			deliveryDateOnly := time.Date(tempQuotation.DeliveryDate.Year(), tempQuotation.DeliveryDate.Month(), tempQuotation.DeliveryDate.Day(), 0, 0, 0, 0, tempQuotation.DeliveryDate.Location())
			tempQuotation.DeliveryDate = &deliveryDateOnly
		}

		// Convert EffectiveDatePrice to date-only format
		if tempQuotation.EffectiveDatePrice != nil {
			effectiveDateOnly := time.Date(tempQuotation.EffectiveDatePrice.Year(), tempQuotation.EffectiveDatePrice.Month(), tempQuotation.EffectiveDatePrice.Day(), 0, 0, 0, 0, tempQuotation.EffectiveDatePrice.Location())
			tempQuotation.EffectiveDatePrice = &effectiveDateOnly
		}

		updateQuotations = append(updateQuotations, tempQuotation)

		//Approval
		verifyReqKey := fmt.Sprintf(`%s|%s`, quotationReq.CompanyCode, quotationReq.SiteCode)
		verifyReq, existVerifyReq := verifyReqMap[verifyReqKey]
		if !existVerifyReq {
			newVerifyReq := verifyService.VerifyApproveRequest{
				IsVerifyPrice: true,
				CompanyCode:   quotationReq.CompanyCode,
				SiteCode:      quotationReq.SiteCode,
				StorageType:   []string{`NORMAL`},
				SaleDate:      nowTruc,
			}

			verifyReq = newVerifyReq
		}

		newApprDoc := verifyService.VerifyApproveDocument{
			DocRef:             tempQuotation.QuotationCode,
			CustomerCode:       quotationReq.CustomerCode,
			EffectiveDatePrice: *quotationReq.EffectiveDatePrice,
			TransportCost:      quotationReq.TotalTransportCost,
			TransportType:      quotationReq.TransportCostType,
			TotalAmount:        quotationReq.SubtotalExclVat,
			TotalWeight:        quotationReq.TotalWeight,
			Items:              []verifyService.VerifyApproveItem{},
		}

		for _, item := range quotationReq.Items {
			// Generate new ID for each item (updateInit approach)
			item.ID = uuid.New()
			item.QuotationID = tempQuotation.ID

			if item.QuotationItem == "" {
				item.QuotationItem = uuid.New().String()
			}

			// Set create/update timestamps and user
			item.CreateDate = &nowDateOnly
			item.CreateBy = user
			item.UpdateDate = &nowDateOnly
			item.UpdateBy = user

			updateQuotationItems = append(updateQuotationItems, item)

			//Approval
			newApprItem := verifyService.VerifyApproveItem{
				ItemRef:       item.QuotationItem,
				ProductCode:   item.ProductCode,
				Qty:           item.Qty,
				Unit:          item.Unit,
				TotalWeight:   item.TotalWeight,
				PriceUnit:     item.PriceUnit,
				PriceListUnit: item.PriceListUnit,
				TotalAmount:   item.SubtotalExclVat,
				SaleUnit:      item.SaleUnit,
				SaleUnitType:  item.SaleUnitType,
			}

			newApprDoc.Items = append(newApprDoc.Items, newApprItem)
		}

		//Approval
		verifyReq.Documents = append(verifyReq.Documents, newApprDoc)
		verifyReqMap[verifyReqKey] = verifyReq
	}

	//Verification
	if req.IsVerifyPrice {
		for _, verifyReq := range verifyReqMap {
			verifyRes, err := verifyService.VerifyApproveLogic(gormx, sqlx, verifyReq)
			if err != nil {
				return nil, err
			}

			for _, doc := range verifyRes.Documents {
				res = append(res, UpdateQuotationResponse{
					IsPass:        doc.IsPassPrice,
					QuotationCode: doc.DocRef,
				})

				for i := range updateQuotations {
					if !doc.IsPassPrice {
						updateQuotations[i].IsApproved = false
						updateQuotations[i].StatusApprove = "PENDING"
					} else {
						updateQuotations[i].IsApproved = true
						updateQuotations[i].StatusApprove = "COMPLETED"
					}
				}
			}
		}
	} else {
		// If not verifying price, create response for all quotations with default values
		for _, quotation := range updateQuotations {
			res = append(res, UpdateQuotationResponse{
				IsPass:        true, // Default to true when not verifying
				QuotationCode: quotation.QuotationCode,
			})
		}
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

	// Update quotations
	for _, quotation := range updateQuotations {
		if err := tx.Model(&models.Quotation{}).
			Where("id = ?", quotation.ID).
			Updates(quotation).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to update quotation %s: %v", quotation.QuotationCode, err)
		}
	}

	// Delete existing quotation items and insert new ones (updateInit approach)
	for _, quotation := range updateQuotations {
		// Delete existing items
		if err := tx.Where("quotation_id = ?", quotation.ID).Delete(&models.QuotationItem{}).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to delete existing quotation items: %v", err)
		}
	}

	// Insert new items
	if len(updateQuotationItems) > 0 {
		if err := tx.Create(&updateQuotationItems).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to create quotation items: %v", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return res, nil
}
