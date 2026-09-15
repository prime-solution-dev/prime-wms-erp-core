//go:build integration

package priceService

import (
	"testing"

	"github.com/google/uuid"
)

// เทสชุดนี้คุมเส้นทางของรหัสคลังตั้งแต่ response ของ warehouse-core จนถึง subgroup
// ที่ฝั่ง build row จะอ่านไปแสดงในคอลัมน์ "โกดัง" / "Stock"
//
// ก่อนหน้านี้คอลัมน์นี้ว่างเสมอทุก pattern เพราะแหล่งข้อมูลเดียวคือ udf_json ซึ่งไม่มี
// โค้ดไหนเขียนลงไปเลย ตอนนี้ค่าจริงมาจากตาราง inventory ผ่าน field warehouse_code
//
// ใช้ harness เดียวกับ avg_kg_stock_expand_test.go (fakeWarehouseServer +
// pointWarehouseEndpointAt) จึงไม่ parallel-safe เช่นกัน ห้ามใส่ t.Parallel()

func TestGetPriceDetailCarriesWarehouseCodePerBatch(t *testing.T) {
	ensureGroupPaymentTablesForTest(t)

	subGroupID := uuid.New()
	// batch เดียวกันกระจายอยู่ 2 คลัง warehouse-core แบ่งยอดตามสัดส่วนมาให้แล้ว
	// และ batch ที่สองอยู่คลังเดียว
	body := `[{
		"id": "` + subGroupID.String() + `",
		"product_code": "P001",
		"supplier_code": "SUP001",
		"supplier_name": "Supplier 1",
		"weight_spec": 12.5,
		"inventory_weight": [
			{"batch_no": "A-1509", "warehouse_code": "07", "total_qty": 75, "total_weight": 1455},
			{"batch_no": "A-1509", "warehouse_code": "09", "total_qty": 25, "total_weight": 485},
			{"batch_no": "B-1509", "warehouse_code": "07", "total_qty": 120, "total_weight": 2328}
		]
	}]`
	srv := fakeWarehouseServer(t, body)
	defer srv.Close()
	pointWarehouseEndpointAt(t, srv.URL)

	result, err := transformToGetPriceListResponse(buildSingleSubGroupResponse(subGroupID))
	if err != nil {
		t.Fatalf("transformToGetPriceListResponse: %v", err)
	}

	if len(result) != 1 || len(result[0].SubGroups) != 3 {
		t.Fatalf("ต้องได้ 1 group กับ 3 subgroup (1 ต่อ batch+คลัง) ได้ %d group", len(result))
	}

	type row struct{ batch, warehouse string }
	got := make([]row, 0, 3)
	for _, sg := range result[0].SubGroups {
		got = append(got, row{sg.BatchNo, sg.WarehouseCode})

		// แต่ละ subgroup ต้องถือ inventory ของตัวเองใบเดียว ไม่งั้นฝั่ง build row
		// ที่อ่าน InventoryWeight[0] จะหยิบของแถวอื่นมาแสดง
		if len(sg.InventoryWeight) != 1 {
			t.Fatalf("batch %s คลัง %s: ต้องมี InventoryWeight ใบเดียว ได้ %d",
				sg.BatchNo, sg.WarehouseCode, len(sg.InventoryWeight))
		}
		if sg.InventoryWeight[0].WarehouseCode != sg.WarehouseCode {
			t.Errorf("คลังใน InventoryWeight ไม่ตรงกับที่ copy ขึ้น subgroup: %q vs %q",
				sg.InventoryWeight[0].WarehouseCode, sg.WarehouseCode)
		}
	}

	want := []row{{"A-1509", "07"}, {"A-1509", "09"}, {"B-1509", "07"}}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("แถว %d ได้ %+v ต้องเป็น %+v", i, got[i], want[i])
		}
	}
}

// response ที่ไม่มี warehouse_code (สินค้าที่มีน้ำหนักบันทึกไว้แต่ไม่มีแถวใน inventory)
// ต้องไม่ทำให้แถวหาย แค่คอลัมน์คลังว่างเหมือนพฤติกรรมเดิม
func TestGetPriceDetailKeepsRowWhenWarehouseUnknown(t *testing.T) {
	ensureGroupPaymentTablesForTest(t)

	subGroupID := uuid.New()
	body := `[{
		"id": "` + subGroupID.String() + `",
		"product_code": "P001",
		"weight_spec": 12.5,
		"inventory_weight": [
			{"batch_no": "A-1509", "total_qty": 100, "total_weight": 1940}
		]
	}]`
	srv := fakeWarehouseServer(t, body)
	defer srv.Close()
	pointWarehouseEndpointAt(t, srv.URL)

	result, err := transformToGetPriceListResponse(buildSingleSubGroupResponse(subGroupID))
	if err != nil {
		t.Fatalf("transformToGetPriceListResponse: %v", err)
	}

	if len(result) != 1 || len(result[0].SubGroups) != 1 {
		t.Fatalf("ต้องได้ 1 subgroup ได้ %#v", result)
	}
	sg := result[0].SubGroups[0]
	if sg.WarehouseCode != "" {
		t.Errorf("ไม่มีข้อมูลคลังต้องว่าง ได้ %q", sg.WarehouseCode)
	}
	if sg.BatchNo != "A-1509" {
		t.Errorf("แถวต้องยังอยู่พร้อม batch เดิม ได้ %q", sg.BatchNo)
	}
}
