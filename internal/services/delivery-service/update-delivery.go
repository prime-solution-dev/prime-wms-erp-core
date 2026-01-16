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
		if deliveryReq.IsDraft {
			tempDelivery.Status = "TEMP"
		} else {
			tempDelivery.Status = "PENDING"
		}

		updateDeliveries = append(updateDeliveries, tempDelivery)

		for _, item := range deliveryReq.Items {
			// Ensure item belongs to this delivery
			item.DeliveryItem.DeliveryID = tempDelivery.ID
			item.DeliveryItem.UpdateDate = nowDateOnly
			item.DeliveryItem.UpdateBy = user
			item.DeliveryItem.Status = "PENDING"

			updateDeliveryItems = append(updateDeliveryItems, item.DeliveryItem)
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

	// Check if any delivery is not a draft and was previously a draft before calling external service
	hasNonDraftDelivery := false
	for _, deliveryReq := range req.Deliveries {
		if !deliveryReq.IsDraft {
			// Check previous status from database
			var previousDelivery models.Delivery
			if err := gormx.Where("id = ?", deliveryReq.Delivery.ID).First(&previousDelivery).Error; err == nil {
				// Only create order if previous status was draft (TEMP) and current is not draft
				if previousDelivery.Status == "TEMP" {
					hasNonDraftDelivery = true
					break
				}
			}
		}
	}

	// Only call external service if there are non-draft deliveries
	var orderCode string
	if hasNonDraftDelivery {
		orderRes, err := CreateOrderForUpdate(req.Deliveries, updateDeliveries, updateDeliveryItems)
		if err != nil {
			return nil, fmt.Errorf("failed to update external order: %v", err)
		}
		if len(orderRes.OrderCode) > 0 {
			orderCode = orderRes.OrderCode[0] // Use first order code
		}
	}

	// Call UpdateOrderByDelivery for each non-draft delivery
	for _, deliveryReq := range req.Deliveries {
		if !deliveryReq.IsDraft {
			err := UpdateOrderByDeliveryForUpdate(deliveryReq, updateDeliveries)
			if err != nil {
				return nil, fmt.Errorf("failed to update order by delivery for %s: %v", deliveryReq.DeliveryCode, err)
			}
		}
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
	for _, deliveryReq := range req {
		// Skip draft deliveries
		if deliveryReq.IsDraft {
			continue
		}

		createOrderItemDetail := []orderExternalService.CreateOrderItemDetail{}
		for _, item := range deliveryReq.Items {
			// find corresponding DeliveryItem from deliveryItemToAdd (match by DocumentRefItem + ProductCode)
			var srcItem *models.DeliveryItem
			for i := range deliveryItemToAdd {
				di := &deliveryItemToAdd[i]
				if di.DocumentRefItem == item.DocumentRefItem && di.ProductCode == item.ProductCode {
					srcItem = di
					break
				}
			}

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

		deliveryCode := ""
		for _, d := range deliveryToAdd {
			if d.DocumentRef == deliveryReq.DocumentRef {
				deliveryCode = d.DeliveryCode
				break
			}
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
			OrderItem:           createOrderItemDetail,
		}

		createOrderdetail = append(createOrderdetail, newOrderDetail)
	}
	createOrderRequest.Orders = createOrderdetail

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
	var delivery models.Delivery
	for _, d := range updateDeliveries {
		if d.DocumentRef == deliveryReq.DocumentRef {
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
