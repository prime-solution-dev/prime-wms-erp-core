package xService

import (
	"encoding/json"
	"fmt"
	"math"
	externalProductService "prime-erp-core/external/product-service"
	"prime-erp-core/internal/db"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ValidateAPOverPurchaseRequest struct {
	InvoiceCode string                              `json:"invoice_code"` // Optional current invoice code to exclude when saving an existing AP.
	Datas       []ValidateAPOverPurchaseRequestData `json:"datas"`
}

type ValidateAPOverPurchaseRequestData struct {
	PurchaseCode string  `json:"purchase_code"`
	PurchaseItem string  `json:"purchase_item"`
	Qty          float64 `json:"qty"`
	TotalWeight  float64 `json:"total_weight"`
	ValidateUnit string  `json:"validate_unit"` // Deprecated: validation uses purchase_item.purchase_unit.
}

type ValidateAPOverPurchaseResponse struct {
	ResponseCode string                               `json:"response_code"`
	Message      string                               `json:"message"`
	Datas        []ValidateAPOverPurchaseResponseData `json:"datas"`
}

type ValidateAPOverPurchaseResponseData struct {
	PurchaseCode string  `json:"purchase_code"`
	PurchaseItem string  `json:"purchase_item"`
	Index        int     `json:"index"`
	Message      string  `json:"message"`
	Status       string  `json:"status"`        //ERROR, SUCCESS
	ValidateUnit string  `json:"validate_unit"` //WEIGHT, UNIT
	RemainByUnit float64 `json:"remain_by_unit"`
}

type apOverPurchaseAmount struct {
	PurchaseUnit string
	ProductCode  string
	CompanyCode  string
	SiteCode     string
	Qty          float64
	TotalWeight  float64
}

type apOverPurchaseTolerance struct {
	Qty    float64
	Weight float64
}

const apOverPurchaseEpsilon = 0.0000001

func ValidateAPOverPurchaseRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := ValidateAPOverPurchaseRequest{}
	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, fmt.Errorf("invalid JSON payload: %w", err)
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	return ValidateAPOverPurchase(ctx, gormx, req)
}

func ValidateAPOverPurchase(ctx *gin.Context, gormx *gorm.DB, req ValidateAPOverPurchaseRequest) (*ValidateAPOverPurchaseResponse, error) {
	res := &ValidateAPOverPurchaseResponse{
		ResponseCode: "200",
		Message:      "success",
		Datas:        []ValidateAPOverPurchaseResponseData{},
	}
	if len(req.Datas) == 0 {
		return res, nil
	}

	purchaseCodes := uniquePurchaseCodes(req.Datas)
	purchaseItems := uniquePurchaseItems(req.Datas)
	poMap, err := loadPurchaseAmounts(gormx, purchaseCodes, purchaseItems)
	if err != nil {
		return nil, err
	}
	usedMap, err := loadUsedAPAmounts(gormx, purchaseCodes, purchaseItems, req.InvoiceCode)
	if err != nil {
		return nil, err
	}

	tolerances, err := loadAPOverPurchaseTolerances(poMap)
	if err != nil {
		return nil, err
	}

	res.Datas = validateAPOverPurchaseLines(req.Datas, poMap, usedMap, tolerances)
	for _, data := range res.Datas {
		if data.Status == "ERROR" {
			res.Message = "validation failed"
			break
		}
	}

	return res, nil
}

