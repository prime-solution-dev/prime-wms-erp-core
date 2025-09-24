package summaryService

import (
	"encoding/json"
	"errors"
	repositorySale "prime-erp-core/internal/repositories/sale"
	paymentService "prime-erp-core/internal/services/payment-service"

	"github.com/gin-gonic/gin"
)

type ConsumedCreditDetail struct {
	SaleCode       string                  `json:"sale_code"`
	SoAmount       float64                 `json:"so_amount"`
	SoRemainAmount float64                 `json:"so_remain_amount"`
	ConsumedAmount float64                 `json:"consumed_amount"`
	Invoice        []ConsumedCreditInvoice `json:"invoice"`
}
type ConsumedCreditInvoice struct {
	InvoiceCode       string  `json:"invoice_code"`
	InvoiceAmount     float64 `json:"invoice_amount"`
	InvoicePaidAmount float64 `json:"invoice_paid_amount"`
	ConsumedAmount    float64 `json:"consumed_amount"`
}

func GetConsumend(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetPaidInvoiceRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	result, errGetSale := repositorySale.GetSalesWithInvoiceItems(req.CustomerCode)
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

	requestDataGetPayment := map[string]interface{}{
		"invoice_code": invoiceCode,
	}

	jsonBytesPayment, err := json.Marshal(requestDataGetPayment)
	if err != nil {
		return nil, err
	}

	paymentle, errGetPayment := paymentService.GetPayment(ctx, string(jsonBytesPayment))
	if errGetPayment != nil {
		return nil, errGetPayment
	}
	resultPayment := paymentle.(paymentService.ResultPayment).Payment
	paymentValueMap := map[string]float64{}
	for _, paymentValue := range resultPayment {
		for _, paymentInvoiceValue := range paymentValue.PaymentInvoice {

			paymentItemMap, exist := paymentValueMap[paymentInvoiceValue.InvoiceCode]
			if exist {
				paymentValueMap[paymentInvoiceValue.InvoiceCode] = paymentItemMap + paymentInvoiceValue.Amount
			} else {
				paymentValueMap[paymentInvoiceValue.InvoiceCode] = paymentItemMap
			}

		}

	}

	for _, resultValue := range result {
		sumInvoiceTotalAmount := 0.00
		consumedCreditInvoice := []ConsumedCreditInvoice{}
		for _, invoiceItemsValue := range resultValue.InvoiceItems {
			invoicePaidAmount := 0.00
			paymentItemMap, exist := paymentValueMap[invoiceItemsValue.InvoiceCode]
			if exist {
				invoicePaidAmount = paymentItemMap
			}
			sumInvoiceTotalAmount += invoiceItemsValue.TotalAmount
			consumedCreditInvoice = append(consumedCreditInvoice, ConsumedCreditInvoice{
				InvoiceCode:       invoiceItemsValue.InvoiceCode,
				InvoiceAmount:     invoiceItemsValue.TotalAmount,
				InvoicePaidAmount: invoicePaidAmount,
				ConsumedAmount:    invoiceItemsValue.TotalAmount - invoicePaidAmount,
			})
			invoiceCode = append(invoiceCode, invoiceItemsValue.InvoiceCode)
		}

		detail := ConsumedCreditDetail{
			SaleCode:       resultValue.Sale.SaleCode,
			SoAmount:       resultValue.Sale.TotalAmount,
			SoRemainAmount: resultValue.Sale.TotalAmount - sumInvoiceTotalAmount,
			ConsumedAmount: resultValue.Sale.TotalAmount - sumInvoiceTotalAmount,
			Invoice:        consumedCreditInvoice,
		}

		resultConsumend = append(resultConsumend, detail)

	}

	/* var req GetPaidInvoiceRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}
	resultConsumend := []ConsumedCreditDetail{}
	requestDataGetSale := map[string]interface{}{
		"customer_code":  []string{req.CustomerCode},
		"status":         []string{"PENDING", "COMPLETED"},
		"status_payment": []string{"PENDING"},
		"is_approved":    []bool{true},
	}

	jsonBytesSale, err := json.Marshal(requestDataGetSale)
	if err != nil {
		return nil, err
	}

	sale, errGetSale := saleService.GetSale(ctx, string(jsonBytesSale))
	if errGetSale != nil {
		return nil, errGetSale
	}
	resultSale := sale.(saleService.ResultSale).Sale
	saleCode := []string{}
	for _, saleValue := range resultSale {
		saleCode = append(saleCode, saleValue.SaleCode)
	}

	requestDataGetInvoice := map[string]interface{}{
		"customer_code": []string{req.CustomerCode},
		"doc_ref":       saleCode,
	}

	jsonBytesInvoice, err := json.Marshal(requestDataGetInvoice)
	if err != nil {
		return nil, err
	}

	invoice, errGetInvoice := invoiceService.GetInvoice(ctx, string(jsonBytesInvoice))
	if errGetInvoice != nil {
		return nil, errGetInvoice
	}
	resultInvoice := invoice.(invoiceService.ResultInvoice).Invoice

	for _, sale := range resultSale {
		var matchedInvoices []models.Invoice
		sumInvoiceTotalAmount := 0.00
		consumedCreditInvoice := []ConsumedCreditInvoice{}
		for _, invoice := range resultInvoice {
			if invoice.DocRef == sale.SaleCode {
				sumInvoiceTotalAmount += invoice.TotalAmount
				matchedInvoices = append(matchedInvoices, invoice)
			}
			consumedCreditInvoice = append(consumedCreditInvoice, ConsumedCreditInvoice{
				InvoiceCode:       invoice.InvoiceCode,
				InvoiceAmount:     invoice.TotalAmount,
				InvoicePaidAmount: 1,
				ConsumedAmount:    1,
			})
		}

		detail := ConsumedCreditDetail{
			SaleCode:       sale.SaleCode,
			SoAmount:       sale.TotalAmount,
			SoRemainAmount: sale.TotalAmount - sumInvoiceTotalAmount,
			ConsumedAmount: sale.TotalAmount - sumInvoiceTotalAmount,
			Invoice:        consumedCreditInvoice,
		}

		resultConsumend = append(resultConsumend, detail)
	} */

	return resultConsumend, nil
}
