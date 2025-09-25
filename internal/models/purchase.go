package models

import (
	"time"

	"github.com/google/uuid"
)

type PrePurchase struct {
	ID               uuid.UUID         `gorm:"primary_key;not null" json:"id"`
	PrePurchaseCode  string            `gorm:"unique;not null" json:"pre_purchase_code"`
	PurchaseType     string            `json:"purchase_type"`
	CompanyCode      string            `json:"company_code"`
	SiteCode         string            `json:"site_code"`
	DocRefType       string            `json:"doc_ref_type"`
	DocRef           *uuid.UUID        `json:"doc_ref"`
	SupplierCode     string            `json:"supplier_code"`
	DeliveryAddress  string            `json:"delivery_address"`
	Status           string            `json:"status"`
	TotalAmount      float64           `json:"total_amount"`
	TotalWeight      float64           `json:"total_weight"`
	TotalDiscount    float64           `json:"total_discount"`
	TotalVat         float64           `json:"total_vat"`
	SubtotalExclVat  float64           `json:"subtotal_excl_vat"`
	IsApproved       bool              `json:"is_approved"`
	StatusApprove    string            `json:"status_approve"`
	Remark           string            `json:"remark"`
	CreateBy         string            `json:"create_by"`
	CreateDate       *time.Time        `gorm:"autoCreateTime;<-:create" json:"create_date"`
	UpdateBy         string            `json:"update_by"`
	UpdateDate       *time.Time        `gorm:"autoUpdateTime;<-" json:"update_date"`
	PrePurchaseItems []PrePurchaseItem `gorm:"foreignKey:PrePurchaseID;references:ID" json:"pre_purchase_items"`
	Purchases        []Purchase        `gorm:"foreignKey:DocRef;references:ID" json:"purchases"`
}

func (PrePurchase) TableName() string {
	return "pre_purchase"
}

type PrePurchaseItem struct {
	ID                   uuid.UUID  `gorm:"primary_key;not null" json:"id"`
	PrePurchaseID        uuid.UUID  `json:"pre_purchase_id"`
	PreItem              string     `json:"pre_item"`
	HierarchyType        string     `json:"hierarchy_type"` // Product Group ex. Group1
	HierarchyCode        string     `json:"hierarchy_code"` // Product Group Code ex. group 1 code
	DocRefItem           string     `json:"doc_ref_item"`
	Qty                  float64    `json:"qty"`
	Unit                 string     `json:"unit"`
	PurchaseQty          float64    `json:"purchase_qty"`
	PurchaseUnit         string     `json:"purchase_unit"`      // Unit <Pcs, Weight>
	PurchaseUnitType     string     `json:"purchase_unit_type"` // ex.KG-Spec, KG, PC
	PriceUnit            float64    `json:"price_unit"`         // ราคาต่อชิ้น => cost
	TotalDiscount        float64    `json:"total_discount"`
	TotalAmount          float64    `json:"total_amount"` // ราคารวม => total_cost - total_discount + total_vat
	UnitUOM              string     `json:"unit_uom"`     // UOM มีสองแบบคือ KG, PC  unit_uom field uom_code
	TotalCost            float64    `json:"total_cost"`   // QTY * price_unit
	TotalDiscountPercent float64    `json:"total_discount_percent"`
	TotalVat             float64    `json:"total_vat"`
	SubtotalExclVat      float64    `json:"subtotal_excl_vat"`
	WeightUnit           float64    `json:"weight_unit"`
	TotalWeight          float64    `json:"total_weight"`
	Status               string     `json:"status"`
	Remark               string     `json:"remark"`
	CreateDate           *time.Time `gorm:"autoCreateTime;<-:create" json:"create_date"`
	CreateBy             string     `json:"create_by"`
	UpdateDate           *time.Time `gorm:"autoUpdateTime;<-" json:"update_date"`
	UpdateBy             string     `json:"update_by"`
}

