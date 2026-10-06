package patterns

import (
	"testing"

	"prime-erp-core/internal/models"
)

// ValueName ที่ว่าง (item_name เป็นค่าว่างโดยตั้งใจ) ต้องไม่ทำให้ field key ของ row/column
// ว่างหรือชนกัน — sanitizeIdentifier รับ code เป็น primary อยู่แล้ว key จึงยังมาจาก code
func TestSanitizeIdentifierKeepsCodeWhenValueNameEmpty(t *testing.T) {
	a := sanitizeIdentifier("PG03_1", "")
	b := sanitizeIdentifier("PG08_5", "")

	if a == "" || b == "" {
		t.Fatalf("field key ต้องไม่ว่าง: a=%q b=%q", a, b)
	}
	if a == b {
		t.Fatalf("field key ของคนละ code ต้องไม่ชนกัน: a=%q b=%q", a, b)
	}
	if a != "pg03_1" {
		t.Fatalf("sanitizeIdentifier(PG03_1, \"\") = %q, want \"pg03_1\"", a)
	}
}

// composite key ข้ามค่าว่างอยู่แล้ว จึงต้องไม่เหลือ separator ห้อยท้าย
func TestBuildCompositeKeySkipsEmptyValueName(t *testing.T) {
	keys := []models.PriceListSubGroupKeyResponse{
		{GroupCode: "PG01", ValueCode: "PG01_1", ValueName: "150x75x5.5x9.5"},
		{GroupCode: "PG03", ValueCode: "PG03_1", ValueName: ""},
	}

	if got := buildCompositeKey(keys, []string{"PG01", "PG03"}); got != "150x75x5.5x9.5" {
		t.Fatalf("buildCompositeKey = %q, want %q", got, "150x75x5.5x9.5")
	}

	// code key ต้องยังครบทั้งสองส่วน เพราะ ValueCode ไม่ว่าง
	if got := buildCompositeCodeKey(keys, []string{"PG01", "PG03"}); got != "PG01_1|PG03_1" {
		t.Fatalf("buildCompositeCodeKey = %q, want %q", got, "PG01_1|PG03_1")
	}
}
