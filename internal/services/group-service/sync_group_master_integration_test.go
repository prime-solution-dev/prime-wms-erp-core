//go:build integration

package groupService_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	groupService "prime-erp-core/internal/services/group-service"

	"github.com/google/uuid"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"
)

var groupTestDB *gorm.DB

func TestMain(m *testing.M) {
	ctx := context.Background()
	req := tc.ContainerRequest{
		Image:        "postgres:16",
		Env:          map[string]string{"POSTGRES_PASSWORD": "test", "POSTGRES_USER": "test", "POSTGRES_DB": "testdb"},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp").WithStartupTimeout(90 * time.Second),
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
	dsn := fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, mapped.Port())
	os.Setenv("database_gorm_url_prime_erp", dsn)

	groupTestDB, err = db.ConnectGORM("prime_erp")
	if err != nil {
		fmt.Printf("failed to connect gorm: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}
	if err := groupTestDB.AutoMigrate(&models.Group{}, &models.GroupItem{}); err != nil {
		fmt.Printf("failed to migrate: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	code := m.Run()
	_ = container.Terminate(ctx)
	os.Exit(code)
}

func seedExistingGroup(t *testing.T) uuid.UUID {
	t.Helper()

	if err := groupTestDB.Exec(`DELETE FROM "group_item"`).Error; err != nil {
		t.Fatalf("clean group_item: %v", err)
	}
	if err := groupTestDB.Exec(`DELETE FROM "group"`).Error; err != nil {
		t.Fatalf("clean group: %v", err)
	}

	staleID := uuid.New()
	if err := groupTestDB.Omit("GroupItems").Create(&models.Group{
		ID: staleID, GroupCode: "STALE", GroupName: "Stale group", Seq: 9,
		CreateDtm: time.Now(), UpdateDtm: time.Now(), CreateBy: "OLD", UpdateBy: "OLD",
	}).Error; err != nil {
		t.Fatalf("seed stale group: %v", err)
	}
	if err := groupTestDB.Create(&models.GroupItem{
		ID: uuid.New(), GroupID: staleID, ItemCode: "STALE01", ItemName: "Stale item",
		CreateDtm: time.Now(), UpdateDtm: time.Now(), CreateBy: "OLD", UpdateBy: "OLD",
	}).Error; err != nil {
		t.Fatalf("seed stale item: %v", err)
	}
	return staleID
}

// The snapshot replaces both tables wholesale, keeping the ids WMS sent.
func TestSyncGroupMasterReplacesBothTables(t *testing.T) {
	staleID := seedExistingGroup(t)

	groupID := uuid.New()
	itemID := uuid.New()
	now := time.Now()
	parentCode := "PG00"
	parentItemCode := "ROOT"

	got, err := groupService.SyncGroupMasterWithDB(groupTestDB, groupService.SyncGroupMasterRequest{
		Groups: []models.Group{{
			ID: groupID, GroupCode: "PG01", GroupName: "Product Group", Value: "PG", ValueInt: 2, Seq: 1,
			CreateDtm: now, UpdateDtm: now, CreateBy: "SYSTEM", UpdateBy: "SYSTEM",
		}},
		GroupItems: []models.GroupItem{{
			ID: itemID, GroupID: groupID, ItemCode: "BU01", ItemName: "Bulk", Value: "BULK", ValueInt: 3,
			ParentGroupCode: &parentCode, ParentGroupItemCode: &parentItemCode,
			CreateDtm: now, UpdateDtm: now, CreateBy: "SYSTEM", UpdateBy: "SYSTEM",
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.GroupsSynced != 1 || got.GroupItemsSynced != 1 {
		t.Errorf("counts = %d/%d, want 1/1", got.GroupsSynced, got.GroupItemsSynced)
	}

	var groups []models.Group
	if err := groupTestDB.Find(&groups).Error; err != nil {
		t.Fatalf("read groups: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("want exactly the snapshot's group, got %d", len(groups))
	}
	if groups[0].ID != groupID {
		t.Errorf("id = %s, want the id WMS assigned (%s)", groups[0].ID, groupID)
	}
	if groups[0].GroupCode != "PG01" || groups[0].ValueInt != 2 || groups[0].Seq != 1 {
		t.Errorf("group fields did not survive: %+v", groups[0])
	}
	if groups[0].ID == staleID {
		t.Error("the stale group must be gone")
	}

	var items []models.GroupItem
	if err := groupTestDB.Find(&items).Error; err != nil {
		t.Fatalf("read items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("want exactly the snapshot's item, got %d", len(items))
	}
	if items[0].ID != itemID || items[0].GroupID != groupID {
		t.Errorf("item ids did not survive: %+v", items[0])
	}
	if items[0].ItemCode != "BU01" || items[0].ValueInt != 3 {
		t.Errorf("item fields did not survive: %+v", items[0])
	}
	// The parent link is a WMS-side field ERP mirrors; dropping it silently would break any
	// hierarchy the ERP side later reads.
	if items[0].ParentGroupCode == nil || *items[0].ParentGroupCode != "PG00" {
		t.Errorf("parent_group_code did not survive: %+v", items[0].ParentGroupCode)
	}
	if items[0].ParentGroupItemCode == nil || *items[0].ParentGroupItemCode != "ROOT" {
		t.Errorf("parent_group_item_code did not survive: %+v", items[0].ParentGroupItemCode)
	}
}

// Duplicate ids inside the snapshot must roll the whole thing back, not leave the tables empty.
func TestSyncGroupMasterRollsBackOnBadData(t *testing.T) {
	seedExistingGroup(t)

	duplicateID := uuid.New()
	groupID := uuid.New()
	now := time.Now()

	_, err := groupService.SyncGroupMasterWithDB(groupTestDB, groupService.SyncGroupMasterRequest{
		Groups: []models.Group{{
			ID: groupID, GroupCode: "PG01", GroupName: "Product Group", Seq: 1,
			CreateDtm: now, UpdateDtm: now, CreateBy: "SYSTEM", UpdateBy: "SYSTEM",
		}},
		GroupItems: []models.GroupItem{
			{ID: duplicateID, GroupID: groupID, ItemCode: "BU01", ItemName: "Bulk", CreateDtm: now, UpdateDtm: now},
			{ID: duplicateID, GroupID: groupID, ItemCode: "BU02", ItemName: "Bulk 2", CreateDtm: now, UpdateDtm: now},
		},
	})
	if err == nil {
		t.Fatal("duplicate primary keys must fail the sync")
	}

	var groupCount int64
	if err := groupTestDB.Model(&models.Group{}).Count(&groupCount).Error; err != nil {
		t.Fatalf("count groups: %v", err)
	}
	if groupCount != 1 {
		t.Errorf("the previous snapshot must survive a failed sync, count = %d", groupCount)
	}

	var itemCount int64
	if err := groupTestDB.Model(&models.GroupItem{}).Count(&itemCount).Error; err != nil {
		t.Fatalf("count items: %v", err)
	}
	if itemCount != 1 {
		t.Errorf("the previous items must survive a failed sync, count = %d", itemCount)
	}
}

// A snapshot with groups but no items is legitimate — every group simply has no items yet.
func TestSyncGroupMasterAcceptsGroupsWithoutItems(t *testing.T) {
	seedExistingGroup(t)

	now := time.Now()
	got, err := groupService.SyncGroupMasterWithDB(groupTestDB, groupService.SyncGroupMasterRequest{
		Groups: []models.Group{{
			ID: uuid.New(), GroupCode: "PG01", GroupName: "Product Group", Seq: 1,
			CreateDtm: now, UpdateDtm: now, CreateBy: "SYSTEM", UpdateBy: "SYSTEM",
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.GroupItemsSynced != 0 {
		t.Errorf("items synced = %d, want 0", got.GroupItemsSynced)
	}

	var itemCount int64
	if err := groupTestDB.Model(&models.GroupItem{}).Count(&itemCount).Error; err != nil {
		t.Fatalf("count items: %v", err)
	}
	if itemCount != 0 {
		t.Errorf("the stale items must be gone, count = %d", itemCount)
	}
}
