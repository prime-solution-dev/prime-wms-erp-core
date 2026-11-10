# Price List Weight-spec from Product Master — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** ทำให้คอลัมน์และตัวแปร `Weight-spec` ของ price list ดึงค่าจาก `product_unit.weight` ของ unit ที่ `flag_base = true` ใน product-master แทนค่าที่ผิดอยู่ตอนนี้ โดยต้องมีค่าแม้สินค้าไม่มีสต็อก แสดงบน web ได้ return ทาง API ได้ และใช้ในสูตรคำนวณได้

**Architecture:** `warehouse-core` มี combination matching ของ subgroup key อยู่แล้ว (`get-inventory-weight-by-key.go:136-179`) และตอน match สำเร็จมี `matchedProduct.Units[]` (มี `Weight` + `FlagBase`) อยู่ในหน่วยความจำแล้ว จึงเติม `weight_spec` ลง response ระดับ result (ระดับเดียวกับ `product_code` ไม่ใช่ใน `inventory_weight[]` ที่จะว่างเมื่อไม่มีสต็อก) แล้วให้ `erp-core` พาค่านั้นไปที่ subgroup แล้วแก้จุดอ่านค่าให้เหลือแหล่งเดียว ฝั่ง web ไม่ต้องแก้เพราะกริดสร้างคอลัมน์จาก config ที่ backend ส่งมา

**Tech Stack:** Go 1.22 (warehouse-core) / Go 1.25 (erp-core), Gin, GORM + sqlx, `expr` (formula engine), testing (stdlib), testcontainers (integration)

**Spec:** `prime-wms-erp-core/docs/superpowers/specs/2026-09-10-pricelist-weight-spec-from-product-master-design.md`

**Branch (แตกจาก Develop แล้วทั้ง 2 repo):** `feat/pricelist-weight-spec-from-product-master`

**สำคัญ:** ห้าม merge เข้า Develop ตรง ๆ / ห้ามแก้ branch Develop / ห้ามยุ่ง branch Crossmax-uat, shi-sit, Pacifica-uat, Pacifica-main, Thaimetal-uat, shi-main

---

## Background — ทำไมค่าถึงผิด (อ่านก่อนเริ่ม)

ตอนนี้มี **3 แหล่ง** ที่ต่างกัน และผิดทั้งหมด:

| ผู้อ่าน | อ่านจาก | ค่าที่ได้จริง |
|---|---|---|
| กริด price list detail — `patterns/shared.go:532` | `sg.InventoryWeight[0].WeightSpec` | `warehouse-core` ไม่เคย assign field นี้ → `0` เสมอ |
| export table — `get-price-export-table.go:448` | `inv.TotalWeight` ใส่ลง `row["total_weight"]` โดยตรง (ไม่ผ่าน `getWeightSpecFromInventory`) | น้ำหนักรวมสต็อก เช่น 1,500,000 kg |
| สูตรคำนวณ — `get-calculated-pricelist-subgroup.go:224-228`, `update-latest-pricelist-subgroup.go:230-246` | `inventoryWeight[0].TotalWeight` | ตัวเดียวกับตัวแปร `kg` เป๊ะ ๆ |

คอลัมน์ที่ผู้ใช้เห็นชื่อ `"Weight-spec"` ผูกกับ field ชื่อ `total_weight` ใน pattern config (ชื่อชวนสับสน แต่**ห้ามเปลี่ยนชื่อ field** — จะพัง pattern config 22 ไฟล์ และโค้ดฝั่ง web ที่อ้าง suffix)

**มี struct subgroup 2 ตัว** ที่ต้องแก้ทั้งคู่ (คนละ flow):
- `models.PriceListSubGroupResponse` (`internal/models/pricelist.go:304`) — ใช้โดย `get-price-detail.go` + `patterns/shared.go`
- `priceService.SubGroup` (`internal/services/price-service/get-pricelist.go:84`) — ใช้โดย `get-price-export-table.go`

---

## File Structure

### prime-wms-warehouse-core

| ไฟล์ | หน้าที่ | สถานะ |
|---|---|---|
| `internal/services/inventory-service/weight-spec.go` | ฟังก์ชัน pure `WeightSpecFromUnits()` หา weight ของ base unit — แยกออกมาเพื่อให้ unit test ได้โดยไม่ต้องมี DB/HTTP | **สร้างใหม่** |
| `internal/services/inventory-service/weight_spec_test.go` | unit test ของฟังก์ชันข้างบน | **สร้างใหม่** |
| `internal/services/inventory-service/get-inventory-weight-by-key.go` | เพิ่ม field `WeightSpec` ใน result struct + เรียก `WeightSpecFromUnits` | แก้ไข |

### prime-wms-erp-core

| ไฟล์ | หน้าที่ | สถานะ |
|---|---|---|
| `external/warehouse-service/get-inventory-by-product-code.go` | รับ `weight_spec` จาก warehouse | แก้ไข `:31-39` |
| `internal/models/pricelist.go` | `WeightSpec` บน `PriceListSubGroupResponse` + `GetCalculatedPriceListSubGroupItem` | แก้ไข `:304`, `:496` |
| `internal/services/price-service/get-pricelist.go` | `WeightSpec` บน `SubGroup` | แก้ไข `:84-115` |
| `internal/services/price-service/domain/domain.go` | `WeightSpec` บน `Price` | แก้ไข |
| `internal/services/price-service/weight-spec.go` | `weightSpecForFormula()` — fallback 1.0 สำหรับสูตร ใช้ร่วมกัน 2 ที่ | **สร้างใหม่** |
| `internal/services/price-service/weight_spec_test.go` | unit test ของ fallback | **สร้างใหม่** |
| `internal/services/price-service/patterns/shared.go` | `getWeightSpecFromInventory()` อ่านจาก `sg.WeightSpec` | แก้ไข `:530-537` |
| `internal/services/price-service/patterns/inventory_udf_test.go` | แก้เทสเดิมให้ตรงแหล่งใหม่ | แก้ไข `:15-30` |
| `internal/services/price-service/get-price-detail.go` | พา weight_spec ลง subgroup (ทั้งเคสมีและไม่มีสต็อก) | แก้ไข `:279-330` |
| `internal/services/price-service/get-price-export-table.go` | พา weight_spec + `row["total_weight"]` ใช้ค่าใหม่ | แก้ไข `:150-176`, `:446-448` |
| `internal/services/price-service/get-calculated-pricelist-subgroup.go` | สูตรใช้ค่าใหม่ + response | แก้ไข `:179-232`, `:250-290` |
| `internal/services/price-service/update-latest-pricelist-subgroup.go` | สูตรใช้ค่าใหม่ + response | แก้ไข `:197-246` |

### prime-wms-web
ไม่มีไฟล์ที่ต้องแก้

---

## Task 1: warehouse-core — ฟังก์ชันหา weight ของ base unit

**Files:**
- Create: `prime-wms-warehouse-core/internal/services/inventory-service/weight-spec.go`
- Test: `prime-wms-warehouse-core/internal/services/inventory-service/weight_spec_test.go`

**Context:** `productService` เป็น alias ของ `wms-service-warehouse/external/services/product` (ดู import ที่ `get-inventory-weight-by-key.go:10`) struct `GetUnitsDetailComponent` อยู่ที่ `external/services/product/get-product-detail-master.go:101-119` มี `Weight float64` (`:109`) และ `FlagBase bool` (`:110`)

- [ ] **Step 1: เขียน test ที่ต้อง fail ก่อน**

สร้าง `prime-wms-warehouse-core/internal/services/inventory-service/weight_spec_test.go`:

```go
package inventoryService

import (
	"testing"

	productService "wms-service-warehouse/external/services/product"
)

// Weight-spec คือน้ำหนักต่อหน่วยของ base unit (flag_base = true) ใน product master
// ไม่ใช่ unit ตัวแรก และไม่ใช่ค่าน้ำหนักรวมของสต็อก
func TestWeightSpecFromUnits(t *testing.T) {
	tests := []struct {
		name  string
		units []productService.GetUnitsDetailComponent
		want  float64
	}{
		{
			name: "หยิบ unit ที่ flag_base = true ไม่ใช่ตัวแรก",
			units: []productService.GetUnitsDetailComponent{
				{UnitCode: "BOX", Weight: 250.0, FlagBase: false},
				{UnitCode: "PCS", Weight: 12.5, FlagBase: true},
			},
			want: 12.5,
		},
		{
			name: "base unit อยู่ตัวแรก",
			units: []productService.GetUnitsDetailComponent{
				{UnitCode: "PCS", Weight: 12.5, FlagBase: true},
				{UnitCode: "BOX", Weight: 250.0, FlagBase: false},
			},
			want: 12.5,
		},
		{
			name: "ไม่มี unit ไหน flag_base = true",
			units: []productService.GetUnitsDetailComponent{
				{UnitCode: "BOX", Weight: 250.0, FlagBase: false},
			},
			want: 0,
		},
		{
			name:  "ไม่มี unit เลย",
			units: []productService.GetUnitsDetailComponent{},
			want:  0,
		},
		{
			name:  "units เป็น nil",
			units: nil,
			want:  0,
		},
		{
			name: "base unit มี weight เป็น 0",
			units: []productService.GetUnitsDetailComponent{
				{UnitCode: "PCS", Weight: 0, FlagBase: true},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WeightSpecFromUnits(tt.units); got != tt.want {
				t.Fatalf("WeightSpecFromUnits() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: รัน test ให้เห็นว่า fail**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-warehouse-core
go test ./internal/services/inventory-service/ -run TestWeightSpecFromUnits -v
```

