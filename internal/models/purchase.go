package models

import (
	"time"

	"github.com/google/uuid"
)

type PrePurchase struct {
	ID                          uuid.UUID         `gorm:"primary_key;not null" json:"id"`
	PrePurchaseCode             string            `gorm:"unique;not null" json:"pre_purchase_code"`
	PurchaseType                string            `json:"purchase_type"`
	CompanyCode                 string            `json:"company_code"`
	SiteCode                    string            `json:"site_code"`
	DocRefType                  string            `json:"doc_ref_type"`
	DocRef                      string            `json:"doc_ref"`
	SupplierCode                string            `json:"supplier_code"`
	SupplierName                string            `json:"supplier_name"`
	SupplierAddress             string            `json:"supplier_address"`
	SupplierPhone               string            `json:"supplier_phone"`
	SupplierEmail               string            `json:"supplier_email"`
	DeliveryAddress             string            `json:"delivery_address"`
	Status                      string            `json:"status"`
	TotalAmount                 float64           `json:"total_amount"`
	TotalWeight                 float64           `json:"total_weight"`
	TotalDiscount               float64           `json:"total_discount"`
	TotalVat                    float64           `json:"total_vat"`
	SubtotalExclVat             float64           `json:"subtotal_excl_vat"`
	SubtotalExclDiscountExclVat float64           `json:"subtotal_excl_discount_excl_vat"`
	IsApproved                  bool              `json:"is_approved"`
	StatusApprove               string            `json:"status_approve"`
	Remark                      string            `json:"remark"`
	CreateBy                    string            `json:"create_by"`
	CreateDtm                   time.Time         `json:"create_dtm"`
	UpdateBy                    string            `json:"update_by"`
	UpdateDtm                   time.Time         `json:"update_dtm"`
	PrePurchaseItems            []PrePurchaseItem `gorm:"foreignKey:PrePurchaseID;references:ID" json:"pre_purchase_items"`
}

func (PrePurchase) TableName() string {
	return "pre_purchase"
}

type PrePurchaseItem struct {
	ID                   uuid.UUID `gorm:"primary_key;not null" json:"id"`
	PrePurchaseID        uuid.UUID `json:"pre_purchase_id"`
	PreItem              string    `json:"pre_item"`
	HierarchyType        string    `json:"hierarchy_type"` // Product Group ex. Group1
	HierarchyCode        string    `json:"hierarchy_code"` // Product Group Code ex. group 1 code
	DocRefItem           string    `json:"doc_ref_item"`
	Qty                  float64   `json:"qty"`
	Unit                 string    `json:"unit"`
	PurchaseQty          float64   `json:"purchase_qty"`
	PurchaseUnit         string    `json:"purchase_unit"`      // Unit <Pcs, Weight>
	PurchaseUnitType     string    `json:"purchase_unit_type"` // ex.KG-Spec, KG, PC
	PriceUnit            float64   `json:"price_unit"`         // ราคาต่อชิ้น => cost
	TotalDiscount        float64   `json:"total_discount"`
	TotalAmount          float64   `json:"total_amount"` // ราคารวม => total_cost - total_discount + total_vat
	UnitUom              string    `json:"unit_uom"`     // UOM มีสองแบบคือ KG, PC  unit_uom field uom_code
	TotalCost            float64   `json:"total_cost"`   // QTY * price_unit
	TotalDiscountPercent float64   `json:"total_discount_percent"`
	DiscountType         string    `json:"discount_type"` // PERCENTAGE, FIXED_AMOUNT
	TotalVat             float64   `json:"total_vat"`
	SubtotalExclVat      float64   `json:"subtotal_excl_vat"`
	WeightUnit           float64   `json:"weight_unit"`
	TotalWeight          float64   `json:"total_weight"`
	Status               string    `json:"status"`
	Remark               string    `json:"remark"`
	CreateDtm            time.Time `json:"create_dtm"`
	CreateBy             string    `json:"create_by"`
	UpdateDtm            time.Time `json:"update_dtm"`
	UpdateBy             string    `json:"update_by"`
}

func (PrePurchaseItem) TableName() string {
	return "pre_purchase_item"
}

