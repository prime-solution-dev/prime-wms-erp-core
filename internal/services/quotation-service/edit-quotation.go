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

type EditQuotationRequest struct {
	QuotationCode string `json:"quotation_code"`
}

func EditQuotation(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := EditQuotationRequest{}

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

	approvalReq := approvalService.GetApprovalRequest{
		Page:         1,
		PageSize:     1,
		DocumentCode: []string{req.QuotationCode},
	}
	approvalPayload, _ := json.Marshal(approvalReq)
	approvalResult, err := approvalService.GetApproval(ctx, string(approvalPayload))
	if err != nil {
		return nil, err
	}

	approvalResponse, ok := approvalResult.(approvalService.ResultApproval)
	var approvalUpdateResult interface{} = nil
	if ok || len(approvalResponse.ApprovalRes) > 0 {
		updateApprovalReq := []struct {
			ID     uuid.UUID `json:"id"`
			Status string    `json:"status"`
			Remark string    `json:"remark"`
		}{
			{
				ID:     approvalResponse.ApprovalRes[0].ID,
				Status: "PENDING",
				Remark: "",
			},
		}

		updateApprovalPayload, _ := json.Marshal(updateApprovalReq)
		approvalUpdateResultTmp, err := approvalService.UpdateApproval(ctx, string(updateApprovalPayload))
		if err != nil {
			return nil, fmt.Errorf("failed to update approval: %v", err)
		}
		approvalUpdateResult = approvalUpdateResultTmp
	}

	if err := gormx.Model(&models.Quotation{}).
		Where("quotation_code = ?", req.QuotationCode).
		Updates(map[string]interface{}{
			"status":         "PENDING",
			"status_approve": "PENDING",
			"is_approved":    false,
			"update_date":    gormx.NowFunc(),
		}).Error; err != nil {
		return nil, fmt.Errorf("failed to update quotation status: %v", err)
	}

	return map[string]interface{}{
		"external_response": approvalUpdateResult,
		"status":            "success",
		"message":           "Edit updated successfully",
	}, nil
}
