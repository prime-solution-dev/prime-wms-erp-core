package patterns

import (
	"testing"

	"prime-erp-core/internal/models"
)

// sgk ย่อการสร้าง SubGroupKey ในเทสต์
func sgk(groupCode, valueName string, value float64, has bool) models.PriceListSubGroupKeyResponse {
	return models.PriceListSubGroupKeyResponse{
		GroupCode:   groupCode,
		ValueCode:   groupCode + "_" + valueName,
		ValueName:   valueName,
		ValueNumber: value,
		HasValue:    has,
	}
}

func sub(keys ...models.PriceListSubGroupKeyResponse) models.PriceListSubGroupResponse {
	return models.PriceListSubGroupResponse{SubGroupKeys: keys}
}

func TestValueOfGroup(t *testing.T) {
	keys := []models.PriceListSubGroupKeyResponse{
		sgk("PG06", "1.2", 1.2, true),
		sgk("PG05", "4' x 8'", 32, true),
	}

	if v, ok := valueOfGroup(keys, "PG06"); v != 1.2 || !ok {
		t.Fatalf("PG06 ต้องได้ (1.2, true) แต่ได้ (%v, %v)", v, ok)
	}
	if v, ok := valueOfGroup(keys, "PG99"); v != 0 || ok {
		t.Fatalf("กลุ่มที่ไม่มี ต้องได้ (0, false) แต่ได้ (%v, %v)", v, ok)
	}
}

func TestCmpGroupValue(t *testing.T) {
	cases := []struct {
		name string
		a    models.PriceListSubGroupKeyResponse
		b    models.PriceListSubGroupKeyResponse
		want int
	}{
		{"ตัวเลขน้อยกว่ามาก่อน", sgk("PG06", "9", 9, true), sgk("PG06", "10", 10, true), -1},
		{"string sort เคยให้ 100 มาก่อน 12 ต้องกลับด้าน", sgk("PG06", "100", 100, true), sgk("PG06", "12", 12, true), 1},
		{"ค่าเท่ากัน tie-break ด้วยชื่อ", sgk("PG05", "100x300", 30000, true), sgk("PG05", "150x200", 30000, true), -1},
		{"ฝ่ายที่มีค่ามาก่อนฝ่ายที่ resolve ไม่ได้", sgk("PG05", "ESP", 0, false), sgk("PG05", "4' x 8'", 32, true), 1},
		{"ไม่มีค่าทั้งคู่ ใช้ string compare", sgk("PG05", "AAA", 0, false), sgk("PG05", "BBB", 0, false), -1},
		{"value = 0 จริง ถือว่ามีค่า มาก่อนตัวที่ resolve ไม่ได้", sgk("PG08", "ศูนย์", 0, true), sgk("PG08", "ไม่รู้", 0, false), -1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := cmpGroupValue([]models.PriceListSubGroupKeyResponse{c.a}, []models.PriceListSubGroupKeyResponse{c.b}, c.a.GroupCode)
			if got != c.want {
				t.Fatalf("cmpGroupValue = %d ต้องเป็น %d", got, c.want)
			}
		})
	}
}

func TestCmpSubGroupKeys_StopsAtFirstDifference(t *testing.T) {
	a := []models.PriceListSubGroupKeyResponse{sgk("PG06", "10", 10, true), sgk("PG05", "5' x 20'", 100, true)}
	b := []models.PriceListSubGroupKeyResponse{sgk("PG06", "12", 12, true), sgk("PG05", "4' x 8'", 32, true)}

	if got := cmpSubGroupKeys(a, b, "PG06", "PG05"); got != -1 {
		t.Fatalf("PG06 ต่างกันแล้วต้องจบที่ตัวแรก ได้ %d ต้องเป็น -1", got)
	}

	c := []models.PriceListSubGroupKeyResponse{sgk("PG06", "10", 10, true), sgk("PG05", "5' x 20'", 100, true)}
	d := []models.PriceListSubGroupKeyResponse{sgk("PG06", "10", 10, true), sgk("PG05", "4' x 8'", 32, true)}

	if got := cmpSubGroupKeys(c, d, "PG06", "PG05"); got != 1 {
		t.Fatalf("PG06 เท่ากันต้องไปเทียบ PG05 ต่อ ได้ %d ต้องเป็น 1", got)
	}

	if got := cmpSubGroupKeys(c, c, "PG06", "PG05"); got != 0 {
		t.Fatalf("เหมือนกันทุกแกนต้องได้ 0 ได้ %d", got)
	}
}

