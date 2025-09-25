package purchaseRepository

import (
	"fmt"
	"math"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
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
func GetPOBigLotList(prePurchaseCodes []string, companyCode string, siteCode string, page int, pageSize int) ([]models.PrePurchase, int, int, int, int, error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	defer db.CloseGORM(gormx)

	prePurchaseList := []models.PrePurchase{}
	totalRecords := int64(0)

	preloadQuery := gormx.Preload("PrePurchaseItems")
	filterSite := preloadQuery.Where("company_code = ? AND site_code = ? AND pre_purchase_code IN ?", companyCode, siteCode, prePurchaseCodes)

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
		result := tx.Model(&models.PrePurchase{}).
			Where("id = ?", prePurchase.ID).
			Updates(prePurchase)

		fmt.Println("pre purchase result: ", result)

		if result.Error != nil {
			err = result.Error
			return
		}

		// delete old items
		delResult := tx.Where("pre_purchase_id = ?", prePurchase.ID).
			Delete(&models.PrePurchaseItem{})

		if result.Error != nil {
			err = result.Error
			return
		}

		fmt.Println("pre purchase result: ", delResult)

		// set PrePurchaseID ให้แน่ใจว่า insert ถูก
		for i := range prePurchase.PrePurchaseItems {
			prePurchase.PrePurchaseItems[i].PrePurchaseID = prePurchase.ID
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
