package deliveryService

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	externalService "prime-erp-core/external/customer-service"
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
	Status                   []string   `json:"status"`
	DeliveryCodeLike         string     `json:"delivery_code_like"`
	DocumentRefLike          string     `json:"document_ref_like"`
	SaleOrderCreateDateStart *time.Time `json:"sale_order_create_date_start"`
	SaleOrderCreateDateEnd   *time.Time `json:"sale_order_create_date_end"`
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
	Page                     int        `json:"page"`
	PageSize                 int        `json:"page_size"`
}

func (GetDeliveryResponse) TableName() string { return "delivery_booking" }

func (GetDeliveryItemResponse) TableName() string { return "delivery_booking_item" }

type GetDeliveryResponse struct {
	ID               uuid.UUID                 `gorm:"type:uuid;primary_key" json:"id"`
	DeliveryCode     string                    `gorm:"type:varchar(50)" json:"delivery_code"`
	CompanyCode      string                    `gorm:"type:varchar(50)" json:"company_code"`
	SiteCode         string                    `gorm:"type:varchar(50)" json:"site_code"`
	DeliveryMethod   string                    `gorm:"type:varchar(50)" json:"delivery_method"`
	DocumentRef      string                    `gorm:"type:varchar(50)" json:"document_ref"`
	CustomerCode     string                    `gorm:"type:varchar(50)" json:"customer_code"`
	ShipToAddress    string                    `gorm:"type:varchar(255)" json:"ship_to_address"`
	DeliveryDate     *time.Time                `gorm:"type:date" json:"delivery_date"`
	DeliveryTimeCode string                    `gorm:"type:varchar(50)" json:"delivery_time_code"`
	DeliveryTimeName string                    `gorm:"type:varchar(100)" json:"delivery_time_name"`
	LicensePlate     string                    `gorm:"type:varchar(50)" json:"license_plate"`
	ContactName      string                    `gorm:"type:varchar(100)" json:"contact_name"`
	Tel              string                    `gorm:"type:varchar(20)" json:"tel"`
	TotalWeight      float64                   `gorm:"type:numeric" json:"total_weight"`
	Status           string                    `gorm:"type:varchar(50)" json:"status"`
	BookingSlotType  string                    `gorm:"type:varchar(50)" json:"booking_slot_type"`
	Remark           string                    `gorm:"type:varchar(255)" json:"remark"`
	CreateDate       *time.Time                `gorm:"type:date" json:"create_date"`
	CreateBy         string                    `gorm:"type:varchar(50)" json:"create_by"`
	UpdateDate       *time.Time                `gorm:"type:date" json:"update_date"`
	UpdateBy         string                    `gorm:"type:varchar(50)" json:"update_by"`
	SaleOrder        models.Sale               `gorm:"foreignKey:DocumentRef;references:SaleCode" json:"sale_order"`
	Items            []GetDeliveryItemResponse `gorm:"foreignKey:DeliveryID" json:"items"`
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

	// Like filters
	if len(req.DeliveryCodeLike) > 0 {
		query = query.Where("delivery_booking.delivery_code ILIKE ?", "%"+req.DeliveryCodeLike+"%")
	}

	if len(req.DocumentRefLike) > 0 {
		query = query.Where("delivery_booking.document_ref ILIKE ?", "%"+req.DocumentRefLike+"%")
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

	resultDelivery := ResultDeliveryResponse{
		Total:      int(totalRecords),
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		Deliveries: res,
	}

	return resultDelivery, nil
}
