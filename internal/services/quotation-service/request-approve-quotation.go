package quotationService

import (
	"encoding/json"
	"errors"
	"fmt"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	approvalService "prime-erp-core/internal/services/approval-service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RequestApproveQuotationRequest struct {
	ID string `json:"id"`
}

type RequestApproveQuotationResponse struct {
	ApproveID uuid.UUID `json:"approve_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
}

func RequestApproveQuotation(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := RequestApproveQuotationRequest{}
	res := RequestApproveQuotationResponse{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	sqlx, err := db.ConnectSqlx(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer sqlx.Close()

	gormx, err := db.ConnectGORM(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	quotationReq := GetQuotationRequest{
		ID: []string{req.ID},
	}
	quotationPayload, _ := json.Marshal(quotationReq)
	quotationResult, err := GetQuotation(ctx, string(quotationPayload))
	if err != nil {
		return nil, err
	}

	// ตรวจสอบว่าได้ข้อมูล quotation มาหรือไม่
	quotationResponse, ok := quotationResult.(ResultQuotationResponse)
	if !ok || len(quotationResponse.Quotations) == 0 {
		return RequestApproveQuotationResponse{
			Status:  "ERROR",
			Message: "Quotation not found",
		}, nil
	}

	quotation := quotationResponse.Quotations[0]

	quotationJSON, err := json.Marshal(quotation)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal quotation to JSON: %v", err)
	}

	createApprovalReq := []models.Approval{{
		ApproveTopic: "QO",
		DocumentType: "QO",
		DocumentCode: quotation.QuotationCode,
		Status:       "PENDING",
		Remark:       "",
		MDItemCode:   "CTM-CTM4",
		CreateBy:     "ADMIN",
		DocumentData: quotationJSON,
	}}

	approvalPayload, _ := json.Marshal(createApprovalReq)
	approvalResult, err := approvalService.CreateApproval(ctx, string(approvalPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create approval: %v", err)
	}

	// Extract approval ID from the response
	approvalResponse, ok := approvalResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected approval response format")
	}

	approvalIDs, ok := approvalResponse["id"].([]uuid.UUID)
	if !ok || len(approvalIDs) == 0 {
		return nil, fmt.Errorf("no approval ID returned")
	}

	// Update quotation status_approve to "PROCESS"
	quotationID, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid quotation ID format: %v", err)
	}

	if err := gormx.Model(&models.Quotation{}).
		Where("id = ?", quotationID).
		Update("status_approve", "PROCESS").Error; err != nil {
		return nil, fmt.Errorf("failed to update quotation status: %v", err)
	}

	// Set response data
	res.ApproveID = approvalIDs[0] // Get first approval ID
	res.Status = "SUCCESS"
	res.Message = fmt.Sprintf("Approval request for quotation %s created successfully", quotation.QuotationCode)

	return res, nil
}
