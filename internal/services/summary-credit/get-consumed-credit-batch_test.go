package summaryService

import (
	"context"
	"errors"
	"prime-erp-core/internal/models"
	repositorySale "prime-erp-core/internal/repositories/sale"
	"reflect"
	"testing"
)

func TestConsumedCreditTotalsBatch(t *testing.T) {
	salesCalls, paymentCalls := 0, 0
	totals, err := getConsumedCreditTotals(context.Background(), []string{"A", "B", "C"},
		func(_ context.Context, codes []string) ([]repositorySale.SaleWithInvoiceItems, error) {
			salesCalls++
			if !reflect.DeepEqual(codes, []string{"A", "B", "C"}) {
				t.Fatalf("customers: %v", codes)
			}
			return []repositorySale.SaleWithInvoiceItems{
				{Sale: models.Sale{CustomerCode: "A", TotalAmount: 100}, InvoiceItems: []models.InvoiceItem{
					{InvoiceCode: "I1", InvoiceType: "AR", InvoiceTotalAmount: 40},
					{InvoiceCode: "I1", InvoiceType: "AR", InvoiceTotalAmount: 40},
				}},
				{Sale: models.Sale{CustomerCode: "A", TotalAmount: 10}},
				{Sale: models.Sale{CustomerCode: "B", TotalAmount: 200}, InvoiceItems: []models.InvoiceItem{
					{InvoiceCode: "I2", InvoiceType: "AR", InvoiceTotalAmount: 70},
				}},
			}, nil
		},
		func(codes []string) ([]models.Payment, error) {
			paymentCalls++
			if !reflect.DeepEqual(codes, []string{"I1", "I2"}) {
				t.Fatalf("invoices must be unique: %v", codes)
			}
			return []models.Payment{
				{PaymentInvoice: []models.PaymentInvoice{
					{InvoiceCode: "I1", Amount: 5}, {InvoiceCode: "I2", Amount: 20}, {InvoiceCode: "unrelated", Amount: 999},
				}},
				{PaymentInvoice: []models.PaymentInvoice{{InvoiceCode: "I1", Amount: 5}}},
			}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	// Legacy TotalAmount subtracts invoice payments once per invoice item.
	if totals["A"] != 170 || totals["B"] != 250 || totals["C"] != 0 {
		t.Fatalf("unexpected totals: %v", totals)
	}
	if salesCalls != 1 || paymentCalls != 1 {
		t.Fatalf("calls: sales=%d payments=%d", salesCalls, paymentCalls)
	}
}

func TestConsumedCreditTotalsWithoutInvoices(t *testing.T) {
	for _, noSales := range []bool{false, true} {
		totals, err := getConsumedCreditTotals(context.Background(), []string{"A"},
			func(context.Context, []string) ([]repositorySale.SaleWithInvoiceItems, error) {
				if noSales {
					return nil, nil
				}
				return []repositorySale.SaleWithInvoiceItems{{Sale: models.Sale{CustomerCode: "A", TotalAmount: 100}}}, nil
			},
			func([]string) ([]models.Payment, error) {
				t.Fatal("must not load all payments for empty invoice list")
				return nil, nil
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		expected := 100.0
		if noSales {
			expected = 0
		}
		if totals["A"] != expected {
			t.Fatalf("unexpected totals: %v", totals)
		}
	}
}

func TestConsumedCreditTotalsEmptyCustomers(t *testing.T) {
	totals, err := getConsumedCreditTotals(context.Background(), nil, nil, nil)
	if err != nil || len(totals) != 0 {
		t.Fatalf("totals=%v err=%v", totals, err)
	}
}

func TestConsumedCreditTotalsErrors(t *testing.T) {
	failure := errors.New("database failure")
	_, err := getConsumedCreditTotals(context.Background(), []string{"A"},
		func(context.Context, []string) ([]repositorySale.SaleWithInvoiceItems, error) { return nil, failure }, nil)
	if !errors.Is(err, failure) {
		t.Fatalf("sales error: %v", err)
	}
	_, err = getConsumedCreditTotals(context.Background(), []string{"A"},
		func(context.Context, []string) ([]repositorySale.SaleWithInvoiceItems, error) {
			return []repositorySale.SaleWithInvoiceItems{{InvoiceItems: []models.InvoiceItem{{InvoiceCode: "I1"}}}}, nil
		},
		func([]string) ([]models.Payment, error) { return nil, failure })
	if !errors.Is(err, failure) {
		t.Fatalf("payment error: %v", err)
	}
}
