package saleService

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UpdateStatusSaleRequest struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
}

type UpdateStatusSaleResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func UpdateStatusSale(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := UpdateStatusSaleRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	if req.ID == uuid.Nil {
		return nil, fmt.Errorf("sale ID is required")
	}

	if req.Status == "" {
		return nil, fmt.Errorf("status is required")
	}

	gormx, err := db.ConnectGORM(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	user := `system` // TODO: get from ctx
	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	updateFields := map[string]interface{}{
		"status":      req.Status,
		"update_date": nowDateOnly,
		"update_by":   user,
	}

	if err := gormx.Model(&models.Sale{}).
		Where("id = ?", req.ID).
		Updates(updateFields).Error; err != nil {
		return nil, fmt.Errorf("failed to update sale status: %v", err)
	}

	return UpdateStatusSaleResponse{
		Status:  "success",
		Message: "Sale status updated successfully",
	}, nil
}
