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

	// เดิมไม่เช็คสถานะเดิมเลย สั่งซ้ำกี่รอบก็ยิง WMS ซ้ำทุกรอบ
	//
	// ใบที่อยู่สถานะปลายทางอยู่แล้วให้ "ข้าม" ไม่ใช่ทำให้ทั้ง request พัง เพราะ endpoint นี้
	// รับได้หลายใบต่อครั้งและ status ว่างจะ default เป็น COMPLETED ซึ่งเป็นรูปแบบของ callback
	// ที่ยิงซ้ำได้ ใบเดียวที่ซ้ำจึงไม่ควรทำให้อีกเก้าใบไม่ถูกอัปเดต
	toUpdate := []string{}
	for _, deliveryCode := range req.DeliveryCodes {
		delivery, found := deliveryOf[deliveryCode]
		if !found {
			return nil, fmt.Errorf("delivery with code %s not found", deliveryCode)
		}

		if delivery.Status == req.Status {
			res = append(res, UpdateStatusDeliveryResponse{
				DeliveryCode: deliveryCode,
				Status:       "success",
				Message:      fmt.Sprintf("Delivery is already %s", req.Status),
			})
			continue
		}

		// ยกเลิกไปแล้วย้อนกลับไม่ได้
		if delivery.Status == "CANCELED" {
			return nil, fmt.Errorf("delivery %s is already canceled", deliveryCode)
		}

		toUpdate = append(toUpdate, deliveryCode)
	}

	if len(toUpdate) == 0 {
		return res, nil
	}

	// ห้ามยกเลิกใบที่คลังหยิบไปทำงานแล้ว หน้าจอปิดปุ่มด้วย isCreateOutbound อยู่แล้ว
	// แต่ฝั่ง server ไม่เคยบังคับ ยิง API ตรงหรือแข่งจังหวะกันก็ผ่าน
	if req.Status == "CANCELED" {
		started, err := deliveriesWithOutbound(toUpdate)
		if err != nil {
			return nil, err
		}

		for _, deliveryCode := range toUpdate {
			if started[deliveryCode] {
				return nil, fmt.Errorf(
					"delivery %s is already being processed in the warehouse and cannot be canceled", deliveryCode)
			}
		}
	}

	// ยกเลิกฝั่ง WMS ให้ครบก่อนเริ่ม transaction
	//
	// เดิมเรียกอยู่ข้างในลูปที่อยู่ใน tx ยกเลิกหลายใบแล้วใบท้ายๆ พัง จะ rollback ฝั่ง ERP
	// แต่ใบแรกๆ ถูกยกเลิกที่ WMS ไปแล้วและไม่มีอะไรย้อนคืน ย้ายมาไว้ก่อนเปิด tx
	// ทำให้พังตรงไหนก็ตาม ERP ยังไม่ถูกแตะเลย ผู้ใช้กดยกเลิกซ้ำได้
	if req.Status == "CANCELED" {
		for _, deliveryCode := range toUpdate {
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

	for _, deliveryCode := range toUpdate {
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

// deliveriesWithOutbound ถาม WMS ว่าใบไหนถูกสร้าง outbound ไปแล้วบ้าง
// ใช้กันไม่ให้ยกเลิกใบที่คลังเริ่มทำงานไปแล้ว
func deliveriesWithOutbound(deliveryCodes []string) (map[string]bool, error) {
	started := map[string]bool{}

	orderRes, err := orderExternalService.GetOrdersDelivery(orderExternalService.GetOrderDeliveryRequest{
		DeliveryCode: deliveryCodes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read warehouse progress before cancelling: %v", err)
	}

	for _, order := range orderRes.Orders {
		for _, orderItem := range order.OrderItem {
			if len(orderItem.OutboundItem) > 0 {
				started[order.DocumentRef] = true
				break
			}
		}
	}

	return started, nil
}
