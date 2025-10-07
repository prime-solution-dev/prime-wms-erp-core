package invoiceService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryInvoice "prime-erp-core/internal/repositories/invoice"

	"github.com/gin-gonic/gin"
)

func UpdateInvoice(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Invoice

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}
	invoiceValue := []models.Invoice{}
	invoiceItemValue := []models.InvoiceItem{}
	for i, invoice := range req {
		req[i].InvoiceItem = []models.InvoiceItem{}
		invoiceValue = append(invoiceValue, req[i])
		for i := range invoice.InvoiceItem {
			invoiceItemValue = append(invoiceItemValue, req[i].InvoiceItem[i])
		}
	}

	rowsAffected, errCreateApproval := repositoryInvoice.UpdateInvoice(invoiceValue, invoiceItemValue)
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