Expected: FAIL — `undefined: WeightSpecFromUnits`

- [ ] **Step 3: เขียน implementation ให้น้อยที่สุด**

สร้าง `prime-wms-warehouse-core/internal/services/inventory-service/weight-spec.go`:

```go
package inventoryService

import (
	productService "wms-service-warehouse/external/services/product"
)

// WeightSpecFromUnits คืนน้ำหนักต่อหน่วยของ base unit (flag_base = true) จาก product master
// ซึ่งเป็นค่าที่ price list ใช้เป็น "Weight-spec"
// คืน 0 เมื่อไม่มี unit ไหนเป็น base unit — ฝั่งผู้เรียกเป็นผู้ตัดสินใจว่าจะ fallback อย่างไร
func WeightSpecFromUnits(units []productService.GetUnitsDetailComponent) float64 {
	for _, u := range units {
		if u.FlagBase {
			return u.Weight
		}
	}
	return 0
}
```

- [ ] **Step 4: รัน test ให้ผ่าน**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-warehouse-core
go test ./internal/services/inventory-service/ -run TestWeightSpecFromUnits -v
```

Expected: PASS ทั้ง 6 subtest

- [ ] **Step 5: ตรวจ format และ vet**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-warehouse-core
gofmt -l internal/services/inventory-service/
go vet ./internal/services/inventory-service/
```

Expected: `gofmt -l` ไม่พิมพ์ชื่อไฟล์ออกมา, `go vet` ไม่มี output

- [ ] **Step 6: Commit**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-warehouse-core
git add internal/services/inventory-service/weight-spec.go internal/services/inventory-service/weight_spec_test.go
git commit -m "feat(inventory): เพิ่ม WeightSpecFromUnits หาน้ำหนักของ base unit

Weight-spec ของ price list คือ product_unit.weight ของ unit ที่
flag_base = true แยกเป็นฟังก์ชัน pure เพื่อให้ unit test ได้

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 2: warehouse-core — ส่ง weight_spec ออกทาง API

**Files:**
- Modify: `prime-wms-warehouse-core/internal/services/inventory-service/get-inventory-weight-by-key.go:43-52` (struct) และ `:236-257` (result build)

**Context:** endpoint นี้คือ `get-inventory-weight-by-key` ที่ `erp-core` เรียกผ่าน `config.GET_INVENTORY_BY_KEY_ENDPOINT`
ตัว `matchedProduct` เป็น `*productService.GetProductsDetailComponent` ที่มี field `Units []GetUnitsDetailComponent`
บล็อกที่จะแก้มี 2 branch: `if matchedProduct != nil` (บรรทัด ~237) และ `else` (บรรทัด ~246)

- [ ] **Step 1: เพิ่ม field ใน result struct**

แก้ `internal/services/inventory-service/get-inventory-weight-by-key.go` struct `GetInventoryWeightByKeyResult` (บรรทัด 43-52) จาก:

```go
// Grouped result per KeyValue ID
type GetInventoryWeightByKeyResult struct {
	ID              string                  `json:"id"`
	GroupCodeKeys   string                  `json:"group_code_keys"`
	GroupValueKeys  string                  `json:"group_value_keys"`
	ProductCode     string                  `json:"product_code"`
	SupplierCode    string                  `json:"supplier_code"`
	SupplierName    string                  `json:"supplier_name"`
	InventoryWeight []InventoryWeightDetail `json:"inventory_weight"`
}
```

เป็น:

```go
// Grouped result per KeyValue ID
type GetInventoryWeightByKeyResult struct {
	ID              string                  `json:"id"`
	GroupCodeKeys   string                  `json:"group_code_keys"`
	GroupValueKeys  string                  `json:"group_value_keys"`
	ProductCode     string                  `json:"product_code"`
	SupplierCode    string                  `json:"supplier_code"`
	SupplierName    string                  `json:"supplier_name"`
	// WeightSpec คือน้ำหนักของ base unit จาก product master วางไว้ระดับ result
	// (ไม่ใช่ใน InventoryWeight) เพราะต้องมีค่าแม้สินค้าไม่มีสต็อก ซึ่งกรณีนั้น
	// InventoryWeight จะเป็น array ว่าง
	WeightSpec      float64                 `json:"weight_spec"`
	InventoryWeight []InventoryWeightDetail `json:"inventory_weight"`
}
```

- [ ] **Step 2: เติมค่าใน branch ที่ match product ได้**

ในไฟล์เดียวกัน หาบล็อกที่ขึ้นต้นด้วย `supplierName := ""` (~บรรทัด 232) แล้วตามด้วย `results = append(results, GetInventoryWeightByKeyResult{`
แก้จาก:

```go
			supplierName := ""
			if matchedProduct.SupplierCode != "" {
				supplierName = supplierNameMap[matchedProduct.SupplierCode]
			}

			results = append(results, GetInventoryWeightByKeyResult{
				ID:              kvGroup.ID,
				GroupCodeKeys:   kvGroup.GroupCodeKeys,
				GroupValueKeys:  kvGroup.GroupValueKeys,
				ProductCode:     matchedProduct.ProductCode,
				SupplierCode:    matchedProduct.SupplierCode,
				SupplierName:    supplierName,
				InventoryWeight: productWeights,
			})
```

เป็น:

```go
			supplierName := ""
			if matchedProduct.SupplierCode != "" {
				supplierName = supplierNameMap[matchedProduct.SupplierCode]
			}

			results = append(results, GetInventoryWeightByKeyResult{
				ID:              kvGroup.ID,
				GroupCodeKeys:   kvGroup.GroupCodeKeys,
				GroupValueKeys:  kvGroup.GroupValueKeys,
				ProductCode:     matchedProduct.ProductCode,
				SupplierCode:    matchedProduct.SupplierCode,
				SupplierName:    supplierName,
				WeightSpec:      WeightSpecFromUnits(matchedProduct.Units),
				InventoryWeight: productWeights,
			})
```

- [ ] **Step 3: เติมค่า 0 ใน branch ที่ match product ไม่ได้**

ในไฟล์เดียวกัน บล็อก `else` ถัดลงไป แก้จาก:

```go
		} else {
			// No matched product, return result with empty product_code and inventory_weight
			results = append(results, GetInventoryWeightByKeyResult{
				ID:              kvGroup.ID,
				GroupCodeKeys:   kvGroup.GroupCodeKeys,
				GroupValueKeys:  kvGroup.GroupValueKeys,
				ProductCode:     "",
				SupplierCode:    "",
				SupplierName:    "",
				InventoryWeight: []InventoryWeightDetail{},
			})
		}
```

เป็น:

```go
		} else {
			// No matched product, return result with empty product_code and inventory_weight
			results = append(results, GetInventoryWeightByKeyResult{
				ID:              kvGroup.ID,
				GroupCodeKeys:   kvGroup.GroupCodeKeys,
				GroupValueKeys:  kvGroup.GroupValueKeys,
				ProductCode:     "",
				SupplierCode:    "",
				SupplierName:    "",
				WeightSpec:      0,
				InventoryWeight: []InventoryWeightDetail{},
			})
		}
```

