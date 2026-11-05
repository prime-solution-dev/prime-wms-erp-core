package migration

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

const (
	dbName = "prime_erp"
	urlEnv = "database_sqlx_url_" + dbName
)

// Set คือชุดไฟล์ goose หนึ่งกลุ่ม แต่ละกลุ่มมีตาราง version ของตัวเอง
// เพื่อไม่ให้ version ของ seed ไปปนกับ schema
type Set struct {
	Dir   string
	Table string
}

const schemaSet = "schema"

var (
	setNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	nonAlnum       = regexp.MustCompile(`[^a-z0-9]+`)
	errDirNotFound = errors.New("directory not found")
)

// ResolveSet: "schema" → migrations/ (goose_db_version)
// ชื่ออื่น → seeds/<name>/ ตาราง goose_seed_<slug>_version เช่น Pacifica-uat → goose_seed_pacifica_uat_version
// ponytail: ชื่อที่ slug ตรงกัน (Pacifica-uat กับ pacifica_uat) จะใช้ตาราง version ร่วมกัน — ตั้งชื่อ folder ตาม branch ก็ไม่ชน
func ResolveSet(name string) (Set, error) {
	if name == schemaSet {
		return Set{Dir: "migrations", Table: "goose_db_version"}, nil
	}
	slug := strings.Trim(nonAlnum.ReplaceAllString(strings.ToLower(name), "_"), "_")
	// 44 = 63 (ขีดจำกัดชื่อ identifier ของ Postgres) - len("goose_seed_") - len("_version")
	if !setNamePattern.MatchString(name) || slug == "" || len(slug) > 44 {
		return Set{}, fmt.Errorf("invalid set %q (use schema or a folder name under seeds/: letters, digits, _ or -)", name)
	}
	return Set{Dir: filepath.Join("seeds", name), Table: "goose_seed_" + slug + "_version"}, nil
}

var commands = map[string]bool{
	"up": true, "down": true, "status": true, "create": true, "reset": true, "fix": true,
}

// Options คือค่าจาก flag ของ cmd/migrate
type Options struct {
	Command string
	Name    string
	Set     string
	Confirm string
}

// Validate ตรวจ flag ก่อนแตะไฟล์หรือ DB แล้วคืน Set ที่เลือก
func Validate(o Options) (Set, error) {
	set, err := ResolveSet(o.Set)
	if err != nil {
		return Set{}, err
	}
	if !commands[o.Command] {
		return Set{}, fmt.Errorf("unknown command %q (valid: up, down, status, create, reset, fix)", o.Command)
	}
	if o.Command == "create" && o.Name == "" {
		return Set{}, fmt.Errorf("NAME is required: make <target> NAME=description")
	}
	guarded := o.Command == "reset" || (o.Set == "dev" && (o.Command == "up" || o.Command == "down"))
	if guarded && o.Confirm != "yes" {
		return Set{}, fmt.Errorf("%s on set %s requires CONFIRM=yes", o.Command, o.Set)
	}
	return set, nil
}

// GetDBConnection returns a database connection for goose migrations
func GetDBConnection() (*sql.DB, error) {
	databaseURL := os.Getenv(urlEnv)
	if databaseURL == "" {
		return nil, fmt.Errorf("database URL not set: pass DB_URL=... or provide .env (%s)", urlEnv)
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// FindDir หา rel (เช่น "migrations") ใต้ root ของโปรเจกต์ คือโฟลเดอร์แรกที่มี go.mod
// เมื่อไล่ขึ้นจาก cwd — ทำให้รันได้ทั้งจาก root (Makefile) และจากโฟลเดอร์ package (go test)
// ถ้าไม่พบโฟลเดอร์ จะคืน errDirNotFound พร้อม path ที่คาดไว้ด้วย (create ใช้สร้างโฟลเดอร์)
func FindDir(rel string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	for dir := wd; ; {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			p := filepath.Join(dir, rel)
			if _, err := os.Stat(p); errors.Is(err, fs.ErrNotExist) {
				return p, fmt.Errorf("%w: %s", errDirNotFound, p)
			} else if err != nil {
				return "", err
			}
			return p, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found above %s", wd)
		}
		dir = parent
	}
}

// Execute รันคำสั่ง goose บน Set ที่เลือก
// create/fix ไม่ต่อ DB (แก้แค่ชื่อไฟล์); โฟลเดอร์ที่ยังไม่มีไฟล์ถือว่าไม่มีอะไรต้องทำ (ให้ db-setup ผ่านตอน seeds ยังว่าง)
func Execute(o Options) error {
	set, err := Validate(o)
	if err != nil {
		return err
	}
	dir, err := FindDir(set.Dir)
	if errors.Is(err, errDirNotFound) && o.Set != schemaSet {
		// seed folder ของ branch ที่ยังไม่มี: up ข้ามได้, create สร้าง folder ให้, คำสั่งอื่นถือเป็น error
		switch o.Command {
		case "up":
			log.Printf("seed set %q: %s not found, skipping", o.Set, dir)
			return nil
		case "create":
			err = os.MkdirAll(dir, 0o755)
		}
	}
	if err != nil {
		return err
	}
	if o.Command == "create" {
		return goose.Create(nil, dir, o.Name, "sql")
	}
	if o.Command == "fix" {
		return goose.Fix(dir)
	}

	db, err := GetDBConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}
	goose.SetTableName(set.Table)

	err = goose.Run(o.Command, db, dir)
	if errors.Is(err, goose.ErrNoMigrationFiles) {
		log.Printf("no migration files in %s, nothing to do", dir)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to run goose command %s: %w", o.Command, err)
	}
	return nil
}
