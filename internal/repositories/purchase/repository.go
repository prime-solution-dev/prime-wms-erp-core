package purchaseRepository

import (
	"math"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Create
func CreatePurchase(purchases []models.Purchase) error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&purchases).Error; err != nil {
			return err
		}
		return nil
	})
}

// Get
func GetPurchaseList(
	purchaseCodes []string,
	supplierCodes []string,
	statusApprove []string,
	statusPayment []string,
	statusPaymentIncomplete bool,
	productCodes []string,
	purchaseType []string,
	docRef []string,
	tradingRef []string,
	companyCode string,
	siteCode string,
	page int,
	pageSize int,
) ([]models.Purchase, int, int, int, int, error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	defer db.CloseGORM(gormx)

	var purchases []models.Purchase
	var totalRecords int64

	// Build base query
	query := gormx.Model(&models.Purchase{}).
		Where("company_code = ? AND site_code = ?", companyCode, siteCode)

	if len(purchaseCodes) > 0 {
		query = query.Where("purchase_code IN ?", purchaseCodes)
	}

	if len(supplierCodes) > 0 {
		query = query.Where("supplier_code IN ?", supplierCodes)
	}

	if len(statusApprove) > 0 {
		query = query.Where("status_approve IN ?", statusApprove)
	}

	if len(statusPayment) > 0 {
		query = query.Where("status_payment IN ?", statusPayment)
	}

	if statusPaymentIncomplete {
		query = query.Where("status_payment != ? OR status_payment IS NULL", "COMPLETED")
	}

	if len(productCodes) > 0 {
		sub := gormx.Model(&models.PurchaseItem{}).
			Select("1").
			Where("purchase.id = purchase_item.purchase_id").
			Where("product_code IN ?", productCodes)

		query = query.Where("EXISTS (?)", sub)
	}

	if len(purchaseType) > 0 {
		query = query.Where("purchase_type IN ?", purchaseType)
	}

	if len(docRef) > 0 {
		query = query.Where("doc_ref IN ?", docRef)
	}

	if len(tradingRef) > 0 {
		query = query.Where("trading_ref IN ?", tradingRef)
	}

	// Count total records (no preload needed)
	if err := query.Count(&totalRecords).Error; err != nil {
		return nil, 0, 0, 0, 0, err
	}

	if pageSize == 0 {
		pageSize = int(totalRecords)
	}

	// Apply pagination
	offset := (page - 1) * pageSize
	if err := query.
		Preload("PurchaseItems").
		Order(`
        CASE
            WHEN status = 'PENDING' AND status_approve = 'PENDING' THEN 1
            WHEN status = 'PENDING' AND status_approve = 'PROCESS' THEN 2
            WHEN status = 'PENDING' AND status_approve = 'COMPLETED' THEN 3
						WHEN status = 'PENDING' AND status_approve = 'REVIEW' THEN 4
						WHEN status = 'PENDING' AND status_approve = 'REJECT' THEN 5
						WHEN status = 'CANCELLED' THEN 6
						WHEN status = 'COMPLETED' THEN 7
						WHEN status = 'TEMP' THEN 8
						ELSE 9
        END ASC,
				create_dtm DESC
    `).
		Limit(pageSize).
		Offset(offset).
		Find(&purchases).Error; err != nil {
		return nil, 0, 0, 0, 0, err
	}

	totalPages := 0
	if totalRecords > 0 {
		totalPages = int(math.Ceil(float64(totalRecords) / float64(pageSize)))
	}

	if page == 0 {
		page = 1
	}

	return purchases, int(totalRecords), page, pageSize, totalPages, nil
}

