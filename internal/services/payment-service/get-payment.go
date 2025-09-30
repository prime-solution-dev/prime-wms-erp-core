package paymentService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryPayment "prime-erp-core/internal/repositories/payment"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetPaymentRequest struct {
	ID           []uuid.UUID `json:"id"`
	CustomerCode []string    `json:"customer_code"`
	Status       []string    `json:"status"`
	InvoiceCode  []string    `json:"invoice_code"`
	Page         int         `json:"page"`
	PageSize     int         `json:"page_size"`
}
type ResultPayment struct {
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
	Payment    []models.Payment `json:"payment"`
}

func GetPayment(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetPaymentRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	payment, totalPages, totalRecords, errPayment := repositoryPayment.GetPaymentPreload(req.ID, req.CustomerCode, req.Status, req.InvoiceCode, req.Page, req.PageSize)
	if errPayment != nil {
		return nil, errPayment
	}

	resultSale := ResultPayment{
		Total:      totalRecords,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		Payment:    payment,
	}

	return resultSale, nil
}
