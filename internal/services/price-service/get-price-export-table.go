package priceService

import (
	"encoding/json"
	"errors"
	"fmt"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"sort"

	"github.com/gin-gonic/gin"
)

// ExportColumn defines a single column for export-ready table data.
// field: stable key (group_code), headerName: human-readable label (group_name).
type ExportColumn struct {
	Field      string `json:"field"`
	HeaderName string `json:"headerName"`
}

// GetPriceExportTableResponse returns export-ready data for CSV generation.
// Other services can call this endpoint and generate CSV client-side.
type GetPriceExportTableResponse struct {
	Columns []ExportColumn           `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
}

// GetPriceExportTable lists all subgroup rows filtered by GroupCodes and returns export-ready table data.
// Columns are derived from subgroup key metadata (group_name) and are keyed by group_code.
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

	// Enrich subgroup keys with group_name and value_name (required for columns and table values).
	groupMap, groupItemMap, _, err := getGroupAndItemMappings()
	if err != nil {
		return nil, fmt.Errorf("failed to get group mappings: %w", err)
	}

	response := buildExportTableTyped(
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
	return response, nil
}

// buildExportTableTyped is the concrete implementation used by the handler.
// Separated to keep unit tests simple and avoid DB/service dependencies.
func buildExportTableTyped(
	groups []GetPriceListGroupResponse,
	groupNameByCode func(code string) string,
	itemNameByCode func(code string) string,
) GetPriceExportTableResponse {
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

	return GetPriceExportTableResponse{Columns: columns, Rows: rows}
}

// Compile-time guard: ensure we actually depend on models package (imported for the types below).
var _ = models.GetGroupResponse{}