// validateAPOverPurchaseLines checks all request lines after grouping duplicate PO items.
// The PO purchase unit selects quantity (PC) or total weight (KG).
func validateAPOverPurchaseLines(lines []ValidateAPOverPurchaseRequestData, poMap map[string]apOverPurchaseAmount, usedMap map[string]apOverPurchaseAmount, tolerances map[string]apOverPurchaseTolerance) []ValidateAPOverPurchaseResponseData {
	requested := map[string]float64{}
	for _, line := range lines {
		key := purchaseItemKey(line.PurchaseCode, line.PurchaseItem)
		unit := apValidationUnit(poMap[key].PurchaseUnit)
		amount := requestedAmount(line, unit)
		if key == "|" || (unit != "UNIT" && unit != "WEIGHT") || amount < 0 {
			continue
		}
		requested[key+"|"+unit] += amount
	}

	responses := make([]ValidateAPOverPurchaseResponseData, 0, len(lines))
	for index, line := range lines {
		purchaseCode := strings.TrimSpace(line.PurchaseCode)
		purchaseItem := strings.TrimSpace(line.PurchaseItem)
		key := purchaseItemKey(purchaseCode, purchaseItem)
		poAmount, exists := poMap[key]
		unit := apValidationUnit(poAmount.PurchaseUnit)
		response := ValidateAPOverPurchaseResponseData{
			PurchaseCode: purchaseCode,
			PurchaseItem: purchaseItem,
			Index:        index,
			ValidateUnit: unit,
			Status:       "ERROR",
		}

		if purchaseCode == "" || purchaseItem == "" {
			response.Message = "purchase_code and purchase_item are required"
			responses = append(responses, response)
			continue
		}
		if !exists {
			response.Message = "purchase item was not found"
			responses = append(responses, response)
			continue
		}
		if unit != "UNIT" && unit != "WEIGHT" {
			response.Message = "purchase_unit must be PC or KG"
			responses = append(responses, response)
			continue
		}

		amount := requestedAmount(line, unit)
		if amount < 0 {
			response.Message = "requested amount must not be negative"
			responses = append(responses, response)
			continue
		}

		base, used := validationAmounts(unit, poAmount, usedMap[key])
		tolerance := tolerances[key].Qty
		if unit == "WEIGHT" {
			tolerance = tolerances[key].Weight
		}
		allowed := base * (1 + tolerance/100)
		requestedTotal := requested[key+"|"+unit]
		remain := allowed - used - requestedTotal
		if math.Abs(remain) < apOverPurchaseEpsilon {
			remain = 0
		}
		response.RemainByUnit = remain

		if remain < -apOverPurchaseEpsilon {
			response.Message = fmt.Sprintf(
				"exceeds PO: allowed %s, used %s, requested %s, over %s",
				formatAPAmount(allowed), formatAPAmount(used), formatAPAmount(requestedTotal), formatAPAmount(-remain),
			)
		} else {
			response.Status = "SUCCESS"
			response.Message = fmt.Sprintf(
				"within PO limit: allowed %s, used %s, requested %s, remaining %s",
				formatAPAmount(allowed), formatAPAmount(used), formatAPAmount(requestedTotal), formatAPAmount(remain),
			)
		}
		responses = append(responses, response)
	}

	return responses
}

// Missing products or unset numeric fields keep the zero tolerance default.
func loadAPOverPurchaseTolerances(poMap map[string]apOverPurchaseAmount) (map[string]apOverPurchaseTolerance, error) {
	type scope struct{ company, site string }
	groups := map[scope]map[string]bool{}
	for _, po := range poMap {
		if strings.TrimSpace(po.ProductCode) == "" {
			continue
		}
		key := scope{po.CompanyCode, po.SiteCode}
		if groups[key] == nil {
			groups[key] = map[string]bool{}
		}
		groups[key][po.ProductCode] = true
	}
	tolerances := map[string]apOverPurchaseTolerance{}
	for group, codes := range groups {
		productCodes := make([]string, 0, len(codes))
		for code := range codes {
			productCodes = append(productCodes, code)
		}
		products := map[string]apOverPurchaseTolerance{}
		totalPages := 1
		for page := 1; page <= totalPages; page++ {
			res, err := externalProductService.GetProduct(externalProductService.GetProductRequest{
				CompanyCode: []string{group.company}, SiteCode: []string{group.site},
				ProductCode: productCodes, Page: page, PageSize: 1000,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to load product tolerance: %w", err)
			}
			if res.TotalPages > totalPages {
				totalPages = res.TotalPages
			}
			for _, product := range res.Products {
				if product.GRTolerance < 0 || product.GRWeightTolerance < 0 {
					return nil, fmt.Errorf("product %s tolerance must not be negative", product.ProductCode)
				}
				products[strings.ToUpper(strings.TrimSpace(product.ProductCode))] = apOverPurchaseTolerance{
					Qty: product.GRTolerance, Weight: product.GRWeightTolerance,
				}
			}
		}
		for key, po := range poMap {
			if po.CompanyCode == group.company && po.SiteCode == group.site {
				tolerances[key] = products[strings.ToUpper(strings.TrimSpace(po.ProductCode))]
			}
		}
	}
	return tolerances, nil
}

func loadPurchaseAmounts(gormx *gorm.DB, purchaseCodes, purchaseItems []string) (map[string]apOverPurchaseAmount, error) {
	amounts := map[string]apOverPurchaseAmount{}
	if len(purchaseCodes) == 0 || len(purchaseItems) == 0 {
		return amounts, nil
	}

	type purchaseRow struct {
		PurchaseUnit string  `gorm:"column:purchase_unit"`
		ProductCode  string  `gorm:"column:product_code"`
		CompanyCode  string  `gorm:"column:company_code"`
		SiteCode     string  `gorm:"column:site_code"`
		PurchaseCode string  `gorm:"column:purchase_code"`
		PurchaseItem string  `gorm:"column:purchase_item"`
		Qty          float64 `gorm:"column:qty"`
		TotalWeight  float64 `gorm:"column:total_weight"`
	}
	rows := []purchaseRow{}
	if err := gormx.Table("purchase p").
		Select("p.purchase_code, pi.purchase_item, pi.qty, pi.total_weight, pi.purchase_unit, pi.product_code, p.company_code, p.site_code").
		Joins("JOIN purchase_item pi ON pi.purchase_id = p.id").
		Where("p.purchase_code IN ?", purchaseCodes).
		Where("pi.purchase_item IN ?", purchaseItems).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load purchase items: %w", err)
	}

	for _, row := range rows {
		amounts[purchaseItemKey(row.PurchaseCode, row.PurchaseItem)] = apOverPurchaseAmount{
			Qty:          row.Qty,
			TotalWeight:  row.TotalWeight,
			ProductCode:  row.ProductCode,
			PurchaseUnit: row.PurchaseUnit,
			CompanyCode:  row.CompanyCode,
			SiteCode:     row.SiteCode,
		}
	}
	return amounts, nil
}

