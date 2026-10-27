package migration

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

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
