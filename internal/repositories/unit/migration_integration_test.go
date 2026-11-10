//go:build integration

package unitRepository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"
)

// migrationFile คือไฟล์ที่ทีมรันด้วยมือตอน deploy เทสต์นี้รันไฟล์เดียวกันจริงๆ
// เพื่อให้ SQL ที่ผิด syntax หรือชื่อคอลัมน์ผิดถูกจับได้ก่อนขึ้น production
const migrationFile = "../../../migrations/2026-09-09-unit-pcs-label.sql"

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

	host, err := container.Host(ctx)
	if err != nil {
		fmt.Printf("failed to get host: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	mapped, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		fmt.Printf("failed to get mapped port: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	os.Setenv("database_gorm_url_prime_erp", fmt.Sprintf(
		"postgres://test:test@%s:%s/testdb?sslmode=disable", host, mapped.Port()))

	code := m.Run()

	_ = container.Terminate(ctx)
	os.Exit(code)
}

// seedUnits สร้างสภาพเดียวกับ unit master บน UAT: unit_code 'PC' ชื่อ 'Pcs'
// และ uom_code 'PC' ชื่อ 'Pcs' โดยมีแถว KG ปนอยู่ด้วยเพื่อกันไม่ให้ migration
// ไปแตะแถวที่ไม่เกี่ยว
func seedUnits(t *testing.T, gormx *gorm.DB) {
	t.Helper()

	require.NoError(t, gormx.Exec(`DROP TABLE IF EXISTS unit_uom, unit_method, unit`).Error)
	require.NoError(t, gormx.AutoMigrate(&models.Unit{}, &models.UnitMethod{}, &models.UnitUom{}))

	poPiece := uuid.New()
	soPiece := uuid.New()
	poKg := uuid.New()

	units := []models.Unit{
		{ID: poPiece, Topic: "PO", UnitCode: "PC", UnitName: "Pcs"},
		{ID: soPiece, Topic: "SO", UnitCode: "PC", UnitName: "Pcs"},
		{ID: poKg, Topic: "PO", UnitCode: "KG", UnitName: "kg"},
	}
	require.NoError(t, gormx.Create(&units).Error)

	method := models.UnitMethod{ID: uuid.New(), UnitID: poPiece, MethodCode: "PC", MethodName: "Pcs"}
	require.NoError(t, gormx.Create(&method).Error)

	uoms := []models.UnitUom{
		{ID: uuid.New(), MethodID: method.ID, UomCode: "PC", UomName: "Pcs"},
		{ID: uuid.New(), MethodID: method.ID, UomCode: "KG", UomName: "kg"},
	}
	require.NoError(t, gormx.Create(&uoms).Error)
}

func runMigration(t *testing.T, gormx *gorm.DB) {
	t.Helper()

	path, err := filepath.Abs(migrationFile)
	require.NoError(t, err)

	sql, err := os.ReadFile(path)
	require.NoError(t, err, "อ่านไฟล์ migration ไม่ได้ ตรวจ path")

	require.NoError(t, gormx.Exec(string(sql)).Error, "migration รันไม่ผ่าน")
}

func TestUnitPcsLabelMigration(t *testing.T) {
	gormx, err := db.ConnectGORM("prime_erp")
	require.NoError(t, err)
	defer db.CloseGORM(gormx)

	seedUnits(t, gormx)
	runMigration(t, gormx)

	t.Run("unit_name ของหน่วยชิ้นเปลี่ยนเป็น PCS ทุก topic", func(t *testing.T) {
		var units []models.Unit
		require.NoError(t, gormx.Where("unit_code = ?", "PC").Order("topic").Find(&units).Error)

		assert.Len(t, units, 2)
		for _, u := range units {
			assert.Equal(t, "PCS", u.UnitName, "topic %s ยังไม่ถูกแก้", u.Topic)
		}
	})

	t.Run("uom_name ของหน่วยชิ้นเปลี่ยนเป็น PCS", func(t *testing.T) {
		var uoms []models.UnitUom
		require.NoError(t, gormx.Where("uom_code = ?", "PC").Find(&uoms).Error)

		assert.Len(t, uoms, 1)
		assert.Equal(t, "PCS", uoms[0].UomName)
	})

	t.Run("ไม่แตะแถว KG", func(t *testing.T) {
		var kg models.Unit
		require.NoError(t, gormx.Where("unit_code = ?", "KG").First(&kg).Error)
		assert.Equal(t, "kg", kg.UnitName)

		var kgUom models.UnitUom
		require.NoError(t, gormx.Where("uom_code = ?", "KG").First(&kgUom).Error)
		assert.Equal(t, "kg", kgUom.UomName)
	})

	t.Run("ไม่แตะ method_name ของ Purchase Method", func(t *testing.T) {
		var method models.UnitMethod
		require.NoError(t, gormx.Where("method_code = ?", "PC").First(&method).Error)
		assert.Equal(t, "Pcs", method.MethodName)
	})

	t.Run("รันซ้ำได้ ผลไม่เปลี่ยน", func(t *testing.T) {
		runMigration(t, gormx)

		var units []models.Unit
		require.NoError(t, gormx.Where("unit_code = ?", "PC").Find(&units).Error)
		assert.Len(t, units, 2)
		for _, u := range units {
			assert.Equal(t, "PCS", u.UnitName)
		}
	})
}
