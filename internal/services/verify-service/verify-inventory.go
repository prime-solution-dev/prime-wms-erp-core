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
}

type VerifyInventoryResponse struct {
	IsPassInventory bool                        `json:"is_pass_inventory"`
	ProductAtps     []VerifyInventoryProductAtp `json:"product_atps"`
}

type VerifyInventoryProductAtp struct {
	CompanyCode   string `json:"company_code"`
	SiteCode      string `json:"site_code"`
	ProductCode   string `json:"product_code"`
	WarehouseCode string `json:"warehouse_code"`
	TodayStockQty int    `json:"current_stock_qty"`
	TodayAtpQty   int    `json:"current_atp_qty"`
	TotalAtpQty   int    `json:"atp_qty"`
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

	if len(req.Products) == 0 {
		return res, fmt.Errorf("require at least one product")
	}

	// ดึง warehouse codes จาก system config
	warehouseCodes, err := GetSystemConfigWarehouse()
	if err != nil {
		return nil, fmt.Errorf("failed to get warehouse config: %v", err)
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
	for _, atp := range resAtp.ProductAtps {
		atpKey := fmt.Sprintf(`%s|%s|%s`, atp.CompanyCode, atp.SiteCode, atp.ProductCode)
		remainingATP[atpKey] += float64(atp.TotalAtpQty)

		res.ProductAtps = append(res.ProductAtps, VerifyInventoryProductAtp{
			CompanyCode:   atp.CompanyCode,
			SiteCode:      atp.SiteCode,
			ProductCode:   atp.ProductCode,
			WarehouseCode: "",
			TodayStockQty: int(atp.TodayStockQty),
			TodayAtpQty:   int(atp.TodayAtpQty),
			TotalAtpQty:   int(atp.TotalAtpQty),
		})
	}

	for _, p := range req.Products {
		atpKey := fmt.Sprintf(`%s|%s|%s`, req.CompanyCode, req.SiteCode, p.ProductCode)
		remain, existAtp := remainingATP[atpKey]
		if !existAtp {
			remain = 0
		}

		if remain < p.Qty {
			res.IsPassInventory = false
			break
		}

		use := min(p.Qty, remain)
		remain -= use
		remainingATP[atpKey] = remain
	}

	return res, nil
}

func min(a float64, b float64) float64 {
	if a > b {
		return a
	}

	return b
}
