# Avg. kg stock ระดับ site และการแก้สูตรราคา — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** ทำให้ `Avg. kg stock` เป็นค่าเฉลี่ยน้ำหนักระดับ product ต่อ site ที่ deterministic และทำให้สูตรคำนวณราคาอ่านค่าที่ถูกประเภทตามลำดับ dependency ที่ถูกต้อง

**Architecture:** `warehouse-core` คงการรวมยอดระดับ batch ไว้ (pattern ที่แสดงคอลัมน์ `batch_no` ต้องใช้) แล้วเติม field `avg_product` ที่เป็นค่าเฉลี่ยระดับ `company|site|product` ลงทุก entry พร้อม sort ผลลัพธ์ให้นิ่ง ฝั่ง `erp-core` เลือกอ่าน `AvgProduct` หรือ `AvgWeight` ตามว่า pattern นั้นแสดง batch หรือไม่ ส่วนสูตรราคาเปลี่ยนไปอ่านค่า running `total_net_price_unit` / `total_net_price_weight` และประเมินสูตรตามลำดับ dependency

**Tech Stack:** Go 1.22 (warehouse-core) / Go (erp-core), Gin, GORM + sqlx, PostgreSQL, `expr-lang/expr` (expression engine), testify-free table-driven tests, testcontainers สำหรับ integration test

**Spec:** `docs/superpowers/specs/2026-09-10-pricelist-avg-kg-stock-site-level-design.md`

---

## File Structure

### `prime-wms-warehouse-core`

| ไฟล์ | ความรับผิดชอบ |
|---|---|
| `internal/services/inventory-service/get-inventory-weight-by-key.go` (modify) | เพิ่ม field `AvgProduct` ใน `InventoryWeightDetail` · แยก logic รวมยอดออกเป็นฟังก์ชัน pure `aggregateInventoryWeights` · sort ผลลัพธ์ |
| `internal/services/inventory-service/aggregate_inventory_weights_test.go` (create) | unit test ของฟังก์ชัน pure ตัวใหม่ |

### `prime-wms-erp-core`

| ไฟล์ | ความรับผิดชอบ |
|---|---|
| `internal/services/price-service/patterns/shared.go` (modify) | แยกฟังก์ชันอ่าน avg เป็นระดับ site / ระดับ batch · เพิ่ม `patternHasBatchColumn` |
| `internal/services/price-service/patterns/avg_kg_stock_test.go` (create) | unit test ของฟังก์ชันอ่าน avg และ `patternHasBatchColumn` |
| `internal/services/price-service/formula_order.go` (create) | เรียงสูตรตาม dependency (F4) — แยกไฟล์เพราะเป็นตรรกะเดี่ยวที่ทดสอบได้เอง |
| `internal/services/price-service/formula_order_test.go` (create) | unit test ของการเรียงลำดับสูตร |
| `internal/services/price-service/update-latest-pricelist-subgroup.go` (modify) | `avgKgStock` อ่าน `AvgProduct` · `Pcs`/`Kg` เป็นค่า running · เรียกตัวเรียงลำดับสูตร |
| `internal/services/price-service/get-calculated-pricelist-subgroup.go` (modify) | เหมือน `update-latest` แต่ไม่บันทึก |
| `internal/services/price-service/avg_kg_stock_price_test.go` (create) | unit test ของการคำนวณราคาด้วย avg และค่า running |
| `internal/services/price-service/get-price-detail.go` (modify) | ตัดเงื่อนไข `if inv.X > 0` · ลบ dead write `AvgBatch` |
| `internal/services/price-service/get-price-export-table.go` (modify) | `avg_weight` อ่าน `AvgProduct` · เติม `0` เมื่อไม่มีสต็อก |
| `internal/services/price-service/build-pricelist-detail-tab.go` (modify) | `avg_weight` เป็น `0` ไม่ใช่ string ว่าง · อ่าน `AvgProduct` |
| `internal/services/price-service/avg_kg_stock_rowbuilder_test.go` (create) | unit test ของ row builder ฝั่ง export และ detail report |
| `internal/services/price-service/avg_kg_stock_flow_test.go` (create) | test ของ flow คำนวณราคาทั้งเส้นโดย inject dependency ไม่พึ่ง DB |
| `internal/services/price-service/avg_kg_stock_integration_test.go` (create) | integration test ของเส้นทางการแสดงผลผ่าน fake warehouse server |
| `migrations/2026-09-10-fix-pcs-weight-spec-expression.sql` (create) | แก้ expression ที่ผิด (F3) |
| `internal/scripts/price_list_formulas/price-list-formulas.json` (modify) | แก้ expression ให้ตรงกับ migration |
| `docs/superpowers/reports/2026-09-10-f5-subgroup-formula-pairing.sql` (create) | query รายงาน subgroup ที่ผูกสูตรไม่ครบคู่ (F5) |

---

## Task 1: แตก branch ใน `prime-wms-warehouse-core`

**Files:** ไม่มีไฟล์ถูกแก้

- [ ] **Step 1: ตรวจสถานะ repo ปัจจุบัน**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-warehouse-core
git status --short --branch
```

Expected: แสดง branch ปัจจุบัน อาจเป็น `feat/pricelist-weight-spec-from-product-master` และอาจมี `M cmd/.env`

`cmd/.env` เป็น config ของเครื่อง local **ห้าม commit และห้าม checkout ทับ** ปล่อยไว้เฉย ๆ

- [ ] **Step 2: fetch และแตก branch ใหม่จาก Develop**

```bash
git fetch origin
git checkout -b feat/pricelist-avg-kg-stock-site-level origin/Develop
```

Expected: `Switched to a new branch 'feat/pricelist-avg-kg-stock-site-level'`

ถ้า checkout ล้มเพราะ `cmd/.env` ให้ `git stash push cmd/.env` ก่อน แล้ว `git stash pop` หลัง checkout

- [ ] **Step 3: ยืนยันว่า branch ถูกต้อง**

```bash
git status --short --branch | head -2
```

Expected: `## feat/pricelist-avg-kg-stock-site-level...origin/Develop`

**ห้ามทำงานบน `Develop` และห้ามแตะ branch** `Crossmax-uat`, `shi-sit`, `Pacifica-uat`, `Pacifica-main`, `Thaimetal-uat`, `shi-main`

---

## Task 2: `warehouse-core` — เพิ่ม `AvgProduct` และทำผลลัพธ์ให้ deterministic

**Files:**
- Modify: `internal/services/inventory-service/get-inventory-weight-by-key.go:32-41` (struct) และ `:299-350` (logic รวมยอด)
- Test: `internal/services/inventory-service/aggregate_inventory_weights_test.go` (create)

บริบท: ฟังก์ชันเดิม `getAggregatedInventoryWeights` (บรรทัด 271) รับ `*gin.Context` และต่อ DB จึงทดสอบตรงไม่ได้
แผนนี้แยก logic รวมยอดออกมาเป็นฟังก์ชัน pure ก่อน แล้วให้ตัวเดิมเรียกใช้

ฟังก์ชันใหม่อยู่ package `inventoryService` เดียวกับ `safeAverage` (นิยามที่ `get-inventory-weight.go:64-75`) จึงเรียกได้เลย

- [ ] **Step 1: เพิ่ม field `AvgProduct` ใน struct**

แก้ `internal/services/inventory-service/get-inventory-weight-by-key.go` บรรทัด 32-41 จาก

```go
type InventoryWeightDetail struct {
	Key          string  `json:"key"`
	ProductCode  string  `json:"product_code"`
	BatchNo      string  `json:"batch_no"`
	SupplierCode string  `json:"supplier_code"`
	SupplierName string  `json:"supplier_name"`
	TotalWeight  float64 `json:"total_weight"`
	TotalQty     float64 `json:"total_qty"`
	AvgWeight    float64 `json:"avg_weight"`
}
```

เป็น

```go
type InventoryWeightDetail struct {
	Key          string  `json:"key"`
	ProductCode  string  `json:"product_code"`
	BatchNo      string  `json:"batch_no"`
	SupplierCode string  `json:"supplier_code"`
	SupplierName string  `json:"supplier_name"`
	TotalWeight  float64 `json:"total_weight"`
	TotalQty     float64 `json:"total_qty"`
	// AvgWeight คือค่าเฉลี่ยน้ำหนักต่อชิ้นระดับ batch (company|site|product|batch)
	// ใช้กับ pattern ที่แสดงคอลัมน์ batch_no ซึ่ง 1 row = 1 batch
	AvgWeight float64 `json:"avg_weight"`
	// AvgProduct คือค่าเฉลี่ยน้ำหนักต่อชิ้นระดับ site (company|site|product) รวมทุก batch
	// เป็นค่าเดียวกันในทุก entry ของ product/site เดียวกัน
	// นี่คือค่าที่ตรงนิยาม "Avg. kg stock" และเป็นค่าที่สูตรคำนวณราคาต้องใช้
	AvgProduct float64 `json:"avg_product"`
}
```

- [ ] **Step 2: เขียน test ที่ต้อง fail**

สร้าง `internal/services/inventory-service/aggregate_inventory_weights_test.go`

```go
package inventoryService

import (
	"testing"

	"wms-service-warehouse/internal/models"
)

// Avg. kg stock = น้ำหนักรวมของ product นั้นใน site นั้น หารด้วยจำนวนชิ้นรวมใน site นั้น
// เป็น 1 ค่าต่อ product ต่อ site รวมทุก batch เข้าด้วยกัน
//
// AvgWeight ยังเป็นค่าระดับ batch เพราะ pattern ที่แสดงคอลัมน์ batch_no ต้องใช้
func TestAggregateInventoryWeightsAvgProductIsSiteLevel(t *testing.T) {
	// ข้อมูลจริงของ RBB610SR24TEF ที่ site TMI_WH จาก prime_wms_warehouse
	// น้ำหนักรวม 30434.1126 + 233.1 + 233.1 + 0 + 233.43 + 110 + 32 = 31275.7426
	// จำนวนรวม 10183 + 105 + 103 + 1 + 31 + 12 + 1 = 10436
	// 31275.7426 / 10436 = 2.996908...
	input := []models.InventoryWeight{
		{CompanyCode: "C1", SiteCode: "TMI_WH", ProductCode: "RBB610SR24TEF", BatchNo: "", Qty: 10183, TotalWeight: 30434.1126},
		{CompanyCode: "C1", SiteCode: "TMI_WH", ProductCode: "RBB610SR24TEF", BatchNo: "12341241", Qty: 105, TotalWeight: 233.1},
		{CompanyCode: "C1", SiteCode: "TMI_WH", ProductCode: "RBB610SR24TEF", BatchNo: "100-001", Qty: 103, TotalWeight: 233.1},
		{CompanyCode: "C1", SiteCode: "TMI_WH", ProductCode: "RBB610SR24TEF", BatchNo: "b1", Qty: 1, TotalWeight: 0},
		{CompanyCode: "C1", SiteCode: "TMI_WH", ProductCode: "RBB610SR24TEF", BatchNo: "AS", Qty: 31, TotalWeight: 233.43},
		{CompanyCode: "C1", SiteCode: "TMI_WH", ProductCode: "RBB610SR24TEF", BatchNo: "TEST_THEO", Qty: 12, TotalWeight: 110},
		{CompanyCode: "C1", SiteCode: "TMI_WH", ProductCode: "RBB610SR24TEF", BatchNo: "b2", Qty: 1, TotalWeight: 32},
	}

	got, _ := aggregateInventoryWeights(input)

	if len(got) != 7 {
		t.Fatalf("ต้องได้ 7 entry (1 ต่อ batch) แต่ได้ %d", len(got))
	}

	wantAvgProduct := 31275.7426 / 10436.0
	for _, d := range got {
		if !nearlyEqual(d.AvgProduct, wantAvgProduct) {
			t.Errorf("batch %q: AvgProduct = %v ต้องเป็น %v (ค่าระดับ site เหมือนกันทุก entry)",
				d.BatchNo, d.AvgProduct, wantAvgProduct)
		}
	}

	// AvgWeight ต้องยังเป็นค่าของ batch ตัวเอง
	wantAvgWeightByBatch := map[string]float64{
		"":          30434.1126 / 10183.0,
		"12341241":  233.1 / 105.0,
		"100-001":   233.1 / 103.0,
		"b1":        0,
		"AS":        233.43 / 31.0,
		"TEST_THEO": 110.0 / 12.0,
		"b2":        32.0,
	}
	for _, d := range got {
		want := wantAvgWeightByBatch[d.BatchNo]
		if !nearlyEqual(d.AvgWeight, want) {
			t.Errorf("batch %q: AvgWeight = %v ต้องเป็น %v (ค่าระดับ batch)", d.BatchNo, d.AvgWeight, want)
		}
	}
}

// product เดียวกันต่าง site ต้องไม่ปนกัน
func TestAggregateInventoryWeightsSeparatesSites(t *testing.T) {
	input := []models.InventoryWeight{
		{CompanyCode: "C1", SiteCode: "S1", ProductCode: "P1", BatchNo: "B1", Qty: 10, TotalWeight: 100},
		{CompanyCode: "C1", SiteCode: "S2", ProductCode: "P1", BatchNo: "B1", Qty: 10, TotalWeight: 500},
	}

	got, _ := aggregateInventoryWeights(input)

	bySite := map[string]float64{}
	for _, d := range got {
		bySite[d.Key] = d.AvgProduct
	}
	if bySite["C1|S1|P1|B1"] != 10 {
		t.Errorf("site S1: AvgProduct = %v ต้องเป็น 10", bySite["C1|S1|P1|B1"])
	}
	if bySite["C1|S2|P1|B1"] != 50 {
		t.Errorf("site S2: AvgProduct = %v ต้องเป็น 50", bySite["C1|S2|P1|B1"])
	}
}

// หลาย row ของ batch เดียวกัน (คนละ location) ต้องถูกรวมเข้า entry เดียว
func TestAggregateInventoryWeightsSumsRowsOfSameBatch(t *testing.T) {
	input := []models.InventoryWeight{
		{CompanyCode: "C1", SiteCode: "S1", ProductCode: "P1", BatchNo: "B1", Qty: 4, TotalWeight: 40},
		{CompanyCode: "C1", SiteCode: "S1", ProductCode: "P1", BatchNo: "B1", Qty: 6, TotalWeight: 80},
	}

	got, _ := aggregateInventoryWeights(input)

	if len(got) != 1 {
		t.Fatalf("ต้องได้ 1 entry แต่ได้ %d", len(got))
	}
	if got[0].TotalQty != 10 || got[0].TotalWeight != 120 {
		t.Fatalf("TotalQty = %v TotalWeight = %v ต้องเป็น 10 และ 120", got[0].TotalQty, got[0].TotalWeight)
	}
	if got[0].AvgWeight != 12 || got[0].AvgProduct != 12 {
		t.Errorf("AvgWeight = %v AvgProduct = %v ต้องเป็น 12 ทั้งคู่", got[0].AvgWeight, got[0].AvgProduct)
	}
}

// qty รวมเป็น 0 ต้องได้ 0 ไม่ panic ไม่เป็น Inf หรือ NaN
func TestAggregateInventoryWeightsZeroQty(t *testing.T) {
	input := []models.InventoryWeight{
		{CompanyCode: "C1", SiteCode: "S1", ProductCode: "P1", BatchNo: "B1", Qty: 0, TotalWeight: 50},
	}

	got, _ := aggregateInventoryWeights(input)

	if len(got) != 1 {
		t.Fatalf("ต้องได้ 1 entry แต่ได้ %d", len(got))
	}
	if got[0].AvgWeight != 0 || got[0].AvgProduct != 0 {
		t.Errorf("AvgWeight = %v AvgProduct = %v ต้องเป็น 0 ทั้งคู่", got[0].AvgWeight, got[0].AvgProduct)
	}
}

// input ว่างและ nil ต้องไม่ panic
func TestAggregateInventoryWeightsEmptyInput(t *testing.T) {
	got, suppliers := aggregateInventoryWeights(nil)
	if len(got) != 0 {
		t.Errorf("input nil ต้องได้ slice ว่าง แต่ได้ %d entry", len(got))
	}
	if len(suppliers) != 0 {
		t.Errorf("input nil ต้องได้ supplier ว่าง แต่ได้ %d", len(suppliers))
	}

	got, _ = aggregateInventoryWeights([]models.InventoryWeight{})
	if len(got) != 0 {
		t.Errorf("input ว่างต้องได้ slice ว่าง แต่ได้ %d entry", len(got))
	}
}

// ลำดับผลลัพธ์ต้องนิ่ง ไม่ขึ้นกับลำดับสุ่มของ map iteration
// test นี้คือตัวพิสูจน์บั๊ก F1 จะ fail บนโค้ดเดิมที่ไม่ sort
func TestAggregateInventoryWeightsIsDeterministic(t *testing.T) {
	input := []models.InventoryWeight{
		{CompanyCode: "C1", SiteCode: "S1", ProductCode: "P1", BatchNo: "b2", Qty: 1, TotalWeight: 32},
		{CompanyCode: "C1", SiteCode: "S1", ProductCode: "P1", BatchNo: "AS", Qty: 31, TotalWeight: 233.43},
		{CompanyCode: "C1", SiteCode: "S1", ProductCode: "P1", BatchNo: "b1", Qty: 1, TotalWeight: 0},
		{CompanyCode: "C1", SiteCode: "S1", ProductCode: "P1", BatchNo: "100-001", Qty: 103, TotalWeight: 233.1},
		{CompanyCode: "C1", SiteCode: "S1", ProductCode: "P1", BatchNo: "TEST_THEO", Qty: 12, TotalWeight: 110},
	}

	first, _ := aggregateInventoryWeights(input)
	firstKeys := make([]string, len(first))
	for i, d := range first {
		firstKeys[i] = d.Key
	}

	for run := 0; run < 50; run++ {
		got, _ := aggregateInventoryWeights(input)
		if len(got) != len(firstKeys) {
			t.Fatalf("run %d: จำนวน entry เปลี่ยนจาก %d เป็น %d", run, len(firstKeys), len(got))
		}
		for i, d := range got {
			if d.Key != firstKeys[i] {
				t.Fatalf("run %d: ลำดับเปลี่ยน ตำแหน่ง %d ได้ %q ต้องเป็น %q", run, i, d.Key, firstKeys[i])
			}
		}
	}
}

// supplier code ที่คืนออกมาต้องไม่ซ้ำ ไม่มีค่าว่าง และเรียงนิ่ง
func TestAggregateInventoryWeightsSupplierCodes(t *testing.T) {
	input := []models.InventoryWeight{
		{CompanyCode: "C1", SiteCode: "S1", ProductCode: "P1", BatchNo: "B1", SupplierCode: "SUP2", Qty: 1, TotalWeight: 1},
		{CompanyCode: "C1", SiteCode: "S1", ProductCode: "P1", BatchNo: "B2", SupplierCode: "SUP1", Qty: 1, TotalWeight: 1},
		{CompanyCode: "C1", SiteCode: "S1", ProductCode: "P1", BatchNo: "B3", SupplierCode: "SUP1", Qty: 1, TotalWeight: 1},
		{CompanyCode: "C1", SiteCode: "S1", ProductCode: "P1", BatchNo: "B4", SupplierCode: "", Qty: 1, TotalWeight: 1},
	}

	_, suppliers := aggregateInventoryWeights(input)

	if len(suppliers) != 2 {
		t.Fatalf("ต้องได้ supplier 2 ตัว แต่ได้ %d: %v", len(suppliers), suppliers)
	}
	if suppliers[0] != "SUP1" || suppliers[1] != "SUP2" {
		t.Errorf("supplier = %v ต้องเป็น [SUP1 SUP2] (เรียงแล้ว)", suppliers)
	}
}

// nearlyEqual เทียบ float แบบยอมคลาดเคลื่อนเล็กน้อยจากการหาร
func nearlyEqual(a, b float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < 0.0000001
}
```

