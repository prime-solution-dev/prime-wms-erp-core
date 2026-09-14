# ปิด SO อัตโนมัติเมื่อส่งของครบ — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** เมื่อใบจองคิวส่ง (DBS) ถูกปิด COMPLETED ให้ erp-core ตรวจยอด GI รวมทุกใบของ SO นั้น แล้วปิด `sale_item` / `sale` ที่ส่งครบตามเกณฑ์ tolerance โดยอัตโนมัติ

**Architecture:** เพิ่มฟังก์ชันใน `erp-core/internal/services/delivery-service/` ที่ `UpdateStatusDelivery` เรียก **หลัง commit** (error log ไม่ return) อ่านยอด GI ผ่าน `GetOrdersDelivery` ที่มีอยู่แล้ว รวมยอดต่อ `sale_item` ผ่าน `delivery_booking_item.document_ref_item` แล้วปิดใบด้วย helper ที่แยกออกมาจาก `UpdateSaleItemStatus` พร้อมลบเส้นเดิมฝั่ง `wms-outbound-service` ที่ไม่เคยถูกเรียก

**Tech Stack:** Go + Gin + GORM v2, Postgres (`prime_erp`), เทสเป็น `go test` มาตรฐาน (pure function ไม่ต่อ DB)

**Spec:** `docs/superpowers/specs/2026-09-09-close-so-on-delivery-complete.md`

**Branch (แตกไว้แล้วทั้ง 2 repo):** `feat/close-so-on-delivery-complete`
- `C:\work-prime\erp-core` (แตกจาก `origin/Develop` = 4373bf8)
- `C:\work-prime\wms-outbound-service` (แตกจาก `origin/Develop` = 61079c5)

## Global Constraints

- commit message เป็นภาษาอังกฤษเสมอ คอมเมนต์ในโค้ดเขียนไทยได้ (ตามสไตล์ไฟล์ข้างเคียง เช่น `validate-booking-qty.go`)
- ห้าม `git add -A` — เพิ่มไฟล์ทีละชื่อ
- ห้ามยิง API หรือเขียน DB ของ UAT (18.138.69.85) ระหว่างทำงาน — ทดสอบด้วย unit test เท่านั้น
- ห้ามเพิ่ม cron job
- ห้ามเปลี่ยน signature ของ `foldWmsProgress` (มีเทสเดิมพึ่งอยู่) — ให้ delegate ไปตัวใหม่แทน
- ทุก task จบด้วย `go build ./...` และ `go test ./...` ผ่านใน repo ที่แก้
- สถานะที่ถือว่า "จบงานแล้ว" คือ `COMPLETED`, `CANCELED`, `CANCELLED` (ตรงกับ `closedOrderItemStatuses` เดิม)
- ยืนยันแล้วว่า `internal/services/sale-service/` ไม่ import `delivery-service` → เรียกข้าม package ทางเดียวได้ ไม่มี import cycle

---

## File Structure

**erp-core (สร้าง 2 ไฟล์ / แก้ 3 ไฟล์)**

| ไฟล์ | หน้าที่ |
|---|---|
| สร้าง `internal/services/delivery-service/close-sale-on-delivered.go` | กติกาโหมดหน่วย + เป้าหมาย + tolerance + รวมยอดต่อ sale_item + ตัวเชื่อม I/O `CloseSalesFullyDelivered` |
| สร้าง `internal/services/delivery-service/close-sale-on-delivered_test.go` | เทสของกติกาทั้งหมด (pure ไม่ต่อ DB) |
| แก้ `internal/services/delivery-service/validate-booking-qty.go` | เพิ่ม `foldWmsIssued` (คืน qty + weight + closed) แล้วให้ `foldWmsProgress` เรียกต่อ |
| แก้ `internal/services/delivery-service/update-status-delivery.go` | เรียก `CloseSalesFullyDelivered` หลัง commit เมื่อ status = COMPLETED |
| แก้ `internal/services/sale-service/update-sale-item-status.go` | แยก `MarkSaleItemsCompleted` ออกมาให้ใช้ร่วมกัน |

**wms-outbound-service (ลบ 2 ไฟล์ / แก้ 2 ไฟล์)**

| ไฟล์ | หน้าที่ |
|---|---|
| ลบ `internal/services/so-service/update-complete-so.go` | โค้ดตาย 492 บรรทัด |
| ลบ `external/services/sale/update-sale-item-status.go` | client ที่ไม่มีใครเรียกต่อ |
| แก้ `internal/routes/routes.go` | ตัด `soRoutes` block |
| แก้ `config/constants.go` | ตัด `UPDATE_SALE_ITEM_STATUS_ENDPOINT` |

---

## Task 1: กติกาโหมดหน่วย + เป้าหมาย + tolerance

**Files:**
- Create: `C:\work-prime\erp-core\internal\services\delivery-service\close-sale-on-delivered.go`
- Test: `C:\work-prime\erp-core\internal\services\delivery-service\close-sale-on-delivered_test.go`

**Interfaces:**
- Consumes: `models.SaleItem` (`internal/models/sale.go:64`) ฟิลด์ที่ใช้: `SaleItem`, `Qty`, `TotalWeight`, `SaleUnit`, `SaleUnitType`, `Status`
- Produces: `saleItemCompletionMode(saleUnit, saleUnitType string) string`, `completionTarget(item models.SaleItem) float64`, `isFullyDelivered(target, issued, tolerancePercent float64) bool`, ค่าคงที่ `completionModeQty` / `completionModeWeight`

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน**

สร้าง `close-sale-on-delivered_test.go`:

