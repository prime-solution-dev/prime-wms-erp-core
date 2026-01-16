package models

import (
	"time"

	"github.com/google/uuid"
)

type Quotation struct {
	ID                          uuid.UUID  `json:"id"`
	QuotationCode               string     `json:"quotation_code"`
	CompanyCode                 string     `json:"company_code"`
	SiteCode                    string     `json:"site_code"`
	CustomerCode                string     `json:"customer_code"`
	CustomerName                string     `json:"customer_name"`
	DeliveryDate                *time.Time `json:"delivery_date"`
	SoldToCode                  string     `json:"sold_to_code"`
	SoldToAddress               string     `json:"sold_to_address"`
	BillToCode                  string     `json:"bill_to_code"`
	BillToAddress               string     `json:"bill_to_address"`
	ShipToCode                  string     `json:"ship_to_code"`
	ShipToType                  string     `json:"ship_to_type"`
	ShipToAddress               string     `json:"ship_to_address"`
	DeliveryMethod              string     `json:"delivery_method"`
	TransportCostType           string     `json:"transport_cost_type"`
	TotalTransportCost          float64    `json:"total_transport_cost"`
	PassPrice                   bool       `json:"pass_price"`
	TotalAmount                 float64    `json:"total_amount"` //TotalPrice
	TotalWeight                 float64    `json:"total_weight"`
	SubtotalExclTransport       float64    `json:"subtotal_excl_transport"`        //TotalNetPrice
	SubtotalWeightExclTransport float64    `json:"subtotal_weight_excl_transport"` //TotalNetPriceWeight
	PaymentMethod               string     `json:"payment_method"`
	PeymentTermCode             string     `json:"peyment_term_code"`
	SalePersonCode              string     `json:"sale_person_code"`
	EffectiveDatePrice          *time.Time `json:"effective_date_price"`
	ExpirePriceDay              int        `json:"expire_price_day"`
	ExpirePriceDate             *time.Time `json:"expire_price_date"`
	PassPriceList               string     `json:"pass_price_list"`
	PassAtpCheck                string     `json:"pass_atp_check"`
	PassCreditLimit             string     `json:"pass_credit_limit"`
	PassPriceExpire             string     `json:"pass_price_expire"`
	Status                      string     `json:"status"`
	Remark                      string     `json:"remark"`
	IsApproved                  bool       `json:"is_approved"`
	StatusApprove               string     `json:"status_approve"`
	TotalVat                    float64    `json:"total_vat"`
	TotalDiscount               float64    `json:"total_discount"`
	SubtotalExclVat             float64    `json:"subtotal_excl_vat"`
	TotalTransportCostVat       float64    `json:"total_transport_cost_vat"`
	RemarkApproval              string     `json:"remark_approval"`
	Revision                    float64    `json:"revision"`
	QuotationCodeRef            string     `json:"quotation_code_ref"`
	CreditTerm                  string     `json:"credit_term"`
	PayerTerm                   string     `json:"payer_term"`
	CreateDate                  *time.Time `json:"create_date"`
	CreateBy                    string     `json:"create_by"`
	UpdateDate                  *time.Time `json:"update_date"`
	UpdateBy                    string     `json:"update_by"`
}

func (Quotation) TableName() string { return "quotation" }

type QuotationItem struct {
	ID                             uuid.UUID  `json:"id"`
	QuotationID                    uuid.UUID  `json:"quotation_id"`
	QuotationItem                  string     `json:"quotation_item"`
	ProductCode                    string     `json:"product_code"`
	ProductDesc                    string     `json:"product_desc"`
	Qty                            float64    `json:"qty"`
	Unit                           string     `json:"unit"`
	PriceListUnit                  float64    `json:"price_list_unit"`
	SaleQty                        float64    `json:"sale_qty"`
	SaleUnit                       string     `json:"sale_unit"`
	SaleUnitType                   string     `json:"sale_unit_type"`
	PassPrice                      string     `json:"pass_price"`
	PassWeight                     string     `json:"pass_weight"`
	PriceUnit                      float64    `json:"price_unit"`
	TotalAmount                    float64    `json:"total_amount"` //TotalPrice
	TransportCostUnit              float64    `json:"transport_cost_unit"`
	SubtotalExclTransport          float64    `json:"subtotal_excl_transport"`       //TotalNetPrice
	NetPriceUnitExclTransport      float64    `json:"net_price_unit_excl_transport"` //NetPriceUnit
	WeightUnit                     float64    `json:"weight_unit"`
	AvgWeightUnit                  float64    `json:"avg_weight_unit"`
	TotalWeight                    float64    `json:"total_weight"`
	TransportCostWeightUnit        float64    `json:"transport_cost_weight_unit"`
	SubtotalWeightExclTransport    float64    `json:"subtotal_weight_excl_transport"`      //TotalNetPriceWeight
	NetPricePerWeightExclTransport float64    `json:"net_price_per_weight_excl_transport"` //NetPriceWeightUnit
	Status                         string     `json:"status"`
	Remark                         string     `json:"remark"`
	SubtotalExclVat                float64    `json:"subtotal_excl_vat"`
	TotalVat                       float64    `json:"total_vat"`
	UnitUom                        string     `json:"unit_uom"`
	TotalDiscount                  float64    `json:"total_discount"`
	TotalDiscountPercent           float64    `json:"total_discount_percent"`
	CreateDate                     *time.Time `json:"create_date"`
	CreateBy                       string     `json:"create_by"`
	UpdateDate                     *time.Time `json:"update_date"`
	UpdateBy                       string     `json:"update_by"`
}

func (QuotationItem) TableName() string { return "quotation_item" }
