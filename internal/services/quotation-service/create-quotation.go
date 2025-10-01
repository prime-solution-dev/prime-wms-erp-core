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

type CreateQuotationRequest struct {
	IsVerifyPrice bool                `json:"is_verify_price"` // true = verify, if not verified can't create
	Quotations    []QuotationDocument `json:"quotations"`
}

type QuotationDocument struct {
	models.Quotation
	Items []models.QuotationItem `json:"items"`
}

type CreateQuotationResponse struct {
	IsPass        bool   `json:"is_pass"`
	QuotationCode string `json:"quotation_code"`
}

func CreateQuotation(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := CreateQuotationRequest{}
	res := []CreateQuotationResponse{}

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

	createQuotations := []models.Quotation{}
	createQuotationItems := []models.QuotationItem{}
	verifyReqMap := map[string]verifyService.VerifyApproveRequest{}

	for _, quotationReq := range req.Quotations {
		tempQuotation := quotationReq.Quotation
		tempQuotation.ID = uuid.New()

		if quotationReq.EffectiveDatePrice == nil {
			return nil, fmt.Errorf("effective date is required for quotation %s", quotationReq.QuotationCode)
		}

		quotationCode := uuid.New().String()

		if tempQuotation.QuotationCode == "" {
			tempQuotation.QuotationCode = quotationCode
		}

		tempQuotation.CreateDate = &nowDateOnly
		tempQuotation.CreateBy = user
		tempQuotation.UpdateDate = &nowDateOnly
		tempQuotation.UpdateBy = user
		tempQuotation.EffectiveDatePrice = &nowDateOnly

		if quotationReq.Status == "PENDING" {
			tempQuotation.ExpirePriceDate = &expiryDate
			tempQuotation.ExpirePriceDay = int(expiryDays)
		}

		// Convert DeliveryDate to date-only format if provided
		if tempQuotation.DeliveryDate != nil {
			deliveryDateOnly := time.Date(tempQuotation.DeliveryDate.Year(), tempQuotation.DeliveryDate.Month(), tempQuotation.DeliveryDate.Day(), 0, 0, 0, 0, tempQuotation.DeliveryDate.Location())
			tempQuotation.DeliveryDate = &deliveryDateOnly
		}

		createQuotations = append(createQuotations, tempQuotation)

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
			DocRef:             quotationCode,
			CustomerCode:       quotationReq.CustomerCode,
			EffectiveDatePrice: *quotationReq.EffectiveDatePrice,
			Items:              []verifyService.VerifyApproveItem{},
		}

		for _, item := range quotationReq.Items {
			item.ID = uuid.New()
			item.QuotationID = tempQuotation.ID

			if item.QuotationItem == "" {
				item.QuotationItem = uuid.New().String()
			}

			item.CreateDate = &nowDateOnly
			item.CreateBy = user
			item.UpdateDate = &nowDateOnly
			item.UpdateBy = user

			createQuotationItems = append(createQuotationItems, item)

			//Approval
			newApprItem := verifyService.VerifyApproveItem{
				ItemRef:       item.QuotationItem,
				ProductCode:   item.ProductCode,
				Qty:           item.Qty,
				Unit:          item.Unit,
				TotalWeight:   item.TotalWeight,
				PriceUnit:     item.PriceUnit,
				PriceListUnit: item.PriceListUnit,
				TotalAmount:   item.TotalAmount,
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
				res = append(res, CreateQuotationResponse{
					IsPass:        doc.IsPassPrice,
					QuotationCode: doc.DocRef,
				})

				for _, quotation := range createQuotations {
					if !doc.IsPassPrice {
						quotation.IsApproved = false
						quotation.StatusApprove = "PENDING"
					} else {
						quotation.IsApproved = true
						quotation.StatusApprove = "COMPLETED"
					}
				}
			}
		}
	} else {
		// If not verifying price, create response for all quotations with default values
		for _, quotation := range createQuotations {
			res = append(res, CreateQuotationResponse{
				IsPass:        true, // Default to true when not verifying
				QuotationCode: quotation.QuotationCode,
			})
		}
	}

	// check duplicate quotation codes
	var existCount int64
	codes := make([]string, 0, len(createQuotations))
	for _, q := range createQuotations {
		codes = append(codes, q.QuotationCode)
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
		if err := tx.Model(&models.Quotation{}).
			Where("quotation_code IN ?", codes).
			Count(&existCount).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		if existCount > 0 {
			tx.Rollback()
			return nil, errors.New("duplicate quotation code detected")
		}
	}

	// Insert quotations
	if len(createQuotations) > 0 {
		if err := tx.Create(&createQuotations).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Insert items
	if len(createQuotationItems) > 0 {
		if err := tx.Create(&createQuotationItems).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return res, nil
}
