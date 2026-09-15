package summaryService

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"math"
	"math/rand"
	"prime-erp-core/internal/models"
	repositorySale "prime-erp-core/internal/repositories/sale"
	invoiceService "prime-erp-core/internal/services/invoice-service"
	paymentService "prime-erp-core/internal/services/payment-service"
	"testing"
)

// Frozen GetConsumend implementation at the time of this audit.
// Only its three data-loading calls are injected; the calculation is copied verbatim.
// This compares service logic, not SQL execution or live database contents.

func legacyConsumedForAudit(ctx *gin.Context, jsonPayload string,
	loadSales func(string, string) ([]repositorySale.SaleWithInvoiceItems, error),
	loadPayment func(*gin.Context, string) (interface{}, error),
	loadInvoice func(*gin.Context, string) (interface{}, error),
) (interface{}, error) {

	var req GetPaidInvoiceRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	result, errGetSale := loadSales(req.CustomerCode, "")
	if errGetSale != nil {
		return nil, errGetSale
	}
	resultConsumend := []ConsumedCreditDetail{}
	invoiceCode := []string{}
	for _, resultValue := range result {
		for _, invoiceItemsValue := range resultValue.InvoiceItems {
			invoiceCode = append(invoiceCode, invoiceItemsValue.InvoiceCode)
		}
	}
	paymentValueMap := map[string]float64{}
	sumPaidInvoice := 0.00
	resultInvoiceMap := map[string]models.Invoice{}
	if len(invoiceCode) > 0 {
		requestDataGetPayment := map[string]interface{}{
			"invoice_code": invoiceCode,
		}

		jsonBytesPayment, err := json.Marshal(requestDataGetPayment)
		if err != nil {
			return nil, err
		}

		paymentle, errGetPayment := loadPayment(ctx, string(jsonBytesPayment))
		if errGetPayment != nil {
			return nil, errGetPayment
		}
		resultPayment := paymentle.(paymentService.ResultPayment).Payment

		for _, paymentValue := range resultPayment {
			for _, paymentInvoiceValue := range paymentValue.PaymentInvoice {

				paymentItemMap, exist := paymentValueMap[paymentInvoiceValue.InvoiceCode]
				if exist {
					paymentValueMap[paymentInvoiceValue.InvoiceCode] = paymentItemMap + paymentInvoiceValue.Amount
				} else {
					paymentValueMap[paymentInvoiceValue.InvoiceCode] = paymentInvoiceValue.Amount
				}
				sumPaidInvoice += paymentInvoiceValue.Amount

			}

		}

		requestDataGetInvoice := map[string]interface{}{
			"invoice_ref": invoiceCode,
		}
		jsonBytesGetInvoice, err := json.Marshal(requestDataGetInvoice)
		if err != nil {
			return nil, err
		}
		invoice, errGetInvoice := loadInvoice(ctx, string(jsonBytesGetInvoice))
		if errGetInvoice != nil {
			return nil, errGetInvoice
		}
		resultInvoice := invoice.(invoiceService.ResultInvoice).Invoice

		for _, resultInvoiceValue := range resultInvoice {
			sumAmunt := 0.0
			for _, invoiceItemValue := range resultInvoiceValue.InvoiceItem {
				sumAmunt += invoiceItemValue.TotalAmount
			}
			resultInvoiceValue.TotalAmount = sumAmunt
			resultInvoiceMap[resultInvoiceValue.InvoiceRef] = resultInvoiceValue
		}
	}

	saleAmount := 0.00
	sumInvoiceTotalAmountAR := 0.00
	sumInvoiceTotalAmountCN := 0.00
	sumInvoiceTotalAmountDN := 0.00
	sumPaymentTotalAmountDN := 0.00
	sumPaymentTotalAmountAR := 0.00

	for _, resultValue := range result {
		consumedCreditInvoice := []ConsumedCreditInvoice{}
		consumedInvoiceItems := 0.0
		for _, invoiceItemsValue := range resultValue.InvoiceItems {
			invoicePaidAmount := 0.00
			paymentItemMap, exist := paymentValueMap[invoiceItemsValue.InvoiceCode]
			if exist {
				invoicePaidAmount = paymentItemMap
			}
			invoiceCode = append(invoiceCode, invoiceItemsValue.InvoiceCode)
			if invoiceItemsValue.InvoiceType == "AR" {
				sumInvoiceTotalAmountAR += invoiceItemsValue.TotalAmount
				//sumPaymentTotalAmountAR += invoicePaidAmount
			}
			if invoiceItemsValue.InvoiceType == "DN" {
				sumInvoiceTotalAmountDN += invoiceItemsValue.TotalAmount
				//sumPaymentTotalAmountDN += invoicePaidAmount
			}
			//invoiceAmount := invoiceItemsValue.TotalAmount
			invoiceItemMap, existResultInvoiceMap := resultInvoiceMap[invoiceItemsValue.InvoiceCode]
			if existResultInvoiceMap {
				if invoiceItemMap.InvoiceType == "DN" {
					//sumInvoiceTotalAmountDN += invoiceItemMap.TotalAmount
					sumPaymentTotalAmountDN += invoicePaidAmount
				}

				/* if invoiceItemMap.InvoiceType == "CN" {
					sumInvoiceTotalAmountCN += invoiceItemMap.TotalAmount
					//invoiceAmount = -invoiceItemMap.TotalAmount
				} */
			}
			invoiceAmount := invoiceItemsValue.InvoiceTotalAmount
			if invoiceItemsValue.InvoiceType == "CN" {
				invoiceAmount = -math.Abs(invoiceItemsValue.InvoiceTotalAmount)
				sumInvoiceTotalAmountCN += invoiceItemMap.TotalAmount
			}

			consumedCreditInvoice = append(consumedCreditInvoice, ConsumedCreditInvoice{
				InvoiceCode:       invoiceItemsValue.InvoiceCode,
				InvoiceAmount:     invoiceAmount,
				InvoicePaidAmount: invoicePaidAmount,
				ConsumedAmount:    invoiceAmount - invoicePaidAmount,
			})
			consumedInvoiceItems += (invoiceAmount - invoicePaidAmount)
		}

		saleAmount += (resultValue.Sale.TotalAmount + consumedInvoiceItems)

		detail := ConsumedCreditDetail{
			SaleCode:       resultValue.Sale.SaleCode,
			SoAmount:       resultValue.Sale.TotalAmount,
			SoRemainAmount: (resultValue.Sale.TotalAmount) - sumInvoiceTotalAmountAR,
			ConsumedAmount: (resultValue.Sale.TotalAmount) - sumInvoiceTotalAmountAR,
			Invoice:        consumedCreditInvoice,
		}
		resultConsumend = append(resultConsumend, detail)
	}

	resultGetPaidInvoices := ResultGetPaidInvoices{
		TotalAmount:             saleAmount,
		SumInvoiceTotalAmountAR: sumInvoiceTotalAmountAR,
		SumInvoiceTotalAmountCN: sumInvoiceTotalAmountCN,
		SumInvoiceTotalAmountDN: sumInvoiceTotalAmountDN,
		SumPaymentTotalAmountDN: sumPaymentTotalAmountDN,
		SumPaymentTotalAmountAR: sumPaymentTotalAmountAR,
		PaidInvoice:             sumPaidInvoice,
	}

	if req.PaidInvoice {
		return resultGetPaidInvoices, nil
	} else {
		return resultConsumend, nil
	}

}

