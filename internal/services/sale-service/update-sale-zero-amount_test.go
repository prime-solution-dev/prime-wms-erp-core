package saleService

import (
	"strings"
	"testing"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	gormtests "gorm.io/gorm/utils/tests"
)

// ยอดเงินหัวใบ SO ต้องเขียนลงแม้เป็น 0 — กติกาเดียวกับ UpdateQuotation
func TestUpdateSaleZeroableAmountsWritesZero(t *testing.T) {
	db, err := gorm.Open(gormtests.DummyDialector{}, &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("gorm.Open(DummyDialector) error = %v", err)
	}
	var sql string
	if err := db.Callback().Update().After("gorm:update").Register("capture_update_sql", func(tx *gorm.DB) {
		sql = tx.Explain(tx.Statement.SQL.String(), tx.Statement.Vars...)
	}); err != nil {
		t.Fatalf("register callback error = %v", err)
	}

	id := uuid.New()
	if err := updateSaleZeroableAmounts(db, models.Sale{ID: id}); err != nil {
		t.Fatalf("updateSaleZeroableAmounts error = %v", err)
	}

	for _, column := range []string{
		"ref_po_doc",
		"total_transport_cost",
		"total_transport_cost_vat",
		"total_discount",
		"total_vat",
		"total_amount",
		"total_weight",
		"subtotal_excl_transport",
		"subtotal_weight_excl_transport",
		"subtotal_excl_vat",
	} {
		if !strings.Contains(sql, column) {
			t.Errorf("UPDATE ไม่มีคอลัมน์ %s แม้ค่าจะเป็น 0: %s", column, sql)
		}
	}
	if !strings.Contains(sql, id.String()) {
		t.Errorf("UPDATE ไม่มี id ที่ระบุ: %s", sql)
	}
}