func GetPurchaseListByGRFilter(
	supplierCodes []string,
	purchaseCodes []string,
	purchaseItemCodes []string,
	statusApprove []string,
	purchaseItemStatus []string,
	productCodes []string,
	notItems []models.ExceptPurchaseAndPurchaseItemRequest,
	companyCode string,
	siteCode string,
	page int,
	pageSize int,
) ([]models.Purchase, int, int, int, int, error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	defer db.CloseGORM(gormx)

	var purchases []models.Purchase
	var totalRecords int64

	// Build base query
	query := gormx.Model(&models.Purchase{}).
		Where("company_code = ? AND site_code = ?", companyCode, siteCode)

	if len(purchaseItemStatus) > 0 {
		query = query.Preload("PurchaseItems", "status IN ?", purchaseItemStatus)
	} else if len(purchaseItemCodes) > 0 {
		query = query.Preload("PurchaseItems", "purchase_item IN ?", purchaseItemCodes)
	} else {
		query = query.Preload("PurchaseItems")
	}

	if len(purchaseCodes) > 0 {
		sub := gormx.Model(&models.PurchaseItem{}).
			Select("1").
			Where("purchase.id = purchase_item.purchase_id").
			Where("purchase.purchase_code IN ?", purchaseCodes)

		query = query.Where("EXISTS (?)", sub)
	}

	if len(notItems) > 0 {
		for _, notItem := range notItems {
			subQuery := gormx.Model(&models.PurchaseItem{}).
				Select("1").
				Where("purchase.id = purchase_item.purchase_id").
				Where("purchase_item.purchase_item IN ?", notItem.PurchaseItemCodes)

			query = query.Where("NOT EXISTS (?)", subQuery)
		}
	}

	if len(supplierCodes) > 0 {
		query = query.Where("supplier_code IN ?", supplierCodes)
	}

	if len(statusApprove) > 0 {
		query = query.Where("status_approve IN ?", statusApprove)
	}

	if len(productCodes) > 0 {
		sub := gormx.Model(&models.PurchaseItem{}).
			Select("1").
			Where("purchase.id = purchase_item.purchase_id").
			Where("product_code IN ?", productCodes)

		query = query.Where("EXISTS (?)", sub)
	}

	// Count total records (no preload needed)
	if err := query.Count(&totalRecords).Error; err != nil {
		return nil, 0, 0, 0, 0, err
	}

	if pageSize == 0 {
		pageSize = int(totalRecords)
	}

	// Apply pagination
	offset := (page - 1) * pageSize
	if err := query.
		Limit(pageSize).
		Offset(offset).
		Find(&purchases).Error; err != nil {
		return nil, 0, 0, 0, 0, err
	}

	totalPages := 0
	if totalRecords > 0 {
		totalPages = int(math.Ceil(float64(totalRecords) / float64(pageSize)))
	}

	if page == 0 {
		page = 1
	}

	return purchases, int(totalRecords), page, pageSize, totalPages, nil
}

