package invoiceService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryInvoice "prime-erp-core/internal/repositories/invoice"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func UpdateInvoiceCN(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Invoice

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	if len(req) == 0 {
		return nil, errors.New("invoice is required")
	}
	ids := make([]uuid.UUID, 0, len(req))
	seen := make(map[uuid.UUID]bool, len(req))
	for _, invoice := range req {
		if invoice.ID == uuid.Nil || seen[invoice.ID] {
			return nil, errors.New("invoice IDs must be non-empty and unique")
		}
		seen[invoice.ID] = true
		ids = append(ids, invoice.ID)
	}
	getPayload, err := json.Marshal(GetInvoiceRequest{ID: ids})
	if err != nil {
		return nil, err
	}
	value, err := GetInvoice(ctx, string(getPayload))
	if err != nil {
		return nil, err
	}
	stored, ok := value.(ResultInvoice)
	if !ok {
		return nil, errors.New("invalid GetInvoice response")
	}
	statuses := make(map[uuid.UUID]string, len(stored.Invoice))
	for _, invoice := range stored.Invoice {
		statuses[invoice.ID] = invoice.Status
	}
	tempIDs := make([]uuid.UUID, 0, len(req))
	for _, invoice := range req {
		status, exists := statuses[invoice.ID]
		if !exists {
			return nil, errors.New("invoice not found: " + invoice.ID.String())
		}
		if strings.EqualFold(status, "TEMP") {
			tempIDs = append(tempIDs, invoice.ID)
		}
	}
	if len(tempIDs) == len(req) {
		if err := repositoryInvoice.DeleteInvoice(tempIDs); err != nil {
			return nil, err
		}
		return CreateInvoiceCN(ctx, jsonPayload)
	}
	if len(tempIDs) > 0 {
		// Mixed batches recreate only the invoices whose stored status is TEMP.
		results := make([]interface{}, 0, len(req))
		for _, invoice := range req {
			payload, err := json.Marshal([]models.Invoice{invoice})
			if err != nil {
				return nil, err
			}
			var result interface{}
			if strings.EqualFold(statuses[invoice.ID], "TEMP") {
				if err := repositoryInvoice.DeleteInvoice([]uuid.UUID{invoice.ID}); err != nil {
					return nil, err
				}
				result, err = CreateInvoiceCN(ctx, string(payload))
			} else {
				result, err = UpdateInvoice(ctx, string(payload))
			}
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}
		return results, nil
	}
	createInvoiceReturn, errCreateInvoice := UpdateInvoice(ctx, jsonPayload)
	if errCreateInvoice != nil {
		return nil, errCreateInvoice
	}
	return createInvoiceReturn, nil

}
