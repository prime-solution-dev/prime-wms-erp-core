package invoiceService

import (
	"encoding/json"
	"errors"
	repositoryInvoice "prime-erp-core/internal/repositories/invoice"
	repositorySale "prime-erp-core/internal/repositories/sale"

	"github.com/gin-gonic/gin"
)

type SaleAutoStatusPaymentReq struct {
	InvoiceCode []string `json:"invoice_code"`
}

func SaleAutoStatusPayment(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req SaleAutoStatusPaymentReq

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}
	saleCode := ""

	invoice, _, _, errDeposit := repositoryInvoice.GetInvoicePreload(nil, req.InvoiceCode, nil, nil, nil, nil, 0, 0)
	if errDeposit != nil {
		return nil, errDeposit
	}
	for _, invoiceValue := range invoice {
		for _, invoiceItemValue := range invoiceValue.InvoiceItem {
			saleCode = invoiceItemValue.DocumentRef
		}
	}
	result, errGetSale := repositorySale.GetSalesWithInvoiceItems("", saleCode)
	if errGetSale != nil {
		return nil, errGetSale
	}

	saleAmount := 0.00
	sumInvoiceTotalAmountAR := 0.00
	sumInvoiceTotalAmountCN := 0.00
	sumInvoiceTotalAmountDN := 0.00
	for _, resultValue := range result {
		for _, invoiceItemsValue := range resultValue.InvoiceItems {
			if invoiceItemsValue.InvoiceType == "CN" {
				sumInvoiceTotalAmountCN += invoiceItemsValue.TotalAmount
				if invoiceItemsValue.InvoiceType == "DN" {
					sumInvoiceTotalAmountDN += invoiceItemsValue.TotalAmount
				}
				if invoiceItemsValue.InvoiceType == "AR" {
					sumInvoiceTotalAmountAR += invoiceItemsValue.TotalAmount
				}
			}
			saleAmount += resultValue.Sale.TotalAmount
		}
	}
	consumedCredit := (saleAmount - sumInvoiceTotalAmountCN + sumInvoiceTotalAmountDN) + sumInvoiceTotalAmountAR
	if consumedCredit <= 0 {

	}

	return nil, nil
}
