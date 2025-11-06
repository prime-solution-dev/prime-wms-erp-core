package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ALTER TABLE public.price_list_sub_group ADD udf_json json NULL;

type PriceListGroup struct {
	ID                   uuid.UUID             `json:"id"`
	CompanyCode          string                `json:"company_code"`
	SiteCode             string                `json:"site_code"`
	GroupCode            string                `json:"group_code"`
	PriceUnit            float64               `json:"price_unit"`
	PriceWeight          float64               `json:"price_weight"`
	BeforePriceUnit      float64               `json:"before_price_unit"`
	BeforePriceWeight    float64               `json:"before_price_weight"`
	Currency             string                `json:"currency"`
	EffectiveDate        *time.Time            `json:"effective_date"`
	Remark               string                `json:"remark"`
	GroupKey             string                `json:"group_key"`
	CreateBy             string                `json:"create_by"`
	CreateDtm            time.Time             `json:"create_dtm"`
	UpdateBy             string                `json:"update_by"`
	UpdateDtm            time.Time             `json:"update_dtm"`
	PriceListGroupTerms  []PriceListGroupTerm  `gorm:"foreignKey:PriceListGroupID;references:ID" json:"price_list_group_terms"`
	PriceListGroupExtras []PriceListGroupExtra `gorm:"foreignKey:PriceListGroupID;references:ID" json:"price_list_group_extras"`
	PriceListSubGroups   []PriceListSubGroup   `gorm:"foreignKey:PriceListGroupID;references:ID" json:"price_list_sub_groups"`
}

func (PriceListGroup) TableName() string { return "price_list_group" }

type PriceListGroupHistory struct {
	ID                uuid.UUID  `json:"id"`
	CompanyCode       string     `json:"company_code"`
	SiteCode          string     `json:"site_code"`
	GroupCode         string     `json:"group_code"`
	PriceUnit         float64    `json:"price_unit"`
	PriceWeight       float64    `json:"price_weight"`
	BeforePriceUnit   float64    `json:"before_price_unit"`
	BeforePriceWeight float64    `json:"before_price_weight"`
	Currency          string     `json:"currency"`
	EffectiveDate     *time.Time `json:"effective_date"`
	ExpiryDate        *time.Time `json:"expiry_date"`
	Remark            string     `json:"remark"`
	CreateBy          string     `json:"create_by"`
	CreateDtm         time.Time  `json:"create_dtm"`
	UpdateBy          string     `json:"update_by"`
	UpdateDtm         time.Time  `json:"update_dtm"`
}

func (PriceListGroupHistory) TableName() string { return "price_list_group_history" }

type PriceListGroupTerm struct {
	ID               uuid.UUID `json:"id"`
	PriceListGroupID uuid.UUID `json:"price_list_group_id"`
	TermCode         string    `json:"term_code"`
	Pdc              float64   `json:"pdc"`
	PdcPercent       int       `json:"pdc_percent"`
	Due              float64   `json:"due"`
	DuePercent       int       `json:"due_percent"`
	CreateBy         string    `json:"create_by"`
	CreateDtm        time.Time `json:"create_dtm"`
	UpdateBy         string    `json:"update_by"`
	UpdateDtm        time.Time `json:"update_dtm"`
}

func (PriceListGroupTerm) TableName() string { return "price_list_group_term" }

type PriceListGroupExtra struct {
	ID               uuid.UUID `json:"id"`
	PriceListGroupID uuid.UUID `json:"price_list_group_id"`
	ExtraKey         string    `json:"extra_key"`
	LengthExtraKey   int       `json:"length_extra_key"`
	Operator         string    `json:"operator"`
	CondRangeMin     float64   `json:"cond_range_min"`
	CondRangeMax     float64   `json:"cond_range_max"`
	CreateBy         string    `json:"create_by"`
	CreateDtm        time.Time `json:"create_dtm"`
	UpdateBy         string    `json:"update_by"`
	UpdateDtm        time.Time `json:"update_dtm"`
}

func (PriceListGroupExtra) TableName() string { return "price_list_group_extra" }

