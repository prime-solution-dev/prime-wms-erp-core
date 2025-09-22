package purchaseRepository

import (
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

// Create
func CreatePOBigLot(prePurchase models.PrePurchase) (uuid.UUID, error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return uuid.Nil, err
	}
	defer db.CloseGORM(gormx)

	result := gormx.Create(&prePurchase)
	if result.Error != nil {
		return uuid.Nil, result.Error
	}

	return prePurchase.ID, nil
}

func CreatePOBigLotItems(prePurchaseItems []models.PrePurchaseItem) error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	result := gormx.Create(&prePurchaseItems)
	if result.Error != nil {
		return result.Error
	}

	return nil
}
