package quotationService

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"prime-erp-core/internal/db"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetQuotationRequest struct {
	ID            []string `json:"id"`
	QuotationCode []string `json:"quotation_code"`
	SiteCode      []string `json:"site_code"`
	CompanyCode   []string `json:"company_code"`
	Page          int      `json:"page"`
	PageSize      int      `json:"page_size"`
}

func (GetQuotationResponse) TableName() string { return "quotation" }

func (GetQuotationItemResponse) TableName() string { return "quotation_item" }

type GetQuotationResponse struct {
	ID                          uuid.UUID                  `gorm:"type:uuid;primary_key" json:"id"`
	QuotationCode               string                     `gorm:"type:varchar(50)" json:"quotation_code"`
	CompanyCode                 string                     `gorm:"type:varchar(50)" json:"company_code"`
	SiteCode                    string                     `gorm:"type:varchar(50)" json:"site_code"`
	CustomerCode                string                     `gorm:"type:varchar(50)" json:"customer_code"`
	CustomerName                string                     `gorm:"type:varchar(255)" json:"customer_name"`
	DeliveryDate                *time.Time                 `gorm:"type:date" json:"delivery_date"`
	SoldToCode                  string                     `gorm:"type:varchar(50)" json:"sold_to_code"`
	SoldToAddress               string                     `gorm:"type:varchar(255)" json:"sold_to_address"`
	BillToCode                  string                     `gorm:"type:varchar(50)" json:"bill_to_code"`
	BillToAddress               string                     `gorm:"type:varchar(255)" json:"bill_to_address"`
	ShipToCode                  string                     `gorm:"type:varchar(50)" json:"ship_to_code"`
	ShipToType                  string                     `gorm:"type:varchar(50)" json:"ship_to_type"`
	ShipToAddress               string                     `gorm:"type:varchar(255)" json:"ship_to_address"`
	DeliveryMethod              string                     `gorm:"type:varchar(50)" json:"delivery_method"`
	TransportCostType           string                     `gorm:"type:varchar(50)" json:"transport_cost_type"`
	TotalTransportCost          float64                    `gorm:"type:numeric" json:"total_transport_cost"`
	PassPrice                   bool                       `gorm:"type:boolean" json:"pass_price"`
	TotalAmount                 float64                    `gorm:"type:numeric" json:"total_amount"` //TotalPrice
	TotalWeight                 float64                    `gorm:"type:numeric" json:"total_weight"`
	SubtotalExclTransport       float64                    `gorm:"type:numeric" json:"subtotal_excl_transport"`        //TotalNetPrice
	SubtotalWeightExclTransport float64                    `gorm:"type:numeric" json:"subtotal_weight_excl_transport"` //TotalNetPriceWeight
	PaymentMethod               string                     `gorm:"type:varchar(50)" json:"payment_method"`
	PeymentTermCode             string                     `gorm:"type:varchar(50)" json:"peyment_term_code"`
	SalePersonCode              string                     `gorm:"type:varchar(50)" json:"sale_person_code"`
	EffectiveDatePrice          *time.Time                 `gorm:"type:date" json:"effective_date_price"`
	ExpirePriceDay              int                        `gorm:"type:int" json:"expire_price_day"`
	ExpirePriceDate             *time.Time                 `gorm:"type:date" json:"expire_price_date"`
	PassPriceList               string                     `gorm:"type:varchar(50)" json:"pass_price_list"`
	PassAtpCheck                string                     `gorm:"type:varchar(50)" json:"pass_atp_check"`
	PassCreditLimit             string                     `gorm:"type:varchar(50)" json:"pass_credit_limit"`
	PassPriceExpire             string                     `gorm:"type:varchar(50)" json:"pass_price_expire"`
	Status                      string                     `gorm:"type:varchar(50)" json:"status"`
	Remark                      string                     `gorm:"type:varchar(255)" json:"remark"`
	IsApproved                  bool                       `gorm:"type:boolean" json:"is_approved"`
	StatusApprove               string                     `gorm:"type:varchar(50)" json:"status_approve"`
	TotalVat                    float64                    `gorm:"type:numeric" json:"total_vat"`
	TotalDiscount               float64                    `gorm:"type:numeric" json:"total_discount"`
	SubtotalExclVat             float64                    `gorm:"type:numeric" json:"subtotal_excl_vat"`
	TotalTransportCostVat       float64                    `gorm:"type:numeric" json:"total_transport_cost_vat"`
	RemarkApproval              string                     `gorm:"type:varchar(255)" json:"remark_approval"`
	CreateDate                  *time.Time                 `gorm:"type:date" json:"create_date"`
	CreateBy                    string                     `gorm:"type:varchar(50)" json:"create_by"`
	UpdateDate                  *time.Time                 `gorm:"type:date" json:"update_date"`
	UpdateBy                    string                     `gorm:"type:varchar(50)" json:"update_by"`
	Items                       []GetQuotationItemResponse `gorm:"foreignKey:QuotationID" json:"items"`
}

