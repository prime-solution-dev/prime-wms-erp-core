package saleService

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

type RequestApproveSaleRequest struct {
	ID uuid.UUID `json:"id"`
}

type RequestApproveSaleResponse struct {
	ApproveID uuid.UUID `json:"approve_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
}

func RequestApproveSale(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := RequestApproveSaleRequest{}
	res := RequestApproveSaleResponse{}
	var approvalIDs []uuid.UUID

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

	saleReq := GetSaleRequest{
		ID: []uuid.UUID{req.ID},
	}
	salePayload, _ := json.Marshal(saleReq)
	saleResult, err := GetSale(ctx, string(salePayload))
	if err != nil {
		return nil, err
	}

	// ตรวจสอบว่าได้ข้อมูล sale มาหรือไม่
	saleResponse, ok := saleResult.(ResultSale)
	if !ok || len(saleResponse.Sale) == 0 {
		return RequestApproveSaleResponse{
			Status:  "ERROR",
			Message: "Sale not found",
		}, nil
	}

	sale := saleResponse.Sale[0]

	saleJSON, err := json.Marshal(sale)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sale to JSON: %v", err)
	}

	approvalReq := approvalService.GetApprovalRequest{
		Page:         1,
		PageSize:     1,
		DocumentCode: []string{sale.SaleCode},
	}
	approvalPayload, _ := json.Marshal(approvalReq)
	approvalResult, err := approvalService.GetApproval(ctx, string(approvalPayload))
	if err != nil {
		return nil, err
	}
	approvalResponse, ok := approvalResult.(approvalService.ResultApproval)
	if !ok || len(approvalResponse.ApprovalRes) == 0 {
		createApprovalReq := []models.Approval{{
			ApproveTopic: "QPC ",
			DocumentType: "SO",
			DocumentCode: sale.SaleCode,
			Status:       "PENDING",
			Remark:       "",
			MDItemCode:   "CTM-CTM5",
			CreateBy:     "ADMIN",
			DocumentData: saleJSON,
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

		approvalIDs, ok = approvalResponse["id"].([]uuid.UUID)
		if !ok || len(approvalIDs) == 0 {
			return nil, fmt.Errorf("no approval ID returned")
		}
	} else {
		// If approval already exists, get its ID
		for _, approval := range approvalResponse.ApprovalRes {
			approvalIDs = append(approvalIDs, approval.ID)
		}
	}

	// Update sale status_approve to "PROCESS"
	saleID := req.ID

	if err := gormx.Model(&models.Sale{}).
		Where("id = ?", saleID).
		Update("status_approve", "PROCESS").Error; err != nil {
		return nil, fmt.Errorf("failed to update sale status: %v", err)
	}

	// Set response data
	res.ApproveID = approvalIDs[0] // Get first approval ID
	res.Status = "SUCCESS"
	res.Message = fmt.Sprintf("Approval request for sale %s created successfully", sale.SaleCode)

	return res, nil
}
