package summaryService

import (
	"encoding/json"
	"errors"
	"prime-erp-core/internal/models"
	invoiceService "prime-erp-core/internal/services/invoice-service"
	saleService "prime-erp-core/internal/services/sale-service"

	"github.com/gin-gonic/gin"
)

type ConsumedCreditDetail struct {
	Sale    models.Sale      `json:"sale"`
	Invoice []models.Invoice `json:"invoice"`
}

func GetConsumend(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetPaidInvoiceRequest

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
		for _, invoice := range resultInvoice {
			if invoice.DocRef == sale.SaleCode {
				matchedInvoices = append(matchedInvoices, invoice)
			}
		}

		detail := ConsumedCreditDetail{
			Sale:    sale,
			Invoice: matchedInvoices,
		}

		resultConsumend = append(resultConsumend, detail)
	}

	return resultConsumend, nil
}