```go
package deliveryService

import (
	"testing"

	"prime-erp-core/internal/models"
)

func TestSaleItemCompletionModeFollowsSaleUnitAndType(t *testing.T) {
	cases := []struct {
		name     string
		saleUnit string
		unitType string
		wantMode string
	}{
		{"ขายเป็นชิ้น", "PC", "PC", completionModeQty},
		{"ขายเป็นกิโลจริง", "KG", "KG", completionModeWeight},
		{"KG_SPEC คุมด้วยชิ้น", "KG", "KG_SPEC", completionModeQty},
		{"KG_SPEC เขียนด้วยขีดกลาง", "KG", "kg-spec", completionModeQty},
		{"KG ที่ตั้งราคาแบบชิ้น ยังคิดเป็นน้ำหนัก", "KG", "PC", completionModeWeight},
		{"ค่าว่างถือเป็นชิ้น", "", "", completionModeQty},
		{"มีช่องว่างและตัวพิมพ์เล็ก", " kg ", " kg ", completionModeWeight},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := saleItemCompletionMode(tc.saleUnit, tc.unitType); got != tc.wantMode {
				t.Errorf("saleItemCompletionMode(%q, %q) = %q, want %q", tc.saleUnit, tc.unitType, got, tc.wantMode)
			}
		})
	}
}

// SO202609-0011 = 10 PCS / 51.50 kg ขายเป็นชิ้น -> เป้าคือ 10
// SO202609-0017 = 53 ชิ้น / 500 kg ขายเป็นกิโล -> เป้าคือ 500
func TestCompletionTargetPicksQtyOrWeight(t *testing.T) {
	pieces := models.SaleItem{Qty: 10, TotalWeight: 51.5, SaleUnit: "PC", SaleUnitType: "PC"}
	if got := completionTarget(pieces); got != 10 {
		t.Errorf("completionTarget(ชิ้น) = %v, want 10", got)
	}

	weighted := models.SaleItem{Qty: 53, TotalWeight: 500, SaleUnit: "KG", SaleUnitType: "KG"}
	if got := completionTarget(weighted); got != 500 {
		t.Errorf("completionTarget(กิโล) = %v, want 500", got)
	}

	kgSpec := models.SaleItem{Qty: 6, TotalWeight: 120, SaleUnit: "KG", SaleUnitType: "KG_SPEC"}
	if got := completionTarget(kgSpec); got != 6 {
		t.Errorf("completionTarget(KG_SPEC) = %v, want 6", got)
	}
}

func TestIsFullyDeliveredUsesLowerBoundOnly(t *testing.T) {
	const tolerance = 3.0 // ค่าจริงของ TMI ใน system_config

	cases := []struct {
		name   string
		target float64
		issued float64
		want   bool
	}{
		{"ส่งครบพอดี", 10, 10, true},
		{"ขาดแต่ยังอยู่ในระยะผ่อนผัน", 100, 97, true},
		{"ขาดเกินระยะผ่อนผัน", 100, 96.9, false},
		{"ส่งเกินถือว่าครบ", 10, 11, true},
		{"ยังไม่ได้ส่งเลย", 10, 0, false},
		{"เป้าเป็นศูนย์ไม่ปิด", 0, 0, false},
		{"เป้าติดลบไม่ปิด", -5, 10, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isFullyDelivered(tc.target, tc.issued, tolerance); got != tc.want {
				t.Errorf("isFullyDelivered(%v, %v, %v) = %v, want %v", tc.target, tc.issued, tolerance, got, tc.want)
			}
		})
	}
}

func TestIsFullyDeliveredTreatsNegativeToleranceAsZero(t *testing.T) {
	if isFullyDelivered(100, 97, -3) {
		t.Error("tolerance ติดลบต้องถือเป็น 0 -> ส่ง 97 จาก 100 ยังไม่ครบ")
	}
}
```

- [ ] **Step 2: รันเทสให้เห็นว่าพัง**

```bash
cd /c/work-prime/erp-core && go test ./internal/services/delivery-service/ -run 'TestSaleItemCompletionMode|TestCompletionTarget|TestIsFullyDelivered' -v
```

Expected: FAIL — `undefined: saleItemCompletionMode` (และตัวอื่น)

- [ ] **Step 3: เขียนโค้ดขั้นต่ำให้ผ่าน**

สร้าง `close-sale-on-delivered.go`:

```go
package deliveryService

import (
	"strings"

	"prime-erp-core/internal/models"
)

// โหมดตัดสินว่า "ส่งครบ" ของ 1 บรรทัดขาย ดูจากจำนวนชิ้นหรือจากน้ำหนัก
const (
	completionModeQty    = "QTY"
	completionModeWeight = "WEIGHT"
)

// saleItemCompletionMode บอกว่าบรรทัดนี้ต้องวัดความครบด้วยชิ้นหรือด้วยน้ำหนัก
//
// กติกาเดียวกับฝั่งจอ (wms-web packingCreate mapUnitCodeByOrder และ deliverySlotCreate:401)
//   - ขายเป็นกิโล (sale_unit = KG) -> วัดด้วยน้ำหนัก
//   - ยกเว้น KG_SPEC ที่เป็นของน้ำหนักต่อชิ้นคงที่ ลูกค้าสั่งเป็นชิ้น -> วัดด้วยชิ้น
//   - นอกนั้น (PC) -> วัดด้วยชิ้น
func saleItemCompletionMode(saleUnit string, saleUnitType string) string {
	unit := strings.ToUpper(strings.TrimSpace(saleUnit))
	unitType := strings.ToUpper(strings.TrimSpace(saleUnitType))
	unitType = strings.ReplaceAll(unitType, "-", "_")

	if unit == "KG" && unitType != "KG_SPEC" {
		return completionModeWeight
	}

	return completionModeQty
}

// completionTarget คือยอดที่ต้องส่งให้ถึงของบรรทัดนั้น ตามโหมดที่ใช้วัด
func completionTarget(item models.SaleItem) float64 {
	if saleItemCompletionMode(item.SaleUnit, item.SaleUnitType) == completionModeWeight {
		return item.TotalWeight
	}

	return item.Qty
}

// isFullyDelivered ถือว่าครบเมื่อยอดที่ตัดจ่ายจริงถึงขอบล่างของระยะผ่อนผัน
//
// ไม่มีเพดานบน — ส่งเกินก็ถือว่าจบงานแล้ว (โค้ดเดิมฝั่ง wms-outbound ใช้กรอบ min..max
// ทำให้ใบที่ส่งเกินไม่ถูกปิดตลอดกาล)
func isFullyDelivered(target float64, issued float64, tolerancePercent float64) bool {
	if target <= 0 {
		return false
	}

	if tolerancePercent < 0 {
		tolerancePercent = 0
	}

	return issued >= target*(1-tolerancePercent/100)
}
```

- [ ] **Step 4: รันเทสให้ผ่าน**

```bash
cd /c/work-prime/erp-core && go test ./internal/services/delivery-service/ -v && go build ./...
```

Expected: PASS ทุกเคส (รวมเทสเดิมของ `validate-booking-qty_test.go` และ `get-delivery_test.go`)

- [ ] **Step 5: Commit**

```bash
cd /c/work-prime/erp-core
git add internal/services/delivery-service/close-sale-on-delivered.go internal/services/delivery-service/close-sale-on-delivered_test.go docs/superpowers/specs/2026-09-09-close-so-on-delivery-complete.md docs/superpowers/plans/2026-09-09-close-so-on-delivery-complete.md
git commit -m "feat(sale): add completion mode and tolerance rules for closing a sale line"
```

---

## Task 2: อ่านยอด GI ให้ได้ทั้งจำนวนและน้ำหนัก

**Files:**
- Modify: `C:\work-prime\erp-core\internal\services\delivery-service\validate-booking-qty.go:248-275`
- Test: `C:\work-prime\erp-core\internal\services\delivery-service\close-sale-on-delivered_test.go` (เพิ่มท้ายไฟล์)

**Interfaces:**
- Consumes: `orderExternalService.GetOrderDeliveryResponse` (`external/order-service/get-order-delivery.go:20`), `OutboundItemWithGoodsIssue` (:150), `GetIssueItemResponse` (:155 มี `Qty`, `Weight`, `Status`), `buildOrder` helper ที่มีอยู่แล้วใน `validate-booking-qty_test.go:9`
- Produces: `foldWmsIssued(orders []orderExternalService.GetOrderDeliveryResponse) (map[string]float64, map[string]float64, map[string]bool)` — คืน (issuedQty, issuedWeight, closed) คีย์ `"<delivery_code>|<delivery_item>"`; `foldWmsProgress` คง signature เดิม

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน**