- [ ] **Step 3: รัน test เพื่อยืนยันว่า fail**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-warehouse-core
go test ./internal/services/inventory-service/ -run TestAggregateInventoryWeights -v
```

Expected: FAIL — compile error `undefined: aggregateInventoryWeights`

- [ ] **Step 4: เขียนฟังก์ชัน pure ตัวใหม่**

เพิ่มฟังก์ชันนี้ใน `internal/services/inventory-service/get-inventory-weight-by-key.go` วางไว้**ก่อน**
`func getAggregatedInventoryWeights` (บรรทัด 271)

```go
// aggregateInventoryWeights รวมยอดน้ำหนักสต็อกจาก row ดิบเป็น 2 ระดับ
//
//   - AvgWeight  = ค่าเฉลี่ยต่อชิ้นระดับ batch (company|site|product|batch)
//     ใช้กับ pattern ที่แสดงคอลัมน์ batch_no ซึ่ง 1 row = 1 batch
//   - AvgProduct = ค่าเฉลี่ยต่อชิ้นระดับ site (company|site|product) รวมทุก batch
//     เป็นค่าเดียวกันทุก entry ของ product/site เดียวกัน และเป็นค่าที่ตรงนิยาม "Avg. kg stock"
//
// ผลลัพธ์เรียงตาม Key เสมอ เพราะการ iterate map ใน Go สุ่มลำดับ
// ผู้เรียกฝั่ง erp-core หยิบ entry แรกมาใช้ในหลายจุด ถ้าลำดับไม่นิ่งราคาที่คำนวณได้จะเปลี่ยนทุกครั้ง
//
// คืน supplier code ที่ไม่ซ้ำและเรียงแล้วเป็นค่าที่สอง
func aggregateInventoryWeights(inventoryWeights []models.InventoryWeight) ([]InventoryWeightDetail, []string) {
	invMap := make(map[string]*InventoryWeightDetail)
	// ยอดรวมระดับ site ใช้ key ที่ไม่มี batch
	productTotals := make(map[string]*struct{ weight, qty float64 })
	uniqueSupplierCodes := make(map[string]bool)

	for _, invW := range inventoryWeights {
		key := fmt.Sprintf("%s|%s|%s|%s", invW.CompanyCode, invW.SiteCode, invW.ProductCode, invW.BatchNo)
		productKey := fmt.Sprintf("%s|%s|%s", invW.CompanyCode, invW.SiteCode, invW.ProductCode)

		if invW.SupplierCode != "" {
			uniqueSupplierCodes[invW.SupplierCode] = true
		}

		if total, exists := productTotals[productKey]; exists {
			total.weight += invW.TotalWeight
			total.qty += invW.Qty
		} else {
			productTotals[productKey] = &struct{ weight, qty float64 }{
				weight: invW.TotalWeight,
				qty:    invW.Qty,
			}
		}

		if result, exists := invMap[key]; exists {
			result.TotalWeight += invW.TotalWeight
			result.TotalQty += invW.Qty
		} else {
			invMap[key] = &InventoryWeightDetail{
				Key:          key,
				ProductCode:  invW.ProductCode,
				BatchNo:      invW.BatchNo,
				SupplierCode: invW.SupplierCode,
				SupplierName: "", // เติมภายหลังจาก supplier service
				TotalWeight:  invW.TotalWeight,
				TotalQty:     invW.Qty,
			}
		}
	}

	result := make([]InventoryWeightDetail, 0, len(invMap))
	for _, v := range invMap {
		v.AvgWeight = safeAverage(v.TotalWeight, v.TotalQty)

		// ตัด batch ออกจาก key เพื่อหายอดรวมระดับ site ของ product เดียวกัน
		productKey := v.Key
		if idx := strings.LastIndex(productKey, "|"); idx >= 0 {
			productKey = productKey[:idx]
		}
		if total, ok := productTotals[productKey]; ok {
			v.AvgProduct = safeAverage(total.weight, total.qty)
		}

		result = append(result, *v)
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })

	supplierCodes := make([]string, 0, len(uniqueSupplierCodes))
	for code := range uniqueSupplierCodes {
		supplierCodes = append(supplierCodes, code)
	}
	sort.Strings(supplierCodes)

	return result, supplierCodes
}
```

`fmt`, `sort`, `strings` ถูก import อยู่แล้วที่บรรทัด 3-16 ของไฟล์นี้ ไม่ต้องเพิ่ม

- [ ] **Step 5: รัน test เพื่อยืนยันว่าผ่าน**

```bash
go test ./internal/services/inventory-service/ -run TestAggregateInventoryWeights -v
```

Expected: PASS ทั้ง 7 test

- [ ] **Step 6: ให้ฟังก์ชันเดิมเรียกใช้ฟังก์ชันใหม่**

ใน `getAggregatedInventoryWeights` แทนที่บล็อกตั้งแต่ `// Aggregate results by key` (บรรทัด 299)
จนจบ `return result, nil` (บรรทัด 350) ด้วย

```go
	// รวมยอดด้วยฟังก์ชัน pure เพื่อให้ทดสอบแยกได้ และให้ผลลัพธ์เรียงนิ่ง
	result, supplierCodes := aggregateInventoryWeights(inventoryWeights)

	if supplierCodesPtr != nil {
		*supplierCodesPtr = append(*supplierCodesPtr, supplierCodes...)
	}

	return result, nil
```

- [ ] **Step 7: ยืนยันว่า build ผ่านและ test เดิมไม่พัง**

```bash
go build ./... && go vet ./internal/services/inventory-service/ && go test ./internal/services/inventory-service/ -v 2>&1 | tail -30
```

Expected: build ผ่าน · vet ไม่มี error · test ทั้ง package PASS

ถ้า test เดิมตัวใดพัง ให้หยุดและรายงาน **ห้ามแก้ test เดิมให้ผ่านโดยไม่เข้าใจสาเหตุ**

- [ ] **Step 8: ตรวจ format เฉพาะไฟล์ที่แก้**

```bash
gofmt -l internal/services/inventory-service/get-inventory-weight-by-key.go internal/services/inventory-service/aggregate_inventory_weights_test.go
```

Expected: ไม่มี output

ถ้ามีชื่อไฟล์โผล่ ให้รัน `gofmt -w` **เฉพาะไฟล์ที่ชื่อโผล่มา** ห้ามรันทั้ง repo เพราะมี pre-existing violation จะทำให้ diff บวม

- [ ] **Step 9: commit**

```bash
git add internal/services/inventory-service/get-inventory-weight-by-key.go \
        internal/services/inventory-service/aggregate_inventory_weights_test.go
git commit -m "fix: เพิ่ม avg_product ระดับ site และทำผลลัพธ์ inventory weight ให้ deterministic

AvgWeight เดิมเป็นค่าเฉลี่ยระดับ batch และ slice ผลลัพธ์ไม่ถูก sort
ทำให้ผู้เรียกที่หยิบ entry แรกได้ batch แบบสุ่มทุกครั้ง

- เพิ่ม field AvgProduct = น้ำหนักรวม / จำนวนรวม ระดับ company|site|product
- sort ผลลัพธ์ตาม Key ให้ลำดับนิ่ง
- แยก logic รวมยอดเป็นฟังก์ชัน pure aggregateInventoryWeights เพื่อทดสอบได้
- ใช้ safeAverage ที่มีอยู่แล้วแทน guard qty > 0 เพื่อกัน Inf/NaN

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 3: แตก branch ใน `prime-wms-erp-core` และยืนยันว่า model รับ `avg_product` ได้

**Files:**
- ตรวจ: `internal/models/pricelist.go:283-302`

- [ ] **Step 1: ตรวจว่า branch ถูกสร้างไว้แล้ว**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
git status --short --branch | head -2
```

Expected: `## feat/pricelist-avg-kg-stock-site-level...origin/Develop`

ถ้าไม่ใช่ ให้รัน

```bash
git fetch origin && git checkout -b feat/pricelist-avg-kg-stock-site-level origin/Develop
```

- [ ] **Step 2: ยืนยันว่า `InventoryWeightResponse` มี `AvgProduct` พร้อม json tag ตรงกับฝั่ง warehouse**

```bash
grep -n "AvgProduct\|AvgBatch\|AvgWeight" internal/models/pricelist.go
```

Expected: เห็น `AvgProduct float64 \`json:"avg_product"\`` อยู่แล้ว

field นี้มีอยู่แล้วแต่ไม่เคยถูกเติมค่า ไม่ต้องแก้ model ฝั่ง erp-core
json tag `avg_product` ตรงกับที่เพิ่มใน Task 2 แล้ว

ถ้า grep ไม่เจอ `avg_product` ให้หยุดและรายงาน — แผนนี้ตั้งอยู่บนสมมติฐานว่า field มีอยู่

---

## Task 4: `erp-core` — แยกฟังก์ชันอ่าน avg เป็นระดับ site และระดับ batch

**Files:**
- Modify: `internal/services/price-service/patterns/shared.go:520-528` และจุดเรียก 11 จุด
- Test: `internal/services/price-service/patterns/avg_kg_stock_test.go` (create)

บริบท: `getAvgProductFromInventory` ชื่อบอกว่าอ่าน AvgProduct แต่จริง ๆ อ่าน `AvgWeight`
pattern ที่แสดงคอลัมน์ `batch_no` (1 row = 1 batch) ต้องใช้ค่าระดับ batch
pattern ที่เหลือต้องใช้ค่าระดับ site

ตัดสินว่า pattern ไหนเป็น per-batch จาก config ไม่ hardcode ชื่อ pattern
เพราะ `buildDynamicRows` ถูกใช้ทั้งโดย pattern ที่มีและไม่มี `batch_no`

- [ ] **Step 1: เขียน test ที่ต้อง fail**

สร้าง `internal/services/price-service/patterns/avg_kg_stock_test.go`

