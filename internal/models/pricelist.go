package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type PriceListGroup struct {
	ID                   uuid.UUID             `gorm:"primary_key;not null" json:"id"`
	CompanyCode          string                `json:"company_code"`
	SiteCode             string                `json:"site_code"`
	GroupCode            string                `json:"group_code"`
	GroupName            string                `json:"group_name"`
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
	ID                uuid.UUID  `gorm:"primary_key;not null" json:"id"`
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
	ID               uuid.UUID  `gorm:"primary_key;not null" json:"id"`
	PriceListGroupID uuid.UUID  `json:"price_list_group_id"`
	TermCode         string     `json:"term_code"`
	Pdc              float64    `json:"pdc"`
	PdcPercent       int        `json:"pdc_percent"`
	Due              float64    `json:"due"`
	DuePercent       int        `json:"due_percent"`
	CreateBy         string     `json:"create_by"`
	CreateDtm        *time.Time `json:"create_dtm"`
	UpdateBy         string     `json:"update_by"`
	UpdateDtm        *time.Time `json:"update_dtm"`
}

func (PriceListGroupTerm) TableName() string { return "price_list_group_term" }

type PriceListGroupExtra struct {
	ID                      uuid.UUID                `gorm:"primary_key;not null" json:"id"`
	PriceListGroupID        uuid.UUID                `json:"price_list_group_id"`
	ExtraKey                string                   `json:"extra_key"`
	ConditionCode           string                   `json:"condition_code"`
	ValueInt                int                      `json:"value_int"`
	LengthExtraKey          int                      `json:"length_extra_key"`
	Operator                string                   `json:"operator"`
	CondRangeMin            float64                  `json:"cond_range_min"`
	CondRangeMax            float64                  `json:"cond_range_max"`
	CreateBy                string                   `json:"create_by"`
	CreateDtm               *time.Time               `json:"create_dtm"`
	UpdateBy                string                   `json:"update_by"`
	UpdateDtm               *time.Time               `json:"update_dtm"`
	PriceListGroupExtraKeys []PriceListGroupExtraKey `gorm:"foreignKey:GroupExtraID;references:ID" json:"price_list_group_extra_keys"`
}

func (PriceListGroupExtra) TableName() string { return "price_list_group_extra" }

type PriceListGroupExtraKey struct {
	ID           uuid.UUID `gorm:"primary_key;not null" json:"id"`
	GroupExtraID uuid.UUID `json:"group_extra_id"`
	Code         string    `json:"code"`
	Value        string    `json:"value"`
	Seq          int       `json:"seq"`
}

func (PriceListGroupExtraKey) TableName() string { return "price_list_group_extra_key" }

type PriceListExtraConfig struct {
	ID         uuid.UUID       `gorm:"primary_key;not null" json:"id"`
	GroupCode  string          `json:"group_code"`
	IsActive   bool            `json:"is_active"`
	ConfigJson json.RawMessage `json:"config_json"`
	CreateDtm  time.Time       `json:"create_dtm"`
	CreateBy   string          `json:"create_by"`
	UpdateDtm  time.Time       `json:"update_dtm"`
	UpdateBy   string          `json:"update_by"`
}

func (PriceListExtraConfig) TableName() string { return "price_list_extra_config" }