เพิ่ม import `orderExternalService "prime-erp-core/external/order-service"` ที่หัว `close-sale-on-delivered_test.go` แล้วเพิ่มท้ายไฟล์:

```go
// buildOutboundGI สร้าง outbound 1 ตัวพร้อมบรรทัด GI (qty, weight) — ตั้งชื่อไม่ให้ชนกับ
// buildOutbound ใน validate-booking-qty_test.go ที่รับแต่ qty
func buildOutboundGI(status string, lines ...[2]float64) orderExternalService.OutboundItemWithGoodsIssue {
	outbound := orderExternalService.OutboundItemWithGoodsIssue{}
	outbound.Status = status

	for _, line := range lines {
		outbound.GoodsIssueItem = append(outbound.GoodsIssueItem, orderExternalService.GetIssueItemResponse{
			Qty:    line[0],
			Weight: line[1],
		})
	}

	return outbound
}

// เคสจริง GI20260909051617-0 ของ CO2609-00032: 2 บรรทัด 7+3 ชิ้น = 36.05+15.50 kg
func TestFoldWmsIssuedSumsQtyAndWeight(t *testing.T) {
	issuedQty, issuedWeight, closed := foldWmsIssued([]orderExternalService.GetOrderDeliveryResponse{
		buildOrder("DBS202609-0012", "ITEM-1", "COMPLETED", []orderExternalService.OutboundItemWithGoodsIssue{
			buildOutboundGI("COMPLETED", [2]float64{7, 36.05}, [2]float64{3, 15.5}),
		}),
	})

	key := "DBS202609-0012|ITEM-1"
	if !closed[key] {
		t.Fatalf("closed[%s] = false, want true", key)
	}
	if issuedQty[key] != 10 {
		t.Errorf("issuedQty[%s] = %v, want 10", key, issuedQty[key])
	}
	if issuedWeight[key] != 51.55 {
		t.Errorf("issuedWeight[%s] = %v, want 51.55", key, issuedWeight[key])
	}
}

func TestFoldWmsIssuedIgnoresOpenOrderItemAndUncompletedOutbound(t *testing.T) {
	issuedQty, issuedWeight, closed := foldWmsIssued([]orderExternalService.GetOrderDeliveryResponse{
		buildOrder("DBS-1", "ITEM-1", "PENDING", []orderExternalService.OutboundItemWithGoodsIssue{
			buildOutboundGI("COMPLETED", [2]float64{8, 80}),
		}),
		buildOrder("DBS-2", "ITEM-2", "COMPLETED", []orderExternalService.OutboundItemWithGoodsIssue{
			buildOutboundGI("PENDING", [2]float64{5, 50}),
		}),
	})

	if closed["DBS-1|ITEM-1"] {
		t.Error("CO ยัง PENDING ต้องไม่ถือว่าปิด")
	}
	if issuedQty["DBS-1|ITEM-1"] != 0 || issuedWeight["DBS-1|ITEM-1"] != 0 {
		t.Error("บรรทัดที่ CO ยังไม่ปิด ต้องไม่นับยอด GI")
	}
	if issuedQty["DBS-2|ITEM-2"] != 0 || issuedWeight["DBS-2|ITEM-2"] != 0 {
		t.Error("outbound ที่ยังไม่ COMPLETED ต้องไม่ถูกนับ")
	}
}
```

- [ ] **Step 2: รันเทสให้เห็นว่าพัง**

```bash
cd /c/work-prime/erp-core && go test ./internal/services/delivery-service/ -run TestFoldWmsIssued -v
```

Expected: FAIL — `undefined: foldWmsIssued`

- [ ] **Step 3: แทนที่ `foldWmsProgress` เดิม (บรรทัด 248-275 ของ `validate-booking-qty.go`) ด้วยสองฟังก์ชันนี้**

```go
// foldWmsProgress คงรูปเดิมไว้ให้ ValidateBookingQty ใช้ (สนใจแค่จำนวน)
// แยกออกมาเพื่อให้เทสกติกา closed/issued ได้โดยไม่ต้องมี WMS จริง
func foldWmsProgress(orders []orderExternalService.GetOrderDeliveryResponse) (map[string]float64, map[string]bool) {
	issuedQty, _, closed := foldWmsIssued(orders)
	return issuedQty, closed
}

// foldWmsIssued พับความคืบหน้าฝั่งคลังเป็น 3 map คีย์ "<delivery_code>|<delivery_item>"
// ยอดน้ำหนักต้องแยกจากจำนวน เพราะบรรทัดที่ขายเป็นกิโลตัดสินความครบด้วยน้ำหนัก
func foldWmsIssued(orders []orderExternalService.GetOrderDeliveryResponse) (map[string]float64, map[string]float64, map[string]bool) {
	issuedQty := map[string]float64{}
	issuedWeight := map[string]float64{}
	closed := map[string]bool{}

	for _, order := range orders {
		for _, orderItem := range order.OrderItem {
			if !closedOrderItemStatuses[orderItem.Status] {
				continue
			}

			key := fmt.Sprintf("%s|%s", order.DocumentRef, orderItem.DocumentRefItem)
			closed[key] = true

			// ยอดที่ตัดจ่ายจริงยังอ่านจาก goods issue ของ outbound เหมือนเดิม
			for _, outbound := range orderItem.OutboundItem {
				if outbound.Status != "COMPLETED" {
					continue
				}
				for _, issued := range outbound.GoodsIssueItem {
					issuedQty[key] += issued.Qty
					issuedWeight[key] += issued.Weight
				}
			}
		}
	}

	return issuedQty, issuedWeight, closed
}
```

- [ ] **Step 4: รันเทสให้ผ่าน**

```bash
cd /c/work-prime/erp-core && go test ./internal/services/delivery-service/ -v && go build ./...
```

Expected: PASS ทั้งเทสใหม่และเทสเดิมของ `foldWmsProgress`

- [ ] **Step 5: Commit**

```bash
cd /c/work-prime/erp-core
git add internal/services/delivery-service/validate-booking-qty.go internal/services/delivery-service/close-sale-on-delivered_test.go
git commit -m "feat(delivery): read issued weight alongside issued qty from WMS progress"
```

---

## Task 3: รวมยอดที่ส่งแล้วต่อ 1 sale_item ข้ามทุกใบจอง

**Files:**
- Modify: `C:\work-prime\erp-core\internal\services\delivery-service\close-sale-on-delivered.go` (เพิ่มท้ายไฟล์ + เพิ่ม `"fmt"` ใน import)
- Test: `C:\work-prime\erp-core\internal\services\delivery-service\close-sale-on-delivered_test.go`

**Interfaces:**
- Consumes: `foldWmsIssued` (Task 2)
- Produces: `type deliveryLine struct { DeliveryCode string; DeliveryItem string; SaleItemCode string }` และ `issuedBySaleItem(lines []deliveryLine, issuedQty map[string]float64, issuedWeight map[string]float64, closed map[string]bool) (map[string]float64, map[string]float64)`

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน**

เพิ่มท้าย `close-sale-on-delivered_test.go`:

