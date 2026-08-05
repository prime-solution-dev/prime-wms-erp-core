package creditService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryCredit "prime-erp-core/internal/repositories/credit"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CreditTransactionRequest struct {
	ID              []uuid.UUID `json:"id"`
	TransactionCode []string    `json:"transaction_code"`
	Status          []string    `json:"status"`
	Page            int         `json:"page"`
	PageSize        int         `json:"page_size"`
}
type ResultCreditTransaction struct {
	Total             int                        `json:"total"`
	Page              int                        `json:"page"`
	PageSize          int                        `json:"page_size"`
	TotalPages        int                        `json:"total_pages"`
	CreditTransaction []models.CreditTransaction `json:"credit_transaction"`
}

func GetTransaction(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req CreditTransactionRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	creditTransaction, totalPages, totalRecords, errApproval := repositoryCredit.GetCreditTransaction(req.ID, req.TransactionCode, req.Status, req.Page, req.PageSize)
	if errApproval != nil {
		return nil, errApproval
	}

	resultApproval := ResultCreditTransaction{
		Total:             totalRecords,
		Page:              req.Page,
		PageSize:          req.PageSize,
		TotalPages:        totalPages,
		CreditTransaction: creditTransaction,
	}

	return resultApproval, nil
}
