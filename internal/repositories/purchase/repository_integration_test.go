//go:build integration

package purchaseRepository

import (
	"context"
	"encoding/json"
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
	"gorm.io/gorm"
)

const (
	testCompanyCode  = "C001"
	testSiteCode     = "S001"
	testDateSiteCode = "S002"
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
	// PO เก่าที่ used_status เป็น NULL ไม่ใช่ค่าว่าง — ตัวกรอง Approved ต้องยังเห็น
	{"PO-APPROVED-NULL-01", "PENDING", "COMPLETED", ""},
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

	// used_status ของแถวนี้ต้องเป็น NULL จริง ๆ GORM เขียนสตริงว่างให้เสมอ
	if err := gormx.Model(&models.Purchase{}).
		Where("purchase_code = ?", "PO-APPROVED-NULL-01").
		Update("used_status", nil).Error; err != nil {
		return err
	}

	return seedCreateDateRows(gormx)
}

// แถวสำหรับทดสอบช่วง create date อยู่คนละ site เพื่อไม่ให้กระทบ test ชุดสถานะ
var createDateRows = []struct {
	code      string
	createDtm time.Time
}{
	{"PO-DATE-0902", time.Date(2025, 9, 2, 9, 0, 0, 0, time.UTC)},
	{"PO-DATE-0903", time.Date(2025, 9, 3, 14, 0, 0, 0, time.UTC)},
	{"PO-DATE-0904", time.Date(2025, 9, 4, 9, 0, 0, 0, time.UTC)},
}

func seedCreateDateRows(gormx *gorm.DB) error {
	for _, row := range createDateRows {
		purchase := models.Purchase{
			ID:            uuid.New(),
			PurchaseCode:  row.code,
			PurchaseType:  "NORMAL",
			CompanyCode:   testCompanyCode,
			SiteCode:      testDateSiteCode,
			SupplierCode:  "SUP-01",
			SupplierName:  "Supplier One",
			Status:        "PENDING",
			StatusApprove: "PENDING",
			CreateDtm:     row.createDtm,
			UpdateDtm:     row.createDtm,
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
	usedStatusNot []string
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
		filter.usedStatusNot,
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
		{"Approved", "COMPLETED", []string{"PO-APPROVED-01", "PO-PARTIAL-01", "PO-APPROVED-NULL-01"}},
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

// ตัวกรอง Approved ต้องไม่กิน PO ที่รับของบางส่วนแล้ว เพราะหน้าจอตัดสิน Partial
// จาก used_status ก่อน status_approve — สองสถานะนี้ใช้ status_approve เดียวกัน
func TestGetPurchaseList_UsedStatusNotExcludesPartial(t *testing.T) {
	list := query(t, statusFilter{
		status:        []string{"PENDING"},
		statusApprove: []string{"COMPLETED"},
		usedStatusNot: []string{"PARTIAL"},
	})

	assert.ElementsMatch(t, []string{"PO-APPROVED-01", "PO-APPROVED-NULL-01"}, codesOf(list))
}

func TestGetPurchaseList_EmptyUsedStatusIsNotAFilter(t *testing.T) {
	list := query(t, statusFilter{status: []string{"PENDING"}})

	assert.Len(t, list, 7)
}

func TestGetPurchaseList_NoStatusFilterReturnsEverything(t *testing.T) {
	list := query(t, statusFilter{})

	assert.Len(t, list, len(seedRows))
}

// ปลายช่วงต้องนับทั้งวัน PO ที่สร้างตอนบ่ายของวันสุดท้ายต้องไม่หลุด
func TestGetPurchaseList_CreateDateRangeIncludesLastDay(t *testing.T) {
	start := time.Date(2025, 9, 2, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 9, 3, 23, 59, 59, 0, time.UTC)

	list, total, _, _, _, err := GetPurchaseList(
		nil, nil, nil, nil, false, nil, nil, nil, nil, nil,
		testCompanyCode, testDateSiteCode,
		1, 50,
		"", "", "", "", "", "", "",
		&start, &end,
		nil, nil,
	)

	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.ElementsMatch(t, []string{"PO-DATE-0902", "PO-DATE-0903"}, codesOf(list))
}

func TestPurchaseNotePersistence(t *testing.T) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		t.Fatal(err)
	}
	defer db.CloseGORM(gormx)

	cases := []struct {
		name       string
		payload    string
		wantCreate string
		wantUpdate string
	}{
		{"text", `{"note":"  หมายเหตุ\nบรรทัดสอง  "}`, "  หมายเหตุ\nบรรทัดสอง  ", "  หมายเหตุ\nบรรทัดสอง  "},
		{"empty", `{"note":""}`, "", ""},
		{"omitted", `{}`, "", "existing note"},
		{"null", `{"note":null}`, "", "existing note"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var req models.PurchaseFormRequest
			if err := json.Unmarshal([]byte(tt.payload), &req); err != nil {
				t.Fatal(err)
			}
			created := models.Purchase{
				ID: uuid.New(), PurchaseCode: "NOTE-" + uuid.NewString(),
				CompanyCode: testCompanyCode, SiteCode: "NOTE-TEST",
				Note: req.Note, Remark: "original remark",
			}
			if err := CreatePurchase(gormx, []models.Purchase{created}); err != nil {
				t.Fatal(err)
			}
			defer gormx.Where("id = ?", created.ID).Delete(&models.Purchase{})

			var got models.Purchase
			if err := gormx.First(&got, "id = ?", created.ID).Error; err != nil {
				t.Fatal(err)
			}
			if got.Note == nil || *got.Note != tt.wantCreate {
				t.Fatalf("created Note = %v, want %q", got.Note, tt.wantCreate)
			}
			if got.Remark != "original remark" {
				t.Fatalf("created Remark = %q", got.Remark)
			}

			if err := gormx.Model(&models.Purchase{}).Where("id = ?", created.ID).Update("note", "existing note").Error; err != nil {
				t.Fatal(err)
			}
			update := models.Purchase{ID: created.ID, Note: req.Note}
			if err := UpdatePurchase([]models.Purchase{update}); err != nil {
				t.Fatal(err)
			}
			got = models.Purchase{}
			if err := gormx.First(&got, "id = ?", created.ID).Error; err != nil {
				t.Fatal(err)
			}
			if got.Note == nil || *got.Note != tt.wantUpdate {
				t.Fatalf("updated Note = %v, want %q", got.Note, tt.wantUpdate)
			}
			if got.Remark != "original remark" {
				t.Fatalf("updated Remark = %q", got.Remark)
			}
		})
	}
}
