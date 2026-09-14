package priceService

import (
	"testing"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

// lookup ต้องแยก value_int = 0 ที่เป็นค่าจริง ออกจากกรณีหาไม่เจอ
// เพราะผู้เรียกใช้ bool นี้ตัดสินว่าจะข้าม extra แถวนั้นหรือไม่
func TestGroupItemValueIntsLookup(t *testing.T) {
	values := groupItemValueInts{
		"PG06": {"PG06_40": 40, "PG06_0": 0},
	}

	for _, tc := range []struct {
		name      string
		group     string
		item      string
		wantValue float64
		wantFound bool
	}{
		{"เจอทั้งคู่", "PG06", "PG06_40", 40, true},
		{"value_int = 0 ที่เป็นค่าจริง", "PG06", "PG06_0", 0, true},
		{"ไม่มี group", "PG99", "PG06_40", 0, false},
		{"มี group แต่ไม่มี item", "PG06", "PG06_99", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, found := values.lookup(tc.group, tc.item)
			if v != tc.wantValue || found != tc.wantFound {
				t.Fatalf("lookup(%q, %q) = (%v, %v) ต้องเป็น (%v, %v)",
					tc.group, tc.item, v, found, tc.wantValue, tc.wantFound)
			}
		})
	}
}

func TestGroupItemValueIntsLookupOnNilMap(t *testing.T) {
	var values groupItemValueInts
	if v, found := values.lookup("PG06", "PG06_40"); v != 0 || found {
		t.Fatalf("map ว่างต้องคืน (0, false) แต่ได้ (%v, %v)", v, found)
	}
}

func subGroupWithConditions(codes ...string) models.PriceListSubGroup {
	extras := make([]models.PriceListGroupExtra, 0, len(codes))
	for _, c := range codes {
		extras = append(extras, models.PriceListGroupExtra{ConditionCode: c})
	}

	return models.PriceListSubGroup{
		PriceListGroup: models.PriceListGroup{PriceListGroupExtras: extras},
	}
}

// ไม่มี condition_code ที่ต้องใช้ ต้องไม่แตะ DB เลย
// ถ้าเผลอยิง query จะ error เพราะ test นี้ไม่มี DB ให้ต่อ
func TestLoadGroupItemValueIntsSkipsDBWhenNoConditionCode(t *testing.T) {
	for _, tc := range []struct {
		name      string
		subGroups []models.PriceListSubGroup
	}{
		{"ไม่มี subgroup", nil},
		{"subgroup ไม่มี extra", []models.PriceListSubGroup{{}}},
		{"extra ที่ condition_code ว่าง", []models.PriceListSubGroup{subGroupWithConditions("", "")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values, err := loadGroupItemValueInts(tc.subGroups)
			if err != nil {
				t.Fatalf("ไม่ควร error: %v", err)
			}
			if len(values) != 0 {
				t.Fatalf("ต้องได้ map ว่าง แต่ได้ %v", values)
			}
		})
	}
}

// condition_code ซ้ำข้าม subgroup ต้อง dedup ก่อนส่งเข้า query เดียว
func TestCollectConditionCodesDedups(t *testing.T) {
	subGroups := []models.PriceListSubGroup{
		subGroupWithConditions("PG06", ""),
		subGroupWithConditions("PG06", "PG03"),
	}

	got := collectConditionCodes(subGroups)
	if len(got) != 2 {
		t.Fatalf("ต้องเหลือ 2 code หลัง dedup แต่ได้ %v", got)
	}

	seen := map[string]bool{}
	for _, c := range got {
		seen[c] = true
	}
	if !seen["PG06"] || !seen["PG03"] || seen[""] {
		t.Fatalf("ต้องได้ PG06 และ PG03 เท่านั้น (ไม่เอา code ว่าง) แต่ได้ %v", got)
	}
}

// group_item ต้องถูกโหลดครั้งเดียวต่อ request ไม่ว่ามี subgroup กี่ตัว
// เดิม lookup อยู่ใน loop subgroup × extra จึงเปิด DB connection ใหม่ 424 ครั้ง
// สำหรับ ITEM_1 (212 subgroup × 2 extra)
func TestUpdateLatestLoadsGroupItemValuesOnce(t *testing.T) {
	pointInventoryEndpointAtFake(t, `[]`)

	const subGroupCount = 50

	subGroups := make([]models.PriceListSubGroup, 0, subGroupCount)
	ids := make([]string, 0, subGroupCount)
	for i := 0; i < subGroupCount; i++ {
		id := uuid.New()
		ids = append(ids, id.String())
		subGroups = append(subGroups, models.PriceListSubGroup{
			ID:           id,
			SubGroupCode: "SUB",
			SubgroupKey:  "SUB",
			PriceListSubGroupKeys: []models.PriceListSubGroupKey{
				{Code: "PG06", Value: "PG06_40"},
			},
			PriceListGroup: models.PriceListGroup{
				PriceListGroupExtras: []models.PriceListGroupExtra{
					{ConditionCode: "PG06", Operator: ">", CondRangeMax: 38, ValueInt: 1},
					{ConditionCode: "PG03", Operator: ">", CondRangeMax: 10, ValueInt: 1},
				},
			},
		})
	}

	originalGetByIDs := getPriceListSubGroupsByIDsFunc
	getPriceListSubGroupsByIDsFunc = func([]uuid.UUID) ([]models.PriceListSubGroup, error) {
		return subGroups, nil
	}
	t.Cleanup(func() { getPriceListSubGroupsByIDsFunc = originalGetByIDs })

	originalGetFormulas := getPriceListSubGroupFormulasMapBySubGroupCodesFunc
	getPriceListSubGroupFormulasMapBySubGroupCodesFunc = func([]string) (map[string][]models.PriceListSubGroupFormulasMap, error) {
		return map[string][]models.PriceListSubGroupFormulasMap{}, nil
	}
	t.Cleanup(func() { getPriceListSubGroupFormulasMapBySubGroupCodesFunc = originalGetFormulas })

	originalUpdate := updateLatestSubGroupFunc
	updateLatestSubGroupFunc = func(models.UpdatePriceListSubGroupRequest) error { return nil }
	t.Cleanup(func() { updateLatestSubGroupFunc = originalUpdate })

	loads := 0
	var gotCodes []string
	originalLoad := loadGroupItemValueIntsFunc
	loadGroupItemValueIntsFunc = func(sgs []models.PriceListSubGroup) (groupItemValueInts, error) {
		loads++
		gotCodes = collectConditionCodes(sgs)
		return groupItemValueInts{"PG06": {"PG06_40": 40}}, nil
	}
	t.Cleanup(func() { loadGroupItemValueIntsFunc = originalLoad })

	if _, err := RunUpdateLatestPriceListSubGroup(models.UpdateLatestPriceListSubGroupRequest{
		SubGroupIDs: ids,
	}); err != nil {
		t.Fatalf("RunUpdateLatestPriceListSubGroup: %v", err)
	}

	if loads != 1 {
		t.Fatalf("%d subgroup × 2 extra ต้องโหลด group_item ครั้งเดียว แต่โหลด %d ครั้ง",
			subGroupCount, loads)
	}
	if len(gotCodes) != 2 {
		t.Fatalf("condition_code ต้องถูก dedup เหลือ 2 code แต่ได้ %v", gotCodes)
	}
}
