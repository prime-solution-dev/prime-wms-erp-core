package patterns

import (
	"fmt"
	"sort"
	"strings"

	"prime-erp-core/internal/models"
)

// valueOfGroup คืนค่าตัวเลขของ groupCode ใน subgroup นี้
// bool ตัวที่สองคือ "resolve ค่าได้ไหม" ไม่ใช่ "ไม่ใช่ศูนย์"
func valueOfGroup(sgks []models.PriceListSubGroupKeyResponse, groupCode string) (float64, bool) {
	for _, k := range sgks {
		if k.GroupCode == groupCode {
			return k.ValueNumber, k.HasValue
		}
	}
	return 0, false
}

// nameOfGroup คืนชื่อที่ผู้ใช้เห็นของ groupCode ใช้เป็น tie-break ให้ลำดับนิ่ง
func nameOfGroup(sgks []models.PriceListSubGroupKeyResponse, groupCode string) string {
	for _, k := range sgks {
		if k.GroupCode == groupCode {
			return k.ValueName
		}
	}
	return ""
}

// cmpGroupValue เทียบ subgroup สองตัวที่ groupCode เดียว คืน -1 / 0 / 1
//
// กติกา:
//  1. มีค่าทั้งคู่และไม่เท่ากัน -> เทียบตัวเลขจากน้อยไปมาก
//  2. มีค่าฝ่ายเดียว -> ฝ่ายที่มีค่ามาก่อน ตัวที่ resolve ไม่ได้ไปท้ายเสมอ
//     ห้ามปล่อยให้ค่า 0 ของตัวที่ resolve ไม่ได้ไปกองอยู่หน้าสุด
//  3. ไม่มีค่าทั้งคู่ -> เทียบ ValueName แบบ string ตามพฤติกรรมเดิม
//  4. ค่าเท่ากัน -> tie-break ด้วย ValueName เพื่อให้ลำดับนิ่ง เช่น 100x300 กับ
//     150x200 ที่ value = 30,000.00 เท่ากัน
func cmpGroupValue(a, b []models.PriceListSubGroupKeyResponse, groupCode string) int {
	va, hasA := valueOfGroup(a, groupCode)
	vb, hasB := valueOfGroup(b, groupCode)

	switch {
	case hasA && hasB:
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
	case hasA && !hasB:
		return -1
	case !hasA && hasB:
		return 1
	}

	return strings.Compare(nameOfGroup(a, groupCode), nameOfGroup(b, groupCode))
}

// cmpSubGroupKeys ไล่ groupCodes จากซ้ายไปขวา เจอตัวแรกที่ไม่เท่ากันแล้วจบ
func cmpSubGroupKeys(a, b []models.PriceListSubGroupKeyResponse, groupCodes ...string) int {
	for _, code := range groupCodes {
		if code == "" {
			continue
		}
		if c := cmpGroupValue(a, b, code); c != 0 {
			return c
		}
	}
	return 0
}

// SortSubGroupsByValue เรียง subGroups in-place ตามค่าของ groupCodes ที่ให้มา
//
// ต้องใช้ sort.SliceStable เท่านั้น ห้าม sort.Slice — shared.go บันทึกไว้ว่า
// ลำดับ relative ของ record ที่มี sg.ID เดียวกันต้องคงเดิม ไม่งั้น "record แรก"
// จะไม่ใช่ inventoryWeights[0] อีกต่อไป
func SortSubGroupsByValue(sgs []models.PriceListSubGroupResponse, groupCodes ...string) {
	sort.SliceStable(sgs, func(i, j int) bool {
		return cmpSubGroupKeys(sgs[i].SubGroupKeys, sgs[j].SubGroupKeys, groupCodes...) < 0
	})
}

// orderedUnique เก็บค่าของ field จาก rows ตามลำดับที่เจอครั้งแรก ไม่ซ้ำ และข้ามค่าว่าง
//
// ใช้แทน sort.Strings(mapKeys) ได้เมื่อ rows ถูกสร้างจาก subGroups ที่เรียงแล้ว
// การเรียง key ที่ประกอบเสร็จแล้วทำไม่ได้ เพราะ buildCompositeKeyBy ข้ามค่าว่าง
// ตอน join และ columnKey ยังผ่าน sanitizeIdentifier มาอีกชั้น จึงแยกกลับไม่ได้
func orderedUnique(rows []AGGridRowData, field string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(rows))

	for _, row := range rows {
		v, ok := row[field]
		if !ok {
			continue
		}
		key := strings.TrimSpace(fmt.Sprintf("%v", v))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}

	return out
}

