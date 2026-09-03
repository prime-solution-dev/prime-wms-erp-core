//go:build integration

package prePurchaseRepository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testCompanyCode = "C001"
	testSiteCode    = "S001"
)

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

	if err := seed(); err != nil {
		fmt.Printf("failed to seed: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	code := m.Run()

	_ = container.Terminate(ctx)
	os.Exit(code)
}

func seed() error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	if err := gormx.AutoMigrate(&models.PrePurchase{}, &models.PrePurchaseItem{}); err != nil {
		return err
	}

	now := time.Now().UTC()
	rows := []struct {
		code          string
		supplierCode  string
		hierarchyCode string
	}{
		{"PB202609-0012", "SUP-STEEL-01", "PG01_1"},
		{"PB202609-0013", "SUP-STEEL-02", "PG01_2"},
		{"PB202510-0099", "VENDOR-77", "PG02_9"},
	}

	for _, row := range rows {
		id := uuid.New()
		prePurchase := models.PrePurchase{
			ID:              id,
			PrePurchaseCode: row.code,
			CompanyCode:     testCompanyCode,
			SiteCode:        testSiteCode,
			SupplierCode:    row.supplierCode,
			Status:          "PENDING",
			StatusApprove:   "COMPLETED",
			CreateDtm:       now,
			UpdateDtm:       now,
			PrePurchaseItems: []models.PrePurchaseItem{{
				ID:            uuid.New(),
				PrePurchaseID: id,
				PreItem:       row.code + "-001",
				HierarchyCode: row.hierarchyCode,
				Status:        "PENDING",
				CreateDtm:     now,
				UpdateDtm:     now,
			}},
		}

		if err := gormx.Create(&prePurchase).Error; err != nil {
			return err
		}
	}

	return nil
}

func codesOf(list []models.PrePurchase) []string {
	codes := []string{}
	for _, prePurchase := range list {
		codes = append(codes, prePurchase.PrePurchaseCode)
	}
	return codes
}

func baseRequest() models.GetPOBigLotListRequest {
	return models.GetPOBigLotListRequest{
		CompanyCode: testCompanyCode,
		SiteCode:    testSiteCode,
		Page:        1,
		PageSize:    50,
	}
}

func TestGetPOBigLotList_PrePurchaseCodeLike(t *testing.T) {
	req := baseRequest()
	req.PrePurchaseCodeLike = "202609"

	list, total, _, _, _, err := GetPOBigLotList(req)

	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.ElementsMatch(t, []string{"PB202609-0012", "PB202609-0013"}, codesOf(list))
}

func TestGetPOBigLotList_PrePurchaseCodeLikeIsCaseInsensitive(t *testing.T) {
	req := baseRequest()
	req.PrePurchaseCodeLike = "pb202510"

	list, _, _, _, _, err := GetPOBigLotList(req)

	assert.NoError(t, err)
	assert.Equal(t, []string{"PB202510-0099"}, codesOf(list))
}

func TestGetPOBigLotList_PrePurchaseCodeLikeNoMatch(t *testing.T) {
	req := baseRequest()
	req.PrePurchaseCodeLike = "ไม่มีอยู่จริง"

	list, total, _, _, _, err := GetPOBigLotList(req)

	assert.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, list)
}

func TestGetPOBigLotList_SupplierCodeLike(t *testing.T) {
	req := baseRequest()
	req.SupplierCodeLike = "STEEL"

	list, _, _, _, _, err := GetPOBigLotList(req)

	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"PB202609-0012", "PB202609-0013"}, codesOf(list))
}

func TestGetPOBigLotList_ProductGroupCodeLike(t *testing.T) {
	req := baseRequest()
	req.ProductGroupCodeLike = "PG01"

	list, _, _, _, _, err := GetPOBigLotList(req)

	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"PB202609-0012", "PB202609-0013"}, codesOf(list))
}

func TestGetPOBigLotList_LikeFiltersAreCombinedWithAnd(t *testing.T) {
	req := baseRequest()
	req.PrePurchaseCodeLike = "202609"
	req.SupplierCodeLike = "STEEL-02"

	list, _, _, _, _, err := GetPOBigLotList(req)

	assert.NoError(t, err)
	assert.Equal(t, []string{"PB202609-0013"}, codesOf(list))
}

// regression: การค้นหาแบบเป๊ะที่หน้าอื่นใช้อยู่ต้องไม่เปลี่ยนพฤติกรรม
func TestGetPOBigLotList_ExactCodesStillExact(t *testing.T) {
	req := baseRequest()
	req.PrePurchaseCodes = []string{"PB202609-0012"}

	list, _, _, _, _, err := GetPOBigLotList(req)

	assert.NoError(t, err)
	assert.Equal(t, []string{"PB202609-0012"}, codesOf(list))

	req.PrePurchaseCodes = []string{"202609"}
	list, _, _, _, _, err = GetPOBigLotList(req)

	assert.NoError(t, err)
	assert.Empty(t, list)
}

// regression: ไม่ส่งเงื่อนไขค้นหา ต้องได้ทุกแถวของ site
func TestGetPOBigLotList_NoFilterReturnsAll(t *testing.T) {
	list, total, _, _, _, err := GetPOBigLotList(baseRequest())

	assert.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, list, 3)
}
