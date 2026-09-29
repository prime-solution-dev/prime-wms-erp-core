package summaryService

import (
	"encoding/json"
	"errors"
	"prime-erp-core/internal/models"
	repositorySale "prime-erp-core/internal/repositories/sale"
	customerService "prime-erp-core/internal/services/customer-service"
	invoiceService "prime-erp-core/internal/services/invoice-service"
	paymentService "prime-erp-core/internal/services/payment-service"
	"slices"
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
	paymentValueMap := map[string]float64{}
	resultInvoiceDepositMap := map[string]float64{}
	resultInvoiceMap := map[string][]models.Invoice{}
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
			return nil, errGetInvoiceDeposit
		}
		resultInvoiceDeposit := invoiceDeposit.(invoiceService.ResultInvoice).Invoice

		for _, resultInvoiceDepositValue := range resultInvoiceDeposit {
			for _, invoiceItem := range resultInvoiceDepositValue.InvoiceItem {
				if invoiceItem.ArticleType == "DEPOSIT" {
					resultInvoiceDepositMap[resultInvoiceDepositValue.InvoiceCode] += invoiceItem.TotalAmount
				}
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
			paymentItemMapDeposit, existDeposit := resultInvoiceDepositMap[invoiceItemsValue.InvoiceCode]
			if existDeposit {
				paidSale += paymentItemMapDeposit
			}
			invoiceItemMap, existResultInvoiceMap := resultInvoiceMap[invoiceItemsValue.InvoiceCode]
			if existResultInvoiceMap {
				for _, invoiceItemMapValue := range invoiceItemMap {
					if invoiceItemMapValue.InvoiceType == "DN" {
						paymentItemMap, exist := paymentValueMap[invoiceItemMapValue.InvoiceCode]
						if exist {
							paidSale += paymentItemMap
						}
					}
				}
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
		if paidSale != 0 {
			resultOutStandingSoRes = append(resultOutStandingSoRes, detail)
		}
	}

	return resultOutStandingSoRes, nil
}
