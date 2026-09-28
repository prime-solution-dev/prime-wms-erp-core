package priceService

import (
	"encoding/json"
	"fmt"

	groupService "prime-erp-core/internal/services/group-service"

	"prime-erp-core/internal/models"
)

// groupItemValueInts คือ group_code -> item_code -> group_item.value_int
//
// เดิม lookup นี้เป็น query ต่อครั้งใน loop subgroup × extra (ITEM_1 = 212 × 2)
// ซึ่งเปิด-ปิด DB connection ใหม่ทุกครั้ง จึงเปลี่ยนมาโหลดล่วงหน้าครั้งเดียวต่อ request
type groupItemValueInts map[string]map[string]float64

// lookup คืน (value_int, resolve ได้ไหม) — แยก value_int = 0 ที่เป็นค่าจริง
// ออกจากกรณีไม่มี group / ไม่มี item เหมือน GetGroupItemValueInt เดิม
func (m groupItemValueInts) lookup(groupCode, itemCode string) (float64, bool) {
	items, ok := m[groupCode]
	if !ok {
		return 0, false
	}

	v, ok := items[itemCode]
	return v, ok
}

// seam สำหรับ test
var loadGroupItemValueIntsFunc = loadGroupItemValueInts

// loadGroupItemValueInts รวบรวม condition_code ที่ไม่ว่างจากทุก subgroup แล้วโหลด
// group_item ของ group เหล่านั้นมาครั้งเดียว
func loadGroupItemValueInts(subGroups []models.PriceListSubGroup) (groupItemValueInts, error) {
	groupCodes := collectConditionCodes(subGroups)

	// ไม่มี group ที่ต้องใช้ ก็ไม่ต้องแตะ DB
	if len(groupCodes) == 0 {
		return groupItemValueInts{}, nil
	}

	// GetGroup โหลดแบบ batch อยู่แล้ว (group WHERE group_code IN + group_item WHERE group_id IN)
	reqJson, err := json.Marshal(models.GetGroupRequest{GroupCodes: groupCodes})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal group request: %w", err)
	}

	resp, err := groupService.GetGroup(nil, string(reqJson))
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	groups, ok := resp.([]models.GetGroupResponse)
	if !ok {
		return nil, fmt.Errorf("failed to cast group response")
	}

	values := make(groupItemValueInts, len(groups))
	for _, g := range groups {
		items := make(map[string]float64, len(g.Items))
		for _, it := range g.Items {
			items[it.ItemCode] = it.ValueInt
		}
		values[g.GroupCode] = items
	}

	return values, nil
}

// collectConditionCodes คืน condition_code ที่ไม่ว่างแบบไม่ซ้ำจากทุก subgroup
func collectConditionCodes(subGroups []models.PriceListSubGroup) []string {
	codeSet := map[string]struct{}{}
	for _, sg := range subGroups {
		for _, e := range sg.PriceListGroup.PriceListGroupExtras {
			if e.ConditionCode != "" {
				codeSet[e.ConditionCode] = struct{}{}
			}
		}
	}

	codes := make([]string, 0, len(codeSet))
	for code := range codeSet {
		codes = append(codes, code)
	}

	return codes
}
