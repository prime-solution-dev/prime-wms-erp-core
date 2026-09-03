package priceService

import (
	"encoding/json"
	"prime-erp-core/internal/models"
	"testing"
	"time"

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

	// Note: UDF columns are now fixed list, so count may vary
	if len(resp.Columns) < 3 {
		t.Fatalf("expected at least 3 group columns, got %d", len(resp.Columns))
	}
	// static column มาก่อน dynamic group key เสมอ จึงเช็กว่ามีคอลัมน์อยู่ ไม่เช็กลำดับ
	var productGroup1 *ExportColumn
	for i := range resp.Columns {
		if resp.Columns[i].Field == "PRODUCT_GROUP1" {
			productGroup1 = &resp.Columns[i]
			break
		}
	}
	if productGroup1 == nil {
		t.Fatal("expected a PRODUCT_GROUP1 column")
	}
	if productGroup1.HeaderName != "หมวดหลัก" {
		t.Fatalf("unexpected PRODUCT_GROUP1 header: %q", productGroup1.HeaderName)
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

	if row["is_highlight"] != true {
		t.Fatalf("expected is_highlight true, got %v", row["is_highlight"])
	}
	// json.Unmarshal to interface{} makes numbers float64
	if row["stock"] != float64(123) {
		t.Fatalf("expected stock 123, got %v", row["stock"])
	}
}

func TestBuildBasedPriceTab_Structure(t *testing.T) {
	gID := uuid.New()
	groups := []GetPriceListGroupResponse{
		{
			PriceListGroup: PriceListGroup{
				ID:          gID,
				GroupCode:   "GROUP_1_ITEM_1",
				PriceWeight: 23.4,
				Terms: []PriceListGroupTerm{
					{
						TermCode:   "T1",
						Pdc:        0.23,
						PdcPercent: 1,
						Due:        0.35,
						DuePercent: 1.5,
					},
				},
			},
		},
	}

	groupMap := map[string]models.GetGroupResponse{
		"GROUP_1_ITEM_1": {
			GroupCode: "GROUP_1_ITEM_1",
			GroupName: "หมวดเหล็กแผ่น",
		},
	}

	paymentTermMap := map[string]GetPaymentTermResponse{
		"T1": {
			TermCode: "T1",
			TermName: "15/30",
		},
	}

	var _ map[string]models.GetGroupResponse = groupMap
	tab := buildBasedPriceTab(groups, paymentTermMap)

	if tab.Name != "Based price" {
		t.Fatalf("expected tab name 'Based price', got %s", tab.Name)
	}

	if tab.Headers.Report != "Pricelist- Based price" {
		t.Fatalf("expected report header 'Pricelist- Based price', got %s", tab.Headers.Report)
	}

	if len(tab.Columns) < 3 {
		t.Fatalf("expected at least 3 columns (product, price_pr, cash_pr), got %d", len(tab.Columns))
	}

	if tab.Columns[0].Field != "product" {
		t.Fatalf("expected first column field 'product', got %s", tab.Columns[0].Field)
	}

	if len(tab.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(tab.Rows))
	}

	row := tab.Rows[0]

	if row["price_pr"] != 23.4 {
		t.Fatalf("expected price_pr 23.4, got %v", row["price_pr"])
	}

	if row["cash_pr"] != 23.4 {
		t.Fatalf("expected cash_pr 23.4, got %v", row["cash_pr"])
	}

	// Check term fields
	if row["term_T1_pdc_baht"] != 0.23 {
		t.Fatalf("expected term_T1_pdc_baht 0.23, got %v", row["term_T1_pdc_baht"])
	}
}

// update_dtm ถูกเขียนเป็น UTC (update-pricelist.go:27) และ container เป็น alpine
// ที่ไม่มี tzdata เวลาที่แสดงในรายงานจึงต้องถูกแปลงเป็นเวลาไทยอย่างชัดเจน
func TestFormatTimestamp_ConvertsToBangkok(t *testing.T) {
	got := formatTimestamp(time.Date(2026, 9, 3, 2, 30, 0, 0, time.UTC))
	if got != "3/9/2026 09:30" {
		t.Fatalf("expected %q, got %q", "3/9/2026 09:30", got)
	}
}

// export ก่อน 07:00 น. เวลาไทย ต้องไม่ทำให้วันที่ย้อนไปหนึ่งวัน
func TestFormatTimestamp_CrossesDateBoundary(t *testing.T) {
	got := formatTimestamp(time.Date(2026, 9, 2, 20, 0, 0, 0, time.UTC))
	if got != "3/9/2026 03:00" {
		t.Fatalf("expected %q, got %q", "3/9/2026 03:00", got)
	}
}

// group key code และ UDF key ชนกับ static column ได้ ต้องไม่ทำให้คอลัมน์ซ้ำ
func TestBuildExportTableTyped_NoDuplicateColumns(t *testing.T) {
	udfJson, err := json.Marshal(map[string]interface{}{
		"line_bundle": 10,
		"coil_id":     "C-001",
	})
	if err != nil {
		t.Fatalf("failed to marshal udf json: %v", err)
	}

	groups := []GetPriceListGroupResponse{
		{
			PriceListGroup: PriceListGroup{
				ID:        uuid.New(),
				GroupCode: "G1",
				SubGroups: []SubGroup{
					{
						ID:      uuid.New(),
						UdfJson: udfJson,
						GroupKeys: []GroupKey{
							{Code: "PG01", Value: "V1", Seq: 1},
							{Code: "PG02", Value: "V2", Seq: 2},
						},
					},
				},
			},
		},
	}

	data := buildExportTableTyped(groups, func(string) string { return "" }, func(string) string { return "" })

	count := map[string]int{}
	for _, c := range data.Columns {
		count[c.Field]++
	}
	for field, n := range count {
		if n > 1 {
			t.Fatalf("column %q appears %d times, expected exactly once", field, n)
		}
	}
}

