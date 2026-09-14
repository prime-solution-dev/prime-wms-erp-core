package deliveryService

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	externalService "prime-erp-core/external/customer-service"
	orderExternalService "prime-erp-core/external/order-service"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetDeliveryRequest struct {
	ID                       []string   `json:"id"`
	DeliveryCode             []string   `json:"delivery_code"`
	NotInDeliveryCode        []string   `json:"not_in_delivery_code"`
	SaleOrderCode            []string   `json:"sale_order_code"`
	SiteCode                 []string   `json:"site_code"`
	CompanyCode              []string   `json:"company_code"`
	CustomerCode             []string   `json:"customer_code"`
	Status                   []string   `json:"status"`
	DeliveryCodeLike         string     `json:"delivery_code_like"`
	DocumentRefLike          string     `json:"document_ref_like"`
	CreateDateStart          *time.Time `json:"create_date_start"`
	CreateDateEnd            *time.Time `json:"create_date_end"`
	SaleOrderCreateDateStart *time.Time `json:"sale_order_create_date_start"`
	SaleOrderCreateDateEnd   *time.Time `json:"sale_order_create_date_end"`
	CompleteDateStart        *time.Time `json:"complete_date_start"`
	CompleteDateEnd          *time.Time `json:"complete_date_end"`
	ProductCodeLike          string     `json:"product_code_like"`
	CustomerCodeLike         string     `json:"customer_code_like"`
	CustomerNameLike         string     `json:"customer_name_like"`
	ShipToAddressLike        string     `json:"ship_to_address_like"`
	DeliveryDateStart        *time.Time `json:"delivery_date_start"`
	DeliveryDateEnd          *time.Time `json:"delivery_date_end"`
	LicensePlateLike         string     `json:"license_plate_like"`
	ContactNameLike          string     `json:"contact_name_like"`
	ShipSlotDateStart        *time.Time `json:"ship_slot_date_start"`
	ShipSlotDateEnd          *time.Time `json:"ship_slot_date_end"`
	DeliveryTimeNameLike     string     `json:"delivery_time_name_like"`
	StatusFilter             []string   `json:"status_filter"`
	PickPackFilter           []string   `json:"pick_pack_filter"`
	PaymentFilter            []string   `json:"payment_filter"`
	Page                     int        `json:"page"`
	PageSize                 int        `json:"page_size"`
}

func (GetDeliveryResponse) TableName() string { return "delivery_booking" }

func (GetDeliveryItemResponse) TableName() string { return "delivery_booking_item" }

type GetDeliveryResponse struct {
	ID               uuid.UUID                                     `gorm:"type:uuid;primary_key" json:"id"`
	DeliveryCode     string                                        `gorm:"type:varchar(50)" json:"delivery_code"`
	CompanyCode      string                                        `gorm:"type:varchar(50)" json:"company_code"`
	SiteCode         string                                        `gorm:"type:varchar(50)" json:"site_code"`
	DeliveryMethod   string                                        `gorm:"type:varchar(50)" json:"delivery_method"`
	DocumentRef      string                                        `gorm:"type:varchar(50)" json:"document_ref"`
	CustomerCode     string                                        `gorm:"type:varchar(50)" json:"customer_code"`
	ShipToAddress    string                                        `gorm:"type:varchar(255)" json:"ship_to_address"`
	DeliveryDate     *time.Time                                    `gorm:"type:date" json:"delivery_date"`
	DeliveryTimeCode string                                        `gorm:"type:varchar(50)" json:"delivery_time_code"`
	DeliveryTimeName string                                        `gorm:"type:varchar(100)" json:"delivery_time_name"`
	LicensePlate     string                                        `gorm:"type:varchar(50)" json:"license_plate"`
	ContactName      string                                        `gorm:"type:varchar(100)" json:"contact_name"`
	Tel              string                                        `gorm:"type:varchar(20)" json:"tel"`
	TotalWeight      float64                                       `gorm:"type:numeric" json:"total_weight"`
	Status           string                                        `gorm:"type:varchar(50)" json:"status"`
	BookingSlotType  string                                        `gorm:"type:varchar(50)" json:"booking_slot_type"`
	Remark           string                                        `gorm:"type:varchar(255)" json:"remark"`
	StatusApproveGi  string                                        `gorm:"type:varchar(50)" json:"status_approve_gi"`
	ExternalID       string                                        `gorm:"type:varchar(255)" json:"external_id"`
	CreateDate       *time.Time                                    `gorm:"type:date" json:"create_date"`
	CreateBy         string                                        `gorm:"type:varchar(50)" json:"create_by"`
	UpdateDate       *time.Time                                    `gorm:"type:date" json:"update_date"`
	UpdateBy         string                                        `gorm:"type:varchar(50)" json:"update_by"`
	SaleOrder        models.Sale                                   `gorm:"foreignKey:DocumentRef;references:SaleCode" json:"sale_order"`
	Order            orderExternalService.GetOrderDeliveryResponse `gorm:"-" json:"order"`
	PickPackStatus   string                                        `gorm:"-" json:"pick_pack_status"`
	Items            []GetDeliveryItemResponse                     `gorm:"foreignKey:DeliveryID" json:"items"`
}

