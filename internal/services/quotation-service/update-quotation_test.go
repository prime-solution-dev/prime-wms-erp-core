package quotationService

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
	gormtests "gorm.io/gorm/utils/tests"
)

// DummyDialector ไม่ต่อ DB และ DryRun ไม่ execute — ตรวจแค่ SQL ที่ประกอบออกมา
func dryRunDB(t *testing.T) (*gorm.DB, *[]string) {
	t.Helper()

	db, err := gorm.Open(gormtests.DummyDialector{}, &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("gorm.Open(DummyDialector) error = %v", err)
	}

	statements := &[]string{}
	err = db.Callback().Update().After("gorm:update").Register("capture_update_sql", func(tx *gorm.DB) {
		*statements = append(*statements, tx.Explain(tx.Statement.SQL.String(), tx.Statement.Vars...))
	})
	if err != nil {
		t.Fatalf("register callback error = %v", err)
	}

	return db, statements
}

// updateQuotationSalePerson ต้องเขียน sale_person_code แม้ค่าจะเป็น "" (ล้าง Sales Person)
//
// เดิม UpdateQuotation ใช้ tx.Model(...).Updates(quotation) แบบส่ง struct — GORM ข้าม
// zero-value field โดยอัตโนมัติ ทำให้กด allow-clear หรือเปลี่ยนลูกค้าแล้ว sale_person_code
// เดิมค้างอยู่ในฐานข้อมูลและถูกก็อปต่อไปที่ SO
func TestUpdateQuotationSalePersonClearsToEmptyString(t *testing.T) {
	db, statements := dryRunDB(t)
	id := uuid.New()

	if err := updateQuotationSalePerson(db, id, ""); err != nil {
		t.Fatalf("updateQuotationSalePerson error = %v", err)
	}

	if len(*statements) != 1 {
		t.Fatalf("ได้ UPDATE %d คำสั่ง want 1: %v", len(*statements), *statements)
	}

	sql := (*statements)[0]
	if !strings.Contains(sql, "sale_person_code") {
		t.Errorf("UPDATE ไม่มีคอลัมน์ sale_person_code แม้ค่าจะเป็นค่าว่าง: %s", sql)
	}
	if !strings.Contains(sql, id.String()) {
		t.Errorf("UPDATE ไม่มี id ที่ระบุ: %s", sql)
	}
}

// ค่าไม่ว่างต้องยังตั้งได้ตามปกติ
func TestUpdateQuotationSalePersonSetsNonEmptyValue(t *testing.T) {
	db, statements := dryRunDB(t)
	id := uuid.New()

	if err := updateQuotationSalePerson(db, id, "U1"); err != nil {
		t.Fatalf("updateQuotationSalePerson error = %v", err)
	}

	if len(*statements) != 1 {
		t.Fatalf("ได้ UPDATE %d คำสั่ง want 1: %v", len(*statements), *statements)
	}

	sql := (*statements)[0]
	if !strings.Contains(sql, "sale_person_code") || !strings.Contains(sql, "U1") {
		t.Errorf("UPDATE ต้องตั้งค่าที่ส่งมา: %s", sql)
	}
}
