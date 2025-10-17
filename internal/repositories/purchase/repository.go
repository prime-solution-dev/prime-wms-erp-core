package purchaseRepository

import (
	"math"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"time"

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
	productCodes []string,
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
		Preload("PurchaseItems").
		Limit(pageSize).
		Offset(offset).
		Find(&purchases).Error; err != nil {
		return nil, 0, 0, 0, 0, err
	}

	totalPages := int(math.Ceil(float64(totalRecords) / float64(pageSize)))

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
