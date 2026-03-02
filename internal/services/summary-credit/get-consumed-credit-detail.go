package summaryService

import (
	"encoding/json"
	"errors"
	"math"
	"prime-erp-core/internal/models"
	repositorySale "prime-erp-core/internal/repositories/sale"
	invoiceService "prime-erp-core/internal/services/invoice-service"
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
	TotalAmount             float64 `json:"total_Amount"`
	SumInvoiceTotalAmountAR float64 `json:"sum_invoice_total_amount_ar"`
	SumInvoiceTotalAmountCN float64 `json:"sum_invoice_total_amount_cn"`
	SumInvoiceTotalAmountDN float64 `json:"sum_invoice_total_amount_dn"`
	SumPaymentTotalAmountDN float64 `json:"sum_payment_total_amount_dn"`
	SumPaymentTotalAmountAR float64 `json:"sum_payment_total_amount_ar"`
	PaidInvoice             float64 `json:"paid_invoice"`
}

func GetConsumend(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetPaidInvoiceRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	result, errGetSale := repositorySale.GetSalesWithInvoiceItems(req.CustomerCode, "")
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
	requestDataGetInvoice := map[string]interface{}{
		"invoice_ref": invoiceCode,
	}
	jsonBytesGetInvoice, err := json.Marshal(requestDataGetInvoice)
	if err != nil {
		return nil, err
	}
	invoice, errGetInvoice := invoiceService.GetInvoice(ctx, string(jsonBytesGetInvoice))
	if errGetInvoice != nil {
		return nil, errGetInvoice
	}
	resultInvoice := invoice.(invoiceService.ResultInvoice).Invoice
	resultInvoiceMap := map[string]models.Invoice{}
	for _, resultInvoiceValue := range resultInvoice {
		sumAmunt := 0.0
		for _, invoiceItemValue := range resultInvoiceValue.InvoiceItem {
			sumAmunt += invoiceItemValue.TotalAmount
		}
		resultInvoiceValue.TotalAmount = sumAmunt
		resultInvoiceMap[resultInvoiceValue.InvoiceRef] = resultInvoiceValue
	}

	saleAmount := 0.00
	sumInvoiceTotalAmountAR := 0.00
	sumInvoiceTotalAmountCN := 0.00
	sumInvoiceTotalAmountDN := 0.00
	sumPaymentTotalAmountDN := 0.00
	sumPaymentTotalAmountAR := 0.00

	for _, resultValue := range result {
		consumedCreditInvoice := []ConsumedCreditInvoice{}
		for _, invoiceItemsValue := range resultValue.InvoiceItems {
			invoicePaidAmount := 0.00
			paymentItemMap, exist := paymentValueMap[invoiceItemsValue.InvoiceCode]
			if exist {
				invoicePaidAmount = paymentItemMap
			}
			invoiceCode = append(invoiceCode, invoiceItemsValue.InvoiceCode)
			if invoiceItemsValue.InvoiceType == "AR" {
				sumInvoiceTotalAmountAR += invoiceItemsValue.TotalAmount
				sumPaymentTotalAmountAR += invoicePaidAmount
			}
			//invoiceAmount := invoiceItemsValue.TotalAmount
			invoiceItemMap, existResultInvoiceMap := resultInvoiceMap[invoiceItemsValue.InvoiceCode]
			if existResultInvoiceMap {
				if invoiceItemMap.InvoiceType == "DN" {
					sumInvoiceTotalAmountDN += invoiceItemMap.TotalAmount
					sumPaymentTotalAmountDN += invoicePaidAmount
				}

				if invoiceItemMap.InvoiceType == "CN" {
					sumInvoiceTotalAmountCN += invoiceItemMap.TotalAmount
					//invoiceAmount = -invoiceItemMap.TotalAmount
				}
			}
			invoiceAmount := invoiceItemsValue.InvoiceTotalAmount
			if invoiceItemsValue.InvoiceType == "CN" {
				invoiceAmount = -math.Abs(invoiceItemsValue.InvoiceTotalAmount)
			}

			consumedCreditInvoice = append(consumedCreditInvoice, ConsumedCreditInvoice{
				InvoiceCode:       invoiceItemsValue.InvoiceCode,
				InvoiceAmount:     invoiceAmount,
				InvoicePaidAmount: invoicePaidAmount,
				ConsumedAmount:    invoiceAmount + invoicePaidAmount,
			})
		}

		saleAmount += resultValue.Sale.TotalAmount

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
