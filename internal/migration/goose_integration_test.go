package migration

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const pgImage = "postgres:15-alpine" // ให้ตรงกับ major version ของ DB จริง

// startPostgres เปิด Postgres ชั่วคราวแล้วตั้ง env URL ให้ Execute ใช้
func startPostgres(t *testing.T) *sql.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test: needs docker")
	}
	ctx := context.Background()
	c, err := postgres.Run(ctx, pgImage,
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(wait.ForAll(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(30*time.Second),
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(30*time.Second),
		)),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(c) })

	url, err := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	t.Setenv(urlEnv, url)

	db, err := sql.Open("postgres", url)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func count(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow(query, args...).Scan(&n))
	return n
}

func TestIntegration_FreshDB_SchemaThenSeeds(t *testing.T) {
	db := startPostgres(t)

	require.NoError(t, Execute(Options{Command: "up", Set: "schema"}))
	require.NoError(t, Execute(Options{Command: "up", Set: "master"}))
	require.NoError(t, Execute(Options{Command: "up", Set: "dev", Confirm: "yes"}))
	require.NoError(t, Execute(Options{Command: "status", Set: "schema"}))

	dir, err := FindDir("migrations")
	require.NoError(t, err)
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	require.NoError(t, err)
	require.Equal(t, len(files), count(t, db, `SELECT count(*) FROM goose_db_version WHERE version_id > 0`))
	require.Positive(t, count(t, db, `SELECT count(*) FROM information_schema.tables WHERE table_name = 'approval'`))
}

func TestIntegration_BaselineIsIdempotentOnExistingDB(t *testing.T) {
	db := startPostgres(t)
	require.NoError(t, Execute(Options{Command: "up", Set: "schema"}))
	cols := `SELECT count(*) FROM information_schema.columns WHERE table_schema = 'public'`
	idx := `SELECT count(*) FROM pg_indexes WHERE schemaname = 'public'`
	cons := `SELECT count(*) FROM pg_constraint c JOIN pg_namespace n ON n.oid = c.connamespace WHERE n.nspname = 'public'`
	before, beforeIdx, beforeCons := count(t, db, cols), count(t, db, idx), count(t, db, cons)

	// จำลอง DB เดิมที่มี schema ครบแต่ยังไม่มีประวัติ baseline
	_, err := db.Exec(`DELETE FROM goose_db_version WHERE version_id > 0`)
	require.NoError(t, err)

	require.NoError(t, Execute(Options{Command: "up", Set: "schema"}))
	require.Equal(t, before, count(t, db, cols))
	require.Equal(t, beforeIdx, count(t, db, idx))
	require.Equal(t, beforeCons, count(t, db, cons))
}

func TestIntegration_SeedUsesOwnVersionTable(t *testing.T) {
	db := startPostgres(t)
	root := newProjectRoot(t, filepath.Join("seeds", "master"))
	seed := "-- +goose Up\nCREATE TABLE demo (id int PRIMARY KEY);\nINSERT INTO demo VALUES (1) ON CONFLICT DO NOTHING;\n\n-- +goose Down\nDROP TABLE demo;\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "seeds", "master", "00001_demo.sql"), []byte(seed), 0o644))

	require.NoError(t, Execute(Options{Command: "up", Set: "master"}))

	require.Equal(t, 1, count(t, db, `SELECT count(*) FROM goose_seed_master_version WHERE version_id = 1`))
	require.Equal(t, 0, count(t, db, `SELECT count(*) FROM information_schema.tables WHERE table_name = 'goose_db_version'`))

	require.NoError(t, Execute(Options{Command: "down", Set: "master"}))
	require.Equal(t, 0, count(t, db, `SELECT count(*) FROM information_schema.tables WHERE table_name = 'demo'`))
}

func TestIntegration_EmptySeedDirIsNoop(t *testing.T) {
	startPostgres(t)
	newProjectRoot(t, filepath.Join("seeds", "master"))
	require.NoError(t, Execute(Options{Command: "up", Set: "master"}))
}

func TestIntegration_FolderSetUsesSlugTable(t *testing.T) {
	db := startPostgres(t)
	root := newProjectRoot(t, filepath.Join("seeds", "Pacifica-uat"))
	seed := "-- +goose Up\nCREATE TABLE demo_client (id int PRIMARY KEY);\n\n-- +goose Down\nDROP TABLE demo_client;\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "seeds", "Pacifica-uat", "00001_demo.sql"), []byte(seed), 0o644))

	require.NoError(t, Execute(Options{Command: "up", Set: "Pacifica-uat"}))

	require.Equal(t, 1, count(t, db, `SELECT count(*) FROM goose_seed_pacifica_uat_version WHERE version_id = 1`))
	require.Equal(t, 1, count(t, db, `SELECT count(*) FROM information_schema.tables WHERE table_name = 'demo_client'`))
}

const restoreKeysVersion = 20261120120000

// restoredIndexes คือ index ที่ 20261120120000_restore_missing_unique_keys.sql ต้องทำให้มี
var restoredIndexes = []string{
	"quotation_quotation_code_key", "sale_sale_code_key", "delivery_booking_delivery_code_key",
	"ux_system_config_running",
	"idx_price_list_sub_group_price_list_group_id", "idx_price_list_sub_group_key_sub_group_id",
}

func indexExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	return count(t, db, `SELECT count(*) FROM pg_indexes WHERE schemaname = 'public' AND indexname = $1`, name) == 1
}