// orderedUniqueBy เวอร์ชันที่ดึง key ด้วยฟังก์ชัน ใช้กับ subGroups โดยตรง
func orderedUniqueBy[T any](items []T, keyOf func(T) string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(items))

	for _, item := range items {
		key := strings.TrimSpace(keyOf(item))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}

	return out
}

// splitGroupCodes แยกสตริงแกนของ pattern เช่น "PG03|PG08|PG05" เป็น slice
// ตัดช่องว่างและข้ามค่าว่าง คืน nil เมื่อไม่เหลืออะไร
func splitGroupCodes(s string) []string {
	parts := strings.Split(s, "|")
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

// codeValue คือค่าตัวเลขของ item หนึ่งตัว พร้อมธงว่า resolve ได้ไหม
type codeValue struct {
	value float64
	has   bool
}

// valueByCode คือ index จาก item_code ไปหาค่าตัวเลข
type valueByCode map[string]codeValue

// newValueByCode สร้าง index จาก SubGroupKeys ทั้งหมดใน subGroups
//
// key คือ ValueCode (item_code) ซึ่งไม่ผ่าน sanitizeIdentifier และไม่ถูก join
// เป็น composite จึง lookup กลับได้อย่างปลอดภัย ต่างจาก columnKey / row_group_value
// ที่ประกอบมาแล้วและแยกกลับไม่ได้
func newValueByCode(sgs []models.PriceListSubGroupResponse) valueByCode {
	idx := valueByCode{}

	for _, sg := range sgs {
		for _, k := range sg.SubGroupKeys {
			if k.ValueCode == "" {
				continue
			}
			if existing, ok := idx[k.ValueCode]; ok && existing.has {
				continue
			}
			idx[k.ValueCode] = codeValue{value: k.ValueNumber, has: k.HasValue}
		}
	}

	return idx
}

// Less เทียบ item สองตัวด้วย code เป็นหลัก ถ้า resolve ไม่ได้ทั้งคู่จึง fallback
// ไปเทียบ label แบบ string ตามพฤติกรรมเดิม
func (idx valueByCode) Less(codeA, labelA, codeB, labelB string) bool {
	a, hasA := idx[codeA]
	b, hasB := idx[codeB]

	resolvedA := hasA && a.has
	resolvedB := hasB && b.has

	switch {
	case resolvedA && resolvedB:
		if a.value != b.value {
			return a.value < b.value
		}
	case resolvedA && !resolvedB:
		return true
	case !resolvedA && resolvedB:
		return false
	}

	return labelA < labelB
}

// newValueByName สร้าง index จาก ValueName ไปหาค่าตัวเลข เฉพาะ groupCode เดียว
//
// จำกัดอยู่กลุ่มเดียวจึงไม่ชนกัน — ถ้าทำ index ข้ามกลุ่มจะชน เช่น PG05 มี item
// ชื่อ "100" (value 100) และ PG06 ก็มี item ชื่อ "100" (value 100) เหมือนกัน
func newValueByName(sgs []models.PriceListSubGroupResponse, groupCode string) valueByCode {
	idx := valueByCode{}

	for _, sg := range sgs {
		for _, k := range sg.SubGroupKeys {
			if k.GroupCode != groupCode || k.ValueName == "" {
				continue
			}
			if existing, ok := idx[k.ValueName]; ok && existing.has {
				continue
			}
			idx[k.ValueName] = codeValue{value: k.ValueNumber, has: k.HasValue}
		}
	}

	return idx
}

// sortLabelsByValue เรียง label in-place ด้วยค่าตัวเลขของ groupCode ที่ให้มา
// label ที่ resolve ไม่ได้จะไปท้ายเสมอ และเรียงกันเองด้วย string compare
func sortLabelsByValue(labels []string, sgs []models.PriceListSubGroupResponse, groupCode string) {
	idx := newValueByName(sgs, groupCode)
	sort.SliceStable(labels, func(i, j int) bool {
		return idx.Less(labels[i], labels[i], labels[j], labels[j])
	})
}
