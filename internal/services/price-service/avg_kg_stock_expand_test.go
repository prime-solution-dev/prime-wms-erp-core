//go:build integration

package priceService

import (
	"testing"

	"github.com/google/uuid"
)

// เทสนี้พิสูจน์บั๊ก: `if inv.TotalQty > 0` / `if inv.TotalWeight > 0` ใน
// transformToGetPriceListResponse (จุด map "new API fields to existing model fields")
// ปล่อยให้ SumQty/SumWeight ค้างค่าจาก field เก่า (sum_qty/sum_weight) ที่ JSON
// ของ inventory_weight ส่งมาตรง ๆ เมื่อ batch นั้นมี total_qty/total_weight = 0 จริง
//
// หมายเหตุสำคัญ (ต่างจากสมมติฐานเดิมของแผน): "subgroup ต้นแบบ" (sg ใน
// transformToGetPriceListResponse บรรทัดที่ทำ `expandedSG := sg`) ไม่มีทางมี
// InventoryWeight ไม่เป็น 0 ได้เลย เพราะการสร้าง subGroup (models.PriceListSubGroupResponse)
// ที่บรรทัดก่อนหน้าไม่เคยคัดลอก InventoryWeight มาจาก input SubGroup (get-pricelist.go)
// เลยสักครั้ง — ต่อให้ buildSingleSubGroupResponse ตั้ง SubGroup.InventoryWeight ไว้ ค่านั้น
// ก็จะถูกทิ้งไปตั้งแต่ก่อนถึง loop expand ไม่มีทางไหลเข้ามาถึงจุดบั๊กได้
//
// ค่าที่ "ค้าง" จริง ๆ ที่พิสูจน์ได้คือค่าจาก field เก่า (sum_qty/sum_weight) ที่ inv เอง
// ได้รับมาจาก JSON โดยตรง (models.InventoryWeightResponse มีทั้ง field เก่าและใหม่) —
// เทสด้านล่างจำลองสถานการณ์นี้ด้วยการส่ง sum_qty/sum_weight ที่ไม่ตรงกับ total_qty/
// total_weight มาใน JSON ของ batch ที่สอง เพื่อพิสูจน์ว่าโค้ดใหม่ต้อง overwrite ทับเสมอ
// ไม่ว่า total_qty/total_weight จะเป็น 0 หรือไม่ก็ตาม

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
