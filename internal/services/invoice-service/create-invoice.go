package invoiceService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryInvoice "prime-erp-core/internal/repositories/invoice"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreateInvoice(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Invoice

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}
	invoiceValue := []models.Invoice{}
	invoiceIDForReturn := []uuid.UUID{}
	for i := range req {
		creditID := uuid.New()
		req[i].ID = creditID

		invoiceIDForReturn = append(invoiceIDForReturn, creditID)

		if req[i].InvoiceCode == "" {
			req[i].InvoiceCode = uuid.New().String()
		}

		invoiceValue = append(invoiceValue, req[i])

	}

	errCreateApproval := repositoryInvoice.CreateInvoice(invoiceValue)
	if errCreateApproval != nil {
		return nil, errCreateApproval
	}

	return map[string]interface{}{
		"id":      invoiceIDForReturn,
		"status":  "success",
		"message": "Approval create Transaction successfully",
	}, nil
}
