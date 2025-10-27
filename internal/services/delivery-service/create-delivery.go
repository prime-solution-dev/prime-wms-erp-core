package deliveryService

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	orderExternalService "prime-erp-core/external/order-service"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CreateDeliveryRequest struct {
	IsDraft           bool                         `json:"is_draft"`
	CompanyCode       string                       `json:"company_code"`
	SiteCode          string                       `json:"site_code"`
	DeliveryMethod    string                       `json:"delivery_method"`
	DocumentRef       string                       `json:"document_ref"`
	CustomerCode      string                       `json:"customer_code"`
	SoldToCode        string                       `json:"sold_to_code"`
	ShipToCode        string                       `json:"ship_to_code"`
	BillToCode        string                       `json:"bill_to_code"`
	InterfaceQty      float64                      `json:"interface_qty"`
	InterfaceUnitCode string                       `json:"interface_unit_code"`
	Qty               float64                      `json:"qty"`
	UnitCode          string                       `json:"unit_code"`
	ShipToAddress     string                       `json:"ship_to_address"`
	DeliveryDate      *time.Time                   `json:"delivery_date"`
	DeliveryTimeCode  string                       `json:"delivery_time_code"`
	LicensePlate      string                       `json:"license_plate"`
	ContactName       string                       `json:"contact_name"`
	Tel               string                       `json:"tel"`
	TotalWeight       float64                      `json:"total_weight"`
	Remark            string                       `json:"remark"`
	DeliveryItems     []CreateDeliveryItemsRequest `json:"delivery_items"`
}

type CreateDeliveryItemsRequest struct {
	ProductCode     string  `json:"product_code"`
	Qty             float64 `json:"qty"`
	UnitCode        string  `json:"unit_code"`
	Weight          float64 `json:"weight"`
	WeightUnit      float64 `json:"weight_unit"`
	DocumentRefItem string  `json:"document_ref_item"`
	SaleUnitCode    string  `json:"sale_unit_code"`
	SaleMethod      string  `json:"sale_method"`
}

func CreateDelivery(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req []CreateDeliveryRequest

	// Bind JSON payload
	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	// Connect to the database
	gormx, err := db.ConnectGORM("prime_erp")
	defer db.CloseGORM(gormx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to database"})
		return nil, err
	}

	tx := gormx.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	user := "SYSTEM" // TODO: get from ctx
	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	deliveryToAdd := []models.Delivery{}
	deliveryItemToAdd := []models.DeliveryItem{}

	for num, deliveryReq := range req {
		deliveryId := uuid.New()

		// แปลง DeliveryDate เป็น date-only format
		var deliveryDateOnly *time.Time
		if deliveryReq.DeliveryDate != nil {
			dateOnly := time.Date(deliveryReq.DeliveryDate.Year(), deliveryReq.DeliveryDate.Month(), deliveryReq.DeliveryDate.Day(), 0, 0, 0, 0, deliveryReq.DeliveryDate.Location())
			deliveryDateOnly = &dateOnly
		}

		newDelivery := models.Delivery{
			ID:               deliveryId,
			DeliveryCode:     "DELIVERY-" + time.Now().Format("20060102150405") + fmt.Sprintf("%d", num),
			CompanyCode:      deliveryReq.CompanyCode,
			SiteCode:         deliveryReq.SiteCode,
			DeliveryMethod:   deliveryReq.DeliveryMethod,
			DocumentRef:      deliveryReq.DocumentRef,
			CustomerCode:     deliveryReq.CustomerCode,
			ShipToAddress:    deliveryReq.ShipToAddress,
			DeliveryDate:     deliveryDateOnly, // ใช้ date-only
			DeliveryTimeCode: deliveryReq.DeliveryTimeCode,
			LicensePlate:     deliveryReq.LicensePlate,
			ContactName:      deliveryReq.ContactName,
			Tel:              deliveryReq.Tel,
			TotalWeight:      deliveryReq.TotalWeight,
			Remark:           deliveryReq.Remark,
			Status: func() string {
				if deliveryReq.IsDraft {
					return "TEMP"
				}
				return "PENDING"
			}(),
			CreateBy:   user,
			CreateDate: nowDateOnly, // date-only format
			UpdateBy:   user,
			UpdateDate: nowDateOnly, // date-only format
		}

		deliveryToAdd = append(deliveryToAdd, newDelivery)

		for numItem, deliveryItem := range deliveryReq.DeliveryItems {
			deliveryItemId := uuid.New()

			newDeliveryItem := models.DeliveryItem{
				ID:              deliveryItemId,
				DeliveryItem:    fmt.Sprintf("ITEM-%s-%d", deliveryId.String(), numItem),
				DeliveryID:      deliveryId,
				ProductCode:     deliveryItem.ProductCode,
				Qty:             deliveryItem.Qty,
				UnitCode:        deliveryItem.UnitCode,
				Weight:          deliveryItem.Weight,
				WeightUnit:      deliveryItem.WeightUnit,
				Status:          "PENDING",
				DocumentRefItem: deliveryItem.DocumentRefItem,
				CreateDate:      nowDateOnly, // date-only format
				CreateBy:        user,
				UpdateDate:      nowDateOnly, // date-only format
				UpdateBy:        user,
			}

			deliveryItemToAdd = append(deliveryItemToAdd, newDeliveryItem)
		}
	}

	if len(deliveryToAdd) > 0 {
		if err := tx.Create(&deliveryToAdd).Error; err != nil {
			return nil, err
		}
	}

	if len(deliveryItemToAdd) > 0 {
		if err := tx.Create(&deliveryItemToAdd).Error; err != nil {
			return nil, err
		}
	}

	// Check if any delivery is not a draft before calling external service
	hasNonDraftDelivery := false
	for _, deliveryReq := range req {
		if !deliveryReq.IsDraft {
			hasNonDraftDelivery = true
			break
		}
	}

	var orderRes orderExternalService.CreateOrderResponse
	// Only call external service if there are non-draft deliveries
	if hasNonDraftDelivery {
		orderRes, err = CreateOrder(req, deliveryToAdd, deliveryItemToAdd)
		if err != nil {
			return nil, err
		}
	}

	// Return the delivery codes of the created deliveries
	deliveryCodes := make([]string, len(deliveryToAdd))
	for i, d := range deliveryToAdd {
		deliveryCodes[i] = d.DeliveryCode
	}

	response := gin.H{
		"status":        "success",
		"message":       "Create delivery successfully",
		"delivery_code": deliveryCodes,
	}

	// Only include order_code if external service was called
	if hasNonDraftDelivery {
		response["order_code"] = orderRes.OrderCode
	}

	return response, nil
}

func CreateOrder(req []CreateDeliveryRequest, deliveryToAdd []models.Delivery, deliveryItemToAdd []models.DeliveryItem) (orderExternalService.CreateOrderResponse, error) {
	createOrderRequest := orderExternalService.CreateOrderRequest{}
	createOrderdetail := []orderExternalService.CreateOrderDetail{}
	for _, deliveryReq := range req {
		createOrderItemDetail := []orderExternalService.CreateOrderItemDetail{}
		for _, item := range deliveryReq.DeliveryItems {
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
				SaleUnitCode:      item.SaleUnitCode,
				SaleMethod:        item.SaleMethod,
				Weight:            item.Weight,
				WeightUnit:        item.WeightUnit,
				Remark:            "",
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
