package summaryService

import (
	"encoding/json"
	"errors"
	repositorySale "prime-erp-core/internal/repositories/sale"
	customerService "prime-erp-core/internal/services/customer-service"
	paymentService "prime-erp-core/internal/services/payment-service"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OutStandingSoRes struct {
	ID            uuid.UUID  `json:"id"`
	CustomerCode  string     `json:"customer_code"`
	CustomerName  string     `json:"customer_name"`
	SaleCode      string     `json:"sale_code"`
	SaleDate      *time.Time `json:"sale_date"`
	SaleAmount    float64    `json:"sale_amount"`
	StatusPayment string     `json:"status_payment"`
	Paid          float64    `json:"paid"`
	OutStandingSo float64    `json:"out_standing_so"`
}

func GetOutStandingSo(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetPaidInvoiceRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	result, errGetSale := repositorySale.GetSalesWithInvoiceItems(req.CustomerCode, "")
	if errGetSale != nil {
		return nil, errGetSale
	}
	resultOutStandingSoRes := []OutStandingSoRes{}
	invoiceCode := []string{}
	customerCode := []string{}
	for _, resultValue := range result {
		for _, invoiceItemsValue := range resultValue.InvoiceItems {
			invoiceCode = append(invoiceCode, invoiceItemsValue.InvoiceCode)
			customerCode = append(customerCode, resultValue.Sale.CustomerCode)
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
				paymentValueMap[paymentInvoiceValue.InvoiceCode] = paymentInvoiceValue.Amount
			}

		}

	}

	requestData := map[string]interface{}{
		"customer_code": customerCode,
	}

	customers, err := customerService.GetCustomers(requestData)
	if err != nil {
		return nil, err
	}

	convertCustomerMap := map[string]customerService.GetCustomerResponse{}
	for _, customer := range customers.Customers {
		convertCustomerMap[customer.CustomerCode] = customer
	}

	for _, resultValue := range result {
		paidSale := 0.00
		for _, invoiceItemsValue := range resultValue.InvoiceItems {

			paymentItemMap, exist := paymentValueMap[invoiceItemsValue.InvoiceCode]
			if exist {
				paidSale += paymentItemMap
			}

		}
		conMapCustomer, exist := convertCustomerMap[resultValue.Sale.CustomerCode]
		if exist {
			resultValue.Sale.CustomerName = conMapCustomer.CustomerName
		}

		detail := OutStandingSoRes{
			ID:            resultValue.Sale.ID,
			CustomerCode:  resultValue.Sale.CustomerCode,
			CustomerName:  resultValue.Sale.CustomerName,
			SaleCode:      resultValue.Sale.SaleCode,
			SaleDate:      resultValue.Sale.DeliveryDate,
			SaleAmount:    resultValue.Sale.TotalAmount,
			Paid:          paidSale,
			OutStandingSo: resultValue.Sale.TotalAmount - paidSale,
			StatusPayment: resultValue.Sale.StatusPayment,
		}
		resultOutStandingSoRes = append(resultOutStandingSoRes, detail)
	}

	return resultOutStandingSoRes, nil
}
