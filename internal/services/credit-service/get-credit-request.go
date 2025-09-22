package creditService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryCredit "prime-erp-core/internal/repositories/credit"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetCreditReq struct {
	ID           []uuid.UUID `json:"id"`
	CustomerCode []string    `json:"customer_code"`
	Page         int         `json:"page"`
	PageSize     int         `json:"page_size"`
}
type ResultCreditRequest struct {
	Total         int                    `json:"total"`
	Page          int                    `json:"page"`
	PageSize      int                    `json:"page_size"`
	TotalPages    int                    `json:"total_pages"`
	CreditRequest []models.CreditRequest `json:"credit_request"`
}

func GetCreditRequests(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetCreditReq

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	creditRequest, totalPages, totalRecords, errApproval := repositoryCredit.GetCreditRequestPreload(req.ID, req.CustomerCode, req.Page, req.PageSize)
	if errApproval != nil {
		return nil, errApproval
	}

	resultApproval := ResultCreditRequest{
		Total:         totalRecords,
		Page:          req.Page,
		PageSize:      req.PageSize,
		TotalPages:    totalPages,
		CreditRequest: creditRequest,
	}

	return resultApproval, nil
}
