package priceService

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/gin-gonic/gin"
)

type GetComparePriceRequest struct {
	TotalAmount        float64            `json:"total_amount"`
	TotalWeight        float64            `json:"total_weight"`
	TotalTransportCost float64            `json:"transport_cost"`
	TransportType      string             `json:"transport_type"`
	UnitCode           string             `json:"unit_code"`        // PCS
	UnitCodeWeight     string             `json:"unit_code_weight"` //KG
	Items              []ItemComparePrice `json:"items"`
}

type GetComparePriceResponse struct {
	TotalAmount                 float64            `json:"total_amount"`
	TotalWeight                 float64            `json:"total_weight"`
	TotalTransportCost          float64            `json:"total_transport_cost"`
	SubtotalExclTransport       float64            `json:"subtotal_excl_transport"`
	AvgPriceUnitExclTransport   float64            `json:"avg_price_unit_excl_transport"`
	AvgPriceWeightExclTransport float64            `json:"avg_price_weight_excl_transport"`
	IsPassPriceUnitAll          bool               `json:"is_pass_price_unit_all"`
	IsPassPriceWeightAll        bool               `json:"is_pass_price_weight_all"`
	Items                       []ItemComparePrice `json:"items"`
}

type ItemComparePrice struct {
	RefItem                 string   `json:"ref_item"`
	ProductCode             string   `json:"product_code"`
	Qty                     float64  `json:"qty"`
	Unit                    string   `json:"unit_code"`
	SaleUnit                string   `json:"sale_unit"`
	SaleUnitType            string   `json:"sale_unit_type"`
	PriceListUnit           float64  `json:"price_list"`
	PriceUnit               float64  `json:"price"`
	TotalAmount             float64  `json:"total_amount"`               //1
	TotalWeight             float64  `json:"total_weight"`               //7
	TransportCostUnit       *float64 `json:"transport_cost_unit"`        //2
	TransportCostUnitWeight *float64 `json:"transport_cost_unit_weight"` //8

	// Results
	SubtotalExclTransport          float64 `json:"subtotal_excl_transport"`             //3
	NetPriceUnitExclTransport      float64 `json:"net_price_unit_excl_transport"`       //4
	PriceDiffUnit                  float64 `json:"price_diff_unit"`                     //5
	IsPassPriceUnit                bool    `json:"is_pass_price_unit"`                  //6
	SubtotalWeightExclTransport    float64 `json:"subtotal_weight_excl_transport"`      //9
	NetPricePerWeightExclTransport float64 `json:"net_price_per_weight_excl_transport"` //10
	PriceDiffUnitWeight            float64 `json:"price_diff_unit_weight"`              //11
	IsPassPriceWeight              bool    `json:"is_pass_price_weight"`
}

func GetComparePrice(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := GetComparePriceRequest{}
	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}
	return ComparePrice(req)
}

