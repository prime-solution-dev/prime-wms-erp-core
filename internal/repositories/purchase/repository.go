package purchaseRepository

import (
	"math"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
)

// Create
func CreatePurchase(purchases []models.Purchase) error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	tx := gormx.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit().Error
		}
	}()

	result := tx.Create(&purchases)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// Get
func GetPurchaseList(purchaseCodes []string, companyCode, siteCode string, page, pageSize int) ([]models.Purchase, int, int, int, int, error) {
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

	// Count total records (no preload needed)
	if err := query.Count(&totalRecords).Error; err != nil {
		return nil, 0, 0, 0, 0, err
	}

	// Apply pagination
	offset := (page - 1) * pageSize
	if err := query.Preload("PurchaseItems").
		Limit(pageSize).
		Offset(offset).
		Find(&purchases).
		Error; err != nil {
		return nil, 0, 0, 0, 0, err
	}

	totalPages := int(math.Ceil(float64(totalRecords) / float64(pageSize)))

	return purchases, int(totalRecords), page, pageSize, totalPages, nil
}