type Purchase struct {
	ID                          uuid.UUID      `gorm:"primary_key;not null" json:"id"`
	PurchaseCode                string         `gorm:"unique;not null" json:"purchase_code"`
	PurchaseType                string         `json:"purchase_type"`
	CompanyCode                 string         `json:"company_code"`
	SiteCode                    string         `json:"site_code"`
	DocRefType                  *string        `json:"doc_ref_type"`
	DocRef                      *string        `json:"doc_ref"`
	TradingRef                  *string        `json:"trading_ref"`
	SupplierCode                string         `json:"supplier_code"`
	SupplierName                string         `json:"supplier_name"`
	SupplierAddress             string         `json:"supplier_address"`
	SupplierPhone               string         `json:"supplier_phone"`
	SupplierEmail               string         `json:"supplier_email"`
	DeliveryDate                *time.Time     `json:"delivery_date"`
	DeliveryAddress             string         `json:"delivery_address"`
	Status                      string         `json:"status"`
	TotalAmount                 float64        `json:"total_amount"`
	TotalWeight                 float64        `json:"total_weight"`
	TotalDiscount               float64        `json:"total_discount"`
	TotalVat                    float64        `json:"total_vat"`
	SubtotalExclVat             float64        `json:"subtotal_excl_vat"`
	SubtotalExclDiscountExclVat float64        `json:"subtotal_excl_discount_excl_vat"`
	IsApproved                  bool           `json:"is_approved"`
	StatusApprove               string         `json:"status_approve"`
	Remark                      string         `json:"remark"`
	StatusPayment               string         `json:"status_payment"` // PENDING, COMPLETED for check invoice
	CreateBy                    string         `json:"create_by"`
	CreateDtm                   time.Time      `json:"create_dtm"`
	UpdateBy                    string         `json:"update_by"`
	UpdateDtm                   time.Time      `json:"update_dtm"`
	PurchaseItems               []PurchaseItem `gorm:"foreignKey:PurchaseID;references:ID" json:"purchase_items"`
}

func (Purchase) TableName() string {
	return "purchase"
}

type PurchaseItem struct {
	ID                   uuid.UUID `gorm:"primary_key;not null" json:"id"`
	PurchaseID           uuid.UUID `json:"purchase_id"`
	PurchaseItem         string    `json:"purchase_item"`
	ProductCode          string    `json:"product_code"`
	ProductDesc          string    `json:"product_desc"`
	ProductGroupCode     string    `json:"product_group_code"`
	ProductGroupName     string    `json:"product_group_name"`
	DocRefItem           string    `json:"doc_ref_item"`
	Qty                  float64   `json:"qty"`
	Unit                 string    `json:"unit"`
	PurchaseQty          float64   `json:"purchase_qty"`
	PurchaseUnit         string    `json:"purchase_unit"`      // Unit <Pcs, Weight>
	PurchaseUnitType     string    `json:"purchase_unit_type"` // ex.KG-Spec, KG, PC
	PriceUnit            float64   `json:"price_unit"`         // ราคาต่อชิ้น => cost
	TotalDiscount        float64   `json:"total_discount"`
	TotalAmount          float64   `json:"total_amount"` // ราคารวม => total_cost - total_discount + total_vat
	UnitUom              string    `json:"unit_uom"`     // UOM มีสองแบบคือ KG, PC  unit_uom field uom_code
	TotalCost            float64   `json:"total_cost"`   // QTY * price_unit
	TotalDiscountPercent float64   `json:"total_discount_percent"`
	DiscountType         string    `json:"discount_type"` // PERCENTAGE, FIXED_AMOUNT
	TotalVat             float64   `json:"total_vat"`
	SubtotalExclVat      float64   `json:"subtotal_excl_vat"`
	WeightUnit           float64   `json:"weight_unit"`
	TotalWeight          float64   `json:"total_weight"`
	Status               string    `json:"status"`
	Remark               string    `json:"remark"`
	StatusPayment        string    `json:"status_payment"` // PENDING, COMPLETED for check invoice
	CreateDtm            time.Time `json:"create_dtm"`
	CreateBy             string    `json:"create_by"`
	UpdateDtm            time.Time `json:"update_dtm"`
	UpdateBy             string    `json:"update_by"`
}

func (PurchaseItem) TableName() string {
	return "purchase_item"
}

