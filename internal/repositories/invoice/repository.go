package saleRepository

import (
	"errors"
	"fmt"
	"math"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Create
func GetInvoicePreload(id []uuid.UUID, invoiceCode []string, invoiceType []string, customerCode []string, status []string, docRef []string, invoiceRef []string, invoiceItemDocRef []string, page int, pageSize int, invoiceCodeLike string, invoiceRefLike string, packingLike string, salesOrderLike string, customerCodeLike string, customerNameLike string, documentDate *time.Time, createDate *time.Time, lastSubmitDate *time.Time) ([]models.Invoice, int, int, error) {
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
	searchInvoiceType := ""
	if len(invoiceType) > 0 {
		quotedStrings := make([]string, len(invoiceType))
		for i, s := range invoiceType {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchInvoiceType = fmt.Sprintf(` and invoice.invoice_type IN (%s)`, whereInClause)
	}

	searchInvoiceRef := ""
	if len(invoiceRef) > 0 {
		quotedStrings := make([]string, len(invoiceRef))
		for i, s := range invoiceRef {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchInvoiceRef = fmt.Sprintf(` and invoice.invoice_ref IN (%s)`, whereInClause)
	}

	searchCustomerCode := ""
	if len(customerCode) > 0 {
		quotedStrings := make([]string, len(customerCode))
		for i, s := range customerCode {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchCustomerCode = fmt.Sprintf(` and invoice.party_code IN (%s)`, whereInClause)
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
		searchDocRef = fmt.Sprintf(` and invoice.document_ref IN (%s)`, whereInClause)
	}
	if len(invoiceItemDocRef) > 0 {
		quotedStrings := make([]string, len(invoiceItemDocRef))
		for i, s := range invoiceItemDocRef {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchDocRef += fmt.Sprintf(` and invoice_item.document_ref IN (%s)`, whereInClause)
	}
	if invoiceCodeLike != "" {
		searchInvoiceCode += fmt.Sprintf(" and invoice.invoice_code ILIKE '%%%s%%' ", invoiceCodeLike)
	}
	if invoiceRefLike != "" {
		searchInvoiceRef += fmt.Sprintf(" and invoice.document_ref ILIKE '%%%s%%' ", invoiceRefLike)
	}
	if packingLike != "" {
		searchDocRef += fmt.Sprintf(" and invoice_item.source_code ILIKE '%%%s%%' ", packingLike)
	}
	if salesOrderLike != "" {
		searchDocRef += fmt.Sprintf(" and invoice_item.document_ref ILIKE '%%%s%%' ", salesOrderLike)
	}
	if customerCodeLike != "" {
		searchCustomerCode += fmt.Sprintf(" and invoice.party_code ILIKE '%%%s%%' ", customerCodeLike)
	}
	if customerNameLike != "" {
		searchCustomerCode += fmt.Sprintf(" and invoice.party_name ILIKE '%%%s%%' ", customerNameLike)
	}
	if documentDate != nil {
		searchDocRef += fmt.Sprintf(" and DATE(invoice.document_date) = DATE('%s') ", documentDate.Format("2006-01-02"))
	}
	if createDate != nil {
		searchDocRef += fmt.Sprintf(" and DATE(invoice.create_dtm) = DATE('%s') ", createDate.Format("2006-01-02"))
	}
	if lastSubmitDate != nil {
		searchDocRef += fmt.Sprintf(" and DATE(invoice.submit_date) = DATE('%s') ", lastSubmitDate.Format("2006-01-02"))
	}

	var invoiceID []uuid.UUID
	gormx.Table("invoice").Select("invoice.id").
		Joins("inner join invoice_item on invoice.id = invoice_item.invoice_id").
		Where("1=1 x" + searchID + "" + searchInvoiceCode + "" + searchInvoiceType + "" + searchInvoiceRef + "" + searchCustomerCode + "" + searchIsStatus + "" + searchDocRef + "").
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

		err = query.Order("update_dtm desc").Find(&invoice).Error
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

func GetInvoiceRelatedByPO(companyCode string, siteCode string, purchaseCodes []string, purchaseItemCodes []string, invoiceType []string, status []string) ([]models.Invoice, error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	var invoices []models.Invoice

	query := gormx.Model(&models.Invoice{}).
		Where("company_code = ? AND site_code = ?", companyCode, siteCode)

	preloadConditionsString := "document_ref IN ? AND document_ref_item IN ? AND status IN ?"

	if len(invoiceType) > 0 {
		query = query.Where("invoice_type IN ?", invoiceType)
	}

	if len(preloadConditionsString) != 0 {
		query = query.Preload("InvoiceItem", preloadConditionsString, purchaseCodes, purchaseItemCodes, status)
	}

	err = query.Find(&invoices).Error
	if err != nil {
		return nil, err
	}

	return invoices, nil
}

func CreateInvoice(invoice []models.Invoice, invoiceItem []models.InvoiceItem, deposit []models.InvoiceDeposit) (err error) {
	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return err
	}
	tx := gormx.Begin()
	defer func() {
		if rc := recover(); rc != nil {
			tx.Rollback()
			err = errors.New("panic error cant't save approval.")
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
	if len(invoiceItem) > 0 {
		result := tx.Create(&invoiceItem)
		if result.Error != nil {
			tx.Rollback()
			return result.Error
		}
	}
	if len(deposit) > 0 {
		result := tx.Create(&deposit)
		if result.Error != nil {
			tx.Rollback()
			return result.Error
		}
	}
	err = tx.Commit().Error
	return err
}
func UpdateInvoice(invoice []models.Invoice, invoiceItem []models.InvoiceItem) (int, error) {
	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return 0, err
	}
	rowsAffected := 0
	for _, invoiceValue := range invoice {
		result := gormx.Table("invoice").Where("id = ?", invoiceValue.ID).Updates(&invoiceValue)

		if result.Error != nil {
			gormx.Rollback()
			return 0, result.Error
		}
		rowsAffected = int(result.RowsAffected)
	}
	for _, invoiceItemValue := range invoiceItem {
		result := gormx.Table("invoice_item").Where("id = ?", invoiceItemValue.ID).Updates(&invoiceItemValue)

		if result.Error != nil {
			gormx.Rollback()
			return 0, result.Error
		}
	}

	return rowsAffected, nil
}
func DeleteInvoice(id []uuid.UUID) (err error) {
	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return err
	}

	resultSuggest := gormx.Table("invoice").Where("id IN (?)", id).Delete(models.Invoice{})
	if resultSuggest.Error != nil {
		gormx.Rollback()
		return resultSuggest.Error
	}
	resultBatch := gormx.Table("invoice_item").Where("invoice_id IN (?)", id).Delete(models.InvoiceItem{})
	if resultBatch.Error != nil {
		gormx.Rollback()
		return resultBatch.Error
	}

	return
}
func DeleteInvoiceItem(id []uuid.UUID) (err error) {
	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return err
	}

	resultBatch := gormx.Table("invoice_item").Where("invoice_id IN (?)", id).Delete(models.InvoiceItem{})
	if resultBatch.Error != nil {
		gormx.Rollback()
		return resultBatch.Error
	}

	return
}
