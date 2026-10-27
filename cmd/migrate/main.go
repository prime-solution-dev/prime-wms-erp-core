package main

import (
	"flag"
	"fmt"
	"os"
	"prime-erp-core/internal/migration"

	"github.com/joho/godotenv"
)

func main() {
	// ไม่มี .env ได้ (CI ส่ง DB_URL มาทาง env) — ถ้าไม่มี URL จริง Execute จะ error ก่อนต่อ DB
	_ = godotenv.Load()

	var o migration.Options
	flag.StringVar(&o.Command, "command", "", "up, down, status, create, reset, fix")
	flag.StringVar(&o.Name, "name", "", "name for create")
	flag.StringVar(&o.Set, "set", "schema", "schema, or a folder under seeds/ (master, dev, Pacifica-uat, ...)")
	flag.StringVar(&o.Confirm, "confirm", "", "yes = allow reset and seed dev up/down")
	flag.Parse()

	if err := migration.Execute(o); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
