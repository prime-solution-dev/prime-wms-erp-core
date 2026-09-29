package summaryService

import (
	"encoding/json"
	"errors"
	"math"
	"prime-erp-core/internal/models"
	repositorySale "prime-erp-core/internal/repositories/sale"
	invoiceService "prime-erp-core/internal/services/invoice-service"
	paymentService "prime-erp-core/internal/services/payment-service"
	"slices"

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
	invoiceCodeSet := make(map[string]struct{})
	for _, resultValue := range result {
		for _, invoiceItemsValue := range resultValue.InvoiceItems {
			if _, exists := invoiceCodeSet[invoiceItemsValue.InvoiceCode]; exists {
				continue
			}
			invoiceCodeSet[invoiceItemsValue.InvoiceCode] = struct{}{}
			invoiceCode = append(invoiceCode, invoiceItemsValue.InvoiceCode)
		}
	}
	paymentValueMap := map[string]float64{}
	sumPaidInvoice := 0.00
	resultInvoiceMap := map[string][]models.Invoice{}
	resultInvoiceDepositMap := map[string]float64{}
	if len(invoiceCode) > 0 {
		invoiceForPayment := slices.Clone(invoiceCode)

		requestDataGetInvoice := map[string]interface{}{
			"invoice_ref": invoiceCode,
			"status":      []string{"COMPLETED"},
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

		for _, resultInvoiceValue := range resultInvoice {
			invoiceForPayment = append(invoiceForPayment, resultInvoiceValue.InvoiceCode)
			resultInvoiceMap[resultInvoiceValue.InvoiceRef] = append(resultInvoiceMap[resultInvoiceValue.InvoiceRef], resultInvoiceValue)
		}

		requestDataGetInvoiceDeposit := map[string]interface{}{
			"invoice_code": invoiceCode,
			"status":       []string{"COMPLETED"},
		}
		jsonBytesGetInvoiceDeposit, err := json.Marshal(requestDataGetInvoiceDeposit)
		if err != nil {
			return nil, err
		}
		invoiceDeposit, errGetInvoiceDeposit := invoiceService.GetInvoice(ctx, string(jsonBytesGetInvoiceDeposit))
		if errGetInvoiceDeposit != nil {
			return nil, errGetInvoice
		}
		resultInvoiceDeposit := invoiceDeposit.(invoiceService.ResultInvoice).Invoice

		for _, resultInvoiceDepositValue := range resultInvoiceDeposit {
			for _, invoiceItem := range resultInvoiceDepositValue.InvoiceItem {
				if invoiceItem.ArticleType == "DEPOSIT" {
					resultInvoiceDepositMap[resultInvoiceDepositValue.InvoiceCode] += invoiceItem.TotalAmount
				}
			}
		}

		requestDataGetPayment := map[string]interface{}{
			"invoice_code": invoiceForPayment,
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

		for _, paymentValue := range resultPayment {
			for _, paymentInvoiceValue := range paymentValue.PaymentInvoice {
				paymentValueMap[paymentInvoiceValue.InvoiceCode] = paymentValue.Amount
			}
			sumPaidInvoice += paymentValue.Amount
		}
	}

	saleAmount := 0.00
	sumInvoiceTotalAmountAR := 0.00
	sumInvoiceTotalAmountCN := 0.00
	sumInvoiceTotalAmountDN := 0.00
	sumPaymentTotalAmountDN := 0.00
	sumPaymentTotalAmountAR := 0.00

	for _, resultValue := range result {
		sumInvoiceItemTotalAmountAR := 0.00
		sumInvoiceItemTotalAmountCN := 0.00
		sumInvoiceItemTotalAmountDN := 0.00
		consumedCreditInvoice := []ConsumedCreditInvoice{}
		consumedInvoiceItems := 0.0
		seenInvoiceCodes := make(map[string]struct{})
		for _, invoiceItemsValue := range resultValue.InvoiceItems {
			invoiceAmount := invoiceItemsValue.InvoiceTotalAmount

			if invoiceItemsValue.InvoiceType == "AR" {
				sumInvoiceItemTotalAmountAR = invoiceItemsValue.InvoiceTotalAmount
			}
			if invoiceItemsValue.InvoiceType == "DN" {
				sumInvoiceItemTotalAmountDN = invoiceItemsValue.InvoiceTotalAmount
			}
			if invoiceItemsValue.InvoiceType == "CN" {
				invoiceAmount = -math.Abs(invoiceItemsValue.InvoiceTotalAmount)
				sumInvoiceItemTotalAmountCN = invoiceItemsValue.InvoiceTotalAmount
			}

			if _, exists := seenInvoiceCodes[invoiceItemsValue.InvoiceCode]; exists {
				continue
			}
			seenInvoiceCodes[invoiceItemsValue.InvoiceCode] = struct{}{}

			invoicePaidAmount := 0.00
			paymentItemMap, exist := paymentValueMap[invoiceItemsValue.InvoiceCode]
			if exist {
				invoicePaidAmount += paymentItemMap
			}
			paymentItemMapDeposit, existDeposit := resultInvoiceDepositMap[invoiceItemsValue.InvoiceCode]
			if existDeposit {
				invoicePaidAmount += paymentItemMapDeposit
			}
			consumedInvoiceItems += (invoiceAmount - invoicePaidAmount)
			consumedCreditInvoice = append(consumedCreditInvoice, ConsumedCreditInvoice{
				InvoiceCode:       invoiceItemsValue.InvoiceCode,
				InvoiceAmount:     invoiceAmount,
				InvoicePaidAmount: invoicePaidAmount,
				ConsumedAmount:    invoiceAmount - invoicePaidAmount,
			})

			invoiceItemMap, existResultInvoiceMap := resultInvoiceMap[invoiceItemsValue.InvoiceCode]
			if existResultInvoiceMap {

				for _, invoiceItemMapValue := range invoiceItemMap {
					amount := invoiceItemMapValue.TotalAmount
					paidAmount := 0.00
					if invoiceItemMapValue.InvoiceType == "DN" {
						paymentItemMap, exist := paymentValueMap[invoiceItemMapValue.InvoiceCode]
						if exist {
							paidAmount = paymentItemMap
							sumPaymentTotalAmountDN += paymentItemMap
						}
						sumInvoiceItemTotalAmountDN += invoiceItemMapValue.TotalAmount
					}

					if invoiceItemMapValue.InvoiceType == "CN" {
						amount = -math.Abs(invoiceItemMapValue.TotalAmount)
						sumInvoiceItemTotalAmountCN += invoiceItemMapValue.TotalAmount
					}
					consumedInvoiceItems += (amount - paidAmount)
					consumedCreditInvoice = append(consumedCreditInvoice, ConsumedCreditInvoice{
						InvoiceCode:       invoiceItemMapValue.InvoiceCode,
						InvoiceAmount:     amount,
						InvoicePaidAmount: paidAmount,
						ConsumedAmount:    amount - paidAmount,
					})
				}

			}

		}

		saleAmount += (resultValue.Sale.TotalAmount + consumedInvoiceItems)

		detail := ConsumedCreditDetail{
			SaleCode:       resultValue.Sale.SaleCode,
			SoAmount:       resultValue.Sale.TotalAmount,
			SoRemainAmount: (resultValue.Sale.TotalAmount) - ((sumInvoiceItemTotalAmountAR + sumInvoiceItemTotalAmountDN) - sumInvoiceItemTotalAmountCN),
			ConsumedAmount: (resultValue.Sale.TotalAmount) - ((sumInvoiceItemTotalAmountAR + sumInvoiceItemTotalAmountDN) - sumInvoiceItemTotalAmountCN),
			Invoice:        consumedCreditInvoice,
		}
		if consumedInvoiceItems != 0 || (resultValue.Sale.TotalAmount)-((sumInvoiceItemTotalAmountAR+sumInvoiceItemTotalAmountDN)-sumInvoiceItemTotalAmountCN) != 0 {
			resultConsumend = append(resultConsumend, detail)
		}

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
