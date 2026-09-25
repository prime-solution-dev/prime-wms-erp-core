package priceService

import (
	"sort"
	"strings"

	externalProductService "prime-erp-core/external/product-service"
)

// keyPart คือ code/value/seq ของ key หนึ่งตัว ใช้ร่วมกันทั้งฝั่ง product และ subgroup
type keyPart struct {
	code  string
	value string
	seq   int
}

// joinKey สร้าง key ตามกติกาเดียวกับ warehouse-core GetInventoryWeightByKey:
// เรียงตาม seq แล้วต่อ code ด้วย "|" และ value ด้วย "|" — ต้องตรงกันทุกตัว (ไม่มี wildcard)
func joinKey(parts []keyPart) string {
	if len(parts) == 0 {
		return ""
	}
	sort.SliceStable(parts, func(i, j int) bool { return parts[i].seq < parts[j].seq })
	codes := make([]string, len(parts))
	values := make([]string, len(parts))
	for i, p := range parts {
		codes[i] = p.code
		values[i] = p.value
	}
	return strings.Join(codes, "|") + "#" + strings.Join(values, "|")
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

func subGroupKey(sg SubGroup) string {
	parts := make([]keyPart, 0, len(sg.GroupKeys))
	for _, k := range sg.GroupKeys {
		if k.Code == "" {
			continue
		}
		parts = append(parts, keyPart{code: k.Code, value: k.Value, seq: k.Seq})
	}
	return joinKey(parts)
}