```go
// SO202609-0009 บรรทัด 94e41ee0… = 50 PCS ถูกแบ่งจอง 3 ใบ 30 + 3 + 17
// ปิดครบทั้ง 3 ใบ -> รวมได้ 50 ชิ้น
func TestIssuedBySaleItemSumsAcrossEveryBooking(t *testing.T) {
	lines := []deliveryLine{
		{DeliveryCode: "DBS202609-0009", DeliveryItem: "ITEM-A", SaleItemCode: "SALE-1"},
		{DeliveryCode: "DBS202609-0010", DeliveryItem: "ITEM-B", SaleItemCode: "SALE-1"},
		{DeliveryCode: "DBS202609-0011", DeliveryItem: "ITEM-C", SaleItemCode: "SALE-1"},
	}

	issuedQty := map[string]float64{
		"DBS202609-0009|ITEM-A": 30,
		"DBS202609-0010|ITEM-B": 3,
		"DBS202609-0011|ITEM-C": 17,
	}
	issuedWeight := map[string]float64{
		"DBS202609-0009|ITEM-A": 87,
		"DBS202609-0010|ITEM-B": 8.7,
		"DBS202609-0011|ITEM-C": 49.3,
	}
	closed := map[string]bool{
		"DBS202609-0009|ITEM-A": true,
		"DBS202609-0010|ITEM-B": true,
		"DBS202609-0011|ITEM-C": true,
	}

	qty, weight := issuedBySaleItem(lines, issuedQty, issuedWeight, closed)

	if qty["SALE-1"] != 50 {
		t.Errorf("qty[SALE-1] = %v, want 50", qty["SALE-1"])
	}
	if weight["SALE-1"] != 145 {
		t.Errorf("weight[SALE-1] = %v, want 145", weight["SALE-1"])
	}
}

// ใบแรกจบแล้วแต่อีก 2 ใบยังไม่เดิน -> ต้องได้แค่ 30 (ห้ามปิด SO ที่เหลือของค้าง)
func TestIssuedBySaleItemSkipsBookingsThatAreNotClosedYet(t *testing.T) {
	lines := []deliveryLine{
		{DeliveryCode: "DBS202609-0009", DeliveryItem: "ITEM-A", SaleItemCode: "SALE-1"},
		{DeliveryCode: "DBS202609-0010", DeliveryItem: "ITEM-B", SaleItemCode: "SALE-1"},
		{DeliveryCode: "DBS202609-0011", DeliveryItem: "ITEM-C", SaleItemCode: "SALE-1"},
	}

	issuedQty := map[string]float64{"DBS202609-0009|ITEM-A": 30}
	issuedWeight := map[string]float64{"DBS202609-0009|ITEM-A": 87}
	closed := map[string]bool{"DBS202609-0009|ITEM-A": true}

	qty, _ := issuedBySaleItem(lines, issuedQty, issuedWeight, closed)

	if qty["SALE-1"] != 30 {
		t.Errorf("qty[SALE-1] = %v, want 30", qty["SALE-1"])
	}
}

func TestIssuedBySaleItemIgnoresLinesWithoutSaleItem(t *testing.T) {
	lines := []deliveryLine{
		{DeliveryCode: "DBS-1", DeliveryItem: "ITEM-A", SaleItemCode: ""},
	}

	qty, weight := issuedBySaleItem(lines,
		map[string]float64{"DBS-1|ITEM-A": 9},
		map[string]float64{"DBS-1|ITEM-A": 9},
		map[string]bool{"DBS-1|ITEM-A": true})

	if len(qty) != 0 || len(weight) != 0 {
		t.Errorf("บรรทัดที่ไม่มี sale_item ต้องถูกข้าม ได้ qty=%v weight=%v", qty, weight)
	}
}
```

- [ ] **Step 2: รันเทสให้เห็นว่าพัง**

```bash
cd /c/work-prime/erp-core && go test ./internal/services/delivery-service/ -run TestIssuedBySaleItem -v
```

Expected: FAIL — `undefined: deliveryLine`

- [ ] **Step 3: เพิ่มลงท้าย `close-sale-on-delivered.go`**

```go
// deliveryLine คือ 1 บรรทัดของใบจอง ที่ผูกกับบรรทัดขายผ่าน document_ref_item
type deliveryLine struct {
	DeliveryCode string
	DeliveryItem string
	SaleItemCode string
}

// issuedBySaleItem รวมยอดที่ตัดจ่ายจริงของทุกใบจอง กลับมาเป็นยอดต่อ 1 บรรทัดขาย
//
// นับเฉพาะบรรทัดที่ฝั่งคลังปิดงานแล้ว บรรทัดที่ยังเดินอยู่แปลว่าของยังไม่ออก
// (ใบจองใบอื่นของ SO เดียวกันที่ยังไม่ถึงคิว ต้องไม่ทำให้ SO ถูกปิดก่อนเวลา)
func issuedBySaleItem(lines []deliveryLine, issuedQty map[string]float64, issuedWeight map[string]float64, closed map[string]bool) (map[string]float64, map[string]float64) {
	qtyOf := map[string]float64{}
	weightOf := map[string]float64{}

	for _, line := range lines {
		if line.SaleItemCode == "" {
			continue
		}

		key := fmt.Sprintf("%s|%s", line.DeliveryCode, line.DeliveryItem)
		if !closed[key] {
			continue
		}

		qtyOf[line.SaleItemCode] += issuedQty[key]
		weightOf[line.SaleItemCode] += issuedWeight[key]
	}

	return qtyOf, weightOf
}
```

- [ ] **Step 4: รันเทสให้ผ่าน**

```bash
cd /c/work-prime/erp-core && go test ./internal/services/delivery-service/ -v && go build ./...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /c/work-prime/erp-core
git add internal/services/delivery-service/close-sale-on-delivered.go internal/services/delivery-service/close-sale-on-delivered_test.go
git commit -m "feat(sale): aggregate issued qty and weight per sale line across bookings"
```

---

## Task 4: แยกตัวปิดบรรทัด/ปิดหัวใบออกจาก UpdateSaleItemStatus

**Files:**
- Modify: `C:\work-prime\erp-core\internal\services\sale-service\update-sale-item-status.go` (แทนบล็อกบรรทัด ~63-140)
- Test: `C:\work-prime\erp-core\internal\services\sale-service\update-sale-item-status_test.go` (สร้างใหม่)

**Interfaces:**
- Consumes: `models.SaleItem`, `models.Sale`, `*gorm.DB` (tx ที่ caller เปิดไว้)
- Produces: `saleService.MarkSaleItemsCompleted(tx *gorm.DB, saleItemCodes []string, status string, user string, now time.Time) (updatedSaleCodes []string, completedSaleCodes []string, err error)` — ต้อง export เพราะ `delivery-service` เรียกข้าม package; `saleService.IsSaleItemClosed(status string) bool` (export ด้วย เหตุผลเดียวกัน — Task 5 ใช้ต่อ); และ `allSaleItemsClosed(statuses []string) bool`

