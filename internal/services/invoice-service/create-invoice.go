package invoiceService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryInvoice "prime-erp-core/internal/repositories/invoice"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreateInvoice(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Invoice

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}
	invoiceValue := []models.Invoice{}
	invoiceItemValue := []models.InvoiceItem{}
	invoiceIDForReturn := []uuid.UUID{}
	for i, invoice := range req {
		invoiceID := uuid.New()
		req[i].ID = invoiceID

		invoiceIDForReturn = append(invoiceIDForReturn, invoiceID)

		if req[i].InvoiceCode == "" {
			req[i].InvoiceCode = uuid.New().String()
		}

		for o := range invoice.InvoiceItem {
			invoiceItemID := uuid.New()
			req[i].InvoiceItem[o].ID = invoiceItemID
			req[i].InvoiceItem[o].InvoiceID = invoiceID
			req[i].InvoiceItem[o].InvoiceItem = strconv.Itoa(i)
			invoiceItemValue = append(invoiceItemValue, req[i].InvoiceItem[o])
		}

		req[i].InvoiceItem = []models.InvoiceItem{}
		req[i].InvoiceDeposit = []models.InvoiceDeposit{}
		invoiceValue = append(invoiceValue, req[i])
	}

	errCreateApproval := repositoryInvoice.CreateInvoice(invoiceValue, invoiceItemValue)
	if errCreateApproval != nil {
		return nil, errCreateApproval
	}

	return map[string]interface{}{
		"id":      invoiceIDForReturn,
		"status":  "success",
		"message": "Create Invoice Successfully",
	}, nil
}
