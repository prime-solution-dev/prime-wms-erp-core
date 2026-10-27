package priceListRepository

import (
	"os"
	"strings"
	"testing"

	"prime-erp-core/internal/db"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const fkIndexesMigrationFile = "testdata/2026-10-16-price-list-fk-indexes.sql"

// รัน migration ทีละ statement เพราะ CREATE INDEX CONCURRENTLY รันใน multi-statement/transaction ไม่ได้
func TestMigration_PriceListFkIndexes(t *testing.T) {
	gormx, err := db.ConnectGORM("prime_erp")
	require.NoError(t, err)
	defer db.CloseGORM(gormx)

	raw, err := os.ReadFile(fkIndexesMigrationFile)
	require.NoError(t, err)
	var stmts []string
	for _, s := range strings.Split(string(raw), ";") {
		var lines []string
		for _, l := range strings.Split(s, "\n") {
			if !strings.HasPrefix(strings.TrimSpace(l), "--") {
				lines = append(lines, l)
			}
		}
		if q := strings.TrimSpace(strings.Join(lines, "\n")); q != "" {
			stmts = append(stmts, q)
		}
	}
	require.Len(t, stmts, 7)

	// รัน 2 รอบเพื่อพิสูจน์ IF NOT EXISTS (idempotent)
	for i := 0; i < 2; i++ {
		for _, q := range stmts {
			require.NoError(t, gormx.Exec(q).Error, q)
		}
	}

	want := map[string]string{
		"idx_price_list_sub_group_price_list_group_id":       "price_list_sub_group",
		"idx_price_list_sub_group_key_sub_group_id":          "price_list_sub_group_key",
		"idx_price_list_group_term_price_list_group_id":      "price_list_group_term",
		"idx_price_list_group_extra_price_list_group_id":     "price_list_group_extra",
		"idx_price_list_group_extra_key_group_extra_id":      "price_list_group_extra_key",
		"idx_price_list_group_company_site":                  "price_list_group",
		"idx_price_list_subgroup_formulas_map_subgroup_code": "price_list_subgroup_formulas_map",
	}
	for name, table := range want {
		var n int64
		require.NoError(t, gormx.Raw(`SELECT count(*) FROM pg_indexes WHERE indexname = ? AND tablename = ?`, name, table).Scan(&n).Error)
		require.EqualValues(t, 1, n, name)
	}
}

// ตรวจว่า query ใหม่ (col IN (...) [OR col IS NULL]) ให้ผลเท่า COALESCE(col,”) IN (...) เดิมทั้งกรณี NULL และ ”
func TestGetPriceListSubGroupFormulasMapBySubGroupCodes_NullAndEmpty(t *testing.T) {
	gormx, err := db.ConnectGORM("prime_erp")
	require.NoError(t, err)
	defer db.CloseGORM(gormx)
	// repository ใช้ sqlx — ชี้ไป DB ตัวเดียวกับ gorm
	t.Setenv("database_sqlx_url_prime_erp", os.Getenv("database_gorm_url_prime_erp"))

	require.NoError(t, gormx.Exec(`ALTER TABLE price_list_subgroup_formulas_map ALTER COLUMN price_list_subgroup_code DROP NOT NULL`).Error)
	require.NoError(t, gormx.Exec(`DELETE FROM price_list_subgroup_formulas_map`).Error)
	require.NoError(t, gormx.Exec(`DELETE FROM price_list_formulas`).Error)
	require.NoError(t, gormx.Exec(`INSERT INTO price_list_formulas (id, formula_code, name, uom, formula_type, expression, params, rounding, create_dtm)
		VALUES (?, 'FX-1', 'n', 'kg', 'price_calc', '', '{}', 2, now())`, uuid.New()).Error)
	for _, code := range []any{"SGX", "SGY", "", nil} {
		require.NoError(t, gormx.Exec(`INSERT INTO price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
			VALUES (?, ?, 'FX-1', true, now())`, uuid.New(), code).Error)
	}

	t.Run("ไม่มี '' ใน list: ไม่เอา NULL/''", func(t *testing.T) {
		got, err := GetPriceListSubGroupFormulasMapBySubGroupCodes([]string{"SGX"})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Len(t, got["SGX"], 1)
	})

	t.Run("มี '' ใน list: แมตช์ทั้ง '' และ NULL (รวมใต้ key '')", func(t *testing.T) {
		got, err := GetPriceListSubGroupFormulasMapBySubGroupCodes([]string{"SGX", ""})
		require.NoError(t, err)
		require.Len(t, got["SGX"], 1)
		require.Len(t, got[""], 2)
		require.Len(t, got["SGY"], 0)
	})

	t.Run("list ว่าง", func(t *testing.T) {
		got, err := GetPriceListSubGroupFormulasMapBySubGroupCodes(nil)
		require.NoError(t, err)
		require.Empty(t, got)
	})
}