**พฤติกรรมที่เปลี่ยนโดยตั้งใจ:** เดิมปิดหัวใบเมื่อทุกบรรทัดเป็น `COMPLETED` เท่านั้น ทำให้ใบที่มีบรรทัดถูกยกเลิกไม่มีวันปิด ของใหม่ให้ถือว่า `CANCELED` / `CANCELLED` คือบรรทัดที่จบงานแล้วเช่นกัน และเพิ่มด่านกันไม่ให้ใบที่ `CANCELED`/`COMPLETED` อยู่แล้วถูกเขียนทับ

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน**

สร้าง `update-sale-item-status_test.go`:

```go
package saleService

import "testing"

func TestAllSaleItemsClosedCountsCanceledLinesAsDone(t *testing.T) {
	cases := []struct {
		name     string
		statuses []string
		want     bool
	}{
		{"ครบทุกบรรทัด", []string{"COMPLETED", "COMPLETED"}, true},
		{"มีบรรทัดที่ถูกยกเลิกปนอยู่", []string{"COMPLETED", "CANCELED"}, true},
		{"สะกดแบบสองแอล", []string{"COMPLETED", "CANCELLED"}, true},
		{"ยังมีบรรทัดค้าง", []string{"COMPLETED", "PENDING"}, false},
		{"ไม่มีบรรทัดเลย", []string{}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := allSaleItemsClosed(tc.statuses); got != tc.want {
				t.Errorf("allSaleItemsClosed(%v) = %v, want %v", tc.statuses, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: รันเทสให้เห็นว่าพัง**

```bash
cd /c/work-prime/erp-core && go test ./internal/services/sale-service/ -run TestAllSaleItemsClosed -v
```

Expected: FAIL — `undefined: allSaleItemsClosed`

- [ ] **Step 3: เพิ่ม 3 อย่างนี้ก่อนฟังก์ชัน `UpdateSaleItemStatus` ในไฟล์เดียวกัน**

```go
// closedSaleItemStatuses คือสถานะของบรรทัดขายที่ถือว่างานบรรทัดนั้นจบแล้ว
// ยกเลิกก็นับว่าจบ ไม่งั้นใบที่มีบรรทัดถูกยกเลิกจะไม่มีวันปิดหัวใบ
var closedSaleItemStatuses = map[string]bool{
	"COMPLETED": true,
	"CANCELED":  true,
	"CANCELLED": true,
}

// IsSaleItemClosed บอกว่าบรรทัดขายบรรทัดเดียวจบงานแล้วหรือยัง
// export เพราะ delivery-service ต้องใช้กติกาเดียวกันตอนเลือกบรรทัดที่จะปิด
func IsSaleItemClosed(status string) bool {
	return closedSaleItemStatuses[status]
}

// allSaleItemsClosed บอกว่าทุกบรรทัดของใบนั้นจบงานแล้วหรือยัง
func allSaleItemsClosed(statuses []string) bool {
	if len(statuses) == 0 {
		return false
	}

	for _, status := range statuses {
		if !IsSaleItemClosed(status) {
			return false
		}
	}

	return true
}

// MarkSaleItemsCompleted ตั้งสถานะบรรทัดขายที่ระบุ แล้วปิดหัวใบที่บรรทัดจบครบทุกบรรทัด
//
// ทำงานบน tx ที่ผู้เรียกเปิดมา เพื่อให้เส้นอัตโนมัติ (ปิดตอนใบจองส่งของครบ ดูที่
// delivery-service/close-sale-on-delivered.go) กับเส้นมือ (POST /sale/UpdateSaleItemStatus)
// ใช้กติกาเดียวกันจริงๆ ไม่ใช่เขียนซ้ำสองที่
func MarkSaleItemsCompleted(tx *gorm.DB, saleItemCodes []string, status string, user string, now time.Time) ([]string, []string, error) {
	updatedSaleCodes := []string{}
	completedSaleCodes := []string{}

	if len(saleItemCodes) == 0 {
		return updatedSaleCodes, completedSaleCodes, nil
	}

	updateFields := map[string]interface{}{
		"status":      status,
		"update_date": now,
		"update_by":   user,
	}

	if err := tx.Model(&models.SaleItem{}).
		Where("sale_item IN ?", saleItemCodes).
		Updates(updateFields).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to update sale items status: %v", err)
	}

	var affectedSaleIDs []uuid.UUID
	if err := tx.Model(&models.SaleItem{}).
		Where("sale_item IN ?", saleItemCodes).
		Distinct("sale_id").
		Pluck("sale_id", &affectedSaleIDs).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to get affected sale IDs: %v", err)
	}

	for _, saleID := range affectedSaleIDs {
		var sale models.Sale
		if err := tx.Where("id = ?", saleID).First(&sale).Error; err != nil {
			return nil, nil, fmt.Errorf("failed to get sale for ID %v: %v", saleID, err)
		}
		updatedSaleCodes = append(updatedSaleCodes, sale.SaleCode)

		var saleItems []models.SaleItem
		if err := tx.Where("sale_id = ?", saleID).Find(&saleItems).Error; err != nil {
			return nil, nil, fmt.Errorf("failed to get sale items for sale ID %v: %v", saleID, err)
		}

		statuses := make([]string, 0, len(saleItems))
		for _, item := range saleItems {
			statuses = append(statuses, item.Status)
		}

		if !allSaleItemsClosed(statuses) {
			continue
		}

		// ใบที่ยกเลิกหรือปิดไปแล้ว ห้ามถูกเขียนทับ
		if sale.Status == "CANCELED" || sale.Status == "COMPLETED" {
			continue
		}

		if err := tx.Model(&models.Sale{}).
			Where("id = ?", saleID).
			Updates(map[string]interface{}{
				"status":      "COMPLETED",
				"update_date": now,
				"update_by":   user,
			}).Error; err != nil {
			return nil, nil, fmt.Errorf("failed to update sale status to completed for sale ID %v: %v", saleID, err)
		}

		completedSaleCodes = append(completedSaleCodes, sale.SaleCode)
	}

	return updatedSaleCodes, completedSaleCodes, nil
}
```

- [ ] **Step 4: ให้ `UpdateSaleItemStatus` เรียก helper แทนโค้ดเดิม**

ลบทั้งช่วงตั้งแต่คอมเมนต์ `// Update sale items status` จนถึงบรรทัดก่อน `// Commit transaction` แล้วใส่แทนด้วย:

```go
	updatedSaleCodes, completedSaleCodes, err := MarkSaleItemsCompleted(tx, req.SaleItem, req.Status, user, nowDateOnly)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
```

ท้ายฟังก์ชันเดิมใช้ตัวแปรชื่อ `updatedSaleCodes` / `completedSaleCodes` อยู่แล้ว จึงไม่ต้องแก้ส่วน response

เพิ่ม `"gorm.io/gorm"` ใน import block (ตัวอื่นที่ใช้อยู่แล้ว: `encoding/json`, `errors`, `fmt`, `time`, `db`, `models`, `gin`, `uuid`) และลบ import ที่เลิกใช้ถ้ามี

- [ ] **Step 5: รันเทสให้ผ่าน**

