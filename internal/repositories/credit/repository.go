package repositoryCredit

import (
	"errors"
	"fmt"
	"math"
	"prime-erp-core/internal/db"
	models "prime-erp-core/internal/models"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetCreditPreload(id []uuid.UUID, customerCode []string, isActive []string, page int, pageSize int) ([]models.Credit, int, int, error) {
	credit := []models.Credit{}

	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return nil, 0, 0, err
	}
	searchID := ""
	if len(id) > 0 {
		quotedStrings := make([]string, len(id))
		for i, s := range id {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchID = fmt.Sprintf(` and credit.id IN (%s)`, whereInClause)
	}

	searchCustomerCode := ""
	if len(customerCode) > 0 {
		quotedStrings := make([]string, len(customerCode))
		for i, s := range customerCode {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchCustomerCode = fmt.Sprintf(` and credit.customer_code IN (%s)`, whereInClause)
	}
	searchIsActive := ""
	if len(isActive) > 0 {
		quotedStrings := make([]string, len(isActive))
		for i, s := range isActive {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchIsActive = fmt.Sprintf(` and credit.is_active IN (%s)`, whereInClause)
	}
	var creditID []uuid.UUID
	gormx.Table("credit").Select("credit.id").
		Joins("left join credit_extra  on credit.id = credit_extra.credit_id").
		Where("1=1 " + searchID + "" + searchCustomerCode + "" + searchIsActive + "").
		Group("credit.id").Scan(&creditID)

	if len(creditID) > 0 {

		var count = len(creditID)

		query := gormx.Preload("CreditExtra")

		query = query.Where("id in (?)", creditID)

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

		err = query.Order("update_date desc").Find(&credit).Error
		sqlDB, err1 := gormx.DB()
		if err1 != nil {
			return nil, 0, 0, err1
		}

		// Close the connection
		if err2 := sqlDB.Close(); err2 != nil {
			return nil, 0, 0, err2
		}
		return credit, totalPages, int(totalRecords), err
	} else {
		return nil, 0, 0, err
	}

}
func CreateCredit(credit []models.Credit, creditExtra []models.CreditExtra) (err error) {
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
	if len(credit) > 0 {
		result := tx.Create(&credit)
		if result.Error != nil {
			tx.Rollback()
			return result.Error
		}
	}
	if len(creditExtra) > 0 {
		result := tx.Create(&creditExtra)
		if result.Error != nil {
			tx.Rollback()
			return result.Error
		}
	}

	err = tx.Commit().Error
	return err
}
func UpdateCredit(credit []models.Credit, creditExtra []models.CreditExtra) (int, error) {
	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return 0, err
	}
	rowsAffected := 0
	for _, creditValue := range credit {
		result := gormx.Table("credit").Where("id = ?", creditValue.ID).Updates(&creditValue)

		if result.Error != nil {
			gormx.Rollback()
			return 0, result.Error
		}
		rowsAffected = int(result.RowsAffected)
	}

	for _, creditExtraValue := range creditExtra {
		result := gormx.Table("credit_extra").Where("id = ?", creditExtraValue.ID).Updates(&creditExtraValue)

		if result.Error != nil {
			gormx.Rollback()
			return 0, result.Error
		}
		rowsAffected = int(result.RowsAffected)
	}

	return rowsAffected, nil
}
func DeleteCredit(creditID []uuid.UUID, creditExtra []uuid.UUID) error {
	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return err
	}
	for _, creditValue := range creditID {
		result := gormx.Table("credit").Where("id = ?", creditValue).Delete(&models.Credit{})

		if result.Error != nil {
			gormx.Rollback()
			return result.Error
		}

	}
	for _, creditExtraValue := range creditExtra {

		resultCreditExtra := gormx.Where("credit_id IN (?)", creditExtraValue).Delete(&models.CreditExtra{})

		if resultCreditExtra.Error != nil {
			gormx.Rollback()
			return resultCreditExtra.Error
		}

	}

	return nil
}
func DeleteCreditExtra(creditExtraID []uuid.UUID) error {
	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return err
	}
	for _, creditExtraValue := range creditExtraID {
		result := gormx.Table("credit_extra").Where("id = ?", creditExtraValue).Delete(&models.CreditExtra{})

		if result.Error != nil {
			gormx.Rollback()
			return result.Error
		}

	}

	return nil
}
func GetCreditRequest(id []uuid.UUID, customerCode []string, isAction []bool, requestType []string, status []string, page int, pageSize int) ([]models.CreditRequest, int, int, error) {
	creditRequest := []models.CreditRequest{}

	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return nil, 0, 0, err
	}

	query := gormx.Model(&models.CreditRequest{})
	if len(id) > 0 {
		query = query.Where("id in (?)", id)
	}
	if len(customerCode) > 0 {
		query = query.Where("customer_code in (?)", customerCode)
	}
	if len(requestType) > 0 {
		query = query.Where("request_type in (?)", requestType)
	}
	if len(isAction) > 0 {
		query = query.Where("is_action in (?)", isAction)
	}
	if len(status) > 0 {
		query = query.Where("status  in (?)", status)
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

	err = query.Order("update_date desc").Find(&creditRequest).Error
	sqlDB, err1 := gormx.DB()
	if err1 != nil {
		return nil, 0, 0, err1
	}

	// Close the connection
	if err2 := sqlDB.Close(); err2 != nil {
		return nil, 0, 0, err2
	}

	return creditRequest, totalPages, int(totalRecords), err

}
func GetCreditRequestPreload(id []uuid.UUID, customerCode []string, isAction []bool, page int, pageSize int, customerCodeLike string, customerNameLike string, creditLimitLike float64, increaseCreditLimitLike float64, startDate *time.Time, endDate *time.Time, customerStatus *bool, pendingApprove string) ([]models.CreditRequest, int, int, error) {
	creditRequest := []models.CreditRequest{}

	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return nil, 0, 0, err
	}

	query := gormx.Model(&models.CreditRequest{})
	if len(id) > 0 {
		query = query.Where("id in (?)", id)
	}
	if len(customerCode) > 0 {
		query = query.Where("customer_code in (?)", customerCode)
	}
	if len(isAction) > 0 {
		query = query.Where("is_action in (?)", isAction)
	}
	if customerCodeLike != "" {
		likePattern := fmt.Sprintf("%%%s%%", customerCodeLike)
		query = query.Where("customer_code ILIKE ?", likePattern)
	}
	if creditLimitLike > 0 {
		query = query.Where("amount::text >= ?", creditLimitLike)
	}
	if increaseCreditLimitLike > 0 {
		query = query.Where("temporary_increase_credit_limit::text >= ?", increaseCreditLimitLike)
	}
	if startDate != nil {
		query = query.Where("effective_dtm >= '%s' ", startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		query = query.Where("effective_dtm <= '%s' ", endDate.Format("2006-01-02"))
	}

	/* 	err = query.Order("is_approve desc").Select("customer_code, SUM(CASE WHEN request_type = 'BASE' THEN amount ELSE 0 END) AS amount, " +
	"SUM(CASE WHEN request_type = 'EXTRA' THEN amount ELSE 0 END) AS temporary_increase_credit_limit, " +
	"MAX(CASE WHEN status = 'PENDING' THEN 1 ELSE 0 END) AS is_approve, " +
	"MIN(CASE WHEN request_type = 'BASE' THEN effective_dtm ELSE NULL END) AS effective_dtm, " +
	"MAX(CASE WHEN status = 'APPROVED' THEN approve_date END) AS approve_date, " +
	"MAX(CASE WHEN request_type = 'BASE' THEN expire_dtm ELSE NULL END) AS expire_dtm ").
	Where("1=1").Group("customer_code").Find(&creditRequest).Error */

	baseQuery := query.
		Select(`
        customer_code,
        SUM(CASE WHEN request_type = 'BASE' THEN amount ELSE 0 END) AS amount,
        SUM(CASE WHEN request_type = 'EXTRA' THEN amount ELSE 0 END) AS temporary_increase_credit_limit,
        MAX(CASE WHEN status = 'PENDING' THEN 1 ELSE 0 END) AS is_approve,
        MIN(CASE WHEN request_type = 'BASE' THEN effective_dtm END) AS effective_dtm,
        MAX(CASE WHEN status = 'APPROVED' THEN approve_date END) AS approve_date,
        MAX(CASE WHEN request_type = 'BASE' THEN expire_dtm END) AS expire_dtm
    `).
		Where("1=1").
		Group("customer_code")

	// ===== นับจำนวน group จริง =====
	var totalRecords int64
	err = query.Session(&gorm.Session{}).
		Table("(?) as sub", baseQuery).
		Count(&totalRecords).Error
	if err != nil {
		return nil, 0, 0, err
	}

	offset := (page - 1) * pageSize
	totalPages := int(math.Ceil(float64(totalRecords) / float64(pageSize)))

	// ===== ดึงข้อมูล =====

	err = baseQuery.
		Order("is_approve desc").
		Limit(pageSize).
		Offset(offset).
		Find(&creditRequest).Error

	return creditRequest, totalPages, int(totalRecords), err

}

