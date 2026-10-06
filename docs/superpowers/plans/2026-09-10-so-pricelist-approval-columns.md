# ด่านราคา price list + คอลัมน์ราคา 2 ช่วงเวลา — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** ให้ผลตรวจราคาตอน convert QU→SO ถูกบันทึกจริง, ใบที่ราคาผ่านใช้ราคาจาก quotation, ใบที่ไม่ผ่านโชว์ราคา 2 ช่วงเวลาให้ผู้อนุมัติเทียบตลอดชีวิตใบ และเมื่ออนุมัติแล้วราคาที่ยึดคือราคาจาก quotation

**Architecture:** `erp-core` เลิกเขียนทับ pass flags และเพิ่มการเขียนราคากลับตอน approve; `wms-web` ส่งผลตรวจราคาจริงเข้าไป เลือกราคาตามผลตรวจ และเปลี่ยนที่มาของ 2 คอลัมน์บนจอจากการยิง API สดเป็นค่าที่บันทึกไว้

**Tech Stack:** Go + Gin + GORM (erp-core), Vue 3 + TS + Pinia (wms-web)

**Spec:** `C:\work-prime\erp-core\docs\superpowers\specs\2026-09-10-so-pricelist-approval-columns.md`

**Branch (แตกไว้แล้วทั้ง 2 repo):** `fix/so-pricelist-approval-columns`
- `C:\work-prime\erp-core` จาก `origin/Develop` = 858a043
- `C:\work-prime\wms-web` จาก `origin/Develop` = bc4ffd37

## Global Constraints

- commit message เป็นภาษาอังกฤษเสมอ คอมเมนต์ในโค้ดเขียนไทยได้ตามสไตล์ไฟล์ข้างเคียง
- ห้าม `git add -A` — เพิ่มไฟล์ทีละชื่อ (wms-web มี `shi-7-step-list.txt` ที่ไม่เกี่ยวข้องค้างอยู่)
- ห้ามยิง API หรือเขียน DB ของ UAT (18.138.69.85) — ทดสอบด้วย unit test เท่านั้น
- ห้ามแตะ `internal/services/price-service/**` ใน erp-core — เจ้าของ service รับไปแก้เองแล้ว
- erp-core: `go build ./...` + `go test ./...` ผ่าน (ยกเว้น 3 suite ที่ต้องมี Docker daemon: `price_list_formulas`, `price_list_sub_group`, `priceList` — พังอยู่ก่อนแล้ว)
- wms-web: `npx vue-tsc --noEmit` ต้องไม่เพิ่ม error จาก baseline (baseline พังอยู่แล้วหลายพัน ให้เทียบเป็นจำนวน) และ `npx vitest run <ไฟล์ที่เกี่ยว>` ผ่าน
- ค่าที่ยอมรับของ pass flag คือ `"Y"` / `"N"` เท่านั้น

---

## File Structure

| ไฟล์ | หน้าที่ |
|---|---|
| แก้ `erp-core/internal/services/sale-service/create-sale.go` | เลิก hardcode pass flags |
| สร้าง `erp-core/internal/services/sale-service/pass-flag.go` + test | ตัว normalize ค่า Y/N |
| แก้ `erp-core/internal/services/sale-service/update-status-approve-sale.go` | approve แล้วเขียนราคาจาก quotation กลับ |
| แก้ `wms-web/src/components/quotation/quotationCreate/index.vue` | ส่ง `isPassPrice` เข้า store |
| แก้ `wms-web/src/stores/saleOrder/sale-order-create.store.ts` | เขียน pass flag จริง + เลือกราคาตามผลตรวจ + ไม่ทับ old |
| แก้ `wms-web/src/types/saleOrder/sale-order-list.type.ts` | เพิ่ม `oldPriceListUnit` |
| แก้ `wms-web/src/components/saleOrder/saleOrderCreate/index.vue` | เลิกยิง getPriceList สด + map ค่าเข้า 2 คอลัมน์ |
| แก้ `wms-web/src/components/saleOrder/saleOrderCreate/saleOrderDetail/index.vue` | เปลี่ยนเงื่อนไขโชว์ 2 คอลัมน์ |

---

