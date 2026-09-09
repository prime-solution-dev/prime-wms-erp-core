//go:build integration

package priceService

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"prime-erp-core/internal/db"

	"github.com/google/uuid"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"
)

var pricelistContainer tc.Container

func TestMain(m *testing.M) {
	ctx := context.Background()
	req := tc.ContainerRequest{
		Image:        "postgres:16",
		Env:          map[string]string{"POSTGRES_PASSWORD": "test", "POSTGRES_USER": "test", "POSTGRES_DB": "testdb"},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}
	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		fmt.Printf("failed to start postgres container: %v\n", err)
		os.Exit(1)
	}
	pricelistContainer = container

	host, err := container.Host(ctx)
	if err != nil {
		fmt.Printf("host: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}
	mapped, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		fmt.Printf("port: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}
	os.Setenv("database_gorm_url_prime_erp",
		fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, mapped.Port()))

	if err := createPricelistSchema(); err != nil {
		fmt.Printf("failed to create schema: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	code := m.Run()
	_ = pricelistContainer.Terminate(ctx)
	os.Exit(code)
}

// createPricelistSchema mirrors the production price list tables the upload
// path writes to, including the unique indexes its ON CONFLICT clauses need.
func createPricelistSchema() error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS price_list_group (
			id uuid PRIMARY KEY,
			company_code text, site_code text, group_code text, group_name text,
			price_unit double precision, price_weight double precision,
			before_price_unit double precision, before_price_weight double precision,
			currency text, effective_date timestamp NULL, remark text, group_key text,
			create_by text, create_dtm timestamp, update_by text, update_dtm timestamp
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_price_list_group_group_code ON price_list_group(group_code);`,
		`CREATE TABLE IF NOT EXISTS price_list_group_history (
			id uuid PRIMARY KEY,
			price_list_group_id uuid REFERENCES price_list_group(id),
			group_code text, create_dtm timestamp
		);`,
		`CREATE TABLE IF NOT EXISTS price_list_sub_group_history (
			id uuid PRIMARY KEY,
			price_list_group_id uuid REFERENCES price_list_group(id) ON DELETE CASCADE,
			subgroup_key text, create_dtm timestamp
		);`,
		`CREATE TABLE IF NOT EXISTS price_list_group_key (
			id uuid PRIMARY KEY,
			price_list_group_id uuid REFERENCES price_list_group(id),
			seq integer, code text, value text
		);`,
		`CREATE TABLE IF NOT EXISTS price_list_group_term (
			id uuid PRIMARY KEY,
			price_list_group_id uuid REFERENCES price_list_group(id),
			term_code text,
			pdc double precision, pdc_percent double precision,
			due double precision, due_percent double precision,
			create_by text, create_dtm timestamp, update_by text, update_dtm timestamp
		);`,
		`CREATE TABLE IF NOT EXISTS price_list_group_extra (
			id uuid PRIMARY KEY,
			price_list_group_id uuid REFERENCES price_list_group(id),
			extra_key text, condition_code text, operator text,
			value_int double precision, length_extra_key integer,
			cond_range_min double precision, cond_range_max double precision,
			create_by text, create_dtm timestamp, update_by text, update_dtm timestamp
		);`,
		`CREATE TABLE IF NOT EXISTS price_list_group_extra_key (
			id uuid PRIMARY KEY,
			group_extra_id uuid REFERENCES price_list_group_extra(id),
			seq integer, code text, value text
		);`,
		`CREATE TABLE IF NOT EXISTS price_list_sub_group (
			id uuid PRIMARY KEY,
			price_list_group_id uuid REFERENCES price_list_group(id),
			subgroup_code text, subgroup_key text, is_trading boolean,
			price_unit double precision, extra_price_unit double precision,
			term_price_unit double precision, total_net_price_unit double precision,
			price_weight double precision, extra_price_weight double precision,
			term_price_weight double precision, total_net_price_weight double precision,
			before_price_unit double precision, before_extra_price_unit double precision,
			before_term_price_unit double precision, before_total_net_price_unit double precision,
			before_price_weight double precision, before_extra_price_weight double precision,
			before_term_price_weight double precision, before_total_net_price_weight double precision,
			effective_date timestamp NULL, remark text, udf_json json NULL,
			create_by text, create_dtm timestamp, update_by text, update_dtm timestamp
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_price_list_sub_group_subgroup_code ON price_list_sub_group(subgroup_code);`,
		`CREATE TABLE IF NOT EXISTS price_list_sub_group_key (
			id uuid PRIMARY KEY,
			sub_group_id uuid REFERENCES price_list_sub_group(id),
			seq integer, code text, value text
		);`,
		`CREATE TABLE IF NOT EXISTS price_list_formulas (
			id uuid PRIMARY KEY,
			formula_code text NOT NULL, name text NOT NULL, uom text NOT NULL,
			formula_type text NOT NULL, expression text NOT NULL,
			params jsonb NOT NULL, rounding integer NOT NULL, create_dtm timestamp NOT NULL
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_price_list_formulas_formula_code ON price_list_formulas(formula_code);`,
		`CREATE TABLE IF NOT EXISTS price_list_subgroup_formulas_map (
			id uuid PRIMARY KEY,
			price_list_subgroup_code text NOT NULL REFERENCES price_list_sub_group(subgroup_code),
			price_list_formulas_code text NOT NULL REFERENCES price_list_formulas(formula_code),
			is_default boolean DEFAULT false, create_dtm timestamp
		);`,
	}
	for _, s := range stmts {
		if err := gormx.Exec(s).Error; err != nil {
			return fmt.Errorf("%s: %w", s, err)
		}
	}
	return nil
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		t.Fatalf("db connect: %v", err)
	}
	t.Cleanup(func() { db.CloseGORM(gormx) })
	return gormx
}

func truncateAll(t *testing.T, gormx *gorm.DB) {
	t.Helper()
	err := gormx.Exec(`TRUNCATE price_list_subgroup_formulas_map, price_list_sub_group_key,
		price_list_sub_group, price_list_group_extra_key, price_list_group_extra,
		price_list_group_term, price_list_group_key, price_list_group_history,
		price_list_sub_group_history, price_list_group, price_list_formulas CASCADE`).Error
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func count(t *testing.T, gormx *gorm.DB, table string) int64 {
	t.Helper()
	var n int64
	if err := gormx.Table(table).Count(&n).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// fullSheets is one group with 2 terms, 3 extras (two sharing extra_key +
// condition_code) and 2 subgroups mapped to 2 master formulas.
func fullSheets() sheets {
	return sheets{
		"price_list_group": groupSheet(
			[]string{testCompany, testSite, "G1", "กลุ่ม 1", "THB", "2026-01-01", "18.5", "18.5", "PG01_1"},
		),
		"price_list_group_term": {
			{"company_code", "site_code", "group_code", "term_code", "pdc", "pdc_percent", "due", "due_percent"},
			{testCompany, testSite, "G1", "T1", "0.19", "1.0%", "0.28", "1.5%"},
			{testCompany, testSite, "G1", "T2", "0.37", "2.0%", "0.56", "3.0%"},
		},
		"price_list_group_extra": {
			{"company_code", "site_code", "group_code", "condition_code", "operator", "value_int", "cond_range_min", "cond_range_max", "PG01"},
			{testCompany, testSite, "G1", "PG06", "to", "0.1", "3.2", "4.5", "PG01_1"},
			{testCompany, testSite, "G1", "PG06", "to", "0.2", "5", "6", "PG01_1"},
			{testCompany, testSite, "G1", "PG06", ">=", "0.3", "6", "", "PG01_2"},
		},
		"price_list_sub_group": subGroupSheet(
			[]string{testCompany, testSite, "G1", "SG01", "PG01_1", "PG02_2"},
			[]string{testCompany, testSite, "G1", "SG02", "PG01_1", "PG02_3"},
		),
		"price_list_formulars": formulaSheet(
			[]string{"", "FM-1", "kg", "kg", "price_calc", "a/b", `{"required":["a"]}`, "2"},
			[]string{"", "FM-2", "pcs", "pcs", "price_calc", "a*b", "", "0"},
		),
		"formulas_map": {
			{"subgroup_code", "formula_code_default", "formula_code_convert"},
			{"SG01", "FM-1", "FM-2"},
			{"SG02", "FM-1", "FM-2"},
		},
	}
}

func uploadFullSheets(t *testing.T, gormx *gorm.DB, replaceAll bool) {
	t.Helper()
	req, err := buildCreatePricelistRequestFromExcel(buildXlsx(t, fullSheets()))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	req.ReplaceAll = replaceAll
	resp, err := CreatePricelist(gormx, *req)
	if err != nil {
		t.Fatalf("CreatePricelist: %v", err)
	}
	if resp.ResponseCode != 0 {
		t.Fatalf("CreatePricelist: %s", resp.Message)
	}
}

func TestCreatePricelist_LoadsEveryRow(t *testing.T) {
	gormx := openTestDB(t)
	truncateAll(t, gormx)
	uploadFullSheets(t, gormx, false)

	for table, want := range map[string]int64{
		"price_list_group":                 1,
		"price_list_group_term":            2,
		"price_list_group_extra":           3, // rows sharing extra_key + condition_code must survive
		"price_list_group_extra_key":       3,
		"price_list_sub_group":             2,
		"price_list_sub_group_key":         4,
		"price_list_formulas":              2,
		"price_list_subgroup_formulas_map": 4,
	} {
		if got := count(t, gormx, table); got != want {
			t.Errorf("%s = %d rows, want %d", table, got, want)
		}
	}

	var term struct{ PdcPercent, DuePercent float64 }
	if err := gormx.Table("price_list_group_term").
		Select("pdc_percent, due_percent").Where("term_code = ?", "T1").Scan(&term).Error; err != nil {
		t.Fatalf("scan term: %v", err)
	}
	if term.PdcPercent != 0.01 || term.DuePercent != 0.015 {
		t.Errorf("T1 percents = %+v, want 0.01 / 0.015", term)
	}

	var values []float64
	if err := gormx.Table("price_list_group_extra").
		Order("value_int").Pluck("value_int", &values).Error; err != nil {
		t.Fatalf("pluck value_int: %v", err)
	}
	if len(values) != 3 || values[0] != 0.1 || values[2] != 0.3 {
		t.Errorf("value_int = %v, want [0.1 0.2 0.3]", values)
	}
}

func TestCreatePricelist_ReuploadIsIdempotent(t *testing.T) {
	gormx := openTestDB(t)
	truncateAll(t, gormx)
	uploadFullSheets(t, gormx, false)

	before := map[string]int64{}
	tables := []string{"price_list_group", "price_list_group_term", "price_list_group_extra",
		"price_list_group_extra_key", "price_list_sub_group", "price_list_sub_group_key",
		"price_list_formulas", "price_list_subgroup_formulas_map"}
	for _, tb := range tables {
		before[tb] = count(t, gormx, tb)
	}

	uploadFullSheets(t, gormx, false)

	for _, tb := range tables {
		if got := count(t, gormx, tb); got != before[tb] {
			t.Errorf("%s = %d rows after re-upload, want %d", tb, got, before[tb])
		}
	}
}

func TestCreatePricelist_ReplaceAllWipesOnlyItsOwnScope(t *testing.T) {
	gormx := openTestDB(t)
	truncateAll(t, gormx)

	// Stale data: one group in the uploaded scope, one in another site.
	staleID, otherID := uuid.New(), uuid.New()
	now := time.Now()
	for _, g := range []struct {
		id            uuid.UUID
		site, code    string
		subGroupCode  string
		subGroupKeyID uuid.UUID
	}{
		{staleID, testSite, "OLD_GROUP", "OLD_SG", uuid.New()},
		{otherID, "SITE-2", "OTHER_GROUP", "OTHER_SG", uuid.New()},
	} {
		if err := gormx.Table("price_list_group").Create(map[string]any{
			"id": g.id, "company_code": testCompany, "site_code": g.site, "group_code": g.code,
			"create_dtm": now, "update_dtm": now,
		}).Error; err != nil {
			t.Fatalf("seed group: %v", err)
		}
		subID := uuid.New()
		if err := gormx.Table("price_list_sub_group").Create(map[string]any{
			"id": subID, "price_list_group_id": g.id, "subgroup_code": g.subGroupCode,
			"subgroup_key": "OLD", "create_dtm": now, "update_dtm": now,
		}).Error; err != nil {
			t.Fatalf("seed subgroup: %v", err)
		}
		if err := gormx.Table("price_list_sub_group_key").Create(map[string]any{
			"id": g.subGroupKeyID, "sub_group_id": subID, "seq": 1, "code": "PG01", "value": "OLD",
		}).Error; err != nil {
			t.Fatalf("seed subgroup key: %v", err)
		}
		if err := gormx.Table("price_list_group_term").Create(map[string]any{
			"id": uuid.New(), "price_list_group_id": g.id, "term_code": "OLD_T",
			"create_dtm": now, "update_dtm": now,
		}).Error; err != nil {
			t.Fatalf("seed term: %v", err)
		}
		// price_list_group_history has no ON DELETE CASCADE in production, so a
		// replace_all that forgets it fails on the final group delete.
		if err := gormx.Table("price_list_group_history").Create(map[string]any{
			"id": uuid.New(), "price_list_group_id": g.id, "group_code": g.code, "create_dtm": now,
		}).Error; err != nil {
			t.Fatalf("seed group history: %v", err)
		}
		if err := gormx.Table("price_list_sub_group_history").Create(map[string]any{
			"id": uuid.New(), "price_list_group_id": g.id, "subgroup_key": "OLD", "create_dtm": now,
		}).Error; err != nil {
			t.Fatalf("seed subgroup history: %v", err)
		}
	}

	uploadFullSheets(t, gormx, true)

	var staleGroups int64
	if err := gormx.Table("price_list_group").
		Where("group_code = ?", "OLD_GROUP").Count(&staleGroups).Error; err != nil {
		t.Fatalf("count stale: %v", err)
	}
	if staleGroups != 0 {
		t.Errorf("OLD_GROUP still present after replace_all")
	}

	var otherGroups int64
	if err := gormx.Table("price_list_group").
		Where("group_code = ?", "OTHER_GROUP").Count(&otherGroups).Error; err != nil {
		t.Fatalf("count other: %v", err)
	}
	if otherGroups != 1 {
		t.Errorf("OTHER_GROUP (site SITE-2) = %d, want 1 — replace_all must not cross sites", otherGroups)
	}

	// The uploaded content is fully present, and no stale children survive.
	if got := count(t, gormx, "price_list_group"); got != 2 { // G1 + OTHER_GROUP
		t.Errorf("price_list_group = %d, want 2", got)
	}
	if got := count(t, gormx, "price_list_sub_group"); got != 3 { // SG01, SG02 + OTHER_SG
		t.Errorf("price_list_sub_group = %d, want 3", got)
	}
	if got := count(t, gormx, "price_list_group_term"); got != 3 { // T1, T2 + OTHER's OLD_T
		t.Errorf("price_list_group_term = %d, want 3", got)
	}
	for _, tb := range []string{"price_list_group_history", "price_list_sub_group_history"} {
		var n int64
		if err := gormx.Table(tb).Where("price_list_group_id = ?", staleID).Count(&n).Error; err != nil {
			t.Fatalf("count %s: %v", tb, err)
		}
		if n != 0 {
			t.Errorf("%s still has %d rows for the wiped scope", tb, n)
		}
		if got := count(t, gormx, tb); got != 1 { // only SITE-2's row survives
			t.Errorf("%s = %d rows, want 1", tb, got)
		}
	}
}

// Two subgroups with different subgroup_code may share the same PG combination, so
// they generate the same subgroup_key. Ids used to be handed out per subgroup_key,
// which gave both rows one uuid and broke on price_list_sub_group_pkey.
func TestCreatePricelist_DuplicateSubGroupKeyKeepsBothRows(t *testing.T) {
	gormx := openTestDB(t)
	truncateAll(t, gormx)

	sh := fullSheets()
	sh["price_list_sub_group"] = subGroupSheet(
		[]string{testCompany, testSite, "G1", "SG01", "PG01_1", "PG02_2"},
		[]string{testCompany, testSite, "G1", "SG02", "PG01_1", "PG02_2"}, // same key as SG01
	)

	req, err := buildCreatePricelistRequestFromExcel(buildXlsx(t, sh))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	resp, err := CreatePricelist(gormx, *req)
	if err != nil {
		t.Fatalf("CreatePricelist: %v", err)
	}
	if resp.ResponseCode != 0 {
		t.Fatalf("CreatePricelist: %s", resp.Message)
	}

	if got := count(t, gormx, "price_list_sub_group"); got != 2 {
		t.Errorf("price_list_sub_group = %d rows, want 2", got)
	}

	var ids []uuid.UUID
	if err := gormx.Table("price_list_sub_group").Pluck("id", &ids).Error; err != nil {
		t.Fatalf("pluck id: %v", err)
	}
	if len(ids) == 2 && ids[0] == ids[1] {
		t.Errorf("both subgroups share id %s", ids[0])
	}
}
