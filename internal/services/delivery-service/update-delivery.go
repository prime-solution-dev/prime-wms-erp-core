package deliveryService

import (
	"encoding/json"
	"errors"
	"fmt"
	externalService "prime-erp-core/external/order-service"
	orderExternalService "prime-erp-core/external/order-service"
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
	// GORM fields for delivery
	models.Delivery

	// Additional fields for external service (not for GORM)
	PaymentMethod     string                       `json:"payment_method" gorm:"-"`
	IsDraft           bool                         `json:"is_draft" gorm:"-"`
	SoldToCode        string                       `json:"sold_to_code" gorm:"-"`
	ShipToCode        string                       `json:"ship_to_code" gorm:"-"`
	BillToCode        string                       `json:"bill_to_code" gorm:"-"`
	InterfaceQty      float64                      `json:"interface_qty" gorm:"-"`
	InterfaceUnitCode string                       `json:"interface_unit_code" gorm:"-"`
	Qty               float64                      `json:"qty" gorm:"-"`
	UnitCode          string                       `json:"unit_code" gorm:"-"`
	Items             []DeliveryItemDocumentUpdate `json:"items" gorm:"-"`        // Items to update
	DeleteItems       []uuid.UUID                  `json:"delete_items" gorm:"-"` // Item IDs to delete
}

type DeliveryItemDocumentUpdate struct {
	// GORM fields for delivery item
	models.DeliveryItem

	// Additional fields for external service (not for GORM)
	SaleUnitCodeForOrder string `json:"sale_unit_code_for_order" gorm:"-"`
	SaleMethodForOrder   string `json:"sale_method_for_order" gorm:"-"`
}

