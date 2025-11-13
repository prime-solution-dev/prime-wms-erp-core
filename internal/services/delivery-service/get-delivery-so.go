package deliveryService

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetDeliverySORequest struct {
	DeliveryCode []string `json:"delivery_code"`
}

func (GetDeliverySOResponse) TableName() string { return "delivery_booking" }

func (GetDeliveryItemSOResponse) TableName() string { return "delivery_booking_item" }

type GetDeliverySOResponse struct {
	ID               uuid.UUID                   `gorm:"type:uuid;primary_key" json:"id"`
	DeliveryCode     string                      `gorm:"type:varchar(50)" json:"delivery_code"`
	CompanyCode      string                      `gorm:"type:varchar(50)" json:"company_code"`
	SiteCode         string                      `gorm:"type:varchar(50)" json:"site_code"`
	DeliveryMethod   string                      `gorm:"type:varchar(50)" json:"delivery_method"`
	DocumentRef      string                      `gorm:"type:varchar(50)" json:"document_ref"`
	CustomerCode     string                      `gorm:"type:varchar(50)" json:"customer_code"`
	ShipToAddress    string                      `gorm:"type:varchar(255)" json:"ship_to_address"`
	DeliveryDate     *time.Time                  `gorm:"type:date" json:"delivery_date"`
	DeliveryTimeCode string                      `gorm:"type:varchar(50)" json:"delivery_time_code"`
	DeliveryTimeName string                      `gorm:"type:varchar(100)" json:"delivery_time_name"`
	LicensePlate     string                      `gorm:"type:varchar(50)" json:"license_plate"`
	ContactName      string                      `gorm:"type:varchar(100)" json:"contact_name"`
	Tel              string                      `gorm:"type:varchar(20)" json:"tel"`
	TotalWeight      float64                     `gorm:"type:numeric" json:"total_weight"`
	Status           string                      `gorm:"type:varchar(50)" json:"status"`
	Remark           string                      `gorm:"type:varchar(255)" json:"remark"`
	BookingSlotType  string                      `gorm:"type:varchar(50)" json:"booking_slot_type"`
	CreateDate       *time.Time                  `gorm:"type:date" json:"create_date"`
	CreateBy         string                      `gorm:"type:varchar(50)" json:"create_by"`
	UpdateDate       *time.Time                  `gorm:"type:date" json:"update_date"`
	UpdateBy         string                      `gorm:"type:varchar(50)" json:"update_by"`
	Items            []GetDeliveryItemSOResponse `gorm:"foreignKey:DeliveryID" json:"items"`
}

type GetDeliveryItemSOResponse struct {
	ID              uuid.UUID   `gorm:"type:uuid;primary_key" json:"id"`
	DeliveryItem    string      `gorm:"type:varchar(50)" json:"delivery_item"`
	DeliveryID      uuid.UUID   `gorm:"type:uuid" json:"delivery_id"`
	ProductCode     string      `gorm:"type:varchar(50)" json:"product_code"`
	Qty             float64     `gorm:"type:numeric" json:"qty"`
	UnitCode        string      `gorm:"type:varchar(20)" json:"unit_code"`
	PriceListUnit   float64     `gorm:"type:numeric" json:"price_list_unit"`
	SaleQty         float64     `gorm:"type:numeric" json:"sale_qty"`
	SaleUnitCode    string      `gorm:"type:varchar(20)" json:"sale_unit_code"`
	TotalWeight     float64     `gorm:"type:numeric" json:"total_weight"`
	DocumentRefItem string      `gorm:"type:varchar(50)" json:"document_ref_item"`
	Status          string      `gorm:"type:varchar(50)" json:"status"`
	Weight          float64     `gorm:"type:numeric" json:"weight"`
	WeightUnit      float64     `gorm:"type:numeric" json:"weight_unit"`
	CreateDate      *time.Time  `gorm:"type:date" json:"create_date"`
	CreateBy        string      `gorm:"type:varchar(50)" json:"create_by"`
	UpdateDate      *time.Time  `gorm:"type:date" json:"update_date"`
	UpdateBy        string      `gorm:"type:varchar(50)" json:"update_by"`
	Sale            models.Sale `gorm:"-" json:"sale"`
}

func GetDeliverySO(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var res []GetDeliverySOResponse
	var req GetDeliverySORequest

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
		Order("delivery_booking.update_date DESC")

	if len(req.DeliveryCode) > 0 {
		query = query.Where("delivery_code IN ?", req.DeliveryCode)
	}

	if err := query.Find(&res).Error; err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve data"})
		return nil, err
	}

	// Query and map sale information to delivery items
	for i := range res {
		delivery := &res[i]

		// Find sale by delivery.DocumentRef == sale.SaleCode
		var sale models.Sale
		if err := gormx.Where("sale_code = ?", delivery.DocumentRef).
			Preload("SaleItem").
			First(&sale).Error; err == nil {

			// Map sale to each delivery item and filter saleItems
			for j := range delivery.Items {
				deliveryItem := &delivery.Items[j]

				// Create a copy of sale for this delivery item
				itemSale := sale
				itemSale.SaleItem = []models.SaleItem{}

				// Filter saleItems: deliveryItem.documentRefItem == saleItem.saleItem
				for _, saleItem := range sale.SaleItem {
					if deliveryItem.DocumentRefItem == saleItem.SaleItem {
						itemSale.SaleItem = append(itemSale.SaleItem, saleItem)
					}
				}

				deliveryItem.Sale = itemSale
			}
		}
	}

	return res, nil
}
