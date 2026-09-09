package priceService

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	priceListRepository "prime-erp-core/internal/repositories/priceList"
)

// groupCodeColumnSuffix ต่อท้าย field ของคอลัมน์ที่เก็บ "รหัส" กลุ่ม
// เพื่อไม่ให้ชนกับคอลัมน์ที่เก็บ "ชื่อ" กลุ่มซึ่งใช้ field เป็น group code ตรง ๆ
const groupCodeColumnSuffix = "__code"

// groupColumn คือคอลัมน์กลุ่มสินค้าหนึ่งตัวที่เก็บได้จากข้อมูลจริง
type groupColumn struct {
	code   string
	name   string
	minSeq int
}

// collectGroupColumns เก็บ group code ทุกตัวที่พบใน subgroup พร้อม seq ต่ำสุด
// แล้วเรียงให้คงที่ — seq มาก่อน ถ้าไม่มี seq ให้เรียงตาม code
func collectGroupColumns(
	groups []GetPriceListGroupResponse,
	groupNameByCode func(code string) string,
) []groupColumn {
	colMap := map[string]*groupColumn{}
	for _, g := range groups {
		for _, sg := range g.SubGroups {
			for _, k := range sg.GroupKeys {
				if k.Code == "" {
					continue
				}
				name := strings.TrimSpace(groupNameByCode(k.Code))
				if existing, ok := colMap[k.Code]; ok {
					if existing.name == "" && name != "" {
						existing.name = name
					}
					if k.Seq > 0 && (existing.minSeq == 0 || k.Seq < existing.minSeq) {
						existing.minSeq = k.Seq
					}
					continue
				}
				colMap[k.Code] = &groupColumn{code: k.Code, name: name, minSeq: k.Seq}
			}
		}
	}

	cols := make([]groupColumn, 0, len(colMap))
	for _, m := range colMap {
		cols = append(cols, *m)
	}
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
		return ai.code < aj.code
	})
	return cols
}

