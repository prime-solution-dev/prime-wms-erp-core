package deliveryService

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

type UpdateDeliveryRequest struct {
	Deliveries []DeliveryDocumentUpdate `json:"deliveries"`
}

type DeliveryDocumentUpdate struct {
	models.Delivery
	Items       []models.DeliveryItem `json:"items"`        // Items to update
	DeleteItems []uuid.UUID           `json:"delete_items"` // Item IDs to delete
}

type UpdateDeliveryResponse struct {
	DeliveryCode string `json:"delivery_code"`
	Status       string `json:"status"`
	Message      string `json:"message"`
}

func UpdateDelivery(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := UpdateDeliveryRequest{}
	res := []UpdateDeliveryResponse{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	user := "SYSTEM" // TODO: get from ctx
	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	updateDeliveries := []models.Delivery{}
	updateDeliveryItems := []models.DeliveryItem{}

	for _, deliveryReq := range req.Deliveries {
		tempDelivery := deliveryReq.Delivery

		if tempDelivery.ID == uuid.Nil {
			return nil, fmt.Errorf("delivery ID is required for update")
		}

		if tempDelivery.DeliveryCode == "" {
			return nil, fmt.Errorf("delivery code is required for update")
		}

		// Convert date fields to date-only format
		if tempDelivery.DeliveryDate != nil {
			deliveryDateOnly := time.Date(tempDelivery.DeliveryDate.Year(), tempDelivery.DeliveryDate.Month(), tempDelivery.DeliveryDate.Day(), 0, 0, 0, 0, tempDelivery.DeliveryDate.Location())
			tempDelivery.DeliveryDate = &deliveryDateOnly
		}

		// Only update timestamp and user for update
		tempDelivery.UpdateDate = nowDateOnly
		tempDelivery.UpdateBy = user
		tempDelivery.Status = "PENDING"

		updateDeliveries = append(updateDeliveries, tempDelivery)

		for _, item := range deliveryReq.Items {
			// Ensure item belongs to this delivery
			item.DeliveryID = tempDelivery.ID
			item.UpdateDate = nowDateOnly
			item.UpdateBy = user
			item.Status = "PENDING"

			updateDeliveryItems = append(updateDeliveryItems, item)
		}
	}

	tx := gormx.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update deliveries
	for _, delivery := range updateDeliveries {
		if err := tx.Model(&models.Delivery{}).
			Where("id = ?", delivery.ID).
			Updates(delivery).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to update delivery %s: %v", delivery.DeliveryCode, err)
		}

		res = append(res, UpdateDeliveryResponse{
			DeliveryCode: delivery.DeliveryCode,
			Status:       "success",
			Message:      "Delivery updated successfully",
		})
	}

	// Delete items if specified
	for _, deliveryReq := range req.Deliveries {
		if len(deliveryReq.DeleteItems) > 0 {
			if err := tx.Where("id IN ? AND delivery_id = ?", deliveryReq.DeleteItems, deliveryReq.Delivery.ID).Delete(&models.DeliveryItem{}).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to delete delivery items: %v", err)
			}
		}
	}

	// Update existing items
	for _, item := range updateDeliveryItems {
		if err := tx.Model(&models.DeliveryItem{}).
			Where("id = ? AND delivery_id = ?", item.ID, item.DeliveryID).
			Updates(item).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to update delivery item %s: %v", item.DeliveryItem, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return res, nil
}
