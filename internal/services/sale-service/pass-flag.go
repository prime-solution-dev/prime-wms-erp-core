package saleService

import "strings"

// normalizePassFlag คุมค่า flag ผลตรวจให้เหลือแค่ Y/N
//
// ของเดิม create-sale.go เขียนทับเป็น "Y" ทุกใบ (คอมเมนต์บอกว่า "หน้าบ้าน validate แล้ว")
// ผลคือใบที่ราคาไม่ผ่านก็ถูกบันทึกว่าผ่าน และไม่มีทางรู้ย้อนหลังว่าใบไหนผ่านแบบมีเงื่อนไข
//
// ค่าที่ไม่ใช่ N ให้ถือเป็น Y เพราะ payload เก่าที่ไม่ได้ส่ง flag มาต้องได้พฤติกรรมเดิม
func normalizePassFlag(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "N") {
		return "N"
	}

	return "Y"
}
