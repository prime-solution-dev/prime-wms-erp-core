package saleRepository

import (
	"strings"
	"testing"

	"gorm.io/gorm"
	gormtests "gorm.io/gorm/utils/tests"
)

func salePersonSQL(t *testing.T, codes []string) string {
	t.Helper()
	db, err := gorm.Open(gormtests.DummyDialector{}, &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("gorm.Open(DummyDialector) error = %v", err)
	}
	return db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		var ids []string
		return applySalePersonFilter(tx.Table("sale").Select("sale.id"), codes).Scan(&ids)
	})
}

func TestApplySalePersonFilterUsesPlaceholder(t *testing.T) {
	sql := salePersonSQL(t, []string{"U1", "O'Brien"})

	if !strings.Contains(sql, "sale.sale_person_code IN") {
		t.Fatalf("ไม่มีเงื่อนไข sale_person_code: %s", sql)
	}
	if !strings.Contains(sql, "U1") {
		t.Errorf("ค่า U1 ไม่ไปถึง SQL: %s", sql)
	}
}

func TestApplySalePersonFilterEmptyDoesNotFilter(t *testing.T) {
	sql := salePersonSQL(t, nil)

	if strings.Contains(sql, "sale_person_code") {
		t.Errorf("list ว่างต้องไม่กรอง: %s", sql)
	}
}
