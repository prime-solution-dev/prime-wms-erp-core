package purchaseRepository

import (
	"math"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
)

// Create
func CreatePOBigLot(prePurchase []models.PrePurchase) error {
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

	result := tx.Create(&prePurchase)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// Get
func GetPOBigLotList(companyCode string, siteCode string, page int, pageSize int) ([]models.PrePurchase, int, int, int, int, error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	defer db.CloseGORM(gormx)

	prePurchaseList := []models.PrePurchase{}
	totalRecords := int64(0)

	preloadQuery := gormx.Preload("PrePurchaseItems")
	filterSite := preloadQuery.Where("company_code = ? AND site_code = ?", companyCode, siteCode)

	if err := filterSite.Find(&prePurchaseList).Count(&totalRecords).Error; err != nil {
		return nil, 0, 0, 0, 0, err
	}

	offset := (page - 1) * pageSize
	result := filterSite.Limit(pageSize).Offset(offset).Find(&prePurchaseList)
	if result.Error != nil {
		return nil, 0, 0, 0, 0, result.Error
	}

	return prePurchaseList, int(totalRecords), page, pageSize, int(math.Ceil(float64(totalRecords) / float64(pageSize))), nil
}
