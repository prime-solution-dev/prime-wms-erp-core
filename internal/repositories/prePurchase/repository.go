package prePurchaseRepository

import (
	"math"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"time"

	"gorm.io/gorm"
)

// Create
func CreatePOBigLot(prePurchases []models.PrePurchase) error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&prePurchases).Error; err != nil {
			return err
		}
		return nil
	})
}

// Get
func GetPOBigLotList(
	prePurchaseCodes []string,
	supplierCodes []string,
	productGroupCodes []string,
	statusApprove []string,
	companyCode,
	siteCode string,
	page int,
	pageSize int,
) ([]models.PrePurchase, int, int, int, int, error) {
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

	if len(supplierCodes) > 0 {
		query = query.Where("supplier_code IN ?", supplierCodes)
	}

	if len(statusApprove) > 0 {
		query = query.Where("status_approve IN ?", statusApprove)
	}

	if len(productGroupCodes) > 0 {
		sub := gormx.Model(&models.PrePurchaseItem{}).
			Select("1").
			Where("pre_purchase.id = pre_purchase_item.pre_purchase_id").
			Where("hierarchy_code IN ?", productGroupCodes)

		query = query.Where("EXISTS (?)", sub)
	}

	// Count total records (no preload needed)
	if err := query.Count(&totalRecords).Error; err != nil {
		return nil, 0, 0, 0, 0, err
	}

	if pageSize == 0 {
		pageSize = int(totalRecords)
	}

	// Pagination
	offset := (page - 1) * pageSize
	if err := query.
		Preload("PrePurchaseItems").
		Limit(pageSize).
		Offset(offset).
		Find(&prePurchaseList).Error; err != nil {
		return nil, 0, 0, 0, 0, err
	}

	totalPages := int(math.Ceil(float64(totalRecords) / float64(pageSize)))

	if page == 0 {
		page = 1
	}

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
