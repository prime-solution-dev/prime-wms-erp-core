package saleService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositorySale "prime-erp-core/internal/repositories/sale"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetSaleRequest struct {
	ID            []uuid.UUID `json:"id"`
	SaleCode      []string    `json:"sale_code"`
	CustomerCode  []string    `json:"customer_code"`
	Status        []string    `json:"status"`
	StatusPayment []string    `json:"status_payment"`
	IsApproved    []bool      `json:"is_approved"`
	Page          int         `json:"page"`
	PageSize      int         `json:"page_size"`
}
type ResultSale struct {
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
	Sale       []models.Sale `json:"sale"`
}

func GetSale(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetSaleRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	sale, totalPages, totalRecords, errApproval := repositorySale.GetSalePreload(req.ID, req.SaleCode, req.CustomerCode, req.Status, req.StatusPayment, req.IsApproved, req.Page, req.PageSize)
	if errApproval != nil {
		return nil, errApproval
	}

	resultSale := ResultSale{
		Total:      totalRecords,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		Sale:       sale,
	}

	return resultSale, nil
}
