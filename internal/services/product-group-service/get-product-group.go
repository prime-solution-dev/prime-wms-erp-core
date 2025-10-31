package productGroupService

import (
	"encoding/json"
	"errors"
	"fmt"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetGroup(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := models.GetGroupRequest{}
	res := []models.GetGroupResponse{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.CloseGORM(gormx)

	var groups []models.Group
	groupQuery := gormx.Model(&models.Group{})

	if len(req.GroupCodes) > 0 {
		groupQuery = groupQuery.Where("group_code IN ?", req.GroupCodes)
	}

	if err := groupQuery.Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("failed to query groups: %v", err)
	}

	groupIDs := []uuid.UUID{}
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
	}

	var items []models.GroupItem
	itemQuery := gormx.Model(&models.GroupItem{}).Where("group_id IN ?", groupIDs)

	if len(req.ItemCodes) > 0 {
		itemQuery = itemQuery.Where("item_code IN ?", req.ItemCodes)
	}

	if err := itemQuery.Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to query items: %v", err)
	}

	itemMap := make(map[uuid.UUID][]models.GroupItem)
	for _, it := range items {
		itemMap[it.GroupID] = append(itemMap[it.GroupID], it)
	}

	for _, g := range groups {
		res = append(res, models.GetGroupResponse{
			Group: g,
			Items: itemMap[g.ID],
		})
	}

	return res, nil
}