```go
package patterns

import (
	"testing"

	"prime-erp-core/internal/models"
)

// Avg. kg stock ที่ pattern ทั่วไปต้องแสดงคือค่าระดับ site (AvgProduct)
// ส่วน pattern ที่แสดงคอลัมน์ batch_no ต้องแสดงค่าระดับ batch (AvgWeight)
func TestGetAvgKgStockFromInventory(t *testing.T) {
	sg := models.PriceListSubGroupResponse{
		InventoryWeight: []models.InventoryWeightResponse{
			{AvgWeight: 32.0, AvgProduct: 2.9969},
		},
	}

	tests := []struct {
		name     string
		perBatch bool
		want     float64
	}{
		{name: "pattern ทั่วไปใช้ค่าระดับ site", perBatch: false, want: 3.0},
		{name: "pattern ที่แสดง batch ใช้ค่าระดับ batch", perBatch: true, want: 32.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAvgKgStockFromInventory(sg, tt.perBatch)
			if got != tt.want {
				t.Errorf("ได้ %v ต้องเป็น %v", got, tt.want)
			}
		})
	}
}

// ไม่มีสต็อกต้องคืน 0 เพื่อให้กริดแสดงเลข 0 ไม่ใช่ช่องว่าง
func TestGetAvgKgStockFromInventoryNoStock(t *testing.T) {
	empty := models.PriceListSubGroupResponse{}

	if got := getAvgKgStockFromInventory(empty, false); got != 0 {
		t.Errorf("ไม่มีสต็อก (site): ได้ %v ต้องเป็น 0", got)
	}
	if got := getAvgKgStockFromInventory(empty, true); got != 0 {
		t.Errorf("ไม่มีสต็อก (batch): ได้ %v ต้องเป็น 0", got)
	}
}

// ค่าที่แสดงต้องถูกปัดเป็น 2 ตำแหน่งเหมือนพฤติกรรมเดิม
func TestGetAvgKgStockFromInventoryRoundsToTwoDecimals(t *testing.T) {
	sg := models.PriceListSubGroupResponse{
		InventoryWeight: []models.InventoryWeightResponse{
			{AvgWeight: 9.16666666, AvgProduct: 2.99690876},
		},
	}

	if got := getAvgKgStockFromInventory(sg, false); got != 3.0 {
		t.Errorf("site: ได้ %v ต้องเป็น 3", got)
	}
	if got := getAvgKgStockFromInventory(sg, true); got != 9.17 {
		t.Errorf("batch: ได้ %v ต้องเป็น 9.17", got)
	}
}

// pattern ที่มีคอลัมน์ batch_no ต้องถูกจัดเป็น per-batch
// GROUP_1_ITEM_7 และ GROUP_1_ITEM_22 ใช้ headerName "โรงงาน"
// GROUP_1_ITEM_8 ใช้ "Ship No." แต่ field ยังเป็น batch_no
func TestPatternHasBatchColumn(t *testing.T) {
	tests := []struct {
		name    string
		pattern PatternConfig
		want    bool
	}{
		{
			name: "มี batch_no ใน Columns",
			pattern: PatternConfig{
				Columns: []ColumnConfigItem{
					{Field: "batch_no", HeaderName: "โรงงาน"},
					{Field: "avg_weight_ton", HeaderName: "Avg.kg stock (Tons)"},
				},
			},
			want: true,
		},
		{
			name: "มี batch_no ใน FixedColumns",
			pattern: PatternConfig{
				FixedColumns: []ColumnConfigItem{
					{Field: "batch_no", HeaderName: "Ship No."},
				},
			},
			want: true,
		},
		{
			name: "มี batch_no เป็น dataMapping เท่านั้น",
			pattern: PatternConfig{
				Columns: []ColumnConfigItem{
					{Field: "factory", HeaderName: "โรงงาน", DataMapping: "batch_no"},
				},
			},
			want: true,
		},
		{
			name: "ไม่มี batch_no เลย",
			pattern: PatternConfig{
				Columns: []ColumnConfigItem{
					{Field: "avg_weight", HeaderName: "Avg.kg stock"},
					{Field: "total_weight", HeaderName: "Weight-spec"},
				},
			},
			want: false,
		},
		{
			name:    "pattern ว่าง",
			pattern: PatternConfig{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := patternHasBatchColumn(&tt.pattern)
			if got != tt.want {
				t.Errorf("ได้ %v ต้องเป็น %v", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: รัน test เพื่อยืนยันว่า fail**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
go test ./internal/services/price-service/patterns/ -run "TestGetAvgKgStock|TestPatternHasBatchColumn" -v
```

Expected: FAIL — compile error `undefined: getAvgKgStockFromInventory` และ `undefined: patternHasBatchColumn`

- [ ] **Step 3: แทนฟังก์ชันเดิมด้วยฟังก์ชันใหม่**

ใน `internal/services/price-service/patterns/shared.go` แทนบล็อกบรรทัด 520-528 จาก

```go
// getAvgProductFromInventory extracts AvgWeight from the first InventoryWeight entry.
// Returns 0 (not an empty string) when inventory data is unavailable so the grid
// renders "0" instead of a blank cell and can format the value as a number.
func getAvgProductFromInventory(sg models.PriceListSubGroupResponse) float64 {
	if len(sg.InventoryWeight) > 0 {
		return roundTo2(sg.InventoryWeight[0].AvgWeight)
	}
	return 0
}
```

เป็น

```go
// getAvgKgStockFromInventory คืนค่าคอลัมน์ "Avg. kg stock"
//
// perBatch = false คือค่าเริ่มต้น ใช้ AvgProduct ซึ่งเป็นค่าเฉลี่ยระดับ site
// (น้ำหนักรวมของ product ใน site นั้น หารจำนวนชิ้นรวม) ตรงตามนิยามทางธุรกิจ
//
// perBatch = true ใช้ AvgWeight ซึ่งเป็นค่าเฉลี่ยระดับ batch สำหรับ pattern ที่
// แสดงคอลัมน์ batch_no โดย 1 row = 1 batch ค่าระดับ site จะไม่สื่ออะไรใน row แบบนั้น
//
// คืน 0 (ไม่ใช่ string ว่าง) เมื่อไม่มีข้อมูลสต็อก เพื่อให้กริดแสดงเลข 0
// และ format เป็นตัวเลขได้
func getAvgKgStockFromInventory(sg models.PriceListSubGroupResponse, perBatch bool) float64 {
	if len(sg.InventoryWeight) == 0 {
		return 0
	}
	if perBatch {
		return roundTo2(sg.InventoryWeight[0].AvgWeight)
	}
	return roundTo2(sg.InventoryWeight[0].AvgProduct)
}

// patternHasBatchColumn บอกว่า pattern นี้แสดงข้อมูลแยกต่อ batch หรือไม่
//
// ตัดสินจาก config ไม่ hardcode ชื่อ pattern เพราะ buildDynamicRows ถูกใช้
// ทั้งโดย pattern ที่มีและไม่มี batch_no
//
// บาง pattern เปลี่ยนชื่อคอลัมน์ไปเป็น "โรงงาน" หรือ "Ship No." และบางตัวอ้าง
// batch_no ผ่าน dataMapping จึงต้องตรวจทั้ง Field และ DataMapping
func patternHasBatchColumn(pattern *PatternConfig) bool {
	if pattern == nil {
		return false
	}
	for _, cols := range [][]ColumnConfigItem{pattern.Columns, pattern.FixedColumns} {
		for _, c := range cols {
			if c.Field == "batch_no" || c.DataMapping == "batch_no" {
				return true
			}
		}
	}
	return false
}
```

- [ ] **Step 4: รัน test เพื่อยืนยันว่าผ่าน**

```bash
go test ./internal/services/price-service/patterns/ -run "TestGetAvgKgStock|TestPatternHasBatchColumn" -v
```

Expected: PASS ทุก test

จะยัง build ไม่ผ่านทั้ง package เพราะจุดเรียกเดิม 11 จุดยังอ้างชื่อเก่า แก้ใน step ต่อไป

- [ ] **Step 5: หาจุดเรียกทั้งหมดที่ต้องแก้**

```bash
grep -n "getAvgProductFromInventory" internal/services/price-service/patterns/shared.go
```

Expected: 11 บรรทัด — 1267, 1399, 1401, 1407, 1715, 1845, 2102, 2104, 2112, 2370, 2372

- [ ] **Step 6: แก้จุดเรียกในฟังก์ชันที่มีตัวแปร `pattern`**

ในแต่ละฟังก์ชันที่มีพารามิเตอร์ `pattern *PatternConfig` ให้คำนวณ flag หนึ่งครั้งที่ต้นฟังก์ชัน
แล้วใช้ซ้ำ ไม่เรียก `patternHasBatchColumn` ซ้ำในลูป

ตัวอย่างใน `buildDynamicRows` (บรรทัด 1000) เพิ่มบรรทัดนี้ต้นฟังก์ชันหลัง signature

```go
	perBatch := patternHasBatchColumn(pattern)
```

แล้วเปลี่ยนทุกจุดเรียกในฟังก์ชันนั้นจาก

```go
getAvgProductFromInventory(sg)
```

เป็น

```go
getAvgKgStockFromInventory(sg, perBatch)
```

ทำแบบเดียวกันกับทุกฟังก์ชันที่มีจุดเรียก ใช้คำสั่งนี้หาชื่อฟังก์ชันและขอบเขตของแต่ละจุดเรียก

```bash
awk 'NR>=990 && NR<=2400 && (/^func /||/getAvgProductFromInventory/) {print NR": "$0}' \
  internal/services/price-service/patterns/shared.go
```

ถ้าฟังก์ชันใดไม่มีตัวแปร `pattern` ให้ดูว่า caller ส่งอะไรมาได้ และเพิ่มพารามิเตอร์
`perBatch bool` เข้าไปใน signature แล้วส่งต่อจาก caller
**ห้ามเดาเป็น `false` ทั้งหมด** เพราะจะทำให้ pattern ที่แสดง batch แสดงค่าผิด

- [ ] **Step 7: ยืนยันว่าไม่มีชื่อเดิมเหลือและ build ผ่าน**

```bash
grep -n "getAvgProductFromInventory" internal/services/price-service/patterns/shared.go || echo "ไม่มีชื่อเดิมเหลือแล้ว"
go build ./... && go vet ./internal/services/price-service/patterns/
```

Expected: ไม่มีชื่อเดิมเหลือ · build ผ่าน · vet ไม่มี error

- [ ] **Step 8: รัน test ทั้ง package**

```bash
go test ./internal/services/price-service/patterns/ 2>&1 | tail -20
```

Expected: PASS

ถ้า test เดิมพัง ให้อ่านว่า test นั้นคาดหวังค่าอะไร — ถ้ามัน assert ว่า `avg_kg_stock`
ต้องเท่ากับ `AvgWeight` ก็ต้องอัปเดต fixture ให้ตั้ง `AvgProduct` ด้วย ไม่ใช่เปลี่ยน assertion กลับ

- [ ] **Step 9: commit**

```bash
gofmt -l internal/services/price-service/patterns/shared.go internal/services/price-service/patterns/avg_kg_stock_test.go
git add internal/services/price-service/patterns/shared.go \
        internal/services/price-service/patterns/avg_kg_stock_test.go
git commit -m "fix: แยก Avg. kg stock เป็นค่าระดับ site และระดับ batch

getAvgProductFromInventory ชื่อบอกว่าอ่าน AvgProduct แต่จริง ๆ อ่าน AvgWeight
ซึ่งเป็นค่าระดับ batch ทำให้ pattern ทั่วไปแสดงค่าของ batch เดียวแทนค่าทั้ง site

- getAvgKgStockFromInventory(sg, perBatch) เลือกค่าตามชนิดของ pattern
- patternHasBatchColumn ตัดสินจาก config ไม่ hardcode ชื่อ pattern

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 5: `erp-core` — เรียงลำดับสูตรตาม dependency (F4)

**Files:**
- Create: `internal/services/price-service/formula_order.go`
- Test: `internal/services/price-service/formula_order_test.go` (create)

บริบท: `internal/repositories/priceList/repository.go:641` ดึงสูตรด้วย `ORDER BY psfm.create_dtm DESC`
ซึ่งไม่ใช่ลำดับ dependency สูตร `Pcs = [kg] x [Avg. kg stock]` ต้องรันหลังสูตร `kg = Base price + Extra`
แต่ 522 subgroup รันสลับลำดับ

แก้ที่ชั้น service ไม่แก้ `ORDER BY` เพราะลำดับตามเวลายังเป็นสิ่งที่ flow อื่นใช้
และการเรียงตาม dependency เป็นตรรกะของการคำนวณ

- [ ] **Step 1: เขียน test ที่ต้อง fail**

สร้าง `internal/services/price-service/formula_order_test.go`

```go
package priceService

import (
	"encoding/json"
	"testing"

	"prime-erp-core/internal/models"
)

// helper สร้าง map entry ของสูตรสำหรับ test
func formulaEntry(name, uom, expression string, required []string) models.PriceListSubGroupFormulasMap {
	params, _ := json.Marshal(map[string]interface{}{
		"required":    required,
		"description": name,
	})
	return models.PriceListSubGroupFormulasMap{
		PriceListFormulas: models.PriceListFormulas{
			Name:        name,
			Uom:         uom,
			FormulaType: "price_calc",
			Expression:  expression,
			Params:      params,
		},
	}
}

func names(formulas []models.PriceListSubGroupFormulasMap) []string {
	out := make([]string, len(formulas))
	for i, f := range formulas {
		out[i] = f.PriceListFormulas.Name
	}
	return out
}

// สูตร pcs ที่อ้าง kg ต้องรันหลังสูตร uom kg ไม่ว่า input จะเรียงมาแบบใด
func TestSortFormulasByDependencyPutsProducerFirst(t *testing.T) {
	kgFormula := formulaEntry("kg = Base price + Extra", "kg", "base_price+extra", []string{"base_price", "extra"})
	pcsFormula := formulaEntry("Pcs = [kg]  x [Avg. kg stock]", "pcs", "kg*avg_kg_stock", []string{"kg", "avg_kg_stock"})

	tests := []struct {
		name  string
		input []models.PriceListSubGroupFormulasMap
	}{
		{name: "input เรียง kg ก่อน", input: []models.PriceListSubGroupFormulasMap{kgFormula, pcsFormula}},
		{name: "input เรียง pcs ก่อน", input: []models.PriceListSubGroupFormulasMap{pcsFormula, kgFormula}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortFormulasByDependency(tt.input)
			gotNames := names(got)
			if len(gotNames) != 2 {
				t.Fatalf("ต้องได้ 2 สูตร แต่ได้ %d", len(gotNames))
			}
			if gotNames[0] != "kg = Base price + Extra" {
				t.Errorf("สูตรแรกคือ %q ต้องเป็น %q", gotNames[0], "kg = Base price + Extra")
			}
		})
	}
}

// คู่ kg = [Pcs] / [Avg. kg stock] + pcs = input
// สูตร kg อ้าง pcs จึงต้องรันหลัง และสูตร input ต้องคงอยู่ในลำดับ
func TestSortFormulasByDependencyInputFormulaFirst(t *testing.T) {
	inputFormula := models.PriceListSubGroupFormulasMap{
		PriceListFormulas: models.PriceListFormulas{
			Name:        "pcs = input",
			Uom:         "pcs",
			FormulaType: "input",
		},
	}
	kgFormula := formulaEntry("kg = [Pcs] / [Avg. kg stock]", "kg", "pcs/avg_kg_stock", []string{"pcs", "avg_kg_stock"})

	got := sortFormulasByDependency([]models.PriceListSubGroupFormulasMap{kgFormula, inputFormula})
	gotNames := names(got)

	if gotNames[0] != "pcs = input" {
		t.Errorf("สูตรแรกคือ %q ต้องเป็น %q", gotNames[0], "pcs = input")
	}
}

