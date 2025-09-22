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

	result := gormx.Create(&prePurchase)
	if result.Error != nil {
		return result.Error
	}

	return nil
}
