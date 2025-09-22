package models

import (
	"time"

	"github.com/google/uuid"
)

type Invoice struct {
	ID                  uuid.UUID        `json:"id"`
	InvoiceCode         string           `json:"invoice_code"`
	DocRef              string           `json:"doc_ref"`
	DocRefType          string           `json:"doc_ref_type"`
	CustomerCode        string           `json:"customer_code"`
	DueDate             *time.Time       `json:"due_date"`
	TransportCostType   string           `json:"transport_cost_type"`
	TotalTransportCost  float64          `json:"total_transport_cost"`
	TotalPrice          float64          `json:"total_price"`
	TotalWeight         float64          `json:"total_weight"`
	TotalDeposit        float64          `json:"total_deposit"`
	TotalNetPrice       float64          `json:"total_net_price"`
	TotalNetPriceWeight float64          `json:"total_net_price_weight"`
	Status              string           `json:"status"`
	Remark              string           `json:"remark"`
	CreateBy            string           `json:"create_by"`
	CreateDtm           *time.Time       `json:"create_dtm"`
	UpdateBy            string           `json:"update_by"`
	UpdateDate          *time.Time       `json:"update_date"`
	InvoiceItem         []InvoiceItem    `json:"invoice_item"`
	InvoiceDeposit      []InvoiceDeposit `json:"invoice_deposit"`
}

func (Invoice) TableName() string { return "invoice" }

type InvoiceItem struct {
	ID                      uuid.UUID  `json:"id"`
	InvoiceItem             string     `json:"invoice_item"`
	InvoiceID               uuid.UUID  `json:"invoice_id"`
	DocRefItem              string     `json:"doc_ref_item"`
	ProductCode             string     `json:"product_code"`
	Qty                     float64    `json:"qty"`
	UnitCode                string     `json:"unit_code"`
	PriceUnit               float64    `json:"price_unit"`
	TotalPrice              float64    `json:"total_price"`
	TransportCostUnit       float64    `json:"transport_cost_unit"`
	TotalNetPrice           float64    `json:"total_net_price"`
	DepositUnit             float64    `json:"deposit_unit"`
	NetPriceUnit            float64    `json:"net_price_unit"`
	WeightUnit              float64    `json:"weight_unit"`
	AvgWeightUnit           float64    `json:"avg_weight_unit"`
	TotalWeight             float64    `json:"total_weight"`
	TransportCostWeightUnit float64    `json:"transport_cost_weight_unit"`
	TotalNetPriceWeight     float64    `json:"total_net_price_weight"`
	DepositWeightUnit       float64    `json:"deposit_weight_unit"`
	NetPriceWeightUnit      float64    `json:"net_price_weight_unit"`
	CreateBy                string     `json:"create_by"`
	CreateDtm               *time.Time `json:"create_dtm"`
	UpdateBy                string     `json:"update_by"`
	UpdateDate              *time.Time `json:"update_date"`
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
