package priceService

import (
	"encoding/json"
	"time"
)

// ============================================================================
// Request Structures
// ============================================================================

// GetPriceDetailRequest represents the request structure
type GetPriceDetailRequest struct {
	CompanyCode       string     `json:"company_code"`
	SiteCodes         []string   `json:"site_codes"`
	GroupCodes        []string   `json:"group_codes"`
	EffectiveDateFrom *time.Time `json:"effective_date_from"`
	EffectiveDateTo   *time.Time `json:"effective_date_to"`
}


type PriceFormula struct {
	Expression string          `json:"expression"`
	Params     json.RawMessage `json:"params"`
	Rounding   int             `json:"rounding"`
}

type PriceData struct {
	BasePrice  float64 `json:"base_price"`
	Extra      float64 `json:"extra"`
	AvgKgStock float64 `json:"avg_kg_stock"`
	WeightSpec float64 `json:"weight_spec"`
	Pcs        float64 `json:"pcs"`
	Kg         float64 `json:"kg"`
}

type Price struct {
	Id                  string  `json:"id"`
	TotalNetPriceUnit   float64 `json:"total_net_price_unit"`
	TotalNetPriceWeight float64 `json:"total_net_price_weight"`
	ExtraPriceUnit      float64 `json:"extra_price_unit"`
	ExtraPriceWeight    float64 `json:"extra_price_weight"`
	DefaultUom          string  `json:"default_uom"`
}

type PriceResult struct {
	Prices []Price `json:"prices"`
}
