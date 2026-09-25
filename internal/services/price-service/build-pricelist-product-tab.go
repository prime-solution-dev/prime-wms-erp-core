package priceService

import (
	"fmt"
	"sort"
	"strings"
	"time"

	externalProductService "prime-erp-core/external/product-service"
	priceListRepository "prime-erp-core/internal/repositories/priceList"
)

// keyPart คือ code/value/seq ของ key หนึ่งตัว ใช้ร่วมกันทั้งฝั่ง product และ subgroup
type keyPart struct {
	code  string
	value string
	seq   int
}

// joinKey สร้าง key ตามกติกาเดียวกับ warehouse-core GetInventoryWeightByKey:
// เรียงตาม seq แล้วต่อ code ด้วย "|" และ value ด้วย "|" — ต้องตรงกันทุกตัว (ไม่มี wildcard)
// ใช้ "\x00" คั่นระหว่างส่วน codes กับส่วน values เพราะเป็นไบต์ที่ไม่มีทางปรากฏใน code/value
// จริง ๆ จึงไม่ชนกัน (ต่างจาก "#" ที่ codes=[A] values=[B#C] จะชนกับ codes=[A#B] values=[C])
func joinKey(parts []keyPart) string {
	if len(parts) == 0 {
		return ""
	}
	// ถือว่า seq ไม่ซ้ำภายใน key ชุดเดียวกัน (warehouse-core ใช้ sort.Slice ซึ่งไม่ stable จึงเลียนแบบกรณี seq ซ้ำไม่ได้)
	sort.SliceStable(parts, func(i, j int) bool { return parts[i].seq < parts[j].seq })
	codes := make([]string, len(parts))
	values := make([]string, len(parts))
	for i, p := range parts {
		codes[i] = p.code
		values[i] = p.value
	}
	return strings.Join(codes, "|") + "\x00" + strings.Join(values, "|")
}

// productKey ใช้เฉพาะ product_group ที่ active เหมือนฝั่ง warehouse-core
func productKey(p externalProductService.GetProductsComponent) string {
	parts := make([]keyPart, 0, len(p.ProductGroup))
	for _, g := range p.ProductGroup {
		if !g.ActiveFlg {
			continue
		}
		parts = append(parts, keyPart{code: g.GroupCode, value: g.GroupValue, seq: g.Seq})
	}
	return joinKey(parts)
}

// subGroupKey ต้องไม่กรอง key ใด ๆ ออก เพื่อให้ตรงกับ buildKeyValueGroups ของ warehouse-core
func subGroupKey(sg SubGroup) string {
	parts := make([]keyPart, 0, len(sg.GroupKeys))
	for _, k := range sg.GroupKeys {
		parts = append(parts, keyPart{code: k.Code, value: k.Value, seq: k.Seq})
	}
	return joinKey(parts)
}

// getProducts เป็น var เพื่อให้ test แทนที่ได้โดยไม่ต้องยิง HTTP จริง
var getProducts = externalProductService.GetProduct

// fetchAllProducts ดึง product master ที่ active ทั้งหมดในคำขอเดียว
// product-core เรียง paging ตาม update_dtm อย่างเดียว (ไม่มี tie-breaker คงที่) การแบ่งหน้า
// จึงข้ามแถวได้เงียบ ๆ เมื่อมีหลาย record update_dtm ชนกันคาบเกี่ยวรอยต่อหน้า — ส่ง Page/PageSize
// เป็นค่าว่าง (0) ให้ normalizePaging คืนทุกแถวในคำขอเดียว เหมือนที่ warehouse-core ทำ
func fetchAllProducts(companyCode string, siteCodes []string) ([]externalProductService.GetProductsComponent, error) {
	res, err := getProducts(externalProductService.GetProductRequest{
		CompanyCode: []string{companyCode},
		SiteCode:    siteCodes,
		ActiveFlg:   []bool{true},
	})
	if err != nil {
		return nil, fmt.Errorf("get products: %w", err)
	}

	seen := map[string]bool{}
	out := make([]externalProductService.GetProductsComponent, 0, len(res.Products))
	for _, p := range res.Products {
		// product ตัวเดียวกันอาจกลับมาหลายครั้งตามจำนวน site
		if seen[p.ProductCode] {
			continue
		}
		seen[p.ProductCode] = true
		out = append(out, p)
	}
	return out, nil
}

type matchedSubGroup struct {
	group GetPriceListGroupResponse
	sg    SubGroup
}