```bash
cd /c/work-prime/erp-core && go test ./... && go build ./...
```

Expected: PASS ทั้ง repo

- [ ] **Step 6: Commit**

```bash
cd /c/work-prime/erp-core
git add internal/services/sale-service/update-sale-item-status.go internal/services/sale-service/update-sale-item-status_test.go
git commit -m "refactor(sale): extract MarkSaleItemsCompleted for reuse by the automatic path"
```

---

## Task 5: ต่อสายเข้า UpdateStatusDelivery

**Files:**
- Modify: `C:\work-prime\erp-core\internal\services\delivery-service\close-sale-on-delivered.go`
- Modify: `C:\work-prime\erp-core\internal\services\delivery-service\update-status-delivery.go` (ช่วงหลัง `tx.Commit()` ก่อน `return res, nil`)
- Test: `C:\work-prime\erp-core\internal\services\delivery-service\close-sale-on-delivered_test.go`

**Interfaces:**
- Consumes: `issuedBySaleItem`, `completionTarget`, `isFullyDelivered`, `saleItemCompletionMode`, `foldWmsIssued` (Task 1-3), `saleService.MarkSaleItemsCompleted` (Task 4), `systemConfigRepository.GetSystemConfig(topicCodes []string, configCodes []string) ([]models.SystemConfig, error)` (`internal/repositories/systemConfig/repository.go:9`), `orderExternalService.GetOrdersDelivery`, `models.Delivery` / `models.DeliveryItem` (`internal/models/delivery.go`)
- Produces: `CloseSalesFullyDelivered(gormx *gorm.DB, saleCodes []string, user string) error`, `selectDeliveredSaleItems(items []models.SaleItem, issuedQty map[string]float64, issuedWeight map[string]float64, tolerancePercent float64) []string`, `toleranceSOPercent() float64`

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน**

เพิ่มท้าย `close-sale-on-delivered_test.go`:

```go
// SO202609-0011 = 10 PCS ส่งครบ 10 -> ปิด
// SO202609-0009 บรรทัด 50 PCS เพิ่งส่งไป 30 -> ไม่ปิด
// บรรทัดกิโล 500 kg ส่ง 487 (ขาด 2.6%) -> ปิด เพราะอยู่ในระยะผ่อนผัน 3%
func TestSelectDeliveredSaleItemsPicksOnlyLinesThatMetTheTarget(t *testing.T) {
	items := []models.SaleItem{
		{SaleItem: "SALE-DONE", Qty: 10, TotalWeight: 51.5, SaleUnit: "PC", SaleUnitType: "PC", Status: "PENDING"},
		{SaleItem: "SALE-PARTIAL", Qty: 50, TotalWeight: 145, SaleUnit: "PC", SaleUnitType: "PC", Status: "PENDING"},
		{SaleItem: "SALE-KG", Qty: 53, TotalWeight: 500, SaleUnit: "KG", SaleUnitType: "KG", Status: "PENDING"},
		{SaleItem: "SALE-ALREADY", Qty: 5, TotalWeight: 5, SaleUnit: "PC", SaleUnitType: "PC", Status: "COMPLETED"},
	}

	issuedQty := map[string]float64{
		"SALE-DONE":    10,
		"SALE-PARTIAL": 30,
		"SALE-KG":      53,
		"SALE-ALREADY": 5,
	}
	issuedWeight := map[string]float64{
		"SALE-DONE":    51.55,
		"SALE-PARTIAL": 87,
		"SALE-KG":      487,
		"SALE-ALREADY": 5,
	}

	got := selectDeliveredSaleItems(items, issuedQty, issuedWeight, 3)

	if len(got) != 2 {
		t.Fatalf("selectDeliveredSaleItems = %v, want 2 รายการ (SALE-DONE, SALE-KG)", got)
	}
	if got[0] != "SALE-DONE" || got[1] != "SALE-KG" {
		t.Errorf("selectDeliveredSaleItems = %v, want [SALE-DONE SALE-KG]", got)
	}
}

// บรรทัด KG_SPEC ต้องวัดด้วยชิ้น ไม่ใช่น้ำหนัก (SO202609-0001 = 6 ชิ้น / 120 kg)
func TestSelectDeliveredSaleItemsUsesQtyForKgSpec(t *testing.T) {
	items := []models.SaleItem{
		{SaleItem: "SALE-KGSPEC", Qty: 6, TotalWeight: 120, SaleUnit: "KG", SaleUnitType: "KG_SPEC", Status: "PENDING"},
	}

	got := selectDeliveredSaleItems(items,
		map[string]float64{"SALE-KGSPEC": 6},
		map[string]float64{"SALE-KGSPEC": 0},
		3)

	if len(got) != 1 || got[0] != "SALE-KGSPEC" {
		t.Errorf("selectDeliveredSaleItems = %v, want [SALE-KGSPEC] (ครบ 6 ชิ้นแล้วแม้ไม่มีน้ำหนัก)", got)
	}
}

func TestSelectDeliveredSaleItemsSkipsLinesWithNoIssueAtAll(t *testing.T) {
	items := []models.SaleItem{
		{SaleItem: "SALE-NONE", Qty: 10, SaleUnit: "PC", SaleUnitType: "PC", Status: "PENDING"},
	}

	got := selectDeliveredSaleItems(items, map[string]float64{}, map[string]float64{}, 3)

	if len(got) != 0 {
		t.Errorf("selectDeliveredSaleItems = %v, want ว่าง", got)
	}
}
```

- [ ] **Step 2: รันเทสให้เห็นว่าพัง**

```bash
cd /c/work-prime/erp-core && go test ./internal/services/delivery-service/ -run TestSelectDeliveredSaleItems -v
```

Expected: FAIL — `undefined: selectDeliveredSaleItems`

- [ ] **Step 3: เพิ่มลงท้าย `close-sale-on-delivered.go` และปรับ import block ให้เป็น**

```go
import (
	"fmt"
	"strconv"
	"strings"
	"time"

	orderExternalService "prime-erp-core/external/order-service"
	"prime-erp-core/internal/models"
	systemConfigRepository "prime-erp-core/internal/repositories/systemConfig"
	saleService "prime-erp-core/internal/services/sale-service"

	"github.com/google/uuid"
	"gorm.io/gorm"
)
```

โค้ดที่เพิ่ม:

