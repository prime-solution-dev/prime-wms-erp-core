package depositService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryDeposit "prime-erp-core/internal/repositories/deposit"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetDepositRequest struct {
	ID             []uuid.UUID `json:"id"`
	DepositCode    []string    `json:"deposit_code"`
	NotDepositCode []string    `json:"not_deposit_code"`
	CustomerCode   []string    `json:"customer_code"`
	Status         []string    `json:"status"`
	Page           int         `json:"page"`
	PageSize       int         `json:"page_size"`
}
type ResultDeposit struct {
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
	Deposit    []models.Deposit `json:"deposit"`
}

func GetDeposit(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetDepositRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	deposit, totalPages, totalRecords, errDeposit := repositoryDeposit.GetDepositPreload(req.ID, req.CustomerCode, req.Status, req.DepositCode, req.Page, req.PageSize, req.NotDepositCode)
	if errDeposit != nil {
		return nil, errDeposit
	}

	resultSale := ResultDeposit{
		Total:      totalRecords,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		Deposit:    deposit,
	}

	return resultSale, nil
}