// Pre Purchase DTOs
type CreatePOBigLotItemRequest struct {
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
	UnitUom              string  `json:"unit_uom"`
	TotalCost            float64 `json:"total_cost"`
	TotalDiscountPercent float64 `json:"total_discount_percent"`
	DiscountType         string  `json:"discount_type"` // PERCENTAGE, FIXED_AMOUNT
	TotalVat             float64 `json:"total_vat"`
	SubtotalExclVat      float64 `json:"subtotal_excl_vat"`
	WeightUnit           float64 `json:"weight_unit"`
	TotalWeight          float64 `json:"total_weight"`
	Status               string  `json:"status"`
	Remark               string  `json:"remark"`
}

type CreatePOBigLotRequest struct {
	CompanyCode     string                      `json:"company_code"`
	SiteCode        string                      `json:"site_code"`
	SupplierCode    string                      `json:"supplier_code"`
	SupplierName    string                      `json:"supplier_name"`
	SupplierAddress string                      `json:"supplier_address"`
	SupplierPhone   string                      `json:"supplier_phone"`
	SupplierEmail   string                      `json:"supplier_email"`
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
	PrePurchaseCodes  []string `json:"pre_purchase_codes"`
	SupplierCodes     []string `json:"supplier_codes"`
	ProductGroupCodes []string `json:"product_group_codes"`
	StatusApprove     []string `json:"status_approve"`
	CompanyCode       string   `json:"company_code"`
	SiteCode          string   `json:"site_code"`
	Page              int      `json:"page"`
	PageSize          int      `json:"page_size"`
}

type GetPOBigLotItemResponse struct {
	ID                   string  `json:"id"`
	PreItem              string  `json:"pre_item"`
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
	UnitUom              string  `json:"unit_uom"`
	TotalCost            float64 `json:"total_cost"`
	TotalDiscountPercent float64 `json:"total_discount_percent"`
	DiscountType         string  `json:"discount_type"` // PERCENTAGE, FIXED_AMOUNT
	TotalVat             float64 `json:"total_vat"`
	SubtotalExclVat      float64 `json:"subtotal_excl_vat"`
	WeightUnit           float64 `json:"weight_unit"`
	TotalWeight          float64 `json:"total_weight"`
	Status               string  `json:"status"`
	Remark               string  `json:"remark"`
	CreateDtm            string  `json:"create_dtm"`
	CreateBy             string  `json:"create_by"`
	UpdateDtm            string  `json:"update_dtm"`
	UpdateBy             string  `json:"update_by"`
}

type GetPOBigLotResponse struct {
	ID                          string                    `json:"id"`
	PrePurchaseCode             string                    `json:"pre_purchase_code"`
	PurchaseType                string                    `json:"purchase_type"`
	CompanyCode                 string                    `json:"company_code"`
	SiteCode                    string                    `json:"site_code"`
	SupplierCode                string                    `json:"supplier_code"`
	SupplierName                string                    `json:"supplier_name"`
	SupplierAddress             string                    `json:"supplier_address"`
	SupplierPhone               string                    `json:"supplier_phone"`
	SupplierEmail               string                    `json:"supplier_email"`
	DeliveryAddress             string                    `json:"delivery_address"`
	Status                      string                    `json:"status"`
	TotalAmount                 float64                   `json:"total_amount"`
	TotalWeight                 float64                   `json:"total_weight"`
	TotalDiscount               float64                   `json:"total_discount"`
	TotalVat                    float64                   `json:"total_vat"`
	SubtotalExclDiscountExclVat float64                   `json:"subtotal_excl_discount_excl_vat"`
	SubtotalExclVat             float64                   `json:"subtotal_excl_vat"`
	IsApproved                  bool                      `json:"is_approved"`
	StatusApprove               string                    `json:"status_approve"`
	Remark                      string                    `json:"remark"`
	CreateBy                    string                    `json:"create_by"`
	CreateDtm                   string                    `json:"create_dtm"`
	UpdateBy                    string                    `json:"update_by"`
	UpdateDtm                   string                    `json:"update_dtm"`
	PrePurchaseItems            []GetPOBigLotItemResponse `json:"pre_purchase_items"`
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
	PreItem              *string    `json:"pre_item"`
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
	UnitUom              string     `json:"unit_uom"`
	TotalCost            float64    `json:"total_cost"`
	TotalDiscountPercent float64    `json:"total_discount_percent"`
	DiscountType         string     `json:"discount_type"` // PERCENTAGE, FIXED_AMOUNT
	TotalVat             float64    `json:"total_vat"`
	SubtotalExclVat      float64    `json:"subtotal_excl_vat"`
	WeightUnit           float64    `json:"weight_unit"`
	TotalWeight          float64    `json:"total_weight"`
	Status               string     `json:"status"`
	Remark               string     `json:"remark"`
	CreateBy             string     `json:"create_by"`
	CreateDtm            time.Time  `json:"create_dtm"`
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
	Remark           string                      `json:"remark"`
	DeliveryAddress  string                      `json:"delivery_address"`
	PrePurchaseItems []UpdatePOBigLotItemRequest `json:"pre_purchase_items"`
}

