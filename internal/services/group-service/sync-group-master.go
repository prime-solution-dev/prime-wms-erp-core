package groupService

import (
	"encoding/json"
	"errors"
	"fmt"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SyncGroupMasterRequest struct {
	Groups     []models.Group     `json:"groups"`
	GroupItems []models.GroupItem `json:"group_items"`
}

type SyncGroupMasterResponse struct {
	GroupsSynced     int    `json:"groups_synced"`
	GroupItemsSynced int    `json:"group_items_synced"`
	Status           string `json:"status"`
	Message          string `json:"message"`
}

func SyncGroupMaster(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req SyncGroupMasterRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.CloseGORM(gormx)

	return SyncGroupMasterWithDB(gormx, req)
}

// SyncGroupMasterWithDB mirrors WMS's group tables: both are replaced by the snapshot, keeping the
// ids WMS assigned so the two sides stay 1:1.
func SyncGroupMasterWithDB(gormx *gorm.DB, req SyncGroupMasterRequest) (*SyncGroupMasterResponse, error) {
	// An empty snapshot is far likelier a broken payload than a real "delete everything", and
	// wiping group/group_item takes price list calculations down with it.
	if len(req.Groups) == 0 {
		return nil, errors.New("groups cannot be empty")
	}
	if gormx == nil {
		return nil, errors.New("database connection is nil")
	}

	// models.Group carries a GroupItems association; the snapshot ships the items separately, so
	// the association is cleared and the two tables are written apart to keep GORM from
	// double-inserting.
	groups := make([]models.Group, len(req.Groups))
	copy(groups, req.Groups)
	for i := range groups {
		groups[i].GroupItems = nil
	}
	items := make([]models.GroupItem, len(req.GroupItems))
	copy(items, req.GroupItems)

	if err := gormx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("1 = 1").Delete(&models.GroupItem{}).Error; err != nil {
			return fmt.Errorf("failed to delete existing group items: %w", err)
		}
		if err := tx.Unscoped().Where("1 = 1").Delete(&models.Group{}).Error; err != nil {
			return fmt.Errorf("failed to delete existing groups: %w", err)
		}
		if err := tx.Omit("GroupItems").Create(&groups).Error; err != nil {
			return fmt.Errorf("failed to create groups: %w", err)
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return fmt.Errorf("failed to create group items: %w", err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &SyncGroupMasterResponse{
		GroupsSynced:     len(groups),
		GroupItemsSynced: len(items),
		Status:           "success",
		Message: fmt.Sprintf("Group master sync completed. Replaced with %d groups and %d group items.",
			len(groups), len(items)),
	}, nil
}
