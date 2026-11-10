package purchaseRepository

import (
	"strings"
	"testing"

	"prime-erp-core/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newPurchaseDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()

	gormx, err := gorm.Open(postgres.New(postgres.Config{
		DSN: "host=localhost user=test password=test dbname=test port=5432 sslmode=disable",
	}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("open dry-run database: %v", err)
	}

	return gormx
}

func TestApplyItemsProductGroupOneNameLikeSearchesCodeOrName(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "product group code", value: "PG01"},
		{name: "product group name", value: "Steel"},
		{name: "case insensitive partial value", value: "ste"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gormx := newPurchaseDryRunDB(t)
			query := gormx.Model(&models.Purchase{}).
				Where("company_code = ? AND site_code = ?", "COMPANY", "SITE")
			query = applyItemsProductGroupOneNameLike(query, gormx, tt.value)

			statement := query.Find(&[]models.Purchase{}).Statement
			sql := statement.SQL.String()

			if !strings.Contains(sql, "product_group_code ILIKE") {
				t.Fatalf("query does not search product_group_code: %s", sql)
			}
			if !strings.Contains(sql, "OR product_group_name ILIKE") {
				t.Fatalf("query does not search product_group_name with OR: %s", sql)
			}
			if !strings.Contains(sql, "EXISTS") {
				t.Fatalf("query does not use the purchase-item EXISTS filter: %s", sql)
			}

			wantPattern := "%" + tt.value + "%"
			patternCount := 0
			for _, variable := range statement.Vars {
				if variable == wantPattern {
					patternCount++
				}
			}
			if patternCount != 2 {
				t.Fatalf("LIKE pattern count = %d, want 2; vars = %#v", patternCount, statement.Vars)
			}
		})
	}
}

func TestApplyItemsProductGroupOneNameLikeSkipsEmptyValue(t *testing.T) {
	gormx := newPurchaseDryRunDB(t)
	query := gormx.Model(&models.Purchase{}).
		Where("company_code = ? AND site_code = ?", "COMPANY", "SITE")
	query = applyItemsProductGroupOneNameLike(query, gormx, "")

	statement := query.Find(&[]models.Purchase{}).Statement
	sql := statement.SQL.String()

	if strings.Contains(sql, "product_group_code") || strings.Contains(sql, "product_group_name") {
		t.Fatalf("empty search value unexpectedly adds a product-group filter: %s", sql)
	}
}