type GetQuotationItemResponse struct {
	ID                             uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	QuotationID                    uuid.UUID  `gorm:"type:uuid" json:"quotation_id"`
	QuotationItem                  string     `gorm:"type:varchar(50)" json:"quotation_item"`
	ProductCode                    string     `gorm:"type:varchar(50)" json:"product_code"`
	ProductDesc                    string     `gorm:"type:varchar(255)" json:"product_desc"`
	Qty                            float64    `gorm:"type:numeric" json:"qty"`
	Unit                           string     `gorm:"type:varchar(50)" json:"unit"`
	PriceListUnit                  float64    `gorm:"type:numeric" json:"price_list_unit"`
	SaleQty                        float64    `gorm:"type:numeric" json:"sale_qty"`
	SaleUnit                       string     `gorm:"type:varchar(50)" json:"sale_unit"`
	SaleUnitType                   string     `gorm:"type:varchar(50)" json:"sale_unit_type"`
	PassPrice                      string     `gorm:"type:varchar(50)" json:"pass_price"`
	PassWeight                     string     `gorm:"type:varchar(50)" json:"pass_weight"`
	PriceUnit                      float64    `gorm:"type:numeric" json:"price_unit"`
	TotalAmount                    float64    `gorm:"type:numeric" json:"total_amount"` //TotalPrice
	TransportCostUnit              float64    `gorm:"type:numeric" json:"transport_cost_unit"`
	SubtotalExclTransport          float64    `gorm:"type:numeric" json:"subtotal_excl_transport"`       //TotalNetPrice
	NetPriceUnitExclTransport      float64    `gorm:"type:numeric" json:"net_price_unit_excl_transport"` //NetPriceUnit
	WeightUnit                     float64    `gorm:"type:numeric" json:"weight_unit"`
	AvgWeightUnit                  float64    `gorm:"type:numeric" json:"avg_weight_unit"`
	TotalWeight                    float64    `gorm:"type:numeric" json:"total_weight"`
	TransportCostWeightUnit        float64    `gorm:"type:numeric" json:"transport_cost_weight_unit"`
	SubtotalWeightExclTransport    float64    `gorm:"type:numeric" json:"subtotal_weight_excl_transport"`      //TotalNetPriceWeight
	NetPricePerWeightExclTransport float64    `gorm:"type:numeric" json:"net_price_per_weight_excl_transport"` //NetPriceWeightUnit
	Status                         string     `gorm:"type:varchar(50)" json:"status"`
	Remark                         string     `gorm:"type:varchar(255)" json:"remark"`
	SubtotalExclVat                float64    `gorm:"type:numeric" json:"subtotal_excl_vat"`
	TotalVat                       float64    `gorm:"type:numeric" json:"total_vat"`
	UnitUom                        string     `gorm:"type:varchar(50)" json:"unit_uom"`
	TotalDiscount                  float64    `gorm:"type:numeric" json:"total_discount"`
	TotalDiscountPercent           float64    `gorm:"type:numeric" json:"total_discount_percent"`
	CreateDate                     *time.Time `gorm:"type:date" json:"create_date"`
	CreateBy                       string     `gorm:"type:varchar(50)" json:"create_by"`
	UpdateDate                     *time.Time `gorm:"type:date" json:"update_date"`
	UpdateBy                       string     `gorm:"type:varchar(50)" json:"update_by"`
}

type ResultQuotationResponse struct {
	Total      int                    `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	TotalPages int                    `json:"total_pages"`
	Quotations []GetQuotationResponse `json:"quotations"`
}

func GetQuotation(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var res []GetQuotationResponse
	var req GetQuotationRequest

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

	query := gormx.Preload("Items").
		Order("quotation.update_date DESC")

	if len(req.ID) > 0 {
		query = query.Where("id IN ?", req.ID)
	}

	if len(req.QuotationCode) > 0 {
		query = query.Where("quotation_code IN ?", req.QuotationCode)
	}

	if len(req.SiteCode) > 0 {
		query = query.Where("site_code IN ?", req.SiteCode)
	}

	if len(req.CompanyCode) > 0 {
		query = query.Where("company_code IN ?", req.CompanyCode)
	}

	var count int64
	gormx.Model(&GetQuotationResponse{}).Count(&count)

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

	resultQuotation := ResultQuotationResponse{
		Total:      int(totalRecords),
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		Quotations: res,
	}

	return resultQuotation, nil
}
