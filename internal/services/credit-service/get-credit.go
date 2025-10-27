package creditService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryCredit "prime-erp-core/internal/repositories/credit"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetApprovalRequest struct {
	ID           []uuid.UUID `json:"id"`
	CustomerCode []string    `json:"customer_code"`
	Status       []string    `json:"status"`
	Page         int         `json:"page"`
	PageSize     int         `json:"page_size"`
}
type ResultCredit struct {
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
	Credit     []models.Credit `json:"approval"`
}

func GetCredit(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetApprovalRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	approval, totalPages, totalRecords, errApproval := repositoryCredit.GetCreditPreload(req.ID, req.CustomerCode, req.Status, req.Page, req.PageSize)
	if errApproval != nil {
		return nil, errApproval
	}

	resultApproval := ResultCredit{
		Total:      totalRecords,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		Credit:     approval,
	}

	return resultApproval, nil
}
