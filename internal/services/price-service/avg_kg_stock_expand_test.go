//go:build integration

package priceService

import (
	"testing"

	"github.com/google/uuid"
)

// เทสนี้พิสูจน์การกันความเสี่ยงเชิงโครงสร้าง ไม่ใช่บั๊กที่ active อยู่ในปัจจุบัน:
// `if inv.TotalQty > 0` / `if inv.TotalWeight > 0` เดิมใน transformToGetPriceListResponse
// (จุด map "new API fields to existing model fields") เขียนทับ SumQty/SumWeight แบบมีเงื่อนไข
// ซึ่งถ้า JSON ของ inventory_weight มี field เก่า (sum_qty/sum_weight) ปนมาด้วย ค่าเก่านั้นจะ
// ค้างอยู่เมื่อ batch นั้นมี total_qty/total_weight = 0 จริง — ปัจจุบัน endpoint
// get-inventory-weight-by-key ที่ price-service เรียกจริง (warehouse-core
// internal/services/inventory-service/get-inventory-weight-by-key.go) ไม่ส่ง field เก่าพวกนี้
// มาเลย field เหล่านั้นจึงเป็น 0 เสมอในปัจจุบัน และโค้ดเดิมกับโค้ดใหม่ให้ผลเหมือนกัน
// ใน production ตอนนี้ — เทสนี้จึงพิสูจน์ความปลอดภัยเชิงโครงสร้าง (ถ้า endpoint เปลี่ยนหรือ
// struct ถูกใช้ซ้ำกับ endpoint อื่นในอนาคต) ไม่ใช่การพิสูจน์บั๊กที่เกิดขึ้นจริงตอนนี้
//
// หมายเหตุสำคัญ (ต่างจากสมมติฐานเดิมของแผน): "subgroup ต้นแบบ" (sg ใน
// transformToGetPriceListResponse บรรทัดที่ทำ `expandedSG := sg`) ไม่มีทางมี
// InventoryWeight ไม่เป็น 0 ได้เลย เพราะการสร้าง subGroup (models.PriceListSubGroupResponse)
// ที่บรรทัดก่อนหน้าไม่เคยคัดลอก InventoryWeight มาจาก input SubGroup (get-pricelist.go)
// เลยสักครั้ง — ต่อให้ buildSingleSubGroupResponse ตั้ง SubGroup.InventoryWeight ไว้ ค่านั้น
// ก็จะถูกทิ้งไปตั้งแต่ก่อนถึง loop expand ไม่มีทางไหลเข้ามาถึงจุดบั๊กได้
//
// กลไกที่พิสูจน์ได้จริงคือ field เก่า (sum_qty/sum_weight) ที่ inv จะได้รับถ้า JSON
// ส่งมา (models.InventoryWeightResponse มีทั้ง field เก่าและใหม่ใน struct เดียวกัน) —
// เทสด้านล่าง "จำลอง" สถานการณ์นี้ด้วยการส่ง sum_qty/sum_weight เองในมือ (ไม่ใช่รูปแบบ
// ที่ warehouse-core ส่งจริงในปัจจุบัน — get-inventory-weight-by-key ไม่มี field เหล่านี้
// ใน response เลย) เพื่อพิสูจน์ว่าโค้ดใหม่ overwrite ทับเสมอไม่ว่า total_qty/total_weight
// จะเป็น 0 หรือไม่ก็ตาม เผื่อวันหน้า endpoint เปลี่ยนหรือ struct ถูกใช้ซ้ำที่อื่น

func TestGetPriceDetailZeroValuesAreNotStale(t *testing.T) {
	ensureGroupPaymentTablesForTest(t)

	subGroupID := uuid.New()
	// batch B1: ค่าปกติ ไม่เป็น 0 ทุก field (sum_* ตรงกับ total_*)
	// batch B2: total_qty/total_weight = 0 จริง แต่ JSON ยังส่ง sum_qty/sum_weight ค้างมา
	// ไม่เป็น 0 (จำลอง field เก่าที่ backend ยังส่งมาเผื่อ compat) — ต้องถูก overwrite เป็น 0
	body := `[{
		"id": "` + subGroupID.String() + `",
		"product_code": "P001",
		"supplier_code": "SUP001",
		"supplier_name": "Supplier 1",
		"weight_spec": 12.5,
		"inventory_weight": [
			{"batch_no": "B1", "total_weight": 1500000, "total_qty": 100, "sum_weight": 1500000, "sum_qty": 100},
			{"batch_no": "B2", "total_weight": 0, "total_qty": 0, "sum_weight": 999999, "sum_qty": 888}
		]
	}]`
	srv := fakeWarehouseServer(t, body)
	defer srv.Close()
	pointWarehouseEndpointAt(t, srv.URL)

	result, err := transformToGetPriceListResponse(buildSingleSubGroupResponse(subGroupID))
	if err != nil {
		t.Fatalf("transformToGetPriceListResponse: %v", err)
	}

	if len(result) != 1 || len(result[0].SubGroups) != 2 {
		t.Fatalf("expected 1 group with 2 expanded subgroups (1 per batch), got %#v", result)
	}

	var b2Found bool
	for _, sg := range result[0].SubGroups {
		if len(sg.InventoryWeight) != 1 || sg.InventoryWeight[0].BatchNo != "B2" {
			continue
		}
		b2Found = true
		iw := sg.InventoryWeight[0]
		if iw.SumWeight != 0 {
			t.Fatalf("SumWeight = %v, want 0 (batch B2's total_weight is 0; must not leak stale sum_weight=999999 from JSON)", iw.SumWeight)
		}
		if iw.SumQty != 0 {
			t.Fatalf("SumQty = %v, want 0 (batch B2's total_qty is 0; must not leak stale sum_qty=888 from JSON)", iw.SumQty)
		}
	}
	if !b2Found {
		t.Fatalf("subgroup for batch B2 not found in result: %#v", result[0].SubGroups)
	}
}

func TestGetPriceDetailDoesNotWriteAvgBatch(t *testing.T) {
	ensureGroupPaymentTablesForTest(t)

	subGroupID := uuid.New()
	body := `[{
		"id": "` + subGroupID.String() + `",
		"product_code": "P001",
		"supplier_code": "SUP001",
		"supplier_name": "Supplier 1",
		"weight_spec": 12.5,
		"inventory_weight": [
			{"batch_no": "B1", "total_weight": 1500000, "total_qty": 100, "avg_weight": 15000}
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
		t.Fatalf("expected 1 group with 1 subgroup, got %#v", result)
	}

	iw := result[0].SubGroups[0].InventoryWeight
	if len(iw) != 1 {
		t.Fatalf("expected 1 inventory weight entry, got %#v", iw)
	}
	if iw[0].AvgWeight != 15000 {
		t.Fatalf("AvgWeight = %v, want 15000 (must come from avg_weight)", iw[0].AvgWeight)
	}
	if iw[0].AvgBatch != 0 {
		t.Fatalf("AvgBatch = %v, want 0 (AvgBatch must not be written anymore — no Go reader for it)", iw[0].AvgBatch)
	}
}
