package invoiceService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"

	"github.com/gin-gonic/gin"
)

func UpdateInvoiceAR(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Invoice

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	createInvoiceReturn, errCreateInvoice := UpdateInvoice(ctx, jsonPayload)
	if errCreateInvoice != nil {
		return nil, errCreateInvoice
	}
	return createInvoiceReturn, nil

}