func (PrePurchaseItem) TableName() string {
	return "pre_purchase_item"
}

type Purchase struct {
	ID              uuid.UUID      `gorm:"primary_key;not null" json:"id"`
	PurchaseCode    string         `gorm:"unique;not null" json:"purchase_code"`
	PurchaseType    string         `json:"purchase_type"`
	CompanyCode     string         `json:"company_code"`
	SiteCode        string         `json:"site_code"`
	DocRefType      string         `json:"doc_ref_type"`
	DocRef          *uuid.UUID     `json:"doc_ref"`
	SupplierCode    string         `json:"supplier_code"`
	DeliveryDate    time.Time      `json:"delivery_date"`
	DeliveryAddress string         `json:"delivery_address"`
	Status          string         `json:"status"`
	TotalAmount     float64        `json:"total_amount"`
	TotalWeight     float64        `json:"total_weight"`
	TotalDiscount   float64        `json:"total_discount"`
	TotalVat        float64        `json:"total_vat"`
	SubtotalExclVat float64        `json:"subtotal_excl_vat"`
	IsApproved      bool           `json:"is_approved"`
	StatusApprove   string         `json:"status_approve"`
	Remark          string         `json:"remark"`
	CreateBy        string         `json:"create_by"`
	CreateDate      *time.Time     `gorm:"autoCreateTime;<-:create" json:"create_date"`
	UpdateBy        string         `json:"update_by"`
	UpdateDate      *time.Time     `gorm:"autoUpdateTime;<-" json:"update_date"`
	PurchaseItems   []PurchaseItem `gorm:"foreignKey:PurchaseID;references:ID" json:"purchase_items"`
}

func (Purchase) TableName() string {
	return "purchase"
}

type PurchaseItem struct {
	ID                   uuid.UUID  `gorm:"primary_key;not null" json:"id"`
	PurchaseID           uuid.UUID  `json:"purchase_id"`
	PurchaseItem         string     `json:"purchase_item"`
	ProductCode          string     `json:"product_code"`
	DocRefItem           string     `json:"doc_ref_item"`
	Qty                  float64    `json:"qty"`
	Unit                 string     `json:"unit"`
	PurchaseQty          float64    `json:"purchase_qty"`
	PurchaseUnit         string     `json:"purchase_unit"`      // Unit <Pcs, Weight>
	PurchaseUnitType     string     `json:"purchase_unit_type"` // ex.KG-Spec, KG, PC
	PriceUnit            float64    `json:"price_unit"`         // ราคาต่อชิ้น => cost
	TotalDiscount        float64    `json:"total_discount"`
	TotalAmount          float64    `json:"total_amount"` // ราคารวม => total_cost - total_discount + total_vat
	UnitUOM              string     `json:"unit_uom"`     // UOM มีสองแบบคือ KG, PC  unit_uom field uom_code
	TotalCost            float64    `json:"total_cost"`   // QTY * price_unit
	TotalDiscountPercent float64    `json:"total_discount_percent"`
	TotalVat             float64    `json:"total_vat"`
	SubtotalExclVat      float64    `json:"subtotal_excl_vat"`
	WeightUnit           float64    `json:"weight_unit"`
	TotalWeight          float64    `json:"total_weight"`
	Status               string     `json:"status"`
	Remark               string     `json:"remark"`
	CreateDate           *time.Time `gorm:"autoCreateTime;<-:create" json:"create_date"`
	CreateBy             string     `json:"create_by"`
	UpdateDate           *time.Time `gorm:"autoUpdateTime;<-" json:"update_date"`
	UpdateBy             string     `json:"update_by"`
}

func (PurchaseItem) TableName() string {
	return "purchase_item"
}

// DTOs
type CreatePOResponse struct {
	ID string `json:"id"`
}

