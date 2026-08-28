package groupService_test

import (
	"testing"

	"prime-erp-core/internal/models"
	groupService "prime-erp-core/internal/services/group-service"

	"github.com/google/uuid"
)

func validSyncRequest() groupService.SyncGroupMasterRequest {
	groupID := uuid.New()
	return groupService.SyncGroupMasterRequest{
		Groups: []models.Group{
			{ID: groupID, GroupCode: "PG01", GroupName: "Product Group", Seq: 1},
		},
		GroupItems: []models.GroupItem{
			{ID: uuid.New(), GroupID: groupID, ItemCode: "BU01", ItemName: "Bulk", ValueInt: 1},
		},
	}
}

// Wiping group/group_item takes price list calculations down with it, so an empty snapshot is
// treated as a broken payload rather than a request to delete everything.
func TestSyncGroupMasterRejectsAnEmptySnapshot(t *testing.T) {
	req := validSyncRequest()
	req.Groups = nil

	if _, err := groupService.SyncGroupMasterWithDB(nil, req); err == nil {
		t.Fatal("an empty snapshot must return an error")
	}
}

func TestSyncGroupMasterRejectsANilConnection(t *testing.T) {
	if _, err := groupService.SyncGroupMasterWithDB(nil, validSyncRequest()); err == nil {
		t.Fatal("a nil connection must return an error")
	}
}