type PriceListSubGroup struct {
	ID                        uuid.UUID              `json:"id"`
	PriceListGroupID          uuid.UUID              `json:"price_list_group_id"`
	SubGroupCode              string                 `json:"subgroup_code" gorm:"column:subgroup_code"`
	SubgroupKey               string                 `json:"subgroup_key"`
	IsTrading                 bool                   `json:"is_trading"`
	PriceUnit                 float64                `json:"price_unit"`
	ExtraPriceUnit            float64                `json:"extra_price_unit"`
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
	CreateDtm                 *time.Time             `json:"create_dtm"`
	UpdateBy                  string                 `json:"update_by"`
	UpdateDtm                 *time.Time             `json:"update_dtm"`
	UdfJson                   json.RawMessage        `json:"udf_json"`
	PriceListGroup            PriceListGroup         `gorm:"foreignKey:PriceListGroupID;references:ID" json:"price_list_group"`
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
	CreateDtm                 *time.Time `json:"create_dtm"`
	UpdateBy                  string     `json:"update_by"`
	UpdateDtm                 *time.Time `json:"update_dtm"`
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
	ID        uuid.UUID  `json:"id"`
	TermCode  string     `json:"term_code"`
	TermName  string     `json:"term_name"`
	TermType  string     `json:"term_type"`
	CreateBy  string     `json:"create_by"`
	CreateDtm *time.Time `json:"create_dtm"`
	UpdateBy  string     `json:"update_by"`
	UpdateDtm *time.Time `json:"update_dtm"`
}

func (PaymentTerm) TableName() string { return "payment_term" }