type PriceListGroupExtraKey struct {
	ID           uuid.UUID `json:"id"`
	GroupExtraID uuid.UUID `json:"group_extra_id"`
	Code         string    `json:"code"`
	Value        string    `json:"value"`
	Seq          int       `json:"seq"`
}

func (PriceListGroupExtraKey) TableName() string { return "price_list_group_extra_key" }

type PriceListSubGroup struct {
	ID                        uuid.UUID              `json:"id"`
	PriceListGroupID          uuid.UUID              `json:"price_list_group_id"`
	SubgroupKey               string                 `json:"subgroup_key"`
	IsTrading                 bool                   `json:"is_trading"`
	PriceUnit                 float64                `json:"price_unit"`
	ExtraPriceUnit            float64                `json:"extra_price_unit"`
	TermPriceUnit             float64                `json:"term_price_unit"`
	TotalNetPriceUnit         float64                `json:"total_net_price_unit"`
	PriceWeight               float64                `json:"price_weight"`
	ExtraPriceWeight          float64                `json:"extra_price_weight"`
	TermPriceWeight           float64                `json:"term_price_weight"`
	TotalNetPriceWeight       float64                `json:"total_net_price_weight"`
	BeforePriceUnit           float64                `json:"before_price_unit"`
	BeforeExtraPriceUnit      float64                `json:"before_extra_price_unit"`
	BeforeTermPriceUnit       float64                `json:"before_term_price_unit"`
	BeforeTotalNetPriceUnit   float64                `json:"before_total_net_price_unit"`
	BeforePriceWeight         float64                `json:"before_price_weight"`
	BeforeExtraPriceWeight    float64                `json:"before_extra_price_weight"`
	BeforeTermPriceWeight     float64                `json:"before_term_price_weight"`
	BeforeTotalNetPriceWeight float64                `json:"before_total_net_price_weight"`
	EffectiveDate             *time.Time             `json:"effective_date"`
	Remark                    string                 `json:"remark"`
	CreateBy                  string                 `json:"create_by"`
	CreateDtm                 time.Time              `json:"create_dtm"`
	UpdateBy                  string                 `json:"update_by"`
	UpdateDtm                 time.Time              `json:"update_dtm"`
	UdfJson                   json.RawMessage        `json:"udf_json"`
	PriceListSubGroupKeys     []PriceListSubGroupKey `gorm:"foreignKey:SubGroupID;references:ID" json:"price_list_sub_group_keys"`
}

func (PriceListSubGroup) TableName() string { return "price_list_sub_group" }

type PriceListSubGroupKey struct {
	ID         uuid.UUID `json:"id"`
	SubGroupID uuid.UUID `json:"sub_group_id"`
	Code       string    `json:"code"`
	Value      string    `json:"value"`
	Seq        int       `json:"seq"`
}

func (PriceListSubGroupKey) TableName() string { return "price_list_sub_group_key" }

type PriceListSubGroupHistory struct {
	ID                        uuid.UUID  `json:"id"`
	PriceListGroupID          uuid.UUID  `json:"price_list_group_id"`
	SubgroupKey               string     `json:"subgroup_key"`
	IsTrading                 bool       `json:"is_trading"`
	PriceUnit                 float64    `json:"price_unit"`
	ExtraPriceUnit            float64    `json:"extra_price_unit"`
	TermPriceUnit             float64    `json:"term_price_unit"`
	TotalNetPriceUnit         float64    `json:"total_net_price_unit"`
	PriceWeight               float64    `json:"price_weight"`
	ExtraPriceWeight          float64    `json:"extra_price_weight"`
	TermPriceWeight           float64    `json:"term_price_weight"`
	TotalNetPriceWeight       float64    `json:"total_net_price_weight"`
	BeforePriceUnit           float64    `json:"before_price_unit"`
	BeforeExtraPriceUnit      float64    `json:"before_extra_price_unit"`
	BeforeTermPriceUnit       float64    `json:"before_term_price_unit"`
	BeforeTotalNetPriceUnit   float64    `json:"before_total_net_price_unit"`
	BeforePriceWeight         float64    `json:"before_price_weight"`
	BeforeExtraPriceWeight    float64    `json:"before_extra_price_weight"`
	BeforeTermPriceWeight     float64    `json:"before_term_price_weight"`
	BeforeTotalNetPriceWeight float64    `json:"before_total_net_price_weight"`
	EffectiveDate             *time.Time `json:"effective_date"`
	ExpiryDate                *time.Time `json:"expiry_date"`
	Remark                    string     `json:"remark"`
	CreateBy                  string     `json:"create_by"`
	CreateDtm                 time.Time  `json:"create_dtm"`
	UpdateBy                  string     `json:"update_by"`
	UpdateDtm                 time.Time  `json:"update_dtm"`
}

