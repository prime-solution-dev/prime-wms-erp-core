//go:build integration

package purchaseRepository

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

// หนึ่งแถวต่อหนึ่งสถานะที่หน้า PO Overview เลือกได้ ค่า status_approve ของแถว
// TEMP/CANCELLED/COMPLETED ตั้งเป็น COMPLETED ไว้จงใจ เพื่อให้ test ล้มทันที
// ถ้า filter กลับไปแนบ status_approve = PENDING เข้ามาอีก
var seedRows = []struct {
	code          string
	status        string
	statusApprove string
	usedStatus    string
}{
	{"PO-DRAFT-01", "TEMP", "COMPLETED", ""},
	{"PO-CANCEL-01", "CANCELLED", "COMPLETED", ""},
	{"PO-COMPLETE-01", "COMPLETED", "COMPLETED", "COMPLETED"},
	{"PO-NEW-01", "PENDING", "PENDING", ""},
	{"PO-WAIT-01", "PENDING", "PROCESS", ""},
	{"PO-REVIEW-01", "PENDING", "REVIEW", ""},
	{"PO-REJECT-01", "PENDING", "REJECT", ""},
	{"PO-APPROVED-01", "PENDING", "COMPLETED", ""},
	{"PO-PARTIAL-01", "PENDING", "COMPLETED", "PARTIAL"},
}

func seed() error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	if err := gormx.AutoMigrate(&models.Purchase{}, &models.PurchaseItem{}); err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, row := range seedRows {
		id := uuid.New()
		purchase := models.Purchase{
			ID:            id,
			PurchaseCode:  row.code,
			PurchaseType:  "NORMAL",
			CompanyCode:   testCompanyCode,
			SiteCode:      testSiteCode,
			SupplierCode:  "SUP-01",
			SupplierName:  "Supplier One",
			Status:        row.status,
			StatusApprove: row.statusApprove,
			UsedStatus:    row.usedStatus,
			CreateDtm:     now,
			UpdateDtm:     now,
			PurchaseItems: []models.PurchaseItem{{
				ID:               uuid.New(),
				PurchaseID:       id,
				PurchaseItem:     row.code + "-001",
				ProductCode:      "P001",
				ProductDesc:      "Steel bar",
				ProductGroupCode: "PG01",
				ProductGroupName: "Steel",
				Status:           "PENDING",
				CreateDtm:        now,
				UpdateDtm:        now,
			}},
		}

		if err := gormx.Create(&purchase).Error; err != nil {
			return err
		}
	}

	return nil
}

func codesOf(list []models.Purchase) []string {
	codes := []string{}
	for _, purchase := range list {
		codes = append(codes, purchase.PurchaseCode)
	}
	return codes
}

type statusFilter struct {
	status        []string
	statusApprove []string
	usedStatus    []string
}

func query(t *testing.T, filter statusFilter) []models.Purchase {
	t.Helper()

	list, _, _, _, _, err := GetPurchaseList(
		nil, nil,
		filter.statusApprove,
		nil, false,
		filter.status,
		nil, nil, nil, nil,
		testCompanyCode, testSiteCode,
		1, 50,
		"", "", "", "", "", "", "",
		nil, nil,
		filter.usedStatus,
	)

	assert.NoError(t, err)
	return list
}

// สถานะที่ตัดสินจาก status อย่างเดียว ต้องเจอแม้ status_approve จะเป็นค่าอื่น
func TestGetPurchaseList_StatusOnlyFilters(t *testing.T) {
	cases := []struct {
		name   string
		status string
		want   string
	}{
		{"Draft", "TEMP", "PO-DRAFT-01"},
		{"Cancel", "CANCELLED", "PO-CANCEL-01"},
		{"Complete", "COMPLETED", "PO-COMPLETE-01"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			list := query(t, statusFilter{status: []string{c.status}})
			assert.Equal(t, []string{c.want}, codesOf(list))
		})
	}
}

// บั๊กเดิม: filter แนบ status_approve = PENDING มาด้วยทุกครั้ง แถว Draft/Cancel/
// Complete ที่ผ่านการอนุมัติแล้วจึงไม่เข้าเงื่อนไข และผลลัพธ์ว่างเปล่า
func TestGetPurchaseList_StatusAndApproveAreAndedTogether(t *testing.T) {
	list := query(t, statusFilter{
		status:        []string{"COMPLETED"},
		statusApprove: []string{"PENDING"},
	})

	assert.Empty(t, list)
}

func TestGetPurchaseList_StatusWithApproveFilters(t *testing.T) {
	cases := []struct {
		name    string
		approve string
		want    []string
	}{
		{"New", "PENDING", []string{"PO-NEW-01"}},
		{"WaitForApprove", "PROCESS", []string{"PO-WAIT-01"}},
		{"Review", "REVIEW", []string{"PO-REVIEW-01"}},
		{"Reject", "REJECT", []string{"PO-REJECT-01"}},
		// Approved กับ Partial ใช้ status_approve เดียวกัน แยกกันด้วย used_status
		{"Approved", "COMPLETED", []string{"PO-APPROVED-01", "PO-PARTIAL-01"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			list := query(t, statusFilter{
				status:        []string{"PENDING"},
				statusApprove: []string{c.approve},
			})
			assert.ElementsMatch(t, c.want, codesOf(list))
		})
	}
}

func TestGetPurchaseList_UsedStatusFilter(t *testing.T) {
	list := query(t, statusFilter{
		status:     []string{"PENDING"},
		usedStatus: []string{"PARTIAL"},
	})

	assert.Equal(t, []string{"PO-PARTIAL-01"}, codesOf(list))
}

func TestGetPurchaseList_EmptyUsedStatusIsNotAFilter(t *testing.T) {
	list := query(t, statusFilter{status: []string{"PENDING"}})

	assert.Len(t, list, 6)
}

func TestGetPurchaseList_NoStatusFilterReturnsEverything(t *testing.T) {
	list := query(t, statusFilter{})

	assert.Len(t, list, len(seedRows))
}
