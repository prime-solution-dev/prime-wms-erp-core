package saleRepository

import (
	"errors"
	"fmt"
	"math"
	externalService "prime-erp-core/external/customer-service"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"strings"
	"time"

	"github.com/google/uuid"
)

// getCustomerCodesByName ค้นหา customer codes จาก customer service โดยใช้ customer name
func getCustomerCodesByName(customerNameLike string) ([]string, error) {
	if len(customerNameLike) == 0 {
		return nil, nil
	}

	getCustomerByNameRequest := externalService.GetCustomerRequest{
		CustomerNameLike: customerNameLike,
		Page:             1,
		PageSize:         1000, // เอาเยอะๆ เพื่อให้ได้ customerCode ทั้งหมดที่ match
	}

	customerByNameData, err := externalService.GetCustomer(getCustomerByNameRequest)
	if err != nil {
		fmt.Println("failed to fetch customers by name:", err)
		return nil, errors.New("failed to fetch customers by name: " + err.Error())
	}

	fmt.Printf("Found %d customers matching name like '%s'\n", len(customerByNameData.Customers), customerNameLike)

	// เก็บ customerCode ทั้งหมดที่ได้จากการค้นหาด้วย name
	var customerCodes []string
	for _, customer := range customerByNameData.Customers {
		customerCodes = append(customerCodes, customer.CustomerCode)
	}

	fmt.Println("Customer codes from name search:", customerCodes)
	return customerCodes, nil
}

// buildStatusFilterConditions สร้างเงื่อนไข SQL สำหรับ status filter
func buildStatusFilterConditions(statusFilters []string) string {
	if len(statusFilters) == 0 {
		return ""
	}

	var conditions []string

	for _, statusFilter := range statusFilters {
		switch strings.ToLower(statusFilter) {
		case "new":
			conditions = append(conditions, "(sale.status = 'PENDING' AND sale.status_approve = 'PENDING' AND delivery_booking_item.document_ref_item IS NULL)")
		case "waitapprove":
			conditions = append(conditions, "(sale.status = 'PENDING' AND sale.status_approve = 'PROCESS' AND delivery_booking_item.document_ref_item IS NULL)")
		case "approved":
			conditions = append(conditions, "(sale.status = 'PENDING' AND sale.status_approve = 'COMPLETED' AND delivery_booking_item.document_ref_item IS NULL)")
		case "reject":
			conditions = append(conditions, "(sale.status = 'PENDING' AND sale.status_approve = 'REJECT' AND delivery_booking_item.document_ref_item IS NULL)")
		case "review":
			conditions = append(conditions, "(sale.status = 'PENDING' AND sale.status_approve = 'REVIEW' AND delivery_booking_item.document_ref_item IS NULL)")
		case "canceled":
			conditions = append(conditions, "sale.status = 'CANCELED'")
		case "draft":
			conditions = append(conditions, "sale.status = 'TEMP'")
		case "completed":
			conditions = append(conditions, "sale.status = 'COMPLETED'")
		case "partial":
			// สำหรับ partial จะต้องมี delivery items อยู่ แต่ยัง PENDING
			conditions = append(conditions, "(sale.status = 'PENDING' AND delivery_booking_item.document_ref_item IS NOT NULL)")
		}
	}

	if len(conditions) > 0 {
		return " AND (" + strings.Join(conditions, " OR ") + ")"
	}
	return ""
}