```go
// defaultToleranceSOPercent ใช้เมื่ออ่าน config ไม่ได้ (ค่าจริงของ TMI ตอนนี้คือ 3)
const defaultToleranceSOPercent = 5.0

// selectDeliveredSaleItems คืนรหัสบรรทัดขายที่ส่งของถึงเป้าแล้วและยังไม่ถูกปิด
//
// กติกา "บรรทัดนี้จบงานแล้ว" ใช้ของ sale-service ที่เดียว (saleService.IsSaleItemClosed)
// ไม่ประกาศ map ซ้ำ เพื่อไม่ให้สองที่เพี้ยนออกจากกันภายหลัง
func selectDeliveredSaleItems(items []models.SaleItem, issuedQty map[string]float64, issuedWeight map[string]float64, tolerancePercent float64) []string {
	delivered := []string{}

	for _, item := range items {
		if saleService.IsSaleItemClosed(item.Status) {
			continue
		}

		issued := issuedQty[item.SaleItem]
		if saleItemCompletionMode(item.SaleUnit, item.SaleUnitType) == completionModeWeight {
			issued = issuedWeight[item.SaleItem]
		}

		if !isFullyDelivered(completionTarget(item), issued, tolerancePercent) {
			continue
		}

		delivered = append(delivered, item.SaleItem)
	}

	return delivered
}

// toleranceSOPercent อ่านระยะผ่อนผันจาก system_config ของ ERP เอง (topic SO / TOLERANCE_SO)
func toleranceSOPercent() float64 {
	configs, err := systemConfigRepository.GetSystemConfig([]string{"SO"}, []string{"TOLERANCE_SO"})
	if err != nil || len(configs) == 0 {
		return defaultToleranceSOPercent
	}

	parsed, err := strconv.ParseFloat(strings.TrimSpace(configs[0].Value), 64)
	if err != nil {
		return defaultToleranceSOPercent
	}

	return parsed
}

// CloseSalesFullyDelivered ปิดบรรทัดขาย (และหัวใบเมื่อครบทุกบรรทัด) ของ SO ที่ส่งของครบแล้ว
//
// เรียกหลัง commit ของ UpdateStatusDelivery เท่านั้น และผู้เรียกต้อง "log ทิ้ง" ถ้าพัง
// ห้ามคืน error ขึ้นไปให้ hook เพราะ hook ORDER/DELIVERY/UPDATE ถูกยิงระหว่างที่
// wms-order-service ยังไม่ commit การคืน error จะทำให้ pack confirm ทั้งใบล้ม
// ทั้งที่สต็อกกับ GI ตัดไปแล้ว (ดูคอมเมนต์ที่ confirm-order-outbound.go:205-208)
func CloseSalesFullyDelivered(gormx *gorm.DB, saleCodes []string, user string) error {
	codes := []string{}
	seen := map[string]bool{}
	for _, saleCode := range saleCodes {
		if saleCode == "" || seen[saleCode] {
			continue
		}
		seen[saleCode] = true
		codes = append(codes, saleCode)
	}

	if len(codes) == 0 {
		return nil
	}

	var sales []models.Sale
	if err := gormx.Where("sale_code IN ? AND status NOT IN ?", codes, []string{"COMPLETED", "CANCELED"}).
		Find(&sales).Error; err != nil {
		return fmt.Errorf("failed to load sales %v: %v", codes, err)
	}

	if len(sales) == 0 {
		return nil
	}

	saleIDs := make([]uuid.UUID, 0, len(sales))
	openSaleCodes := make([]string, 0, len(sales))
	for _, sale := range sales {
		saleIDs = append(saleIDs, sale.ID)
		openSaleCodes = append(openSaleCodes, sale.SaleCode)
	}

	var saleItems []models.SaleItem
	if err := gormx.Where("sale_id IN ?", saleIDs).Find(&saleItems).Error; err != nil {
		return fmt.Errorf("failed to load sale items of %v: %v", openSaleCodes, err)
	}

	if len(saleItems) == 0 {
		return nil
	}

	// ใบจองทุกใบของ SO เหล่านี้ ใบที่ยกเลิกไม่นับเพราะของถูกคืนไปแล้ว
	var deliveries []models.Delivery
	if err := gormx.Where("document_ref IN ? AND status <> ?", openSaleCodes, "CANCELED").
		Find(&deliveries).Error; err != nil {
		return fmt.Errorf("failed to load delivery bookings of %v: %v", openSaleCodes, err)
	}

	if len(deliveries) == 0 {
		return nil
	}

	deliveryIDs := make([]uuid.UUID, 0, len(deliveries))
	deliveryCodes := make([]string, 0, len(deliveries))
	deliveryCodeOf := map[uuid.UUID]string{}
	for _, delivery := range deliveries {
		deliveryIDs = append(deliveryIDs, delivery.ID)
		deliveryCodes = append(deliveryCodes, delivery.DeliveryCode)
		deliveryCodeOf[delivery.ID] = delivery.DeliveryCode
	}

	var bookedItems []models.DeliveryItem
	if err := gormx.Where("delivery_id IN ?", deliveryIDs).Find(&bookedItems).Error; err != nil {
		return fmt.Errorf("failed to load delivery booking items of %v: %v", openSaleCodes, err)
	}

	lines := make([]deliveryLine, 0, len(bookedItems))
	for _, item := range bookedItems {
		lines = append(lines, deliveryLine{
			DeliveryCode: deliveryCodeOf[item.DeliveryID],
			DeliveryItem: item.DeliveryItem,
			SaleItemCode: item.DocumentRefItem,
		})
	}

	orderRes, err := orderExternalService.GetOrdersDelivery(orderExternalService.GetOrderDeliveryRequest{
		DeliveryCode: deliveryCodes,
	})
	if err != nil {
		return fmt.Errorf("failed to read WMS progress of %v: %v", deliveryCodes, err)
	}

	issuedQtyByLine, issuedWeightByLine, closed := foldWmsIssued(orderRes.Orders)
	issuedQty, issuedWeight := issuedBySaleItem(lines, issuedQtyByLine, issuedWeightByLine, closed)

	toClose := selectDeliveredSaleItems(saleItems, issuedQty, issuedWeight, toleranceSOPercent())
	if len(toClose) == 0 {
		return nil
	}

	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	tx := gormx.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	_, completedSaleCodes, err := saleService.MarkSaleItemsCompleted(tx, toClose, "COMPLETED", user, nowDateOnly)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}

	fmt.Printf("CloseSalesFullyDelivered: closed %d sale items %v, completed sales %v\n",
		len(toClose), toClose, completedSaleCodes)

	return nil
}
```

- [ ] **Step 4: ต่อสายใน `update-status-delivery.go`** — แทรกหลังบล็อก `if err := tx.Commit().Error; err != nil { ... }` และก่อน `return res, nil`

```go
	// ใบจองที่เพิ่งปิด แปลว่าของออกไปแล้ว ให้ไปดูว่า SO ต้นทางส่งครบหรือยัง
	//
	// ต้องทำหลัง commit และ "log ทิ้งถ้าพัง" ห้ามคืน error — hook ORDER/DELIVERY/UPDATE
	// ยิงเข้ามาระหว่างที่ wms-order-service ยังไม่ commit ถ้าเราคืน error ฝั่งนั้นจะ rollback
	// แล้วยืนยัน pack ล้มทั้งใบ ทั้งที่สต็อกกับ GI ตัดไปแล้ว
	if req.Status == "COMPLETED" {
		saleCodes := []string{}
		for _, deliveryCode := range toUpdate {
			saleCodes = append(saleCodes, deliveryOf[deliveryCode].DocumentRef)
		}

		if err := CloseSalesFullyDelivered(gormx, saleCodes, user); err != nil {
			fmt.Printf("UpdateStatusDelivery: cannot close sales of %v: %v\n", toUpdate, err)
		}
	}
```

- [ ] **Step 5: รันเทสและ build**