func CreateCreditRequest(creditRequest []models.CreditRequest) (err error) {
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
	if len(creditRequest) > 0 {
		result := tx.Create(&creditRequest)
		if result.Error != nil {
			tx.Rollback()
			return result.Error
		}
	}

	err = tx.Commit().Error
	return err
}
func UpdateCreditRequest(creditRequest []models.CreditRequest) (int, error) {
	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return 0, err
	}
	rowsAffected := 0
	for _, creditRequestValue := range creditRequest {
		result := gormx.Table("credit_request").Where("id = ?", creditRequestValue.ID).Updates(&creditRequestValue)

		if result.Error != nil {
			gormx.Rollback()
			return 0, result.Error
		}
		rowsAffected = int(result.RowsAffected)
	}
	if rowsAffected == 0 {
		for _, creditRequestValue := range creditRequest {
			result := gormx.Table("credit_request").Where("request_code = ?", creditRequestValue.RequestCode).Updates(&creditRequestValue)

			if result.Error != nil {
				gormx.Rollback()
				return 0, result.Error
			}
			rowsAffected = int(result.RowsAffected)
		}
	}

	return rowsAffected, nil
}
func DeleteCreditRequest(creditRequestID []uuid.UUID) error {
	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return err
	}
	for _, creditValue := range creditRequestID {
		result := gormx.Table("credit_request").Where("id = ?", creditValue).Delete(&models.Credit{})

		if result.Error != nil {
			gormx.Rollback()
			return result.Error
		}

	}

	return nil
}
func CreateCreditTransaction(creditTransaction []models.CreditTransaction) (err error) {
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
	if len(creditTransaction) > 0 {
		result := tx.Create(&creditTransaction)
		if result.Error != nil {
			tx.Rollback()
			return result.Error
		}
	}

	err = tx.Commit().Error
	return err
}
func GetCreditTransaction(id []uuid.UUID, transactionCode []string, status []string, page int, pageSize int) ([]models.CreditTransaction, int, int, error) {
	creditTransaction := []models.CreditTransaction{}

	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return nil, 0, 0, err
	}

	query := gormx.Model(&models.CreditTransaction{})
	if len(id) > 0 {
		query = query.Where("id in (?)", id)
	}
	if len(transactionCode) > 0 {
		query = query.Where("transaction_code in (?)", transactionCode)
	}
	if len(status) > 0 {
		query = query.Where("status in (?)", status)
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

	err = query.Order("update_date desc").Find(&creditTransaction).Error
	sqlDB, err1 := gormx.DB()
	if err1 != nil {
		return nil, 0, 0, err1
	}

	// Close the connection
	if err2 := sqlDB.Close(); err2 != nil {
		return nil, 0, 0, err2
	}

	return creditTransaction, totalPages, int(totalRecords), err

}
