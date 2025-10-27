package priceListRepository

import (
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UpdatePriceListGroup
func UpdatePriceListBase(priceListGroup []models.PriceListGroup) error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		for _, group := range priceListGroup {
			oldPriceListGroup := models.PriceListGroup{}

			if err := tx.Model(&models.PriceListGroup{}).
				Where("id = ?", group.ID).
				Find(&oldPriceListGroup).
				Error; err != nil {
				return err
			}

			now := time.Now().UTC()

			historyFormat := models.PriceListGroupHistory{
				ID:                uuid.New(),
				CompanyCode:       oldPriceListGroup.CompanyCode,
				SiteCode:          oldPriceListGroup.SiteCode,
				GroupCode:         oldPriceListGroup.GroupCode,
				PriceUnit:         oldPriceListGroup.PriceUnit,
				PriceWeight:       oldPriceListGroup.PriceWeight,
				BeforePriceUnit:   oldPriceListGroup.BeforePriceUnit,
				BeforePriceWeight: oldPriceListGroup.BeforePriceWeight,
				Currency:          oldPriceListGroup.Currency,
				EffectiveDate:     oldPriceListGroup.EffectiveDate,
				ExpiryDate:        &now,
				Remark:            oldPriceListGroup.Remark,
				CreateBy:          oldPriceListGroup.CreateBy,
				CreateDtm:         oldPriceListGroup.CreateDtm,
				UpdateBy:          oldPriceListGroup.UpdateBy,
				UpdateDtm:         oldPriceListGroup.UpdateDtm,
			}

			// Insert old record into history table
			if err := tx.Model(&models.PriceListGroupHistory{}).Create(&historyFormat).Error; err != nil {
				return err
			}

			if err := tx.Model(&models.PriceListGroup{}).
				Where("id = ?", group.ID).
				Updates(map[string]interface{}{
					"price_unit":          group.PriceUnit,
					"price_weight":        group.PriceWeight,
					"before_price_unit":   oldPriceListGroup.PriceUnit,
					"before_price_weight": oldPriceListGroup.PriceWeight,
					"currency":            group.Currency,
					"effective_date":      group.EffectiveDate,
					"remark":              group.Remark,
					"update_by":           group.UpdateBy,
					"update_dtm":          group.UpdateDtm,
				}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func UpdatePriceListTerm(priceListGroupTerms []models.PriceListGroupTerm) error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		for _, term := range priceListGroupTerms {
			if err := tx.Model(&models.PriceListGroupTerm{}).
				Where("id = ?", term.ID).
				Updates(map[string]interface{}{
					"pdc":         term.Pdc,
					"pdc_percent": term.PdcPercent,
					"due":         term.Due,
					"due_percent": term.DuePercent,
					"update_by":   term.UpdateBy,
					"update_dtm":  term.UpdateDtm,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// DeletePriceListGroup
func DeletePriceListBase(ids []string) error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)
	return gormx.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			oldPriceListGroup := models.PriceListGroup{}

			if err := tx.Model(&models.PriceListGroup{}).
				Where("id = ?", id).
				Find(&oldPriceListGroup).
				Error; err != nil {
				return err
			}

			now := time.Now().UTC()

			historyFormat := models.PriceListGroupHistory{
				ID:                uuid.New(),
				CompanyCode:       oldPriceListGroup.CompanyCode,
				SiteCode:          oldPriceListGroup.SiteCode,
				GroupCode:         oldPriceListGroup.GroupCode,
				PriceUnit:         oldPriceListGroup.PriceUnit,
				PriceWeight:       oldPriceListGroup.PriceWeight,
				BeforePriceUnit:   oldPriceListGroup.BeforePriceUnit,
				BeforePriceWeight: oldPriceListGroup.BeforePriceWeight,
				Currency:          oldPriceListGroup.Currency,
				EffectiveDate:     oldPriceListGroup.EffectiveDate,
				ExpiryDate:        &now,
				Remark:            oldPriceListGroup.Remark,
				CreateBy:          oldPriceListGroup.CreateBy,
				CreateDtm:         oldPriceListGroup.CreateDtm,
				UpdateBy:          oldPriceListGroup.UpdateBy,
				UpdateDtm:         oldPriceListGroup.UpdateDtm,
			}

			// Insert old record into history table
			if err := tx.Model(&models.PriceListGroupHistory{}).Create(&historyFormat).Error; err != nil {
				return err
			}

			// Delete from main table
			if err := tx.Model(&models.PriceListGroup{}).
				Where("id = ?", id).
				Delete(&models.PriceListGroup{}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