// buildPricelistDetailTab ประกอบ tab "Template" ของ Pricelist Detail Report
//
// คอลัมน์กลุ่มสินค้าไม่ได้ hardcode ไว้ — เก็บจาก GroupKey.Code ที่พบจริงในข้อมูล
// หัวคอลัมน์มาจากตาราง group ผ่าน groupNameByCode และค่าในเซลล์มาจากตาราง group_item
// ผ่าน itemNameByCode ทั้งคู่ fallback เป็นรหัสดิบเมื่อ resolve ไม่ได้
//
// formulas เป็น map จาก subgroup_code ไปยังสูตรของ subgroup นั้น ส่งค่า nil ได้
// เมื่อไม่ต้องการเติมคอลัมน์สูตร
func buildPricelistDetailTab(
	groups []GetPriceListGroupResponse,
	groupNameByCode func(code string) string,
	itemNameByCode func(code string) string,
	formulas map[string][]priceListRepository.SubgroupFormula,
	lastUpdated *time.Time,
) ExportTab {
	cols := collectGroupColumns(groups, groupNameByCode)

	// ลำดับคอลัมน์ตามชีท Template ของไฟล์ตัวอย่าง
	columns := []ExportColumn{
		{Field: "pricelist_group_name", HeaderName: "Pricelist group name"},
	}
	for _, c := range cols {
		header := c.name
		if header == "" {
			header = c.code
		}
		columns = append(columns, ExportColumn{Field: c.code, HeaderName: header})
	}
	columns = append(columns,
		ExportColumn{Field: "total_weight", HeaderName: "Weight-spec"},
		ExportColumn{Field: "avg_weight", HeaderName: "Avg. kg stock"},
		ExportColumn{Field: "price_per_kg", HeaderName: "Price per kg"},
		ExportColumn{Field: "price_per_unit", HeaderName: "Price per unit"},
		ExportColumn{Field: "extra_price", HeaderName: "Extra price"},
		ExportColumn{Field: "formula_kg_name", HeaderName: "Price per kg formula"},
		ExportColumn{Field: "formula_unit_name", HeaderName: "Price per unit formula"},
		ExportColumn{Field: "pricelist_group_code", HeaderName: "Pricelist group code"},
	)
	for _, c := range cols {
		columns = append(columns, ExportColumn{
			Field:      c.code + groupCodeColumnSuffix,
			HeaderName: c.code,
		})
	}
	columns = append(columns,
		ExportColumn{Field: "formula_kg_code", HeaderName: "Price per kg formula code"},
		ExportColumn{Field: "formula_unit_code", HeaderName: "Price per unit formula code"},
		ExportColumn{Field: "subgroup_code", HeaderName: "subgroup_code"},
	)

	rows := make([]map[string]interface{}, 0)
	for _, g := range groups {
		groupName := itemNameByCode(g.GroupCode)
		if groupName == "" {
			groupName = g.GroupCode
		}

		for _, sg := range g.SubGroups {
			if isInactiveSubGroup(sg.UdfJson) {
				continue
			}

			row := map[string]interface{}{
				"pricelist_group_name": groupName,
				"pricelist_group_code": g.GroupCode,
				"subgroup_code":        sg.SubgroupCode,
				"price_per_kg":         sg.TotalNetPriceWeight,
				"price_per_unit":       sg.TotalNetPriceUnit,
				"extra_price":          sg.ExtraPriceWeight,
				"total_weight":         "",
				"avg_weight":           "",
				"formula_kg_name":      "",
				"formula_kg_code":      "",
				"formula_unit_name":    "",
				"formula_unit_code":    "",
			}

			// เติมเซลล์ว่างให้ทุกคอลัมน์กลุ่มก่อน เพื่อไม่ให้แถวที่ไม่มีกลุ่มนั้น
			// เหลือ key ขาดหายจนอ่านค่าไม่ได้ตอนเขียนไฟล์
			for _, c := range cols {
				row[c.code] = ""
				row[c.code+groupCodeColumnSuffix] = ""
			}
			for _, k := range sg.GroupKeys {
				if k.Code == "" {
					continue
				}
				name := itemNameByCode(k.Value)
				if name == "" {
					name = k.Value
				}
				row[k.Code] = name
				row[k.Code+groupCodeColumnSuffix] = k.Value
			}

			if len(sg.InventoryWeight) > 0 {
				inv := sg.InventoryWeight[0]
				row["total_weight"] = inv.TotalWeight
				row["avg_weight"] = inv.AvgWeight
			}

			for _, f := range formulas[sg.SubgroupCode] {
				switch f.Uom {
				case "kg":
					row["formula_kg_name"] = f.Name
					row["formula_kg_code"] = f.FormulaCode
				case "pcs":
					row["formula_unit_name"] = f.Name
					row["formula_unit_code"] = f.FormulaCode
				}
			}

			rows = append(rows, row)
		}
	}

	return ExportTab{
		Name: "Template",
		Headers: ExportTabHeaders{
			Report:      "Pricelist Detail",
			LastUpdated: formatOptionalTimestamp(lastUpdated),
			Download:    formatTimestamp(time.Now()),
		},
		Columns: columns,
		Rows:    rows,
	}
}

// isInactiveSubGroup อ่าน flag inactive จาก udf_json — แถวที่ inactive ไม่ถูก export
// เหมือนพฤติกรรมของ buildExportTableTyped
func isInactiveSubGroup(udfJson json.RawMessage) bool {
	if len(udfJson) == 0 {
		return false
	}
	udfData := map[string]interface{}{}
	if err := json.Unmarshal(udfJson, &udfData); err != nil {
		return false
	}
	val, _ := udfData["inactive"].(bool)
	return val
}

// selectExportTabs เลือกชุด tab ตาม report type
// แยกออกมาเป็นฟังก์ชันเดี่ยวเพื่อให้ทดสอบได้โดยไม่ต้องต่อ DB
func selectExportTabs(
	reportType string,
	groups []GetPriceListGroupResponse,
	groupNameByCode func(code string) string,
	itemNameByCode func(code string) string,
	formulas map[string][]priceListRepository.SubgroupFormula,
	paymentTermMap map[string]GetPaymentTermResponse,
	lastUpdated *time.Time,
) []ExportTab {
	if reportType == ReportTypePricelistDetail {
		return []ExportTab{
			buildPricelistDetailTab(groups, groupNameByCode, itemNameByCode, formulas, lastUpdated),
		}
	}
	return []ExportTab{
		buildDetailTab(groups, groupNameByCode, itemNameByCode, lastUpdated),
		buildBasedPriceTab(groups, paymentTermMap, lastUpdated),
	}
}
