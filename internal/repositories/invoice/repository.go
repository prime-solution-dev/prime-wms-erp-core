package saleRepository

import (
	"fmt"
	"math"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"strings"

	"github.com/google/uuid"
)

// Create
func GetInvoicePreload(id []uuid.UUID, invoiceCode []string, customerCode []string, status []string, docRef []string, page int, pageSize int) ([]models.Invoice, int, int, error) {
	invoice := []models.Invoice{}

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
		searchID = fmt.Sprintf(` and invoice.id IN (%s)`, whereInClause)
	}

	searchInvoiceCode := ""
	if len(invoiceCode) > 0 {
		quotedStrings := make([]string, len(invoiceCode))
		for i, s := range invoiceCode {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchInvoiceCode = fmt.Sprintf(` and invoice.invoice_code IN (%s)`, whereInClause)
	}
	searchCustomerCode := ""
	if len(customerCode) > 0 {
		quotedStrings := make([]string, len(customerCode))
		for i, s := range customerCode {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchCustomerCode = fmt.Sprintf(` and invoice.customer_code IN (%s)`, whereInClause)
	}
	searchIsStatus := ""
	if len(status) > 0 {
		quotedStrings := make([]string, len(status))
		for i, s := range status {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchIsStatus = fmt.Sprintf(` and invoice.status IN (%s)`, whereInClause)
	}
	searchDocRef := ""
	if len(docRef) > 0 {
		quotedStrings := make([]string, len(docRef))
		for i, s := range docRef {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchDocRef = fmt.Sprintf(` and invoice.doc_ref IN (%s)`, whereInClause)
	}

	var invoiceID []uuid.UUID
	gormx.Table("invoice").Select("invoice.id").
		Joins("inner join invoice_item on invoice.id = invoice_item.invoice_id").
		Joins("inner join invoice_deposit on invoice.id = invoice_deposit.invoice_id").
		Where("1=1 " + searchID + "" + searchInvoiceCode + "" + searchCustomerCode + "" + searchIsStatus + "" + searchDocRef + "").
		Group("invoice.id").Scan(&invoiceID)

	if len(invoiceID) > 0 {

		var count = len(invoiceID)

		query := gormx.Preload("InvoiceItem").Preload("InvoiceDeposit")

		query = query.Where("id in (?)", invoiceID)

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

		err = query.Order("update_date desc").Find(&invoice).Error
		sqlDB, err1 := gormx.DB()
		if err1 != nil {
			return nil, 0, 0, err1
		}

		// Close the connection
		if err2 := sqlDB.Close(); err2 != nil {
			return nil, 0, 0, err2
		}
		return invoice, totalPages, int(totalRecords), err
	} else {
		return nil, 0, 0, err
	}
}
