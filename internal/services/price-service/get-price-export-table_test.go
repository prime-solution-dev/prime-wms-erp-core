package priceService

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestBuildExportTableTyped_ColumnsAndRows(t *testing.T) {
	gID := uuid.New()
	sgID := uuid.New()

	udfJson, err := json.Marshal(map[string]interface{}{
		"is_highlight": true,
		"stock":        123,
	})
	if err != nil {
		t.Fatalf("failed to marshal udf json: %v", err)
	}

	groups := []GetPriceListGroupResponse{
		{
			PriceListGroup: PriceListGroup{
				ID:        gID,
				GroupCode: "GROUP_1_ITEM_1",
				SubGroups: []SubGroup{
					{
						ID:          sgID,
						SubGroupKey: "GROUP_1_ITEM_1|GROUP_4_ITEM_1|GROUP_6_ITEM_1",
						IsTrading:   false,
						PriceUnit:   10,
						PriceWeight: 1.23,
						UdfJson:     udfJson,
						GroupKeys: []GroupKey{
							{Code: "PRODUCT_GROUP1", Value: "GROUP_1_ITEM_1", Seq: 1},
							{Code: "PRODUCT_GROUP4", Value: "GROUP_4_ITEM_1", Seq: 2},
							{Code: "PRODUCT_GROUP6", Value: "GROUP_6_ITEM_1", Seq: 3},
						},
					},
				},
			},
		},
	}

	groupNameByCode := func(code string) string {
		switch code {
		case "PRODUCT_GROUP1":
			return "หมวดหลัก"
		case "PRODUCT_GROUP4":
			return "ขนาด"
		case "PRODUCT_GROUP6":
			return "หนา"
		default:
			return ""
		}
	}

	itemNameByCode := func(code string) string {
		switch code {
		case "GROUP_1_ITEM_1":
			return "หมวดเหล็กแผ่น"
		case "GROUP_4_ITEM_1":
			return "75x45x15"
		case "GROUP_6_ITEM_1":
			return "1.2"
		default:
			return ""
		}
	}

	resp := buildExportTableTyped(groups, groupNameByCode, itemNameByCode)

	if len(resp.Columns) != 6 {
		t.Fatalf("expected 6 columns (3 group + udf_json + 2 udf), got %d", len(resp.Columns))
	}
	if resp.Columns[0].Field != "PRODUCT_GROUP1" || resp.Columns[0].HeaderName != "หมวดหลัก" {
		t.Fatalf("unexpected first column: %+v", resp.Columns[0])
	}

	if len(resp.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(resp.Rows))
	}
	row := resp.Rows[0]

	if row["id"] != sgID.String() {
		t.Fatalf("expected id %s, got %v", sgID.String(), row["id"])
	}
	if row["PRODUCT_GROUP1"] != "หมวดเหล็กแผ่น" {
		t.Fatalf("expected PRODUCT_GROUP1 value_name, got %v", row["PRODUCT_GROUP1"])
	}
	if row["PRODUCT_GROUP4"] != "75x45x15" {
		t.Fatalf("expected PRODUCT_GROUP4 value_name, got %v", row["PRODUCT_GROUP4"])
	}
	if row["PRODUCT_GROUP6"] != "1.2" {
		t.Fatalf("expected PRODUCT_GROUP6 value_name, got %v", row["PRODUCT_GROUP6"])
	}

	if row["udf_json"] == "" {
		t.Fatalf("expected udf_json to be present, got empty")
	}
	if row["is_highlight"] != true {
		t.Fatalf("expected is_highlight true, got %v", row["is_highlight"])
	}
	// json.Unmarshal to interface{} makes numbers float64
	if row["stock"] != float64(123) {
		t.Fatalf("expected stock 123, got %v", row["stock"])
	}
}
