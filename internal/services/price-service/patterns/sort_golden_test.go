package patterns

import (
	"fmt"
	"testing"

	"prime-erp-core/internal/models"
)

// ค่าจริงของ PG06 "ความหนา" จาก group_item ใน UAT
var uatPG06 = map[string]float64{
	"1.2": 1.20, "1.4": 1.40, "1.9": 1.90, "2.3": 2.30, "2.5": 2.50,
	"2.9": 2.90, "3.8": 3.80, "4.3": 4.30, "4.8": 4.80, "5.5": 5.50,
	"5.8": 5.80, "8": 8.00, "9": 9.00, "10": 10.00, "12": 12.00,
	"15": 15.00, "16": 16.00, "19": 19.00, "20": 20.00, "22": 22.00,
	"25": 25.00, "27": 27.00, "30": 30.00, "32": 32.00, "38": 38.00,
	"40": 40.00, "45": 45.00, "50": 50.00, "100": 100.00,
}

// ค่าจริงของ PG05 "ขนาดหน้ากว้าง" จาก group_item ใน UAT
// value คือผลคูณสองด้าน (พื้นที่) ไม่ใช่ความกว้าง
var uatPG05 = map[string]float64{
	"4' x 8'": 32.00, "5' x 10'": 50.00, "5' x 20'": 100.00,
	"75x75": 5625.00, "4'x1500": 6000.00, "4'x2400": 9600.00,
	"1250x8'": 10000.00, "100x100": 10000.00, "125x125": 15625.00,
	"40x520": 20800.00, "150x150": 22500.00, "150x175": 26250.00,
	"100x300": 30000.00, "150x200": 30000.00, "65x500": 32500.00,
	"200x200": 40000.00, "200x220": 44000.00, "200x300": 60000.00,
	"250x250": 62500.00, "300x300": 90000.00, "5'x5700": 28500.00,
}

// ค่าจริงของ PG03 "เกรด/รูปแบบ" จาก group_item ใน UAT
var uatPG03 = map[string]float64{"SS400": 10.00, "LT": 21.00, "T": 8.00, "S4": 33.00}

// ค่าจริงของ PG02 "หมวดย่อย" จาก group_item ใน UAT
var uatPG02 = map[string]float64{
	"เหล็กแผ่น": 6.00, "แผ่นลาย": 9.00,
	"เหล็กแผ่น special": 19.00, "เหล็กแผ่นตัด SIZE": 20.00,
}

func uatKey(groupCode, name string, table map[string]float64) models.PriceListSubGroupKeyResponse {
	v, ok := table[name]
	return models.PriceListSubGroupKeyResponse{
		GroupCode:   groupCode,
		ValueCode:   fmt.Sprintf("%s_%s", groupCode, name),
		ValueName:   name,
		ValueNumber: v,
		HasValue:    ok,
	}
}

func uatSubGroup(id, pg02, pg05, pg06 string) models.PriceListSubGroupResponse {
	return models.PriceListSubGroupResponse{
		ID:           id,
		SubgroupCode: id,
		SubGroupKeys: []models.PriceListSubGroupKeyResponse{
			uatKey("PG02", pg02, uatPG02),
			uatKey("PG05", pg05, uatPG05),
			uatKey("PG06", pg06, uatPG06),
		},
	}
}

// บั๊กที่ business รายงาน: แถว "แผ่น mm." แสดง 1.2, 1.4, 1.9, 10, 100, 12, 15
// เพราะ sort.Strings เทียบเป็น string
func TestGolden_RowOrder_ThicknessIsNumeric(t *testing.T) {
	// ลำดับตั้งต้นคือผลของ lexicographic sort ที่เป็นบั๊กอยู่ตอนนี้
	input := []string{
		"1.2", "1.4", "1.9", "10", "100", "12", "15", "16", "19",
		"2.3", "2.5", "2.9", "20", "22", "25", "27", "3.8", "30",
		"32", "38", "4.3", "4.8", "40", "45", "5.5", "5.8", "50", "8", "9",
	}

	subs := make([]models.PriceListSubGroupResponse, 0, len(input))
	for i, name := range input {
		subs = append(subs, uatSubGroup(fmt.Sprintf("sg-%d", i), "เหล็กแผ่น", "4' x 8'", name))
	}

	SortSubGroupsByValue(subs, "PG06")

	want := []string{
		"1.2", "1.4", "1.9", "2.3", "2.5", "2.9", "3.8", "4.3", "4.8",
		"5.5", "5.8", "8", "9", "10", "12", "15", "16", "19", "20",
		"22", "25", "27", "30", "32", "38", "40", "45", "50", "100",
	}

	for i := range want {
		got := subs[i].SubGroupKeys[2].ValueName
		if got != want[i] {
			t.Fatalf("แถวที่ %d ได้ %q ต้องเป็น %q", i, got, want[i])
		}
	}
}

