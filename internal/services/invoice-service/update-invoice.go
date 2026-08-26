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

func UpdateInvoice(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Invoice

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}
	conUserID, _ := ctx.Get("user")
	userID := ""
	if conUserID != nil {
		userID = conUserID.(string)
	}
	invoiceValue := []models.Invoice{}
	invoiceItemValue := []models.InvoiceItem{}
	for i, invoice := range req {
		req[i].UpdateBy = userID
		for o := range invoice.InvoiceItem {
			invoiceItemID := uuid.New()
			req[i].InvoiceItem[o].ID = invoiceItemID
			req[i].InvoiceItem[o].InvoiceID = invoice.ID
			req[i].InvoiceItem[o].InvoiceItem = strconv.Itoa(i)
			invoiceItemValue = append(invoiceItemValue, req[i].InvoiceItem[o])
		}
		req[i].InvoiceItem = []models.InvoiceItem{}
		req[i].InvoiceDeposit = []models.InvoiceDeposit{}
		invoiceValue = append(invoiceValue, req[i])
	}
	invoiceId := []uuid.UUID{req[0].ID}
	errDeleteInvoiceItem := repositoryInvoice.DeleteInvoiceItem(invoiceId)
	if errDeleteInvoiceItem != nil {
		return nil, errDeleteInvoiceItem
	}
	errCreateApproval := repositoryInvoice.CreateInvoice([]models.Invoice{}, invoiceItemValue, []models.InvoiceDeposit{})
	if errCreateApproval != nil {
		return nil, errCreateApproval
	}

	rowsAffected, errCreateApproval := repositoryInvoice.UpdateInvoice(invoiceValue, []models.InvoiceItem{})
	if errCreateApproval != nil {
		return nil, errCreateApproval
	}

	if rowsAffected > 0 {
		return map[string]interface{}{
			"status":  "success",
			"message": "Approval updated successfully",
		}, nil
	} else {
		return map[string]interface{}{
			"status":  "success",
			"message": "Approval Not Have Rows Affected ",
		}, nil
	}
}
