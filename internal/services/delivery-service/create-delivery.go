package deliveryService

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	orderExternalService "prime-erp-core/external/order-service"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	interfaceService "prime-erp-core/internal/services/interface-service"
	systemConfigService "prime-erp-core/internal/services/system-config"
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
	BookingSlotType   string                       `json:"booking_slot_type"`
	PaymentMethod     string                       `json:"payment_method"`
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
	Remark          string  `json:"remark"`
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

	externalId := ""
	externaldocNo := ""
	requestData := map[string]interface{}{
		"module":    []string{"DELIVERY"},
		"topic":     []string{"DELIVERY"},
		"sub_topic": []string{"CREATE"},
	}

	hookConfig, err := interfaceService.GetHookConfig(requestData)
	if err != nil {
		return nil, err
	}
	if len(hookConfig) > 0 {
		urlHook := ""
		for _, hookConfigValue := range hookConfig {
			urlHook = hookConfigValue.HookUrl
		}

		requestDataCreateHook := interfaceService.HookInterfaceRequest{
			RequestData: req,
			UrlHook:     urlHook,
		}
		HookInterfaceValue, err := interfaceService.HookInterface(requestDataCreateHook)
		if err != nil {
			return nil, err
		}
		if HookInterfaceValue != nil {
			externalID := HookInterfaceValue.(map[string]interface{})
			str, _ := externalID["id"].(string)
			externalId = str
			externaldocNo = externalID["last"].(string)
		}
	}

	user := "SYSTEM" // TODO: get from ctx
	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	deliveryToAdd := []models.Delivery{}
	deliveryItemToAdd := []models.DeliveryItem{}

	// Generate all delivery codes first
	deliveryCodes, err := generateDeliveryCodes(ctx, len(req))
	if err != nil {
		return nil, err
	}

	for num, deliveryReq := range req {
		deliveryId := uuid.New()

		// แปลง DeliveryDate เป็น date-only format
		var deliveryDateOnly *time.Time
		if deliveryReq.DeliveryDate != nil {
			dateOnly := time.Date(deliveryReq.DeliveryDate.Year(), deliveryReq.DeliveryDate.Month(), deliveryReq.DeliveryDate.Day(), 0, 0, 0, 0, deliveryReq.DeliveryDate.Location())
			deliveryDateOnly = &dateOnly
		}

		var statusApproveGi string
		if deliveryReq.PaymentMethod == "CASH" {
			statusApproveGi = "PENDING"
		} else {
			statusApproveGi = "COMPLETED"
		}

		newDelivery := models.Delivery{
			ID:               deliveryId,
			DeliveryCode:     deliveryCodes[num], // Use pre-generated delivery code
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
			BookingSlotType:        deliveryReq.BookingSlotType,
			StatusApproveGi:        statusApproveGi,
			ExternalID:             externalId,
			ExternalDocumentNumber: externaldocNo,
			CreateBy:               user,
			CreateDate:             nowDateOnly, // date-only format
			UpdateBy:               user,
			UpdateDate:             nowDateOnly, // date-only format
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
				Remark:          deliveryItem.Remark,
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

	// Update running number after successful creation
	if err := updateDeliveryRunningConfig(ctx, len(deliveryToAdd)); err != nil {
		// Log error but don't fail the transaction as deliveries are already created
		fmt.Printf("Warning: failed to update running config: %v\n", err)
	}

	// Return the delivery codes of the created deliveries
	finalDeliveryCodes := make([]string, len(deliveryToAdd))
	for i, d := range deliveryToAdd {
		finalDeliveryCodes[i] = d.DeliveryCode
	}

	response := gin.H{
		"status":        "success",
		"message":       "Create delivery successfully",
		"delivery_code": finalDeliveryCodes,
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

		var statusApproveGi string
		if deliveryReq.PaymentMethod == "CASH" {
			statusApproveGi = "PENDING"
		} else {
			statusApproveGi = "COMPLETED"
		}

		newOrderDetail := orderExternalService.CreateOrderDetail{
			Action:       "X",
			OrderID:      uuid.New(),
			OrderCode:    "",
			OrderType:    "DELIVERY",
			OrderDate:    time.Now(),
			TenantID:     nil,
			CustomerCode: deliveryReq.CustomerCode,
			SoldToCode:   deliveryReq.SoldToCode,
			ShipToCode: func() string {
				if deliveryReq.ShipToCode == "" {
					return deliveryReq.ShipToAddress
				}
				return deliveryReq.ShipToCode
			}(),
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

	fmt.Println("createOrderRequest : ", createOrderRequest)

	createOrderResponse, err := orderExternalService.CreateOrder(createOrderRequest)
	if err != nil {
		return orderExternalService.CreateOrderResponse{}, errors.New("Error create order : " + err.Error())
	}
	fmt.Println("createOrderResponse : ", createOrderResponse)

	return createOrderResponse, nil
}

// updateDeliveryRunningConfig updates the running number configuration for deliveries
func updateDeliveryRunningConfig(ctx *gin.Context, count int) error {
	if count <= 0 {
		return nil // No deliveries created, nothing to update
	}

	updateReq := systemConfigService.UpdateRunningSystemConfigRequest{
		ConfigCode: "RUNNING_DBS",
		Count:      count,
	}

	reqJSON, err := json.Marshal(updateReq)
	if err != nil {
		return fmt.Errorf("failed to marshal update request: %v", err)
	}

	_, err = systemConfigService.UpdateRunningSystemConfig(ctx, string(reqJSON))
	if err != nil {
		return fmt.Errorf("failed to update running config: %v", err)
	}

	return nil
}

// generateDeliveryCodes generates delivery codes using system config
func generateDeliveryCodes(ctx *gin.Context, count int) ([]string, error) {
	if count <= 0 {
		return []string{}, nil // No deliveries to generate codes for
	}

	getReq := systemConfigService.GetRunningSystemConfigRequest{
		ConfigCode: "RUNNING_DBS",
		Count:      count,
	}

	reqJSON, err := json.Marshal(getReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get request: %v", err)
	}

	deliveryCodeResponse, err := systemConfigService.GetRunningSystemConfig(ctx, string(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to generate delivery codes: %v", err)
	}

	deliveryResult, ok := deliveryCodeResponse.(systemConfigService.GetRunningSystemConfigResponse)
	if !ok || len(deliveryResult.Data) != count {
		return nil, errors.New("failed to get correct number of delivery codes from system config")
	}

	return deliveryResult.Data, nil
}
