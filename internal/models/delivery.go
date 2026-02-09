package models

import (
	"time"

	"github.com/google/uuid"
)

type Delivery struct {
	ID               uuid.UUID  `json:"id"`
	DeliveryCode     string     `json:"delivery_code"`
	CompanyCode      string     `json:"company_code"`
	SiteCode         string     `json:"site_code"`
	DeliveryMethod   string     `json:"delivery_method"`
	DocumentRef      string     `json:"document_ref"`
	CustomerCode     string     `json:"customer_code"`
	ShipToAddress    string     `json:"ship_to_address"`
	DeliveryDate     *time.Time `json:"delivery_date"`
	DeliveryTimeCode string     `json:"delivery_time_code"`
	LicensePlate     string     `json:"license_plate"`
	ContactName      string     `json:"contact_name"`
	Tel              string     `json:"tel"`
	TotalWeight      float64    `json:"total_weight"`
	Remark           string     `json:"remark"`
	Status           string     `json:"status"`
	BookingSlotType  string     `json:"booking_slot_type"`
	StatusPayment    string     `json:"status_payment"`
	CreateDate       time.Time  `json:"create_date"`
	CreateBy         string     `json:"create_by"`
	UpdateDate       time.Time  `json:"update_date"`
	UpdateBy         string     `json:"update_by"`
}

func (Delivery) TableName() string { return "delivery_booking" }

type DeliveryItem struct {
	ID              uuid.UUID `json:"id"`
	DeliveryItem    string    `json:"delivery_item"`
	DeliveryID      uuid.UUID `json:"delivery_id"`
	ProductCode     string    `json:"product_code"`
	Qty             float64   `json:"qty"`
	UnitCode        string    `json:"unit_code"`
	PriceListUnit   float64   `json:"price_list_unit"`
	SaleQty         float64   `json:"sale_qty"`
	SaleUnitCode    string    `json:"sale_unit_code"`
	TotalWeight     float64   `json:"total_weight"`
	DocumentRefItem string    `json:"document_ref_item"`
	Status          string    `json:"status"`
	Weight          float64   `json:"weight"`
	WeightUnit      float64   `json:"weight_unit"`
	Remark          string    `json:"remark"`
	CreateDate      time.Time `json:"create_date"`
	CreateBy        string    `json:"create_by"`
	UpdateDate      time.Time `json:"update_date"`
	UpdateBy        string    `json:"update_by"`
}

func (DeliveryItem) TableName() string { return "delivery_booking_item" }
