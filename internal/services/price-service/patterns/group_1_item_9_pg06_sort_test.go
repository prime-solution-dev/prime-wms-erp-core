package patterns

import (
	"fmt"
	"testing"

	"prime-erp-core/internal/models"
)

// config ต้องมี key "productGroup6" -> PG06 ไม่งั้น handler หา PG06 ไม่เจอ
// แล้ว fallback เป็น "PRODUCT_GROUP6" ทำให้เรียง Size ไม่ได้
func TestGroup1Item9_Config_HasProductGroup6Mapping(t *testing.T) {
	cfg, err := LoadConfiguration("GROUP_1_ITEM_9")
	if err != nil {
		t.Fatalf("โหลด config ไม่ได้: %v", err)
	}
	if got := cfg.ValueMappings.GroupCodeMappings["productGroup6"]; got != "PG06" {
		t.Fatalf(`groupCodeMappings["productGroup6"] = %q, ต้องเป็น "PG06"`, got)
	}
}

// subgroup_key เรียงตาม string (PG06_10, PG06_11, PG06_3 ...) แต่ต้องออกเป็นแถวเรียงตามค่า Size
func TestGroup1Item9_RowsSortedBySizeValue(t *testing.T) {
	cfg, err := LoadConfiguration("GROUP_1_ITEM_9")
	if err != nil {
		t.Fatalf("โหลด config ไม่ได้: %v", err)
	}
	pattern := &cfg.Patterns[0]

	inputs := []struct {
		code string
		size float64
	}{
		{"PG06_10", 28}, {"PG06_11", 32}, {"PG06_3", 12}, {"PG06_6", 25},
		{"PG06_7", 10}, {"PG06_8", 16}, {"PG06_9", 20},
	}
	subGroups := make([]models.PriceListSubGroupResponse, 0, len(inputs))
	for i, in := range inputs {
		sg := item9SubGroup(fmt.Sprintf("sg-%d", i), in.code, fmt.Sprintf("%g", in.size), 100)
		for k := range sg.SubGroupKeys {
			if sg.SubGroupKeys[k].GroupCode == "PG06" {
				sg.SubGroupKeys[k].ValueNumber = in.size
				sg.SubGroupKeys[k].HasValue = true
			}
		}
		subGroups = append(subGroups, sg)
	}

	// เหมือน pattern_group_1_item_9.go
	pg2 := getGroupCodeFromConfig(cfg, pattern, "productGroup2", "PRODUCT_GROUP2")
	pg7 := getGroupCodeFromConfig(cfg, pattern, "productGroup7", "PRODUCT_GROUP7")
	pg6 := getGroupCodeFromConfig(cfg, pattern, "productGroup6", "PRODUCT_GROUP6")
	SortSubGroupsByValue(subGroups, pg2, pg7, pg6)

	merged := mergeGroup1Item9Rows(buildDynamicRows(cfg, pattern, subGroups))

	want := []string{"10", "12", "16", "20", "25", "28", "32"}
	if len(merged) != len(want) {
		t.Fatalf("ต้องได้ %d แถว ได้ %d", len(want), len(merged))
	}
	for i, row := range merged {
		if got, _ := row["pg_06"].(string); got != want[i] {
			t.Fatalf("แถว %d size = %q ต้องเป็น %q", i, got, want[i])
		}
	}
}