- [ ] **Step 4: build และ vet**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-warehouse-core
go build ./... && go vet ./internal/services/inventory-service/ && gofmt -l internal/services/inventory-service/
```

Expected: ไม่มี output ใด ๆ (build ผ่าน, vet สะอาด, format ถูก)

- [ ] **Step 5: รัน test ทั้ง package**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-warehouse-core
go test ./internal/services/inventory-service/ -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-warehouse-core
git add internal/services/inventory-service/get-inventory-weight-by-key.go
git commit -m "feat(inventory): ส่ง weight_spec ของ base unit ออกทาง get-inventory-weight-by-key

วางไว้ระดับ result ไม่ใช่ใน inventory_weight[] เพราะต้องมีค่าแม้สินค้า
ไม่มีสต็อก ซึ่งกรณีนั้น inventory_weight จะเป็น array ว่าง
matchedProduct.Units อยู่ในหน่วยความจำอยู่แล้วจึงไม่ต้องเพิ่ม HTTP call

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 3: erp-core — รับ weight_spec จาก warehouse

**Files:**
- Modify: `prime-wms-erp-core/external/warehouse-service/get-inventory-by-product-code.go:31-39`

- [ ] **Step 1: เพิ่ม field ใน response struct**

แก้ `external/warehouse-service/get-inventory-by-product-code.go` จาก:

```go
// InventoryByProductCodeResponse represents the response structure from inventory service
type InventoryByProductCodeResponse struct {
	ID              string                           `json:"id"`
	GroupCodeKeys   string                           `json:"group_code_keys"`
	GroupValueKeys  string                           `json:"group_value_keys"`
	ProductCode     string                           `json:"product_code"`
	SupplierCode    string                           `json:"supplier_code"`
	SupplierName    string                           `json:"supplier_name"`
	InventoryWeight []models.InventoryWeightResponse `json:"inventory_weight"`
}
```

เป็น:

```go
// InventoryByProductCodeResponse represents the response structure from inventory service
type InventoryByProductCodeResponse struct {
	ID              string                           `json:"id"`
	GroupCodeKeys   string                           `json:"group_code_keys"`
	GroupValueKeys  string                           `json:"group_value_keys"`
	ProductCode     string                           `json:"product_code"`
	SupplierCode    string                           `json:"supplier_code"`
	SupplierName    string                           `json:"supplier_name"`
	// WeightSpec คือน้ำหนักของ base unit (flag_base = true) จาก product master
	// มีค่าแม้สินค้าไม่มีสต็อก ต่างจาก InventoryWeight ที่จะว่างเมื่อไม่มีสต็อก
	WeightSpec      float64                          `json:"weight_spec"`
	InventoryWeight []models.InventoryWeightResponse `json:"inventory_weight"`
}
```

- [ ] **Step 2: build**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
go build ./... && gofmt -l external/warehouse-service/
```

Expected: ไม่มี output

- [ ] **Step 3: Commit**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
git add external/warehouse-service/get-inventory-by-product-code.go
git commit -m "feat(price-service): รับ weight_spec จาก warehouse inventory endpoint

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 4: erp-core — เพิ่ม WeightSpec ลง struct ที่เกี่ยวข้องทั้งหมด

**Files:**
- Modify: `prime-wms-erp-core/internal/models/pricelist.go:304-340` (`PriceListSubGroupResponse`) และ `:496-505` (`GetCalculatedPriceListSubGroupItem`)
- Modify: `prime-wms-erp-core/internal/services/price-service/get-pricelist.go:84-115` (`SubGroup`)
- Modify: `prime-wms-erp-core/internal/services/price-service/domain/domain.go` (`Price`)

**Context:** มี struct subgroup 2 ตัวคนละ flow ต้องแก้ทั้งคู่ (ดูหัวข้อ Background)

- [ ] **Step 1: `PriceListSubGroupResponse`**

แก้ `internal/models/pricelist.go` — ในบล็อก `type PriceListSubGroupResponse struct` หาบรรทัด:

```go
	ProductCode               string                         `json:"product_code,omitempty"`
```

เพิ่มบรรทัดถัดจากมัน:

```go
	// WeightSpec คือน้ำหนักของ base unit จาก product master — เก็บระดับ subgroup
	// ไม่ใช่ใน InventoryWeight เพราะต้องมีค่าแม้ subgroup นั้นไม่มีสต็อก
	WeightSpec                float64                        `json:"weight_spec"`
```

- [ ] **Step 2: `GetCalculatedPriceListSubGroupItem`**

แก้ `internal/models/pricelist.go` struct `GetCalculatedPriceListSubGroupItem` (~บรรทัด 496) จาก:

```go
type GetCalculatedPriceListSubGroupItem struct {
	SubGroupID                string  `json:"subgroup_id"`
	TotalNetPriceUnit         float64 `json:"total_net_price_unit"`
	TotalNetPriceWeight       float64 `json:"total_net_price_weight"`
	ExtraPriceUnit            float64 `json:"extra_price_unit"`
	ExtraPriceWeight          float64 `json:"extra_price_weight"`
	DefaultUom                string  `json:"default_uom,omitempty"`
}
```

เป็น (คงบรรทัดอื่นที่มีอยู่เดิมไว้ทั้งหมด เพิ่มเฉพาะ `WeightSpec`):

```go
type GetCalculatedPriceListSubGroupItem struct {
	SubGroupID                string  `json:"subgroup_id"`
	TotalNetPriceUnit         float64 `json:"total_net_price_unit"`
	TotalNetPriceWeight       float64 `json:"total_net_price_weight"`
	ExtraPriceUnit            float64 `json:"extra_price_unit"`
	ExtraPriceWeight          float64 `json:"extra_price_weight"`
	WeightSpec                float64 `json:"weight_spec"`
	DefaultUom                string  `json:"default_uom,omitempty"`
}
```

**หมายเหตุ:** struct จริงในไฟล์มี `BeforeTotalNetPriceUnit` / `BeforeTotalNetPriceWeight` ด้วย (ดูจุดที่สร้าง literal ที่ `get-calculated-pricelist-subgroup.go:309-318`) — ให้เพิ่ม `WeightSpec` เข้าไปโดยไม่ลบ field เดิมใด ๆ

- [ ] **Step 3: `SubGroup` (ตัวที่ export table ใช้)**

แก้ `internal/services/price-service/get-pricelist.go` struct `SubGroup` (บรรทัด 84-115) หาบรรทัด:

```go
	ProductCode               string          `json:"product_code,omitempty"`
```

เพิ่มบรรทัดถัดจากมัน:

```go
	WeightSpec                float64         `json:"weight_spec"`
```

- [ ] **Step 4: `domain.Price`**

แก้ `internal/services/price-service/domain/domain.go` struct `Price` จาก:

```go
type Price struct {
	Id                  string  `json:"id"`
	TotalNetPriceUnit   float64 `json:"total_net_price_unit"`
	TotalNetPriceWeight float64 `json:"total_net_price_weight"`
	ExtraPriceUnit      float64 `json:"extra_price_unit"`
	ExtraPriceWeight    float64 `json:"extra_price_weight"`
	DefaultUom          string  `json:"default_uom"`
}
```

เป็น:

```go
type Price struct {
	Id                  string  `json:"id"`
	TotalNetPriceUnit   float64 `json:"total_net_price_unit"`
	TotalNetPriceWeight float64 `json:"total_net_price_weight"`
	ExtraPriceUnit      float64 `json:"extra_price_unit"`
	ExtraPriceWeight    float64 `json:"extra_price_weight"`
	WeightSpec          float64 `json:"weight_spec"`
	DefaultUom          string  `json:"default_uom"`
}
```

- [ ] **Step 5: build**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
go build ./... && gofmt -l internal/ && go vet ./internal/services/price-service/...
```

Expected: ไม่มี output

- [ ] **Step 6: Commit**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
git add internal/models/pricelist.go internal/services/price-service/get-pricelist.go internal/services/price-service/domain/domain.go
git commit -m "feat(price-service): เพิ่ม field WeightSpec ลง struct subgroup และ response

มี struct subgroup 2 ตัวคนละ flow: models.PriceListSubGroupResponse
(get-price-detail + patterns) และ priceService.SubGroup (export table)
เพิ่มทั้งคู่ พร้อม GetCalculatedPriceListSubGroupItem และ domain.Price
ที่เดิมไม่มี field นี้เลย

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 5: erp-core — fallback 1.0 สำหรับสูตรคำนวณ

**Files:**
- Create: `prime-wms-erp-core/internal/services/price-service/weight-spec.go`
- Test: `prime-wms-erp-core/internal/services/price-service/weight_spec_test.go`

**Context:** ตามที่ตกลงใน spec — การ**แสดงผล**ใช้ `0` เพื่อให้ user เห็นว่า master data ขาด แต่**สูตรคำนวณ**ใช้ `1.0` เพื่อกันราคาเพี้ยนเป็น 0 หรือ division by zero (ตรงกับ default เดิมในโค้ด `weightSpec := 1.0`) ฟังก์ชันนี้ถูกใช้ 2 ที่ (Task 8 และ Task 9) จึงแยกออกมาไฟล์เดียว
ทั้ง `get-calculated-pricelist-subgroup.go` และ `update-latest-pricelist-subgroup.go` อยู่ package `priceService` เดียวกัน

- [ ] **Step 1: เขียน test ที่ต้อง fail ก่อน**

สร้าง `prime-wms-erp-core/internal/services/price-service/weight_spec_test.go`:

```go
package priceService

import "testing"

