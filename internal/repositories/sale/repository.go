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
func GetSalePreload(id []uuid.UUID, customerCode []string, status []string, statusPayment []string, isApproved []bool, page int, pageSize int) ([]models.Sale, int, int, error) {
	credit := []models.Sale{}

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
		searchID = fmt.Sprintf(` and sale.id IN (%s)`, whereInClause)
	}

	searchCustomerCode := ""
	if len(customerCode) > 0 {
		quotedStrings := make([]string, len(customerCode))
		for i, s := range customerCode {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchCustomerCode = fmt.Sprintf(` and sale.customer_code IN (%s)`, whereInClause)
	}
	searchIsStatus := ""
	if len(status) > 0 {
		quotedStrings := make([]string, len(status))
		for i, s := range status {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchIsStatus = fmt.Sprintf(` and sale.status IN (%s)`, whereInClause)
	}
	searchStatusPayment := ""
	if len(statusPayment) > 0 {
		quotedStrings := make([]string, len(statusPayment))
		for i, s := range statusPayment {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchStatusPayment = fmt.Sprintf(` and sale.status_payment IN (%s)`, whereInClause)
	}
	searchIsApproved := ""
	if len(isApproved) > 0 {
		boolStrings := make([]string, len(isApproved))
		for i, b := range isApproved {
			boolStrings[i] = fmt.Sprintf("%t", b)
		}
		whereInClause := strings.Join(boolStrings, ", ")
		searchIsApproved = fmt.Sprintf(` AND sale.is_approved IN (%s)`, whereInClause)
	}

	var saleID []uuid.UUID
	gormx.Table("sale").Select("sale.id").
		Joins("inner join sale_item on sale.id = sale_item.sale_id").
		Where("1=1 " + searchID + "" + searchCustomerCode + "" + searchIsStatus + "" + searchStatusPayment + "" + searchIsApproved + "").
		Group("sale.id").Scan(&saleID)

	if len(saleID) > 0 {

		var count = len(saleID)

		query := gormx.Preload("SaleItem")

		query = query.Where("id in (?)", saleID)

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

type SaleWithInvoiceItems struct {
	Sale         models.Sale
	InvoiceItems []models.InvoiceItem
}

func GetSalesWithInvoiceItems(customerCode string) ([]SaleWithInvoiceItems, error) {

	sqlx, err := db.ConnectSqlx(`prime_erp`)
	if err != nil {
		return nil, err
	}
	searchCustomerCode := ""
	if customerCode != "" {
		searchCustomerCode = fmt.Sprintf(` and s.customer_code  = '%s'`, customerCode)
	}

	query := fmt.Sprintf(`
		    SELECT 
        s.sale_code, 
        s.customer_code, 
        s.total_amount,
        it.id as item_id, 
        it.document_ref, 
        it.total_amount as invoice_total_amount
    FROM sale s
    LEFT JOIN invoice_item it ON s.sale_code = it.document_ref
		where 1=1 
		%s
		 ORDER BY s.sale_code
	`, searchCustomerCode)

	rows, err := db.ExecuteQuery(sqlx, query)
	if err != nil {
		return nil, err
	}
	defer sqlx.Close()

	saleMap := make(map[string]*SaleWithInvoiceItems)

	for _, row := range rows {
		// ดึงข้อมูลจาก map
		saleCode := row["sale_code"].(string)
		sumInvoiceTotalAmount := 0.00
		// สร้าง Sale object
		sale := models.Sale{
			SaleCode:    saleCode,
			CompanyCode: row["customer_code"].(string),
			TotalAmount: row["total_amount"].(float64), // หรือแปลงถ้าเป็น string
		}

		// สร้าง InvoiceItem object
		var invoiceItem models.InvoiceItem
		idStr, _ := row["item_id"].(string)

		id, _ := uuid.Parse(idStr)

		if id != uuid.Nil { // ถ้ามี invoice item จริง
			invoiceItem = models.InvoiceItem{
				ID:          id,
				DocumentRef: row["document_ref"].(string),
				TotalAmount: row["invoice_total_amount"].(float64),
			}
			sumInvoiceTotalAmount += row["invoice_total_amount"].(float64)
		}

		// group by sale_code
		if existing, ok := saleMap[saleCode]; ok {
			if invoiceItem.ID != uuid.Nil {
				existing.InvoiceItems = append(existing.InvoiceItems, invoiceItem)
			}
		} else {
			newSale := &SaleWithInvoiceItems{
				Sale:         sale,
				InvoiceItems: []models.InvoiceItem{},
			}

			if invoiceItem.ID != uuid.Nil {
				newSale.InvoiceItems = append(newSale.InvoiceItems, invoiceItem)
			}

			saleMap[saleCode] = newSale
		}
	}

	// แปลง map เป็น slice
	var results []SaleWithInvoiceItems
	for _, v := range saleMap {
		results = append(results, *v)
	}

	return results, nil
}
