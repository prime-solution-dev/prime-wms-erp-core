package summaryService

import (
	"context"
	"math"
	"prime-erp-core/internal/models"
	repositoryPayment "prime-erp-core/internal/repositories/payment"
	repositorySale "prime-erp-core/internal/repositories/sale"
)

// GetConsumedCreditTotals loads only the data needed for the credit list's TotalAmount.
func GetConsumedCreditTotals(ctx context.Context, customerCodes []string) (map[string]float64, error) {
	return getConsumedCreditTotals(ctx, customerCodes,
		repositorySale.GetSalesWithInvoiceItemsForCustomers,
		func(invoiceCodes []string) ([]models.Payment, error) {
			payments, _, _, err := repositoryPayment.GetPaymentPreload(nil, nil, nil, invoiceCodes, 0, 0)
			return payments, err
		})
}

func getConsumedCreditTotals(ctx context.Context, customerCodes []string,
	loadSales func(context.Context, []string) ([]repositorySale.SaleWithInvoiceItems, error),
	loadPayments func([]string) ([]models.Payment, error),
) (map[string]float64, error) {
	totals := map[string]float64{}
	if len(customerCodes) == 0 {
		return totals, nil
	}
	sales, err := loadSales(ctx, customerCodes)
	if err != nil {
		return nil, err
	}
	invoiceCodes := []string{}
	seen := map[string]bool{}
	for _, sale := range sales {
		for _, item := range sale.InvoiceItems {
			if !seen[item.InvoiceCode] {
				seen[item.InvoiceCode] = true
				invoiceCodes = append(invoiceCodes, item.InvoiceCode)
			}
		}
	}
	paid := map[string]float64{}
	if len(invoiceCodes) > 0 {
		payments, err := loadPayments(invoiceCodes)
		if err != nil {
			return nil, err
		}
		for _, payment := range payments {
			for _, item := range payment.PaymentInvoice {
				paid[item.InvoiceCode] += item.Amount
			}
		}
	}
	for _, sale := range sales {
		consumed := 0.0
		for _, item := range sale.InvoiceItems {
			amount := item.InvoiceTotalAmount
			if item.InvoiceType == "CN" {
				amount = -math.Abs(amount)
			}
			// Preserve GetConsumend's per-item calculation, including repeated invoices.
			consumed += amount - paid[item.InvoiceCode]
		}
		totals[sale.Sale.CustomerCode] += sale.Sale.TotalAmount + consumed
	}
	return totals, nil
}
