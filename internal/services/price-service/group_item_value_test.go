package priceService

import (
	"testing"

	"prime-erp-core/internal/models"
)

func TestParseGroupItemValue(t *testing.T) {
	m := map[string]models.GetGroupItemResponse{
		"PG05_22": {ItemCode: "PG05_22", ItemName: "1250x8'", Value: "10,000.00"},
		"PG05_3":  {ItemCode: "PG05_3", ItemName: "4' x 8'", Value: "32.00"},
		"PG06_4":  {ItemCode: "PG06_4", ItemName: "1.2", Value: "1.20"},
		"PG08_5":  {ItemCode: "PG08_5", ItemName: "", Value: "0"},
		"PG05_40": {ItemCode: "PG05_40", ItemName: "ESP", Value: ""},
		"PG09_1":  {ItemCode: "PG09_1", ItemName: "N/A", Value: "ไม่ระบุ"},
	}

	cases := []struct {
		name    string
		code    string
		wantVal float64
		wantHas bool
	}{
		{"thousand separator ต้องถูกตัดก่อน parse", "PG05_22", 10000, true},
		{"ค่าธรรมดา", "PG05_3", 32, true},
		{"ทศนิยม", "PG06_4", 1.2, true},
		{"value = 0 จริง ต้องนับว่ามีค่า", "PG08_5", 0, true},
		{"value ว่าง ต้องนับว่าไม่มีค่า", "PG05_40", 0, false},
		{"value ที่ parse ไม่ได้ ต้องนับว่าไม่มีค่า", "PG09_1", 0, false},
		{"ไม่มี record ต้องนับว่าไม่มีค่า", "ไม่มีจริง", 0, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotVal, gotHas := parseGroupItemValue(m, c.code)
			if gotVal != c.wantVal || gotHas != c.wantHas {
				t.Fatalf("parseGroupItemValue(%q) = (%v, %v) ต้องเป็น (%v, %v)",
					c.code, gotVal, gotHas, c.wantVal, c.wantHas)
			}
		})
	}
}

func TestParseGroupItemValue_NilMap(t *testing.T) {
	gotVal, gotHas := parseGroupItemValue(nil, "PG05_22")
	if gotVal != 0 || gotHas {
		t.Fatalf("map เป็น nil ต้องได้ (0, false) แต่ได้ (%v, %v)", gotVal, gotHas)
	}
}
