package invoiceService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryInvoice "prime-erp-core/internal/repositories/invoice"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetInvoiceRequest struct {
	ID           []uuid.UUID `json:"id"`
	InvoiceCode  []string    `json:"invoice_code"`
	CustomerCode []string    `json:"customer_code"`
	Status       []string    `json:"status"`
	DocRef       []string    `json:"doc_ref"`
	Page         int         `json:"page"`
	PageSize     int         `json:"page_size"`
}
type resultInvoice struct {
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
	Invoice    []models.Invoice `json:"invoice"`
}

func GetInvoice(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetInvoiceRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	invoice, totalPages, totalRecords, errDeposit := repositoryInvoice.GetInvoicePreload(req.ID, req.InvoiceCode, req.CustomerCode, req.Status, req.DocRef, req.Page, req.PageSize)
	if errDeposit != nil {
		return nil, errDeposit
	}

	resultInvoice := resultInvoice{
		Total:      totalRecords,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		Invoice:    invoice,
	}

	return resultInvoice, nil
}