type CreatePOBigLotItemRequest struct {
	PreItem              string  `json:"pre_item"`
	ProductGroupType     string  `json:"product_group_type"`
	ProductGroupCode     string  `json:"product_group_code"`
	DocRefItem           string  `json:"doc_ref_item"`
	Qty                  float64 `json:"qty"`
	Unit                 string  `json:"unit"`
	PurchaseQty          float64 `json:"purchase_qty"`
	PurchaseUnit         string  `json:"purchase_unit"`
	PurchaseUnitType     string  `json:"purchase_unit_type"`
	PriceUnit            float64 `json:"price_unit"`
	TotalDiscount        float64 `json:"total_discount"`
	TotalAmount          float64 `json:"total_amount"`
	UnitUOM              string  `json:"unit_uom"`
	TotalCost            float64 `json:"total_cost"`
	TotalDiscountPercent float64 `json:"total_discount_percent"`
	TotalVat             float64 `json:"total_vat"`
	SubtotalExclVat      float64 `json:"subtotal_excl_vat"`
	WeightUnit           float64 `json:"weight_unit"`
	TotalWeight          float64 `json:"total_weight"`
	Status               string  `json:"status"`
	Remark               string  `json:"remark"`
}

type CreatePOBigLotRequest struct {
	PrePurchaseCode string                      `json:"pre_purchase_code"`
	CompanyCode     string                      `json:"company_code"`
	SiteCode        string                      `json:"site_code"`
	SupplierCode    string                      `json:"supplier_code"`
	DeliveryAddress string                      `json:"delivery_address"`
	Status          string                      `json:"status"`
	TotalAmount     float64                     `json:"total_amount"`
	TotalWeight     float64                     `json:"total_weight"`
	TotalDiscount   float64                     `json:"total_discount"`
	TotalVat        float64                     `json:"total_vat"`
	SubtotalExclVat float64                     `json:"subtotal_excl_vat"`
	IsApproved      bool                        `json:"is_approved"`
	StatusApprove   string                      `json:"status_approve"`
	Remark          string                      `json:"remark"`
	Items           []CreatePOBigLotItemRequest `json:"items"`
}

type GetPOBigLotListRequest struct {
	PrePurchaseCodes []string `json:"pre_purchase_codes"`
	CompanyCode      string   `json:"company_code"`
	SiteCode         string   `json:"site_code"`
	Page             int      `json:"page"`
	PageSize         int      `json:"page_size"`
}

type GetPOBigLotItemResponse struct {
	ID                   string  `json:"id"`
	PrePurchaseID        string  `json:"pre_purchase_id"`
	ProductGroupType     string  `json:"product_group_type"`
	ProductGroupCode     string  `json:"product_group_code"`
	ProductGroupName     string  `json:"product_group_name"`
	Qty                  float64 `json:"qty"`
	Unit                 string  `json:"unit"`
	PurchaseQty          float64 `json:"purchase_qty"`
	PurchaseUnit         string  `json:"purchase_unit"`
	PurchaseUnitType     string  `json:"purchase_unit_type"`
	PriceUnit            float64 `json:"price_unit"`
	TotalDiscount        float64 `json:"total_discount"`
	TotalAmount          float64 `json:"total_amount"`
	UnitUOM              string  `json:"unit_uom"`
	TotalCost            float64 `json:"total_cost"`
	TotalDiscountPercent float64 `json:"total_discount_percent"`
	TotalVat             float64 `json:"total_vat"`
	SubtotalExclVat      float64 `json:"subtotal_excl_vat"`
	WeightUnit           float64 `json:"weight_unit"`
	TotalWeight          float64 `json:"total_weight"`
	Status               string  `json:"status"`
	Remark               string  `json:"remark"`
	CreateDate           string  `json:"create_date"`
	CreateBy             string  `json:"create_by"`
	UpdateDate           string  `json:"update_date"`
	UpdateBy             string  `json:"update_by"`
}