type UpdateStatusApprovePOBigLotRequest struct {
	ID              uuid.UUID `json:"id"`
	PrePurchaseCode string    `json:"pre_purchase_code"`
	IsApproved      bool      `json:"is_approved"`
	StatusApprove   string    `json:"status_approve"`
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
	PostCode     string    `json:"post_code"`
	Province     string    `json:"province"`
	Country      string    `json:"country"`
	Contact      string    `json:"contact"`
	Phone        string    `json:"phone"`
	Email        string    `json:"email"`
	Address      string    `json:"address"`
	ActiveFlg    bool      `json:"active_flg"`
	CreditTerm   int       `json:"credit_term"`
	ExternalID   string    `json:"external_id"`
	TaxID        string    `json:"tax_id"`
	Branch       string    `json:"branch"`
	CreateBy     string    `json:"create_by"`
	CreateDtm    time.Time `json:"create_dtm"`
	UpdateBy     string    `json:"update_by"`
	UpdateDtm    time.Time `json:"update_dtm"`
}

// Purchase DTOs
type PurchaseItemFormRequest struct {
	ID                   *uuid.UUID `json:"id"`
	PurchaseID           *uuid.UUID `json:"purchase_id"`
	PurchaseItem         *string    `json:"purchase_item"`
	ProductCode          string     `json:"product_code"`
	ProductDesc          string     `json:"product_desc"`
	ProductGroupCode     string     `json:"product_group_code"`
	ProductGroupName     string     `json:"product_group_name"`
	DocRefItem           *string    `json:"doc_ref_item"`
	Qty                  float64    `json:"qty"`
	Unit                 string     `json:"unit"`
	PurchaseQty          float64    `json:"purchase_qty"`
	PurchaseUnit         string     `json:"purchase_unit"`
	PurchaseUnitType     string     `json:"purchase_unit_type"`
	PriceUnit            float64    `json:"price_unit"`
	TotalDiscount        float64    `json:"total_discount"`
	TotalAmount          float64    `json:"total_amount"`
	UnitUom              string     `json:"unit_uom"`
	TotalCost            float64    `json:"total_cost"`
	TotalDiscountPercent float64    `json:"total_discount_percent"`
	DiscountType         string     `json:"discount_type"` // PERCENTAGE, FIXED_AMOUNT
	TotalVat             float64    `json:"total_vat"`
	SubtotalExclVat      float64    `json:"subtotal_excl_vat"`
	WeightUnit           float64    `json:"weight_unit"`
	TotalWeight          float64    `json:"total_weight"`
	Status               string     `json:"status"`
	Remark               string     `json:"remark"`
	CreateBy             *string    `json:"create_by"`
	CreateDtm            *time.Time `json:"create_dtm"`
}

type PurchaseFormRequest struct {
	ID              *uuid.UUID                `json:"id"`
	PurchaseCode    *string                   `json:"purchase_code"`
	PurchaseType    string                    `json:"purchase_type"`
	DocRefType      *string                   `json:"doc_ref_type"`
	DocRef          *string                   `json:"doc_ref"`
	TradingRef      *string                   `json:"trading_ref"`
	SupplierCode    *string                   `json:"supplier_code"`
	SupplierName    *string                   `json:"supplier_name"`
	SupplierAddress *string                   `json:"supplier_address"`
	SupplierPhone   *string                   `json:"supplier_phone"`
	SupplierEmail   *string                   `json:"supplier_email"`
	DeliveryDate    *time.Time                `json:"delivery_date"`
	DeliveryAddress string                    `json:"delivery_address"`
	Status          string                    `json:"status"`
	TotalAmount     float64                   `json:"total_amount"`
	TotalWeight     float64                   `json:"total_weight"`
	TotalDiscount   float64                   `json:"total_discount"`
	TotalVat        float64                   `json:"total_vat"`
	SubtotalExclVat float64                   `json:"subtotal_excl_vat"`
	IsApproved      bool                      `json:"is_approved"`
	StatusApprove   string                    `json:"status_approve"`
	Remark          string                    `json:"remark"`
	Items           []PurchaseItemFormRequest `json:"items"`
}

