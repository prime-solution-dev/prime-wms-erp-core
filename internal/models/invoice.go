package models

import (
	"time"

	"github.com/google/uuid"
)

type Invoice struct {
	ID              uuid.UUID        `json:"id"`
	InvoiceCode     string           `json:"invoice_code"`
	InvoiceRef      string           `json:"invoice_ref"`
	InvoiceType     string           `json:"invoice_type"`
	DocumentRefType string           `json:"document_ref_type"`
	DocumentRef     string           `json:"document_ref"`
	CreditTermDay   string           `json:"credit_term_day"`
	PaymentDate     *time.Time       `json:"payment_date"`
	DocumentDate    *time.Time       `json:"document_date"`
	TaxDate         *time.Time       `json:"tax_date"`
	TaxInvoice      string           `json:"tax_invoice"`
	PartyType       string           `json:"party_type"`
	PartyCode       string           `json:"party_code"`
	DueDate         *time.Time       `json:"due_date"`
	TotalAmount     float64          `json:"total_amount"`
	TotalVat        float64          `json:"total_vat"`
	Status          string           `json:"status"`
	Remark          string           `json:"remark"`
	CreateBy        string           `json:"create_by"`
	CreateDtm       *time.Time       `json:"create_dtm"`
	UpdateBy        string           `json:"update_by"`
	UpdateDate      *time.Time       `json:"update_date"`
	InvoiceItem     []InvoiceItem    `json:"invoice_item"`
	InvoiceDeposit  []InvoiceDeposit `json:"invoice_deposit"`
}

func (Invoice) TableName() string { return "invoice" }

type InvoiceItem struct {
	ID                    uuid.UUID  `json:"id"`
	InvoiceItem           string     `json:"invoice_item"`
	InvoiceID             uuid.UUID  `json:"invoice_id"`
	InvoiceQty            float64    `json:"invoice_qty"`
	InvoiceUnit           string     `json:"invoice_unit"`
	InvoiceUnitType       string     `json:"invoice_unit_type"`
	ProductCode           string     `json:"product_code"`
	Qty                   float64    `json:"qty"`
	UnitCode              string     `json:"unit_code"`
	PriceUnit             float64    `json:"price_unit"`
	UnitUom               string     `json:"unit_uom"`
	WeightUnit            string     `json:"weight_unit"`
	Avg_weightUnit        string     `json:"avg_weight_unit"`
	TotalWeight           string     `json:"total_weight"`
	TotalDiscount         string     `json:"total_discount"`
	TotalDiscount_percent string     `json:"total_discount_percent"`
	DocumentRefType       string     `json:"document_ref_type"`
	DocumentRef           string     `json:"document_ref"`
	DocumentRefItem       string     `json:"document_ref_item"`
	SourceType            string     `json:"source_type"`
	SourceCode            string     `json:"source_code"`
	SourceItem            string     `json:"source_item"`
	TotalAmount           float64    `json:"total_amount"`
	TotalVat              float64    `json:"total_vat"`
	Status                string     `json:"status"`
	Remark                string     `json:"remark"`
	CreateBy              string     `json:"create_by"`
	CreateDtm             *time.Time `json:"create_dtm"`
	UpdateBy              string     `json:"update_by"`
	UpdateDate            *time.Time `json:"update_date"`
}

func (InvoiceItem) TableName() string { return "invoice_item" }

type InvoiceDeposit struct {
	ID          uuid.UUID  `json:"id"`
	InvoiceID   uuid.UUID  `json:"invoice_id"`
	DepositCode string     `json:"deposit_code"`
	ApplyDate   *time.Time `json:"apply_date"`
	Amount      float64    `json:"amount"`
	CreateBy    string     `json:"create_by"`
	CreateDtm   *time.Time `json:"create_dtm"`
	UpdateBy    string     `json:"update_by"`
	UpdateDate  *time.Time `json:"update_date"`
}

func (InvoiceDeposit) TableName() string { return "invoice_deposit" }