// weight_spec ที่หาไม่ได้จะเป็น 0 ซึ่งถ้าปล่อยเข้าสูตรจะทำให้ราคาเป็น 0
// หรือ division by zero จึงต้อง fallback เป็น 1.0 เฉพาะตอนคำนวณ
// ส่วนการแสดงผลยังต้องโชว์ 0 ตามเดิมเพื่อให้เห็นว่า master data ขาด
func TestWeightSpecForFormula(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{name: "ค่าปกติผ่านไปตรง ๆ", in: 12.5, want: 12.5},
		{name: "0 กลายเป็น 1.0", in: 0, want: 1.0},
		{name: "ค่าติดลบกลายเป็น 1.0", in: -3.2, want: 1.0},
		{name: "ค่าน้อยมากแต่เป็นบวกผ่านไปตรง ๆ", in: 0.001, want: 0.001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := weightSpecForFormula(tt.in); got != tt.want {
				t.Fatalf("weightSpecForFormula(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: รัน test ให้เห็นว่า fail**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
go test ./internal/services/price-service/ -run TestWeightSpecForFormula -v
```

Expected: FAIL — `undefined: weightSpecForFormula`

- [ ] **Step 3: เขียน implementation**

สร้าง `prime-wms-erp-core/internal/services/price-service/weight-spec.go`:

```go
package priceService

// weightSpecForFormula แปลง weight_spec ให้ปลอดภัยต่อการนำไปเข้าสูตรคำนวณราคา
//
// weight_spec มาจาก product master (น้ำหนักของ unit ที่ flag_base = true) ถ้าหาไม่เจอ
// จะได้ 0 ซึ่งในสูตรมักถูกใช้เป็นตัวคูณหรือตัวหาร ปล่อยไปจะได้ราคา 0 หรือ +Inf
// จึงแทนด้วย 1.0 (ค่าเดียวกับ default เดิมของโค้ด)
//
// ฟังก์ชันนี้ใช้เฉพาะตอนคำนวณ การแสดงผลบนกริดและใน API response ยังต้องโชว์ค่าดิบ
// เพื่อให้ผู้ใช้เห็นว่า master data ของสินค้าตัวนั้นยังไม่ครบ
func weightSpecForFormula(weightSpec float64) float64 {
	if weightSpec <= 0 {
		return 1.0
	}
	return weightSpec
}
```

- [ ] **Step 4: รัน test ให้ผ่าน**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
go test ./internal/services/price-service/ -run TestWeightSpecForFormula -v
```

Expected: PASS ทั้ง 4 subtest

- [ ] **Step 5: Commit**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
git add internal/services/price-service/weight-spec.go internal/services/price-service/weight_spec_test.go
git commit -m "feat(price-service): เพิ่ม weightSpecForFormula fallback 1.0 สำหรับสูตรคำนวณ

การแสดงผลใช้ค่าดิบ (0 เมื่อ master data ขาด) แต่สูตรคำนวณต้อง fallback
เป็น 1.0 เพราะ weight_spec ถูกใช้เป็นตัวคูณ/ตัวหาร

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 6: erp-core — ให้กริดทุก pattern อ่านจากแหล่งเดียว

**Files:**
- Modify: `prime-wms-erp-core/internal/services/price-service/patterns/shared.go:530-537`
- Modify: `prime-wms-erp-core/internal/services/price-service/patterns/inventory_udf_test.go:9-30`

**Context:** `getWeightSpecFromInventory()` ถูกเรียกจาก 8 จุดใน `shared.go` (บรรทัด 1265, 1396, 1404, 1712, 1842, 2099, 2109, 2367) ซึ่งครอบคลุม pattern config ทั้ง 22 ไฟล์ที่มีคอลัมน์ `Weight-spec` — **แก้ที่ฟังก์ชันนี้ที่เดียว ถูกหมดทุก pattern** ห้ามไปไล่แก้ที่จุดเรียกทีละจุด

- [ ] **Step 1: แก้เทสเดิมให้สะท้อนแหล่งใหม่**

แก้ `internal/services/price-service/patterns/inventory_udf_test.go` — เปลี่ยนเฉพาะ `TestGetWeightSpecFromInventory` (บรรทัด 15-30) จาก:

```go
// The "Weight-spec" column must read WeightSpec, not TotalWeight (the total
// on-hand stock weight). Reading TotalWeight showed values like 1,500,000 kg
// in a column that is meant to hold a per-unit spec weight.
func TestGetWeightSpecFromInventory(t *testing.T) {
	sg := sgWithInventory(models.InventoryWeightResponse{
		WeightSpec:  12.5,
		TotalWeight: 1500000,
	})
	if got := getWeightSpecFromInventory(sg); got != 12.5 {
		t.Fatalf("want WeightSpec 12.5, got %v", got)
	}

	if got := getWeightSpecFromInventory(models.PriceListSubGroupResponse{}); got != 0 {
		t.Fatalf("want 0 with no inventory, got %v", got)
	}
}
```

เป็น:

```go
// คอลัมน์ "Weight-spec" ต้องอ่านจาก sg.WeightSpec ซึ่งเป็นน้ำหนักของ base unit
// จาก product master ไม่ใช่จาก InventoryWeight ที่ผูกกับสต็อก
//
// เดิมอ่านจาก InventoryWeight[0].WeightSpec ซึ่ง warehouse-core ไม่เคย set ค่าเลย
// จึงได้ 0 เสมอ และค่าต้องมีแม้ subgroup นั้นไม่มีสต็อก (InventoryWeight ว่าง)
func TestGetWeightSpecFromInventory(t *testing.T) {
	sg := models.PriceListSubGroupResponse{WeightSpec: 12.5}
	if got := getWeightSpecFromInventory(sg); got != 12.5 {
		t.Fatalf("want WeightSpec 12.5, got %v", got)
	}

	// ไม่มีสต็อกเลยแต่ยังต้องได้ค่า weight spec
	noStock := models.PriceListSubGroupResponse{
		WeightSpec:      12.5,
		InventoryWeight: []models.InventoryWeightResponse{},
	}
	if got := getWeightSpecFromInventory(noStock); got != 12.5 {
		t.Fatalf("want 12.5 with no stock, got %v", got)
	}

	// มีสต็อกน้ำหนักรวมมหาศาล แต่ต้องไม่หลุดมาเป็น weight spec
	withStock := sgWithInventory(models.InventoryWeightResponse{TotalWeight: 1500000})
	withStock.WeightSpec = 12.5
	if got := getWeightSpecFromInventory(withStock); got != 12.5 {
		t.Fatalf("want 12.5, must not read TotalWeight, got %v", got)
	}

	// ไม่มีข้อมูลอะไรเลย → 0 (ฝั่งสูตรจะ fallback เป็น 1.0 เอง)
	if got := getWeightSpecFromInventory(models.PriceListSubGroupResponse{}); got != 0 {
		t.Fatalf("want 0 with no data, got %v", got)
	}
}
```

- [ ] **Step 2: รัน test ให้เห็นว่า fail**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
go test ./internal/services/price-service/patterns/ -run TestGetWeightSpecFromInventory -v
```

Expected: FAIL — subtest แรกได้ `0` แทน `12.5` เพราะฟังก์ชันยังอ่านจาก `InventoryWeight`

- [ ] **Step 3: แก้ implementation**

แก้ `internal/services/price-service/patterns/shared.go` บรรทัด 530-537 จาก:

```go
// getWeightSpecFromInventory extracts WeightSpec from the first InventoryWeight entry
// Returns 0.0 if inventory data is not available
func getWeightSpecFromInventory(sg models.PriceListSubGroupResponse) float64 {
	if len(sg.InventoryWeight) > 0 {
		return sg.InventoryWeight[0].WeightSpec
	}
	return 0.0
}
```

เป็น:

```go
// getWeightSpecFromInventory คืนน้ำหนักของ base unit (flag_base = true) จาก product master
// ซึ่งคือค่าที่คอลัมน์ "Weight-spec" ต้องแสดง
//
// ค่านี้อยู่ระดับ subgroup ไม่ใช่ใน InventoryWeight เพราะต้องมีค่าแม้สินค้าไม่มีสต็อก
// คืน 0 เมื่อหาไม่เจอ เพื่อให้ผู้ใช้เห็นว่า master data ยังไม่ครบ
// (ฝั่งสูตรคำนวณ fallback เป็น 1.0 เองผ่าน weightSpecForFormula)
func getWeightSpecFromInventory(sg models.PriceListSubGroupResponse) float64 {
	return sg.WeightSpec
}
```

- [ ] **Step 4: รัน test ให้ผ่าน**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
go test ./internal/services/price-service/patterns/ -v
```

Expected: PASS ทุกเทสใน package `patterns` (รวมเทสเดิมที่มีอยู่ เช่น `TestGetAvgProductFromInventory`, `editable_suffixes_test.go`, `shared_test.go`)

หากมีเทสเดิมตัวอื่นที่ตั้งค่า `InventoryWeight[0].WeightSpec` แล้วคาดหวังผลลัพธ์จาก `getWeightSpecFromInventory` ให้แก้ให้ set `sg.WeightSpec` แทน

- [ ] **Step 5: Commit**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
git add internal/services/price-service/patterns/shared.go internal/services/price-service/patterns/inventory_udf_test.go
git commit -m "fix(price-service): กริดอ่าน Weight-spec จาก subgroup แทน InventoryWeight

เดิมอ่าน InventoryWeight[0].WeightSpec ซึ่ง warehouse-core ประกาศ field ไว้
แต่ไม่เคย assign ค่าเลย จึงได้ 0 เสมอ และค่าจะหายไปเมื่อสินค้าไม่มีสต็อก

แก้ที่ฟังก์ชันร่วมจุดเดียว ครอบคลุมจุดเรียก 8 จุดใน shared.go และ
pattern config ทั้ง 22 ไฟล์ที่มีคอลัมน์ Weight-spec

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 7: erp-core — พา weight_spec เข้า GetPriceDetail (กริดหลัก)

**Files:**
- Modify: `prime-wms-erp-core/internal/services/price-service/get-price-detail.go:279-330`

**Context:** flow นี้ **expand subgroup 1 แถวต่อ 1 inventory batch** และมี branch `else` ที่เก็บ subgroup เดิมไว้เมื่อไม่มีสต็อก — **ต้อง set `WeightSpec` ทั้ง 2 branch** ไม่งั้นเคส "ไม่มีสต็อก" (Goal ข้อ 2) จะยังพัง
`inventoryMap` ถูกสร้างจาก `invItem.InventoryWeight` ซึ่งไม่มี weight_spec จึงต้องสร้าง map แยก

- [ ] **Step 1: สร้าง map weight_spec คู่กับ inventoryMap**

แก้ `internal/services/price-service/get-price-detail.go` — หาบล็อกนี้ (~บรรทัด 283-288):

```go
		} else {
			// Create a map of inventory data by ID for quick lookup
			inventoryMap := make(map[string][]models.InventoryWeightResponse)
			for _, invItem := range inventoryResponse {
				inventoryMap[invItem.ID] = invItem.InventoryWeight
			}
```

เป็น:

```go
		} else {
			// Create a map of inventory data by ID for quick lookup
			inventoryMap := make(map[string][]models.InventoryWeightResponse)
			// weight_spec มาระดับ result ไม่ได้อยู่ใน InventoryWeight จึงต้องเก็บ map แยก
			// และต้องใช้ได้แม้ subgroup นั้นไม่มีสต็อก
			weightSpecMap := make(map[string]float64)
			for _, invItem := range inventoryResponse {
				inventoryMap[invItem.ID] = invItem.InventoryWeight
				weightSpecMap[invItem.ID] = invItem.WeightSpec
			}
```

- [ ] **Step 2: set ค่าใน branch ที่มีสต็อก**

ในไฟล์เดียวกัน หาบล็อกที่ expand subgroup (~บรรทัด 299-310) แก้จาก:

```go
						for _, inv := range inventoryWeights {
							expandedSG := sg // Copy the subgroup

							// Set inventory-specific data
							expandedSG.InventoryWeight = []models.InventoryWeightResponse{inv}
							expandedSG.ProductCode = inv.ProductCode
							expandedSG.SupplierCode = inv.SupplierCode
							expandedSG.SupplierName = inv.SupplierName
							expandedSG.BatchNo = inv.BatchNo
```

เป็น:

```go
						for _, inv := range inventoryWeights {
							expandedSG := sg // Copy the subgroup

							// Set inventory-specific data
							expandedSG.InventoryWeight = []models.InventoryWeightResponse{inv}
							expandedSG.ProductCode = inv.ProductCode
							expandedSG.SupplierCode = inv.SupplierCode
							expandedSG.SupplierName = inv.SupplierName
							expandedSG.BatchNo = inv.BatchNo
							expandedSG.WeightSpec = weightSpecMap[sg.ID]
```

- [ ] **Step 3: set ค่าใน branch ที่ไม่มีสต็อก**

ในไฟล์เดียวกัน หาบล็อก `else` ถัดลงไป (~บรรทัด 323-326) แก้จาก:

```go
					} else {
						// No inventory data, keep original subgroup
						expandedSubGroups = append(expandedSubGroups, sg)
					}
```

เป็น:

```go
					} else {
						// No inventory data, keep original subgroup.
						// weight_spec ยังต้องมีค่าเพราะมาจาก product master ไม่ได้มาจากสต็อก
						sg.WeightSpec = weightSpecMap[sg.ID]
						expandedSubGroups = append(expandedSubGroups, sg)
					}
```

- [ ] **Step 4: build + รัน test ทั้ง package**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
go build ./... && gofmt -l internal/services/price-service/ && go test ./internal/services/price-service/... 2>&1 | tail -20
```

Expected: build ผ่าน, `gofmt -l` ไม่พิมพ์ชื่อไฟล์, test ทั้งหมด `ok`

- [ ] **Step 5: Commit**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
git add internal/services/price-service/get-price-detail.go
git commit -m "fix(price-service): พา weight_spec เข้า GetPriceDetail ทั้งเคสมีและไม่มีสต็อก

flow นี้ expand subgroup 1 แถวต่อ 1 batch และมี branch ที่เก็บ subgroup เดิม
ไว้เมื่อไม่มีสต็อก จึงต้อง set WeightSpec ทั้งสอง branch

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 8: erp-core — แก้ export table (มีทั้ง map และจุดสร้าง row)

**Files:**
- Modify: `prime-wms-erp-core/internal/services/price-service/get-price-export-table.go:155-176` และ `:445-460`
- Test: `prime-wms-erp-core/internal/services/price-service/get-price-export-table_test.go`

**Context:** export table **ไม่ได้เรียก `getWeightSpecFromInventory`** — มันสร้าง row เองที่บรรทัด 448 ด้วย `row["total_weight"] = inv.TotalWeight` ซึ่งเป็นแหล่งผิดที่สาม ต้องแก้แยก
struct ที่ใช้คือ `priceService.SubGroup` (ไม่ใช่ `models.PriceListSubGroupResponse`) และ key ของ map เป็น `sg.ID.String()`
บล็อก `if len(sg.InventoryWeight) > 0` ที่บรรทัด 446 จะไม่ทำงานเมื่อไม่มีสต็อก — `row["total_weight"]` จึงต้องถูก set นอกบล็อกนั้น

- [ ] **Step 1: เขียน test ที่ต้อง fail ก่อน**

เพิ่มฟังก์ชันนี้ต่อท้าย `internal/services/price-service/get-price-export-table_test.go`:

```go
// คอลัมน์ Weight-spec ใน export ผูกกับ field "total_weight" (ดู ExportColumn
// ที่ get-price-export-table.go:303) ค่าต้องมาจาก weight_spec ของ base unit
// ใน product master ไม่ใช่ TotalWeight ซึ่งเป็นน้ำหนักรวมของสต็อก
func TestExportRowWeightSpecUsesProductMaster(t *testing.T) {
	sg := SubGroup{
		WeightSpec: 12.5,
		InventoryWeight: []models.InventoryWeightResponse{
			{TotalWeight: 1500000, AvgWeight: 30},
		},
	}

	row := map[string]interface{}{}
	applyInventoryFieldsToRow(row, sg)

	if got := row["total_weight"]; got != 12.5 {
		t.Fatalf("total_weight (คอลัมน์ Weight-spec) = %v, want 12.5", got)
	}
}

// สินค้าที่ไม่มีสต็อกต้องยังแสดง Weight-spec ได้
func TestExportRowWeightSpecWithoutStock(t *testing.T) {
	sg := SubGroup{
		WeightSpec:      12.5,
		InventoryWeight: []models.InventoryWeightResponse{},
	}

	row := map[string]interface{}{}
	applyInventoryFieldsToRow(row, sg)

	if got := row["total_weight"]; got != 12.5 {
		t.Fatalf("total_weight ตอนไม่มีสต็อก = %v, want 12.5", got)
	}
}
```

หากไฟล์เทสยังไม่ได้ import `models` ให้เพิ่ม `"prime-erp-core/internal/models"` ในบล็อก import

- [ ] **Step 2: รัน test ให้เห็นว่า fail**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
go test ./internal/services/price-service/ -run TestExportRowWeightSpec -v
```

Expected: FAIL — `undefined: applyInventoryFieldsToRow`

- [ ] **Step 3: แยกการเติม inventory fields ออกมาเป็นฟังก์ชัน**

แก้ `internal/services/price-service/get-price-export-table.go` — แทนที่บล็อก (~บรรทัด 445-460):

```go
			// Add inventory weight fields
			if len(sg.InventoryWeight) > 0 {
				inv := sg.InventoryWeight[0]
				row["total_weight"] = inv.TotalWeight
				row["avg_weight"] = inv.AvgWeight
				row["market_weight"] = inv.WeightSpec
				row["stock"] = inv.SumQty
				row["stock_quantity"] = inv.TotalQty
				row["quantity"] = inv.SumQty
				row["batch_no"] = inv.BatchNo
				row["brand"] = inv.SupplierName
				row["code"] = inv.ProductCode
				row["warehouse"] = inv.SiteCode
				row["supplier_name"] = inv.SupplierName
			}
```

ด้วย:

```go
			// Add inventory weight fields
			applyInventoryFieldsToRow(row, sg)
```

แล้วเพิ่มฟังก์ชันนี้ที่ท้ายไฟล์ `get-price-export-table.go`:

```go
// applyInventoryFieldsToRow เติมค่าที่มาจาก inventory และ product master ลงใน row ของ export
//
// row["total_weight"] คือคอลัมน์ที่ผู้ใช้เห็นชื่อ "Weight-spec" ค่าที่ถูกต้องคือน้ำหนัก
// ของ base unit จาก product master (sg.WeightSpec) ไม่ใช่ inv.TotalWeight ซึ่งเป็น
// น้ำหนักรวมของสต็อก และต้องเติมนอกเงื่อนไข len(InventoryWeight) > 0 เพราะสินค้าที่
// ไม่มีสต็อกก็ต้องแสดง Weight-spec ได้
func applyInventoryFieldsToRow(row map[string]interface{}, sg SubGroup) {
	row["total_weight"] = sg.WeightSpec

	if len(sg.InventoryWeight) == 0 {
		return
	}

	inv := sg.InventoryWeight[0]
	row["avg_weight"] = inv.AvgWeight
	row["market_weight"] = inv.WeightSpec
	row["stock"] = inv.SumQty
	row["stock_quantity"] = inv.TotalQty
	row["quantity"] = inv.SumQty
	row["batch_no"] = inv.BatchNo
	row["brand"] = inv.SupplierName
	row["code"] = inv.ProductCode
	row["warehouse"] = inv.SiteCode
	row["supplier_name"] = inv.SupplierName
}
```

**หมายเหตุ:** `row["market_weight"]` คงพฤติกรรมเดิมไว้ (ยังอ่าน `inv.WeightSpec` ซึ่งเป็น 0 มาตลอด) — เป็นคอลัมน์คนละตัวและอยู่นอก scope งานนี้ ห้ามเปลี่ยน

- [ ] **Step 4: รัน test ให้ผ่าน**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
go test ./internal/services/price-service/ -run TestExportRowWeightSpec -v
```

Expected: PASS ทั้ง 2 เทส

- [ ] **Step 5: พา weight_spec เข้า subgroup ของ export flow**

แก้ `internal/services/price-service/get-price-export-table.go` บล็อก ~บรรทัด 155-176 จาก:

```go
		} else {
			// Create a map of inventory data by ID for quick lookup
			inventoryMap := make(map[string][]models.InventoryWeightResponse)
			for _, invItem := range inventoryResponse {
				inventoryMap[invItem.ID] = invItem.InventoryWeight
			}

			// Enrich subgroups with inventory data
			for i := range res {
				for j := range res[i].SubGroups {
					sg := &res[i].SubGroups[j]
					if inventoryWeights, ok := inventoryMap[sg.ID.String()]; ok && len(inventoryWeights) > 0 {
						// For export, use first inventory record per subgroup
						inv := inventoryWeights[0]
						sg.InventoryWeight = []models.InventoryWeightResponse{inv}
						sg.ProductCode = inv.ProductCode
						sg.SupplierCode = inv.SupplierCode
						sg.SupplierName = inv.SupplierName
						sg.BatchNo = inv.BatchNo
					}
				}
			}
```

เป็น:

```go
		} else {
			// Create a map of inventory data by ID for quick lookup
			inventoryMap := make(map[string][]models.InventoryWeightResponse)
			// weight_spec มาระดับ result และต้องใช้ได้แม้ subgroup ไม่มีสต็อก
			weightSpecMap := make(map[string]float64)
			for _, invItem := range inventoryResponse {
				inventoryMap[invItem.ID] = invItem.InventoryWeight
				weightSpecMap[invItem.ID] = invItem.WeightSpec
			}

			// Enrich subgroups with inventory data
			for i := range res {
				for j := range res[i].SubGroups {
					sg := &res[i].SubGroups[j]
					sg.WeightSpec = weightSpecMap[sg.ID.String()]
					if inventoryWeights, ok := inventoryMap[sg.ID.String()]; ok && len(inventoryWeights) > 0 {
						// For export, use first inventory record per subgroup
						inv := inventoryWeights[0]
						sg.InventoryWeight = []models.InventoryWeightResponse{inv}
						sg.ProductCode = inv.ProductCode
						sg.SupplierCode = inv.SupplierCode
						sg.SupplierName = inv.SupplierName
						sg.BatchNo = inv.BatchNo
					}
				}
			}
```

- [ ] **Step 6: รัน test ทั้ง package**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
go build ./... && go test ./internal/services/price-service/... 2>&1 | tail -20
```

Expected: `ok` ทุก package

หากเทสเดิม `get-price-export-table_test.go:496` ที่ตั้ง `WeightSpec: 2.5` ไว้ใน `InventoryWeightResponse` fail ให้ย้ายไปตั้งที่ `SubGroup.WeightSpec` แทน แล้วปรับค่าที่คาดหวังให้ตรง

- [ ] **Step 7: Commit**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
git add internal/services/price-service/get-price-export-table.go internal/services/price-service/get-price-export-table_test.go
git commit -m "fix(price-service): export table ใช้ weight_spec จาก product master

export สร้าง row เองไม่ผ่าน getWeightSpecFromInventory จึงเป็นแหล่งผิดอีกจุด
row[total_weight] (คอลัมน์ Weight-spec) เดิมได้ inv.TotalWeight ซึ่งเป็น
น้ำหนักรวมของสต็อก และย้ายมาเติมนอกเงื่อนไข len(InventoryWeight) > 0
เพื่อให้สินค้าที่ไม่มีสต็อกยังแสดงค่าได้

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 9: erp-core — สูตรคำนวณใน GetCalculated ใช้ค่าใหม่

**Files:**
- Modify: `prime-wms-erp-core/internal/services/price-service/get-calculated-pricelist-subgroup.go:157-232` และจุดสร้าง response

**Context:** ตัวแปร `weightSpec` เดิมรับค่าจาก `inventoryWeight[0].TotalWeight` ซึ่งเป็นตัวเดียวกับ `kg` เป๊ะ ๆ (บรรทัดถัดไป)
**`kg` และ `avgKgStock` ถูกต้องอยู่แล้ว ห้ามแตะ** — แก้เฉพาะ `weightSpec`
`CalculatePrice` รับตัวแปร `weight_spec` ผ่าน `PriceData.WeightSpec` เข้า expression engine

- [ ] **Step 1: สร้าง map weight_spec**

แก้ `internal/services/price-service/get-calculated-pricelist-subgroup.go` — หาบล็อก (~บรรทัด 156-158):

```go
	// Create inventory maps for quick lookup
	inventoryMap := make(map[string][]models.InventoryWeightResponse)
	supplierCodeMap := make(map[string]string)
```

เป็น:

```go
	// Create inventory maps for quick lookup
	inventoryMap := make(map[string][]models.InventoryWeightResponse)
	supplierCodeMap := make(map[string]string)
	// weight_spec มาจาก product master ระดับ result ใช้ได้แม้ subgroup ไม่มีสต็อก
	weightSpecMap := make(map[string]float64)
```

แล้วหาบล็อก (~บรรทัด 184-188):

```go
			// Build maps of inventory data by ID for quick lookup
			for _, invItem := range inventoryResponse {
				inventoryMap[invItem.ID] = invItem.InventoryWeight
				supplierCodeMap[invItem.ID] = invItem.SupplierCode
			}
```

เป็น:

```go
			// Build maps of inventory data by ID for quick lookup
			for _, invItem := range inventoryResponse {
				inventoryMap[invItem.ID] = invItem.InventoryWeight
				supplierCodeMap[invItem.ID] = invItem.SupplierCode
				weightSpecMap[invItem.ID] = invItem.WeightSpec
			}
```

- [ ] **Step 2: เปลี่ยนแหล่งของ weightSpec**

ในไฟล์เดียวกัน หาบล็อก (~บรรทัด 213-231) แก้จาก:

```go
		// Get inventory data for this subgroup
		avgKgStock := 1.0
		weightSpec := 1.0
		pcs := 0.0
		kg := 0.0
		if inventoryWeight, ok := inventoryMap[subGroupID.String()]; ok && len(inventoryWeight) > 0 {
			// Use AvgProduct from first inventory weight response
			if inventoryWeight[0].AvgWeight == 0 {
				avgKgStock = 1.0
			} else {
				avgKgStock = inventoryWeight[0].AvgWeight
			}
			if inventoryWeight[0].TotalWeight == 0 {
				weightSpec = 1.0
			} else {
				weightSpec = inventoryWeight[0].TotalWeight
			}
			pcs = inventoryWeight[0].TotalQty
			kg = inventoryWeight[0].TotalWeight
		}
```

เป็น:

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

		// weight_spec คือน้ำหนักของ base unit จาก product master ไม่ได้ผูกกับสต็อก
		// จึงอ่านนอกบล็อก inventory ข้างบน (เดิมอ่าน TotalWeight ซึ่งเป็นตัวเดียวกับ kg)
		// rawWeightSpec เก็บค่าดิบไว้ส่งกลับใน response ส่วนสูตรใช้ค่าที่ fallback แล้ว
		rawWeightSpec := weightSpecMap[subGroupID.String()]
		weightSpec := weightSpecForFormula(rawWeightSpec)
```

- [ ] **Step 3: ใส่ค่าดิบลง response**

ในไฟล์เดียวกัน หาจุดที่สร้าง `models.GetCalculatedPriceListSubGroupItem{` (อยู่ท้ายลูปของแต่ละ subgroup)
เพิ่ม field `WeightSpec: rawWeightSpec,` เข้าไปใน struct literal นั้น โดยไม่ลบ field เดิมใด ๆ เช่น:

แก้บล็อกที่บรรทัด 309-318 จาก:

```go
		responseData = append(responseData, models.GetCalculatedPriceListSubGroupItem{
			SubGroupID:                subGroupID.String(),
			TotalNetPriceUnit:         totalNetPriceUnit,
			TotalNetPriceWeight:       totalNetPriceWeight,
			ExtraPriceUnit:            extraPriceUnit,
			ExtraPriceWeight:          extraPriceWeight,
			BeforeTotalNetPriceUnit:   beforeTotalNetPriceUnit,
			BeforeTotalNetPriceWeight: beforeTotalNetPriceWeight,
			DefaultUom:                defaultUom,
		})
```

เป็น:

```go
		responseData = append(responseData, models.GetCalculatedPriceListSubGroupItem{
			SubGroupID:                subGroupID.String(),
			TotalNetPriceUnit:         totalNetPriceUnit,
			TotalNetPriceWeight:       totalNetPriceWeight,
			ExtraPriceUnit:            extraPriceUnit,
			ExtraPriceWeight:          extraPriceWeight,
			BeforeTotalNetPriceUnit:   beforeTotalNetPriceUnit,
			BeforeTotalNetPriceWeight: beforeTotalNetPriceWeight,
			WeightSpec:                rawWeightSpec,
			DefaultUom:                defaultUom,
		})
```

- [ ] **Step 4: build และรัน test**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
go build ./... && go vet ./internal/services/price-service/... && go test ./internal/services/price-service/... 2>&1 | tail -20
```

Expected: build ผ่าน, vet สะอาด, test `ok`

- [ ] **Step 5: Commit**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
git add internal/services/price-service/get-calculated-pricelist-subgroup.go
git commit -m "fix(price-service): GetCalculated ใช้ weight_spec จาก product master

เดิม weightSpec = inventoryWeight[0].TotalWeight ซึ่งเป็นน้ำหนักรวมของสต็อก
ตัวเดียวกับตัวแปร kg เป๊ะ ๆ สูตรจึงคำนวณด้วยค่าผิดชนิด

ย้ายมาอ่านนอกบล็อก inventory เพราะ weight_spec ไม่ได้ผูกกับสต็อก และเพิ่ม
weight_spec (ค่าดิบ) ลง response ที่เดิมไม่มี field นี้

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 10: erp-core — สูตรคำนวณใน UpdateLatest ใช้ค่าใหม่

**Files:**
- Modify: `prime-wms-erp-core/internal/services/price-service/update-latest-pricelist-subgroup.go:175`, `:204`, `:228-247`

**Context:** flow นี้โครงสร้างเหมือน Task 9 แต่เป็นคนละไฟล์และ **save ลง DB จริง**

**อย่าไปแตะบรรทัด 430** — `"weight_spec": priceData.WeightSpec` ตรงนั้นคือ env ของ expression
engine ข้างใน `CalculatePrice` (ตัวแปรที่สูตรเรียกใช้) ไม่ใช่ response — มันต้องได้ค่าที่ fallback
เป็น 1.0 แล้วตามเจตนา ถูกอยู่แล้ว

`RunUpdateLatestPriceListSubGroup` return แค่ `map[string]interface{}{"success": ..., "message": ...}`
(บรรทัด 329-332) ไม่มีข้อมูลราย subgroup จึงไม่มี response field ให้เพิ่มในไฟล์นี้

- [ ] **Step 1: สร้าง map weight_spec**

แก้ `internal/services/price-service/update-latest-pricelist-subgroup.go` — หาบรรทัด 175:

```go
	inventoryMap := make(map[string][]models.InventoryWeightResponse)
```

เพิ่มบรรทัดถัดจากมัน:

```go
	// weight_spec มาจาก product master ระดับ result ใช้ได้แม้ subgroup ไม่มีสต็อก
	weightSpecMap := make(map[string]float64)
```

แล้วหาบรรทัด 204:

```go
				inventoryMap[invItem.ID] = invItem.InventoryWeight
```

เพิ่มบรรทัดถัดจากมัน:

```go
				weightSpecMap[invItem.ID] = invItem.WeightSpec
```

- [ ] **Step 2: เปลี่ยนแหล่งของ weightSpec**

ในไฟล์เดียวกัน แก้บล็อกที่บรรทัด 228-247 จาก:

```go
		// Get inventory data for this subgroup
		avgKgStock := 1.0
		weightSpec := 1.0
		pcs := 0.0
		kg := 0.0
		if inventoryWeight, ok := inventoryMap[subGroupID.String()]; ok && len(inventoryWeight) > 0 {
			// Use AvgProduct from first inventory weight response
			if inventoryWeight[0].AvgWeight == 0 {
				avgKgStock = 1.0
			} else {
				avgKgStock = inventoryWeight[0].AvgWeight
			}
			if inventoryWeight[0].TotalWeight == 0 {
				weightSpec = 1.0
			} else {
				weightSpec = inventoryWeight[0].TotalWeight
			}
			pcs = inventoryWeight[0].TotalQty
			kg = inventoryWeight[0].TotalWeight
		}
```

เป็น:

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

		// weight_spec คือน้ำหนักของ base unit จาก product master ไม่ได้ผูกกับสต็อก
		// จึงอ่านนอกบล็อก inventory (เดิมอ่าน TotalWeight ซึ่งเป็นตัวเดียวกับ kg)
		weightSpec := weightSpecForFormula(weightSpecMap[subGroupID.String()])
```

- [ ] **Step 3: build และรัน test**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
go build ./... && go vet ./internal/services/price-service/... && go test ./internal/services/price-service/... 2>&1 | tail -20
```

Expected: build ผ่าน, vet สะอาด, test `ok` (รวม `update-latest-pricelist-subgroup_service_test.go` ที่มีอยู่เดิม)

- [ ] **Step 4: Commit**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
git add internal/services/price-service/update-latest-pricelist-subgroup.go
git commit -m "fix(price-service): UpdateLatest ใช้ weight_spec จาก product master

แหล่งเดิมเป็น TotalWeight เหมือน GetCalculated ซึ่งเป็นตัวเดียวกับตัวแปร kg
flow นี้ save ลง DB จริงจึงเป็นจุดที่ทำให้ราคาที่บันทึกไว้ผิดไปด้วย

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 11: integration test — flow เต็มพร้อม warehouse ปลอม

**Files:**
- Create: `prime-wms-erp-core/internal/services/price-service/weight_spec_integration_test.go`

**Context:** repo มีไฟล์ integration test อยู่แล้วเป็นตัวอย่างรูปแบบ: `get-price-last-updated_integration_test.go` และ `upload-pricelist_integration_test.go` — **อ่าน 2 ไฟล์นั้นก่อนเขียน** เพื่อ copy build tag, การ setup testcontainers และ helper ที่มีอยู่ อย่าประดิษฐ์รูปแบบใหม่
`erp-core` เรียก warehouse ผ่าน `config.GET_INVENTORY_BY_KEY_ENDPOINT` ซึ่งอ่านจาก env — จึงชี้ไปที่ `httptest.Server` ได้

- [ ] **Step 1: อ่านรูปแบบ integration test ที่มีอยู่**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
head -40 internal/services/price-service/get-price-last-updated_integration_test.go
grep -n "GET_INVENTORY_BY_KEY_ENDPOINT" config/*.go
```

จดไว้ว่า: build tag คืออะไร, ชื่อ env var ของ endpoint คืออะไร, มี helper setup อะไรให้ใช้ซ้ำได้บ้าง

- [ ] **Step 2: เขียน integration test**

สร้าง `internal/services/price-service/weight_spec_integration_test.go` โดยใช้ build tag เดียวกับไฟล์ integration test ที่มีอยู่ เนื้อหาต้องครอบคลุม 3 เคส:

1. **มีสต็อก** — warehouse ปลอมตอบ `{"weight_spec": 12.5, "inventory_weight": [{"total_weight": 1500000, "total_qty": 100}]}` → `GetPriceDetail` ต้องได้ subgroup ที่ `WeightSpec == 12.5` (ไม่ใช่ 1500000)
2. **ไม่มีสต็อก** — warehouse ปลอมตอบ `{"weight_spec": 12.5, "inventory_weight": []}` → subgroup ต้องยังได้ `WeightSpec == 12.5`
3. **ไม่ match product** — warehouse ปลอมตอบ `{"product_code": "", "weight_spec": 0, "inventory_weight": []}` → subgroup ได้ `WeightSpec == 0`

โครงของ warehouse ปลอม:

```go
func fakeWarehouseServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("write fake warehouse response: %v", err)
		}
	}))
}
```

แล้วชี้ env ของ endpoint ไปที่ `srv.URL` ด้วย `t.Setenv(...)` ก่อนเรียก service (ใช้ชื่อ env ที่จดไว้จาก Step 1) และเรียก `config.Initialize()` ใหม่ถ้าไฟล์ integration test ที่มีอยู่ทำแบบนั้น

- [ ] **Step 3: รัน integration test**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
make test-integration 2>&1 | tail -30
```

Expected: PASS ทั้ง 3 เคส

หาก `make test-integration` รันเฉพาะ priceList repo tests ให้เพิ่ม target หรือรันตรง ๆ ด้วย build tag ที่จดไว้จาก Step 1

- [ ] **Step 4: Commit**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
git add internal/services/price-service/weight_spec_integration_test.go
git commit -m "test(price-service): integration test weight_spec ครบ 3 เคส

มีสต็อก / ไม่มีสต็อก / ไม่ match product โดยใช้ httptest แทน warehouse
เคสไม่มีสต็อกคือเคสหลักที่พฤติกรรมเดิมพัง

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 12: ตรวจ coverage ให้ถึง 80%

**Context:** CLAUDE.md บังคับ coverage ไม่ต่ำกว่า 80% วัดเฉพาะโค้ดที่งานนี้แตะ ไม่ใช่ทั้ง repo

- [ ] **Step 1: วัด coverage ของ warehouse-core**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-warehouse-core
go test ./internal/services/inventory-service/ -coverprofile=/tmp/wh-cover.out
go tool cover -func=/tmp/wh-cover.out | grep -E "WeightSpecFromUnits|total:"
```

Expected: `WeightSpecFromUnits` = `100.0%`

- [ ] **Step 2: วัด coverage ของ erp-core**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
go test ./internal/services/price-service/... -coverprofile=/tmp/erp-cover.out
go tool cover -func=/tmp/erp-cover.out | grep -E "weightSpecForFormula|getWeightSpecFromInventory|applyInventoryFieldsToRow"
```

Expected: ทั้ง 3 ฟังก์ชัน `100.0%`

- [ ] **Step 3: เติมเทสถ้ายังไม่ถึง**

ถ้าฟังก์ชันไหนต่ำกว่า 80% ให้เพิ่ม subtest ในไฟล์เทสของฟังก์ชันนั้น (`weight_spec_test.go` ของแต่ละ repo หรือ `inventory_udf_test.go`) จนครบทุก branch แล้ววัดซ้ำ

- [ ] **Step 4: รัน suite เต็มทั้ง 2 repo ครั้งสุดท้าย**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-warehouse-core && go build ./... && go vet ./... && go test ./... 2>&1 | grep -v "no test files" | tail -20
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core && go build ./... && go vet ./... && go test ./... 2>&1 | grep -v "no test files" | tail -20
```

Expected: ไม่มี `FAIL` ในผลลัพธ์ของทั้งสอง repo

- [ ] **Step 5: ตรวจว่าไม่มี placeholder หลงเหลือ**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms
git -C prime-wms-erp-core diff Develop --stat
grep -rn "TODO\|FIXME\|t.Skip\|\.only(" prime-wms-erp-core/internal/services/price-service/ prime-wms-warehouse-core/internal/services/inventory-service/ | grep -v "_test.go:.*// " | head
```

Expected: ไม่มี TODO/FIXME/t.Skip ที่เกิดจากงานนี้ (ของเดิมที่มีอยู่ก่อนไม่นับ — ตรวจด้วย `git diff` ว่าไม่ได้เป็นบรรทัดที่เราเพิ่ม)

- [ ] **Step 6: Commit เทสที่เพิ่ม (ถ้ามี)**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
git add -A internal/services/price-service/
git commit -m "test(price-service): เติม test ให้ coverage ถึงเกณฑ์ 80%

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01DSaH9siDvMo3PucXvY5SfC"
```

---

## Task 13: verify บนของจริง

**Context:** ทั้ง 2 service ต้องรันพร้อมกัน erp-core (9115) เรียก warehouse-core (9103) และ warehouse-core เรียก product-core (9102) — ต้องรัน product-core ด้วย

- [ ] **Step 1: รัน 3 service**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-product-core && go run ./cmd &
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-warehouse-core && go run ./cmd &
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core && go run ./cmd &
```

รอจนทั้ง 3 พิมพ์ว่า listen แล้ว

- [ ] **Step 2: หาค่าที่ถูกต้องจาก product master มาเทียบ**

เลือก product code หนึ่งตัวที่อยู่ใน price list แล้ว query หาน้ำหนัก base unit:

```sql
SELECT p.product_code, u.unit_code, pu.weight, pu.flag_base
FROM product p
JOIN product_unit pu ON pu.product_id = p.id
JOIN unit u ON u.id = pu.unit_id
WHERE p.product_code = '<PRODUCT_CODE>' AND pu.flag_base = true;
```

จดค่า `weight` ไว้

- [ ] **Step 3: ยิง GetPriceDetail แล้วเทียบ**

```bash
curl -s -X POST http://localhost:9115/price/GetPriceDetail \
  -H "Content-Type: application/json" \
  -d '{"company_code":"<COMPANY>","site_codes":["<SITE>"],"group_codes":["GROUP_1_ITEM_2"]}' \
  | python3 -m json.tool | grep -i "weight_spec" | head
```

Expected: ค่า `weight_spec` ตรงกับ `pu.weight` ที่จดไว้จาก Step 2 และ**ไม่ใช่** 0 หรือเลขหลักแสน/หลักล้าน

- [ ] **Step 4: ตรวจเคสไม่มีสต็อก**

หา subgroup ที่ไม่มีสต็อกในคลัง (`inventory_weight` เป็น `[]` ใน response) แล้วยืนยันว่า `weight_spec` ของมันยังมีค่า

Expected: `"inventory_weight": []` แต่ `"weight_spec"` ไม่เป็น 0 (ถ้า product นั้นมี base unit ใน master)

- [ ] **Step 5: ตรวจหน้าเว็บ**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-web && yarn dev
```

เปิดหน้า price list detail ของ GROUP_1_ITEM_2 → คอลัมน์ `Weight-spec` ต้องแสดงค่าเดียวกับ Step 2
**ไม่ต้องแก้โค้ด web ใด ๆ** ถ้าค่าไม่ขึ้น แปลว่า backend ยังส่งไม่ถูก ให้กลับไปดู Task 6-8 ไม่ใช่มาแก้ที่ web

- [ ] **Step 6: ปิด service ที่รันค้างไว้**

```bash
pkill -f "go run ./cmd"
```

- [ ] **Step 7: ตรวจ branch ก่อนส่งงาน**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core && git status --short --branch && git log --oneline Develop..HEAD
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-warehouse-core && git status --short --branch && git log --oneline Develop..HEAD
```

Expected: ทั้ง 2 repo อยู่บน branch `feat/pricelist-weight-spec-from-product-master` (ไม่ใช่ `Develop`) และเห็นรายการ commit ของงานนี้ครบ

---

## หมายเหตุก่อน push

- **warehouse-core remote เป็น HTTPS และ auth fail** (`Password authentication is not supported for Git operations`) branch แตกจาก local `Develop` ที่ commit `c4ee578` (2026-07-14) ซึ่งค่อนข้างเก่า — ต้องแก้ remote เป็น SSH หรือตั้ง credential ก่อน แล้ว `git fetch` เทียบว่า local Develop ตรงกับ remote ก่อน push ถ้าไม่ตรงต้อง rebase ก่อน
- **ห้าม merge เข้า Develop ตรง ๆ** — เปิด PR เท่านั้น
- ทั้ง 2 repo ต้อง deploy พร้อมกัน: erp-core ที่อ่าน `weight_spec` จะได้ 0 ถ้า warehouse-core ยังเป็นเวอร์ชันเก่า (ไม่ crash แต่ค่าจะยังผิด)
- หลังงานเสร็จ รัน `graphify update .` ที่ root project ตาม CLAUDE.md