type CreatePurchaseRequest struct {
	CompanyCode string                `json:"company_code"`
	SiteCode    string                `json:"site_code"`
	Purchases   []PurchaseFormRequest `json:"purchases"`
}

type GetPurchaseRequest struct {
	PurchaseCodes           []string `json:"purchase_codes"`
	SupplierCodes           []string `json:"supplier_codes"`
	StatusApprove           []string `json:"status_approve"`
	StatusPayment           []string `json:"status_payment"` // PENDING, COMPLETED for check invoice
	StatusPaymentIncomplete bool     `json:"status_payment_incomplete"`
	ProductCodes            []string `json:"product_codes"`
	PurchaseType            []string `json:"purchase_type"`
	DocRef                  []string `json:"doc_ref"`
	TradingRef              []string `json:"trading_ref"`
	CompanyCode             string   `json:"company_code"`
	SiteCode                string   `json:"site_code"`
	Page                    int      `json:"page"`
	PageSize                int      `json:"page_size"`
}

type PurchaseItemResponse struct {
	ID                   string  `json:"id"`
	PurchaseID           string  `json:"purchase_id"`
	PurchaseItem         string  `json:"purchase_item"`
	DocRefItem           string  `json:"doc_ref_item"`
	ProductCode          string  `json:"product_code"`
	ProductDesc          string  `json:"product_desc"`
	ProductGroupOneCode  string  `json:"product_group_one_code"`
	ProductGroupOneName  string  `json:"product_group_one_name"`
	Qty                  float64 `json:"qty"`
	Unit                 string  `json:"unit"`
	PurchaseQty          float64 `json:"purchase_qty"`
	PurchaseUnit         string  `json:"purchase_unit"`
	PurchaseUnitType     string  `json:"purchase_unit_type"`
	PriceUnit            float64 `json:"price_unit"`
	TotalDiscount        float64 `json:"total_discount"`
	TotalAmount          float64 `json:"total_amount"`
	UnitUom              string  `json:"unit_uom"`
	TotalCost            float64 `json:"total_cost"`
	TotalDiscountPercent float64 `json:"total_discount_percent"`
	DiscountType         string  `json:"discount_type"` // PERCENTAGE, FIXED_AMOUNT
	TotalVat             float64 `json:"total_vat"`
	SubtotalExclVat      float64 `json:"subtotal_excl_vat"`
	WeightUnit           float64 `json:"weight_unit"`
	TotalWeight          float64 `json:"total_weight"`
	Status               string  `json:"status"`
	StatusPayment        string  `json:"status_payment"` // PENDING, COMPLETED for check invoice
	Remark               string  `json:"remark"`
	CreateDtm            string  `json:"create_dtm"`
	CreateBy             string  `json:"create_by"`
	UpdateDtm            string  `json:"update_dtm"`
	UpdateBy             string  `json:"update_by"`
}

type PurchaseResponse struct {
	ID                          string                 `json:"id"`
	PurchaseCode                string                 `json:"purchase_code"`
	PurchaseType                string                 `json:"purchase_type"`
	CompanyCode                 string                 `json:"company_code"`
	SiteCode                    string                 `json:"site_code"`
	DocRefType                  *string                `json:"doc_ref_type"`
	DocRef                      *string                `json:"doc_ref"`
	TradingRef                  *string                `json:"trading_ref"`
	SupplierCode                string                 `json:"supplier_code"`
	SupplierName                string                 `json:"supplier_name"`
	SupplierAddress             string                 `json:"supplier_address"`
	SupplierPhone               string                 `json:"supplier_phone"`
	SupplierEmail               string                 `json:"supplier_email"`
	DeliveryDate                string                 `json:"delivery_date"`
	DeliveryAddress             string                 `json:"delivery_address"`
	Status                      string                 `json:"status"`
	TotalAmount                 float64                `json:"total_amount"`
	TotalWeight                 float64                `json:"total_weight"`
	TotalDiscount               float64                `json:"total_discount"`
	TotalVat                    float64                `json:"total_vat"`
	SubtotalExclDiscountExclVat float64                `json:"subtotal_excl_discount_excl_vat"`
	SubtotalExclVat             float64                `json:"subtotal_excl_vat"`
	IsApproved                  bool                   `json:"is_approved"`
	StatusApprove               string                 `json:"status_approve"`
	StatusPayment               string                 `json:"status_payment"` // PENDING, COMPLETED for check invoice
	Remark                      string                 `json:"remark"`
	CreateBy                    string                 `json:"create_by"`
	CreateDtm                   string                 `json:"create_dtm"`
	UpdateBy                    string                 `json:"update_by"`
	UpdateDtm                   string                 `json:"update_dtm"`
	Items                       []PurchaseItemResponse `json:"items"`
	RefBigLot                   *GetPOBigLotResponse   `json:"ref_big_lot"`
}

