package depositRepository

import (
	"errors"
	"math"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func GetDepositPreload(id []uuid.UUID, customerCode []string, status []string, depositCode []string, page int, pageSize int, NotdepositCode []string) ([]models.Deposit, int, int, error) {
	deposit := []models.Deposit{}

	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return nil, 0, 0, err
	}

	query := gormx.Model(&models.Deposit{})
	if len(id) > 0 {
		query = query.Where("id in (?)", id)
	}
	if len(customerCode) > 0 {
		query = query.Where("customer_code in (?)", customerCode)
	}
	if len(status) > 0 {
		query = query.Where("status in (?)", status)
	}
	if len(depositCode) > 0 {
		query = query.Where("deposit_code in (?)", depositCode)
	}
	if len(NotdepositCode) > 0 {
		query = query.Where("deposit_code not in (?)", NotdepositCode)
	}

	var count int64
	query.Count(&count)

	totalRecords := count
	totalPages := 0
	offset := (page - 1) * pageSize
	if totalRecords > 0 {

		if pageSize > 0 && page > 0 {
			query = query.Limit(pageSize).Offset(offset)
			totalPages = int(math.Ceil(float64(totalRecords) / float64(pageSize)))
		} else {
			query = query.Limit(int(totalRecords)).Offset(offset)
			totalPages = (int(totalRecords) / 1)
		}

	}

	err = query.Order("update_date desc").Find(&deposit).Error
	sqlDB, err1 := gormx.DB()
	if err1 != nil {
		return nil, 0, 0, err1
	}

	// Close the connection
	if err2 := sqlDB.Close(); err2 != nil {
		return nil, 0, 0, err2
	}

	return deposit, totalPages, int(totalRecords), err

}
func CreateDeposit(invoice []models.Deposit) (err error) {
	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return err
	}
	tx := gormx.Begin()
	defer func() {
		if rc := recover(); rc != nil {
			tx.Rollback()
			err = errors.New("panic error cant't save approval")
		}
	}()
	if err = tx.Error; err != nil {
		return err
	}
	if len(invoice) > 0 {
		result := tx.Create(&invoice)
		if result.Error != nil {
			tx.Rollback()
			return result.Error
		}
	}
	err = tx.Commit().Error
	return err
}
func DeleteDeposit(id []uuid.UUID) (err error) {
	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return err
	}

	resultSuggest := gormx.Table("deposit").Where("id IN (?)", id).Delete(models.Deposit{})
	if resultSuggest.Error != nil {
		gormx.Rollback()
		return resultSuggest.Error
	}

	return
}
