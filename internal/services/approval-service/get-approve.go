package approvalService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryApproval "prime-erp-core/internal/repositories/approval"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetApprovalRequest struct {
	ID           []uuid.UUID `json:"id"`
	ApproveCode  []string    `json:"approval_code"`
	Status       []string    `json:"status"`
	DocumentCode []string    `json:"document_code"`
	Page         int         `json:"page"`
	PageSize     int         `json:"page_size"`
}
type ResultApproval struct {
	Total       int               `json:"total"`
	Page        int               `json:"page"`
	PageSize    int               `json:"page_size"`
	TotalPages  int               `json:"total_pages"`
	ApprovalRes []models.Approval `json:"approval"`
}

func GetApproval(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetApprovalRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	approval, totalPages, totalRecords, errApproval := repositoryApproval.GetApprovalPreload(req.ID, req.ApproveCode, req.Status, req.DocumentCode, req.Page, req.PageSize)
	if errApproval != nil {
		return nil, errApproval
	}

	resultApproval := ResultApproval{
		Total:       totalRecords,
		Page:        req.Page,
		PageSize:    req.PageSize,
		TotalPages:  totalPages,
		ApprovalRes: approval,
	}

	return resultApproval, nil
}