// สูตรที่ไม่อ้างถึงกันต้องคงลำดับเดิม
func TestSortFormulasByDependencyKeepsOrderWhenIndependent(t *testing.T) {
	a := formulaEntry("Pcs = ( [Base price] + 2.1 ) x [Avg kg. stock] x (1+2%)", "pcs",
		"(base_price+2.1)*avg_kg_stock*1.02", []string{"base_price", "avg_kg_stock"})
	b := formulaEntry("kg = Base price + Extra", "kg", "base_price+extra", []string{"base_price", "extra"})

	got := sortFormulasByDependency([]models.PriceListSubGroupFormulasMap{a, b})
	gotNames := names(got)

	if gotNames[0] != a.PriceListFormulas.Name || gotNames[1] != b.PriceListFormulas.Name {
		t.Errorf("ลำดับ = %v ต้องคงเดิม", gotNames)
	}
}

// สูตรที่อ้างอิงวนกันต้องคงลำดับเดิมและไม่ panic
func TestSortFormulasByDependencyCircularKeepsOrder(t *testing.T) {
	pcsFormula := formulaEntry("Pcs from kg", "pcs", "kg*avg_kg_stock", []string{"kg"})
	kgFormula := formulaEntry("kg from Pcs", "kg", "pcs/avg_kg_stock", []string{"pcs"})

	got := sortFormulasByDependency([]models.PriceListSubGroupFormulasMap{pcsFormula, kgFormula})
	gotNames := names(got)

	if len(gotNames) != 2 {
		t.Fatalf("ต้องได้ 2 สูตร แต่ได้ %d", len(gotNames))
	}
	if gotNames[0] != "Pcs from kg" || gotNames[1] != "kg from Pcs" {
		t.Errorf("ลำดับ = %v ต้องคงเดิมเมื่ออ้างอิงวนกัน", gotNames)
	}
}

// input ว่าง nil และสูตรเดียวต้องไม่ panic
func TestSortFormulasByDependencyEdgeCases(t *testing.T) {
	if got := sortFormulasByDependency(nil); len(got) != 0 {
		t.Errorf("input nil ต้องได้ slice ว่าง แต่ได้ %d", len(got))
	}
	if got := sortFormulasByDependency([]models.PriceListSubGroupFormulasMap{}); len(got) != 0 {
		t.Errorf("input ว่างต้องได้ slice ว่าง แต่ได้ %d", len(got))
	}

	single := formulaEntry("kg = Base price + Extra", "kg", "base_price+extra", []string{"base_price"})
	got := sortFormulasByDependency([]models.PriceListSubGroupFormulasMap{single})
	if len(got) != 1 || got[0].PriceListFormulas.Name != single.PriceListFormulas.Name {
		t.Errorf("สูตรเดียวต้องคืนตัวเดิม")
	}
}

// params ที่ parse ไม่ได้ต้องไม่ทำให้ panic และคงลำดับเดิม
func TestSortFormulasByDependencyInvalidParams(t *testing.T) {
	broken := models.PriceListSubGroupFormulasMap{
		PriceListFormulas: models.PriceListFormulas{
			Name:        "broken params",
			Uom:         "pcs",
			FormulaType: "price_calc",
			Expression:  "kg*avg_kg_stock",
			Params:      json.RawMessage(`{not json`),
		},
	}
	kgFormula := formulaEntry("kg = Base price + Extra", "kg", "base_price+extra", []string{"base_price"})

	got := sortFormulasByDependency([]models.PriceListSubGroupFormulasMap{broken, kgFormula})
	if len(got) != 2 {
		t.Fatalf("ต้องได้ 2 สูตร แต่ได้ %d", len(got))
	}
}
```

- [ ] **Step 2: รัน test เพื่อยืนยันว่า fail**

```bash
go test ./internal/services/price-service/ -run TestSortFormulasByDependency -v
```

Expected: FAIL — compile error `undefined: sortFormulasByDependency`

- [ ] **Step 3: เขียน implementation**

สร้าง `internal/services/price-service/formula_order.go`

```go
package priceService

import (
	"encoding/json"
	"fmt"

	"prime-erp-core/internal/models"
)

// sortFormulasByDependency เรียงสูตรให้สูตรที่ผลิตค่าอยู่ก่อนสูตรที่บริโภคค่านั้น
//
// สูตรใน price_list_formulas อ้างถึงผลลัพธ์ของกันเอง เช่น
// Pcs = [kg] x [Avg. kg stock] มี expression kg*avg_kg_stock ซึ่ง kg คือผลลัพธ์
// ของสูตร uom = "kg" ในคู่เดียวกัน ไม่ใช่ราคาตั้งของกลุ่ม
//
// repository ดึงสูตรมาด้วย ORDER BY create_dtm DESC ซึ่งเป็นลำดับที่ผูกสูตร
// ไม่ใช่ลำดับ dependency ทำให้บาง subgroup ประเมินสูตร pcs ก่อนที่ kg จะถูก
// คำนวณใหม่ แล้วอ่านค่าของรอบคำนวณก่อนหน้า
//
// ตัดสิน dependency จาก params.required ที่เก็บไว้ใน DB อยู่แล้ว
// ถ้าสูตร A มี required ที่ตรงกับ uom ของสูตร B แล้ว B ต้องมาก่อน A
//
// กรณีอ้างอิงวนกันให้คงลำดับเดิมและไม่คืน error เพราะตอนนี้ไม่มีข้อมูลเช่นนั้น
// และการทำให้ request ล้มจะแย่กว่าการคำนวณด้วยลำดับเดิม
func sortFormulasByDependency(formulas []models.PriceListSubGroupFormulasMap) []models.PriceListSubGroupFormulasMap {
	if len(formulas) < 2 {
		return formulas
	}

	// uom ของสูตรแต่ละตัวคือชื่อตัวแปรที่สูตรนั้นผลิต ("pcs" หรือ "kg")
	producedBy := make(map[string][]int)
	for i, f := range formulas {
		uom := f.PriceListFormulas.Uom
		if uom != "" {
			producedBy[uom] = append(producedBy[uom], i)
		}
	}

	// needs[i] = เซ็ตของ index ที่สูตร i ต้องรอ
	needs := make([]map[int]bool, len(formulas))
	for i, f := range formulas {
		needs[i] = make(map[int]bool)
		for _, varName := range formulaRequiredVars(f.PriceListFormulas) {
			for _, producer := range producedBy[varName] {
				if producer != i {
					needs[i][producer] = true
				}
			}
		}
	}

	// topological sort แบบคงลำดับเดิมไว้มากที่สุด
	done := make([]bool, len(formulas))
	result := make([]models.PriceListSubGroupFormulasMap, 0, len(formulas))

	for len(result) < len(formulas) {
		progressed := false
		for i := range formulas {
			if done[i] {
				continue
			}
			ready := true
			for dep := range needs[i] {
				if !done[dep] {
					ready = false
					break
				}
			}
			if ready {
				result = append(result, formulas[i])
				done[i] = true
				progressed = true
			}
		}
		if !progressed {
			// อ้างอิงวนกัน เติมตัวที่เหลือตามลำดับเดิมแล้วจบ
			fmt.Printf("Warning: พบการอ้างอิงวนกันระหว่างสูตร คงลำดับเดิมไว้\n")
			for i := range formulas {
				if !done[i] {
					result = append(result, formulas[i])
					done[i] = true
				}
			}
		}
	}

	return result
}

// formulaRequiredVars อ่านรายชื่อตัวแปรที่สูตรต้องใช้จาก params.required
// คืน slice ว่างเมื่อ params ไม่มีหรือ parse ไม่ได้ เพื่อไม่ให้ข้อมูลเสียทำให้ request ล้ม
func formulaRequiredVars(formula models.PriceListFormulas) []string {
	if len(formula.Params) == 0 {
		return nil
	}
	var parsed struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(formula.Params, &parsed); err != nil {
		return nil
	}
	return parsed.Required
}
```

- [ ] **Step 4: รัน test เพื่อยืนยันว่าผ่าน**

```bash
go test ./internal/services/price-service/ -run TestSortFormulasByDependency -v
```

Expected: PASS ทุก test

- [ ] **Step 5: commit**

```bash
gofmt -l internal/services/price-service/formula_order.go internal/services/price-service/formula_order_test.go
go vet ./internal/services/price-service/
git add internal/services/price-service/formula_order.go internal/services/price-service/formula_order_test.go
git commit -m "feat: เรียงลำดับสูตรราคาตาม dependency

สูตร Pcs = [kg] x [Avg. kg stock] อ้างผลลัพธ์ของสูตร uom kg ในคู่เดียวกัน
แต่ repository ดึงสูตรด้วย ORDER BY create_dtm DESC ทำให้ 522 subgroup
ประเมินสูตร pcs ก่อน kg แล้วอ่านค่าของรอบคำนวณก่อนหน้า

อ่าน dependency จาก params.required ที่เก็บไว้ใน DB อยู่แล้ว
กรณีอ้างอิงวนกันคงลำดับเดิมและ log warning ไม่ทำให้ request ล้ม

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 6: `erp-core` — `avgKgStock` อ่าน `AvgProduct` และ `pcs`/`kg` เป็นค่า running

**Files:**
- Modify: `internal/services/price-service/update-latest-pricelist-subgroup.go:231-243` และ `:250-300`
- Modify: `internal/services/price-service/get-calculated-pricelist-subgroup.go:218-229` และบล็อกสูตรที่ตรงกัน
- Test: `internal/services/price-service/avg_kg_stock_price_test.go` (create)

บริบท: `CalculatePrice` (บรรทัด 427) สร้าง env จาก `PriceData` ตัวแปร `pcs` และ `kg`
ถูกป้อน `TotalQty` / `TotalWeight` ของสต็อก ทั้งที่สูตรต้องการราคาต่อชิ้นและราคาต่อกิโล
ผลคือ 694 จาก 712 subgroup มี `total_net_price_unit = 0`

ค่าที่ถูกคือตัวแปร running `totalNetPriceUnit` / `totalNetPriceWeight` ที่ init จากค่าใน DB
ที่บรรทัด 222-223 อยู่แล้ว

**Task 5 ต้องเสร็จก่อน** เพราะถ้าอ่านค่า running โดยไม่เรียงลำดับสูตร 522 subgroup
จะอ่านค่าของรอบก่อนหน้า กลายเป็นผิดแบบใหม่

- [ ] **Step 1: เขียน test ที่ต้อง fail**

สร้าง `internal/services/price-service/avg_kg_stock_price_test.go`

```go
package priceService

import (
	"encoding/json"
	"testing"

	priceDomain "prime-erp-core/internal/services/price-service/domain"
)

// คู่สูตรหลักคือ pipeline 2 ขั้น
//   1. kg  = base_price + extra          -> total_net_price_weight
//   2. pcs = kg * avg_kg_stock           -> total_net_price_unit
// [kg] ในขั้นที่ 2 คือผลลัพธ์ของขั้นที่ 1 ไม่ใช่ราคาตั้งของกลุ่ม
//
// ตัวเลขจาก SG711 @ TMI_WH: price_weight 19.40 extra_price_weight 0.20
func TestCalculatePricePipelineUsesRunningValues(t *testing.T) {
	params, _ := json.Marshal(map[string]interface{}{"required": []string{"base_price", "extra"}})
	kgFormula := priceDomain.PriceFormula{
		Expression: "base_price+extra",
		Params:     params,
		Rounding:   2,
	}

	kgPrice, err := CalculatePrice(kgFormula, priceDomain.PriceData{
		BasePrice: 19.40,
		Extra:     0.20,
	})
	if err != nil {
		t.Fatalf("คำนวณสูตร kg ล้มเหลว: %v", err)
	}
	if kgPrice != 19.60 {
		t.Fatalf("ราคาต่อกิโล = %v ต้องเป็น 19.60", kgPrice)
	}

	pcsParams, _ := json.Marshal(map[string]interface{}{"required": []string{"kg", "avg_kg_stock"}})
	pcsFormula := priceDomain.PriceFormula{
		Expression: "kg*avg_kg_stock",
		Params:     pcsParams,
		Rounding:   0,
	}

	// kg ต้องเป็นผลลัพธ์ของสูตรก่อนหน้า (19.60) ไม่ใช่ราคาตั้ง (19.40) และไม่ใช่น้ำหนักสต็อก
	pcsPrice, err := CalculatePrice(pcsFormula, priceDomain.PriceData{
		AvgKgStock: 3.0,
		Kg:         kgPrice,
	})
	if err != nil {
		t.Fatalf("คำนวณสูตร pcs ล้มเหลว: %v", err)
	}
	if pcsPrice != 59 {
		t.Errorf("ราคาต่อชิ้น = %v ต้องเป็น 59 (19.60 x 3 ปัดเป็นจำนวนเต็ม)", pcsPrice)
	}
}

// ถ้าป้อน kg ด้วยน้ำหนักสต็อกจะได้ค่าที่ไม่ใช่ราคา
// test นี้บันทึกพฤติกรรมที่ผิดไว้เพื่อกันการถอยกลับ
func TestCalculatePriceRejectsStockWeightAsKg(t *testing.T) {
	pcsParams, _ := json.Marshal(map[string]interface{}{"required": []string{"kg", "avg_kg_stock"}})
	pcsFormula := priceDomain.PriceFormula{
		Expression: "kg*avg_kg_stock",
		Params:     pcsParams,
		Rounding:   0,
	}

	// ไม่มีสต็อก TotalWeight = 0 -> ราคาต่อชิ้นกลายเป็น 0
	zeroStock, err := CalculatePrice(pcsFormula, priceDomain.PriceData{AvgKgStock: 3.0, Kg: 0})
	if err != nil {
		t.Fatalf("คำนวณล้มเหลว: %v", err)
	}
	if zeroStock != 0 {
		t.Errorf("kg = 0 ต้องได้ 0 แต่ได้ %v", zeroStock)
	}

	// มีสต็อก 211000 kg -> ราคาต่อชิ้นกลายเป็นหลักแสน
	bigStock, err := CalculatePrice(pcsFormula, priceDomain.PriceData{AvgKgStock: 3.0, Kg: 211000})
	if err != nil {
		t.Fatalf("คำนวณล้มเหลว: %v", err)
	}
	if bigStock != 633000 {
		t.Errorf("kg = 211000 ต้องได้ 633000 แต่ได้ %v", bigStock)
	}
}

// สูตรที่ใช้ avg_kg_stock ทั้ง 4 ตัวต้องคำนวณตรงกับการคิดมือ
func TestCalculatePriceAllAvgKgStockFormulas(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		rounding   int
		data       priceDomain.PriceData
		want       float64
	}{
		{
			name:       "Pcs = [kg] x [Avg. kg stock]",
			expression: "kg*avg_kg_stock",
			rounding:   0,
			data:       priceDomain.PriceData{Kg: 19.60, AvgKgStock: 3.0},
			want:       59,
		},
		{
			name:       "kg = [Pcs] / [Avg. kg stock]",
			expression: "pcs/avg_kg_stock",
			rounding:   2,
			data:       priceDomain.PriceData{Pcs: 58.80, AvgKgStock: 3.0},
			want:       19.60,
		},
		{
			name:       "Pcs = ( [Base price] + 1.4 ) x [Avg kg. stock] x (1+2%)",
			expression: "(base_price+1.4)*avg_kg_stock*1.02",
			rounding:   0,
			data:       priceDomain.PriceData{BasePrice: 18.30, AvgKgStock: 3.0},
			want:       60,
		},
		{
			name:       "Pcs = ( [Base price] + 2.1 ) x [Avg kg. stock] x (1+2%)",
			expression: "(base_price+2.1)*avg_kg_stock*1.02",
			rounding:   0,
			data:       priceDomain.PriceData{BasePrice: 18.30, AvgKgStock: 3.0},
			want:       62,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculatePrice(priceDomain.PriceFormula{
				Expression: tt.expression,
				Rounding:   tt.rounding,
			}, tt.data)
			if err != nil {
				t.Fatalf("คำนวณล้มเหลว: %v", err)
			}
			if got != tt.want {
				t.Errorf("ได้ %v ต้องเป็น %v", got, tt.want)
			}
		})
	}
}

// สูตร weight_spec คู่ผกผันต้องกลับไปกลับมาได้หลังแก้ expression (F3)
func TestCalculatePriceWeightSpecFormulasAreInverse(t *testing.T) {
	pcsFromKg, err := CalculatePrice(priceDomain.PriceFormula{
		Expression: "kg*weight_spec",
		Rounding:   2,
	}, priceDomain.PriceData{Kg: 19.60, WeightSpec: 3.0})
	if err != nil {
		t.Fatalf("คำนวณล้มเหลว: %v", err)
	}
	if pcsFromKg != 58.80 {
		t.Fatalf("ราคาต่อชิ้น = %v ต้องเป็น 58.80", pcsFromKg)
	}

	kgFromPcs, err := CalculatePrice(priceDomain.PriceFormula{
		Expression: "pcs/weight_spec",
		Rounding:   2,
	}, priceDomain.PriceData{Pcs: pcsFromKg, WeightSpec: 3.0})
	if err != nil {
		t.Fatalf("คำนวณล้มเหลว: %v", err)
	}
	if kgFromPcs != 19.60 {
		t.Errorf("ราคาต่อกิโล = %v ต้องกลับมาเป็น 19.60", kgFromPcs)
	}
}
```