func TestSortSubGroupsByValue(t *testing.T) {
	// ลำดับตั้งต้นคือผลของ lexicographic sort ที่เป็นบั๊กอยู่ตอนนี้
	subs := []models.PriceListSubGroupResponse{
		sub(sgk("PG06", "1.2", 1.2, true)),
		sub(sgk("PG06", "10", 10, true)),
		sub(sgk("PG06", "100", 100, true)),
		sub(sgk("PG06", "12", 12, true)),
		sub(sgk("PG06", "15", 15, true)),
		sub(sgk("PG06", "2.3", 2.3, true)),
	}

	SortSubGroupsByValue(subs, "PG06")

	want := []string{"1.2", "2.3", "10", "12", "15", "100"}
	for i, w := range want {
		got := subs[i].SubGroupKeys[0].ValueName
		if got != w {
			t.Fatalf("ตำแหน่ง %d ได้ %q ต้องเป็น %q — ลำดับทั้งหมด %v", i, got, w, valueNames(subs))
		}
	}
}

func TestSortSubGroupsByValue_IsStableAndDeterministic(t *testing.T) {
	build := func() []models.PriceListSubGroupResponse {
		return []models.PriceListSubGroupResponse{
			sub(sgk("PG05", "150x200", 30000, true)),
			sub(sgk("PG05", "100x300", 30000, true)),
			sub(sgk("PG05", "75x75", 5625, true)),
		}
	}

	var first []string
	for round := 0; round < 50; round++ {
		subs := build()
		SortSubGroupsByValue(subs, "PG05")
		got := valueNames(subs)
		if round == 0 {
			first = got
			continue
		}
		for i := range got {
			if got[i] != first[i] {
				t.Fatalf("รอบที่ %d ได้ %v ต่างจากรอบแรก %v — ลำดับไม่นิ่ง", round, got, first)
			}
		}
	}

	want := []string{"75x75", "100x300", "150x200"}
	for i, w := range want {
		if first[i] != w {
			t.Fatalf("ตำแหน่ง %d ได้ %q ต้องเป็น %q", i, first[i], w)
		}
	}
}

func valueNames(subs []models.PriceListSubGroupResponse) []string {
	out := make([]string, 0, len(subs))
	for _, s := range subs {
		if len(s.SubGroupKeys) > 0 {
			out = append(out, s.SubGroupKeys[0].ValueName)
		}
	}
	return out
}

func TestOrderedUnique(t *testing.T) {
	rows := []AGGridRowData{
		{"row_group_value": "1.2"},
		{"row_group_value": "1.2"},
		{"row_group_value": "10"},
		{"row_group_value": ""},
		{"row_group_value": "100"},
		{"row_group_value": "10"},
	}

	got := orderedUnique(rows, "row_group_value")
	want := []string{"1.2", "10", "100"}

	if len(got) != len(want) {
		t.Fatalf("ได้ %v ต้องเป็น %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ตำแหน่ง %d ได้ %q ต้องเป็น %q — ทั้งหมด %v", i, got[i], want[i], got)
		}
	}
}

func TestOrderedUniqueBy(t *testing.T) {
	subs := []models.PriceListSubGroupResponse{
		sub(sgk("PG02", "เหล็กแผ่น", 6, true)),
		sub(sgk("PG02", "เหล็กแผ่น", 6, true)),
		sub(sgk("PG02", "แผ่นลาย", 9, true)),
	}

	got := orderedUniqueBy(subs, func(s models.PriceListSubGroupResponse) string {
		return s.SubGroupKeys[0].ValueName
	})
	want := []string{"เหล็กแผ่น", "แผ่นลาย"}

	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("ได้ %v ต้องเป็น %v", got, want)
	}
}

func TestSplitGroupCodes(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"PG03|PG08|PG05", []string{"PG03", "PG08", "PG05"}},
		{" PG06 | PG05 ", []string{"PG06", "PG05"}},
		{"PG06", []string{"PG06"}},
		{"", nil},
		{"||", nil},
	}

	for _, c := range cases {
		got := splitGroupCodes(c.in)
		if len(got) != len(c.want) {
			t.Fatalf("splitGroupCodes(%q) = %v ต้องเป็น %v", c.in, got, c.want)
		}
		for i := range c.want {
			if got[i] != c.want[i] {
				t.Fatalf("splitGroupCodes(%q)[%d] = %q ต้องเป็น %q", c.in, i, got[i], c.want[i])
			}
		}
	}
}

