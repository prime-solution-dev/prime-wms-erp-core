package deliveryService

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetDeliveryRequest struct {
	ID                []string `json:"id"`
	DeliveryCode      []string `json:"delivery_code"`
	NotInDeliveryCode []string `json:"not_in_delivery_code"`
	SaleOrderCode     []string `json:"sale_order_code"`
	SiteCode          []string `json:"site_code"`
	CompanyCode       []string `json:"company_code"`
	Status            []string `json:"status"`
	Page              int      `json:"page"`
	PageSize          int      `json:"page_size"`
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

	query := gormx.Select("delivery_booking.*, time.name as delivery_time_name").
		Joins("LEFT JOIN time ON delivery_booking.delivery_time_code = time.code").
		Preload("Items").
		Preload("SaleOrder").
		Preload("SaleOrder.SaleItem").
		Order("delivery_booking.update_date DESC")

	if len(req.ID) > 0 {
		query = query.Where("id IN ?", req.ID)
	}

	if len(req.DeliveryCode) > 0 {
		query = query.Where("delivery_code IN ?", req.DeliveryCode)
	}

	if len(req.NotInDeliveryCode) > 0 {
		query = query.Where("delivery_code NOT IN ?", req.NotInDeliveryCode)
	}

	if len(req.SaleOrderCode) > 0 {
		query = query.Where("document_ref IN ?", req.SaleOrderCode)
	}

	if len(req.SiteCode) > 0 {
		query = query.Where("site_code IN ?", req.SiteCode)
	}

	if len(req.CompanyCode) > 0 {
		query = query.Where("company_code IN ?", req.CompanyCode)
	}

	if len(req.Status) > 0 {
		query = query.Where("status IN ?", req.Status)
	}

	// Build base query for counting
	countQuery := gormx.Model(&GetDeliveryResponse{})

	if len(req.ID) > 0 {
		countQuery = countQuery.Where("id IN ?", req.ID)
	}

	if len(req.DeliveryCode) > 0 {
		countQuery = countQuery.Where("delivery_code IN ?", req.DeliveryCode)
	}

	if len(req.NotInDeliveryCode) > 0 {
		countQuery = countQuery.Where("delivery_code NOT IN ?", req.NotInDeliveryCode)
	}

	if len(req.SaleOrderCode) > 0 {
		countQuery = countQuery.Where("document_ref IN ?", req.SaleOrderCode)
	}

	if len(req.SiteCode) > 0 {
		countQuery = countQuery.Where("site_code IN ?", req.SiteCode)
	}

	if len(req.CompanyCode) > 0 {
		countQuery = countQuery.Where("company_code IN ?", req.CompanyCode)
	}

	if len(req.Status) > 0 {
		countQuery = countQuery.Where("status IN ?", req.Status)
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
