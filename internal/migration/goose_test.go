package migration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveSet(t *testing.T) {
	cases := []struct {
		name    string
		want    Set
		wantErr string
	}{
		{"schema", Set{Dir: "migrations", Table: "goose_db_version"}, ""},
		{"master", Set{Dir: filepath.Join("seeds", "master"), Table: "goose_seed_master_version"}, ""},
		{"dev", Set{Dir: filepath.Join("seeds", "dev"), Table: "goose_seed_dev_version"}, ""},
		{"Pacifica-uat", Set{Dir: filepath.Join("seeds", "Pacifica-uat"), Table: "goose_seed_pacifica_uat_version"}, ""},
		{"shi_sit", Set{Dir: filepath.Join("seeds", "shi_sit"), Table: "goose_seed_shi_sit_version"}, ""},
		{"", Set{}, "invalid set"},
		{"../x", Set{}, "invalid set"},
		{"a/b", Set{}, "invalid set"},
		{"--", Set{}, "invalid set"},
		{strings.Repeat("a", 45), Set{}, "invalid set"},
		{strings.Repeat("a", 44), Set{Dir: filepath.Join("seeds", strings.Repeat("a", 44)), Table: "goose_seed_" + strings.Repeat("a", 44) + "_version"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveSet(tc.name)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestValidate(t *testing.T) {
	schema := Set{Dir: "migrations", Table: "goose_db_version"}
	master := Set{Dir: filepath.Join("seeds", "master"), Table: "goose_seed_master_version"}
	dev := Set{Dir: filepath.Join("seeds", "dev"), Table: "goose_seed_dev_version"}
	client := Set{Dir: filepath.Join("seeds", "Pacifica-uat"), Table: "goose_seed_pacifica_uat_version"}
	cases := []struct {
		name    string
		opts    Options
		want    Set
		wantErr string
	}{
		{"schema up", Options{Command: "up", Set: "schema"}, schema, ""},
		{"master status", Options{Command: "status", Set: "master"}, master, ""},
		{"client folder up needs no confirm", Options{Command: "up", Set: "Pacifica-uat"}, client, ""},
		{"dev up confirmed", Options{Command: "up", Set: "dev", Confirm: "yes"}, dev, ""},
		{"dev status needs no confirm", Options{Command: "status", Set: "dev"}, dev, ""},
		{"dev create needs no confirm", Options{Command: "create", Set: "dev", Name: "x"}, dev, ""},
		{"reset confirmed", Options{Command: "reset", Set: "schema", Confirm: "yes"}, schema, ""},
		{"invalid set", Options{Command: "up", Set: "../prod"}, Set{}, "invalid set"},
		{"unknown command", Options{Command: "drop", Set: "schema"}, Set{}, "unknown command"},
		{"create without name", Options{Command: "create", Set: "schema"}, Set{}, "NAME is required"},
		{"reset without confirm", Options{Command: "reset", Set: "schema"}, Set{}, "requires CONFIRM=yes"},
		{"dev up without confirm", Options{Command: "up", Set: "dev"}, Set{}, "requires CONFIRM=yes"},
		{"dev down wrong confirm", Options{Command: "down", Set: "dev", Confirm: "y"}, Set{}, "requires CONFIRM=yes"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Validate(tc.opts)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

// newProjectRoot สร้าง root ปลอม (มี go.mod) พร้อมโฟลเดอร์ที่ระบุ แล้ว chdir เข้าไป
func newProjectRoot(t *testing.T, dirs ...string) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644))
	for _, d := range dirs {
		require.NoError(t, os.MkdirAll(filepath.Join(root, d), 0o755))
	}
	t.Chdir(root)
	return root
}

// samePath เทียบ path หลัง resolve symlink (macOS: /var → /private/var)
func samePath(t *testing.T, want, got string) {
	t.Helper()
	w, err := filepath.EvalSymlinks(want)
	require.NoError(t, err)
	g, err := filepath.EvalSymlinks(got)
	require.NoError(t, err)
	require.Equal(t, w, g)
}

func TestFindDir_WalksUpToGoMod(t *testing.T) {
	root := newProjectRoot(t, "migrations", filepath.Join("sub", "deeper"))
	t.Chdir(filepath.Join(root, "sub", "deeper"))

	got, err := FindDir("migrations")
	require.NoError(t, err)
	samePath(t, filepath.Join(root, "migrations"), got)
}

func TestFindDir_MissingDir(t *testing.T) {
	root := newProjectRoot(t)
	got, err := FindDir(filepath.Join("seeds", "dev"))
	require.ErrorIs(t, err, errDirNotFound)
	// path ยังไม่มีจริง → EvalSymlinks เฉพาะ root แล้วต่อ suffix
	resolved, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(resolved, "seeds", "dev"), got)
}

func TestFindDir_NoGoMod(t *testing.T) {
	t.Chdir(t.TempDir())
	_, err := FindDir("migrations")
	require.ErrorContains(t, err, "go.mod not found")
}

func TestExecute_ValidationRunsFirst(t *testing.T) {
	err := Execute(Options{Command: "up", Set: "dev"})
	require.ErrorContains(t, err, "requires CONFIRM=yes")
}

func TestExecute_EmptyURL(t *testing.T) {
	newProjectRoot(t, "migrations")
	t.Setenv(urlEnv, "")
	err := Execute(Options{Command: "up", Set: "schema"})
	require.ErrorContains(t, err, "database URL not set")
}

func TestExecute_CreateDoesNotNeedDB(t *testing.T) {
	root := newProjectRoot(t, filepath.Join("seeds", "dev"))
	t.Setenv(urlEnv, "")

	require.NoError(t, Execute(Options{Command: "create", Set: "dev", Name: "add_demo"}))

	files, err := filepath.Glob(filepath.Join(root, "seeds", "dev", "*_add_demo.sql"))
	require.NoError(t, err)
	require.Len(t, files, 1)
}

func TestExecute_FixDoesNotNeedDB(t *testing.T) {
	root := newProjectRoot(t, "migrations")
	t.Setenv(urlEnv, "")
	f := filepath.Join(root, "migrations", "20260101000000_a.sql")
	require.NoError(t, os.WriteFile(f, []byte("-- +goose Up\nSELECT 1;\n\n-- +goose Down\nSELECT 1;\n"), 0o644))

	require.NoError(t, Execute(Options{Command: "fix", Set: "schema"}))

	require.NoFileExists(t, f)
	require.FileExists(t, filepath.Join(root, "migrations", "00001_a.sql"))
}

func TestExecute_MissingSeedFolderIsNoop(t *testing.T) {
	root := newProjectRoot(t, "migrations")
	t.Setenv(urlEnv, "")
	require.NoError(t, Execute(Options{Command: "up", Set: "Pacifica-uat"}))
	require.NoDirExists(t, filepath.Join(root, "seeds", "Pacifica-uat"))
}

func TestExecute_MissingSeedFolderOnlyUpSkips(t *testing.T) {
	newProjectRoot(t, "migrations")
	t.Setenv(urlEnv, "")
	for _, cmd := range []string{"status", "down", "fix"} {
		require.ErrorIs(t, Execute(Options{Command: cmd, Set: "Pacifica-uat"}), errDirNotFound, cmd)
	}
}

func TestExecute_MissingSchemaFolderIsError(t *testing.T) {
	newProjectRoot(t)
	t.Setenv(urlEnv, "")
	require.ErrorContains(t, Execute(Options{Command: "up", Set: "schema"}), "directory not found")
}

func TestExecute_CreateMakesSeedFolder(t *testing.T) {
	root := newProjectRoot(t)
	t.Setenv(urlEnv, "")

	require.NoError(t, Execute(Options{Command: "create", Set: "Pacifica-uat", Name: "add_demo"}))

	files, err := filepath.Glob(filepath.Join(root, "seeds", "Pacifica-uat", "*_add_demo.sql"))
	require.NoError(t, err)
	require.Len(t, files, 1)
}
