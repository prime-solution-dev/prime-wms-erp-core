package patterns

import (
	"testing"

	"prime-erp-core/internal/models"
)

// helper สร้าง subgroup พร้อม AvgWeight ไว้ดูว่าเก็บ record ตัวไหนไว้
func sgWith(id string, avg float64) models.PriceListSubGroupResponse {
	return models.PriceListSubGroupResponse{
		ID: id,
		InventoryWeight: []models.InventoryWeightResponse{
			{AvgWeight: avg},
		},
	}
}

func ids(subGroups []models.PriceListSubGroupResponse) []string {
	out := make([]string, 0, len(subGroups))
	for _, sg := range subGroups {
		out = append(out, sg.ID)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// pattern ที่ไม่มีคอลัมน์ batch_no ต้องยุบให้เหลือ 1 แถวต่อ 1 sub_group
// และต้องคงลำดับเดิมของ slice ไว้ พร้อมเก็บ record ตัวแรกของแต่ละ ID
func TestCollapseNonBatchSubGroupsCollapsesWhenNotPerBatch(t *testing.T) {
	in := []models.PriceListSubGroupResponse{
		sgWith("a", 1),
		sgWith("b", 2),
		sgWith("a", 3),
		sgWith("c", 4),
		sgWith("b", 5),
		sgWith("a", 6),
	}

	got := collapseNonBatchSubGroups(in, false)

	if want := []string{"a", "b", "c"}; !equalStrings(ids(got), want) {
		t.Fatalf("ได้ %v ต้องเป็น %v", ids(got), want)
	}
	wantAvg := []float64{1, 2, 4}
	for i, sg := range got {
		if sg.InventoryWeight[0].AvgWeight != wantAvg[i] {
			t.Errorf("index %d ได้ %v ต้องเป็น %v (ต้องเก็บ record ตัวแรก)", i, sg.InventoryWeight[0].AvgWeight, wantAvg[i])
		}
	}
	if len(in) != 6 {
		t.Errorf("ห้ามแก้ slice ต้นฉบับ ได้ len %d", len(in))
	}
}

// happy path: ไม่มี ID ซ้ำ ต้องได้ทุกแถวครบและลำดับเดิม
func TestCollapseNonBatchSubGroupsKeepsAllWhenNoDuplicate(t *testing.T) {
	in := []models.PriceListSubGroupResponse{sgWith("a", 1), sgWith("b", 2), sgWith("c", 3)}

	got := collapseNonBatchSubGroups(in, false)

	if want := []string{"a", "b", "c"}; !equalStrings(ids(got), want) {
		t.Fatalf("ได้ %v ต้องเป็น %v", ids(got), want)
	}
}

// pattern ที่มีคอลัมน์ batch_no (ITEM_7/8/22) ต้องไม่ถูกยุบเลย
func TestCollapseNonBatchSubGroupsKeepsAllWhenPerBatch(t *testing.T) {
	in := []models.PriceListSubGroupResponse{
		sgWith("a", 1),
		sgWith("a", 2),
		sgWith("a", 3),
	}

	got := collapseNonBatchSubGroups(in, true)

	if want := []string{"a", "a", "a"}; !equalStrings(ids(got), want) {
		t.Fatalf("ได้ %v ต้องเป็น %v", ids(got), want)
	}
	for i, sg := range got {
		if sg.InventoryWeight[0].AvgWeight != in[i].InventoryWeight[0].AvgWeight {
			t.Errorf("index %d ค่าเปลี่ยนไป", i)
		}
	}
}

// slice ว่าง / nil ต้องไม่ panic
func TestCollapseNonBatchSubGroupsEmpty(t *testing.T) {
	if got := collapseNonBatchSubGroups(nil, false); len(got) != 0 {
		t.Errorf("nil ต้องได้ผลลัพธ์ว่าง ได้ len %d", len(got))
	}
	if got := collapseNonBatchSubGroups([]models.PriceListSubGroupResponse{}, true); len(got) != 0 {
		t.Errorf("slice ว่างต้องได้ผลลัพธ์ว่าง ได้ len %d", len(got))
	}
}
