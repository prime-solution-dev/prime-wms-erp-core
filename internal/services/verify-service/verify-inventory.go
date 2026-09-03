package verifyService

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	externalService "prime-erp-core/external/warehouse-service"
)

type VerifyInventoryRequest struct {
	CompanyCode    string                   `json:"company_code"`
	SiteCode       string                   `json:"site_code"`
	WarehouseCodes []string                 `json:"warehouse_codes"`
	StorageTypes   []string                 `json:"storage_types"`
	Products       []VerifyInventoryProduct `json:"products"`
	ToDate         *time.Time               `json:"to_date"`
}

type VerifyInventoryProduct struct {
	ProductCode string  `json:"product_code"`
	Qty         float64 `json:"qty"`
	TotalWeight float64 `json:"total_weight"`
}

type VerifyInventoryResponse struct {
	IsPassInventory       bool                         `json:"is_pass_inventory"`
	ProductAtps           []VerifyInventoryProductAtp  `json:"product_atps"`
	InventoryCalculations []VerifyInventoryCalculation `json:"inventory_calculations"`
}

type VerifyInventoryProductAtp struct {
	CompanyCode      string  `json:"company_code"`
	SiteCode         string  `json:"site_code"`
	ProductCode      string  `json:"product_code"`
	WarehouseCode    string  `json:"warehouse_code"`
	TodayStockQty    float64 `json:"current_stock_qty"`
	TodayStockWeight float64 `json:"current_stock_weight"`
	TodayAtpQty      float64 `json:"current_atp_qty"`
	TodayAtpWeight   float64 `json:"current_atp_weight"`
	TotalAtpQty      float64 `json:"atp_qty"`
	TotalAtpWeight   float64 `json:"atp_weight"`
}

type VerifyInventoryCalculation struct {
	Subject         string  `json:"subject"`
	ProductCode     string  `json:"product_code"`
	NeedQty         float64 `json:"need_qty"`
	NeedWeight      float64 `json:"need_weight"`
	AvailableQty    float64 `json:"available_qty"`
	AvailableWeight float64 `json:"available_weight"`
	AllocatedQty    float64 `json:"allocated_qty"`
	AllocatedWeight float64 `json:"allocated_weight"`
	RemainQty       float64 `json:"remain_qty"`
	RemainWeight    float64 `json:"remain_weight"`
	ShortageQty     float64 `json:"shortage_qty"`
	ShortageWeight  float64 `json:"shortage_weight"`
	IsPass          bool    `json:"is_pass"`
}

func GetSystemConfigWarehouse() ([]string, error) {
	getSystemConfigWarehouseRequest := externalService.GetSystemConfigRequest{
		TopicCodes:  []string{"ATP"},
		ConfigCodes: []string{"ATP_CONDITION"},
	}
	systemConfigData, err := externalService.GetSystemConfigWarehouse(getSystemConfigWarehouseRequest)
	if err != nil {
		return nil, err
	}

	warehouseCodes := []string{}

	// ประมวลผล response และแยก warehouse codes
	for _, config := range systemConfigData.SystemConfigs {
		if config.TopicCode == "ATP" && config.ConfigCode == "ATP_CONDITION" && config.Value != "" {
			// แยกค่าด้วย comma และลบ whitespace
			codes := strings.Split(config.Value, ",")
			for _, code := range codes {
				code = strings.TrimSpace(code)
				if code != "" {
					warehouseCodes = append(warehouseCodes, code)
				}
			}
		}
	}

	return warehouseCodes, nil
}