- [ ] **Step 2: รัน test เพื่อดูสถานะก่อนแก้**

```bash
go test ./internal/services/price-service/ -run "TestCalculatePrice" -v 2>&1 | tail -30
```

Expected: test ส่วนใหญ่ PASS เพราะทดสอบ `CalculatePrice` ตรง ๆ ซึ่งยังไม่ถูกแก้
test เหล่านี้เป็นการตรึงพฤติกรรมที่ถูกต้องไว้ก่อนแก้ caller

ถ้า `TestCalculatePriceWeightSpecFormulasAreInverse` FAIL ให้บันทึกไว้ จะผ่านหลัง Task 9

- [ ] **Step 3: แก้ `update-latest-pricelist-subgroup.go` — แหล่งของ `avgKgStock` และลบ `pcs`/`kg` จากสต็อก**

แทนบล็อกบรรทัด 231-243 จาก

```go
		// Get inventory data for this subgroup
		avgKgStock := 1.0
		pcs := 0.0
		kg := 0.0
		if inventoryWeight, ok := inventoryMap[subGroupID.String()]; ok && len(inventoryWeight) > 0 {
			// Use AvgProduct from first inventory weight response
			if inventoryWeight[0].AvgWeight == 0 {
				avgKgStock = 1.0
			} else {
				avgKgStock = inventoryWeight[0].AvgWeight
			}
			pcs = inventoryWeight[0].TotalQty
			kg = inventoryWeight[0].TotalWeight
		}
```

เป็น

```go
		// avg_kg_stock คือน้ำหนักเฉลี่ยต่อชิ้นของ product ใน site นั้น รวมทุก batch
		// จึงต้องอ่าน AvgProduct ไม่ใช่ AvgWeight ซึ่งเป็นค่าระดับ batch
		// AvgProduct เป็นค่าเดียวกันทุก entry จึงหยิบ entry แรกได้อย่างปลอดภัย
		//
		// fallback เป็น 1.0 เมื่อไม่มีสต็อก เป็นพฤติกรรมที่ตกลงกันไว้
		// ทำให้สูตรที่คูณด้วย avg_kg_stock ให้ผลเหมือนไม่มีตัวคูณ
		avgKgStock := 1.0
		if inventoryWeight, ok := inventoryMap[subGroupID.String()]; ok && len(inventoryWeight) > 0 {
			if inventoryWeight[0].AvgProduct != 0 {
				avgKgStock = inventoryWeight[0].AvgProduct
			}
		}
```

ตัวแปร `pcs` และ `kg` ถูกลบทิ้งเพราะสูตรไม่ได้ต้องการจำนวนสต็อก

- [ ] **Step 4: เรียงลำดับสูตรก่อนเข้าลูป**

แทนบรรทัด 220 จาก

```go
		priceListFormulas := formulasMap[subGroup.SubGroupCode]
```

เป็น

```go
		// สูตร pcs อ้างผลลัพธ์ของสูตร kg ในคู่เดียวกัน จึงต้องเรียงตาม dependency
		// ไม่ใช่ตามลำดับ create_dtm ที่ repository คืนมา
		priceListFormulas := sortFormulasByDependency(formulasMap[subGroup.SubGroupCode])
```

- [ ] **Step 5: ป้อน `Pcs` / `Kg` จากค่า running ใน `PriceData` ทั้งสอง case**

ใน `case "pcs"` (บรรทัด ~269) แทน

```go
						priceData := priceDomain.PriceData{
							BasePrice:  subGroup.PriceListGroup.PriceUnit,
							Extra:      extraPriceUnit,
							AvgKgStock: avgKgStock,
							WeightSpec: weightSpec,
							Pcs:        pcs,
							Kg:         kg,
						}
```

ด้วย

```go
						// Pcs และ Kg คือราคาต่อชิ้นและราคาต่อกิโลล่าสุด ไม่ใช่จำนวนสต็อก
						// อ่านจากตัวแปร running ที่ถูกอัปเดตเมื่อสูตรก่อนหน้าคำนวณเสร็จ
						priceData := priceDomain.PriceData{
							BasePrice:  subGroup.PriceListGroup.PriceUnit,
							Extra:      extraPriceUnit,
							AvgKgStock: avgKgStock,
							WeightSpec: weightSpec,
							Pcs:        totalNetPriceUnit,
							Kg:         totalNetPriceWeight,
						}
```

ใน `case "kg"` (บรรทัด ~289) แทน

```go
						priceData := priceDomain.PriceData{
							BasePrice:  subGroup.PriceListGroup.PriceWeight,
							Extra:      extraPriceWeight,
							AvgKgStock: avgKgStock,
							WeightSpec: weightSpec,
							Pcs:        pcs,
							Kg:         kg,
						}
```

ด้วย

```go
						priceData := priceDomain.PriceData{
							BasePrice:  subGroup.PriceListGroup.PriceWeight,
							Extra:      extraPriceWeight,
							AvgKgStock: avgKgStock,
							WeightSpec: weightSpec,
							Pcs:        totalNetPriceUnit,
							Kg:         totalNetPriceWeight,
						}
```

`priceData` ถูกสร้างใหม่ในแต่ละรอบของลูปอยู่แล้ว จึงเห็นค่าที่สูตรก่อนหน้าเพิ่งเขียน

- [ ] **Step 6: ทำแบบเดียวกันใน `get-calculated-pricelist-subgroup.go`**

แก้บล็อก `avgKgStock` ที่บรรทัด 218-229 · การเรียงสูตรที่บรรทัด ~207 · และ `PriceData`
ทั้งสอง case ที่บรรทัด ~257-264 และ ~277-284 ด้วยรูปแบบเดียวกับ step 3-5

หาตำแหน่งที่แน่นอนด้วย

```bash
grep -n "avgKgStock\|formulasMap\[\|Pcs:\|Kg:" internal/services/price-service/get-calculated-pricelist-subgroup.go
```

ไฟล์นี้คำนวณให้ดูไม่บันทึกลง DB ตรรกะการอ่านค่าต้องเหมือน `update-latest` ทุกจุด
ไม่เช่นนั้นราคาที่แสดงกับราคาที่บันทึกจะไม่ตรงกัน

- [ ] **Step 7: ยืนยัน build และไม่มีตัวแปรค้าง**

```bash
go build ./... && go vet ./internal/services/price-service/
grep -n "inventoryWeight\[0\].TotalQty\|inventoryWeight\[0\].TotalWeight" \
  internal/services/price-service/update-latest-pricelist-subgroup.go \
  internal/services/price-service/get-calculated-pricelist-subgroup.go || echo "ไม่มีการอ่านจำนวนสต็อกเข้าสูตรแล้ว"
```

Expected: build ผ่าน · vet ไม่มี error · grep ไม่เจอ

ถ้า build ฟ้อง `declared and not used` สำหรับ `pcs` หรือ `kg` แปลว่าลบไม่ครบ ให้ลบให้หมด

- [ ] **Step 8: รัน test ทั้ง package**

```bash
go test ./internal/services/price-service/ 2>&1 | tail -25
```

Expected: PASS

- [ ] **Step 9: commit**

```bash
gofmt -l internal/services/price-service/update-latest-pricelist-subgroup.go \
         internal/services/price-service/get-calculated-pricelist-subgroup.go \
         internal/services/price-service/avg_kg_stock_price_test.go
git add internal/services/price-service/update-latest-pricelist-subgroup.go \
        internal/services/price-service/get-calculated-pricelist-subgroup.go \
        internal/services/price-service/avg_kg_stock_price_test.go
git commit -m "fix: สูตรราคาอ่าน avg ระดับ site และราคา running ไม่ใช่จำนวนสต็อก

ตัวแปร pcs และ kg ในสูตรหมายถึงราคาต่อชิ้นและราคาต่อกิโล แต่โค้ดป้อน
TotalQty และ TotalWeight ของสต็อกเข้าไป ทำให้ 694 จาก 712 subgroup
มี total_net_price_unit = 0 และที่เหลือได้ค่าหลักแสน

- avgKgStock อ่าน AvgProduct (ระดับ site) แทน AvgWeight (ระดับ batch)
- Pcs/Kg อ่านจากตัวแปร running totalNetPriceUnit/totalNetPriceWeight
- เรียงสูตรตาม dependency ก่อนเข้าลูปประเมิน

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 7: `erp-core` — แก้ `get-price-detail.go` เรื่องค่าค้างและ dead write

**Files:**
- Modify: `internal/services/price-service/get-price-detail.go:316-327`

บริบท: การเขียนค่าแบบมีเงื่อนไข `if inv.X > 0` ทำให้ field คงค่าที่ค้างจาก subgroup ต้นแบบ
เมื่อ batch ใหม่มีค่า 0 เป็นรูปแบบเดียวกับบั๊ก Weight-spec รอบก่อน
และ `AvgBatch` ถูกเขียนแต่ไม่มีใครอ่านในโปรเจกต์

- [ ] **Step 1: ยืนยันว่าไม่มีผู้อ่าน `AvgBatch` ในฝั่ง Go**

```bash
grep -rn "AvgBatch" --include=*.go . | grep -v "_test.go"
```

Expected: เห็นเพียง 2 บรรทัด — นิยาม field ใน `internal/models/pricelist.go:294`
และจุดเขียนใน `internal/services/price-service/get-price-detail.go:324`

ถ้าเจอจุดอ่านเพิ่ม ให้หยุดและรายงาน — แผนนี้ตั้งอยู่บนสมมติฐานว่าเป็น dead write

ฝั่ง web อ่าน `avgBatch` จริงแต่รับค่าจาก endpoint `get-inventory-weight` คนละตัว
การลบจุดเขียนนี้ไม่กระทบ

- [ ] **Step 2: แก้โค้ด**

แทนบรรทัด 316-326 จาก

```go
							// Map new API fields to existing model fields
							if inv.TotalQty > 0 {
								expandedSG.InventoryWeight[0].SumQty = inv.TotalQty
							}
							if inv.TotalWeight > 0 {
								expandedSG.InventoryWeight[0].SumWeight = inv.TotalWeight
							}
							if inv.AvgWeight > 0 {
								expandedSG.InventoryWeight[0].AvgBatch = inv.AvgWeight
							}
```

เป็น

```go
							// เขียนค่าตรง ๆ ไม่ใช้เงื่อนไข > 0
							// เงื่อนไขเดิมทำให้ field คงค่าที่ค้างจาก subgroup ต้นแบบ
							// เมื่อ batch นี้มีค่าเป็น 0 จริง ๆ
							//
							// ไม่เขียน AvgBatch อีกต่อไปเพราะไม่มีผู้อ่านในโปรเจกต์
							// ค่าระดับ batch อ่านได้จาก AvgWeight และระดับ site จาก AvgProduct
							expandedSG.InventoryWeight[0].SumQty = inv.TotalQty
							expandedSG.InventoryWeight[0].SumWeight = inv.TotalWeight
```

- [ ] **Step 3: ยืนยัน build และ test**

```bash
go build ./... && go test ./internal/services/price-service/ 2>&1 | tail -20
```

Expected: build ผ่าน · test PASS

- [ ] **Step 4: commit**

```bash
gofmt -l internal/services/price-service/get-price-detail.go
git add internal/services/price-service/get-price-detail.go
git commit -m "fix: ลบการเขียนค่าแบบมีเงื่อนไขและ dead write AvgBatch

if inv.X > 0 ทำให้ field คงค่าที่ค้างจาก subgroup ต้นแบบเมื่อ batch มีค่า 0 จริง
และ AvgBatch ถูกเขียนแต่ไม่มีผู้อ่านในฝั่ง Go เลย

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 8: `erp-core` — รวมพฤติกรรมของ row builder ทั้ง 3 ตัว

**Files:**
- Modify: `internal/services/price-service/get-price-export-table.go:630-648`
- Modify: `internal/services/price-service/build-pricelist-detail-tab.go:169` และ `:198-202`
- Test: `internal/services/price-service/avg_kg_stock_rowbuilder_test.go` (create)

บริบท: เมื่อไม่มีสต็อก 3 row builder ให้ผลต่างกัน — กริดคืน `0` · export table ไม่ set key เลย
เพราะ early return · Pricelist Detail Report ใส่ string ว่าง ผู้ใช้เห็นไม่เหมือนกัน
ทั้งสามต้องคืน `0` และต้องอ่าน `AvgProduct`

- [ ] **Step 1: เขียน test ที่ต้อง fail**

สร้าง `internal/services/price-service/avg_kg_stock_rowbuilder_test.go`