func TestNewValueByCode(t *testing.T) {
	subs := []models.PriceListSubGroupResponse{
		sub(sgk("PG05", "4' x 8'", 32, true), sgk("PG06", "1.2", 1.2, true)),
		sub(sgk("PG05", "1250x8'", 10000, true)),
		sub(models.PriceListSubGroupKeyResponse{GroupCode: "PG05", ValueCode: "PG05_40", ValueName: "ESP"}),
	}

	idx := newValueByCode(subs)

	if v, ok := idx["PG05_4' x 8'"]; !ok || v.value != 32 || !v.has {
		t.Fatalf("PG05_4' x 8' ต้องได้ (32, true) แต่ได้ %+v ok=%v", v, ok)
	}
	if v, ok := idx["PG05_1250x8'"]; !ok || v.value != 10000 {
		t.Fatalf("PG05_1250x8' ต้องได้ 10000 แต่ได้ %+v ok=%v", v, ok)
	}
	if v, ok := idx["PG05_40"]; !ok || v.has {
		t.Fatalf("PG05_40 ต้อง resolve ไม่ได้ แต่ได้ %+v ok=%v", v, ok)
	}
	if _, ok := idx["ไม่มีจริง"]; ok {
		t.Fatal("code ที่ไม่มีต้องไม่อยู่ใน index")
	}
}

func TestValueByCodeLess(t *testing.T) {
	idx := valueByCode{
		"PG05_22": {value: 10000, has: true},
		"PG05_3":  {value: 32, has: true},
		"PG05_40": {value: 0, has: false},
	}

	if !idx.Less("PG05_3", "4' x 8'", "PG05_22", "1250x8'") {
		t.Fatal("32 ต้องมาก่อน 10000")
	}
	if idx.Less("PG05_22", "1250x8'", "PG05_3", "4' x 8'") {
		t.Fatal("10000 ต้องไม่มาก่อน 32")
	}
	if !idx.Less("PG05_3", "4' x 8'", "PG05_40", "ESP") {
		t.Fatal("ตัวที่มีค่าต้องมาก่อนตัวที่ resolve ไม่ได้")
	}
	if !idx.Less("missing", "AAA", "missing2", "BBB") {
		t.Fatal("ไม่มีค่าทั้งคู่ ต้อง fallback เป็น string compare ของ label")
	}
}

func TestSortLabelsByValue(t *testing.T) {
	subs := []models.PriceListSubGroupResponse{
		sub(sgk("PG02", "เหล็กแผ่นตัด SIZE", 20, true)),
		sub(sgk("PG02", "เหล็กแผ่น", 6, true)),
		sub(sgk("PG02", "แผ่นลาย", 9, true)),
		sub(sgk("PG02", "เหล็กแผ่น special", 19, true)),
	}

	labels := []string{"เหล็กแผ่น", "เหล็กแผ่นตัด SIZE", "แผ่นลาย", "เหล็กแผ่น special"}
	sortLabelsByValue(labels, subs, "PG02")

	want := []string{"เหล็กแผ่น", "แผ่นลาย", "เหล็กแผ่น special", "เหล็กแผ่นตัด SIZE"}
	for i := range want {
		if labels[i] != want[i] {
			t.Fatalf("ตำแหน่ง %d ได้ %q ต้องเป็น %q — ทั้งหมด %v", i, labels[i], want[i], labels)
		}
	}
}

func TestSortLabelsByValue_UnknownGoesLast(t *testing.T) {
	subs := []models.PriceListSubGroupResponse{
		sub(sgk("PG02", "แผ่นลาย", 9, true)),
		sub(models.PriceListSubGroupKeyResponse{GroupCode: "PG02", ValueCode: "PG02_X", ValueName: "ไม่รู้จัก"}),
	}

	labels := []string{"ไม่รู้จัก", "แผ่นลาย"}
	sortLabelsByValue(labels, subs, "PG02")

	if labels[0] != "แผ่นลาย" || labels[1] != "ไม่รู้จัก" {
		t.Fatalf("ตัวที่ resolve ไม่ได้ต้องไปท้าย แต่ได้ %v", labels)
	}
}
