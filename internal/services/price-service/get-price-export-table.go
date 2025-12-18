package priceService

import (
	"encoding/json"
	"errors"
	"fmt"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"sort"
	"time"

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

	// Build "Based price" tab (new functionality).
	basedPriceTab := buildBasedPriceTab(res, groupMap, paymentTermMap)

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

	columns := make([]ExportColumn, 0, len(cols))
	for _, c := range cols {
		header := c.name
		if header == "" {
			header = c.code
		}
		columns = append(columns, ExportColumn{Field: c.code, HeaderName: header})
	}

	udfColumns := []string{"is_highlight",
		"inactive",
		"line_bundle",
		"market_weight",
		"od",
		"stock",
		"import_date",
		"ton",
		"producer",
		"selling_fast",
		"selling_slow",
		"awaiting_production_import_date",
		"awaiting_production_delivery_date",
		"awaiting_production_ton",
		"awaiting_production_producer",
		"bkk",
		"factory",
		"country",
		"ship_no",
		"tsm",
		"remark",
	}
	udfColumnsHeaders := []string{"Highlight สีฟ้า",
		"Inactive",
		"เส้น/มัด",
		"น้ำหนักตลาด",
		"OD",
		"Stock",
		"วัน เข้า",
		"ตัน",
		"ผู้ผลิต",
		"ขายช้า",
		"ขายเร็ว",
		"รอผลิตวันเข้า",
		"รอผลิตวันจัดส่ง",
		"รอผลิตตัน",
		"รอผลิตผู้ผลิต",
		"BKK",
		"โรงงงาน",
		"ประเทศ",
		"โกดัง",
		"สถาบัน",
		"Remark",
	}
	for i, col := range udfColumns {
		columns = append(columns, ExportColumn{Field: col, HeaderName: udfColumnsHeaders[i]})
	}

	// Build rows: 1 row per subgroup.
	rows := make([]map[string]interface{}, 0)
	for _, g := range groups {
		for _, sg := range g.SubGroups {
			row := map[string]interface{}{
				"id":                     sg.ID.String(),
				"total_net_price_unit":   sg.TotalNetPriceUnit,
				"total_net_price_weight": sg.TotalNetPriceWeight,
				"remark":                 sg.Remark,
			}
			if len(sg.UdfJson) > 0 {
				udfData := make(map[string]interface{})
				if err := json.Unmarshal(sg.UdfJson, &udfData); err == nil {
					for _, col := range udfColumns {
						if val, ok := udfData[col]; ok {
							row[col] = val
						}
					}
				}
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
func buildBasedPriceTab(
	groups []GetPriceListGroupResponse,
	groupMap map[string]models.GetGroupResponse,
	paymentTermMap map[string]GetPaymentTermResponse,
) ExportTab {
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

// formatTimestamp formats time as "DD/MM/YYYY HH:MM" (Thai date format).
func formatTimestamp(t time.Time) string {
	return t.Format("2/1/2006 15:04")
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