func (PriceListSubGroupHistory) TableName() string { return "price_list_sub_group_history" }

type PriceListSubGroupKeyHistory struct {
	ID                uuid.UUID `json:"id"`
	SubGroupHistoryID uuid.UUID `json:"sub_group_history_id"`
	Code              string    `json:"code"`
	Value             string    `json:"value"`
	Seq               int       `json:"seq"`
}

func (PriceListSubGroupKeyHistory) TableName() string { return "price_list_sub_group_key_history" }

type PaymentTerm struct {
	ID        uuid.UUID `json:"id"`
	TermCode  string    `json:"term_code"`
	TermName  string    `json:"term_name"`
	TermType  string    `json:"term_type"`
	CreateBy  string    `json:"create_by"`
	CreateDtm time.Time `json:"create_dtm"`
	UpdateBy  string    `json:"update_by"`
	UpdateDtm time.Time `json:"update_dtm"`
}

func (PaymentTerm) TableName() string { return "payment_term" }

// DTOs
type GetPriceListRequest struct {
	CompanyCode string   `json:"company_code"`
	SiteCode    string   `json:"site_code"`
	GroupKeys   []string `json:"group_keys"`
}

type PriceListTermResponse struct {
	ID               string  `json:"id"`
	PriceListGroupID string  `json:"price_list_group_id"`
	TermCode         string  `json:"term_code"`
	TermName         string  `json:"term_name"`
	TermType         string  `json:"term_type"`
	Pdc              float64 `json:"pdc"`
	PdcPercent       int     `json:"pdc_percent"`
	Due              float64 `json:"due"`
	DuePercent       int     `json:"due_percent"`
	CreateBy         string  `json:"create_by"`
	CreateDtm        string  `json:"create_dtm"`
	UpdateBy         string  `json:"update_by"`
	UpdateDtm        string  `json:"update_dtm"`
}

type PriceListExtraResponse struct {
	ID               string  `json:"id"`
	PriceListGroupID string  `json:"price_list_group_id"`
	ExtraKey         string  `json:"extra_key"`
	LengthExtraKey   int     `json:"length_extra_key"`
	Operator         string  `json:"operator"`
	CondRangeMin     float64 `json:"cond_range_min"`
	CondRangeMax     float64 `json:"cond_range_max"`
	CreateBy         string  `json:"create_by"`
	CreateDtm        string  `json:"create_dtm"`
	UpdateBy         string  `json:"update_by"`
	UpdateDtm        string  `json:"update_dtm"`
}

type PriceListSubGroupKeyResponse struct {
	ID         string `json:"id"`
	SubGroupID string `json:"sub_group_id"`
	GroupCode  string `json:"group_code"`
	GroupName  string `json:"group_name"`
	ValueCode  string `json:"value_code"`
	ValueName  string `json:"value_name"`
	Seq        int    `json:"seq"`
}

