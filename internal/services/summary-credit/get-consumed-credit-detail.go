package summaryService

import (
	"encoding/json"
	"errors"
	repositorySale "prime-erp-core/internal/repositories/sale"
	paymentService "prime-erp-core/internal/services/payment-service"

	"github.com/gin-gonic/gin"
)

type GetPaidInvoiceRequest struct {
	CustomerCode string `json:"customer_code"`
	PaidInvoice  bool   `json:"paid_invoice"`
}
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
type ResultGetPaidInvoices struct {
	TotalAmount float64 `json:"total_Amount"`
	PaidInvoice float64 `json:"paid_invoice"`
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
	sumPaidInvoice := 0.00
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
	totalAmount := 0.00
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
		totalAmount += resultValue.Sale.TotalAmount

		detail := ConsumedCreditDetail{
			SaleCode:       resultValue.Sale.SaleCode,
			SoAmount:       resultValue.Sale.TotalAmount,
			SoRemainAmount: resultValue.Sale.TotalAmount - sumInvoiceTotalAmount,
			ConsumedAmount: resultValue.Sale.TotalAmount - sumInvoiceTotalAmount,
			Invoice:        consumedCreditInvoice,
		}
		resultConsumend = append(resultConsumend, detail)
	}

	resultGetPaidInvoices := ResultGetPaidInvoices{
		TotalAmount: totalAmount,
		PaidInvoice: sumPaidInvoice,
	}

	if req.PaidInvoice {
		return resultGetPaidInvoices, nil
	} else {
		return resultConsumend, nil
	}

}