// productWeightSpec คืนน้ำหนักของ base unit (flag_base = true) จาก product master —
// นิยามเดียวกับ WeightSpecFromUnits ของ warehouse-core (get-inventory-weight-by-key.go)
// คืน 0 เมื่อไม่มี unit ไหนเป็น base unit
func productWeightSpec(p externalProductService.GetProductsComponent) float64 {
	for _, u := range p.Units {
		if u.FlagBase {
			return u.Weight
		}
	}
	return 0
}

// buildPricelistProductTab ประกอบ tab "Template" ของ Product Pricelist Report
// แถวต่อ (product × subgroup ที่ key ตรงกัน) นำหน้าด้วย Product Code / Product Name
// ส่วนที่เหลือเหมือน Pricelist Detail Report ทุกคอลัมน์
//
// onlyMatched = true เมื่อผู้ใช้กรอง Pricelist group — product ที่ไม่ตรงจะไม่ออก
// ถ้า false product ที่ไม่ตรงจะออก 1 แถว ช่อง pricelist ว่าง แต่คอลัมน์กลุ่มสินค้า
// เติมจาก product_groups ของสินค้าเอง
func buildPricelistProductTab(
	groups []GetPriceListGroupResponse,
	products []externalProductService.GetProductsComponent,
	groupNameByCode func(code string) string,
	itemNameByCode func(code string) (string, bool),
	fixedColumns []priceListRepository.SubGroupKeyColumn,
	formulas map[string][]priceListRepository.SubgroupFormula,
	lastUpdated *time.Time,
	onlyMatched bool,
) ExportTab {
	cols := collectGroupColumns(groups, groupNameByCode, fixedColumns)
	colSet := map[string]bool{}
	for _, c := range cols {
		colSet[c.code] = true
	}

	byKey := map[string][]matchedSubGroup{}
	for _, g := range groups {
		for _, sg := range g.SubGroups {
			if isInactiveSubGroup(sg.UdfJson) {
				continue
			}
			k := subGroupKey(sg)
			if k == "" {
				continue
			}
			byKey[k] = append(byKey[k], matchedSubGroup{group: g, sg: sg})
		}
	}

	sorted := make([]externalProductService.GetProductsComponent, len(products))
	copy(sorted, products)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].ProductCode < sorted[j].ProductCode })

	rows := make([]map[string]interface{}, 0, len(sorted))
	for _, p := range sorted {
		matches := byKey[productKey(p)]
		for _, m := range matches {
			row := pricelistDetailRow(m.group, m.sg, cols, itemNameByCode, formulas)
			row["product_code"] = p.ProductCode
			row["product_name"] = p.ProductName
			// Weight-spec ต้องเป็นน้ำหนักของสินค้าแถวนั้นเอง ไม่ใช่ของ subgroup —
			// สินค้าสองตัวที่จับคู่ subgroup เดียวกันมีน้ำหนักต่างกันได้
			row["total_weight"] = productWeightSpec(p)
			rows = append(rows, row)
		}
		if len(matches) > 0 || onlyMatched {
			continue
		}
		rows = append(rows, unmatchedProductRow(p, cols, colSet, itemNameByCode))
	}

	columns := append([]ExportColumn{
		{Field: "product_code", HeaderName: "Product Code"},
		{Field: "product_name", HeaderName: "Product Name"},
	}, pricelistDetailColumns(cols)...)

	return ExportTab{
		Name: "Template",
		Headers: ExportTabHeaders{
			Report:      "Pricelist Detail By Product",
			LastUpdated: formatOptionalTimestamp(lastUpdated),
			Download:    formatTimestamp(time.Now()),
		},
		Columns: columns,
		Rows:    rows,
	}
}

// unmatchedProductRow ทุกช่อง pricelist เป็นค่าว่าง — ใช้ "" แทน 0 เพื่อไม่ให้ดูเหมือนราคา 0
func unmatchedProductRow(
	p externalProductService.GetProductsComponent,
	cols []groupColumn,
	colSet map[string]bool,
	itemNameByCode func(code string) (string, bool),
) map[string]interface{} {
	row := map[string]interface{}{
		"product_code": p.ProductCode,
		"product_name": p.ProductName,
		"total_weight": productWeightSpec(p),
		"avg_weight":   float64(0),
	}
	for _, c := range pricelistDetailColumns(cols) {
		if _, ok := row[c.Field]; !ok {
			row[c.Field] = ""
		}
	}
	for _, g := range p.ProductGroup {
		if !g.ActiveFlg || !colSet[g.GroupCode] {
			continue
		}
		name, found := itemNameByCode(g.GroupValue)
		if !found {
			name = g.GroupValue
		}
		row[g.GroupCode] = name
		row[g.GroupCode+groupCodeColumnSuffix] = g.GroupValue
	}
	return row
}
