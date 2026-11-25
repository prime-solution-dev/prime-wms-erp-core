//go:build integration

package quotationService

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/migration"
	"prime-erp-core/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestMain เปิด Postgres แล้วรัน migrations ทั้งหมด (baseline + ไฟล์ใหม่) ให้ schema ตรงกับ DB จริง
func TestMain(m *testing.M) {
	ctx := context.Background()
	c, err := postgres.Run(ctx, "postgres:15-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(wait.ForAll(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second),
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second),
		)),
	)
	if err != nil {
		fmt.Printf("start postgres: %v\n", err)
		os.Exit(1)
	}
	url, err := c.ConnectionString(ctx, "sslmode=disable")
	if err == nil {
		os.Setenv("database_sqlx_url_prime_erp", url)
		os.Setenv("database_gorm_url_prime_erp", url)
		err = migration.Execute(migration.Options{Command: "up", Set: "schema"})
	}
	if err != nil {
		fmt.Printf("prepare schema: %v\n", err)
		_ = testcontainers.TerminateContainer(c)
		os.Exit(1)
	}

	code := m.Run()
	_ = testcontainers.TerminateContainer(c)
	os.Exit(code)
}

// TMI issue log #40/#42: ผู้ติดต่อที่กรอกบนใบเสนอราคาต้องบันทึกได้ อ่านกลับผ่าน GetQuotation
// (document-core ใช้ endpoint นี้พิมพ์ PDF) และลบออกได้ผ่าน UpdateQuotation
func TestIntegration_QuotationContactPersonRoundTrip(t *testing.T) {
	gormx, err := db.ConnectGORM("prime_erp")
	require.NoError(t, err)
	defer db.CloseGORM(gormx)

	id := uuid.New()
	code := "QU-CONTACT-" + id.String()[:8]
	require.NoError(t, gormx.Create(&models.Quotation{ID: id, QuotationCode: code, ContactPerson: "คุณสมชาย"}).Error)

	got := func() string {
		res, err := GetQuotation(context.Background(), fmt.Sprintf(`{"quotation_code":[%q]}`, code))
		require.NoError(t, err)
		quotations := res.(ResultQuotationResponse).Quotations
		require.Len(t, quotations, 1)
		return quotations[0].ContactPerson
	}
	require.Equal(t, "คุณสมชาย", got())

	require.NoError(t, updateQuotationClearableFields(gormx, models.Quotation{ID: id}))
	require.Equal(t, "", got(), "ลบผู้ติดต่อแล้วต้องเป็นค่าว่าง")
}
