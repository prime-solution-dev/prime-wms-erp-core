package unitRepository

import (
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
)

func GetAllUnit(topics []string) ([]models.Unit, error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	units := []models.Unit{}
	query := gormx.Preload("UnitMethodItems").Preload("UnitMethodItems.UnitUomItems")

	if len(topics) > 0 {
		query = query.Where("topic IN ?", topics)
	}

	if err := query.Find(&units).Error; err != nil {
		return nil, err
	}

	return units, nil
}