type GetDeliveryItemResponse struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	DeliveryItem    string     `gorm:"type:varchar(50)" json:"delivery_item"`
	DeliveryID      uuid.UUID  `gorm:"type:uuid" json:"delivery_id"`
	ProductCode     string     `gorm:"type:varchar(50)" json:"product_code"`
	Qty             float64    `gorm:"type:numeric" json:"qty"`
	UnitCode        string     `gorm:"type:varchar(20)" json:"unit_code"`
	PriceListUnit   float64    `gorm:"type:numeric" json:"price_list_unit"`
	SaleQty         float64    `gorm:"type:numeric" json:"sale_qty"`
	SaleUnitCode    string     `gorm:"type:varchar(20)" json:"sale_unit_code"`
	TotalWeight     float64    `gorm:"type:numeric" json:"total_weight"`
	DocumentRefItem string     `gorm:"type:varchar(50)" json:"document_ref_item"`
	Status          string     `gorm:"type:varchar(50)" json:"status"`
	Weight          float64    `gorm:"type:numeric" json:"weight"`
	WeightUnit      float64    `gorm:"type:numeric" json:"weight_unit"`
	Remark          string     `gorm:"type:varchar(255)" json:"remark"`
	CreateDate      *time.Time `gorm:"type:date" json:"create_date"`
	CreateBy        string     `gorm:"type:varchar(50)" json:"create_by"`
	UpdateDate      *time.Time `gorm:"type:date" json:"update_date"`
	UpdateBy        string     `gorm:"type:varchar(50)" json:"update_by"`
}