```go
package priceService

import (
	"testing"

	"prime-erp-core/internal/models"
)

// export table ต้องเติม avg_weight เป็น 0 เมื่อไม่มีสต็อก
// เดิม early return ทำให้ key หายไปทั้งคอลัมน์
func TestApplyInventoryFieldsToRowNoStock(t *testing.T) {
	row := map[string]interface{}{}
	sg := SubGroup{WeightSpec: 12.5}

	applyInventoryFieldsToRow(row, sg)

	if row["total_weight"] != 12.5 {
		t.Errorf("total_weight = %v ต้องเป็น 12.5 (มาจาก product master)", row["total_weight"])
	}
	avg, ok := row["avg_weight"]
	if !ok {
		t.Fatal("ไม่มี key avg_weight ต้องมีและเป็น 0 เมื่อไม่มีสต็อก")
	}
	if avg != float64(0) {
		t.Errorf("avg_weight = %v (%T) ต้องเป็น float64(0)", avg, avg)
	}
}

// มีสต็อกต้องอ่าน AvgProduct ซึ่งเป็นค่าระดับ site ไม่ใช่ AvgWeight ระดับ batch
func TestApplyInventoryFieldsToRowUsesAvgProduct(t *testing.T) {
	row := map[string]interface{}{}
	sg := SubGroup{
		WeightSpec: 12.5,
		InventoryWeight: []models.InventoryWeightResponse{
			{AvgWeight: 32.0, AvgProduct: 2.9969, TotalQty: 10436, SumQty: 10436},
		},
	}

	applyInventoryFieldsToRow(row, sg)

	if row["avg_weight"] != 2.9969 {
		t.Errorf("avg_weight = %v ต้องเป็น 2.9969 (AvgProduct ระดับ site)", row["avg_weight"])
	}
}
```

- [ ] **Step 2: รัน test เพื่อยืนยันว่า fail**

```bash
go test ./internal/services/price-service/ -run TestApplyInventoryFieldsToRow -v
```

Expected: FAIL — `ไม่มี key avg_weight` และ `avg_weight` อ่านจาก `AvgWeight`

- [ ] **Step 3: แก้ `applyInventoryFieldsToRow`**

ใน `internal/services/price-service/get-price-export-table.go` แทนบรรทัด 630-648 จาก

```go
func applyInventoryFieldsToRow(row map[string]interface{}, sg SubGroup) {
	row["total_weight"] = sg.WeightSpec

	if len(sg.InventoryWeight) == 0 {
		return
	}

	inv := sg.InventoryWeight[0]
	row["avg_weight"] = inv.AvgWeight
```

เป็น

```go
func applyInventoryFieldsToRow(row map[string]interface{}, sg SubGroup) {
	row["total_weight"] = sg.WeightSpec

	if len(sg.InventoryWeight) == 0 {
		// ต้องเติม 0 ไม่ใช่ปล่อยให้ key หาย เพื่อให้ตรงกับกริดและ Pricelist Detail Report
		row["avg_weight"] = float64(0)
		return
	}

	inv := sg.InventoryWeight[0]
	// AvgProduct คือค่าเฉลี่ยระดับ site ตรงตามนิยาม "Avg. kg stock"
	// AvgWeight เป็นค่าระดับ batch ซึ่งไม่ใช่สิ่งที่คอลัมน์นี้ต้องแสดง
	row["avg_weight"] = inv.AvgProduct
```

บรรทัดที่เหลือในฟังก์ชัน (`market_weight`, `stock`, `batch_no` ฯลฯ) ไม่ต้องแก้

- [ ] **Step 4: แก้ `build-pricelist-detail-tab.go`**

แทนบรรทัด 169 จาก

```go
				"avg_weight":           "",
```

เป็น

```go
				"avg_weight":           float64(0),
```

และแทนบรรทัด 200-202 จาก

```go
			if len(sg.InventoryWeight) > 0 {
				row["avg_weight"] = sg.InventoryWeight[0].AvgWeight
			}
```

เป็น

```go
			// AvgProduct คือค่าเฉลี่ยระดับ site ตรงตามนิยาม "Avg. kg stock"
			if len(sg.InventoryWeight) > 0 {
				row["avg_weight"] = sg.InventoryWeight[0].AvgProduct
			}
```

- [ ] **Step 5: รัน test เพื่อยืนยันว่าผ่าน**

```bash
go test ./internal/services/price-service/ -run TestApplyInventoryFieldsToRow -v
go test ./internal/services/price-service/ 2>&1 | tail -20
```

Expected: test ใหม่ PASS · test ทั้ง package PASS

ถ้า `build-pricelist-detail-tab_test.go` เดิมพังเพราะ assert string ว่าง
ให้อัปเดต expectation เป็น `float64(0)` — นี่คือการเปลี่ยนพฤติกรรมที่ตั้งใจ

- [ ] **Step 6: commit**

```bash
gofmt -l internal/services/price-service/get-price-export-table.go \
         internal/services/price-service/build-pricelist-detail-tab.go \
         internal/services/price-service/avg_kg_stock_rowbuilder_test.go
git add internal/services/price-service/get-price-export-table.go \
        internal/services/price-service/build-pricelist-detail-tab.go \
        internal/services/price-service/avg_kg_stock_rowbuilder_test.go
git commit -m "fix: row builder ทั้ง 3 ตัวอ่าน AvgProduct และคืน 0 เมื่อไม่มีสต็อก

เดิมกริดคืน 0 · export table ไม่ set key เลย · Pricelist Detail Report ใส่ string ว่าง
ผู้ใช้เห็นคอลัมน์เดียวกันไม่เหมือนกันใน 3 ที่

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 9: `erp-core` — migration แก้ expression ที่ผิด (F3)

**Files:**
- Create: `migrations/2026-09-10-fix-pcs-weight-spec-expression.sql`
- Modify: `internal/scripts/price_list_formulas/price-list-formulas.json`

บริบท: สูตรชื่อ `kg = [Pcs] / [Weight Spec]` มี expression เป็น `pcs*weight_spec`
ทั้งที่ชื่อระบุว่าเป็นการหาร คู่ผกผันของมันคือ `Pcs = [kg] x [Weight Spec]` (`kg*weight_spec`)

- [ ] **Step 1: ยืนยันสถานะใน DB ก่อนแก้**

```bash
PGPASSWORD='H3dPq8Tz1Lm6Rk9VbY' psql -h 18.138.69.85 -U dev_champ -d prime_erp \
  -c "select formula_code, name, expression from price_list_formulas where name like '%Weight Spec%' order by name"
```

Expected: เห็น `kg = [Pcs] / [Weight Spec]` มี expression `pcs*weight_spec`
บันทึก `formula_code` ที่ได้ไว้ใช้ใน migration

- [ ] **Step 2: เขียน migration**

สร้าง `migrations/2026-09-10-fix-pcs-weight-spec-expression.sql`

```sql
-- แก้ expression ของสูตร kg = [Pcs] / [Weight Spec]
--
-- expression เดิมเป็น pcs*weight_spec ซึ่งเป็นการคูณ ไม่ตรงกับชื่อสูตรที่ระบุว่าหาร
-- คู่ผกผันของสูตรนี้คือ Pcs = [kg] x [Weight Spec] (kg*weight_spec)
-- weight_spec มีหน่วย kg ต่อชิ้น ดังนั้น ราคาต่อกิโล = ราคาต่อชิ้น / kg ต่อชิ้น
--
-- idempotent: อัปเดตเฉพาะแถวที่ยังมี expression ผิด
-- ระบุแถวด้วย expression + name ไม่ใช้ id เพราะ id ต่างกันในแต่ละ environment
UPDATE price_list_formulas
SET expression = 'pcs/weight_spec'
WHERE name = 'kg = [Pcs] / [Weight Spec]'
  AND expression = 'pcs*weight_spec';

-- ตรวจผล: ต้องไม่เหลือแถวที่ชื่อบอกหารแต่ expression คูณ
DO $$
DECLARE
    wrong_count integer;
BEGIN
    SELECT count(*) INTO wrong_count
    FROM price_list_formulas
    WHERE name = 'kg = [Pcs] / [Weight Spec]'
      AND expression <> 'pcs/weight_spec';

    IF wrong_count > 0 THEN
        RAISE EXCEPTION 'ยังเหลือสูตร kg = [Pcs] / [Weight Spec] ที่ expression ไม่ถูกต้อง % แถว', wrong_count;
    END IF;
END $$;
```

- [ ] **Step 3: แก้ seed script ให้ตรงกับ migration**

ใน `internal/scripts/price_list_formulas/price-list-formulas.json` หาบล็อกของสูตรนี้

```bash
grep -n -B2 -A8 "Weight Spec" internal/scripts/price_list_formulas/price-list-formulas.json
```

เปลี่ยน `"expression": "pcs*weight_spec"` ที่อยู่ในบล็อกซึ่งมี
`"name": "kg = [Pcs] / [Weight Spec]"` เป็น `"expression": "pcs/weight_spec"`

**ระวัง** บล็อก `Pcs = [kg]  x [Weight Spec]` มี expression `kg*weight_spec` ซึ่ง **ถูกต้องแล้ว**
ห้ามแก้ตัวนั้น

- [ ] **Step 4: ยืนยันว่า JSON ยังถูก format และแก้ถูกบล็อก**

```bash
python3 -c "
import json
d = json.load(open('internal/scripts/price_list_formulas/price-list-formulas.json'))
for f in d['price_list_formulas']:
    if 'Weight Spec' in (f.get('name') or ''):
        print(f['name'], '->', f['expression'])
"
```

Expected:
```
Pcs = [kg]  x [Weight Spec] -> kg*weight_spec
kg = [Pcs] / [Weight Spec] -> pcs/weight_spec
```

- [ ] **Step 5: รัน test ที่ตรึงคู่ผกผันไว้**

```bash
go test ./internal/services/price-service/ -run TestCalculatePriceWeightSpecFormulasAreInverse -v
```

Expected: PASS

- [ ] **Step 6: commit**

```bash
git add migrations/2026-09-10-fix-pcs-weight-spec-expression.sql \
        internal/scripts/price_list_formulas/price-list-formulas.json
git commit -m "fix: แก้ expression ของสูตร kg = [Pcs] / [Weight Spec]

expression เดิมเป็น pcs*weight_spec ซึ่งคูณ ไม่ตรงกับชื่อสูตรที่ระบุว่าหาร
และไม่เป็นผกผันของคู่ของมันคือ Pcs = [kg] x [Weight Spec]

migration เขียนแบบ idempotent และมีการตรวจผลในตัว

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

**ยังไม่ต้องรัน migration กับ DB** — รันตอน deploy ตามลำดับใน Task 12

---

## Task 10: `erp-core` — test ของ flow คำนวณราคาทั้งเส้น

**Files:**
- Test: `internal/services/price-service/avg_kg_stock_flow_test.go` (create)
- Test: `internal/services/price-service/avg_kg_stock_integration_test.go` (create)

บริบท: `UpdateLatestPriceListSubGroup` inject dependency ทั้งหมดผ่าน package-level function var
(`getPriceListSubGroupsByIDsFunc`, `getPriceListSubGroupFormulasMapBySubGroupCodesFunc`,
`updateLatestSubGroupFunc`) จึงทดสอบ flow เต็มเส้นได้โดยไม่ต้องพึ่ง DB
ดูรูปแบบจาก `TestUpdateLatestPriceListSubGroup_WithPcsFormula` ที่มีอยู่แล้ว

ส่วนเส้นทางการแสดงผลใช้ helper จาก `weight_spec_integration_test.go` ที่มีอยู่
(`fakeWarehouseServer`, `pointWarehouseEndpointAt`, `buildSingleSubGroupResponse`,
`ensureGroupPaymentTablesForTest`)

- [ ] **Step 1: เขียน test ของ flow คำนวณราคา (ไม่พึ่ง DB)**

สร้าง `internal/services/price-service/avg_kg_stock_flow_test.go`