```bash
cd /c/work-prime/erp-core && go test ./... && go build ./...
```

Expected: PASS ทั้งหมด — ถ้าเจอ `import cycle not allowed` ให้ย้าย `MarkSaleItemsCompleted` ไปไว้ที่ `internal/repositories/sale/` แล้วให้ทั้งสอง service เรียก repository นั้น (ตรวจแล้วตอนเขียนแผนว่า `sale-service` ไม่ import `delivery-service` จึงไม่ควรเกิด)

- [ ] **Step 6: Commit**

```bash
cd /c/work-prime/erp-core
git add internal/services/delivery-service/close-sale-on-delivered.go internal/services/delivery-service/close-sale-on-delivered_test.go internal/services/delivery-service/update-status-delivery.go
git commit -m "feat(delivery): close fully delivered sales after a booking is completed"
```

---

## Task 6: ลบเส้นเดิมที่ไม่เคยถูกเรียกใน wms-outbound-service

**Files:**
- Delete: `C:\work-prime\wms-outbound-service\internal\services\so-service\update-complete-so.go`
- Delete: `C:\work-prime\wms-outbound-service\external\services\sale\update-sale-item-status.go`
- Modify: `C:\work-prime\wms-outbound-service\internal\routes\routes.go` (บล็อก `soRoutes` บรรทัด ~116-120 + import `soService`)
- Modify: `C:\work-prime\wms-outbound-service\config\constants.go:52`

**Interfaces:**
- Consumes: ไม่มี
- Produces: ไม่มี (เป็นการลบโค้ดตาย)

- [ ] **Step 1: ยืนยันอีกครั้งว่าไม่มีใครเรียก**

```bash
cd /c/work-prime && grep -rn --include=*.go --include=*.ts --include=*.vue "update-completed-so\|UpdateCompletedSo\|UpdateSaleItemStatus" wms-outbound-service/internal wms-outbound-service/external wms-web/src wms-order-service wms-document-service wms-pack-service wms-gr-service | grep -v "so-service/update-complete-so.go" | grep -v "external/services/sale/update-sale-item-status.go" | grep -v "routes.go" | grep -v "constants.go"
```

Expected: ไม่มีผลลัพธ์ — ถ้ามี ให้หยุดและรายงาน

- [ ] **Step 2: ลบไฟล์**

```bash
cd /c/work-prime/wms-outbound-service
rm internal/services/so-service/update-complete-so.go
rm external/services/sale/update-sale-item-status.go
```

- [ ] **Step 3: ตัด route และ constant**

ใน `internal/routes/routes.go` ลบทั้งบล็อกนี้:

```go
	soRoutes := ctx.Group("/so")

	soRoutes.POST("/update-completed-so", func(c *gin.Context) {
		utils.ProcessRequest(c, soService.UpdateCompletedSo)
	})
```

และลบ import ของ `soService` ที่หัวไฟล์

ใน `config/constants.go` ลบบรรทัด:

```go
	UPDATE_SALE_ITEM_STATUS_ENDPOINT = baseURL + "/erp/sale/UpdateSaleItemStatus"
```

- [ ] **Step 4: build ให้ผ่าน**

```bash
cd /c/work-prime/wms-outbound-service && go build ./... && go test ./...
```

Expected: PASS — ถ้าเจอ `imported and not used` ให้ลบ import ที่ค้าง; ถ้าโฟลเดอร์ `so-service` / `external/services/sale` ว่างแล้ว ปล่อยให้ git จัดการเอง (git ไม่เก็บโฟลเดอร์ว่าง)

- [ ] **Step 5: Commit**

```bash
cd /c/work-prime/wms-outbound-service
git add internal/routes/routes.go config/constants.go internal/services/so-service external/services/sale
git commit -m "chore(so): drop the never-wired SO completion path in favour of erp-core"
```

---

## Task 7: ตรวจงานด้วยข้อมูลจริง (ห้ามยิงเองก่อนได้อนุญาต)

**Files:** ไม่มีการแก้โค้ด

- [ ] **Step 1: ตรวจสถานะก่อนทดสอบ (อ่านอย่างเดียว ทำได้เลย)**

```bash
export PGPASSWORD='T8kL2mQ9Xr5Zp7V4aB'
"/c/Program Files/PostgreSQL/16/bin/psql.exe" -h 18.138.69.85 -U dev_theo -d prime_erp -c "select s.sale_code, s.status, si.sale_item, si.qty, si.total_weight, si.sale_unit, si.sale_unit_type, si.status from sale s join sale_item si on si.sale_id = s.id where s.sale_code in ('SO202609-0011','SO202609-0009') order by s.sale_code"
```

Expected: `SO202609-0009` 2 บรรทัด PENDING, `SO202609-0011` 1 บรรทัด 10 PCS PENDING

- [ ] **Step 2: หยุดแล้วขออนุญาตเจ้าของก่อนทดสอบจริง**

การทดสอบต้องยิง `POST /erp/delivery/UpdateStatusDelivery` ด้วย body
`{"delivery_codes":["DBS202609-0012"],"status":"COMPLETED"}` ซึ่ง **เขียน `sale` / `sale_item` ของ UAT**

ผลที่คาดไว้ (บอกเจ้าของก่อนกด):
- `SO202609-0011` → `COMPLETED` (GI 10/10 ชิ้น)
- `SO202609-0009` → ยัง `PENDING` (บรรทัด 50 ชิ้นเพิ่งส่ง 30, บรรทัดกิโลยังไม่ได้ส่ง)
- ใบที่เคยเป็น COMPLETED อยู่แล้วจะได้ข้อความ "Delivery is already COMPLETED" และไม่เข้าเงื่อนไข `toUpdate`
  ⇒ **การยิงซ้ำใบเดิมจะไม่ปิด SO ให้** ถ้าต้องทดสอบซ้ำ ให้ทดสอบกับ DBS ใบที่ยัง PENDING

- [ ] **Step 3: คำถามที่ต้องให้ SA ยืนยัน (รายงานไว้ ไม่บล็อกงาน)**

บรรทัดที่ `sale_unit = KG` แต่ `sale_unit_type = PC` (`SO202609-0003` = 34 ชิ้น / 100 kg)
แผนนี้ตัดสินให้วัดด้วย **น้ำหนัก** — ต้องให้ SA ยืนยันว่าถูก

- [ ] **Step 4: รายงานสรุปให้เจ้าของ**

- ไฟล์ที่เปลี่ยนทั้ง 2 repo + commit ที่เกิดขึ้น (ยังไม่ push ยังไม่ merge)
- ลำดับ deploy: **erp-core ก่อน** (มีของใหม่) แล้ว wms-outbound-service ตามได้ทุกเมื่อ (ลบโค้ดตาย ไม่มีคนเรียก)
- ใบเก่าที่ค้าง PENDING อยู่แล้ว 16 ใบ จะไม่ถูกปิดย้อนหลัง เพราะ hook ยิงตอนใบจองปิดเท่านั้น — ต้องถามเจ้าของว่าจะ backfill ด้วย `POST /sale/UpdateSaleItemStatus` หรือปล่อย
