package saleRepository

import (
	"context"
	"github.com/google/uuid"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
)

// GetSalesWithInvoiceItemsForCustomers loads eligible sales and AR items for a batch.
func GetSalesWithInvoiceItemsForCustomers(ctx context.Context, customerCodes []string) ([]SaleWithInvoiceItems, error) {
	result := []SaleWithInvoiceItems{}
	if len(customerCodes) == 0 {
		return result, nil
	}
	database, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(database)
	var rows []struct {
		SaleCode           string
		CustomerCode       string
		TotalAmount        float64
		ItemID             *uuid.UUID
		InvoiceCode        *string
		InvoiceType        string
		InvoiceTotalAmount float64
	}
	err = database.WithContext(ctx).Raw(`
 SELECT s.sale_code, s.customer_code, s.total_amount,
 it.id AS item_id, i.invoice_code,
 COALESCE(i.invoice_type, '') AS invoice_type,
 COALESCE(i.total_amount, 0) AS invoice_total_amount
 FROM sale s
 LEFT JOIN invoice_item it ON s.sale_code = it.document_ref
 LEFT JOIN invoice i ON i.id = it.invoice_id AND i.invoice_type = 'AR'
 WHERE s.status IN ('PENDING', 'COMPLETED')
 AND s.status_payment = 'PENDING' AND s.payment_method <> 'CASH'
 AND s.is_approved = true AND s.customer_code IN ?
 ORDER BY s.sale_code`, customerCodes).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	indices := map[[2]string]int{}
	for _, row := range rows {
		key := [2]string{row.CustomerCode, row.SaleCode}
		index, exists := indices[key]
		if !exists {
			index = len(result)
			indices[key] = index
			result = append(result, SaleWithInvoiceItems{Sale: models.Sale{
				SaleCode: row.SaleCode, CustomerCode: row.CustomerCode, TotalAmount: row.TotalAmount,
			}})
		}
		if row.ItemID != nil && *row.ItemID != uuid.Nil && row.InvoiceCode != nil {
			result[index].InvoiceItems = append(result[index].InvoiceItems, models.InvoiceItem{
				ID: *row.ItemID, InvoiceCode: *row.InvoiceCode,
				InvoiceType: row.InvoiceType, InvoiceTotalAmount: row.InvoiceTotalAmount,
			})
		}
	}
	return result, nil
}