```go
package priceService

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"prime-erp-core/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// formulaMapEntry สร้าง mapping ของสูตรหนึ่งตัวสำหรับ test flow
func formulaMapEntry(code, name, uom, expression string, rounding int, required []string) models.PriceListSubGroupFormulasMap {
	params, _ := json.Marshal(map[string]interface{}{"required": required})
	return models.PriceListSubGroupFormulasMap{
		ID:                    uuid.New(),
		PriceListSubGroupCode: "SUB001",
		PriceListFormulasCode: code,
		PriceListFormulas: models.PriceListFormulas{
			ID:          uuid.New(),
			FormulaCode: code,
			Name:        name,
			Uom:         uom,
			FormulaType: "price_calc",
			Expression:  expression,
			Params:      params,
			Rounding:    rounding,
		},
	}
}

// subGroupForFlowTest สร้าง subgroup ที่มีราคาตั้งและค่า total_net_price เดิมค้างอยู่
//
// TotalNetPriceWeight เดิมเป็น 45 ซึ่งต่างจากค่าที่สูตร kg จะคำนวณได้ (50 + 5 = 55)
// ความต่างนี้คือสิ่งที่ทำให้แยกออกว่าสูตร pcs อ่านค่าใหม่หรือค่าค้าง
func subGroupForFlowTest(subGroupID, groupID uuid.UUID) models.PriceListSubGroup {
	now := time.Now()
	return models.PriceListSubGroup{
		ID:                  subGroupID,
		PriceListGroupID:    groupID,
		SubGroupCode:        "SUB001",
		SubgroupKey:         "SUB",
		IsTrading:           true,
		PriceUnit:           100.0,
		PriceWeight:         50.0,
		ExtraPriceUnit:      10.0,
		ExtraPriceWeight:    5.0,
		TotalNetPriceUnit:   90.0,
		TotalNetPriceWeight: 45.0,
		CreateBy:            "tester",
		CreateDtm:           &now,
		UpdateBy:            "tester",
		UpdateDtm:           &now,
		PriceListGroup: models.PriceListGroup{
			ID:          groupID,
			PriceUnit:   100.0,
			PriceWeight: 50.0,
		},
	}
}

// runUpdateLatestFlow เรียก UpdateLatestPriceListSubGroup ด้วยสูตรที่กำหนด
// แล้วคืน change ที่ถูกส่งไปบันทึก
func runUpdateLatestFlow(t *testing.T, formulas []models.PriceListSubGroupFormulasMap) models.UpdatePriceListSubGroupItem {
	t.Helper()
	gin.SetMode(gin.TestMode)

	groupID := uuid.New()
	subGroupID := uuid.New()
	subGroup := subGroupForFlowTest(subGroupID, groupID)

	originalGetByIDs := getPriceListSubGroupsByIDsFunc
	getPriceListSubGroupsByIDsFunc = func([]uuid.UUID) ([]models.PriceListSubGroup, error) {
		return []models.PriceListSubGroup{subGroup}, nil
	}
	defer func() { getPriceListSubGroupsByIDsFunc = originalGetByIDs }()

	originalGetFormulas := getPriceListSubGroupFormulasMapBySubGroupCodesFunc
	getPriceListSubGroupFormulasMapBySubGroupCodesFunc = func([]string) (map[string][]models.PriceListSubGroupFormulasMap, error) {
		return map[string][]models.PriceListSubGroupFormulasMap{"SUB001": formulas}, nil
	}
	defer func() { getPriceListSubGroupFormulasMapBySubGroupCodesFunc = originalGetFormulas }()

	var updateRequest models.UpdatePriceListSubGroupRequest
	originalUpdate := updateLatestSubGroupFunc
	updateLatestSubGroupFunc = func(req models.UpdatePriceListSubGroupRequest) error {
		updateRequest = req
		return nil
	}
	defer func() { updateLatestSubGroupFunc = originalUpdate }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(map[string]interface{}{"subgroup_ids": []string{subGroupID.String()}})
	req := httptest.NewRequest(http.MethodPost, "/price/SubGroup/UpdateLatest", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	_, err := UpdateLatestPriceListSubGroup(c)
	assert.NoError(t, err)
	assert.Len(t, updateRequest.Changes, 1)

	return updateRequest.Changes[0]
}

// คู่สูตรหลัก: kg = base_price + extra แล้ว pcs = kg * avg_kg_stock
//
// ไม่มีสต็อกจึง avg_kg_stock ตกไป fallback 1.0
// ราคาต่อกิโลที่ถูกคือ 50 + 5 = 55 และราคาต่อชิ้นคือ 55 x 1.0 = 55
//
// ถ้าสูตร pcs อ่าน total_net_price_weight ค่าเก่า (45) จะได้ 45 ซึ่งผิด
// ถ้าสูตร pcs อ่านน้ำหนักสต็อก (0) จะได้ 0 ซึ่งเป็นพฤติกรรมเดิมที่พัง
func TestUpdateLatestUsesRunningWeightPriceInPcsFormula(t *testing.T) {
	kgFormula := formulaMapEntry("F_KG", "kg = Base price + Extra", "kg", "base_price+extra", 2,
		[]string{"base_price", "extra"})
	pcsFormula := formulaMapEntry("F_PCS", "Pcs = [kg]  x [Avg. kg stock]", "pcs", "kg*avg_kg_stock", 2,
		[]string{"kg", "avg_kg_stock"})

	change := runUpdateLatestFlow(t, []models.PriceListSubGroupFormulasMap{kgFormula, pcsFormula})

	if change.TotalNetPriceWeight == nil {
		t.Fatal("TotalNetPriceWeight เป็น nil ต้องถูกคำนวณ")
	}
	assert.Equal(t, 55.0, *change.TotalNetPriceWeight, "ราคาต่อกิโล = PriceWeight + ExtraPriceWeight")

	if change.TotalNetPriceUnit == nil {
		t.Fatal("TotalNetPriceUnit เป็น nil ต้องถูกคำนวณ")
	}
	assert.Equal(t, 55.0, *change.TotalNetPriceUnit,
		"ราคาต่อชิ้นต้องมาจากราคาต่อกิโลที่เพิ่งคำนวณ (55) ไม่ใช่ค่าเก่า (45) และไม่ใช่น้ำหนักสต็อก (0)")
}

// ลำดับสูตรที่ส่งมาสลับ (pcs มาก่อน) ต้องให้ผลเหมือนกัน
// เพราะ sortFormulasByDependency เรียงใหม่ให้สูตร kg มาก่อน
//
// นี่คือเคสของ 522 subgroup ที่ create_dtm ของสูตร pcs ใหม่กว่า
func TestUpdateLatestCorrectWhenFormulaOrderReversed(t *testing.T) {
	kgFormula := formulaMapEntry("F_KG", "kg = Base price + Extra", "kg", "base_price+extra", 2,
		[]string{"base_price", "extra"})
	pcsFormula := formulaMapEntry("F_PCS", "Pcs = [kg]  x [Avg. kg stock]", "pcs", "kg*avg_kg_stock", 2,
		[]string{"kg", "avg_kg_stock"})

	// ส่ง pcs มาก่อน kg เลียนแบบลำดับที่ repository คืนมาจริง
	change := runUpdateLatestFlow(t, []models.PriceListSubGroupFormulasMap{pcsFormula, kgFormula})

	if change.TotalNetPriceUnit == nil {
		t.Fatal("TotalNetPriceUnit เป็น nil ต้องถูกคำนวณ")
	}
	assert.Equal(t, 55.0, *change.TotalNetPriceUnit,
		"ต้องได้ราคาที่ถูกจากการคำนวณครั้งเดียว ไม่ต้องกดสองครั้ง")
}

// เรียก flow ซ้ำหลายครั้งด้วย input เดียวกันต้องได้ผลเท่ากันทุกครั้ง
func TestUpdateLatestIsDeterministicAcrossRuns(t *testing.T) {
	kgFormula := formulaMapEntry("F_KG", "kg = Base price + Extra", "kg", "base_price+extra", 2,
		[]string{"base_price", "extra"})
	pcsFormula := formulaMapEntry("F_PCS", "Pcs = [kg]  x [Avg. kg stock]", "pcs", "kg*avg_kg_stock", 2,
		[]string{"kg", "avg_kg_stock"})
	formulas := []models.PriceListSubGroupFormulasMap{pcsFormula, kgFormula}

	first := runUpdateLatestFlow(t, formulas)
	if first.TotalNetPriceUnit == nil || first.TotalNetPriceWeight == nil {
		t.Fatal("รอบแรกคำนวณไม่ครบ")
	}
	wantUnit := *first.TotalNetPriceUnit
	wantWeight := *first.TotalNetPriceWeight

	for run := 0; run < 20; run++ {
		change := runUpdateLatestFlow(t, formulas)
		if change.TotalNetPriceUnit == nil || change.TotalNetPriceWeight == nil {
			t.Fatalf("run %d: คำนวณไม่ครบ", run)
		}
		if *change.TotalNetPriceUnit != wantUnit {
			t.Fatalf("run %d: ราคาต่อชิ้น = %v ต้องเป็น %v ทุกครั้ง", run, *change.TotalNetPriceUnit, wantUnit)
		}
		if *change.TotalNetPriceWeight != wantWeight {
			t.Fatalf("run %d: ราคาต่อกิโล = %v ต้องเป็น %v ทุกครั้ง", run, *change.TotalNetPriceWeight, wantWeight)
		}
	}
}
```

- [ ] **Step 2: รัน test ของ flow**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
go test ./internal/services/price-service/ -run "TestUpdateLatestUsesRunningWeightPrice|TestUpdateLatestCorrectWhenFormulaOrderReversed|TestUpdateLatestIsDeterministic" -v
```

Expected: PASS ทั้ง 3 test

ถ้า `TestUpdateLatestUsesRunningWeightPriceInPcsFormula` ได้ `0` แปลว่า Task 6 step 5 ยังไม่ถูกแก้
ถ้าได้ `45` แปลว่า Task 5 ยังไม่ถูกเรียกใช้ใน Task 6 step 4

ถ้า `models.UpdatePriceListSubGroupItem` ใช้ field ที่ไม่ใช่ pointer ให้ตัด `if ... == nil`
ออกและเทียบค่าตรง ๆ ตรวจด้วย

```bash
grep -n -A12 "type UpdatePriceListSubGroupItem struct" internal/models/pricelist.go
```

- [ ] **Step 3: เขียน integration test ของเส้นทางการแสดงผล**

สร้าง `internal/services/price-service/avg_kg_stock_integration_test.go`

```go
//go:build integration

package priceService

import (
	"testing"

	"github.com/google/uuid"
)