## Task 1: erp-core เลิกเขียนทับ pass flags

**Files:**
- Create: `internal/services/sale-service/pass-flag.go`
- Create: `internal/services/sale-service/pass-flag_test.go`
- Modify: `internal/services/sale-service/create-sale.go:134-138`

**Interfaces:**
- Produces: `normalizePassFlag(value string) string` — คืน `"N"` เมื่อค่าที่ส่งมาคือ N (ไม่สนตัวพิมพ์/ช่องว่าง) นอกนั้นคืน `"Y"`

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน** — สร้าง `pass-flag_test.go`

```go
package saleService

import "testing"

func TestNormalizePassFlagKeepsOnlyExplicitN(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"N", "N"},
		{"n", "N"},
		{" N ", "N"},
		{"Y", "Y"},
		{"y", "Y"},
		{"", "Y"},
		{"TRUE", "Y"},
	}

	for _, tc := range cases {
		if got := normalizePassFlag(tc.in); got != tc.want {
			t.Errorf("normalizePassFlag(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: รันเทสให้เห็นว่าพัง**

```bash
cd /c/work-prime/erp-core && go test ./internal/services/sale-service/ -run TestNormalizePassFlag -v
```

Expected: FAIL — `undefined: normalizePassFlag`

- [ ] **Step 3: สร้าง `pass-flag.go`**

```go
package saleService

import "strings"

// normalizePassFlag คุมค่า flag ผลตรวจให้เหลือแค่ Y/N
//
// ของเดิม create-sale.go เขียนทับเป็น "Y" ทุกใบ (คอมเมนต์บอกว่า "หน้าบ้าน validate แล้ว")
// ผลคือใบที่ราคาไม่ผ่านก็ถูกบันทึกว่าผ่าน และไม่มีทางรู้ย้อนหลังว่าใบไหนผ่านแบบมีเงื่อนไข
//
// ค่าที่ไม่ใช่ N ให้ถือเป็น Y เพราะ payload เก่าที่ไม่ได้ส่ง flag มาต้องได้พฤติกรรมเดิม
func normalizePassFlag(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "N") {
		return "N"
	}

	return "Y"
}
```

- [ ] **Step 4: แก้ `create-sale.go`** — แทนที่บล็อกบรรทัด 134-138

```go
		// ผลตรวจมาจากหน้าบ้าน (ยิง ValidateSaleOrder ก่อน convert) ห้ามเขียนทับเป็น Y
		// ไม่งั้นใบที่ราคาไม่ผ่านจะดูเหมือนผ่าน และจอรออนุมัติจะหาใบไม่เจอ
		tempSale.PassPriceList = normalizePassFlag(tempSale.PassPriceList)
		tempSale.PassPriceExpire = normalizePassFlag(tempSale.PassPriceExpire)
		tempSale.PassCreditLimit = normalizePassFlag(tempSale.PassCreditLimit)
		tempSale.PassAtpCheck = normalizePassFlag(tempSale.PassAtpCheck)
```

- [ ] **Step 5: รันเทสและ build**

```bash
cd /c/work-prime/erp-core && go test ./internal/services/sale-service/ -v && go build ./...
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd /c/work-prime/erp-core
git add internal/services/sale-service/pass-flag.go internal/services/sale-service/pass-flag_test.go internal/services/sale-service/create-sale.go docs/superpowers/specs/2026-09-10-so-pricelist-approval-columns.md docs/superpowers/plans/2026-09-10-so-pricelist-approval-columns.md
git commit -m "fix(sale): keep the price-check result sent by the client instead of forcing it to Y"
```

---

## Task 2: erp-core อนุมัติแล้วยึดราคาจาก quotation

**Files:**
- Modify: `internal/services/sale-service/update-status-approve-sale.go` (หลังบล็อกที่ update `models.Sale` สำเร็จ)

**Interfaces:**
- Consumes: `models.SaleItem` มี `PriceListUnit`, `OldPriceListUnit`, `SaleID`
- Produces: ไม่มี export ใหม่

**หมายเหตุ:** ในฟังก์ชันนี้ยังไม่มีตัวแปร `user` ให้ใช้ `ctx.GetString("user")` และ fallback เป็น `"system"` แบบเดียวกับ `update-status-sale.go`

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน** — สร้าง `internal/services/sale-service/approve-price-fallback_test.go`

```go
package saleService

