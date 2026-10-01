package groupService

import (
	"encoding/json"
	"testing"

	"prime-erp-core/internal/models"
)

// หน้า price list ฝั่ง web ต้องได้สาย parent ไปกรองตัวเลือก ถ้า DTO ตัดทิ้งจะเห็นทุก item
func TestToGroupItemResponseKeepsParent(t *testing.T) {
	code, item := "PG02", "PG02_2"

	got := toGroupItemResponse(models.GroupItem{ItemCode: "PG01_8", ParentGroupCode: &code, ParentGroupItemCode: &item})
	if got.ParentGroupCode == nil || *got.ParentGroupCode != code {
		t.Fatalf("ParentGroupCode = %v, want %q", got.ParentGroupCode, code)
	}
	if got.ParentGroupItemCode == nil || *got.ParentGroupItemCode != item {
		t.Fatalf("ParentGroupItemCode = %v, want %q", got.ParentGroupItemCode, item)
	}
	if got.ItemCode != "PG01_8" {
		t.Fatalf("ItemCode = %q", got.ItemCode)
	}
}

func TestToGroupItemResponseNilParentSerializesNull(t *testing.T) {
	b, err := json.Marshal(toGroupItemResponse(models.GroupItem{}))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"parent_group_code", "parent_group_item_code"} {
		v, ok := m[k]
		if !ok || v != nil {
			t.Fatalf("%s = %v (present=%v), want null", k, v, ok)
		}
	}
}