type GetPurchaseResponse struct {
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
	DataList   []PurchaseResponse `json:"data_list"`
}

type UpdateStatusApprovePurchaseRequest struct {
	ID            uuid.UUID `json:"id"`
	PurchaseCode  string    `json:"purchase_code"`
	IsApproved    bool      `json:"is_approved"`
	StatusApprove string    `json:"status_approve"`
}

type CompleteStatusPaymentPurchaseRequest struct {
	PurchaseCodes []string `json:"purchase_codes"`
	PurchaseItems []string `json:"purchase_items"`
}

type CompletePurchaseRequest struct {
	PurchaseCodes []string `json:"purchase_codes"`
}

// Product DTOs
type GetProductRequest struct {
	ProductCode []string `json:"product_code"`
	SiteCode    []string `json:"site_code"`
	CompanyCode []string `json:"company_code"`
}

type GetProductsDetailResponse struct {
	Total      int                          `json:"total"`
	Page       int                          `json:"page"`
	PageSize   int                          `json:"page_size"`
	TotalPages int                          `json:"total_pages"`
	Products   []GetProductsDetailComponent `json:"products"`
}

type GetProductsDetailComponent struct {
	ProductId                     string                         `json:"product_id"`
	ProductCode                   string                         `json:"product_code"`
	TenantId                      string                         `json:"tenant_id"`
	ProductName                   string                         `json:"product_name"`
	Description                   string                         `json:"description"`
	ActiveFlg                     bool                           `json:"active_flg"`
	CategoryCode                  string                         `json:"category_code"`
	ImgUrl                        string                         `json:"img_url"`
	ProductType                   string                         `json:"product_type"`
	ShelfLifeDay                  int                            `json:"shelf_life_day"`
	FlagProductExpire             bool                           `json:"flag_product_expire"`
	IsBatch                       bool                           `json:"is_batch"`
	IsSerial                      bool                           `json:"is_serial"`
	Width                         float64                        `json:"width"`
	Height                        float64                        `json:"height"`
	Length                        float64                        `json:"length"`
	Weight                        float64                        `json:"weight"`
	Brand                         string                         `json:"brand"`
	ABCIndicator                  string                         `json:"abc_indicator"`
	FlgExcludeAccounting          bool                           `json:"flg_exclude_accounting"`
	FlgIgnoreCustomerMinShelfLife bool                           `json:"flg_ignore_customer_min_shelf_life"`
	FreeGoodsMinShelfLifeDay      int                            `json:"free_goods_min_shelf_life_day"`
	NormalGoodsMinShelfLifeDay    int                            `json:"normal_goods_min_shelf_life_day"`
	StorageZone                   string                         `json:"storage_zone"`
	PutawayStrategyCode           string                         `json:"putaway_strategy_code"`
	SupplierCode                  string                         `json:"supplier_code"`
	SubstitutionMaterialCode      string                         `json:"substitution_material_code"`
	SubstitutionMaterialName      string                         `json:"substitution_material_name"`
	MaxQty                        float64                        `json:"max_qty"`
	ReorderQty                    float64                        `json:"reorder_qty"`
	RoundingUnit                  string                         `json:"rounding_unit"`
	QtyBomKitting                 *float64                       `json:"qty_bom_kitting"`
	QtyBomProduction              *float64                       `json:"qty_bom_production"`
	MinSellableDay                int64                          `json:"min_sellable_day"`
	SiteCode                      string                         `json:"site_code"`
	CompanyCode                   string                         `json:"company_code"`
	Attributes                    []GetAttributesDetailComponent `json:"attributes"`
	Tags                          []GetTagsDetailComponent       `json:"tags"`
	Component                     []GetComponentDetailComponent  `json:"component"`
	Units                         []GetUnitsDetailComponent      `json:"units"`
	ProductGroups                 []ProductGroup                 `json:"product_groups"`
	ProductPicks                  []ProductPickWith              `json:"product_pick_with"`
	PickFaceLocation              []PickFaceLocation             `json:"pick_face_location"`
	Versions                      []GetComponentVersion          `json:"versions"`
}