// gooseTo ย้าย schema ไปที่ version ที่ระบุ (Execute ไม่มี up-to/down-to)
func gooseTo(t *testing.T, db *sql.DB, up bool, version int64) {
	t.Helper()
	dir, err := FindDir("migrations")
	require.NoError(t, err)
	require.NoError(t, goose.SetDialect("postgres"))
	goose.SetTableName("goose_db_version")
	if up {
		require.NoError(t, goose.UpTo(db, dir, version))
	} else {
		require.NoError(t, goose.DownTo(db, dir, version))
	}
}

func TestIntegration_RestoreKeys_FreshDBRejectsDuplicateCodes(t *testing.T) {
	db := startPostgres(t)
	require.NoError(t, Execute(Options{Command: "up", Set: "schema"}))
	for _, ix := range restoredIndexes {
		require.True(t, indexExists(t, db, ix), ix)
	}

	for _, q := range []string{
		`INSERT INTO quotation (id, quotation_code) VALUES (gen_random_uuid(), 'QU-1'), (gen_random_uuid(), 'QU-1')`,
		`INSERT INTO sale (id, sale_code) VALUES (gen_random_uuid(), 'SO-1'), (gen_random_uuid(), 'SO-1')`,
		`INSERT INTO delivery_booking (id, delivery_code) VALUES (gen_random_uuid(), 'DB-1'), (gen_random_uuid(), 'DB-1')`,
		`INSERT INTO system_config (topic_code, config_code) VALUES ('RUNNING', 'RUNNING_X'), ('RUNNING', 'RUNNING_X')`,
	} {
		_, err := db.Exec(q)
		require.ErrorContains(t, err, "duplicate key", q)
	}
	// unique เฉพาะ RUNNING: topic อื่นใช้ config_code ซ้ำได้เหมือนเดิม
	_, err := db.Exec(`INSERT INTO system_config (topic_code, config_code) VALUES ('OTHER', 'C1'), ('OTHER', 'C1')`)
	require.NoError(t, err)

	// down แล้ว up ซ้ำได้
	gooseTo(t, db, false, restoreKeysVersion-1)
	for _, ix := range restoredIndexes {
		require.False(t, indexExists(t, db, ix), ix)
	}
	_, err = db.Exec(`DELETE FROM system_config WHERE topic_code = 'OTHER'`)
	require.NoError(t, err)
	require.NoError(t, Execute(Options{Command: "up", Set: "schema"}))
	require.True(t, indexExists(t, db, "ux_system_config_running"))
}

func TestIntegration_RestoreKeys_DedupesSystemConfigAndKeepsPreexistingIndex(t *testing.T) {
	db := startPostgres(t)
	gooseTo(t, db, true, restoreKeysVersion-1)

	// จำลอง env เดิม: RUNNING ซ้ำ (คนละ period), config อื่นซ้ำทุกคอลัมน์, และ index ที่มีอยู่ก่อนแล้ว
	_, err := db.Exec(`
INSERT INTO system_config (topic_code, config_code, "json") VALUES
  ('RUNNING', 'RUNNING_PO', '{"year":"2026","month":"02","current_running":26}'),
  ('RUNNING', 'RUNNING_PO', '{"year":"2026","month":"03","current_running":1}'),
  ('RUNNING', 'RUNNING_SO', '{"year":"2026","month":"02","current_running":12}'),
  ('RUNNING', 'RUNNING_SO', '{"year":"2026","month":"02","current_running":12}'),
  ('RUNNING', 'RUNNING_QU', '{"year":"2026","month":"02","current_running":21}');
INSERT INTO system_config (topic_code, config_code, value) VALUES ('VAT', 'VAT', '7'), ('VAT', 'VAT', '7'), ('VAT', 'VAT', '10');
CREATE UNIQUE INDEX sale_sale_code_key ON public.sale (sale_code);`)
	require.NoError(t, err)

	require.NoError(t, Execute(Options{Command: "up", Set: "schema"}))

	require.Equal(t, 3, count(t, db, `SELECT count(*) FROM system_config WHERE topic_code = 'RUNNING'`))
	require.Equal(t, 1, count(t, db, `SELECT count(*) FROM system_config WHERE config_code = 'RUNNING_PO' AND "json"->>'month' = '03'`))
	require.Equal(t, 2, count(t, db, `SELECT count(*) FROM system_config WHERE topic_code = 'VAT'`))
	for _, ix := range restoredIndexes {
		require.True(t, indexExists(t, db, ix), ix)
	}

	// Down ถอดเฉพาะที่ไฟล์นี้สร้าง: sale_sale_code_key มีมาก่อน ต้องอยู่ต่อ
	gooseTo(t, db, false, restoreKeysVersion-1)
	require.True(t, indexExists(t, db, "sale_sale_code_key"))
	require.False(t, indexExists(t, db, "quotation_quotation_code_key"))
	require.False(t, indexExists(t, db, "ux_system_config_running"))
}

func TestIntegration_RestoreKeys_FailsOnDuplicateDocumentCode(t *testing.T) {
	db := startPostgres(t)
	gooseTo(t, db, true, restoreKeysVersion-1)
	_, err := db.Exec(`INSERT INTO quotation (id, quotation_code) VALUES (gen_random_uuid(), 'QU-1'), (gen_random_uuid(), 'QU-1')`)
	require.NoError(t, err)

	require.Error(t, Execute(Options{Command: "up", Set: "schema"}))
	// ล้มทั้งไฟล์: ไม่มีข้อมูลธุรกิจถูกลบ และไม่มี index ค้างครึ่งทาง
	require.Equal(t, 2, count(t, db, `SELECT count(*) FROM quotation`))
	require.False(t, indexExists(t, db, "ux_system_config_running"))
}
