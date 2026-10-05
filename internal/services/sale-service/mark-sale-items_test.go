package saleService

import (
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	gormtests "gorm.io/gorm/utils/tests"
)

// dryRunDB คืน *gorm.DB ที่ "สร้าง SQL แต่ไม่ยิงออกไปไหน"
//
// DummyDialector ไม่เปิด connection และ DryRun ไม่ execute — เทสชุดนี้จึงไม่ต่อ DB จริง
// แต่ยังตรวจได้ว่าเงื่อนไขที่เราคิดว่าใส่ไว้ ไปถึงคำสั่ง UPDATE จริงหรือเปล่า
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

// เส้นอัตโนมัติต้องแตะแค่บรรทัดที่ยังเปิดอยู่
//
// เดิม MarkSaleItemsCompleted ยิง WHERE sale_item IN (...) เปล่าๆ บรรทัดที่ถูกยกเลิก
// ระหว่างอ่านกับเขียนจะถูกเขียนทับเป็น COMPLETED และใบจองสองใบของ SO เดียวกันก็เขียนซ้ำกัน
func TestMarkOpenSaleItemsCompletedOnlyTouchesOpenLines(t *testing.T) {
	db, statements := dryRunDB(t)

	if _, _, err := MarkOpenSaleItemsCompleted(db, []string{"SALE-1", "SALE-2"}, "system", time.Now()); err != nil {
		t.Fatalf("MarkOpenSaleItemsCompleted error = %v", err)
	}

	if len(*statements) != 1 {
		t.Fatalf("ได้ UPDATE %d คำสั่ง want 1: %v", len(*statements), *statements)
	}

	sql := (*statements)[0]

	if !strings.Contains(sql, "status IS NULL OR status NOT IN") {
		t.Errorf("UPDATE ไม่มีการ์ดสถานะ: %s", sql)
	}

	for _, status := range closedSaleItemStatusList {
		if !strings.Contains(sql, status) {
			t.Errorf("การ์ดสถานะไม่ครอบ %q: %s", status, sql)
		}
	}

	if !strings.Contains(sql, "COMPLETED") {
		t.Errorf("เส้นอัตโนมัติต้องตั้งเป็น COMPLETED เท่านั้น: %s", sql)
	}
}

// เส้นมือ (POST /sale/UpdateSaleItemStatus) ต้องยังตั้งสถานะอะไรก็ได้และเขียนทับได้
// เพราะหน้าที่ของมันคือแก้สถานะตามที่ผู้ใช้สั่ง รวมถึงแก้ที่ตั้งผิดกลับ
func TestMarkSaleItemsCompletedKeepsTheManualPathUnguarded(t *testing.T) {
	db, statements := dryRunDB(t)

	if _, _, err := MarkSaleItemsCompleted(db, []string{"SALE-1"}, "CANCELED", "someone", time.Now()); err != nil {
		t.Fatalf("MarkSaleItemsCompleted error = %v", err)
	}

	if len(*statements) != 1 {
		t.Fatalf("ได้ UPDATE %d คำสั่ง want 1: %v", len(*statements), *statements)
	}

	sql := (*statements)[0]

	if strings.Contains(sql, "status IS NULL OR status NOT IN") {
		t.Errorf("เส้นมือต้องไม่มีการ์ดสถานะ ไม่งั้นแก้สถานะที่ตั้งผิดกลับไม่ได้: %s", sql)
	}

	if !strings.Contains(sql, "CANCELED") {
		t.Errorf("เส้นมือต้องตั้งสถานะตามที่สั่ง: %s", sql)
	}
}

func TestMarkSaleItemsIgnoresEmptyInput(t *testing.T) {
	db, statements := dryRunDB(t)

	updated, completed, err := MarkOpenSaleItemsCompleted(db, nil, "system", time.Now())
	if err != nil {
		t.Fatalf("MarkOpenSaleItemsCompleted(nil) error = %v", err)
	}

	if len(updated) != 0 || len(completed) != 0 || len(*statements) != 0 {
		t.Errorf("ไม่มีบรรทัดให้ปิด ต้องไม่ยิง UPDATE เลย ได้ %v", *statements)
	}
}