// auditPaymentSelection models the repository's header selection and full
// PaymentInvoice preload. A matching payment header must only appear once.
func auditPaymentSelection(payments []models.Payment, codes []string) []models.Payment {
	selected := []models.Payment{}
	for _, payment := range payments {
		matched := false
		for _, item := range payment.PaymentInvoice {
			for _, code := range codes {
				if item.InvoiceCode == code {
					matched = true
				}
			}
		}
		if matched {
			selected = append(selected, payment)
		}
	}
	return selected
}

func TestBatchMatchesLegacyConsumedCalculation(t *testing.T) {
	for seed := int64(0); seed < 200; seed++ {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			rng := rand.New(rand.NewSource(seed))
			customers := []string{"A", "B", "C", "no-sales"}
			sales := []repositorySale.SaleWithInvoiceItems{}
			for _, customer := range customers[:3] {
				for n := 0; n < rng.Intn(6); n++ {
					sale := repositorySale.SaleWithInvoiceItems{Sale: models.Sale{
						CustomerCode: customer, SaleCode: fmt.Sprintf("%s-%d", customer, n),
						TotalAmount: float64(rng.Intn(100000)) / 100,
					}}
					for j := 0; j < rng.Intn(6); j++ {
						sale.InvoiceItems = append(sale.InvoiceItems, models.InvoiceItem{
							InvoiceCode:        fmt.Sprintf("I%d", rng.Intn(6)),
							InvoiceType:        []string{"AR", "CN", "DN"}[rng.Intn(3)],
							InvoiceTotalAmount: float64(rng.Intn(100000)-20000) / 100,
							TotalAmount:        float64(rng.Intn(10000)) / 100,
						})
					}
					sales = append(sales, sale)
				}
			}
			payments := []models.Payment{}
			for n := 0; n < 8; n++ {
				payment := models.Payment{}
				for j := 0; j < rng.Intn(6); j++ {
					payment.PaymentInvoice = append(payment.PaymentInvoice, models.PaymentInvoice{
						InvoiceCode: fmt.Sprintf("I%d", rng.Intn(8)),
						Amount:      float64(rng.Intn(50000)-10000) / 100,
					})
				}
				payments = append(payments, payment)
			}
			batch, err := getConsumedCreditTotals(context.Background(), customers,
				func(context.Context, []string) ([]repositorySale.SaleWithInvoiceItems, error) { return sales, nil },
				func(codes []string) ([]models.Payment, error) { return auditPaymentSelection(payments, codes), nil },
			)
			if err != nil {
				t.Fatal(err)
			}
			for _, customer := range customers {
				payload, _ := json.Marshal(GetPaidInvoiceRequest{CustomerCode: customer, PaidInvoice: true})
				legacy, err := legacyConsumedForAudit(nil, string(payload),
					func(code, _ string) ([]repositorySale.SaleWithInvoiceItems, error) {
						selected := []repositorySale.SaleWithInvoiceItems{}
						for _, sale := range sales {
							if sale.Sale.CustomerCode == code {
								selected = append(selected, sale)
							}
						}
						return selected, nil
					},
					func(_ *gin.Context, payload string) (interface{}, error) {
						var req paymentService.GetPaymentRequest
						if err := json.Unmarshal([]byte(payload), &req); err != nil {
							return nil, err
						}
						return paymentService.ResultPayment{Payment: auditPaymentSelection(payments, req.InvoiceCode)}, nil
					},
					func(*gin.Context, string) (interface{}, error) {
						// Deliberately nonzero related invoice data: it must not affect TotalAmount.
						return invoiceService.ResultInvoice{Invoice: []models.Invoice{
							{InvoiceRef: "I1", InvoiceType: "DN", InvoiceItem: []models.InvoiceItem{{TotalAmount: 9876.54}}},
							{InvoiceRef: "I2", InvoiceType: "CN", InvoiceItem: []models.InvoiceItem{{TotalAmount: 1234.56}}},
						}}, nil
					},
				)
				if err != nil {
					t.Fatal(err)
				}
				expected := legacy.(ResultGetPaidInvoices).TotalAmount
				actual := batch[customer]
				if math.Abs(expected-actual) > 1e-8 {
					t.Fatalf("customer %s legacy %.12f batch %.12f", customer, expected, actual)
				}
				// Exercise the unchanged deposit/VAT and available-credit formulas as well.
				deposit := math.Round((123.45*1.07)*100) / 100
				oldConsumed, newConsumed := expected-deposit, actual-deposit
				oldBalance, newBalance := (10000.0+500.0)-oldConsumed, (10000.0+500.0)-newConsumed
				if math.Abs(oldBalance-newBalance) > 1e-8 {
					t.Fatalf("balance differs: %v %v", oldBalance, newBalance)
				}
			}
		})
	}
}