type GetPOBigLotResponse struct {
	ID               string                    `json:"id"`
	PrePurchaseCode  string                    `json:"pre_purchase_code"`
	PurchaseType     string                    `json:"purchase_type"`
	CompanyCode      string                    `json:"company_code"`
	SiteCode         string                    `json:"site_code"`
	SupplierCode     string                    `json:"supplier_code"`
	SupplierName     string                    `json:"supplier_name"`
	DeliveryAddress  string                    `json:"delivery_address"`
	Status           string                    `json:"status"`
	TotalAmount      float64                   `json:"total_amount"`
	TotalWeight      float64                   `json:"total_weight"`
	TotalDiscount    float64                   `json:"total_discount"`
	TotalVat         float64                   `json:"total_vat"`
	SubtotalExclVat  float64                   `json:"subtotal_excl_vat"`
	IsApproved       bool                      `json:"is_approved"`
	StatusApprove    string                    `json:"status_approve"`
	Remark           string                    `json:"remark"`
	CreateBy         string                    `json:"create_by"`
	CreateDate       string                    `json:"create_date"`
	UpdateBy         string                    `json:"update_by"`
	UpdateDate       string                    `json:"update_date"`
	PrePurchaseItems []GetPOBigLotItemResponse `json:"pre_purchase_items"`
}

type GetPOBigLotListResponse struct {
	Total      int                   `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalPages int                   `json:"total_pages"`
	BigLotList []GetPOBigLotResponse `json:"big_lot_list"`
}

type UpdatePOBigLotItemRequest struct {
	ID                   *uuid.UUID `json:"id"`
	PrePurchaseID        uuid.UUID  `json:"pre_purchase_id"`
	ProductGroupType     string     `json:"product_group_type"`
	ProductGroupCode     string     `json:"product_group_code"`
	Qty                  float64    `json:"qty"`
	Unit                 string     `json:"unit"`
	PurchaseQty          float64    `json:"purchase_qty"`
	PurchaseUnit         string     `json:"purchase_unit"`
	PurchaseUnitType     string     `json:"purchase_unit_type"`
	PriceUnit            float64    `json:"price_unit"`
	TotalDiscount        float64    `json:"total_discount"`
	TotalAmount          float64    `json:"total_amount"`
	UnitUOM              string     `json:"unit_uom"`
	TotalCost            float64    `json:"total_cost"`
	TotalDiscountPercent float64    `json:"total_discount_percent"`
	TotalVat             float64    `json:"total_vat"`
	SubtotalExclVat      float64    `json:"subtotal_excl_vat"`
	WeightUnit           float64    `json:"weight_unit"`
	TotalWeight          float64    `json:"total_weight"`
	Status               string     `json:"status"`
	Remark               string     `json:"remark"`
}

type UpdatePOBigLotRequest struct {
	ID               uuid.UUID                   `json:"id"`
	Status           string                      `json:"status"`
	TotalAmount      float64                     `json:"total_amount"`
	TotalWeight      float64                     `json:"total_weight"`
	TotalDiscount    float64                     `json:"total_discount"`
	TotalVat         float64                     `json:"total_vat"`
	SubtotalExclVat  float64                     `json:"subtotal_excl_vat"`
	IsApproved       bool                        `json:"is_approved"`
	StatusApprove    string                      `json:"status_approve"`
	PrePurchaseItems []UpdatePOBigLotItemRequest `json:"pre_purchase_items"`
}

// Supplier DTOs
type GetSupplierListRequest struct {
	SupplierCodes []string `json:"supplier_code"`
}

type GetSupplierListResponse struct {
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
	Supplier   []Supplier `json:"supplier"`
}

type Supplier struct {
	ID           uuid.UUID `json:"id"`
	SupplierCode string    `json:"supplier_code"`
	SupplierName string    `json:"supplier_name"`
	CreateBy     string    `json:"create_by"`
	CreateDtm    time.Time `json:"create_dtm"`
	UpdateBy     string    `json:"update_by"`
	UpdateDtm    time.Time `json:"update_dtm"`
}