type ResultDeliveryResponse struct {
	Total      int                   `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalPages int                   `json:"total_pages"`
	Deliveries []GetDeliveryResponse `json:"deliveries"`
}

// buildStatusConditions สร้างเงื่อนไขการกรองตาม status ที่ซับซ้อน
func buildStatusConditions(statusFilters []string) ([]string, []interface{}) {
	var conditions []string
	var args []interface{}

	for _, statusFilter := range statusFilters {
		switch strings.ToLower(statusFilter) {
		case "new":
			// status='PENDING'
			conditions = append(conditions, "delivery_booking.status = ?")
			args = append(args, "PENDING")
		case "canceled", "cancelled":
			// status='CANCELED'
			conditions = append(conditions, "delivery_booking.status = ?")
			args = append(args, "CANCELED")
		case "draft":
			// status='TEMP'
			conditions = append(conditions, "delivery_booking.status = ?")
			args = append(args, "TEMP")
		case "completed":
			// status='COMPLETED'
			conditions = append(conditions, "delivery_booking.status = ?")
			args = append(args, "COMPLETED")
		}
	}

	return conditions, args
}

// paymentFilterValues คือ 2 ค่าที่ payment_filter รับ ("cash" กับ "non-cash")
var paymentFilterValues = map[string]bool{
	"cash":     true,
	"non-cash": true,
}

// buildPaymentFilterCondition แปลง payment_filter เป็นเงื่อนไข EXISTS/NOT EXISTS บนตาราง sale
// (คีย์ด้วย sale.sale_code = delivery_booking.document_ref)
//
// หมายเหตุ: query กับ countQuery มี LEFT JOIN sale อยู่แล้ว (บรรทัด 352 และ 556) และ sale_code เป็น
// unique index จึงไม่มีปัญหาแถวซ้ำ จะเขียนเงื่อนไขบน sale.payment_method ที่ join มาแล้วก็ได้เหมือนกัน
// ที่เลือก EXISTS เพราะตัวเงื่อนไขไม่ผูกกับ join นั้น ถ้าวันหน้ามีคนถอดหรือแก้ join (มันมีไว้ให้ตัวกรอง
// อื่น) ตัวกรองนี้ยังทำงานถูกอยู่ และ NOT EXISTS อ่านง่ายกว่าเวลาต้องครอบคลุมแถวที่ join ไม่เจอ
// ค่าถูก parameterize ผ่าน ? ไม่ interpolate ลงในสตริง
//
//   - "cash": ต้องมี sale ที่ sale.payment_method = 'CASH' เป๊ะๆ -> EXISTS
//   - "non-cash": ทุกอย่างที่ไม่ใช่ cash ข้างบน ครอบคลุม payment_method อื่น, ว่าง/null, และ booking ที่
//     document_ref ไม่ match sale แถวไหนเลย -> NOT EXISTS (... payment_method = 'CASH') ตัวเดียวครอบคลุม
//     ครบทุกกรณีในคำสั่งเดียว โดยไม่ต้องแจกแจงเป็นเงื่อนไขย่อย
//
// หมายเหตุ (จงใจต่างจาก normalizePickPackFilter/impliedDeliveryStatuses): "cash" กับ "non-cash" คือ
// ทุกความเป็นไปได้ (exhaustive) ของ booking ทุกแถว ไม่เหมือน pick_pack_filter ที่ค่าที่ส่งมาอาจ "พิมพ์ผิด"
// แล้วต้องตีความว่าตั้งใจกรองแต่กรองแล้วไม่มีอะไร match (คืนว่างทั้งหน้า) เพราะค่าที่รู้จักเป็นแค่ส่วนหนึ่ง
// ของ status ที่เป็นไปได้ทั้งหมด ที่นี่ตรงกันข้าม: ค่าที่ไม่รู้จักล้วน (พิมพ์ผิดทั้งหมด) หรือไม่ส่งมาเลย
// กับการเลือกครบทั้ง "cash" และ "non-cash" ล้วนแปลว่า "ทุกแถว" เหมือนกันทั้งคู่ ("ไม่มีตัวกรองที่ใช้ได้"
// กับ "เลือกครบทุกกลุ่ม" คือเซตเดียวกันเมื่อสองค่านี้ exhaustive) จึงต้องถือว่า "ไม่กรอง" เหมือนกัน ไม่ใช่
// คืนผลว่างแบบ pick_pack_filter
func buildPaymentFilterCondition(paymentFilters []string) (string, []interface{}) {
	seen := make(map[string]bool)
	for _, f := range paymentFilters {
		lf := strings.ToLower(f)
		if paymentFilterValues[lf] {
			seen[lf] = true
		}
	}

	hasCash := seen["cash"]
	hasNonCash := seen["non-cash"]

	switch {
	case hasCash && !hasNonCash:
		return "EXISTS (SELECT 1 FROM sale WHERE sale.sale_code = delivery_booking.document_ref AND sale.payment_method = ?)", []interface{}{"CASH"}
	case hasNonCash && !hasCash:
		return "NOT EXISTS (SELECT 1 FROM sale WHERE sale.sale_code = delivery_booking.document_ref AND sale.payment_method = ?)", []interface{}{"CASH"}
	default:
		// ทั้งสองค่า หรือไม่มีค่าไหนใช้ได้เลย (ว่าง/ไม่รู้จักล้วน) -> ไม่กรอง (ดูหมายเหตุด้านบน)
		return "", nil
	}
}

// pickPackFilterValues คือ 6 ค่าที่ ComputePickPackStatus คืนได้ (ดู pick-pack-status.go)
var pickPackFilterValues = map[string]bool{
	PickPackStatusDraft:       true,
	PickPackStatusCanceled:    true,
	PickPackStatusCompleted:   true,
	PickPackStatusNew:         true,
	PickPackStatusPendingPick: true,
	PickPackStatusPendingPack: true,
}

// normalizePickPackFilter กรองค่าที่ไม่รู้จักออกจาก pick_pack_filter เหลือแต่ค่าที่ใช้กรองได้จริง
// (ตัวพิมพ์เล็ก ไม่ซ้ำ) ตามที่ระบุว่า "ignore unknown values"
func normalizePickPackFilter(filter []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, f := range filter {
		lf := strings.ToLower(f)
		if pickPackFilterValues[lf] && !seen[lf] {
			seen[lf] = true
			result = append(result, lf)
		}
	}
	return result
}

// impliedDeliveryStatuses แปลงค่ากรอง pick_pack_filter (ที่ normalize แล้ว) เป็นชุด delivery_booking.status
// ที่เป็นไปได้ เพื่อใช้ตีกรอบ query ฝั่ง DB ก่อน แล้วค่อยกรองละเอียดในหน่วยความจำ:
// draft->TEMP, canceled->CANCELED, completed->COMPLETED, new/pending-pick/pending-pack->PENDING
func impliedDeliveryStatuses(validPickPackFilter []string) []string {
	seen := make(map[string]bool)
	var statuses []string
	for _, f := range validPickPackFilter {
		var status string
		switch f {
		case PickPackStatusDraft:
			status = "TEMP"
		case PickPackStatusCanceled:
			status = "CANCELED"
		case PickPackStatusCompleted:
			status = "COMPLETED"
		case PickPackStatusNew, PickPackStatusPendingPick, PickPackStatusPendingPack:
			status = "PENDING"
		}
		if status != "" && !seen[status] {
			seen[status] = true
			statuses = append(statuses, status)
		}
	}
	return statuses
}

// computePageBounds คำนวณขอบเขต slice (start, end) และจำนวนหน้าทั้งหมด จากจำนวนแถวที่ match
// ทั้งหมด, หน้าที่ขอ, และขนาดหน้า เป็น pure function ล้วนๆ แยกออกมาจาก path ที่กรองด้วย
// pick_pack_filter เพื่อให้ unit test เรียกตรงๆ ได้โดยไม่ต้องมี DB
//
//   - totalRecords <= 0: ไม่มีอะไรให้แบ่งหน้า คืน (0, 0, 0)
//   - page หรือ pageSize <= 0: เดิม (path ไม่ filter) ใช้สูตรคืนทั้งหมดเป็นหน้าเดียว คงพฤติกรรมนี้ไว้
//   - page เลยหน้าสุดท้ายไปแล้ว: คืน start=end=totalRecords (slice ว่างแต่ index ยังใช้ slicing ได้ปลอดภัย)
func computePageBounds(totalRecords, page, pageSize int) (start, end, totalPages int) {
	if totalRecords <= 0 {
		return 0, 0, 0
	}
	if pageSize <= 0 || page <= 0 {
		return 0, totalRecords, totalRecords
	}

	totalPages = int(math.Ceil(float64(totalRecords) / float64(pageSize)))
	start = (page - 1) * pageSize
	if start >= totalRecords {
		return totalRecords, totalRecords, totalPages
	}
	end = start + pageSize
	if end > totalRecords {
		end = totalRecords
	}
	return start, end, totalPages
}

// getCustomerCodesByName ค้นหา customer codes จาก customer service โดยใช้ customer name
func getCustomerCodesByName(customerNameLike string) ([]string, error) {
	if len(customerNameLike) == 0 {
		return nil, nil
	}

	getCustomerByNameRequest := externalService.GetCustomerRequest{
		CustomerNameLike: customerNameLike,
		Page:             1,
		PageSize:         1000, // เอาเยอะๆ เพื่อให้ได้ customerCode ทั้งหมดที่ match
	}

	customerByNameData, err := externalService.GetCustomer(getCustomerByNameRequest)
	if err != nil {
		fmt.Println("failed to fetch customers by name:", err)
		return nil, errors.New("failed to fetch customers by name: " + err.Error())
	}

	fmt.Printf("Found %d customers matching name like '%s'\n", len(customerByNameData.Customers), customerNameLike)

	// เก็บ customerCode ทั้งหมดที่ได้จากการค้นหาด้วย name
	var customerCodes []string
	for _, customer := range customerByNameData.Customers {
		customerCodes = append(customerCodes, customer.CustomerCode)
	}

	fmt.Println("Customer codes from name search:", customerCodes)
	return customerCodes, nil
}

// GetOrderDeliveryForDelivery ฟังก์ชันสำหรับเรียก GetOrdersDelivery สำหรับ GetDeliveryResponse
func GetOrderDeliveryForDelivery(allDeliveries []GetDeliveryResponse) (orderExternalService.ResultOrderDeliveryResponse, error) {
	getOrderRequest := orderExternalService.GetOrderDeliveryRequest{}
	for _, row := range allDeliveries {
		getOrderRequest.DeliveryCode = append(getOrderRequest.DeliveryCode, row.DeliveryCode)

		for _, item := range row.Items {
			getOrderRequest.DeliveryItem = append(getOrderRequest.DeliveryItem, item.DeliveryItem)
		}
	}

	fmt.Println("getOrderRequest : ", getOrderRequest)
	getOrderResponse, err := orderExternalService.GetOrdersDelivery(getOrderRequest)
	if err != nil {
		return orderExternalService.ResultOrderDeliveryResponse{}, errors.New("Error get orders delivery : " + err.Error())
	}
	fmt.Println("getOrderResponse : ", getOrderResponse)

	return getOrderResponse, nil
}

func GetDelivery(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var res []GetDeliveryResponse
	var req GetDeliveryRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {

		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to database"})
		return nil, err
	}
	defer db.CloseGORM(gormx)

	// ถ้ามี CustomerNameLike ให้ไปค้นหา customerCode จาก customer service ก่อน
	customerCodesFromName, err := getCustomerCodesByName(req.CustomerNameLike)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return nil, err
	}

	query := gormx.Select("delivery_booking.*, time.name as delivery_time_name").
		Joins("LEFT JOIN time ON delivery_booking.delivery_time_code = time.code").
		Joins("LEFT JOIN sale ON delivery_booking.document_ref = sale.sale_code").
		Preload("Items").
		Preload("SaleOrder").
		Preload("SaleOrder.SaleItem").
		Order("delivery_booking.update_date DESC")

	if len(req.ID) > 0 {
		query = query.Where("delivery_booking.id IN ?", req.ID)
	}

	if len(req.DeliveryCode) > 0 {
		query = query.Where("delivery_booking.delivery_code IN ?", req.DeliveryCode)
	}

	if len(req.NotInDeliveryCode) > 0 {
		query = query.Where("delivery_booking.delivery_code NOT IN ?", req.NotInDeliveryCode)
	}

	if len(req.SaleOrderCode) > 0 {
		query = query.Where("delivery_booking.document_ref IN ?", req.SaleOrderCode)
	}

	if len(req.SiteCode) > 0 {
		query = query.Where("delivery_booking.site_code IN ?", req.SiteCode)
	}

	if len(req.CompanyCode) > 0 {
		query = query.Where("delivery_booking.company_code IN ?", req.CompanyCode)
	}

	if len(req.Status) > 0 {
		query = query.Where("delivery_booking.status IN ?", req.Status)
	}

	if len(req.CustomerCode) > 0 {
		query = query.Where("delivery_booking.customer_code IN ?", req.CustomerCode)
	}

	// Like filters
	if len(req.DeliveryCodeLike) > 0 {
		query = query.Where("delivery_booking.delivery_code ILIKE ?", "%"+req.DeliveryCodeLike+"%")
	}

	if len(req.DocumentRefLike) > 0 {
		query = query.Where("delivery_booking.document_ref ILIKE ?", "%"+req.DocumentRefLike+"%")
	}

	if len(req.ProductCodeLike) > 0 {
		query = query.Where("EXISTS (SELECT 1 FROM delivery_booking_item dbi WHERE dbi.delivery_id = delivery_booking.id AND dbi.product_code ILIKE ?)", "%"+req.ProductCodeLike+"%")
	}

	if len(req.CustomerCodeLike) > 0 {
		query = query.Where("delivery_booking.customer_code ILIKE ?", "%"+req.CustomerCodeLike+"%")
	}

	if len(req.CustomerNameLike) > 0 {
		// ใช้ customerCode ที่ได้จากการค้นหาด้วย customer name แทนการค้นหาตรงๆ ด้วย customer_name
		if len(customerCodesFromName) > 0 {
			query = query.Where("delivery_booking.customer_code IN ?", customerCodesFromName)
		} else {
			// ถ้าไม่เจอ customer ใดๆ ที่ match กับ name ให้ return ข้อมูลว่างเปล่า
			query = query.Where("1 = 0") // condition ที่จะไม่ match อะไรเลย
		}
	}

	if len(req.ShipToAddressLike) > 0 {
		query = query.Where("delivery_booking.ship_to_address ILIKE ?", "%"+req.ShipToAddressLike+"%")
	}

	if len(req.LicensePlateLike) > 0 {
		query = query.Where("delivery_booking.license_plate ILIKE ?", "%"+req.LicensePlateLike+"%")
	}

	if len(req.ContactNameLike) > 0 {
		query = query.Where("delivery_booking.contact_name ILIKE ?", "%"+req.ContactNameLike+"%")
	}

	if len(req.DeliveryTimeNameLike) > 0 {
		query = query.Where("time.name ILIKE ?", "%"+req.DeliveryTimeNameLike+"%")
	}

	// Date range filters
	if req.CompleteDateStart != nil && req.CompleteDateEnd != nil {
		query = query.Where("delivery_booking.update_date BETWEEN ? AND ? AND delivery_booking.status = ?", req.CompleteDateStart, req.CompleteDateEnd, "COMPLETED")
	}

	if req.CreateDateStart != nil && req.CreateDateEnd != nil {
		query = query.Where("delivery_booking.create_date BETWEEN ? AND ?", req.CreateDateStart, req.CreateDateEnd)
	}

	if req.SaleOrderCreateDateStart != nil && req.SaleOrderCreateDateEnd != nil {
		query = query.Where("sale.create_date BETWEEN ? AND ?", req.SaleOrderCreateDateStart, req.SaleOrderCreateDateEnd)
	}

	if req.DeliveryDateStart != nil && req.DeliveryDateEnd != nil {
		query = query.Where("sale.delivery_date BETWEEN ? AND ?", req.DeliveryDateStart, req.DeliveryDateEnd)
	}

	if req.ShipSlotDateStart != nil && req.ShipSlotDateEnd != nil {
		query = query.Where("delivery_booking.delivery_date BETWEEN ? AND ?", req.ShipSlotDateStart, req.ShipSlotDateEnd)
	}

	// Apply status filter conditions
	if len(req.StatusFilter) > 0 {
		conditions, args := buildStatusConditions(req.StatusFilter)
		if len(conditions) > 0 {
			// Join conditions with OR
			combinedCondition := "(" + strings.Join(conditions, " OR ") + ")"
			query = query.Where(combinedCondition, args...)
		}
	}

	// payment_filter: AND กับ status_filter/pick_pack_filter ด้านบน/ล่าง เนื่องจาก query ตัวนี้ถูกใช้ร่วมกัน
	// ทั้ง path ไม่กรอง (ด้านล่าง) และ path pick_pack_filter (ด้านล่างถัดไป) การเติมเงื่อนไขตรงนี้จุดเดียว
	// จึงครอบคลุมทั้งสอง path โดยไม่ต้องเติมซ้ำ
	if condition, args := buildPaymentFilterCondition(req.PaymentFilter); condition != "" {
		query = query.Where(condition, args...)
	}

	// pick_pack_filter: กรอง delivery ตาม pick/pack status ที่คำนวณจากข้อมูล order (in-memory)
	// เพราะ status นี้ไม่ได้เก็บใน DB ตรงๆ ต้องดึงข้อมูลมาคำนวณก่อนถึงจะรู้
	//
	// เช็คจาก req.PickPackFilter (ค่าดิบที่ผู้เรียกส่งมา) ไม่ใช่ผลจาก normalizePickPackFilter เพื่อแยก
	// สองกรณีออกจากกันให้ชัด: (1) ไม่ได้ส่ง pick_pack_filter มาเลย (req.PickPackFilter ว่าง) ต้องตกไป
	// path เดิมด้านล่างที่ไม่กรองอะไรตาม pick/pack status; (2) ส่ง pick_pack_filter มาแต่ค่าที่ส่งมา
	// ไม่มีตัวไหนรู้จักเลย (เช่น พิมพ์ "cancelled" ซึ่ง pick_pack_filter รองรับแค่ "canceled" ต่างจาก
	// status_filter ที่รองรับทั้งสองสะกด) ต้องตีความว่าผู้เรียก "ตั้งใจจะกรอง" แต่กรองแล้วไม่มีอะไร match
	// เลย จึงต้องคืนหน้าว่าง total=0 ไม่ใช่ตกไป path ไม่กรองซึ่งจะคืนข้อมูลทั้งตารางกลับไปแทน
	if len(req.PickPackFilter) > 0 {
		validPickPackFilter := normalizePickPackFilter(req.PickPackFilter)
		if len(validPickPackFilter) == 0 {
			return ResultDeliveryResponse{
				Total:      0,
				Page:       req.Page,
				PageSize:   req.PageSize,
				TotalPages: 0,
				Deliveries: []GetDeliveryResponse{},
			}, nil
		}

		// ตีกรอบ query ด้วย delivery status ที่ pick_pack_filter เป็นไปได้ก่อน (AND กับเงื่อนไขอื่นๆ
		// รวมถึง status_filter ด้านบน) เพื่อไม่ให้ต้องดึงทั้งตารางมากรองในหน่วยความจำ
		query = query.Where("delivery_booking.status IN ?", impliedDeliveryStatuses(validPickPackFilter))

		var allRows []GetDeliveryResponse
		if err := query.Find(&allRows).Error; err != nil {
			fmt.Println(err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve data"})
			return nil, err
		}

		// GetOrderDelivery สำหรับทั้งชุดที่ตีกรอบไว้ (ไม่ใช่แค่หน้าเดียว) เพราะต้องคำนวณ
		// pick_pack_status ให้ครบก่อนถึงจะกรอง+แบ่งหน้าในหน่วยความจำได้
		orderDeliveryResponse, err := GetOrderDeliveryForDelivery(allRows)
		if err != nil {
			fmt.Println("Error in GetOrderDelivery:", err)
			// ต่างจาก path ไม่กรอง (ด้านล่าง) ที่ปล่อยผ่านได้เพราะข้อมูล order เป็นแค่ของตกแต่งหน้าจอ
			// เท่านั้น แต่ที่นี่ order data ถูกเอาไปคำนวณ pick_pack_status จริง ถ้าดึงไม่ได้ ทุกแถวที่
			// status='PENDING' จะกลายเป็น "new" ไปหมด (order ว่าง -> new ตาม ComputePickPackStatus ข้อ
			// 4/6) ทำให้คนกรองหา pending-pick/pending-pack เห็นรายการว่างหรือขาดหายไปแบบเงียบๆ พร้อม
			// 200 OK ทั้งที่ order-service กำลังล่มอยู่ ต้องคืน error ให้ request ทั้งก้อนล้มแทนที่จะตอบ
			// ข้อมูลที่คำนวณมาจากข้อมูล order ที่หายไป
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve order data for pick/pack filter: " + err.Error()})
			return nil, err
		}

		orderMap := make(map[string]orderExternalService.GetOrderDeliveryResponse)
		for _, order := range orderDeliveryResponse.Orders {
			orderMap[order.DocumentRef] = order
		}
		for i := range allRows {
			if matchingOrder, exists := orderMap[allRows[i].DeliveryCode]; exists {
				allRows[i].Order = matchingOrder
			}
		}

		filterSet := make(map[string]bool, len(validPickPackFilter))
		for _, f := range validPickPackFilter {
			filterSet[f] = true
		}

		var filtered []GetDeliveryResponse
		for i := range allRows {
			allRows[i].PickPackStatus = ComputePickPackStatus(allRows[i].Status, allRows[i].Order)
			if filterSet[allRows[i].PickPackStatus] {
				filtered = append(filtered, allRows[i])
			}
		}

		totalRecords := len(filtered)
		start, end, totalPages := computePageBounds(totalRecords, req.Page, req.PageSize)

		return ResultDeliveryResponse{
			Total:      totalRecords,
			Page:       req.Page,
			PageSize:   req.PageSize,
			TotalPages: totalPages,
			Deliveries: filtered[start:end],
		}, nil
	}

	// Build base query for counting
	countQuery := gormx.Model(&GetDeliveryResponse{}).
		Joins("LEFT JOIN time ON delivery_booking.delivery_time_code = time.code").
		Joins("LEFT JOIN sale ON delivery_booking.document_ref = sale.sale_code")

	if len(req.ID) > 0 {
		countQuery = countQuery.Where("delivery_booking.id IN ?", req.ID)
	}

	if len(req.DeliveryCode) > 0 {
		countQuery = countQuery.Where("delivery_booking.delivery_code IN ?", req.DeliveryCode)
	}

	if len(req.NotInDeliveryCode) > 0 {
		countQuery = countQuery.Where("delivery_booking.delivery_code NOT IN ?", req.NotInDeliveryCode)
	}

	if len(req.ProductCodeLike) > 0 {
		countQuery = countQuery.Where("EXISTS (SELECT 1 FROM delivery_booking_item dbi WHERE dbi.delivery_id = delivery_booking.id AND dbi.product_code ILIKE ?)", "%"+req.ProductCodeLike+"%")
	}

	if len(req.SaleOrderCode) > 0 {
		countQuery = countQuery.Where("delivery_booking.document_ref IN ?", req.SaleOrderCode)
	}

	if len(req.SiteCode) > 0 {
		countQuery = countQuery.Where("delivery_booking.site_code IN ?", req.SiteCode)
	}

	if len(req.CompanyCode) > 0 {
		countQuery = countQuery.Where("delivery_booking.company_code IN ?", req.CompanyCode)
	}

	if len(req.Status) > 0 {
		countQuery = countQuery.Where("delivery_booking.status IN ?", req.Status)
	}

	if len(req.CustomerCode) > 0 {
		countQuery = countQuery.Where("delivery_booking.customer_code IN ?", req.CustomerCode)
	}

	// Apply same like filters to count query
	if len(req.DeliveryCodeLike) > 0 {
		countQuery = countQuery.Where("delivery_booking.delivery_code ILIKE ?", "%"+req.DeliveryCodeLike+"%")
	}

	if len(req.DocumentRefLike) > 0 {
		countQuery = countQuery.Where("delivery_booking.document_ref ILIKE ?", "%"+req.DocumentRefLike+"%")
	}

	if len(req.CustomerCodeLike) > 0 {
		countQuery = countQuery.Where("delivery_booking.customer_code ILIKE ?", "%"+req.CustomerCodeLike+"%")
	}

	if len(req.CustomerNameLike) > 0 {
		if len(customerCodesFromName) > 0 {
			countQuery = countQuery.Where("delivery_booking.customer_code IN ?", customerCodesFromName)
		} else {
			countQuery = countQuery.Where("1 = 0")
		}
	}

	if len(req.ShipToAddressLike) > 0 {
		countQuery = countQuery.Where("delivery_booking.ship_to_address ILIKE ?", "%"+req.ShipToAddressLike+"%")
	}

	if len(req.LicensePlateLike) > 0 {
		countQuery = countQuery.Where("delivery_booking.license_plate ILIKE ?", "%"+req.LicensePlateLike+"%")
	}

	if len(req.ContactNameLike) > 0 {
		countQuery = countQuery.Where("delivery_booking.contact_name ILIKE ?", "%"+req.ContactNameLike+"%")
	}

	if len(req.DeliveryTimeNameLike) > 0 {
		countQuery = countQuery.Where("time.name ILIKE ?", "%"+req.DeliveryTimeNameLike+"%")
	}

	// Apply same date range filters to count query
	if req.CompleteDateStart != nil && req.CompleteDateEnd != nil {
		countQuery = countQuery.Where("delivery_booking.update_date BETWEEN ? AND ? AND delivery_booking.status = ?", req.CompleteDateStart, req.CompleteDateEnd, "COMPLETED")
	}

	if req.CreateDateStart != nil && req.CreateDateEnd != nil {
		countQuery = countQuery.Where("delivery_booking.create_date BETWEEN ? AND ?", req.CreateDateStart, req.CreateDateEnd)
	}

	if req.SaleOrderCreateDateStart != nil && req.SaleOrderCreateDateEnd != nil {
		countQuery = countQuery.Where("sale.create_date BETWEEN ? AND ?", req.SaleOrderCreateDateStart, req.SaleOrderCreateDateEnd)
	}

	if req.DeliveryDateStart != nil && req.DeliveryDateEnd != nil {
		countQuery = countQuery.Where("sale.delivery_date BETWEEN ? AND ?", req.DeliveryDateStart, req.DeliveryDateEnd)
	}

	if req.ShipSlotDateStart != nil && req.ShipSlotDateEnd != nil {
		countQuery = countQuery.Where("delivery_booking.delivery_date BETWEEN ? AND ?", req.ShipSlotDateStart, req.ShipSlotDateEnd)
	}

	// Apply status filter conditions to count query
	if len(req.StatusFilter) > 0 {
		conditions, args := buildStatusConditions(req.StatusFilter)
		if len(conditions) > 0 {
			combinedCondition := "(" + strings.Join(conditions, " OR ") + ")"
			countQuery = countQuery.Where(combinedCondition, args...)
		}
	}

	// payment_filter: ต้องเติมเงื่อนไขเดียวกันกับ query (ด้านบน) เพื่อให้ total ตรงกับแถวที่ query จะคืนจริง
	// (path นี้คือ path ไม่มี pick_pack_filter ซึ่งใช้ countQuery หา total แยกจาก query ที่ใช้ดึงข้อมูลหน้า)
	if condition, args := buildPaymentFilterCondition(req.PaymentFilter); condition != "" {
		countQuery = countQuery.Where(condition, args...)
	}

	var count int64
	countQuery.Count(&count)

	totalRecords := count
	totalPages := 0
	offset := (req.Page - 1) * req.PageSize
	if totalRecords > 0 {
		if req.PageSize > 0 && req.Page > 0 {
			query = query.Limit(req.PageSize).Offset(offset)
			totalPages = int(math.Ceil(float64(totalRecords) / float64(req.PageSize)))
		} else {
			query = query.Limit(int(totalRecords)).Offset(offset)
			totalPages = (int(totalRecords) / 1)
		}
	}

	if err := query.Find(&res).Error; err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve data"})
		return nil, err
	}

	// GetOrderDelivery
	orderDeliveryResponse, err := GetOrderDeliveryForDelivery(res)
	if err != nil {
		fmt.Println("Error in GetOrderDelivery:", err)
		// path นี้ (ไม่มี pick_pack_filter) ไม่ทำให้ request ทั้งก้อนล้มเมื่อ order-service เรียกไม่สำเร็จ
		// เพราะที่นี่ order data ใช้แค่ตกแต่งหน้าจอ (field "order" ใน response) และคำนวณ pick_pack_status
		// เพื่อแสดงผลเฉยๆ ไม่ได้เอาไปกรองแถวออก ต่างจาก path ที่มี pick_pack_filter (ด้านบน) ที่ order
		// data ผิดพลาดแล้วจะทำให้แถว PENDING กลายเป็น "new" ผิดๆ และหลุดออกจากผลการกรอง จึงต้อง fail
		// ดังๆ ที่นั่นแทน ส่วนที่นี่ยังปล่อยแถวไปแบบไม่มี order ผูกได้ตามเดิม
	} else {
		// Map orders from orderDeliveryResponse to delivery header
		// Create map for efficient lookup of orders by delivery_code
		orderMap := make(map[string]orderExternalService.GetOrderDeliveryResponse)

		for _, order := range orderDeliveryResponse.Orders {
			// Map by order.DocumentRef to delivery_code
			orderMap[order.DocumentRef] = order
		}

		// Map orders to delivery header in res
		for i := range res {
			delivery := &res[i]

			// Try to find matching order by deliveryCode = order.DocumentRef
			if matchingOrder, exists := orderMap[delivery.DeliveryCode]; exists {
				delivery.Order = matchingOrder
			}
		}
	}

	// คำนวณ pick_pack_status ให้ทุกแถวเสมอ ไม่ว่าจะใช้ pick_pack_filter หรือไม่ (ทันทีหลังผูก order data)
	for i := range res {
		res[i].PickPackStatus = ComputePickPackStatus(res[i].Status, res[i].Order)
	}

	resultDelivery := ResultDeliveryResponse{
		Total:      int(totalRecords),
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		Deliveries: res,
	}

	return resultDelivery, nil
}
