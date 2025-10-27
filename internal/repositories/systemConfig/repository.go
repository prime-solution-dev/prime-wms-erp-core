package systemConfigRepository

import (
	"errors"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
)

func GetSystemConfig(topicCodes []string, configCodes []string) ([]models.SystemConfig, error) {
	if len(topicCodes) < 1 {
		return nil, errors.New("missing required parameter")
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	systemConfigs := []models.SystemConfig{}

	query := gormx.Model(&models.SystemConfig{}).
		Where("topic_code IN ?", topicCodes)

	if len(configCodes) > 0 {
		query = query.Where("config_code IN ?", configCodes)
	}

	if err := query.Find(&systemConfigs).Error; err != nil {
		return nil, err
	}

	return systemConfigs, nil
}

func UpdateSystemConfig(systemConfigs []models.SystemConfig) (err error) {
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

	for _, systemConfig := range systemConfigs {
		if result := tx.Model(&models.SystemConfig{}).
			Where(
				"topic_code = ? AND config_code = ?",
				systemConfig.TopicCode,
				systemConfig.ConfigCode,
			).
			Update("value", systemConfig.Value); result.Error != nil {
			err = result.Error
			return
		}
	}

	return
}