func loadUsedAPAmounts(gormx *gorm.DB, purchaseCodes, purchaseItems []string, invoiceCode string) (map[string]apOverPurchaseAmount, error) {
	amounts := map[string]apOverPurchaseAmount{}
	if len(purchaseCodes) == 0 || len(purchaseItems) == 0 {
		return amounts, nil
	}

	type usedRow struct {
		PurchaseCode string  `gorm:"column:purchase_code"`
		PurchaseItem string  `gorm:"column:purchase_item"`
		Qty          float64 `gorm:"column:qty"`
		Weight       float64 `gorm:"column:weight"`
	}
	rows := []usedRow{}
	query := gormx.Table("invoice_item ii").
		Select(`ii.document_ref AS purchase_code, ii.document_ref_item AS purchase_item,
			COALESCE(SUM(ii.qty), 0) AS qty, COALESCE(SUM(ii.weight), 0) AS weight`).
		Joins("JOIN invoice i ON i.id = ii.invoice_id").
		Joins("JOIN purchase p ON p.purchase_code = ii.document_ref AND p.company_code = i.company_code AND p.site_code = i.site_code").
		Where("i.invoice_type IN ?", []string{"AP", "AP-FAB"}).
		Where("i.status IN ?", []string{"PENDING", "COMPLETED"}).
		Where("ii.document_ref IN ?", purchaseCodes).
		Where("ii.document_ref_item IN ?", purchaseItems)
	if invoiceCode = strings.TrimSpace(invoiceCode); invoiceCode != "" {
		// Exclude the entire saved invoice; its replacement amounts are in req.Datas.
		query = query.Where("i.invoice_code <> ?", invoiceCode)
	}
	if err := query.Group("ii.document_ref, ii.document_ref_item").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load used AP amounts: %w", err)
	}

	for _, row := range rows {
		amounts[purchaseItemKey(row.PurchaseCode, row.PurchaseItem)] = apOverPurchaseAmount{
			Qty:         row.Qty,
			TotalWeight: row.Weight,
		}
	}
	return amounts, nil
}

func uniquePurchaseCodes(lines []ValidateAPOverPurchaseRequestData) []string {
	seen := map[string]bool{}
	codes := []string{}
	for _, line := range lines {
		code := strings.TrimSpace(line.PurchaseCode)
		key := strings.ToUpper(code)
		if code == "" || seen[key] {
			continue
		}
		seen[key] = true
		codes = append(codes, code)
	}
	return codes
}

func uniquePurchaseItems(lines []ValidateAPOverPurchaseRequestData) []string {
	seen := map[string]bool{}
	items := []string{}
	for _, line := range lines {
		item := strings.TrimSpace(line.PurchaseItem)
		key := strings.ToUpper(item)
		if item == "" || seen[key] {
			continue
		}
		seen[key] = true
		items = append(items, item)
	}
	return items
}

func purchaseItemKey(purchaseCode, purchaseItem string) string {
	return strings.ToUpper(strings.TrimSpace(purchaseCode)) + "|" + strings.ToUpper(strings.TrimSpace(purchaseItem))
}

func apValidationUnit(purchaseUnit string) string {
	switch strings.ToUpper(strings.TrimSpace(purchaseUnit)) {
	case "PC":
		return "UNIT"
	case "KG":
		return "WEIGHT"
	default:
		return ""
	}
}

func requestedAmount(line ValidateAPOverPurchaseRequestData, validateUnit string) float64 {
	if validateUnit == "UNIT" {
		return line.Qty
	}
	return line.TotalWeight
}

func validationAmounts(validateUnit string, po, used apOverPurchaseAmount) (float64, float64) {
	if validateUnit == "UNIT" {
		return po.Qty, used.Qty
	}
	return po.TotalWeight, used.TotalWeight
}

func formatAPAmount(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
