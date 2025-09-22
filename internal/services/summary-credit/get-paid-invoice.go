package summaryService

import (
	"encoding/json"
	"errors"
	invoiceService "prime-erp-core/internal/services/invoice-service"
	paymentService "prime-erp-core/internal/services/payment-service"
	saleService "prime-erp-core/internal/services/sale-service"

	"github.com/gin-gonic/gin"
)

type GetPaidInvoiceRequest struct {
	CustomerCode  []string `json:"customer_code"`
	SaleOrderCode []string `json:"sale_order_code"`
}
type ResultGetPaidInvoice struct {
	TotalAmount float64 `json:"total_Amount"`
	PaidInvoice float64 `json:"paid_invoice"`
}

func GetPaidInvoice(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetPaidInvoiceRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	requestDataGetSale := map[string]interface{}{
		"customer_code":  req.CustomerCode,
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
	totalAmount := 0.00
	for _, saleValue := range resultSale {
		saleCode = append(saleCode, saleValue.SaleCode)
		totalAmount += saleValue.TotalAmount
	}

	requestDataGetInvoice := map[string]interface{}{
		"customer_code": req.CustomerCode,
		"doc_ref":       req.SaleOrderCode,
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
	invoiceCode := []string{}
	for _, invoiceValue := range resultInvoice {
		invoiceCode = append(invoiceCode, invoiceValue.InvoiceCode)
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
	paidInvoice := 0.00
	resultPayment := paymentle.(paymentService.ResultPayment).Payment
	for _, paymentValue := range resultPayment {
		for _, paymentInvoiceValue := range paymentValue.PaymentInvoice {
			paidInvoice += paymentInvoiceValue.Amount
		}
	}

	resultGetPaidInvoice := ResultGetPaidInvoice{
		TotalAmount: totalAmount,
		PaidInvoice: paidInvoice,
	}

	return resultGetPaidInvoice, nil
}
