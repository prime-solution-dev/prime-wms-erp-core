//go:build integration

package groupService_test

import (
	"context"
	"testing"
	"time"

	"prime-erp-core/internal/models"
	groupService "prime-erp-core/internal/services/group-service"

	"github.com/google/uuid"
)

// GetGroupMaster ต้องส่งสาย parent ออกไปด้วย หน้า price list ใช้กรองตัวเลือกตามแม่
func TestGetGroupReturnsParentFields(t *testing.T) {
	seedExistingGroup(t)

	now := time.Now()
	groupID := uuid.New()
	parentCode, parentItemCode := "PG02", "PG02_2"
	if err := groupTestDB.Omit("GroupItems").Create(&models.Group{
		ID: groupID, GroupCode: "PG01", GroupName: "Product Group", Seq: 1,
		CreateDtm: now, UpdateDtm: now, CreateBy: "SYSTEM", UpdateBy: "SYSTEM",
	}).Error; err != nil {
		t.Fatalf("seed group: %v", err)
	}
	for _, it := range []models.GroupItem{
		{ID: uuid.New(), GroupID: groupID, ItemCode: "CHILD", ItemName: "Child",
			ParentGroupCode: &parentCode, ParentGroupItemCode: &parentItemCode},
		{ID: uuid.New(), GroupID: groupID, ItemCode: "ROOT", ItemName: "Root"},
	} {
		it.CreateDtm, it.UpdateDtm, it.CreateBy, it.UpdateBy = now, now, "SYSTEM", "SYSTEM"
		if err := groupTestDB.Create(&it).Error; err != nil {
			t.Fatalf("seed item %s: %v", it.ItemCode, err)
		}
	}

	out, err := groupService.GetGroup(context.Background(), `{"group_codes":["PG01"]}`)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	res := out.([]models.GetGroupResponse)
	if len(res) != 1 || len(res[0].Items) != 2 {
		t.Fatalf("want 1 group with 2 items, got %+v", res)
	}

	byCode := map[string]models.GetGroupItemResponse{}
	for _, it := range res[0].Items {
		byCode[it.ItemCode] = it
	}
	child := byCode["CHILD"]
	if child.ParentGroupCode == nil || *child.ParentGroupCode != parentCode ||
		child.ParentGroupItemCode == nil || *child.ParentGroupItemCode != parentItemCode {
		t.Errorf("CHILD parent = %v/%v, want %s/%s", child.ParentGroupCode, child.ParentGroupItemCode, parentCode, parentItemCode)
	}
	root := byCode["ROOT"]
	if root.ParentGroupCode != nil || root.ParentGroupItemCode != nil {
		t.Errorf("ROOT parent should be nil, got %v/%v", root.ParentGroupCode, root.ParentGroupItemCode)
	}
}
