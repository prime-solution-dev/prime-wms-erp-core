package purchaseRepository

import (
	"math"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"time"
)

// Create
func CreatePOBigLot(prePurchases []models.PrePurchase) error {
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

	result := tx.Create(&prePurchases)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// Get
func GetPOBigLotList(prePurchaseCodes []string, companyCode, siteCode string, page, pageSize int) ([]models.PrePurchase, int, int, int, int, error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	defer db.CloseGORM(gormx)

	var prePurchaseList []models.PrePurchase
	var totalRecords int64

	// Build base query
	query := gormx.Model(&models.PrePurchase{}).
		Where("company_code = ? AND site_code = ?", companyCode, siteCode)

	if len(prePurchaseCodes) > 0 {
		query = query.Where("pre_purchase_code IN ?", prePurchaseCodes)
	}

	// Count total records (no preload needed)
	if err := query.Count(&totalRecords).Error; err != nil {
		return nil, 0, 0, 0, 0, err
	}

	// Pagination
	offset := (page - 1) * pageSize
	if err := query.Preload("PrePurchaseItems").
		Limit(pageSize).
		Offset(offset).
		Find(&prePurchaseList).Error; err != nil {
		return nil, 0, 0, 0, 0, err
	}

	totalPages := int(math.Ceil(float64(totalRecords) / float64(pageSize)))

	return prePurchaseList, int(totalRecords), page, pageSize, totalPages, nil
}

// Update
func UpdatePOBigLot(prePurchases []models.PrePurchase) (err error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	tx := gormx.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit().Error
		}
	}()

	for _, prePurchase := range prePurchases {
		// update pre_purchase
		if result := tx.Model(&models.PrePurchase{}).
			Where("id = ?", prePurchase.ID).
			Updates(prePurchase); result.Error != nil {
			err = result.Error
			return
		}

		// delete old items
		if result := tx.Where("pre_purchase_id = ?", prePurchase.ID).
			Delete(&models.PrePurchaseItem{}); result.Error != nil {
			err = result.Error
			return
		}

		for i := range prePurchase.PrePurchaseItems {
			prePurchase.PrePurchaseItems[i].PrePurchaseID = prePurchase.ID
			prePurchase.PrePurchaseItems[i].Status = prePurchase.Status
		}

		// insert new items
		if len(prePurchase.PrePurchaseItems) > 0 {
			if result := tx.Create(&prePurchase.PrePurchaseItems); result.Error != nil {
				err = result.Error
				return
			}
		}
	}

	return
}

// Update
func UpdateStatusApprovePOBigLot(prePurchases []models.UpdateStatusApprovePOBigLotRequest) (err error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	tx := gormx.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit().Error
		}
	}()

	for _, prePurchase := range prePurchases {
		// update pre_purchase
		if result := tx.Model(&models.PrePurchase{}).
			Where("id = ?", prePurchase.ID).Updates(map[string]interface{}{
			"status_approve": prePurchase.StatusApprove,
			"is_approved":    prePurchase.IsApproved,
			"update_dtm":     time.Now().UTC(),
		}); result.Error != nil {
			err = result.Error
			return
		}
	}

	return
}
