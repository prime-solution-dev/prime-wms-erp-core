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

type UpdateSaleItemStatusRequest struct {
	SaleItem []string `json:"sale_item"`
	Status   string   `json:"status"`
}

type UpdateSaleItemStatusResponse struct {
	Status         string   `json:"status"`
	Message        string   `json:"message"`
	UpdatedItems   []string `json:"updated_items"`
	UpdatedSales   []string `json:"updated_sales,omitempty"`
	CompletedSales []string `json:"completed_sales,omitempty"`
}

func UpdateSaleItemStatus(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := UpdateSaleItemStatusRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	if len(req.SaleItem) == 0 {
		return nil, fmt.Errorf("sale items are required")
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

	// Start transaction
	tx := gormx.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update sale items status
	updateFields := map[string]interface{}{
		"status":      req.Status,
		"update_date": nowDateOnly,
		"update_by":   user,
	}

	if err := tx.Model(&models.SaleItem{}).
		Where("sale_item IN ?", req.SaleItem).
		Updates(updateFields).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update sale items status: %v", err)
	}

	// Get all affected sale IDs
	var affectedSaleIDs []uuid.UUID
	if err := tx.Model(&models.SaleItem{}).
		Where("sale_item IN ?", req.SaleItem).
		Distinct("sale_id").
		Pluck("sale_id", &affectedSaleIDs).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to get affected sale IDs: %v", err)
	}

	var completedSaleCodes []string
	var updatedSaleCodes []string

	// Check each affected sale
	for _, saleID := range affectedSaleIDs {
		// Get all sale items for this sale
		var saleItems []models.SaleItem
		if err := tx.Where("sale_id = ?", saleID).Find(&saleItems).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to get sale items for sale ID %v: %v", saleID, err)
		}

		// Check if all sale items are completed
		allCompleted := true
		for _, item := range saleItems {
			if item.Status != "COMPLETED" {
				allCompleted = false
				break
			}
		}

		// If all items are completed, update the parent sale to completed
		if allCompleted {
			var sale models.Sale
			if err := tx.Where("id = ?", saleID).First(&sale).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to get sale for ID %v: %v", saleID, err)
			}

			// Update sale status to completed
			if err := tx.Model(&models.Sale{}).
				Where("id = ?", saleID).
				Updates(map[string]interface{}{
					"status":      "COMPLETED",
					"update_date": nowDateOnly,
					"update_by":   user,
				}).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to update sale status to completed for sale ID %v: %v", saleID, err)
			}

			completedSaleCodes = append(completedSaleCodes, sale.SaleCode)
		}

		// Get sale code for response
		var sale models.Sale
		if err := tx.Where("id = ?", saleID).First(&sale).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to get sale code for ID %v: %v", saleID, err)
		}
		updatedSaleCodes = append(updatedSaleCodes, sale.SaleCode)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	response := UpdateSaleItemStatusResponse{
		Status:       "success",
		Message:      fmt.Sprintf("Updated %d sale items status to %s", len(req.SaleItem), req.Status),
		UpdatedItems: req.SaleItem,
		UpdatedSales: updatedSaleCodes,
	}

	if len(completedSaleCodes) > 0 {
		response.CompletedSales = completedSaleCodes
		response.Message += fmt.Sprintf(", and completed %d sales", len(completedSaleCodes))
	}

	return response, nil
}