// Create
func GetSalePreload(id []uuid.UUID, saleCode []string, customerCode []string, status []string, statusApprove []string, statusPayment []string, isApproved []bool, saleCodeLike string, documentRefLike string, customerCodeLike string, customerNameLike string, createDateStart string, createDateEnd string, expirePriceDateStart string, expirePriceDateEnd string, deliveryDateStart string, deliveryDateEnd string, statusFilter []string, page int, pageSize int) ([]models.Sale, int, int, error) {
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

	searchSaleCode := ""
	if len(saleCode) > 0 {
		quotedStrings := make([]string, len(saleCode))
		for i, s := range saleCode {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchSaleCode = fmt.Sprintf(` and sale.sale_code IN (%s)`, whereInClause)
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
	searchStatusApprove := ""
	if len(statusApprove) > 0 {
		quotedStrings := make([]string, len(statusApprove))
		for i, s := range statusApprove {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchStatusApprove = fmt.Sprintf(` and sale.status_approve IN (%s)`, whereInClause)
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

	// New search conditions
	searchSaleCodeLike := ""
	if len(saleCodeLike) > 0 {
		searchSaleCodeLike = fmt.Sprintf(` AND sale.sale_code ILIKE '%%%s%%'`, saleCodeLike)
	}

	searchCustomerCodeLike := ""
	if len(customerCodeLike) > 0 {
		searchCustomerCodeLike = fmt.Sprintf(` AND sale.customer_code ILIKE '%%%s%%'`, customerCodeLike)
	}

	searchDocumentRefLike := ""
	if len(documentRefLike) > 0 {
		searchDocumentRefLike = fmt.Sprintf(` AND sale_item.document_ref ILIKE '%%%s%%'`, documentRefLike)
	}

	// Handle customer name search
	searchCustomerByName := ""
	if len(customerNameLike) > 0 {
		customerCodesFromName, err := getCustomerCodesByName(customerNameLike)
		if err != nil {
			return nil, 0, 0, err
		}
		if len(customerCodesFromName) > 0 {
			quotedStrings := make([]string, len(customerCodesFromName))
			for i, s := range customerCodesFromName {
				quotedStrings[i] = fmt.Sprintf("'%s'", s)
			}
			whereInClause := strings.Join(quotedStrings, ", ")
			searchCustomerByName = fmt.Sprintf(` AND sale.customer_code IN (%s)`, whereInClause)
		} else {
			// No customers found, return empty result
			searchCustomerByName = " AND 1 = 0"
		}
	}

	// Date range searches
	searchCreateDate := ""
	if len(createDateStart) > 0 && len(createDateEnd) > 0 {
		searchCreateDate = fmt.Sprintf(` AND sale.create_date BETWEEN '%s' AND '%s'`, createDateStart, createDateEnd)
	}

	searchExpirePriceDate := ""
	if len(expirePriceDateStart) > 0 && len(expirePriceDateEnd) > 0 {
		searchExpirePriceDate = fmt.Sprintf(` AND sale.expire_price_date BETWEEN '%s' AND '%s'`, expirePriceDateStart, expirePriceDateEnd)
	}

	searchDeliveryDate := ""
	if len(deliveryDateStart) > 0 && len(deliveryDateEnd) > 0 {
		searchDeliveryDate = fmt.Sprintf(` AND sale.delivery_date BETWEEN '%s' AND '%s'`, deliveryDateStart, deliveryDateEnd)
	}

	// Status filter conditions
	statusFilterCondition := buildStatusFilterConditions(statusFilter)

	var saleID []uuid.UUID
	gormx.Table("sale").Select("sale.id").
		Joins("inner join sale_item on sale.id = sale_item.sale_id").
		Joins("left join sale_deposit on sale.id = sale_deposit.sale_id").
		Joins("left join delivery_booking_item on sale_item.sale_item = delivery_booking_item.document_ref_item").
		Where("1=1 " + searchID + "" + searchSaleCode + "" + searchCustomerCode + "" + searchIsStatus + "" + searchStatusApprove + "" + searchStatusPayment + "" + searchIsApproved + "" + searchSaleCodeLike + "" + searchCustomerCodeLike + "" + searchDocumentRefLike + "" + searchCustomerByName + "" + searchCreateDate + "" + searchExpirePriceDate + "" + searchDeliveryDate + "" + statusFilterCondition + "").
		Group("sale.id").Scan(&saleID)

	if len(saleID) > 0 {

		var count = len(saleID)

		query := gormx.Preload("SaleItem.DeliveryItems").Preload("SaleDeposit")

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

func GetSalesWithInvoiceItems(customerCode string, saleCode string) ([]SaleWithInvoiceItems, error) {

	sqlx, err := db.ConnectSqlx(`prime_erp`)
	if err != nil {
		return nil, err
	}
	search := ""
	if customerCode != "" {
		search += fmt.Sprintf(` and s.customer_code  = '%s'`, customerCode)
	}
	if saleCode != "" {
		search += fmt.Sprintf(` and s.sale_code  = '%s'`, saleCode)
	}

	query := fmt.Sprintf(`
		    SELECT 
			s.id,
        s.sale_code, 
        s.customer_code, 
		s.status_payment,
        s.total_amount,
		s.delivery_date,
        it.id as item_id, 
        it.document_ref, 
        it.total_amount as invoice_total_amount,
		i.invoice_code,
		i.invoice_type,
		i.total_amount as invoice_total_amount_header 
    FROM sale s
    LEFT JOIN invoice_item it ON s.sale_code = it.document_ref 
	LEFT JOIN invoice  i  ON i.id = it.invoice_id  and (i.invoice_type = 'AR')
		where   s.status in ('PENDING','COMPLETED') and status_payment = 'PENDING' and is_approved = true 
		%s
		 ORDER BY s.sale_code
	`, search)

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
		idStrSale, _ := row["id"].(string)

		idSale, _ := uuid.Parse(idStrSale)
		// สร้าง Sale object
		var deliveryDate *time.Time

		if row["delivery_date"] != nil {
			t := row["delivery_date"].(time.Time)
			deliveryDate = &t
		} else {
			deliveryDate = nil
		}
		sale := models.Sale{
			ID:            idSale,
			SaleCode:      saleCode,
			DeliveryDate:  deliveryDate,
			CustomerCode:  row["customer_code"].(string),
			TotalAmount:   row["total_amount"].(float64),
			StatusPayment: row["status_payment"].(string),
		}

		// สร้าง InvoiceItem object
		var invoiceItem models.InvoiceItem
		id := uuid.Nil
		if row["item_id"] != nil {
			idStr, _ := row["item_id"].(string)
			id, _ = uuid.Parse(idStr)
		}

		if id != uuid.Nil { // ถ้ามี invoice item จริง
			sumInvoiceTotalAmount += row["invoice_total_amount"].(float64)
			if row["invoice_code"] != nil {
				invoiceItem = models.InvoiceItem{
					ID:                 id,
					InvoiceCode:        row["invoice_code"].(string),
					DocumentRef:        row["document_ref"].(string),
					TotalAmount:        row["invoice_total_amount"].(float64),
					InvoiceType:        row["invoice_type"].(string),
					InvoiceTotalAmount: row["invoice_total_amount_header"].(float64),
				}
			}

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
func UpdateStatusPayment(sale []models.Sale) (int, error) {
	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return 0, err
	}
	rowsAffected := 0
	for _, saleValue := range sale {
		result := gormx.Table("sale").Where("id = ?", saleValue.ID).Select("status_payment").Updates(&saleValue)

		if result.Error != nil {
			gormx.Rollback()
			return 0, result.Error
		}
		rowsAffected = int(result.RowsAffected)
	}

	return rowsAffected, nil
}