// is_highlight ใช้ทำสีตัวอักษรฝั่ง document-core จึงต้องอยู่ใน row แต่ไม่ใช่คอลัมน์
func TestBuildExportTableTyped_DropsIsHighlightColumnButKeepsValue(t *testing.T) {
	udfJson, err := json.Marshal(map[string]interface{}{"is_highlight": true})
	if err != nil {
		t.Fatalf("failed to marshal udf json: %v", err)
	}

	groups := []GetPriceListGroupResponse{
		{
			PriceListGroup: PriceListGroup{
				ID:        uuid.New(),
				GroupCode: "G1",
				SubGroups: []SubGroup{
					{ID: uuid.New(), UdfJson: udfJson},
				},
			},
		},
	}

	data := buildExportTableTyped(groups, func(string) string { return "" }, func(string) string { return "" })

	for _, c := range data.Columns {
		if c.Field == "is_highlight" {
			t.Fatal("is_highlight must not be an exported column")
		}
	}

	if len(data.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(data.Rows))
	}
	if data.Rows[0]["is_highlight"] != true {
		t.Fatalf("is_highlight value must survive for cell styling, got %v", data.Rows[0]["is_highlight"])
	}
}

// stock_quantity (TotalQty) กับ quantity (SumQty) เคยใช้หัวเดียวกันคือ "จำนวน"
func TestBuildExportTableTyped_QuantityHeadersAreDistinct(t *testing.T) {
	data := buildExportTableTyped(nil, func(string) string { return "" }, func(string) string { return "" })

	headers := map[string]string{}
	for _, c := range data.Columns {
		headers[c.Field] = c.HeaderName
	}

	if headers["stock_quantity"] != "จำนวนรวม" {
		t.Fatalf("expected stock_quantity header %q, got %q", "จำนวนรวม", headers["stock_quantity"])
	}
	if headers["quantity"] != "จำนวน" {
		t.Fatalf("expected quantity header %q, got %q", "จำนวน", headers["quantity"])
	}
}

// group_name ที่ client ตั้งไว้ใน DB ต้องชนะหัวคอลัมน์ hardcode ของ PG01-PG10
func TestBuildExportTableTyped_GroupNameOverridesStaticHeader(t *testing.T) {
	groups := []GetPriceListGroupResponse{
		{
			PriceListGroup: PriceListGroup{
				ID:        uuid.New(),
				GroupCode: "G1",
				SubGroups: []SubGroup{
					{
						ID: uuid.New(),
						GroupKeys: []GroupKey{
							{Code: "PG01", Value: "V1", Seq: 1},
							{Code: "PG04", Value: "V4", Seq: 2},
						},
					},
				},
			},
		},
	}

	groupNameByCode := func(code string) string {
		switch code {
		case "PG01":
			return "กลุ่มสินค้าตามที่ลูกค้าตั้ง"
		case "PG04":
			return "กลุ่มที่สี่"
		default:
			return ""
		}
	}

	data := buildExportTableTyped(groups, groupNameByCode, func(string) string { return "" })

	headers := map[string]string{}
	count := map[string]int{}
	for _, c := range data.Columns {
		headers[c.Field] = c.HeaderName
		count[c.Field]++
	}

	// PG01 มีใน static list -> ต้องถูกเขียนทับด้วยชื่อจาก DB ไม่ใช่ "หมวดหลัก"
	if headers["PG01"] != "กลุ่มสินค้าตามที่ลูกค้าตั้ง" {
		t.Fatalf("expected PG01 header from DB, got %q", headers["PG01"])
	}
	// PG04 ไม่มีใน static list -> ต้องถูกเพิ่มเข้ามาพร้อมชื่อจาก DB
	if headers["PG04"] != "กลุ่มที่สี่" {
		t.Fatalf("expected PG04 header from DB, got %q", headers["PG04"])
	}
	// เขียนทับ ไม่ใช่เพิ่มคอลัมน์ใหม่
	if count["PG01"] != 1 {
		t.Fatalf("expected exactly one PG01 column, got %d", count["PG01"])
	}
}

// DB ไม่มีชื่อกลุ่ม -> ต้องคงหัว static ไว้ ไม่ใช่กลายเป็นค่าว่างหรือรหัสดิบ
func TestBuildExportTableTyped_StaticHeaderIsFallback(t *testing.T) {
	groups := []GetPriceListGroupResponse{
		{
			PriceListGroup: PriceListGroup{
				ID:        uuid.New(),
				GroupCode: "G1",
				SubGroups: []SubGroup{
					{
						ID:        uuid.New(),
						GroupKeys: []GroupKey{{Code: "PG01", Value: "V1", Seq: 1}},
					},
				},
			},
		},
	}

	data := buildExportTableTyped(groups, func(string) string { return "" }, func(string) string { return "" })

	for _, c := range data.Columns {
		if c.Field == "PG01" {
			if c.HeaderName != "หมวดหลัก" {
				t.Fatalf("expected static fallback header %q, got %q", "หมวดหลัก", c.HeaderName)
			}
			return
		}
	}
	t.Fatal("expected a PG01 column")
}