// Update
func UpdatePurchase(purchases []models.Purchase) (err error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		for _, purchase := range purchases {
			// Update purchase
			if err := tx.Model(&models.Purchase{}).
				Where("id = ?", purchase.ID).
				Updates(purchase).Error; err != nil {
				return err
			}

			// Delete old items
			if result := tx.Where("purchase_id = ?", purchase.ID).Delete(&models.PurchaseItem{}); result.Error != nil {
				return result.Error
			}

			// Insert new items
			for _, item := range purchase.PurchaseItems {
				item.PurchaseID = purchase.ID // Ensure foreign key is set
				if err := tx.Create(&item).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func UpdatePurchaseStatusApprove(purchases []models.UpdateStatusApprovePurchaseRequest) (err error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		for _, purchase := range purchases {
			if result := tx.Model(&models.Purchase{}).
				Where("id = ?", purchase.ID).
				Updates(map[string]interface{}{
					"status_approve": purchase.StatusApprove,
					"is_approved":    purchase.IsApproved,
					"update_dtm":     time.Now().UTC(),
				}); result.Error != nil {
				err = result.Error
			}
		}
		return nil
	})
}

func CompletePOPayment(purchaseCodes []string, purchaseItems []string) (err error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		if len(purchaseCodes) > 0 {
			var purchaseItemCodes []string
			subQuery := tx.Model(&models.Purchase{}).
				Select("id").
				Where("purchase_code IN ?", purchaseCodes)

			if err := tx.Model(&models.PurchaseItem{}).
				Where("purchase_id IN (?)", subQuery).
				Pluck("purchase_item", &purchaseItemCodes).Error; err != nil {
				return err
			}

			purchaseItems = append(purchaseItems, purchaseItemCodes...)
		}

		if len(purchaseItems) > 0 {
			if result := tx.Model(&models.PurchaseItem{}).
				Where("purchase_item IN ?", purchaseItems).
				Updates(map[string]interface{}{
					"status_payment": "COMPLETED",
					"update_dtm":     time.Now().UTC(),
				}); result.Error != nil {
				err = result.Error
			}
		}

		var purchaseAutoCompleteCodes []string
		subQueryCompletePurchaseItem := tx.Model(&models.PurchaseItem{}).
			Select("purchase_id").
			Where("status_payment = ?", "COMPLETED")

		if err := tx.Model(&models.Purchase{}).
			Where("id IN (?)", subQueryCompletePurchaseItem).
			Pluck("purchase_code", &purchaseAutoCompleteCodes).Error; err != nil {
			return err
		}

		purchaseCodes = append(purchaseCodes, purchaseAutoCompleteCodes...)

		for _, code := range purchaseCodes {
			var totalItems int64
			var completedItems int64

			queryPurchaseID := tx.Model(&models.Purchase{}).Select("id").Where("purchase_code = ?", code)

			if err := tx.Model(&models.PurchaseItem{}).Where("purchase_id = (?)", queryPurchaseID).Count(&totalItems).Error; err != nil {
				return err
			}

			if err := tx.Model(&models.PurchaseItem{}).Where("purchase_id = (?) AND status_payment = ?", queryPurchaseID, "COMPLETED").Count(&completedItems).Error; err != nil {
				return err
			}

			status := "PENDING"
			if totalItems == 0 {
				status = "COMPLETED"
			}

			if totalItems > 0 && totalItems == completedItems {
				status = "COMPLETED"
			}

			if err := tx.Model(&models.Purchase{}).
				Where("purchase_code = ?", code).
				Updates(map[string]interface{}{
					"status_payment": status,
					"update_dtm":     time.Now().UTC(),
				}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func CompletePO(purchaseCodes []string) (err error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Purchase{}).
			Where("purchase_code IN ?", purchaseCodes).
			Updates(map[string]interface{}{
				"status":     "COMPLETED",
				"update_dtm": time.Now().UTC(),
			}).Error; err != nil {
			return err
		}
		return nil
	})
}

func CompletePOItem(usedType string, purchaseItemCodes []string) error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.PurchaseItem{}).
			Where("purchase_item IN ?", purchaseItemCodes).
			Updates(map[string]interface{}{
				"status":     "COMPLETED",
				"update_dtm": time.Now().UTC(),
			}).Error; err != nil {
			return err
		}

		purchaseIDs := []uuid.UUID{}
		if err := tx.Model(&models.PurchaseItem{}).
			Select("purchase_id").
			Where("purchase_item IN ?", purchaseItemCodes).
			Group("purchase_id").
			Scan(&purchaseIDs).Error; err != nil {
			return err
		}

		for _, purchaseID := range purchaseIDs {
			var completedCount int64
			var itemsCount int64

			if err := tx.Model(&models.PurchaseItem{}).
				Where("purchase_id = ? AND status = ?", purchaseID, "COMPLETED").
				Count(&completedCount).Error; err != nil {
				return err
			}

			if err := tx.Model(&models.PurchaseItem{}).
				Where("purchase_id = ?", purchaseID).
				Count(&itemsCount).Error; err != nil {
				return err
			}

			if completedCount == itemsCount {
				if err := tx.Model(&models.Purchase{}).
					Where("id = ?", purchaseID).
					Updates(map[string]interface{}{
						"status":      "COMPLETED",
						"used_type":   usedType,
						"used_status": "COMPLETED",
						"update_dtm":  time.Now().UTC(),
					}).Error; err != nil {
					return err
				}
			} else if completedCount != itemsCount {
				if err := tx.Model(&models.Purchase{}).
					Where("id = ?", purchaseID).
					Updates(map[string]interface{}{
						"used_type":   usedType,
						"used_status": "PARTIAL",
						"update_dtm":  time.Now().UTC(),
					}).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}