type UpdateDeliveryResponse struct {
	DeliveryCode string `json:"delivery_code"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	OrderCode    string `json:"order_code,omitempty"`
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

	user := ctx.GetString("user")
	if user == "" {
		user = `system` // fallback
	}
	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// อ่านสถานะเดิมของทุกใบก่อน ใช้ทั้งกันไม่ให้แก้ใบที่จบไปแล้ว
	// และดูว่าใบไหนเพิ่งออกจาก draft (เดิมอ่านทีละใบด้วย gormx ระหว่างที่ tx เปิดค้างอยู่)
	deliveryIDs := []uuid.UUID{}
	for _, deliveryReq := range req.Deliveries {
		if deliveryReq.ID == uuid.Nil {
			return nil, fmt.Errorf("delivery ID is required for update")
		}
		if deliveryReq.DeliveryCode == "" {
			return nil, fmt.Errorf("delivery code is required for update")
		}
		deliveryIDs = append(deliveryIDs, deliveryReq.ID)
	}

	var previousDeliveries []models.Delivery
	if err := gormx.Where("id IN ?", deliveryIDs).Find(&previousDeliveries).Error; err != nil {
		return nil, fmt.Errorf("failed to load deliveries: %v", err)
	}

	previousOf := map[uuid.UUID]models.Delivery{}
	for _, delivery := range previousDeliveries {
		previousOf[delivery.ID] = delivery
	}

	updateDeliveries := []models.Delivery{}
	updateDeliveryItems := []models.DeliveryItem{}
	deliveryUpdateFields := map[uuid.UUID]map[string]interface{}{}
	itemUpdateFields := map[uuid.UUID]map[string]interface{}{}

	for _, deliveryReq := range req.Deliveries {
		previous, found := previousOf[deliveryReq.ID]
		if !found {
			return nil, fmt.Errorf("delivery with code %s not found", deliveryReq.DeliveryCode)
		}

		// เดิมไม่ดูสถานะเดิมเลย บังคับเขียนทับเป็น TEMP/PENDING เสมอ
		// ใบที่คลังส่งจบแล้วจึงถูกดันกลับมาเป็น PENDING ได้จากการแก้อะไรนิดเดียว
		if previous.Status == "COMPLETED" || previous.Status == "CANCELED" {
			return nil, fmt.Errorf("delivery %s is already %s and cannot be updated",
				deliveryReq.DeliveryCode, previous.Status)
		}

		tempDelivery := deliveryReq.Delivery

		// Convert date fields to date-only format
		if tempDelivery.DeliveryDate != nil {
			deliveryDateOnly := time.Date(tempDelivery.DeliveryDate.Year(), tempDelivery.DeliveryDate.Month(), tempDelivery.DeliveryDate.Day(), 0, 0, 0, 0, tempDelivery.DeliveryDate.Location())
			tempDelivery.DeliveryDate = &deliveryDateOnly
		}

		tempDelivery.UpdateDate = nowDateOnly
		tempDelivery.UpdateBy = user
		if deliveryReq.IsDraft {
			tempDelivery.Status = "TEMP"
		} else {
			tempDelivery.Status = "PENDING"
		}

		updateDeliveries = append(updateDeliveries, tempDelivery)

		// ระบุคอลัมน์เป็น map ไม่ส่ง struct เพราะ GORM v2 ข้าม zero-value ของ struct ทิ้ง
		// ทำให้ล้าง license_plate / remark ให้ว่าง หรือตั้ง total_weight เป็น 0 ไม่ติด
		deliveryUpdateFields[tempDelivery.ID] = map[string]interface{}{
			"delivery_method":    tempDelivery.DeliveryMethod,
			"document_ref":       tempDelivery.DocumentRef,
			"customer_code":      tempDelivery.CustomerCode,
			"ship_to_address":    tempDelivery.ShipToAddress,
			"delivery_date":      tempDelivery.DeliveryDate,
			"delivery_time_code": tempDelivery.DeliveryTimeCode,
			"license_plate":      tempDelivery.LicensePlate,
			"contact_name":       tempDelivery.ContactName,
			"tel":                tempDelivery.Tel,
			"total_weight":       tempDelivery.TotalWeight,
			"remark":             tempDelivery.Remark,
			"booking_slot_type":  tempDelivery.BookingSlotType,
			"status":             tempDelivery.Status,
			"update_date":        tempDelivery.UpdateDate,
			"update_by":          tempDelivery.UpdateBy,
		}

		for _, item := range deliveryReq.Items {
			// Ensure item belongs to this delivery
			item.DeliveryItem.DeliveryID = tempDelivery.ID
			item.DeliveryItem.UpdateDate = nowDateOnly
			item.DeliveryItem.UpdateBy = user
			item.DeliveryItem.Status = "PENDING"

			updateDeliveryItems = append(updateDeliveryItems, item.DeliveryItem)

			itemUpdateFields[item.DeliveryItem.ID] = map[string]interface{}{
				"product_code":      item.DeliveryItem.ProductCode,
				"qty":               item.DeliveryItem.Qty,
				"unit_code":         item.DeliveryItem.UnitCode,
				"price_list_unit":   item.DeliveryItem.PriceListUnit,
				"sale_qty":          item.DeliveryItem.SaleQty,
				"sale_unit_code":    item.DeliveryItem.SaleUnitCode,
				"total_weight":      item.DeliveryItem.TotalWeight,
				"weight":            item.DeliveryItem.Weight,
				"weight_unit":       item.DeliveryItem.WeightUnit,
				"document_ref_item": item.DeliveryItem.DocumentRefItem,
				"remark":            item.DeliveryItem.Remark,
				"status":            item.DeliveryItem.Status,
				"update_date":       item.DeliveryItem.UpdateDate,
				"update_by":         item.DeliveryItem.UpdateBy,
			}
		}
	}

	// กันจองเกินจำนวนใน sale order โดยไม่นับจำนวนของใบที่กำลังแก้ซ้ำเข้าไปเอง
	bookingLines := []bookingLine{}
	editingDeliveryCodes := []string{}
	for _, deliveryReq := range req.Deliveries {
		editingDeliveryCodes = append(editingDeliveryCodes, deliveryReq.DeliveryCode)
		for _, item := range deliveryReq.Items {
			bookingLines = append(bookingLines, bookingLine{
				SaleCode:        deliveryReq.DocumentRef,
				DocumentRefItem: item.DeliveryItem.DocumentRefItem,
				ProductCode:     item.DeliveryItem.ProductCode,
				Qty:             item.DeliveryItem.Qty,
			})
		}
	}
	if err := ValidateBookingQty(gormx, bookingLines, editingDeliveryCodes); err != nil {
		return nil, err
	}

	// แยกว่าใบไหนต้องสร้าง order ใหม่ (เพิ่งออกจาก draft) กับใบไหนแค่อัปเดต order เดิม
	newOrderDeliveries := []DeliveryDocumentUpdate{}
	updateOrderDeliveries := []DeliveryDocumentUpdate{}
	for _, deliveryReq := range req.Deliveries {
		if deliveryReq.IsDraft {
			continue
		}

		if previousOf[deliveryReq.ID].Status == "TEMP" {
			newOrderDeliveries = append(newOrderDeliveries, deliveryReq)
		} else {
			updateOrderDeliveries = append(updateOrderDeliveries, deliveryReq)
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
			Updates(deliveryUpdateFields[delivery.ID]).Error; err != nil {
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
			Updates(itemUpdateFields[item.ID]).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to update delivery item %s: %v", item.DeliveryItem, err)
		}
	}

	// ยิง WMS ก่อน commit เหมือนที่ create-delivery ทำ
	// เดิม commit ไปแล้วค่อยยิง ถ้า WMS พังใบจะดูเหมือน submit สำเร็จแต่คลังไม่เคยได้ order
	var orderCode string
	if len(newOrderDeliveries) > 0 {
		orderRes, err := CreateOrderForUpdate(newOrderDeliveries, updateDeliveries, updateDeliveryItems)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to update external order: %v", err)
		}
		if len(orderRes.OrderCode) > 0 {
			orderCode = orderRes.OrderCode[0] // Use first order code
		}
	}

	for _, deliveryReq := range updateOrderDeliveries {
		if err := UpdateOrderByDeliveryForUpdate(deliveryReq, updateDeliveries); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to update order by delivery for %s: %v", deliveryReq.DeliveryCode, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Update response with order code if available
	for i := range res {
		if orderCode != "" {
			res[i].OrderCode = orderCode
		}
	}

	return res, nil
}

func CreateOrderForUpdate(req []DeliveryDocumentUpdate, deliveryToAdd []models.Delivery, deliveryItemToAdd []models.DeliveryItem) (orderExternalService.CreateOrderResponse, error) {
	createOrderRequest := orderExternalService.CreateOrderRequest{}
	createOrderdetail := []orderExternalService.CreateOrderDetail{}

	// deliveryItemToAdd เป็น list แบนรวมทุกใบ ต้องจัดกลุ่มตาม delivery_id ก่อน
	// ไม่งั้นการหาแบบ DocumentRefItem + ProductCode จะไปเจอ item ของ "ใบอื่น"
	// (ที่นี่ใช้ index เทียบไม่ได้ เพราะ req ถูกกรองมาแล้วไม่ตรงกับ deliveryToAdd)
	itemsOfDelivery := map[uuid.UUID][]models.DeliveryItem{}
	for _, di := range deliveryItemToAdd {
		itemsOfDelivery[di.DeliveryID] = append(itemsOfDelivery[di.DeliveryID], di)
	}

	for _, deliveryReq := range req {
		// Skip draft deliveries
		if deliveryReq.IsDraft {
			continue
		}

		srcItems := itemsOfDelivery[deliveryReq.ID]

		createOrderItemDetail := []orderExternalService.CreateOrderItemDetail{}
		for itemNum, item := range deliveryReq.Items {
			if itemNum >= len(srcItems) {
				return orderExternalService.CreateOrderResponse{}, fmt.Errorf(
					"delivery %s item %d was not prepared for update", deliveryReq.DeliveryCode, itemNum)
			}

			srcItem := srcItems[itemNum]

			newOrderItemDetail := orderExternalService.CreateOrderItemDetail{
				OrderItem:         "",
				DocumentRefItem:   srcItem.DeliveryItem,
				ProductCode:       item.ProductCode,
				ProductType:       "normal",
				InterfaceOrderQty: item.Qty,
				Qty:               item.Qty,
				UnitCode:          item.UnitCode,
				IsFocGwp:          false,
				WarehouseCode:     "",
				BatchNo:           "",
				SerialCode:        "",
				SaleUnitCode:      item.SaleUnitCodeForOrder,
				SaleMethod:        item.SaleMethodForOrder,
				Weight:            item.Weight,
				WeightUnit:        item.WeightUnit,
				Remark:            item.Remark,
				Status:            "PENDING",
			}
			createOrderItemDetail = append(createOrderItemDetail, newOrderItemDetail)
		}

		// เดิมหาด้วย DocumentRef ตัวแรกที่เจอ แก้สองใบของ SO เดียวกันในครั้งเดียว
		// ใบที่สองจะพก delivery_code ของใบแรกไปเป็น document_ref ฝั่ง WMS
		deliveryCode := deliveryReq.DeliveryCode

		var statusApproveGi string
		if deliveryReq.PaymentMethod == "CASH" {
			statusApproveGi = "PENDING"
		} else {
			statusApproveGi = "COMPLETED"
		}

		newOrderDetail := orderExternalService.CreateOrderDetail{
			Action:              "X",
			OrderID:             uuid.New(),
			OrderCode:           "",
			OrderType:           "DELIVERY",
			OrderDate:           time.Now(),
			TenantID:            nil,
			CustomerCode:        deliveryReq.CustomerCode,
			SoldToCode:          deliveryReq.SoldToCode,
			ShipToCode:          deliveryReq.ShipToCode,
			BillToCode:          deliveryReq.BillToCode,
			TransportZone:       "BKK",
			InterfaceQty:        deliveryReq.InterfaceQty,
			InterfaceUnitCode:   deliveryReq.InterfaceUnitCode,
			Qty:                 deliveryReq.Qty,
			UnitCode:            deliveryReq.UnitCode,
			EstimatePickingDate: nil,
			DeliveryDate:        deliveryReq.DeliveryDate,
			SubmitDate:          nil,
			Status:              "PENDING",
			DocumentRefType:     "DELIVERY",
			DocumentRef:         deliveryCode,
			Remark:              deliveryReq.Remark,
			CompanyCode:         deliveryReq.CompanyCode,
			SiteCode:            deliveryReq.SiteCode,
			DocumentRef2:        deliveryReq.DocumentRef,
			DocumentRefType2:    "SALES_ORDER",
			PartyCode:           "",
			PartyName:           "",
			PartyType:           "",
			Reason:              "",
			ShippingAddress:     "",
			DeliveryMethod:      deliveryReq.DeliveryMethod,
			BookingDate:         deliveryReq.DeliveryDate,
			DeliveryTimeCode:    deliveryReq.DeliveryTimeCode,
			Tel:                 deliveryReq.Tel,
			LicensePlate:        deliveryReq.LicensePlate,
			ContactName:         deliveryReq.ContactName,
			StatusApproveGi:     statusApproveGi,
			OrderItem:           createOrderItemDetail,
		}

		createOrderdetail = append(createOrderdetail, newOrderDetail)
	}
	createOrderRequest.Orders = createOrderdetail

	requestJSON, _ := json.MarshalIndent(createOrderRequest, "", "  ")
	fmt.Println("CreateGoodsIssueRequest JSON:")
	fmt.Println(string(requestJSON))
	fmt.Println("createOrderRequest : ", createOrderRequest)
	createOrderResponse, err := orderExternalService.CreateOrder(createOrderRequest)
	if err != nil {
		return orderExternalService.CreateOrderResponse{}, errors.New("Error create order : " + err.Error())
	}
	fmt.Println("createOrderResponse : ", createOrderResponse)

	return createOrderResponse, nil
}

func UpdateOrderByDeliveryForUpdate(deliveryReq DeliveryDocumentUpdate, updateDeliveries []models.Delivery) error {
	// Find the corresponding delivery from updateDeliveries
	// เทียบด้วย id ของใบ ไม่ใช่ DocumentRef (เลข SO) ซึ่งซ้ำกันได้หลายใบ
	var delivery models.Delivery
	for _, d := range updateDeliveries {
		if d.ID == deliveryReq.ID {
			delivery = d
			break
		}
	}

	// Create order items from the new items only
	orderItems := []externalService.UpdateOrderByDeliveryItemDetail{}
	for _, item := range deliveryReq.Items {
		orderItem := externalService.UpdateOrderByDeliveryItemDetail{
			OrderItem:            "",
			DocumentRefItem:      item.DocumentRefItem,
			ProductCode:          item.ProductCode,
			ProductType:          "normal",
			InterfaceOrderQty:    item.Qty,
			Qty:                  item.Qty,
			UnitCode:             item.UnitCode,
			IsFocGwp:             false,
			WarehouseCode:        "",
			BatchNo:              "",
			SerialCode:           "",
			SaleUnitCode:         item.SaleUnitCodeForOrder,
			SaleMethod:           item.SaleMethodForOrder,
			InterfaceOrderWeight: item.Weight,
			Weight:               item.Weight,
			WeightUnit:           item.WeightUnit,
			MfgDate:              nil,
			ExpDate:              nil,
			LocationCode:         "",
			StorageType:          "",
			Remark:               item.Remark,
			Status:               "PENDING",
		}
		orderItems = append(orderItems, orderItem)
	}

	updateOrderReq := externalService.UpdateOrderByDeliveryRequest{
		DocumentRef:      delivery.DeliveryCode,
		DeliveryMethod:   deliveryReq.DeliveryMethod,
		BookingDate:      deliveryReq.DeliveryDate,
		DeliveryTimeCode: deliveryReq.DeliveryTimeCode,
		Tel:              deliveryReq.Tel,
		LicensePlate:     deliveryReq.LicensePlate,
		ContactName:      deliveryReq.ContactName,
		Remark:           deliveryReq.Remark,
		OrderItem:        orderItems,
	}

	// Call UpdateOrderByDelivery
	resp, err := externalService.UpdateOrderByDelivery(updateOrderReq)
	if err != nil {
		return fmt.Errorf("failed to call UpdateOrderByDelivery: %v", err)
	}

	fmt.Println("updateOrderResponse : ", resp)
	return nil
}
