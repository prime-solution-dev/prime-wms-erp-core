package groupService_test

import (
	"net/http/httptest"
	"testing"

	"prime-erp-core/internal/models"
	groupService "prime-erp-core/internal/services/group-service"

	"github.com/gin-gonic/gin"
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

	// Asserting the message, not just err != nil: this call also passes a nil connection, so a
	// generic check would still pass if the guards were ever reordered and the nil check fired
	// first — leaving the wipe-protection untested.
	_, err := groupService.SyncGroupMasterWithDB(nil, req)
	if err == nil {
		t.Fatal("an empty snapshot must return an error")
	}
	if err.Error() != "groups cannot be empty" {
		t.Errorf("the empty-snapshot guard must fire first, got %v", err)
	}
}

func TestSyncGroupMasterRejectsANilConnection(t *testing.T) {
	if _, err := groupService.SyncGroupMasterWithDB(nil, validSyncRequest()); err == nil {
		t.Fatal("a nil connection must return an error")
	}
}

// The handler is what the route actually calls; a malformed body must be rejected before it ever
// reaches the database.
func TestSyncGroupMasterRejectsBrokenJson(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	if _, err := groupService.SyncGroupMaster(ctx, `{`); err == nil {
		t.Fatal("broken JSON must return an error")
	}
}