// หัวคอลัมน์ tab "เหล็กแผ่น" ปัจจุบันเรียง 1250x8', 4' x 8', 4'x1500, ...
// ต้องเรียงด้วย group_item.value
func TestGolden_ColumnOrder_SteelSheetTab(t *testing.T) {
	input := []string{"1250x8'", "4' x 8'", "4'x1500", "4'x2400", "5' x 10'", "5' x 20'", "5'x5700"}

	subs := make([]models.PriceListSubGroupResponse, 0, len(input))
	for i, name := range input {
		subs = append(subs, uatSubGroup(fmt.Sprintf("sg-%d", i), "เหล็กแผ่น", name, "1.2"))
	}

	SortSubGroupsByValue(subs, "PG05")

	want := []string{"4' x 8'", "5' x 10'", "5' x 20'", "4'x1500", "4'x2400", "1250x8'", "5'x5700"}
	for i := range want {
		got := subs[i].SubGroupKeys[1].ValueName
		if got != want[i] {
			t.Fatalf("คอลัมน์ที่ %d ได้ %q ต้องเป็น %q", i, got, want[i])
		}
	}
}

// หัวคอลัมน์ tab "เหล็กแผ่นตัด SIZE"
func TestGolden_ColumnOrder_CutSizeTab(t *testing.T) {
	input := []string{
		"100x100", "100x300", "125x125", "150x150", "150x175", "150x200",
		"200x200", "200x220", "200x300", "250x250", "300x300", "40x520",
		"65x500", "75x75",
	}

	subs := make([]models.PriceListSubGroupResponse, 0, len(input))
	for i, name := range input {
		subs = append(subs, uatSubGroup(fmt.Sprintf("sg-%d", i), "เหล็กแผ่นตัด SIZE", name, "10"))
	}

	SortSubGroupsByValue(subs, "PG05")

	got := make([]string, 0, len(subs))
	for _, s := range subs {
		got = append(got, s.SubGroupKeys[1].ValueName)
	}

	// 100x300 กับ 150x200 มี value = 30,000.00 เท่ากัน tie-break ด้วยชื่อ
	expected := []string{
		"75x75", "100x100", "125x125", "40x520", "150x150", "150x175",
		"100x300", "150x200", "65x500", "200x200", "200x220", "200x300",
		"250x250", "300x300",
	}

	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("คอลัมน์ที่ %d ได้ %q ต้องเป็น %q — ทั้งหมด %v", i, got[i], expected[i], got)
		}
	}
}

// tab "เหล็กแผ่น special" เรียงด้วยเกรด ไม่ใช่ตัวเลขในชื่อ
func TestGolden_ColumnOrder_SpecialTabUsesGrade(t *testing.T) {
	subs := []models.PriceListSubGroupResponse{
		{ID: "sg-lt", SubGroupKeys: []models.PriceListSubGroupKeyResponse{uatKey("PG03", "LT", uatPG03)}},
		{ID: "sg-ss", SubGroupKeys: []models.PriceListSubGroupKeyResponse{uatKey("PG03", "SS400", uatPG03)}},
	}

	SortSubGroupsByValue(subs, "PG03")

	if subs[0].SubGroupKeys[0].ValueName != "SS400" || subs[1].SubGroupKeys[0].ValueName != "LT" {
		t.Fatalf("ต้องได้ SS400 แล้ว LT แต่ได้ %q, %q",
			subs[0].SubGroupKeys[0].ValueName, subs[1].SubGroupKeys[0].ValueName)
	}
}

// ลำดับ tab ต้องเรียงด้วยค่าของ PG02
func TestGolden_TabOrder(t *testing.T) {
	labels := []string{"เหล็กแผ่น", "เหล็กแผ่นตัด SIZE", "แผ่นลาย", "เหล็กแผ่น special"}

	subs := make([]models.PriceListSubGroupResponse, 0, len(labels))
	for i, l := range labels {
		subs = append(subs, uatSubGroup(fmt.Sprintf("sg-%d", i), l, "4' x 8'", "10"))
	}

	sortLabelsByValue(labels, subs, "PG02")

	want := []string{"เหล็กแผ่น", "แผ่นลาย", "เหล็กแผ่น special", "เหล็กแผ่นตัด SIZE"}
	for i := range want {
		if labels[i] != want[i] {
			t.Fatalf("tab ที่ %d ได้ %q ต้องเป็น %q — ทั้งหมด %v", i, labels[i], want[i], labels)
		}
	}
}
