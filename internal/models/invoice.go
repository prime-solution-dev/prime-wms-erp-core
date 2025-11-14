package models

import (
	"time"

	"github.com/google/uuid"
)

type Invoice struct {
	ID                     uuid.UUID        `json:"id"`
	InvoiceCode            string           `json:"invoice_code"`
	InvoiceRef             string           `json:"invoice_ref"`
	InvoiceType            string           `json:"invoice_type"`
	DocumentRefType        string           `json:"document_ref_type"`
	DocumentRef            string           `json:"document_ref"`
	CreditTermDay          float64          `json:"credit_term_day"`
	PaymentDate            *time.Time       `json:"payment_date"`
	DocumentDate           *time.Time       `json:"document_date"`
	TaxDate                *time.Time       `json:"tax_date"`
	TaxInvoice             string           `json:"tax_invoice"`
	PartyType              string           `json:"party_type"`
	PartyCode              string           `json:"party_code"`
	PartyName              string           `json:"party_name"`
	PartyBranch            string           `json:"party_branch"`
	PartyAddress           string           `json:"party_address"`
	PartyEmail             string           `json:"party_email"`
	PartyTel               string           `json:"party_tel"`
	PartyTaxID             string           `json:"party_tax_id"`
	PartyExternalID        string           `gorm:"-"  json:"party_external_id"`
	DueDate                *time.Time       `json:"due_date"`
	TotalAmount            float64          `json:"total_amount"`
	TotalVat               float64          `json:"total_vat"`
	Status                 string           `json:"status"`
	Remark                 string           `json:"remark"`
	CreateBy               string           `gorm:"type:varchar(100)" json:"create_by"`
	CreateDtm              time.Time        `gorm:"autoCreateTime;<-:create" json:"create_dtm"`
	UpdateBy               string           `gorm:"type:varchar(100)" json:"update_by"`
	UpdateDTM              time.Time        `gorm:"autoUpdateTime;<-" json:"update_dtm"`
	InvoiceItem            []InvoiceItem    `json:"invoice_item"`
	InvoiceDeposit         []InvoiceDeposit `json:"invoice_deposit"`
	CompanyCode            string           `gorm:"-" json:"company_code"`
	SiteCode               string           `gorm:"-" json:"site_code"`
	ExternalID             string           `json:"external_id"`
	SubtotalExclVat        float64          `json:"subtotal_excl_vat"`
	SubtotalExclVatDeposit float64          `json:"subtotal_excl_vat_deposit"`
	PaymentMethod          string           `json:"payment_method"`
	OwnerName              string           `json:"owner_name"`
	PartyDocumentRef       string           `json:"party_document_ref"`
	InvoiceDate            *time.Time       `json:"invoice_date"`
	TotalDiscount          float64          `json:"total_discount"`
}

func (Invoice) TableName() string { return "invoice" }

type InvoiceItem struct {
	ID                     uuid.UUID  `json:"id"`
	InvoiceID              uuid.UUID  `json:"invoice_id"`
	InvoiceItem            string     `json:"invoice_item"`
	InvoiceCode            string     `gorm:"-" json:"invoice_code"`
	PurchaseItem           string     `gorm:"-" json:"purchase_item"`
	DocRefItem             string     `json:"doc_ref_item"`
	ProductCode            string     `json:"product_code"`
	ProductName            string     `gorm:"-" json:"product_name"`
	ProductGroup           string     `gorm:"-" json:"product_group"`
	Qty                    float64    `json:"qty"`
	UnitCode               string     `json:"unit_code"`
	PriceUnit              float64    `json:"price_unit"`
	CreateBy               string     `gorm:"type:varchar(100)" json:"create_by"`
	CreateDtm              time.Time  `gorm:"autoCreateTime;<-:create" json:"create_dtm"`
	UpdateBy               string     `gorm:"type:varchar(100)" json:"update_by"`
	UpdateDTM              time.Time  `gorm:"autoUpdateTime;<-" json:"update_dtm"`
	InvoiceQty             float64    `json:"invoice_qty"`
	InvoiceUnit            string     `json:"invoice_unit"`
	InvoiceUnitType        string     `json:"invoice_unit_type"`
	UnitUom                string     `json:"unit_uom"`
	WeightUnit             float64    `json:"weight_unit"`
	Avg_weightUnit         float64    `json:"avg_weight_unit"`
	InvoiceWeight          float64    `json:"invoice_weight"`
	Weight                 float64    `json:"weight"`
	TotalDiscount          float64    `json:"total_discount"`
	TotalDiscount_percent  float64    `json:"total_discount_percent"`
	DocumentRefType        string     `json:"document_ref_type"`
	DocumentRef            string     `json:"document_ref"`
	DocumentRefItem        string     `json:"document_ref_item"`
	SourceType             string     `json:"source_type"`
	SourceCode             string     `json:"source_code"`
	SourceItem             string     `json:"source_item"`
	TotalAmount            float64    `json:"total_amount"`
	TotalVat               float64    `json:"total_vat"`
	Status                 string     `json:"status"`
	Remark                 string     `json:"remark"`
	ProductDescription     string     `json:"product_description"`
	SubtotalExclVat        float64    `json:"subtotal_excl_vat"`
	SubtotalExclVatDeposit float64    `json:"subtotal_excl_vat_deposit"`
	ArticleCode            string     `json:"article_code"`
	ProductDesc            string     `json:"product_desc"`
	InvoiceType            string     `gorm:"-" json:"invoice_type"`
	ArticleType            string     `json:"article_type"`
	TotalWeight            float64    `json:"total_weight"`
	PriceListUnit          float64    `json:"price_list_unit"`
	DocumentDate           *time.Time `json:"document_date"`
}

func (InvoiceItem) TableName() string { return "invoice_item" }

type InvoiceDeposit struct {
	ID          uuid.UUID  `json:"id"`
	InvoiceID   uuid.UUID  `json:"invoice_id"`
	DepositCode string     `json:"deposit_code"`
	ApplyDate   *time.Time `json:"apply_date"`
	Amount      float64    `json:"amount"`
	CreateBy    string     `gorm:"type:varchar(100)" json:"create_by"`
	CreateDtm   time.Time  `gorm:"autoCreateTime;<-:create" json:"create_dtm"`
	UpdateBy    string     `gorm:"type:varchar(100)" json:"update_by"`
	UpdateDTM   time.Time  `gorm:"autoUpdateTime;<-" json:"update_dtm"`
}

func (InvoiceDeposit) TableName() string { return "invoice_deposit" }
