package priceService

import (
	"encoding/json"
	"errors"
	"fmt"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"sort"
	"time"

	externalService "prime-erp-core/external/warehouse-service"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// ExportColumn defines a single column for export-ready table data.
// field: stable key (group_code), headerName: human-readable label (group_name).
type ExportColumn struct {
	Field      string `json:"field"`
	HeaderName string `json:"headerName"`
}

// ExportTabHeaders contains report metadata headers for each tab.
type ExportTabHeaders struct {
	Report      string `json:"report"`
	LastUpdated string `json:"last_updated"`
	Download    string `json:"download"`
}

// ExportTab represents a single tab in the export response.
type ExportTab struct {
	Name    string                   `json:"name"`
	Headers ExportTabHeaders         `json:"headers"`
	Columns []ExportColumn           `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
}

// GetPriceExportTableResponse returns export-ready data for CSV generation with multiple tabs.
// Other services can call this endpoint and generate CSV client-side.
type GetPriceExportTableResponse struct {
	Tabs []ExportTab `json:"tabs"`
}

// GetPriceExportTable lists all subgroup rows filtered by GroupCodes and returns export-ready table data.
// Returns multiple tabs: "Detail" (subgroup-based) and "Based price" (group-level with Terms).
func GetPriceExportTable(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req GetPriceListGroupRequest
	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	sqlxDB, err := db.ConnectSqlx(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer sqlxDB.Close()

	// Reuse existing query logic (already supports GroupCodes filtering).
	res, err := getGroupSubGroup(sqlxDB, req)
	if err != nil {
		return nil, fmt.Errorf("GetGroupSubGroup error: %w", err)
	}

	// Get Terms data (required for "Based price" tab).
	res, err = getTerms(sqlxDB, res)
	if err != nil {
		return nil, fmt.Errorf("GetTerms error: %w", err)
	}

	// Enrich subgroup keys with group_name and value_name (required for columns and table values).
	groupMap, groupItemMap, paymentTermMap, err := getGroupAndItemMappings()
	if err != nil {
		return nil, fmt.Errorf("failed to get group mappings: %w", err)
	}

	// Build "Detail" tab (existing functionality).
	detailTab := buildDetailTab(
		res,
		func(code string) string {
			if g, ok := groupMap[code]; ok {
				return g.GroupName
			}
			return ""
		},
		func(code string) string {
			if it, ok := groupItemMap[code]; ok {
				return it.ItemName
			}
			return ""
		},
	)

	// Collect all unique company codes and site codes from the response
	companyCodeSet := make(map[string]bool)
	siteCodeSet := make(map[string]bool)

	for _, resp := range res {
		if resp.CompanyCode != "" {
			companyCodeSet[resp.CompanyCode] = true
		}
		if resp.SiteCode != "" {
			siteCodeSet[resp.SiteCode] = true
		}
	}

	// Collect all key values for inventory service request
	keyValues := []externalService.InventoryByProductCodeKeyValue{}
	for _, resp := range res {
		for _, sg := range resp.SubGroups {
			for _, sgk := range sg.GroupKeys {
				keyValues = append(keyValues, externalService.InventoryByProductCodeKeyValue{
					ID:         sg.ID.String(),
					GroupCode:  sgk.Code,
					GroupValue: sgk.Value,
					Seq:        sgk.Seq,
				})
			}
		}
	}

	// Call inventory service if we have key values
	if len(keyValues) > 0 {
		// Convert sets to slices
		companyCodes := []string{}
		for code := range companyCodeSet {
			companyCodes = append(companyCodes, code)
		}
		siteCodes := []string{}
		for code := range siteCodeSet {
			siteCodes = append(siteCodes, code)
		}

		// Use first company code for the request
		companyCode := ""
		if len(companyCodes) > 0 {
			companyCode = companyCodes[0]
		}

		// Call inventory service
		inventoryResponse, err := externalService.GetInventoryWeightByKey(companyCode, siteCodes, keyValues)
		if err != nil {
			// Log error but continue without inventory data
			fmt.Printf("Warning: failed to get inventory data: %v\n", err)
		} else {
			// Create a map of inventory data by ID for quick lookup
			inventoryMap := make(map[string][]models.InventoryWeightResponse)
			for _, invItem := range inventoryResponse {
				inventoryMap[invItem.ID] = invItem.InventoryWeight
			}

			// Enrich subgroups with inventory data
			for i := range res {
				for j := range res[i].SubGroups {
					sg := &res[i].SubGroups[j]
					if inventoryWeights, ok := inventoryMap[sg.ID.String()]; ok && len(inventoryWeights) > 0 {
						// For export, use first inventory record per subgroup
						inv := inventoryWeights[0]
						sg.InventoryWeight = []models.InventoryWeightResponse{inv}
						sg.ProductCode = inv.ProductCode
						sg.SupplierCode = inv.SupplierCode
						sg.SupplierName = inv.SupplierName
						sg.BatchNo = inv.BatchNo
					}
				}
			}
		}
	}

	// Build "Based price" tab (new functionality).
	basedPriceTab := buildBasedPriceTab(res, paymentTermMap)

	response := GetPriceExportTableResponse{
		Tabs: []ExportTab{detailTab, basedPriceTab},
	}
	return response, nil
}

// detailTabData is the internal structure for Detail tab data (before wrapping in ExportTab).
type detailTabData struct {
	Columns []ExportColumn
	Rows    []map[string]interface{}
}

// buildExportTableTyped is the concrete implementation used by the handler.
// Separated to keep unit tests simple and avoid DB/service dependencies.
func buildExportTableTyped(
	groups []GetPriceListGroupResponse,
	groupNameByCode func(code string) string,
	itemNameByCode func(code string) string,
) detailTabData {
	// Collect columns: group_code -> group_name, track min seq for stable ordering.
	type colMeta struct {
		code   string
		name   string
		minSeq int
	}

	colMap := map[string]*colMeta{}

	for _, g := range groups {
		for _, sg := range g.SubGroups {
			for _, k := range sg.GroupKeys {
				code := k.Code
				if code == "" {
					continue
				}
				name := groupNameByCode(code)
				seq := k.Seq
				if existing, ok := colMap[code]; ok {
					if existing.name == "" && name != "" {
						existing.name = name
					}
					if seq > 0 && (existing.minSeq == 0 || seq < existing.minSeq) {
						existing.minSeq = seq
					}
					continue
				}

				colMap[code] = &colMeta{code: code, name: name, minSeq: seq}
			}
		}
	}

	cols := make([]colMeta, 0, len(colMap))
	for _, m := range colMap {
		cols = append(cols, *m)
	}

	// Sort by seq (if present) then by code to be deterministic.
	sort.Slice(cols, func(i, j int) bool {
		ai, aj := cols[i], cols[j]
		if ai.minSeq != 0 && aj.minSeq != 0 && ai.minSeq != aj.minSeq {
			return ai.minSeq < aj.minSeq
		}
		if ai.minSeq != 0 && aj.minSeq == 0 {
			return true
		}
		if ai.minSeq == 0 && aj.minSeq != 0 {
			return false
		}
		if ai.name != "" && aj.name != "" && ai.name != aj.name {
			return ai.name < aj.name
		}
		return ai.code < aj.code
	})

	// Start with fixed columns first (id is internal, not exported)
	columns := []ExportColumn{
		{Field: "PG01", HeaderName: "หมวดหลัก"},
		{Field: "PG02", HeaderName: "หมวดย่อย"},
		{Field: "PG03", HeaderName: "เกรด/รูปแบบ"},
		{Field: "PG05", HeaderName: "ขนาด"},
		{Field: "PG06", HeaderName: "ขนาดหน้ากว้าง"},
		{Field: "PG07", HeaderName: "ความหนา"},
		{Field: "PG08", HeaderName: "ความยาว"},
		{Field: "PG09", HeaderName: "Other_1"},
		{Field: "PG10", HeaderName: "Other_2"},
		{Field: "total_weight", HeaderName: "Weight-spec"},
		{Field: "avg_weight", HeaderName: "Avg.kg stock"},
		{Field: "market_weight", HeaderName: "น.น. ตลาด"},
		{Field: "total_net_price_weight", HeaderName: "ราคาขาย กก"},
		{Field: "total_net_price_unit", HeaderName: "ราคาขาย เส้น"},
		{Field: "remark", HeaderName: "Remark"},
		{Field: "line_bundle", HeaderName: "เส้น/มัด"},
		{Field: "stock", HeaderName: "Stock"},
		{Field: "stock_quantity", HeaderName: "จำนวน"},
		{Field: "quantity", HeaderName: "จำนวน"},
		{Field: "batch_no", HeaderName: "Ship No."},
		{Field: "brand", HeaderName: "ยี่ห้อ"},
		{Field: "code", HeaderName: "Code"},
		{Field: "warehouse", HeaderName: "โกดัง"},
		{Field: "face", HeaderName: "หน้า"},
		{Field: "z_value", HeaderName: "Z"},
		{Field: "bkk", HeaderName: "BKK"},
		{Field: "country", HeaderName: "ปท."},
		{Field: "tsm", HeaderName: "สมอ."},
		{Field: "institute", HeaderName: "สถาบัน"},
		{Field: "length", HeaderName: "ความยาว"},
		{Field: "od", HeaderName: "OD"},
		{Field: "delivery_date", HeaderName: "ส่งมอบ ว/ด/ป"},
		{Field: "ton", HeaderName: "จำนวนตัน"},
		{Field: "next_production", HeaderName: "ผลิตงวดต่อไป"},
		{Field: "import_date", HeaderName: "วัน เข้า"},
		{Field: "producer", HeaderName: "ผู้ผลิต"},
		{Field: "fast", HeaderName: "เร็ว"},
		{Field: "slow", HeaderName: "ช้า"},
		{Field: "inactive", HeaderName: "Inactive"},
		{Field: "is_highlight", HeaderName: "Highlight สีฟ้า"},
		{Field: "coil_id", HeaderName: "Coil ID"},
		{Field: "supplier_name", HeaderName: "โรงงาน"},
		{Field: "size", HeaderName: "ขนาด"},
		{Field: "spec", HeaderName: "spec"},
	}

	// Then add dynamic group key columns (sorted by seq, then code)
	for _, c := range cols {
		header := c.name
		if header == "" {
			header = c.code
		}
		columns = append(columns, ExportColumn{Field: c.code, HeaderName: header})
	}

	// Collect all unique UDF keys from all subgroups' udf_json data dynamically
	udfKeyMap := make(map[string]bool)
	for _, g := range groups {
		for _, sg := range g.SubGroups {
			if len(sg.UdfJson) > 0 {
				udfData := make(map[string]interface{})
				if err := json.Unmarshal(sg.UdfJson, &udfData); err == nil {
					for key := range udfData {
						udfKeyMap[key] = true
					}
				}
			}
		}
	}

	// Collect and sort UDF keys for deterministic column order
	udfKeys := make([]string, 0, len(udfKeyMap))
	for key := range udfKeyMap {
		udfKeys = append(udfKeys, key)
	}
	sort.Strings(udfKeys)

	// Generate dynamic UDF columns with headers
	for _, key := range udfKeys {
		// Use key as header (can be enhanced with mapping later if needed)
		columns = append(columns, ExportColumn{Field: key, HeaderName: key})
	}

	// Build rows: 1 row per subgroup.
	rows := make([]map[string]interface{}, 0)
	for _, g := range groups {
		for _, sg := range g.SubGroups {
			// Check inactive field from UDF to filter out inactive rows
			inactiveValue := false
			if len(sg.UdfJson) > 0 {
				udfData := make(map[string]interface{})
				if err := json.Unmarshal(sg.UdfJson, &udfData); err == nil {
					if val, ok := udfData["inactive"].(bool); ok {
						inactiveValue = val
					}
				}
			}

			// Skip row if inactive is true
			if inactiveValue {
				continue
			}

			row := map[string]interface{}{
				"id":                           sg.ID.String(),
				"total_net_price_unit":         sg.TotalNetPriceUnit,
				"total_net_price_weight":       sg.TotalNetPriceWeight,
				"before_total_net_price_unit":   sg.BeforeTotalNetPriceUnit,
				"before_total_net_price_weight": sg.BeforeTotalNetPriceWeight,
				"remark":                       sg.Remark,
			}
			// Map UDF values dynamically from udf_json to their corresponding columns
			if len(sg.UdfJson) > 0 {
				udfData := make(map[string]interface{})
				if err := json.Unmarshal(sg.UdfJson, &udfData); err == nil {
					for key, val := range udfData {
						// Only add UDF fields that we have columns for
						if udfKeyMap[key] {
							row[key] = val
						}
					}
				}
			}

			// Add inventory weight fields
			if len(sg.InventoryWeight) > 0 {
				inv := sg.InventoryWeight[0]
				row["total_weight"] = inv.TotalWeight
				row["avg_weight"] = inv.AvgWeight
				row["market_weight"] = inv.WeightSpec
				row["stock"] = inv.SumQty
				row["stock_quantity"] = inv.TotalQty
				row["quantity"] = inv.SumQty
				row["batch_no"] = inv.BatchNo
				row["brand"] = inv.SupplierName
				row["code"] = inv.ProductCode
				row["warehouse"] = inv.SiteCode
				row["supplier_name"] = inv.SupplierName
			}

			// Fill dynamic group_code fields with value_name.
			for _, k := range sg.GroupKeys {
				if k.Code == "" {
					continue
				}
				row[k.Code] = itemNameByCode(k.Value)
			}

			rows = append(rows, row)
		}
	}

	return detailTabData{Columns: columns, Rows: rows}
}

// buildDetailTab wraps the existing buildExportTableTyped logic and adds headers.
func buildDetailTab(
	groups []GetPriceListGroupResponse,
	groupNameByCode func(code string) string,
	itemNameByCode func(code string) string,
) ExportTab {
	// Reuse existing logic but get the old response structure.
	oldResponse := buildExportTableTyped(groups, groupNameByCode, itemNameByCode)

	now := time.Now()
	return ExportTab{
		Name: "Detail",
		Headers: ExportTabHeaders{
			Report:      "Pricelist",
			LastUpdated: formatTimestamp(now),
			Download:    formatTimestamp(now),
		},
		Columns: oldResponse.Columns,
		Rows:    oldResponse.Rows,
	}
}

// buildBasedPriceTab creates the "Based price" tab with group-level data and Terms.
func buildBasedPriceTab(groups []GetPriceListGroupResponse, paymentTermMap map[string]GetPaymentTermResponse) ExportTab {
	now := time.Now()

	// Build columns.
	columns := []ExportColumn{
		{Field: "product", HeaderName: "สินค้า"},
		{Field: "price_pr", HeaderName: "ราคา PR (Last update)"},
		{Field: "cash_pr", HeaderName: "PR"},
	}

	// Discover all unique TermCodes and build term columns.
	termCodeSet := make(map[string]bool)
	for _, g := range groups {
		for _, term := range g.Terms {
			if term.TermCode != "" {
				termCodeSet[term.TermCode] = true
			}
		}
	}

	// Sort term codes for deterministic column order.
	termCodes := make([]string, 0, len(termCodeSet))
	for code := range termCodeSet {
		termCodes = append(termCodes, code)
	}
	sort.Strings(termCodes)

	// Build term columns: PDC (บาท, %) and DUE จ่าย (บาท, %).
	for _, termCode := range termCodes {
		termName := termCode
		if term, ok := paymentTermMap[termCode]; ok && term.TermName != "" {
			termName = term.TermName
		}

		columns = append(columns, ExportColumn{
			Field:      fmt.Sprintf("term_%s_pdc_baht", termCode),
			HeaderName: fmt.Sprintf("%s - PDC - บาท", termName),
		})
		columns = append(columns, ExportColumn{
			Field:      fmt.Sprintf("term_%s_pdc_percent", termCode),
			HeaderName: fmt.Sprintf("%s - PDC - %%", termName),
		})
		columns = append(columns, ExportColumn{
			Field:      fmt.Sprintf("term_%s_due_baht", termCode),
			HeaderName: fmt.Sprintf("%s - DUE จ่าย - บาท", termName),
		})
		columns = append(columns, ExportColumn{
			Field:      fmt.Sprintf("term_%s_due_percent", termCode),
			HeaderName: fmt.Sprintf("%s - DUE จ่าย - %%", termName),
		})
	}

	// Build rows: one row per PriceListGroup.
	rows := make([]map[string]interface{}, 0)
	for _, g := range groups {
		row := map[string]interface{}{
			"product":  g.GroupName,
			"price_pr": g.PriceWeight,
			"cash_pr":  g.PriceWeight, // Same as price_pr per requirements
		}

		// Initialize all term fields to nil.
		for _, termCode := range termCodes {
			row[fmt.Sprintf("term_%s_pdc_baht", termCode)] = nil
			row[fmt.Sprintf("term_%s_pdc_percent", termCode)] = nil
			row[fmt.Sprintf("term_%s_due_baht", termCode)] = nil
			row[fmt.Sprintf("term_%s_due_percent", termCode)] = nil
		}

		// Populate term values from group's Terms.
		for _, term := range g.Terms {
			if term.TermCode == "" {
				continue
			}
			row[fmt.Sprintf("term_%s_pdc_baht", term.TermCode)] = term.Pdc
			row[fmt.Sprintf("term_%s_pdc_percent", term.TermCode)] = term.PdcPercent
			row[fmt.Sprintf("term_%s_due_baht", term.TermCode)] = term.Due
			row[fmt.Sprintf("term_%s_due_percent", term.TermCode)] = term.DuePercent
		}

		rows = append(rows, row)
	}

	return ExportTab{
		Name: "Based price",
		Headers: ExportTabHeaders{
			Report:      "Pricelist- Based price",
			LastUpdated: formatTimestamp(now),
			Download:    formatTimestamp(now),
		},
		Columns: columns,
		Rows:    rows,
	}
}

// bangkok คือ timezone ที่ใช้แสดงผลทุกเวลาในรายงาน
// container ของ service นี้เป็น alpine ที่ไม่ได้ติดตั้ง tzdata และไม่ได้ตั้ง ENV TZ
// LoadLocation จึงล้มเหลวได้ ต้องมี FixedZone สำรองไว้เสมอ
var bangkok = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		return time.FixedZone("ICT", 7*60*60)
	}
	return loc
}()

// formatTimestamp formats time as "DD/MM/YYYY HH:MM" (Thai date format) in Asia/Bangkok.
func formatTimestamp(t time.Time) string {
	return t.In(bangkok).Format("2/1/2006 15:04")
}

// getGroupNameByCode looks up group name from group code.
func getGroupNameByCode(groupCode string, groupMap map[string]models.GetGroupResponse) string {
	if g, ok := groupMap[groupCode]; ok {
		return g.GroupName
	}
	return groupCode
}

// Compile-time guard: ensure we actually depend on models package (imported for the types below).
var _ = models.GetGroupResponse{}
var _ *sqlx.DB