type ProductGroup struct {
	ID         string `json:"id"`
	ProductId  string `json:"product_id"`
	GroupCode  string `json:"group_code"`
	GroupValue string `json:"group_value"`
	ActiveFlg  bool   `json:"active_flg"`
	Seq        int    `json:"seq"`
	CreateDtm  string `json:"create_dtm"`
}

type ProductPickWith struct {
	ID          string  `json:"id"`
	ProductId   string  `json:"product_id"`
	ProductCode string  `json:"product_code"`
	ProductName string  `json:"product_name"`
	Qty         float64 `json:"qty"`
	Unit        string  `json:"unit"`
	ActiveFlg   bool    `json:"active_flg"`
	CreateDtm   string  `json:"create_dtm"`
}

type PickFaceLocation struct {
	ID         string `json:"id"`
	ProductId  string `json:"product_id"`
	Rack       string `json:"rack"`
	StartBay   string `json:"start_bay"`
	StartLevel string `json:"start_level"`
	EndBay     string `json:"end_bay"`
	EndLevel   string `json:"end_level"`
	ActiveFlg  bool   `json:"active_flg"`
	CreateDtm  string `json:"create_dtm"`
}

type GetAttributesDetailComponent struct {
	AttributeId    string `json:"attribute_id"`
	AttributeCode  string `json:"attribute_code"`
	AttributeDesc  string `json:"attribute_desc"`
	AttributeType  string `json:"attribute_type"`
	AttributeValue string `json:"attribute_value"`
}

type GetTagsDetailComponent struct {
	TagId   string `json:"tag_id"`
	TagCode string `json:"tag_code"`
}

type GetComponentDetailComponent struct {
	ID                   string  `json:"id"`
	ProductId            string  `json:"product_id"`
	ComponentProductId   string  `json:"component_product_id"`
	ComponentProductCode string  `json:"component_product_code"`
	ComponentProductName string  `json:"component_product_name"`
	Qty                  float64 `json:"qty"`
	UnitCode             string  `json:"unit_code"`
	Waste                float64 `json:"waste"`
	ValidFrom            string  `json:"valid_from"`
	ValidTo              string  `json:"valid_to"`
	BomKittingFlg        bool    `json:"bom_kitting_flg"`
	BomProdFlg           bool    `json:"bom_prod_flg"`
	ActiveFlg            bool    `json:"active_flg"`
	VersionCode          string  `json:"version_code"`
}

type GetUnitsDetailComponent struct {
	ID            string                           `json:"id"`
	UnitCode      string                           `json:"unit_code"`
	Ratio         float64                          `json:"ratio"`
	RatioStandard float64                          `json:"ratio_standard"`
	Width         float64                          `json:"width"`
	Height        float64                          `json:"height"`
	Length        float64                          `json:"length"`
	Weight        float64                          `json:"weight"`
	FlagBase      bool                             `json:"flag_base"`
	FlagGr        bool                             `json:"flag_gr"`
	FlagSale      bool                             `json:"flag_sale"`
	FlagGi        bool                             `json:"flag_gi"`
	FlagPick      bool                             `json:"flag_pick"`
	FlagTransfer  bool                             `json:"flag_transfer"`
	FlagCount     bool                             `json:"flag_count"`
	ActiveFlg     bool                             `json:"active_flg"`
	FlagPallet    bool                             `json:"flag_pallet"`
	FlagContainer bool                             `json:"flag_container"`
	Cubic         float64                          `json:"cubic"`
	Barcodes      []GetUnitsDetailBarcodeComponent `json:"barcodes"`
}

type GetComponentVersion struct {
	ID              string `json:"id"`
	ProductId       string `json:"product_id"`
	VersionCode     string `json:"version_code"`
	VersionName     string `json:"version_name"`
	AvtiveStartDate string `json:"avtive_start_date"`
	ActiveEndDate   string `json:"active_end_date"`
	CreateDtm       string `json:"create_dtm"`
	IsDefault       bool   `json:"is_default"`
}

