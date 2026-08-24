package deliveryService

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	orderExternalService "prime-erp-core/external/order-service"
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

	user := ctx.GetString("user")
	if user == "" {
		user = `system` // fallback
	}
	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// อ่านใบทั้งหมดก่อน จะได้ไม่ต้องอ่านคนละ connection ระหว่างที่ tx เปิดอยู่
	var deliveries []models.Delivery
	if err := gormx.Where("delivery_code IN ?", req.DeliveryCodes).Find(&deliveries).Error; err != nil {
		return nil, fmt.Errorf("failed to load deliveries: %v", err)
	}

	deliveryOf := map[string]models.Delivery{}
	for _, delivery := range deliveries {
		deliveryOf[delivery.DeliveryCode] = delivery
	}

	for _, deliveryCode := range req.DeliveryCodes {
		if _, found := deliveryOf[deliveryCode]; !found {
			return nil, fmt.Errorf("delivery with code %s not found", deliveryCode)
		}
	}

	// ยกเลิกฝั่ง WMS ให้ครบก่อนเริ่ม transaction
	//
	// เดิมเรียกอยู่ข้างในลูปที่อยู่ใน tx ยกเลิกหลายใบแล้วใบท้ายๆ พัง จะ rollback ฝั่ง ERP
	// แต่ใบแรกๆ ถูกยกเลิกที่ WMS ไปแล้วและไม่มีอะไรย้อนคืน ย้ายมาไว้ก่อนเปิด tx
	// ทำให้พังตรงไหนก็ตาม ERP ยังไม่ถูกแตะเลย ผู้ใช้กดยกเลิกซ้ำได้
	if req.Status == "CANCELED" {
		for _, deliveryCode := range req.DeliveryCodes {
			if _, err := CancelOrder(deliveryOf[deliveryCode]); err != nil {
				return nil, fmt.Errorf("failed to cancel order for delivery %s: %v", deliveryCode, err)
			}
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

	for _, deliveryCode := range req.DeliveryCodes {
		delivery := deliveryOf[deliveryCode]

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
		tx.Rollback()
		return nil, err
	}

	return res, nil
}

func CancelOrder(delivery models.Delivery) (orderExternalService.CancelOrderResponse, error) {
	cancelOrderRequest := orderExternalService.CancelOrderRequest{
		DocumentRef: []string{delivery.DeliveryCode},
	}

	fmt.Println("cancelOrderRequest : ", cancelOrderRequest)
	cancelOrderResponse, err := orderExternalService.CancelOrder(cancelOrderRequest)
	if err != nil {
		return orderExternalService.CancelOrderResponse{}, errors.New("Error cancel order : " + err.Error())
	}
	fmt.Println("cancelOrderResponse : ", cancelOrderResponse)

	return cancelOrderResponse, nil
}