// DTOs
type GetPriceListRequest struct {
	CompanyCode string   `json:"company_code"`
	SiteCode    string   `json:"site_code"`
	GroupCodes  []string `json:"group_codes"`
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

type PriceListGroupExtraKeyResponse struct {
	ID           string `json:"id"`
	GroupExtraID string `json:"group_extra_id"`
	GroupCode    string `json:"group_code"`
	GroupName    string `json:"group_name"`
	ValueCode    string `json:"value_code"`
	ValueName    string `json:"value_name"`
	Seq          int    `json:"seq"`
}

type PriceListExtraResponse struct {
	ID                      string                           `json:"id"`
	PriceListGroupID        string                           `json:"price_list_group_id"`
	ExtraKey                string                           `json:"extra_key"`
	ConditionCode           string                           `json:"condition_code"`
	ValueInt                float64                          `json:"value_int"`
	LengthExtraKey          float64                          `json:"length_extra_key"`
	Operator                string                           `json:"operator"`
	CondRangeMin            float64                          `json:"cond_range_min"`
	CondRangeMax            float64                          `json:"cond_range_max"`
	CreateBy                string                           `json:"create_by"`
	CreateDtm               string                           `json:"create_dtm"`
	UpdateBy                string                           `json:"update_by"`
	UpdateDtm               string                           `json:"update_dtm"`
	PriceListGroupExtraKeys []PriceListGroupExtraKeyResponse `json:"price_list_group_extra_keys"`
}

type PriceListExtraConfigResponse struct {
	GroupCode  string          `json:"group_code"`
	IsActive   bool            `json:"is_active"`
	ConfigJson json.RawMessage `json:"config_json"`
}

type PriceListExtraResponseWithConfig struct {
	Config PriceListExtraConfigResponse `json:"config"`
	Data   []PriceListExtraResponse     `json:"data"`
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

type InventoryWeightResponse struct {
	InventoryWeightKeyCode string  `json:"inventory_weightkey_code"`
	ProductCode            string  `json:"product_code"`
	CompanyCode            string  `json:"company_code"`
	SiteCode               string  `json:"site_code"`
	BatchNo                string  `json:"batch_no"`
	SerialCode             string  `json:"serial_code"`
	AvgProduct             float64 `json:"avg_product"`
	AvgBatch               float64 `json:"avg_batch"`
	AvgSerial              float64 `json:"avg_serial"`
}

type PriceListSubGroupResponse struct {
	ID                        string                         `json:"id"`
	PriceListGroupID          string                         `json:"price_list_group_id"`
	SubgroupKey               string                         `json:"subgroup_key"`
	IsTrading                 bool                           `json:"is_trading"`
	PriceUnit                 float64                        `json:"price_unit"`
	ExtraPriceUnit            float64                        `json:"extra_price_unit"`
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
	EffectiveDate             *string                        `json:"effective_date"`
	Remark                    string                         `json:"remark"`
	CreateBy                  string                         `json:"create_by"`
	CreateDtm                 string                         `json:"create_dtm"`
	UpdateBy                  string                         `json:"update_by"`
	UpdateDtm                 string                         `json:"update_dtm"`
	UdfJson                   json.RawMessage                `json:"udf_json,omitempty"`
	SubGroupKeys              []PriceListSubGroupKeyResponse `json:"sub_group_keys"`
	InventoryWeight           []InventoryWeightResponse      `json:"inventory_weight,omitempty"`
	SupplierCode              string                         `json:"supplier_code,omitempty"`
}

type GetPriceListResponse struct {
	ID                string                           `json:"id"`
	CompanyCode       string                           `json:"company_code"`
	SiteCode          string                           `json:"site_code"`
	GroupCode         string                           `json:"group_code"`
	GroupName         string                           `json:"group_name"`
	PriceUnit         float64                          `json:"price_unit"`
	PriceWeight       float64                          `json:"price_weight"`
	BeforePriceUnit   float64                          `json:"before_price_unit"`
	BeforePriceWeight float64                          `json:"before_price_weight"`
	Currency          string                           `json:"currency"`
	EffectiveDate     *string                          `json:"effective_date"`
	Remark            string                           `json:"remark"`
	GroupKey          string                           `json:"group_key"`
	CreateBy          string                           `json:"create_by"`
	CreateDtm         string                           `json:"create_dtm"`
	UpdateBy          string                           `json:"update_by"`
	UpdateDtm         string                           `json:"update_dtm"`
	Terms             []PriceListTermResponse          `json:"terms"`
	Extras            PriceListExtraResponseWithConfig `json:"extras"`
	SubGroups         []PriceListSubGroupResponse      `json:"sub_groups"`
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

type UpdatePriceListGroupTermRequest struct {
	ID               uuid.UUID  `json:"id"`
	PriceListGroupID uuid.UUID  `json:"price_list_group_id"`
	TermCode         string     `json:"term_code"`
	Pdc              float64    `json:"pdc"`
	PdcPercent       int        `json:"pdc_percent"`
	Due              float64    `json:"due"`
	DuePercent       int        `json:"due_percent"`
	CreateBy         string     `json:"create_by"`
	CreateDtm        *time.Time `json:"create_dtm"`
	UpdateBy         string     `json:"update_by"`
	UpdateDtm        *time.Time `json:"update_dtm"`
}

type UpdatePriceListBaseRequest struct {
	ID            uuid.UUID                         `json:"id"`
	PriceUnit     float64                           `json:"price_unit"`
	PriceWeight   float64                           `json:"price_weight"`
	Currency      string                            `json:"currency"`
	EffectiveDate *time.Time                        `json:"effective_date"`
	Remark        string                            `json:"remark"`
	Terms         []UpdatePriceListGroupTermRequest `json:"terms"`
}

type DeletePriceListBaseRequest struct {
	ID []string `json:"id"`
}

type UpdatePriceListSubGroupItem struct {
	SubGroupID              uuid.UUID       `json:"subgroup_id" binding:"required"`
	IsTrading               *bool           `json:"is_trading,omitempty"`
	PriceUnit               *float64        `json:"price_unit,omitempty" binding:"omitempty,min=0"`
	ExtraPriceUnit          *float64        `json:"extra_price_unit,omitempty" binding:"omitempty,min=0"`
	TermPriceUnit           *float64        `json:"term_price_unit,omitempty" binding:"omitempty,min=0"`
	TotalNetPriceUnit       *float64        `json:"total_net_price_unit,omitempty" binding:"omitempty,min=0"`
	PriceWeight             *float64        `json:"price_weight,omitempty" binding:"omitempty,min=0"`
	ExtraPriceWeight        *float64        `json:"extra_price_weight,omitempty" binding:"omitempty,min=0"`
	TermPriceWeight         *float64        `json:"term_price_weight,omitempty" binding:"omitempty,min=0"`
	TotalNetPriceWeight     *float64        `json:"total_net_price_weight,omitempty" binding:"omitempty,min=0"`
	BeforeTotalNetPriceUnit *float64        `json:"before_total_net_price_unit,omitempty" binding:"omitempty,min=0"`
	BeforeTermPriceWeight   *float64        `json:"before_term_price_weight,omitempty" binding:"omitempty,min=0"`
	EffectiveDate           *time.Time      `json:"effective_date,omitempty"`
	Remark                  *string         `json:"remark,omitempty"`
	UdfJson                 json.RawMessage `json:"udf_json,omitempty"`
}

type UpdatePriceListSubGroupRequest struct {
	SiteCode string                        `json:"site_code"`
	Changes  []UpdatePriceListSubGroupItem `json:"changes" binding:"required,dive"`
}

type UpdateLatestPriceListSubGroupRequest struct {
	// UpdateType determines how the latest price list subgroup update will be performed.
	// Allowed values:
	// - "subgroup": update by explicit subgroup_ids (default when empty for backward compatibility)
	// - "group"   : update all subgroups under the given group_codes
	UpdateType  string   `json:"update_type" binding:"oneof=subgroup group"`
	GroupCodes  []string `json:"group_codes" binding:"omitempty,dive"`
	SubGroupIDs []string `json:"subgroup_ids" binding:"omitempty,dive,uuid4"`
}

type UpdatePriceListGroupExtraKeyRequest struct {
	ID           *uuid.UUID `json:"id"`
	GroupExtraID *uuid.UUID `json:"group_extra_id"`
	Code         string     `json:"code"`
	Value        string     `json:"value"`
	Seq          int        `json:"seq"`
}

type UpdatePriceListExtraRequest struct {
	ID                      *uuid.UUID                            `json:"id"`
	PriceListGroupID        uuid.UUID                             `json:"price_list_group_id"`
	ExtraKey                string                                `json:"extra_key"`
	ConditionCode           string                                `json:"condition_code"`
	ValueInt                int                                   `json:"value_int"`
	LengthExtraKey          int                                   `json:"length_extra_key"`
	Operator                string                                `json:"operator"`
	CondRangeMin            float64                               `json:"cond_range_min"`
	CondRangeMax            float64                               `json:"cond_range_max"`
	CreateBy                string                                `json:"create_by"`
	CreateDtm               time.Time                             `json:"create_dtm"`
	PriceListGroupExtraKeys []UpdatePriceListGroupExtraKeyRequest `json:"price_list_group_extra_keys"`
}

type PriceListFormulas struct {
	ID          uuid.UUID       `json:"id" gorm:"primary_key;not null"`
	FormulaCode string          `json:"formula_code" gorm:"not null"`
	Name        string          `json:"name" gorm:"not null"`
	Uom         string          `json:"uom" gorm:"not null"`
	FormulaType string          `json:"formula_type" gorm:"not null"`
	Expression  string          `json:"expression" gorm:"not null"`
	Params      json.RawMessage `json:"params" gorm:"not null"`
	Rounding    int             `json:"rounding" gorm:"not null"`
	CreateDtm   time.Time       `json:"create_dtm"`
}

func (PriceListFormulas) TableName() string { return "price_list_formulas" }

type PriceListSubGroupFormulasMap struct {
	ID                    uuid.UUID         `json:"id" gorm:"primary_key;not null"`
	PriceListSubGroupCode string            `json:"price_list_subgroup_code" gorm:"not null"`
	PriceListFormulasCode string            `json:"price_list_formulas_code" gorm:"not null"`
	IsDefault             bool              `json:"is_default" gorm:"default:false"`
	CreateDtm             time.Time         `json:"create_dtm"`
	PriceListFormulas     PriceListFormulas `gorm:"foreignKey:PriceListFormulasCode;references:FormulaCode" json:"price_list_formulas"`
	PriceListSubGroup     PriceListSubGroup `gorm:"foreignKey:PriceListSubGroupCode;references:SubgroupKey" json:"price_list_sub_group"`
}

func (PriceListSubGroupFormulasMap) TableName() string {
	return "price_list_subgroup_formulas_map"
}
