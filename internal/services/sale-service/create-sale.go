package saleService

import (
	"encoding/json"
	"errors"
	"fmt"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	verifyService "prime-erp-core/internal/services/verify-service"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CreateSaleRequest struct {
	IsVerifyPrice      bool `json:"is_verify_price"`       // true = verify, if not verified can't create
	IsVerifyCredit     bool `json:"is_verify_credit"`      // true = verify, if not verified can't create
	IsVerifyExpiryDate bool `json:"is_verify_expiry_date"` // true = verify, if not verified can't create
	IsVerifyInventory  bool `json:"is_verify_inventory"`
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
	nowTruc := now.Truncate(24 * time.Hour)

	createSales := []models.Sale{}
	createSaleItems := []models.SaleItem{}
	verifyReqMap := map[string]verifyService.VerifyApproveRequest{}

	for _, saleReq := range req.Sales {
		tempSale := saleReq.Sale
		tempSale.ID = uuid.New()

		saleCode := uuid.New().String()

		if tempSale.SaleCode == "" {
			tempSale.SaleCode = saleCode
		}

		tempSale.CreateDate = &now
		tempSale.CreateBy = user
		tempSale.UpdateDate = &now
		tempSale.UpdateBy = user
		tempSale.StatusApprove = "COMPLETED"
		tempSale.IsApproved = true
		tempSale.Status = "PENDING"

		createSales = append(createSales, tempSale)

		//Approval
		verifyReqKey := fmt.Sprintf(`%s|%s`, saleReq.CompanyCode, saleReq.SiteCode)
		verifyReq, existVerifyReq := verifyReqMap[verifyReqKey]
		if !existVerifyReq {
			newVerifyReq := verifyService.VerifyApproveRequest{
				IsVerifyPrice:       req.IsVerifyPrice,
				IsVerifyCredit:      req.IsVerifyCredit,
				IsVerifyExpiryPrice: req.IsVerifyExpiryDate,
				IsVerifyInventory:   req.IsVerifyInventory,
				CompanyCode:         saleReq.CompanyCode,
				SiteCode:            saleReq.SiteCode,
				StorageType:         []string{`NORMAL`},
				SaleDate:            nowTruc,
			}
			verifyReq = newVerifyReq
		}

		newApprDoc := verifyService.VerifyApproveDocument{
			DocRef:       saleCode,
			CustomerCode: saleReq.CustomerCode,
			Items:        []verifyService.VerifyApproveItem{},
		}

		for _, item := range saleReq.Items {
			item.ID = uuid.New()
			item.SaleID = tempSale.ID
			item.CreateDate = &now
			item.CreateBy = user
			item.UpdateDate = &now
			item.UpdateBy = user

			createSaleItems = append(createSaleItems, item)

			//Approval
			newApprItem := verifyService.VerifyApproveItem{
				ItemRef:       item.SaleItem,
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

	// Verification
	for _, verifyReq := range verifyReqMap {
		verifyRes, err := verifyService.VerifyApproveLogic(gormx, sqlx, verifyReq)
		if err != nil {
			return nil, err
		}

		for _, doc := range verifyRes.Documents {
			// Check if critical validations fail - don't allow creation if they fail
			if !doc.IsPassCredit || !doc.IsPassInventory || !doc.IsPassExpiryPrice {
				res = append(res, CreateSaleResponse{
					IsPass:           false,
					IsPassPrice:      doc.IsPassPrice,
					IsPassCredit:     doc.IsPassCredit,
					IsPassInventory:  doc.IsPassInventory,
					IsPassExpiryDate: doc.IsPassExpiryPrice,
					SaleCode:         doc.DocRef,
				})
				// Return immediately - don't create sale if critical validations fail
				return res, nil
			}

			res = append(res, CreateSaleResponse{
				IsPass:           doc.IsPassPrice && doc.IsPassCredit && doc.IsPassInventory && doc.IsPassExpiryPrice,
				IsPassPrice:      doc.IsPassPrice,
				IsPassCredit:     doc.IsPassCredit,
				IsPassInventory:  doc.IsPassInventory,
				IsPassExpiryDate: doc.IsPassExpiryPrice,
				SaleCode:         doc.DocRef,
			})

			for _, sale := range createSales {
				if doc.IsPassPrice {
					sale.PassPriceList = "Y"
				} else {
					sale.PassPriceList = "N"
				}
				if doc.IsPassExpiryPrice {
					sale.PassPriceExpire = "Y"
				} else {
					sale.PassPriceExpire = "N"
				}
				if doc.IsPassCredit {
					sale.PassCreditLimit = "Y"
				} else {
					sale.PassCreditLimit = "N"
				}
				if doc.IsPassInventory {
					sale.PassAtpCheck = "Y"
				} else {
					sale.PassAtpCheck = "N"
				}
			}
		}
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

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return res, nil
}