import "testing"

func TestShouldAdoptQuotationPriceOnlyWhenApproved(t *testing.T) {
	cases := []struct {
		status string
		want   bool
	}{
		{"COMPLETED", true},
		{"REVIEW", false},
		{"REJECT", false},
		{"", false},
	}

	for _, tc := range cases {
		if got := shouldAdoptQuotationPrice(tc.status); got != tc.want {
			t.Errorf("shouldAdoptQuotationPrice(%q) = %v, want %v", tc.status, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: รันเทสให้เห็นว่าพัง**

```bash
cd /c/work-prime/erp-core && go test ./internal/services/sale-service/ -run TestShouldAdoptQuotationPrice -v
```

Expected: FAIL — `undefined: shouldAdoptQuotationPrice`

- [ ] **Step 3: เพิ่มโค้ดใน `update-status-approve-sale.go`**

เพิ่มฟังก์ชันท้ายไฟล์:

```go
// shouldAdoptQuotationPrice บอกว่าผลอนุมัตินี้ทำให้ใบยึดราคาจาก quotation หรือยัง
// อนุมัติผ่าน = ผู้อนุมัติยอมรับราคาที่ห่างจาก price list master แล้ว (SA เคาะ 2026-09-10)
func shouldAdoptQuotationPrice(status string) bool {
	return status == "COMPLETED"
}
```

แล้วแทรกหลังบล็อก `if err := gormx.Model(&models.Sale{}).Where("id = ?", req.ID).Updates(updateFields).Error; err != nil { ... }`:

```go
	// อนุมัติผ่านแล้วให้ราคาที่ใช้จริงกลับไปเป็นราคาจาก quotation
	// บรรทัดที่ไม่มีราคาจาก quotation (0) ไม่แตะ เพราะไม่มีอะไรให้ยึด
	if shouldAdoptQuotationPrice(req.Status) {
		user := ctx.GetString("user")
		if user == "" {
			user = `system`
		}

		if err := gormx.Model(&models.SaleItem{}).
			Where("sale_id = ? AND old_price_list_unit > 0", req.ID).
			Updates(map[string]interface{}{
				"price_list_unit": gorm.Expr("old_price_list_unit"),
				"update_date":     nowDateOnly,
				"update_by":       user,
			}).Error; err != nil {
			return nil, fmt.Errorf("failed to adopt quotation price for sale %v: %v", req.ID, err)
		}
	}
```

เพิ่ม `"gorm.io/gorm"` ใน import block

- [ ] **Step 4: รันเทสและ build**

```bash
cd /c/work-prime/erp-core && go test ./internal/services/sale-service/ -v && go build ./...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /c/work-prime/erp-core
git add internal/services/sale-service/update-status-approve-sale.go internal/services/sale-service/approve-price-fallback_test.go
git commit -m "feat(sale): adopt the quotation price list once the sale order is approved"
```

---

## Task 3: wms-web ส่งผลตรวจราคาจริง และเลือกราคาตามผล

**Files:**
- Modify: `src/components/quotation/quotationCreate/index.vue` (จุดเรียก `onCreateSaleOrder` ~บรรทัด 1229 และตัวฟังก์ชัน ~1240)
- Modify: `src/stores/saleOrder/sale-order-create.store.ts` (`createSaleOrderStore` ~41, `mapSaleOrderItem` ~275, `mapSaleOrderRequest` ~332)
- Test: `src/stores/saleOrder/__tests__/sale-order-price.spec.ts` (สร้างใหม่ — ถ้าโฟลเดอร์ `__tests__` ยังไม่มีให้วางไฟล์ข้างไฟล์ store แทน แล้วตั้งชื่อ `sale-order-create.store.spec.ts`)

**Interfaces:**
- Consumes: `isPassPrice` ที่คำนวณไว้แล้วใน `onValidateSo` (`index.vue:1186`)
- Produces: `resolveConvertPriceList(isPassPrice, quotationPriceList, refreshedPriceList)` — helper ตัวใหม่ที่ export จาก `src/utils/helper/quotationCalculation.ts`

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน**

```ts
import { describe, expect, it } from 'vitest';
import { resolveConvertPriceList } from '@/utils/helper/quotationCalculation';

describe('resolveConvertPriceList', () => {
  it('ราคาผ่าน -> ใช้ราคาจาก quotation', () => {
    expect(resolveConvertPriceList(true, 26.5, 54428890000)).toBe(26.5);
  });

  it('ราคาไม่ผ่าน -> ใช้ราคาที่ดึงใหม่ ให้ผู้อนุมัติเห็นตัวเทียบ', () => {
    expect(resolveConvertPriceList(false, 26.5, 54428890000)).toBe(54428890000);
  });

  it('ราคาไม่ผ่านแต่ดึงใหม่ไม่ได้ -> ตกกลับไปใช้ราคาจาก quotation', () => {
    expect(resolveConvertPriceList(false, 26.5, 0)).toBe(26.5);
  });

  it('ไม่มีราคาทั้งสองทาง -> 0', () => {
    expect(resolveConvertPriceList(true, 0, 0)).toBe(0);
  });
});
```

- [ ] **Step 2: รันเทสให้เห็นว่าพัง**

```bash
cd /c/work-prime/wms-web && npx vitest run src/stores/saleOrder
```

Expected: FAIL — ไม่มี export `resolveConvertPriceList`

- [ ] **Step 3: เพิ่ม helper ใน `src/utils/helper/quotationCalculation.ts`**

```ts
/**
 * เลือก price list ที่จะบันทึกลง SO ตอน convert
 *
 * ราคาผ่านด่าน = ยึดราคาที่เสนอลูกค้าไปใน quotation (SA เคาะ 2026-09-10)
 * ราคาไม่ผ่าน = เก็บราคาที่ดึงใหม่ไว้ให้ผู้อนุมัติเทียบกับราคาที่เสนอ
 * (ราคาที่เสนอถูกเก็บคู่กันไว้ที่ old_price_list_unit เสมอ)
 */
export const resolveConvertPriceList = (
  isPassPrice: boolean,
  quotationPriceList: number,
  refreshedPriceList: number,
): number => {
  if (isPassPrice) {
    return quotationPriceList;
  }

  return refreshedPriceList || quotationPriceList;
};
```

- [ ] **Step 4: ต่อสายใน store และหน้า quotation**

4.1 `createSaleOrderStore` รับพารามิเตอร์เพิ่ม `isPassPrice: boolean` (วางต่อจาก `isPass`) แล้วส่งต่อให้ `mapSaleOrderItem` และ `mapSaleOrderRequest`

4.2 `mapSaleOrderItem` — เปลี่ยนบรรทัด `price_list_unit`

```ts
            price_list_unit: resolveConvertPriceList(
              isPassPrice,
              element.priceListUnit || 0,
              resolvePriceListByUom(
                element.unitUom,
                adjustedPrice,
                element.priceListUnit || 0,
              ),
            ),
            old_price_list_unit: element.priceListUnit || 0,
```

4.3 `mapSaleOrderRequest` — เปลี่ยน `pass_price_list`

```ts
            // ผลตรวจราคาจริงจาก ValidateSaleOrder ไม่ใช่ค่าที่ยกมาจาก quotation
            pass_price_list: isPassPrice ? 'Y' : 'N',
```

4.4 `quotationCreate/index.vue` — ส่ง `isPassPrice` ต่อ

```ts
      onOk: () => onCreateSaleOrder(allPassed, isPassPrice),
```

และแก้ signature `const onCreateSaleOrder = async (isPass: boolean, isPassPrice: boolean) => {` แล้วส่งเข้า `createSaleOrderStore(allData.value, isPass, isPassPrice, adjustedPriceMap.value)`

- [ ] **Step 5: รันเทสและ type check**

```bash
cd /c/work-prime/wms-web && npx vitest run src/utils/helper src/stores/saleOrder && npx vue-tsc --noEmit 2>&1 | tail -3
```

Expected: vitest PASS, จำนวน error ของ vue-tsc ไม่เพิ่มจาก baseline (จดจำนวน baseline ไว้ก่อนแก้ด้วยการ `git stash` แล้วรันเทียบ ถ้าจำเป็น)

- [ ] **Step 6: Commit**

```bash
cd /c/work-prime/wms-web
git add src/utils/helper/quotationCalculation.ts src/stores/saleOrder/sale-order-create.store.ts src/components/quotation/quotationCreate/index.vue
git add <ไฟล์เทสที่สร้าง>
git commit -m "fix(sale-order): record the real price-check result and keep the quotation price when it passes"
```

---

## Task 4: wms-web อย่าทับราคาจาก quotation ตอนแก้ใบ

**Files:**
- Modify: `src/types/saleOrder/sale-order-list.type.ts` (เพิ่ม field ใกล้บรรทัด 111 ที่มี `priceListUnit`)
- Modify: `src/components/saleOrder/saleOrderCreate/index.vue` (จุด map แถวตอนโหลดหน้า ~บรรทัด 775-800)
- Modify: `src/stores/saleOrder/sale-order-create.store.ts` (`mapSaleOrderUpdateItem` บรรทัด ~160)

**Interfaces:**
- Consumes: `GetSale` ส่ง `old_price_list_unit` มาแล้ว (ยืนยันด้วยการยิง API จริง) — camelCase ที่ FE จะได้คือ `oldPriceListUnit`
- Produces: แถวในตารางหน้า SO มีฟิลด์ `oldPriceList`

- [ ] **Step 1: เพิ่ม field ใน type**

```ts
  oldPriceListUnit: number;
```

- [ ] **Step 2: map เข้าแถวตาราง** ใน `saleOrderCreate/index.vue` บล็อก `return { ... }` ที่มี `priceList: item.priceListUnit,`

```ts
    priceList: item.priceListUnit,
    oldPriceList: item.oldPriceListUnit ?? 0,
```

- [ ] **Step 3: ห้ามทับตอน update** — `mapSaleOrderUpdateItem` บรรทัด `old_price_list_unit: item.priceList,`

```ts
            // ราคาจาก quotation ห้ามถูกเขียนทับด้วยราคาปัจจุบันของแถว
            // ไม่งั้นกดแก้ใบครั้งเดียว ราคาที่เสนอลูกค้าหายถาวร
            old_price_list_unit: item.oldPriceList ?? 0,
```

(ถ้า type ของ `item` ในฟังก์ชันนี้ยังไม่มี `oldPriceList` ให้เพิ่มใน interface ของแถวตาราง — ตามหาไฟล์ที่ประกาศ `SaleOrderItem` ที่ฟังก์ชันนี้ใช้)

- [ ] **Step 4: type check**

```bash
cd /c/work-prime/wms-web && npx vue-tsc --noEmit 2>&1 | tail -3
```

Expected: จำนวน error ไม่เพิ่มจาก baseline

- [ ] **Step 5: Commit**

```bash
cd /c/work-prime/wms-web
git add src/types/saleOrder/sale-order-list.type.ts src/components/saleOrder/saleOrderCreate/index.vue src/stores/saleOrder/sale-order-create.store.ts
git commit -m "fix(sale-order): stop overwriting the quotation price list when a sale order is edited"
```

---

## Task 5: wms-web คอลัมน์ราคา 2 ช่วงเวลา

**Files:**
- Modify: `src/components/saleOrder/saleOrderCreate/saleOrderDetail/index.vue:635`
- Modify: `src/components/saleOrder/saleOrderCreate/index.vue:760-772` (บล็อกยิง `getPriceList` สด)
- Modify: `src/components/saleOrder/saleOrderCreate/saleOrderDetail/saleOrderColumns.ts:103-119` (ผูก dataIndex ใหม่)

**Interfaces:**
- Consumes: `saleOrderData.passPriceList` (มีใน type แล้วที่ `sale-order-list.type.ts:74`), `oldPriceList` จาก Task 4

- [ ] **Step 1: เปลี่ยนเงื่อนไขโชว์คอลัมน์** — `saleOrderDetail/index.vue:634-636`

```ts
// ใบที่ convert มาแบบราคาไม่ผ่าน ต้องเห็นราคา 2 ช่วงเวลาไปตลอดชีวิตใบ
// ถึงจะอนุมัติไปแล้วก็ตาม (SA เคาะ 2026-09-10) ของเดิมผูกกับ statusApprove
// ทำให้คอลัมน์หายทันทีที่กด approve
const column = computed(() =>
  buildColumns(props.saleOrderData?.passPriceList === 'N'),
);
```

- [ ] **Step 2: เปลี่ยนที่มาของคอลัมน์** — `saleOrderColumns.ts` ในบล็อก `isWaitingApproval`

```ts
        {
          title: 'Price list at create date (THB)',
          dataIndex: 'oldPriceList',
          key: 'oldPriceList',
          customHeaderCell,
          width: 150,
        },
        {
          title: 'Price list at current date (THB)',
          dataIndex: 'priceList',
          key: 'priceList',
          className: 'price-list-current',
          customHeaderCell,
          width: 150,
        },
```

และแก้ template ที่ `saleOrderDetail/index.vue:248-250` ให้ตรงกับ key ใหม่ (เดิมเป็น `priceListCurrentDate`)

- [ ] **Step 3: เลิกยิง getPriceList สด** — `saleOrderCreate/index.vue:760-772`
ลบบล็อก `let priceListCurrentDate = 0; if (findProduct) { ... }` และลบ `priceListCurrentDate: priceListCurrentDate,` ออกจาก object ที่ return
(ถ้า `getPriceList` ไม่มีผู้เรียกอื่นในไฟล์นี้แล้ว ให้ลบทิ้งพร้อม import ที่ค้าง — ตรวจด้วย grep ก่อนลบ)

- [ ] **Step 4: type check + เทส**

```bash
cd /c/work-prime/wms-web && npx vue-tsc --noEmit 2>&1 | tail -3 && npx vitest run src/utils/helper src/stores/saleOrder
```

Expected: error ไม่เพิ่มจาก baseline, vitest PASS

- [ ] **Step 5: Commit**

```bash
cd /c/work-prime/wms-web
git add src/components/saleOrder/saleOrderCreate/saleOrderDetail/index.vue src/components/saleOrder/saleOrderCreate/saleOrderDetail/saleOrderColumns.ts src/components/saleOrder/saleOrderCreate/index.vue
git commit -m "fix(sale-order): show the quotation price beside the converted price for price-flagged orders"
```

---

## Task 6: ตรวจงานรวม (ไม่แตะ UAT)

**Files:** ไม่มีการแก้โค้ด

- [ ] **Step 1: build + test ทั้งสอง repo**

```bash
cd /c/work-prime/erp-core && go build ./... && go test ./... 2>&1 | tail -5
cd /c/work-prime/wms-web && npx vue-tsc --noEmit 2>&1 | tail -3 && npx vitest run 2>&1 | tail -5
```

- [ ] **Step 2: ไล่ flow ด้วยตาแล้วบันทึกในรายงาน**

ต้องตอบได้ทั้ง 4 ข้อ โดยชี้ไฟล์+บรรทัด:
1. convert ราคาผ่าน → `price_list_unit` = ราคาจาก QU และ `pass_price_list = 'Y'`
2. convert ราคาไม่ผ่าน → `price_list_unit` = ราคาที่ดึงใหม่, `old_price_list_unit` = ราคาจาก QU, `pass_price_list = 'N'`
3. approve → `price_list_unit` ถูกเขียนกลับเป็น `old_price_list_unit`
4. ใบ `pass_price_list = 'N'` โชว์ 2 คอลัมน์ ทั้งก่อนและหลัง approve

- [ ] **Step 3: สรุปสิ่งที่ต้องทดสอบบน UAT ให้เจ้าของ (ห้ามทดสอบเอง)**

ต้องระบุลำดับ deploy (erp-core ก่อน แล้ว wms-web) และเคสที่ต้องกดจริง:
สร้าง QU ที่ราคาต่างจาก price list → convert → ดูใบเข้าสถานะรออนุมัติและโชว์ 2 คอลัมน์ → approve → ดูราคากลับเป็นของ QU และคอลัมน์ยังอยู่
