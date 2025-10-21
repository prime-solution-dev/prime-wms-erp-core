package deliveryService

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"

	"github.com/gin-gonic/gin"
)

type UpdateStatusDeliveryRequest struct {
	DeliveryCodes []string `json:"delivery_codes"`
	Status        string   `json:"status"`
}

type UpdateStatusDeliveryResponse struct {
	DeliveryCode string `json:"delivery_code"`
	Status       string `json:"status"`
	Message      string `json:"message"`
}

func UpdateStatusDelivery(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := UpdateStatusDeliveryRequest{}
	res := []UpdateStatusDeliveryResponse{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	// Validate request
	if len(req.DeliveryCodes) == 0 {
		return nil, errors.New("delivery_codes is required")
	}

	if req.Status == "" {
		req.Status = "COMPLETED" // Default status
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	user := "SYSTEM" // TODO: get from ctx
	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	tx := gormx.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update deliveries status
	for _, deliveryCode := range req.DeliveryCodes {
		// Update delivery
		result := tx.Model(&models.Delivery{}).
			Where("delivery_code = ?", deliveryCode).
			Updates(map[string]interface{}{
				"status":      req.Status,
				"update_date": nowDateOnly,
				"update_by":   user,
			})

		if result.Error != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to update delivery %s: %v", deliveryCode, result.Error)
		}

		if result.RowsAffected == 0 {
			tx.Rollback()
			return nil, fmt.Errorf("delivery with code %s not found", deliveryCode)
		}

		var delivery models.Delivery
		if err := gormx.Where("delivery_code = ?", deliveryCode).First(&delivery).Error; err != nil {
			gormx.Rollback()
			return nil, fmt.Errorf("delivery not found: %v", err)
		}

		// Update delivery items
		result = tx.Model(&models.DeliveryItem{}).
			Where("delivery_id = ?", delivery.ID).
			Updates(map[string]interface{}{
				"status":      req.Status,
				"update_date": nowDateOnly,
				"update_by":   user,
			})

		if result.Error != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to update delivery items for %s: %v", deliveryCode, result.Error)
		}

		res = append(res, UpdateStatusDeliveryResponse{
			DeliveryCode: deliveryCode,
			Status:       "success",
			Message:      fmt.Sprintf("Delivery and items updated to %s successfully", req.Status),
		})
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return res, nil
}
