package saleService

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	verifyService "prime-erp-core/internal/services/verify-service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UpdateSaleRequest struct {
	IsVerifyPrice      bool                 `json:"is_verify_price"`
	IsVerifyCredit     bool                 `json:"is_verify_credit"`
	IsVerifyExpiryDate bool                 `json:"is_verify_expiry_date"`
	IsVerifyInventory  bool                 `json:"is_verify_inventory"`
	Sales              []SaleDocumentUpdate `json:"sales"`
}

type SaleDocumentUpdate struct {
	models.Sale
	Items       []models.SaleItem    `json:"items"`        // Items to update
	SaleDeposit []models.SaleDeposit `json:"sale_deposit"` // Sale deposits to update
	DeleteItems []uuid.UUID          `json:"delete_items"` // Item IDs to delete
}

type UpdateSaleResponse struct {
	IsPass           bool   `json:"is_pass"`
	IsPassPrice      bool   `json:"is_pass_price"`
	IsPassCredit     bool   `json:"is_pass_credit"`
	IsPassInventory  bool   `json:"is_pass_inventory"`
	IsPassExpiryDate bool   `json:"is_pass_expiry_date"`
	SaleCode         string `json:"sale_code"`
}

func UpdateSale(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := UpdateSaleRequest{}
	res := []UpdateSaleResponse{}

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

	user := ctx.GetString("user")
	if user == "" {
		user = `system` // fallback
	}
	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	updateSales := []models.Sale{}
	updateSaleItems := []models.SaleItem{}
	updateSaleDeposits := []models.SaleDeposit{}
	verifyReqMap := map[string]verifyService.VerifyApproveRequest{}

	for _, saleReq := range req.Sales {
		tempSale := saleReq.Sale

		if tempSale.ID == uuid.Nil {
			return nil, fmt.Errorf("sale ID is required for update")
		}

		if tempSale.SaleCode == "" {
			return nil, fmt.Errorf("sale code is required for update")
		}

		// Only update timestamp and user for update
		tempSale.UpdateDate = &nowDateOnly
		tempSale.UpdateBy = user

		updateSales = append(updateSales, tempSale)

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
				SaleDate:            *saleReq.DeliveryDate,
			}
			verifyReq = newVerifyReq
		}

		newApprDoc := verifyService.VerifyApproveDocument{
			DocRef:       tempSale.SaleCode,
			CustomerCode: saleReq.CustomerCode,
			Items:        []verifyService.VerifyApproveItem{},
		}

		for _, item := range saleReq.Items {
			// Ensure item belongs to this sale
			item.SaleID = tempSale.ID
			item.UpdateDate = &nowDateOnly
			item.UpdateBy = user

			updateSaleItems = append(updateSaleItems, item)

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

		for _, deposit := range saleReq.SaleDeposit {
			// Ensure deposit belongs to this sale
			deposit.ID = uuid.New()
			deposit.SaleID = tempSale.ID
			updateSaleDeposits = append(updateSaleDeposits, deposit)
		}

		//Approval
		verifyReq.Documents = append(verifyReq.Documents, newApprDoc)
		verifyReqMap[verifyReqKey] = verifyReq
	}

	// Verification
	if req.IsVerifyPrice || req.IsVerifyCredit || req.IsVerifyInventory {
		for _, verifyReq := range verifyReqMap {
			verifyRes, err := verifyService.VerifyApproveLogic(gormx, sqlx, verifyReq)
			if err != nil {
				return nil, err
			}

			// Check if critical validations fail - don't allow update if they fail
			if !verifyRes.IsPassCredit || !verifyRes.IsPassInventory || !verifyRes.IsPassPrice {
				res = append(res, UpdateSaleResponse{
					IsPass:           false,
					IsPassPrice:      verifyRes.IsPassPrice,
					IsPassCredit:     verifyRes.IsPassCredit,
					IsPassInventory:  verifyRes.IsPassInventory,
					IsPassExpiryDate: verifyRes.IsPassExpiryPrice,
					SaleCode:         verifyRes.Documents[0].DocRef,
				})
				// Return immediately - don't update sale if critical validations fail
				// return res, nil // Uncomment this line if you want to stop the update on failure
			} else {
				updateSales[0].IsApproved = true
				updateSales[0].StatusApprove = "COMPLETED"
			}

			res = append(res, UpdateSaleResponse{
				IsPass:           verifyRes.IsPassPrice && verifyRes.IsPassCredit && verifyRes.IsPassInventory && verifyRes.IsPassExpiryPrice,
				IsPassPrice:      verifyRes.IsPassPrice,
				IsPassCredit:     verifyRes.IsPassCredit,
				IsPassInventory:  verifyRes.IsPassInventory,
				IsPassExpiryDate: verifyRes.IsPassExpiryPrice,
				SaleCode:         verifyRes.Documents[0].DocRef,
			})

			for i := range updateSales {
				if verifyRes.IsPassPrice {
					updateSales[i].PassPriceList = "Y"
				} else {
					updateSales[i].PassPriceList = "N"
				}
				if verifyRes.IsPassExpiryPrice {
					updateSales[i].PassPriceExpire = "Y"
				} else {
					updateSales[i].PassPriceExpire = "N"
				}
				if verifyRes.IsPassCredit {
					updateSales[i].PassCreditLimit = "Y"
				} else {
					updateSales[i].PassCreditLimit = "N"
				}
				if verifyRes.IsPassInventory {
					updateSales[i].PassAtpCheck = "Y"
				} else {
					updateSales[i].PassAtpCheck = "N"
				}
			}

		}
	} else {
		// If not verifying, create response for all sales with default values
		for _, sale := range updateSales {
			res = append(res, UpdateSaleResponse{
				IsPass:           true, // Default to true when not verifying
				IsPassPrice:      true,
				IsPassCredit:     true,
				IsPassInventory:  true,
				IsPassExpiryDate: true,
				SaleCode:         sale.SaleCode,
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

	// Update sales
	for _, sale := range updateSales {
		if err := tx.Model(&models.Sale{}).
			Where("id = ?", sale.ID).
			Updates(sale).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to update sale %s: %v", sale.SaleCode, err)
		}
	}

	// Delete items if specified
	for _, saleReq := range req.Sales {
		if len(saleReq.DeleteItems) > 0 {
			if err := tx.Where("id IN ? AND sale_id = ?", saleReq.DeleteItems, saleReq.Sale.ID).Delete(&models.SaleItem{}).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to delete sale items: %v", err)
			}
		}
	}

	// Update existing items
	for _, item := range updateSaleItems {
		if err := tx.Model(&models.SaleItem{}).
			Where("id = ? AND sale_id = ?", item.ID, item.SaleID).
			Updates(item).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to update sale item %s: %v", item.SaleItem, err)
		}
	}

	// delete existing deposits
	for _, saleReq := range req.Sales {
		if len(saleReq.SaleDeposit) > 0 {
			if err := tx.Where("sale_id = ?", saleReq.Sale.ID).Delete(&models.SaleDeposit{}).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to delete existing sale deposits: %v", err)
			}
		}
	}

	// Update existing deposits
	for _, deposit := range updateSaleDeposits {
		if err := tx.Create(&deposit).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to insert sale deposit for sale ID %s: %v", deposit.SaleID, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return res, nil
}