func ComparePrice(req GetComparePriceRequest) (GetComparePriceResponse, error) {
	res := GetComparePriceResponse{}

	if len(req.Items) == 0 {
		return res, fmt.Errorf("items is required")
	}

	totalPriceAll := req.TotalAmount
	totalWeightAll := req.TotalWeight
	totalTransportCostAll := req.TotalTransportCost
	unitCode := req.UnitCode
	unitCodeWeight := req.UnitCodeWeight

	sumTransportUnit := 0.0
	sumTransportUnitWeight := 0.0

	// ----- Initial calculation -----
	for _, item := range req.Items {
		newItem := item

		if req.TransportType == `INCL` {
			newItem.TransportCostUnit = calculateTransportCost(item.TotalAmount, totalPriceAll, totalTransportCostAll, item.TransportCostUnit)
			sumTransportUnit += float64Val(newItem.TransportCostUnit)

			newItem.TransportCostUnitWeight = calculateTransportCost(item.TotalWeight, totalWeightAll, totalTransportCostAll, item.TransportCostUnitWeight)
			sumTransportUnitWeight += float64Val(newItem.TransportCostUnitWeight)
		} else {
			newItem.TransportCostUnit = float64Ptr(0)
			newItem.TransportCostUnitWeight = float64Ptr(0)
		}

		res.Items = append(res.Items, newItem)
	}

	sort.SliceStable(res.Items, func(i, j int) bool {
		return res.Items[i].TotalAmount > res.Items[j].TotalAmount
	})

	//  Adjust Transport Cost Unit to match total
	if req.TransportType == `INCL` {
		adjustTransportCosts(res.Items, totalTransportCostAll, &sumTransportUnit, true)
		adjustTransportCosts(res.Items, totalTransportCostAll, &sumTransportUnitWeight, false)
	}

	//  Calculate net price
	var subtotalExclTransport, subtotalExclTransportWeight float64
	passUnitAll := true
	passWeightAll := true

	for i := range res.Items {
		item := res.Items[i]

		//Unit
		item.SubtotalExclTransport = round2(item.TotalAmount - float64Val(item.TransportCostUnit))
		if item.SaleUnit == unitCode && item.Qty > 0 {
			item.NetPriceUnitExclTransport = round2(item.SubtotalExclTransport / item.Qty)
		} else if item.SaleUnit == unitCodeWeight && item.TotalWeight > 0 {
			item.NetPriceUnitExclTransport = round2(item.SubtotalExclTransport / item.TotalWeight)
		} else {
			item.NetPriceUnitExclTransport = 0
		}

		item.PriceDiffUnit = round2(item.NetPriceUnitExclTransport - item.PriceListUnit)
		item.IsPassPriceUnit = item.PriceDiffUnit >= 0

		if !item.IsPassPriceUnit {
			passUnitAll = false
		}

		//Weight
		item.SubtotalWeightExclTransport = round2(item.TotalAmount - float64Val(item.TransportCostUnitWeight))
		if item.SaleUnit == unitCode && item.Qty > 0 {
			item.NetPricePerWeightExclTransport = round2(item.SubtotalWeightExclTransport / item.Qty)
		} else if item.SaleUnit == unitCodeWeight && item.TotalWeight > 0 {
			item.NetPricePerWeightExclTransport = round2(item.SubtotalWeightExclTransport / item.TotalWeight)
		} else {
			item.NetPricePerWeightExclTransport = 0
		}

		item.PriceDiffUnitWeight = round2(item.NetPricePerWeightExclTransport - item.PriceListUnit)
		item.IsPassPriceWeight = item.PriceDiffUnitWeight >= 0

		if !item.IsPassPriceWeight {
			passWeightAll = false
		}

		// Update slice and totals
		res.Items[i] = item
		subtotalExclTransport += item.SubtotalExclTransport
		subtotalExclTransportWeight += item.SubtotalWeightExclTransport
	}

	//Summary
	res.TotalAmount = round2(totalPriceAll)
	res.TotalWeight = round2(totalWeightAll)
	res.TotalTransportCost = round2(totalTransportCostAll)
	res.SubtotalExclTransport = round2(subtotalExclTransport)

	if totalWeightAll > 0 {
		res.AvgPriceWeightExclTransport = round2(subtotalExclTransportWeight / totalWeightAll)
	}
	if totalPriceAll > 0 {
		res.AvgPriceUnitExclTransport = round2(subtotalExclTransport / totalPriceAll)
	}

	res.IsPassPriceUnitAll = passUnitAll
	res.IsPassPriceWeightAll = passWeightAll

	return res, nil
}

func calculateTransportCost(itemValue, totalValue, totalTransport float64, existing *float64) *float64 {
	if existing != nil && *existing > 0 {
		return existing
	}

	if totalValue > 0 {
		return float64Ptr(round2(itemValue / totalValue * totalTransport))
	}

	return float64Ptr(0)
}

func adjustTransportCosts(items []ItemComparePrice, totalTransport float64, sumTransport *float64, isUnit bool) {
	diff := totalTransport - *sumTransport
	diff = math.Round(diff*100) / 100
	if diff == 0 {
		return
	}

	for diff != 0 {
		for i := range items {
			var val float64
			if isUnit {
				val = float64Val(items[i].TransportCostUnit)
			} else {
				val = float64Val(items[i].TransportCostUnitWeight)
			}

			if diff > 0 {
				val += 0.01
				diff -= 0.01
			} else if diff < 0 {
				val -= 0.01
				diff += 0.01
			}

			if isUnit {
				items[i].TransportCostUnit = float64Ptr(val)
			} else {
				items[i].TransportCostUnitWeight = float64Ptr(val)
			}

			if diff == 0 {
				break
			}
		}
	}
}

func round2(val float64) float64 {
	return math.Round(val*100) / 100
}

func float64Ptr(v float64) *float64 {
	return &v
}

func float64Val(p *float64) float64 {
	if p != nil {
		return *p
	}

	return 0
}