type GetUnitsDetailBarcodeComponent struct {
	ID      string `json:"id"`
	Barcode string `json:"barcode"`
}

// For GR or GR Plan DTOs
type CompletePurchaseItemRequest struct {
	PurchaseItemCodes []string `json:"purchase_item_codes"`
}

type ExceptPurchaseAndPurchaseItemRequest struct {
	PurchaseCode      string   `json:"purchase_code"`
	PurchaseItemCodes []string `json:"purchase_item_codes"`
}

type GetPurchaseItemRequest struct {
	NotItems        []ExceptPurchaseAndPurchaseItemRequest `json:"not_items"`
	SupplierCodes   []string                               `json:"supplier_codes"`
	POStatusApprove []string                               `json:"po_status_approve"`
	POItemStatus    []string                               `json:"po_item_status"`
	ProductCodes    []string                               `json:"product_codes"`
	CompanyCode     string                                 `json:"company_code"`
	SiteCode        string                                 `json:"site_code"`
	Page            int                                    `json:"page"`
	PageSize        int                                    `json:"page_size"`
}

type GetPurchaseItemResponse struct {
	ID                   string               `json:"id"`
	PurchaseCode         string               `json:"purchase_code"`
	PurchaseType         string               `json:"purchase_type"`
	CompanyCode          string               `json:"company_code"`
	SiteCode             string               `json:"site_code"`
	DocRefType           *string              `json:"doc_ref_type"`
	DocRef               *string              `json:"doc_ref"`
	TradingRef           *string              `json:"trading_ref"`
	SupplierCode         string               `json:"supplier_code"`
	SupplierName         string               `json:"supplier_name"`
	SupplierAddress      string               `json:"supplier_address"`
	SupplierPhone        string               `json:"supplier_phone"`
	SupplierEmail        string               `json:"supplier_email"`
	PurchaseItem         string               `json:"purchase_item"`
	DocRefItem           string               `json:"doc_ref_item"`
	ProductCode          string               `json:"product_code"`
	ProductDesc          string               `json:"product_desc"`
	ProductGroupOneCode  string               `json:"product_group_one_code"`
	ProductGroupOneName  string               `json:"product_group_one_name"`
	Qty                  float64              `json:"qty"`
	Unit                 string               `json:"unit"`
	PurchaseQty          float64              `json:"purchase_qty"`
	PurchaseUnit         string               `json:"purchase_unit"`
	PurchaseUnitType     string               `json:"purchase_unit_type"`
	PriceUnit            float64              `json:"price_unit"`
	TotalDiscount        float64              `json:"total_discount"`
	TotalAmount          float64              `json:"total_amount"`
	UnitUom              string               `json:"unit_uom"`
	TotalCost            float64              `json:"total_cost"`
	TotalDiscountPercent float64              `json:"total_discount_percent"`
	DiscountType         string               `json:"discount_type"` // PERCENTAGE, FIXED_AMOUNT
	TotalVat             float64              `json:"total_vat"`
	SubtotalExclVat      float64              `json:"subtotal_excl_vat"`
	WeightUnit           float64              `json:"weight_unit"`
	TotalWeight          float64              `json:"total_weight"`
	Status               string               `json:"status"`
	StatusPayment        string               `json:"status_payment"` // PENDING, COMPLETED for check invoice
	IsApproved           bool                 `json:"is_approved"`
	StatusApprove        string               `json:"status_approve"`
	Remark               string               `json:"remark"`
	CreateDtm            string               `json:"create_dtm"`
	CreateBy             string               `json:"create_by"`
	UpdateDtm            string               `json:"update_dtm"`
	UpdateBy             string               `json:"update_by"`
	RefBigLot            *GetPOBigLotResponse `json:"ref_big_lot"`
	RemainQty            float64              `json:"remain_qty"`
	RemainWeight         float64              `json:"remain_weight"`
}

type GetPurchaseItemListResponse struct {
	Total      int                       `json:"total"`
	Page       int                       `json:"page"`
	PageSize   int                       `json:"page_size"`
	TotalPages int                       `json:"total_pages"`
	DataList   []GetPurchaseItemResponse `json:"data_list"`
}