func VerifyInventoryLogic(req VerifyInventoryRequest) (*VerifyInventoryResponse, error) {
	res := &VerifyInventoryResponse{}
	res.IsPassInventory = true
	res.InventoryCalculations = []VerifyInventoryCalculation{}

	if len(req.Products) == 0 {
		return res, fmt.Errorf("require at least one product")
	}

	warehouseCodes := req.WarehouseCodes
	if len(warehouseCodes) == 0 {
		// ถ้า request ไม่ระบุ warehouse ให้ใช้ค่า default จาก system config
		var err error
		warehouseCodes, err = GetSystemConfigWarehouse()
		if err != nil {
			return nil, fmt.Errorf("failed to get warehouse config: %v", err)
		}
	}

	productExists := map[string]bool{}
	reqAtp := externalService.GetInventoryAtpRequest{
		CompanyCodes: []string{req.CompanyCode},
		SiteCodes:    []string{req.SiteCode},
		StorageTypes: req.StorageTypes,
		ToDate:       req.ToDate,
	}

	// รวม warehouse codes ถ้ามีข้อมูล
	if len(warehouseCodes) > 0 {
		reqAtp.WarehouseCodes = warehouseCodes
	}

	for _, p := range req.Products {
		if p.Qty <= 0 {
			continue
		}

		if !productExists[p.ProductCode] {
			reqAtp.ProductCodes = append(reqAtp.ProductCodes, p.ProductCode)
			productExists[p.ProductCode] = true
		}
	}

	requestJSON, _ := json.MarshalIndent(reqAtp, "", "  ")
	fmt.Println("CreateGoodsIssueRequest JSON:")
	fmt.Println(string(requestJSON))

	resAtp, err := externalService.GetInventoryATP(reqAtp)
	if err != nil {
		return nil, err
	}

	remainingATP := map[string]float64{}
	remainingATPWeight := map[string]float64{}
	for _, atp := range resAtp.ProductAtps {
		atpKey := fmt.Sprintf(`%s|%s|%s`, atp.CompanyCode, atp.SiteCode, atp.ProductCode)
		remainingATP[atpKey] += atp.TotalAtpQty
		remainingATPWeight[atpKey] += atp.TotalAtpWeight

		res.ProductAtps = append(res.ProductAtps, VerifyInventoryProductAtp{
			CompanyCode:      atp.CompanyCode,
			SiteCode:         atp.SiteCode,
			ProductCode:      atp.ProductCode,
			WarehouseCode:    "",
			TodayStockQty:    atp.TodayStockQty,
			TodayStockWeight: atp.TodayStockWeight,
			TodayAtpQty:      atp.TodayAtpQty,
			TodayAtpWeight:   atp.TodayAtpWeight,
			TotalAtpQty:      atp.TotalAtpQty,
			TotalAtpWeight:   atp.TotalAtpWeight,
		})
	}

	for _, p := range req.Products {
		atpKey := fmt.Sprintf(`%s|%s|%s`, req.CompanyCode, req.SiteCode, p.ProductCode)
		available, existAtp := remainingATP[atpKey]
		if !existAtp {
			available = 0
		}
		availableWeight := remainingATPWeight[atpKey]

		allocated := minFloat(p.Qty, available)
		remain := available - allocated
		shortage := maxFloat(p.Qty-available, 0)
		allocatedWeight := 0.0
		remainWeight := availableWeight
		shortageWeight := 0.0
		if p.TotalWeight > 0 {
			allocatedWeight = minFloat(p.TotalWeight, availableWeight)
			remainWeight = availableWeight - allocatedWeight
			shortageWeight = maxFloat(p.TotalWeight-availableWeight, 0)
		}
		isPass := shortage == 0 && shortageWeight == 0

		res.InventoryCalculations = append(res.InventoryCalculations, VerifyInventoryCalculation{
			Subject:         "inventory",
			ProductCode:     p.ProductCode,
			NeedQty:         p.Qty,
			NeedWeight:      p.TotalWeight,
			AvailableQty:    available,
			AvailableWeight: availableWeight,
			AllocatedQty:    allocated,
			AllocatedWeight: allocatedWeight,
			RemainQty:       remain,
			RemainWeight:    remainWeight,
			ShortageQty:     shortage,
			ShortageWeight:  shortageWeight,
			IsPass:          isPass,
		})

		if !isPass {
			res.IsPassInventory = false
		}

		remainingATP[atpKey] = remain
		remainingATPWeight[atpKey] = remainWeight
	}

	return res, nil
}

func minFloat(a float64, b float64) float64 {
	if a < b {
		return a
	}

	return b
}

func maxFloat(a float64, b float64) float64 {
	if a > b {
		return a
	}

	return b
}