// คอลัมน์ Avg. kg stock ต้องได้ค่าจาก avg_product ซึ่งเป็นค่าเฉลี่ยระดับ site
// ไม่ใช่ avg_weight ซึ่งเป็นค่าระดับ batch
//
// ตัวเลขจากข้อมูลจริงของ RBB610SR24TEF @ TMI_WH
// avg ระดับ site = 31275.7426 / 10436 = 2.9969...
// batch b2 มี avg = 32 ซึ่งต่างกันมาก ใช้แยกว่าอ่าน field ถูกตัวหรือไม่
func TestGetPriceDetailAvgProductPropagates(t *testing.T) {
	ensureGroupPaymentTablesForTest(t)

	subGroupID := uuid.New()
	body := `[{
		"id": "` + subGroupID.String() + `",
		"product_code": "RBB610SR24TEF",
		"supplier_code": "SUP001",
		"supplier_name": "Supplier 1",
		"weight_spec": 12.5,
		"inventory_weight": [
			{"batch_no": "b2", "total_weight": 32, "total_qty": 1, "avg_weight": 32, "avg_product": 2.9969},
			{"batch_no": "", "total_weight": 30434.1126, "total_qty": 10183, "avg_weight": 2.9887, "avg_product": 2.9969}
		]
	}]`
	srv := fakeWarehouseServer(t, body)
	defer srv.Close()
	pointWarehouseEndpointAt(t, srv.URL)

	result, err := transformToGetPriceListResponse(buildSingleSubGroupResponse(subGroupID))
	if err != nil {
		t.Fatalf("transformToGetPriceListResponse: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("ต้องได้ 1 group แต่ได้ %d", len(result))
	}

	// expand 1 row ต่อ 1 batch
	subGroups := result[0].SubGroups
	if len(subGroups) != 2 {
		t.Fatalf("ต้องได้ 2 subgroup (1 ต่อ batch) แต่ได้ %d", len(subGroups))
	}

	// ทุก row ต้องมี avg_product ค่าเดียวกัน เพราะเป็นค่าระดับ site
	for _, sg := range subGroups {
		if len(sg.InventoryWeight) != 1 {
			t.Fatalf("batch %q: ต้องมี InventoryWeight 1 ตัว แต่มี %d", sg.BatchNo, len(sg.InventoryWeight))
		}
		if sg.InventoryWeight[0].AvgProduct != 2.9969 {
			t.Errorf("batch %q: AvgProduct = %v ต้องเป็น 2.9969 เท่ากันทุก row",
				sg.BatchNo, sg.InventoryWeight[0].AvgProduct)
		}
	}

	// avg_weight ต้องยังเป็นค่าของ batch ตัวเอง
	byBatch := map[string]float64{}
	for _, sg := range subGroups {
		byBatch[sg.BatchNo] = sg.InventoryWeight[0].AvgWeight
	}
	if byBatch["b2"] != 32 {
		t.Errorf("batch b2: AvgWeight = %v ต้องเป็น 32", byBatch["b2"])
	}
	if byBatch[""] != 2.9887 {
		t.Errorf("batch ว่าง: AvgWeight = %v ต้องเป็น 2.9887", byBatch[""])
	}
}

// ไม่มีสต็อกต้องไม่ทำให้ค่าค้างจาก subgroup ต้นแบบหลุดออกมา
func TestGetPriceDetailNoStockLeavesNoStaleValue(t *testing.T) {
	ensureGroupPaymentTablesForTest(t)

	subGroupID := uuid.New()
	body := `[{
		"id": "` + subGroupID.String() + `",
		"product_code": "P001",
		"supplier_code": "SUP001",
		"supplier_name": "Supplier 1",
		"weight_spec": 12.5,
		"inventory_weight": []
	}]`
	srv := fakeWarehouseServer(t, body)
	defer srv.Close()
	pointWarehouseEndpointAt(t, srv.URL)

	result, err := transformToGetPriceListResponse(buildSingleSubGroupResponse(subGroupID))
	if err != nil {
		t.Fatalf("transformToGetPriceListResponse: %v", err)
	}

	if len(result) != 1 || len(result[0].SubGroups) != 1 {
		t.Fatalf("ต้องได้ 1 group 1 subgroup แต่ได้ %#v", result)
	}

	sg := result[0].SubGroups[0]
	if sg.WeightSpec != 12.5 {
		t.Errorf("WeightSpec = %v ต้องเป็น 12.5 แม้ไม่มีสต็อก", sg.WeightSpec)
	}
	if len(sg.InventoryWeight) != 0 {
		t.Errorf("ไม่มีสต็อกแต่มี InventoryWeight %d ตัว", len(sg.InventoryWeight))
	}
}

// batch ที่มี qty จริงแต่น้ำหนักเป็น 0 ต้องได้ 0 ไม่ใช่ค่าค้างจาก batch ก่อนหน้า
func TestGetPriceDetailZeroWeightBatchIsNotOverwrittenByStaleValue(t *testing.T) {
	ensureGroupPaymentTablesForTest(t)

	subGroupID := uuid.New()
	body := `[{
		"id": "` + subGroupID.String() + `",
		"product_code": "RBB610SR24TEF",
		"supplier_code": "SUP001",
		"supplier_name": "Supplier 1",
		"weight_spec": 12.5,
		"inventory_weight": [
			{"batch_no": "b1", "total_weight": 0, "total_qty": 1, "avg_weight": 0, "avg_product": 2.9969}
		]
	}]`
	srv := fakeWarehouseServer(t, body)
	defer srv.Close()
	pointWarehouseEndpointAt(t, srv.URL)

	result, err := transformToGetPriceListResponse(buildSingleSubGroupResponse(subGroupID))
	if err != nil {
		t.Fatalf("transformToGetPriceListResponse: %v", err)
	}

	sg := result[0].SubGroups[0]
	inv := sg.InventoryWeight[0]
	if inv.SumWeight != 0 {
		t.Errorf("SumWeight = %v ต้องเป็น 0 ตามข้อมูลจริงของ batch นี้", inv.SumWeight)
	}
	if inv.SumQty != 1 {
		t.Errorf("SumQty = %v ต้องเป็น 1", inv.SumQty)
	}
	if inv.AvgWeight != 0 {
		t.Errorf("AvgWeight = %v ต้องเป็น 0", inv.AvgWeight)
	}
}
```

- [ ] **Step 4: รัน integration test**

```bash
make test-integration-pricelist 2>&1 | tail -40
```

Expected: PASS ทุก test รวม test เดิมของ weight_spec

ถ้า test เดิมพังเพราะชื่อ helper ซ้ำ แปลว่าผมตั้งชื่อ helper ใหม่ชนกับของเดิม
ให้เปลี่ยนชื่อ helper ที่เพิ่มใหม่ ไม่ใช่แก้ของเดิม

- [ ] **Step 5: ยืนยันว่าไม่มี test ถูก skip**

```bash
go test ./internal/services/price-service/ -v 2>&1 | grep -c "^--- SKIP" || echo "ไม่มี test ถูก skip"
```

Expected: `ไม่มี test ถูก skip`

- [ ] **Step 6: commit**

```bash
gofmt -l internal/services/price-service/avg_kg_stock_flow_test.go \
         internal/services/price-service/avg_kg_stock_integration_test.go
git add internal/services/price-service/avg_kg_stock_flow_test.go \
        internal/services/price-service/avg_kg_stock_integration_test.go
git commit -m "test: ครอบ flow คำนวณราคาและเส้นทางการแสดงผลของ avg_kg_stock

flow test พิสูจน์ว่าสูตร pcs อ่านราคาต่อกิโลที่เพิ่งคำนวณ (55) ไม่ใช่ค่าเก่า (45)
และไม่ใช่น้ำหนักสต็อก (0) พร้อมเคสที่ลำดับสูตรส่งมาสลับ และเรียกซ้ำ 20 ครั้ง

integration test ยืนยันว่า avg_product กระจายเท่ากันทุก row ขณะที่ avg_weight
ยังเป็นค่าของ batch ตัวเอง

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 11: รายงาน F5 — subgroup ที่ผูกสูตรไม่ครบคู่

**Files:**
- Create: `docs/superpowers/reports/2026-09-10-f5-subgroup-formula-pairing.sql`

บริบท: 147 subgroup ผูกสูตร uom `pcs` ทั้งสองตัว ไม่มีสูตร uom `kg`
131 ตัวผูกสูตรเดียวกันซ้ำสองครั้ง เป็นปัญหาข้อมูลที่ธุรกิจต้องตัดสิน **งานนี้ไม่แก้ข้อมูล**

- [ ] **Step 1: เขียน query รายงาน**

สร้าง `docs/superpowers/reports/2026-09-10-f5-subgroup-formula-pairing.sql`

```sql
-- รายงาน F5: subgroup ที่ผูกสูตรราคาไม่ครบคู่
--
-- แต่ละ subgroup ควรผูกสูตร uom 'kg' หนึ่งตัวและ uom 'pcs' หนึ่งตัว
-- เพื่อให้ทั้ง total_net_price_weight และ total_net_price_unit ถูกคำนวณ
--
-- read-only ไม่แก้ข้อมูลใด ๆ
-- ใช้กับ database prime_erp

-- 1) สรุปภาพรวมของการจับคู่ uom
SELECT
    uom_combo,
    formula_names,
    count(*) AS subgroups
FROM (
    SELECT
        m.price_list_subgroup_code,
        string_agg(f.uom, '+' ORDER BY f.uom) AS uom_combo,
        string_agg(f.name, '  ||  ' ORDER BY f.uom, f.name) AS formula_names
    FROM price_list_subgroup_formulas_map m
    JOIN price_list_formulas f ON f.formula_code = m.price_list_formulas_code
    GROUP BY m.price_list_subgroup_code
) t
GROUP BY uom_combo, formula_names
ORDER BY subgroups DESC;

-- 2) รายการ subgroup ที่ไม่มีสูตร uom 'kg' เลย
SELECT
    m.price_list_subgroup_code,
    string_agg(f.name, '  ||  ' ORDER BY f.name) AS formulas_bound,
    s.total_net_price_weight,
    s.total_net_price_unit
FROM price_list_subgroup_formulas_map m
JOIN price_list_formulas f ON f.formula_code = m.price_list_formulas_code
LEFT JOIN price_list_sub_group s ON s.subgroup_code = m.price_list_subgroup_code
GROUP BY m.price_list_subgroup_code, s.total_net_price_weight, s.total_net_price_unit
HAVING count(*) FILTER (WHERE f.uom = 'kg') = 0
ORDER BY m.price_list_subgroup_code;

-- 3) รายการ subgroup ที่ผูกสูตรเดียวกันซ้ำมากกว่าหนึ่งครั้ง
SELECT
    m.price_list_subgroup_code,
    f.name AS duplicated_formula,
    f.uom,
    count(*) AS times_bound
FROM price_list_subgroup_formulas_map m
JOIN price_list_formulas f ON f.formula_code = m.price_list_formulas_code
GROUP BY m.price_list_subgroup_code, f.name, f.uom
HAVING count(*) > 1
ORDER BY times_bound DESC, m.price_list_subgroup_code;
```

- [ ] **Step 2: รัน query เพื่อยืนยันว่าใช้งานได้**

```bash
PGPASSWORD='H3dPq8Tz1Lm6Rk9VbY' psql -h 18.138.69.85 -U dev_champ -d prime_erp \
  -f docs/superpowers/reports/2026-09-10-f5-subgroup-formula-pairing.sql 2>&1 | head -40
```

Expected: query ทั้ง 3 ชุดรันผ่าน · ชุดที่ 1 แสดง `pcs+pcs` จำนวน 131 และ 16
· ชุดที่ 3 แสดงรายการที่ `times_bound = 2`

- [ ] **Step 3: commit**

```bash
mkdir -p docs/superpowers/reports
git add docs/superpowers/reports/2026-09-10-f5-subgroup-formula-pairing.sql
git commit -m "docs: query รายงาน subgroup ที่ผูกสูตรราคาไม่ครบคู่ (F5)

147 subgroup ผูกสูตร uom pcs ทั้งคู่ ไม่มีสูตร uom kg
131 ตัวผูกสูตรเดียวกันซ้ำสองครั้ง ทำให้ total_net_price_weight ไม่ถูกคำนวณ

read-only ไม่แก้ข้อมูล ให้ธุรกิจตัดสินว่าแต่ละกลุ่มควรผูกสูตร kg ตัวไหน

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 12: ตรวจ coverage และเปิด PR

**Files:** ไม่มีไฟล์ถูกแก้

- [ ] **Step 1: ตรวจ coverage ของ `warehouse-core`**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-warehouse-core
go test ./internal/services/inventory-service/ -coverprofile=/tmp/wh-cover.out 2>&1 | tail -5
go tool cover -func=/tmp/wh-cover.out | grep -E "aggregateInventoryWeights|safeAverage|total:"
```

Expected: `aggregateInventoryWeights` ≥ 80%

ถ้าต่ำกว่า 80% ให้เพิ่ม test เคสที่ยังไม่ถูกครอบ แล้ว commit เพิ่ม

- [ ] **Step 2: ตรวจ coverage ของ `erp-core`**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
go test ./internal/services/price-service/... -coverprofile=/tmp/erp-cover.out 2>&1 | tail -8
go tool cover -func=/tmp/erp-cover.out | grep -E "sortFormulasByDependency|formulaRequiredVars|getAvgKgStockFromInventory|patternHasBatchColumn|applyInventoryFieldsToRow|total:"
```

Expected: ฟังก์ชันใหม่ทุกตัว ≥ 80%

- [ ] **Step 3: รัน test ทั้งหมดของทั้ง 2 repo ครั้งสุดท้าย**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-warehouse-core && go build ./... && go vet ./... 2>&1 | tail -5 && go test ./... 2>&1 | grep -vE "no test files" | tail -20
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core && go build ./... && go vet ./... 2>&1 | tail -5 && make test 2>&1 | grep -vE "no test files" | tail -20 && make test-integration-pricelist 2>&1 | tail -15
```

Expected: build ผ่านทั้งคู่ · vet ไม่มี error · test ทั้งหมด PASS · ไม่มี SKIP

- [ ] **Step 4: ตรวจว่าไม่มี placeholder หลงเหลือในโค้ดที่แก้**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
git diff origin/Develop --name-only | xargs grep -nE "TODO|FIXME|TODO_REPLACE|t\.Skip|\.only\(" 2>/dev/null || echo "ไม่มี placeholder"
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-warehouse-core
git diff origin/Develop --name-only | xargs grep -nE "TODO|FIXME|t\.Skip" 2>/dev/null || echo "ไม่มี placeholder"
```

Expected: `ไม่มี placeholder` ทั้งสอง repo

- [ ] **Step 5: ตรวจว่า diff ไม่บวมจาก gofmt**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core && git diff origin/Develop --stat | tail -5
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-warehouse-core && git diff origin/Develop --stat | tail -5
```

ตรวจว่าไม่มีไฟล์ที่ไม่เกี่ยวข้องโผล่มา ถ้ามีไฟล์ที่มีแต่การเปลี่ยน whitespace
ให้ `git checkout origin/Develop -- <ไฟล์นั้น>`

`cmd/.env` ต้อง **ไม่** อยู่ใน diff

- [ ] **Step 6: push `warehouse-core` และเปิด PR**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-warehouse-core
git push -u origin feat/pricelist-avg-kg-stock-site-level
gh pr create --base Develop --title "fix: เพิ่ม avg_product ระดับ site และทำ inventory weight ให้ deterministic" --body "$(cat <<'BODY'
## ปัญหา

`get-inventory-weight-by-key` รวมยอดถึงระดับ batch แล้วคืน slice ที่ไม่ถูก sort
การ iterate map ใน Go สุ่มลำดับ ทำให้ผู้เรียกฝั่ง `erp-core` ที่หยิบ entry แรก
ได้ค่าของ batch แบบสุ่มทุกครั้ง

ตัวอย่างจริง `RBB610SR24TEF` @ `TMI_WH` มี 7 batch ค่าเฉลี่ยที่ถูกต้องคือ `2.9969`
แต่ค่าที่หยิบได้กระจายจาก `0` ถึง `32` ทำให้ราคาที่คำนวณต่างกันได้ 32 เท่า

## การแก้

- เพิ่ม field `avg_product` = น้ำหนักรวม / จำนวนรวม ระดับ `company|site|product`
  เป็นค่าเดียวกันทุก entry จึงปลอดภัยต่อการหยิบ entry แรก
- `sort` ผลลัพธ์ตาม `Key` ให้ลำดับนิ่ง
- คง `avg_weight` ระดับ batch ไว้เพราะ pattern ที่แสดงคอลัมน์ `batch_no` ต้องใช้
- แยก logic รวมยอดเป็นฟังก์ชัน pure `aggregateInventoryWeights` เพื่อทดสอบได้
- ใช้ `safeAverage` ที่มีอยู่แล้วแทน guard `qty > 0` เพื่อกัน `Inf` / `NaN`

## ผลกระทบต่อระบบอื่น

`GetInventoryWeightByKey` มีผู้เรียกเพียง 4 จุด ทั้งหมดอยู่ใน price-service ของ `erp-core`
ระบบที่ใช้ค่าเฉลี่ยระดับ batch (Transfer, Production, Picking, Packing) ใช้ endpoint
`get-inventory-weight` คนละตัว จึงไม่กระทบ

การเพิ่ม field เป็น backward compatible — client เดิมที่ไม่รู้จัก `avg_product` ยังทำงานได้

## Test

unit test 7 เคส รวมเคสที่เรียกซ้ำ 50 ครั้งเพื่อยืนยันว่าลำดับนิ่ง (fail บนโค้ดเดิม)

🤖 Generated with [Claude Code](https://claude.com/claude-code)

https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC
BODY
)"
```

- [ ] **Step 7: push `erp-core` และเปิด PR**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
git push -u origin feat/pricelist-avg-kg-stock-site-level
gh pr create --base Develop --title "fix: Avg. kg stock ระดับ site และแก้สูตรราคาที่ป้อนค่าผิดประเภท" --body "$(cat <<'BODY'
## ปัญหา

สอบสวนที่มาของคอลัมน์ `Avg. kg stock` พบบั๊ก 4 ตัว ยืนยันด้วยข้อมูลจริงจาก DB

**F1** `avg_kg_stock` อ่านค่าเฉลี่ยระดับ batch จาก entry แรกของ slice ที่ไม่ถูก sort
ทำให้ได้ batch แบบสุ่ม ราคาเปลี่ยนทุกครั้งที่กดคำนวณ และถูกบันทึกลง DB (990 subgroup)

**F2** ตัวแปร `pcs` และ `kg` ในสูตรหมายถึงราคาต่อชิ้นและราคาต่อกิโล แต่โค้ดป้อน
`TotalQty` และ `TotalWeight` ของสต็อกเข้าไป ผลคือ **694 จาก 712 subgroup
มี `total_net_price_unit = 0`** และที่เหลือได้ค่าสูงถึง `632,016` ขณะที่ราคาต่อกิโลอยู่ราว `19`
(1,203 subgroup)

**F3** สูตร `kg = [Pcs] / [Weight Spec]` มี expression `pcs*weight_spec` ชื่อบอกหารแต่โค้ดคูณ

**F4** สูตรถูกประเมินตามลำดับ `create_dtm DESC` ไม่ใช่ลำดับ dependency
ทำให้ 522 subgroup ประเมินสูตร `pcs` ก่อนที่ `kg` จะถูกคำนวณใหม่
แล้วอ่านค่าของรอบก่อนหน้า ต้องกดคำนวณสองครั้งจึงจะตรง

## การแก้

- `avgKgStock` อ่าน `AvgProduct` (ระดับ site) แทน `AvgWeight` (ระดับ batch)
- `Pcs` / `Kg` ใน env อ่านจากตัวแปร running `totalNetPriceUnit` / `totalNetPriceWeight`
- `sortFormulasByDependency` เรียงสูตรจาก `params.required` ก่อนเข้าลูปประเมิน
- แยก `getAvgKgStockFromInventory(sg, perBatch)` โดยตัดสินจาก config ว่า pattern มีคอลัมน์
  `batch_no` หรือไม่ ไม่ hardcode ชื่อ pattern
- migration แก้ expression ของ F3 แบบ idempotent พร้อมตรวจผลในตัว
- row builder ทั้ง 3 ตัวคืน `0` เมื่อไม่มีสต็อก (เดิมต่างกัน 3 แบบ)
- ลบการเขียนค่าแบบมีเงื่อนไข `if inv.X > 0` ที่ทำให้ค่าค้าง และลบ dead write `AvgBatch`

## ลำดับ deploy

**ต้องขึ้นหลัง** `prime-wms-warehouse-core` PR ที่เพิ่ม field `avg_product`
ระหว่างที่ warehouse ยังไม่ขึ้น `AvgProduct` จะเป็น 0 แล้วตกไป fallback `1.0`
ซึ่งเป็นพฤติกรรมเดียวกับกรณีไม่มีสต็อก ไม่ทำให้ระบบล่ม แต่ราคายังไม่ถูก

## ไม่อยู่ใน PR นี้

- **F5** 147 subgroup ผูกสูตร uom `pcs` ทั้งคู่ ไม่มีสูตร `kg` เลย — ส่ง query รายงานให้ธุรกิจตัดสิน
- คอลัมน์ `avg_weight_ton` ที่ header เขียน "(Tons)" แต่ `dataMapping` ชี้ไป `avg_weight`
  ซึ่งเป็น kg และไม่พบการหารด้วย 1000 ที่ใดในระบบ — รอคำตอบจากธุรกิจ
- ไม่ backfill ราคาที่คำนวณผิดไปแล้วใน DB — รอการตัดสินใจเรื่อง recalculate ย้อนหลัง
- fallback `1.0` เมื่อไม่มีสต็อกคงไว้ตามที่ตกลง

## Test

unit test ของการเรียงลำดับสูตร การอ่าน avg และการคำนวณราคาทั้ง 4 สูตรที่ใช้ `avg_kg_stock`
integration test ยืนยันว่าคำนวณซ้ำ 20 ครั้งได้ผลเท่ากัน และลำดับสูตรที่สลับมายังให้ผลถูก
ตั้งแต่การกดคำนวณครั้งแรก

🤖 Generated with [Claude Code](https://claude.com/claude-code)

https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC
BODY
)"
```

- [ ] **Step 8: อัปเดต knowledge graph**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms && graphify update .
```

Expected: อัปเดตสำเร็จ ไม่มี error

---

## ลำดับการทำงานและ dependency ระหว่าง task

```
Task 1 (branch warehouse) -> Task 2 (AvgProduct + sort)
Task 3 (branch erp) -> Task 4 (shared.go)
                    -> Task 5 (formula order) -> Task 6 (avgKgStock + running values)
                    -> Task 7 (get-price-detail)
                    -> Task 8 (row builders)
                    -> Task 9 (migration F3)
Task 4..9 -> Task 10 (flow test + integration test) -> Task 12 (coverage + PR)
Task 11 (รายงาน F5) ทำเมื่อไหร่ก็ได้ ไม่มี dependency
```

**Task 5 ต้องเสร็จก่อน Task 6** และทั้งคู่ต้องอยู่ใน PR เดียวกัน
แก้ Task 6 อย่างเดียวโดยไม่มี Task 5 จะทำให้ 522 subgroup อ่านค่า running ของรอบก่อนหน้า
ซึ่งผิดในรูปแบบใหม่แทนที่จะหายไป

**Task 2 ต้องขึ้น environment ก่อน Task 6 ทำงานได้จริง** เพราะ `AvgProduct` มาจาก response
ของ `warehouse-core`
