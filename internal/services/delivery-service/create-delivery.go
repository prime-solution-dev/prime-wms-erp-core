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
	"gorm.io/gorm"
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
	// defer ต้องอยู่หลังเช็ค err ไม่งั้นต่อ DB ไม่ได้แล้ว gormx = nil และ CloseGORM จะ panic
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to database"})
		return nil, err
	}
	defer db.CloseGORM(gormx)

	// กันจองเกินจำนวนใน sale order ก่อนแตะอะไรทั้งนั้น (ทั้ง DB และ hook ภายนอก)
	bookingLines := []bookingLine{}
	for _, deliveryReq := range req {
		for _, item := range deliveryReq.DeliveryItems {
			bookingLines = append(bookingLines, bookingLine{
				SaleCode:        deliveryReq.DocumentRef,
				DocumentRefItem: item.DocumentRefItem,
				ProductCode:     item.ProductCode,
				Qty:             item.Qty,
			})
		}
	}
	if err := ValidateBookingQty(gormx, bookingLines, nil); err != nil {
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
		HookInterfaceValue, hookErr := interfaceService.HookInterface(requestDataCreateHook)
		if hookErr != nil {
			// hook เป็นข้อมูลเสริม (external id) ไม่ควรทำให้สร้างใบไม่ได้ แต่ต้องเห็นใน log
			fmt.Printf("CreateDelivery: delivery hook failed, continuing without external id: %v\n", hookErr)
		}

		// เดิม assert ตรงๆ ทั้ง 2 ชั้น hook ที่ตอบมาไม่ใช่ map หรือไม่มี key "last" จะ panic
		if hookResult, ok := HookInterfaceValue.(map[string]interface{}); ok {
			externalId, _ = hookResult["id"].(string)
			externaldocNo, _ = hookResult["last"].(string)
		}
	}

	user := ctx.GetString("user")
	if user == "" {
		user = `system` // fallback
	}
	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	deliveryToAdd := []models.Delivery{}
	deliveryItemToAdd := []models.DeliveryItem{}

	// Generate all delivery codes first
	deliveryCodes, err := generateDeliveryCodes(gormx, len(req))
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
			TotalWeight:      roundWeight(deliveryReq.TotalWeight),
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
				Weight:          roundWeight(deliveryItem.Weight),
				WeightUnit:      roundWeight(deliveryItem.WeightUnit),
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
		// ใช้ = ไม่ใช่ := เพื่อให้ deferred func ด้านบนเห็น err แล้ว Rollback จริง
		// ของเดิม := บังตัวนอกไว้ ทำให้ deferred เรียก Commit บน tx ที่ล้มไปแล้ว
		if err = tx.Create(&deliveryToAdd).Error; err != nil {
			return nil, err
		}
	}

	if len(deliveryItemToAdd) > 0 {
		if err = tx.Create(&deliveryItemToAdd).Error; err != nil {
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
	// deliveryToAdd เรียงตรงกับ req ทีละใบ (สร้างมาจากลูปเดียวกัน) ส่วน deliveryItemToAdd
	// เป็น list แบนรวมทุกใบ จึงต้องจัดกลุ่มตาม delivery_id ก่อน ไม่งั้นการหาแบบ
	// DocumentRefItem + ProductCode จะไปเจอ item ของ "ใบอื่น" ในกรณีที่ payload มีหลายใบ
	itemsOfDelivery := map[uuid.UUID][]models.DeliveryItem{}
	for _, di := range deliveryItemToAdd {
		itemsOfDelivery[di.DeliveryID] = append(itemsOfDelivery[di.DeliveryID], di)
	}

	for num, deliveryReq := range req {
		if num >= len(deliveryToAdd) {
			return orderExternalService.CreateOrderResponse{}, fmt.Errorf("delivery at index %d was not created", num)
		}

		delivery := deliveryToAdd[num]
		srcItems := itemsOfDelivery[delivery.ID]

		createOrderItemDetail := []orderExternalService.CreateOrderItemDetail{}
		for itemNum, item := range deliveryReq.DeliveryItems {
			if itemNum >= len(srcItems) {
				return orderExternalService.CreateOrderResponse{}, fmt.Errorf(
					"delivery %s item %d was not created", delivery.DeliveryCode, itemNum)
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
				SaleUnitCode:      item.SaleUnitCode,
				SaleMethod:        item.SaleMethod,
				Weight:            roundWeight(item.Weight),
				WeightUnit:        roundWeight(item.WeightUnit),
				Remark:            item.Remark,
				Status:            "PENDING",
			}
			createOrderItemDetail = append(createOrderItemDetail, newOrderItemDetail)
		}

		// เดิมหาด้วย DocumentRef ตัวแรกที่เจอ จองสองใบให้ SO เดียวกันในครั้งเดียว
		// ใบที่สองจะพก delivery_code ของใบแรกไปเป็น document_ref ฝั่ง WMS
		deliveryCode := delivery.DeliveryCode

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

// generateDeliveryCodes จองเลขที่ใบจองแบบ atomic (ล็อกแถว config จนกว่าจะเดินเลขเสร็จ)
// เดิมแยกเป็นอ่านเลขตอนนี้ แล้วค่อยเดินเลขหลัง insert ซึ่งทำให้สองคนที่กดพร้อมกันได้เลขเดียวกัน
func generateDeliveryCodes(gormx *gorm.DB, count int) ([]string, error) {
	if count <= 0 {
		return []string{}, nil // No deliveries to generate codes for
	}

	codes, err := systemConfigService.ReserveRunningCodes(
		gormx, "RUNNING_DBS", count, "", systemConfigService.StandardRunningPeriod())
	if err != nil {
		return nil, fmt.Errorf("failed to generate delivery codes: %v", err)
	}

	if len(codes) != count {
		return nil, errors.New("failed to get correct number of delivery codes from system config")
	}

	return codes, nil
}