type PriceListSubGroupResponse struct {
	ID                        string                         `json:"id"`
	PriceListGroupID          string                         `json:"price_list_group_id"`
	SubgroupKey               string                         `json:"subgroup_key"`
	IsTrading                 bool                           `json:"is_trading"`
	PriceUnit                 float64                        `json:"price_unit"`
	ExtraPriceUnit            float64                        `json:"extra_price_unit"`
	TermPriceUnit             float64                        `json:"term_price_unit"`
	TotalNetPriceUnit         float64                        `json:"total_net_price_unit"`
	PriceWeight               float64                        `json:"price_weight"`
	ExtraPriceWeight          float64                        `json:"extra_price_weight"`
	TermPriceWeight           float64                        `json:"term_price_weight"`
	TotalNetPriceWeight       float64                        `json:"total_net_price_weight"`
	BeforePriceUnit           float64                        `json:"before_price_unit"`
	BeforeExtraPriceUnit      float64                        `json:"before_extra_price_unit"`
	BeforeTermPriceUnit       float64                        `json:"before_term_price_unit"`
	BeforeTotalNetPriceUnit   float64                        `json:"before_total_net_price_unit"`
	BeforePriceWeight         float64                        `json:"before_price_weight"`
	BeforeExtraPriceWeight    float64                        `json:"before_extra_price_weight"`
	BeforeTermPriceWeight     float64                        `json:"before_term_price_weight"`
	BeforeTotalNetPriceWeight float64                        `json:"before_total_net_price_weight"`
	EffectiveDate             string                         `json:"effective_date"`
	Remark                    string                         `json:"remark"`
	CreateBy                  string                         `json:"create_by"`
	CreateDtm                 string                         `json:"create_dtm"`
	UpdateBy                  string                         `json:"update_by"`
	UpdateDtm                 string                         `json:"update_dtm"`
	UdfJson                   json.RawMessage                `json:"udf_json,omitempty"`
	SubGroupKeys              []PriceListSubGroupKeyResponse `json:"sub_group_keys"`
}

type GetPriceListResponse struct {
	ID                string                      `json:"id"`
	CompanyCode       string                      `json:"company_code"`
	SiteCode          string                      `json:"site_code"`
	GroupCode         string                      `json:"group_code"`
	PriceUnit         float64                     `json:"price_unit"`
	PriceWeight       float64                     `json:"price_weight"`
	BeforePriceUnit   float64                     `json:"before_price_unit"`
	BeforePriceWeight float64                     `json:"before_price_weight"`
	Currency          string                      `json:"currency"`
	EffectiveDate     string                      `json:"effective_date"`
	Remark            string                      `json:"remark"`
	GroupKey          string                      `json:"group_key"`
	GroupKeyName      string                      `json:"group_key_name"`
	CreateBy          string                      `json:"create_by"`
	CreateDtm         string                      `json:"create_dtm"`
	UpdateBy          string                      `json:"update_by"`
	UpdateDtm         string                      `json:"update_dtm"`
	Terms             []PriceListTermResponse     `json:"terms"`
	Extras            []PriceListExtraResponse    `json:"extras"`
	SubGroups         []PriceListSubGroupResponse `json:"sub_groups"`
}

type CreatePriceListGroupTermRequest struct {
	TermCode   string  `json:"term_code"`
	Pdc        float64 `json:"pdc"`
	PdcPercent int     `json:"pdc_percent"`
	Due        float64 `json:"due"`
	DuePercent int     `json:"due_percent"`
}

type CreatePriceListBaseRequest struct {
	CompanyCode   string                            `json:"company_code"`
	SiteCode      string                            `json:"site_code"`
	GroupCode     string                            `json:"group_code"`
	PriceUnit     float64                           `json:"price_unit"`
	PriceWeight   float64                           `json:"price_weight"`
	Currency      string                            `json:"currency"`
	EffectiveDate *time.Time                        `json:"effective_date"`
	Remark        string                            `json:"remark"`
	Terms         []CreatePriceListGroupTermRequest `json:"terms"`
}

type UpdatePriceListBaseRequest struct {
	ID            uuid.UUID            `json:"id"`
	PriceUnit     float64              `json:"price_unit"`
	PriceWeight   float64              `json:"price_weight"`
	Currency      string               `json:"currency"`
	EffectiveDate *time.Time           `json:"effective_date"`
	Remark        string               `json:"remark"`
	Terms         []PriceListGroupTerm `json:"terms"`
}

type DeletePriceListBaseRequest struct {
	ID []string `json:"id"`
}
