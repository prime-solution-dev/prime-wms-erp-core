package purchaseRepository

import (
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
