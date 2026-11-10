package utils

import (
	"strings"

	"gorm.io/gorm"
)

// ตัวกรองสำหรับช่องค้นหา "Multi Select" ที่หัวคอลัมน์ (Search / SELECT)
//
// สำคัญ: ชื่อคอลัมน์ (codeColumn, MultiSelectTerm.Column) ต้องเป็นค่าคงที่ในโค้ด (whitelist)
// ห้ามรับมาจาก request เพราะถูกต่อลง SQL ตรงๆ — ส่วนค่าค้นหาทุกตัวใช้ parameter binding

// MultiSelectTerm = คำค้น 1 คำของคอลัมน์หนึ่ง
//   - มี In  → ติ๊กเลือกทีละตัว: เอาเฉพาะรหัสใน In (Like เป็นแค่คำค้นตอนเลือก)
//   - ไม่มี In → Search / SELECT กด All: ทุกแถวที่ Column ILIKE Like ("" = ทุกแถว) ตัดรหัสใน Not ออก
type MultiSelectTerm struct {
	Column string
	Like   string
	In     []string
	Not    []string
}

// MultiSelectCondition → "((codeColumn IN ?) OR (col ILIKE ? AND codeColumn NOT IN ?) OR ...)"
func MultiSelectCondition(codeColumn string, terms []MultiSelectTerm) (string, []interface{}) {
	if len(terms) == 0 {
		return "", nil
	}
	parts := make([]string, 0, len(terms))
	var args []interface{}
	for _, term := range terms {
		if len(term.In) > 0 {
			parts = append(parts, "("+codeColumn+" IN ?)")
			args = append(args, term.In)
			continue
		}

		var conds []string
		if term.Like != "" {
			conds = append(conds, term.Column+" ILIKE ?")
			args = append(args, "%"+term.Like+"%")
		}
		if len(term.Not) > 0 {
			conds = append(conds, codeColumn+" NOT IN ?")
			args = append(args, term.Not)
		}
		if len(conds) == 0 {
			// All ของคำค้นว่าง ไม่ได้ตัดอะไร = ทุกแถว → OR ทั้งกลุ่มเป็นจริง
			return "", nil
		}
		parts = append(parts, "("+strings.Join(conds, " AND ")+")")
	}
	return "(" + strings.Join(parts, " OR ") + ")", args
}

// ApplyMultiSelectFilter → AND (term1 OR term2 ...) — เรียก 1 ครั้งต่อ 1 คอลัมน์
func ApplyMultiSelectFilter(query *gorm.DB, codeColumn string, terms []MultiSelectTerm) *gorm.DB {
	if cond, args := MultiSelectCondition(codeColumn, terms); cond != "" {
		return query.Where(cond, args...)
	}
	return query
}
