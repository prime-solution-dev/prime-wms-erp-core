package deliveryService

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	orderExternalService "prime-erp-core/external/order-service"
	"prime-erp-core/internal/apperr"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"prime-erp-core/internal/requestcontext"

	"gorm.io/gorm"
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

func UpdateStatusDelivery(ctx context.Context, jsonPayload string) (interface{}, error) {
	req := UpdateStatusDeliveryRequest{}
	res := []UpdateStatusDeliveryResponse{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	// Validate request
	if len(req.DeliveryCodes) == 0 {
		return nil, apperr.BadRequest("delivery_codes is required")
	}

	if req.Status == "" {
		req.Status = "COMPLETED" // Default status
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	user := requestcontext.GetUserOrDefault(ctx)
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

	toUpdate, alreadyAtStatus, err := partitionDeliveriesByStatus(req.DeliveryCodes, deliveryOf, req.Status)
	if err != nil {
		return nil, err
	}

	for _, deliveryCode := range alreadyAtStatus {
		res = append(res, UpdateStatusDeliveryResponse{
			DeliveryCode: deliveryCode,
			Status:       "success",
			Message:      fmt.Sprintf("Delivery is already %s", req.Status),
		})
	}

	if len(toUpdate) == 0 {
		// ไม่มีใบไหนต้องอัปเดต แต่ยังต้องลองปิด SO ของใบที่อยู่ COMPLETED อยู่แล้ว
		// (hook ตัวที่สองของ outbound เดียวกันมาถึงตรงนี้ และเป็นรอบที่ข้อมูลครบ)
		if req.Status == "COMPLETED" {
			closeSalesOfDeliveries(ctx, gormx, deliveryOf, alreadyAtStatus, user)
		}

		return res, nil
	}

	// ห้ามยกเลิกใบที่คลังหยิบไปทำงานแล้ว หน้าจอปิดปุ่มด้วย isCreateOutbound อยู่แล้ว
	// แต่ฝั่ง server ไม่เคยบังคับ ยิง API ตรงหรือแข่งจังหวะกันก็ผ่าน
	if req.Status == "CANCELED" {
		started, err := deliveriesWithOutbound(ctx, toUpdate)
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
			if _, err := CancelOrder(ctx, deliveryOf[deliveryCode]); err != nil {
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

	// ใบจองที่ปิดแล้ว แปลว่าของออกไปแล้ว ให้ไปดูว่า SO ต้นทางส่งครบหรือยัง
	//
	// นับทั้งใบที่รอบนี้เพิ่งพลิกและใบที่อยู่ COMPLETED มาก่อนแล้ว การปิดเป็น idempotent
	// รันซ้ำได้และนั่นคือเจตนา (ดูเหตุผลที่ partitionDeliveriesByStatus)
	if req.Status == "COMPLETED" {
		completed := make([]string, 0, len(toUpdate)+len(alreadyAtStatus))
		completed = append(completed, toUpdate...)
		completed = append(completed, alreadyAtStatus...)

		closeSalesOfDeliveries(ctx, gormx, deliveryOf, completed, user)
	}

	return res, nil
}

// partitionDeliveriesByStatus แยกใบที่ต้องอัปเดตจริง ออกจากใบที่อยู่สถานะปลายทางอยู่แล้ว
//
// เดิมไม่เช็คสถานะเดิมเลย สั่งซ้ำกี่รอบก็ยิง WMS ซ้ำทุกรอบ
//
// ใบที่อยู่สถานะปลายทางอยู่แล้วให้ "ข้าม" ไม่ใช่ทำให้ทั้ง request พัง เพราะ endpoint นี้
// รับได้หลายใบต่อครั้งและ status ว่างจะ default เป็น COMPLETED ซึ่งเป็นรูปแบบของ callback
// ที่ยิงซ้ำได้ ใบเดียวที่ซ้ำจึงไม่ควรทำให้อีกเก้าใบไม่ถูกอัปเดต
//
// คืน alreadyAtStatus แยกออกมาด้วย เพราะเส้นปิด SO ต้องนับใบพวกนี้ด้วย:
// outbound ใบเดียวยืนยันได้หลาย CO และแต่ละ CO ยิง hook ของตัวเอง hook ตัวแรกพลิก DBS
// เป็น COMPLETED แล้วรันปิด SO ตอนที่บรรทัดของ CO ตัวที่สองยังเปิดอยู่ พอ hook ตัวที่สอง
// มาถึง DBS ก็ COMPLETED ไปแล้ว ถ้าตัดใบพวกนี้ออกจากการปิด จะไม่มีรอบไหนเลยที่ข้อมูลครบ
func partitionDeliveriesByStatus(deliveryCodes []string, deliveryOf map[string]models.Delivery, targetStatus string) ([]string, []string, error) {
	toUpdate := []string{}
	alreadyAtStatus := []string{}

	for _, deliveryCode := range deliveryCodes {
		delivery, found := deliveryOf[deliveryCode]
		if !found {
			return nil, nil, fmt.Errorf("delivery with code %s not found", deliveryCode)
		}

		if delivery.Status == targetStatus {
			alreadyAtStatus = append(alreadyAtStatus, deliveryCode)
			continue
		}

		// ยกเลิกไปแล้วย้อนกลับไม่ได้
		if delivery.Status == "CANCELED" {
			return nil, nil, fmt.Errorf("delivery %s is already canceled", deliveryCode)
		}

		toUpdate = append(toUpdate, deliveryCode)
	}

	return toUpdate, alreadyAtStatus, nil
}

// closeSalesOfDeliveries ไล่ปิด SO ต้นทางของใบจองที่ตอนนี้อยู่ COMPLETED
//
// ต้องทำหลัง commit และ "log ทิ้งถ้าพัง" ห้ามคืน error — hook ORDER/DELIVERY/UPDATE
// ยิงเข้ามาระหว่างที่ wms-order-service ยังไม่ commit ถ้าเราคืน error ฝั่งนั้นจะ rollback
// แล้วยืนยัน pack ล้มทั้งใบ ทั้งที่สต็อกกับ GI ตัดไปแล้ว
func closeSalesOfDeliveries(ctx context.Context, gormx *gorm.DB, deliveryOf map[string]models.Delivery, deliveryCodes []string, user string) {
	if len(deliveryCodes) == 0 {
		return
	}

	// เส้นนี้ทำงานหลัง commit และห้ามล้มตาม caller
	// postCommitContext เก็บ user/token ไว้ครบ แต่ตัดการยกเลิกทิ้ง ไม่งั้นพอ hook ฝั่ง
	// wms-order-service หมดเวลาแล้วตัดสาย ctx จะถูกยกเลิกและการปิด SO จะไม่เกิดขึ้นเลย
	ctx = postCommitContext(ctx)

	saleCodes := []string{}
	for _, deliveryCode := range deliveryCodes {
		saleCodes = append(saleCodes, deliveryOf[deliveryCode].DocumentRef)
	}

	if err := CloseSalesFullyDelivered(ctx, gormx, saleCodes, user); err != nil {
		fmt.Printf("UpdateStatusDelivery: cannot close sales of %v: %v\n", deliveryCodes, err)
	}
}

// postCommitContext คืน context สำหรับงานที่ทำหลัง commit
// เก็บ user/token ไว้ครบ แต่ตัดการยกเลิกทิ้ง ไม่งั้นพอ caller หมดเวลาแล้วตัดสาย
// งานที่เหลือจะไม่เกิดขึ้นเลยและเงียบด้วย
func postCommitContext(ctx context.Context) context.Context {
	return context.WithoutCancel(ctx)
}

func CancelOrder(ctx context.Context, delivery models.Delivery) (orderExternalService.CancelOrderResponse, error) {
	cancelOrderRequest := orderExternalService.CancelOrderRequest{
		DocumentRef: []string{delivery.DeliveryCode},
	}

	cancelOrderResponse, err := orderExternalService.CancelOrder(ctx, cancelOrderRequest)
	if err != nil {
		return orderExternalService.CancelOrderResponse{}, errors.New("Error cancel order : " + err.Error())
	}

	return cancelOrderResponse, nil
}

// deliveriesWithOutbound ถาม WMS ว่าใบไหนถูกสร้าง outbound ไปแล้วบ้าง
// ใช้กันไม่ให้ยกเลิกใบที่คลังเริ่มทำงานไปแล้ว
func deliveriesWithOutbound(ctx context.Context, deliveryCodes []string) (map[string]bool, error) {
	started := map[string]bool{}

	orderRes, err := orderExternalService.GetOrdersDelivery(ctx, orderExternalService.GetOrderDeliveryRequest{
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
