package paymentRepository

import (
	"errors"
	"fmt"
	"math"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"strings"

	"github.com/google/uuid"
)

// Create
func GetPaymentPreload(id []uuid.UUID, customerCode []string, status []string, invoiceCode []string, page int, pageSize int) ([]models.Payment, int, int, error) {
	credit := []models.Payment{}

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
		searchID = fmt.Sprintf(` and payment.id IN (%s)`, whereInClause)
	}

	searchCustomerCode := ""
	if len(customerCode) > 0 {
		quotedStrings := make([]string, len(customerCode))
		for i, s := range customerCode {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchCustomerCode = fmt.Sprintf(` and payment.customer_code IN (%s)`, whereInClause)
	}
	searchIsStatus := ""
	if len(status) > 0 {
		quotedStrings := make([]string, len(status))
		for i, s := range status {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchIsStatus = fmt.Sprintf(` and payment.status IN (%s)`, whereInClause)
	}
	searchInvoiceCode := ""
	if len(invoiceCode) > 0 {
		quotedStrings := make([]string, len(invoiceCode))
		for i, s := range invoiceCode {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchInvoiceCode = fmt.Sprintf(` and payment_invoice.invoice_code IN (%s)`, whereInClause)
	}

	var paymentID []uuid.UUID
	gormx.Table("payment").Select("payment.id").
		Joins("inner join payment_invoice on payment.id = payment_invoice.payment_id").
		Where("1=1 " + searchID + "" + searchCustomerCode + "" + searchIsStatus + "" + searchInvoiceCode + "").
		Group("payment.id").Scan(&paymentID)

	if len(paymentID) > 0 {

		var count = len(paymentID)

		query := gormx.Preload("PaymentInvoice")

		query = query.Where("id in (?)", paymentID)

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
func CreatePayment(payment []models.Payment, paymentInvoice []models.PaymentInvoice) (err error) {
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
	if len(payment) > 0 {
		result := tx.Create(&payment)
		if result.Error != nil {
			tx.Rollback()
			return result.Error
		}
	}
	if len(paymentInvoice) > 0 {
		result := tx.Create(&paymentInvoice)
		if result.Error != nil {
			tx.Rollback()
			return result.Error
		}
	}
	err = tx.Commit().Error
	return err
}
func DeletePayment(paymentID []uuid.UUID, invoiceCode []string) (err error) {
	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return err
	}
	resultPayment := gormx.Where("id IN (?)", paymentID).Delete(&models.Payment{})
	if resultPayment.Error != nil {
		gormx.Rollback()
		return resultPayment.Error
	}

	resultconfirm := gormx.Where("invoice_code IN (?)", invoiceCode).Delete(&models.PaymentInvoice{})
	if resultconfirm.Error != nil {
		gormx.Rollback()
		return resultconfirm.Error
	}

	return
}
