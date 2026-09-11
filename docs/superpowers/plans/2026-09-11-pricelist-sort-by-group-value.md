# Price List Detail — เรียงลำดับด้วย `group_item.value` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** ทำให้ทุกแกนของหน้า Price List Detail (แถว / หัวคอลัมน์ / ลำดับ tab) เรียงจากน้อยไปมากด้วยค่าตัวเลขใน `group_item.value` แทน lexicographic string sort

**Architecture:** เติมค่าตัวเลขจาก `group_item.value` ลง `PriceListSubGroupKeyResponse` ตอนประกอบ response แล้วเรียง `subGroups` ให้เสร็จตั้งแต่ต้นทางของทุก pattern handler จุดปลายน้ำที่เคย `sort.Strings` บน map key เปลี่ยนเป็นเก็บ key ตามลำดับที่เจอครั้งแรก ไม่มีการแปลงชื่อกลับเป็นตัวเลข

**Tech Stack:** Go 1.24, `testing` (stdlib), testcontainers (integration), GORM

**Spec:** `docs/superpowers/specs/2026-09-11-pricelist-sort-by-group-value-design.md`

**Branch:** `feat/pricelist-sort-by-group-value` (แตกจาก `origin/Develop` แล้ว)

---

## File Structure

| ไฟล์ | หน้าที่ | สถานะ |
|------|---------|-------|
| `internal/models/pricelist.go` | เพิ่ม `ValueNumber` / `HasValue` ใน `PriceListSubGroupKeyResponse` | แก้ |
| `internal/services/price-service/group_item_value.go` | `parseGroupItemValue` — อ่าน `group_item.value` แล้วแปลงเป็นตัวเลข | สร้าง |
| `internal/services/price-service/group_item_value_test.go` | unit test ของ `parseGroupItemValue` | สร้าง |
| `internal/services/price-service/get-price-detail.go` | เติมค่าตอนประกอบ `PriceListSubGroupKeyResponse` (บรรทัด 203) | แก้ |
| `internal/services/price-service/get-pricelist.go` | เติมค่าตอนประกอบ `PriceListSubGroupKeyResponse` (บรรทัด 828) | แก้ |
| `internal/services/price-service/patterns/sort_by_value.go` | helper เรียง subGroups + เก็บ key ตามลำดับ | สร้าง |
| `internal/services/price-service/patterns/sort_by_value_test.go` | unit test ของ helper | สร้าง |
| `internal/services/price-service/patterns/pattern_group_1_item_{1,3,4,5,6,7,8,9,10,11,12,13}.go` | เปลี่ยนจุด sort | แก้ |
| `internal/services/price-service/patterns/shared.go` | เปลี่ยนจุด sort 4 จุด | แก้ |
| `internal/services/price-service/patterns/sort_golden_test.go` | golden order test จากข้อมูลจริง UAT | สร้าง |
| `internal/services/price-service/pricelist_sort_integration_test.go` | integration test บน testcontainers | สร้าง |

`pattern_group_1_item_2.go` ไม่มีจุด sort จึงไม่ต้องแก้

---

## Task 1: `parseGroupItemValue` — แปลง `group_item.value` เป็นตัวเลข

**Files:**
- Create: `internal/services/price-service/group_item_value.go`
- Test: `internal/services/price-service/group_item_value_test.go`

บริบท: `group_item.value` เป็น `varchar` ที่เก็บตัวเลขพร้อม thousand separator เช่น `"10,000.00"` ส่วน `group_item.value_int` มีคอลัมน์อยู่แต่เป็น `0` ทั้งตาราง จึงห้ามใช้

- [ ] **Step 1: เขียน test ที่ต้องแดงก่อน**

สร้าง `internal/services/price-service/group_item_value_test.go`:

```go
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
		name     string
		code     string
		wantVal  float64
		wantHas  bool
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
```

- [ ] **Step 2: รัน test ให้เห็นว่าแดง**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
go test ./internal/services/price-service/ -run TestParseGroupItemValue -v
```

Expected: FAIL — `undefined: parseGroupItemValue`

- [ ] **Step 3: เขียน implementation ขั้นต่ำ**

สร้าง `internal/services/price-service/group_item_value.go`:

```go
package priceService

import (
	"strconv"
	"strings"

	"prime-erp-core/internal/models"
)

// parseGroupItemValue อ่าน group_item.value ของ item_code ที่ให้มาแล้วแปลงเป็นตัวเลข
//
// value เก็บเป็น varchar พร้อม thousand separator เช่น "10,000.00" จึงต้องตัด ","
// ก่อน ParseFloat ส่วน value_int มีคอลัมน์อยู่แต่เป็น 0 ทั้งตาราง (SyncGroupMaster
// ไม่ populate) จึงใช้แทนไม่ได้
//
// bool ที่คืนคือ "resolve ค่าได้ไหม" ไม่ใช่ "ไม่ใช่ศูนย์" — ต้องแยก value = "0"
// ที่เป็นค่าจริง (คืน (0, true)) ออกจากกรณีไม่มี record / value ว่าง / parse ไม่ได้
// (คืน (0, false)) เพราะ comparator ใช้ bool นี้ตัดสินว่าจะดันไปท้ายรายการหรือไม่
func parseGroupItemValue(m map[string]models.GetGroupItemResponse, code string) (float64, bool) {
	item, ok := m[code]
	if !ok {
		return 0, false
	}

	raw := strings.TrimSpace(strings.ReplaceAll(item.Value, ",", ""))
	if raw == "" {
		return 0, false
	}

	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}

	return v, true
}
```

- [ ] **Step 4: รัน test ให้ผ่าน**

```bash
go test ./internal/services/price-service/ -run TestParseGroupItemValue -v
```

Expected: PASS ทุก subtest

- [ ] **Step 5: Commit**

```bash
git add internal/services/price-service/group_item_value.go internal/services/price-service/group_item_value_test.go
git commit -m "feat: เพิ่ม parseGroupItemValue อ่านค่าตัวเลขจาก group_item.value

value เก็บเป็น varchar มี thousand separator ต้องตัด \",\" ก่อน ParseFloat
และต้องแยก value = \"0\" ที่เป็นค่าจริง ออกจากกรณี resolve ไม่ได้

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 2: เพิ่ม `ValueNumber` / `HasValue` ลง `PriceListSubGroupKeyResponse`

**Files:**
- Modify: `internal/models/pricelist.go:274-282`

- [ ] **Step 1: แก้ struct**

เดิม (บรรทัด 274-282):

```go
type PriceListSubGroupKeyResponse struct {
	ID         string `json:"id"`
	SubGroupID string `json:"sub_group_id"`
	GroupCode  string `json:"group_code"`
	GroupName  string `json:"group_name"`
	ValueCode  string `json:"value_code"`
	ValueName  string `json:"value_name"`
	Seq        int    `json:"seq"`
}
```

ใหม่:

```go
type PriceListSubGroupKeyResponse struct {
	ID         string `json:"id"`
	SubGroupID string `json:"sub_group_id"`
	GroupCode  string `json:"group_code"`
	GroupName  string `json:"group_name"`
	ValueCode  string `json:"value_code"`
	ValueName  string `json:"value_name"`
	Seq        int    `json:"seq"`

	// ValueNumber คือ group_item.value ของ ValueCode แปลงเป็นตัวเลข ใช้เป็นลำดับ
	// การแสดงผลของทุกแกนในหน้า Price List Detail
	//
	// Seq ด้านบนใช้แทนไม่ได้ — upload-pricelist.go:1476 กำหนด Seq = i + 1 ซึ่งเป็น
	// ลำดับของ PG0x ในคีย์ (PG01 -> 1, PG05 -> 5) ไม่ใช่ลำดับของค่า
	ValueNumber float64 `json:"value_number"`

	// HasValue แยก "ValueNumber = 0 จริง" ออกจาก "resolve ค่าไม่ได้"
	// ไม่ส่งออก JSON เพราะเป็นข้อมูลภายในสำหรับ comparator เท่านั้น
	HasValue bool `json:"-"`
}
```

- [ ] **Step 2: ยืนยันว่า build ผ่าน**

```bash
go build ./...
```

Expected: ไม่มี output (สำเร็จ)

- [ ] **Step 3: Commit**

```bash
git add internal/models/pricelist.go
git commit -m "feat: เพิ่ม ValueNumber/HasValue ใน PriceListSubGroupKeyResponse

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 3: เติมค่าใน `get-price-detail.go`

**Files:**
- Modify: `internal/services/price-service/get-price-detail.go:203-211`

บริบท: `transformToGetPriceListResponse` มี `groupItemMap` อยู่ในมือแล้ว (มาจาก `getGroupAndItemMappings()` บรรทัด 140) จึงไม่ต้องยิง query เพิ่ม

- [ ] **Step 1: แก้จุดประกอบ key**

เดิม (บรรทัด 199-212):

```go
			subGroupKeys := []models.PriceListSubGroupKeyResponse{}
			for _, sgk := range sg.GroupKeys {
				itemName := resolveGroupItemName(groupItemMap, sgk.Value)

				subGroupKeys = append(subGroupKeys, models.PriceListSubGroupKeyResponse{
					ID:         uuid.New().String(),
					SubGroupID: sg.ID.String(),
					GroupCode:  sgk.Code,
					GroupName:  groupMap[sgk.Code].GroupName,
					ValueCode:  sgk.Value,
					ValueName:  itemName,
					Seq:        sgk.Seq,
				})
			}
```

ใหม่:

```go
			subGroupKeys := []models.PriceListSubGroupKeyResponse{}
			for _, sgk := range sg.GroupKeys {
				itemName := resolveGroupItemName(groupItemMap, sgk.Value)
				valueNumber, hasValue := parseGroupItemValue(groupItemMap, sgk.Value)

				subGroupKeys = append(subGroupKeys, models.PriceListSubGroupKeyResponse{
					ID:          uuid.New().String(),
					SubGroupID:  sg.ID.String(),
					GroupCode:   sgk.Code,
					GroupName:   groupMap[sgk.Code].GroupName,
					ValueCode:   sgk.Value,
					ValueName:   itemName,
					Seq:         sgk.Seq,
					ValueNumber: valueNumber,
					HasValue:    hasValue,
				})
			}
```

- [ ] **Step 2: ยืนยันว่า build และ test เดิมยังผ่าน**

```bash
go build ./... && go test ./internal/services/price-service/... 2>&1 | tail -20
```

Expected: build ผ่าน และ test เดิมทั้งหมด `ok` / `PASS`

- [ ] **Step 3: Commit**

```bash
git add internal/services/price-service/get-price-detail.go
git commit -m "feat: เติม ValueNumber ตอนประกอบ SubGroupKey ใน GetPriceDetail

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 4: เติมค่าใน `get-pricelist.go`

**Files:**
- Modify: `internal/services/price-service/get-pricelist.go:828-836`

บริบท: `GetPriceList` ประกอบ `PriceListSubGroupKeyResponse` อีกทางหนึ่ง ต้องเติมให้ตรงกัน ไม่งั้นเส้นทางนี้จะป้อนข้อมูลที่ `HasValue = false` ทั้งหมดให้ pattern แล้วเรียงตกกลับไปเป็น string

- [ ] **Step 1: แก้จุดประกอบ key**

เดิม (บรรทัด 828-836):

```go
						subGroupKeys = append(subGroupKeys, models.PriceListSubGroupKeyResponse{
							ID:         sgk.ID.String(),
							SubGroupID: sgk.SubGroupID.String(),
							GroupCode:  sgk.Code,
							GroupName:  groupMap[sgk.Code].GroupName,
							ValueCode:  sgk.Value,
							ValueName:  groupItemMap[sgk.Value].ItemName,
							Seq:        sgk.Seq,
						})
```

ใหม่:

```go
						valueNumber, hasValue := parseGroupItemValue(groupItemMap, sgk.Value)
						subGroupKeys = append(subGroupKeys, models.PriceListSubGroupKeyResponse{
							ID:          sgk.ID.String(),
							SubGroupID:  sgk.SubGroupID.String(),
							GroupCode:   sgk.Code,
							GroupName:   groupMap[sgk.Code].GroupName,
							ValueCode:   sgk.Value,
							ValueName:   groupItemMap[sgk.Value].ItemName,
							Seq:         sgk.Seq,
							ValueNumber: valueNumber,
							HasValue:    hasValue,
						})
```

- [ ] **Step 2: ยืนยันว่า build และ test เดิมยังผ่าน**

```bash
go build ./... && go test ./internal/services/price-service/... 2>&1 | tail -20
```

Expected: build ผ่าน และ test เดิมทั้งหมดผ่าน

- [ ] **Step 3: Commit**

```bash
git add internal/services/price-service/get-pricelist.go
git commit -m "feat: เติม ValueNumber ตอนประกอบ SubGroupKey ใน GetPriceList

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 5: helper `sort_by_value.go`

**Files:**
- Create: `internal/services/price-service/patterns/sort_by_value.go`
- Test: `internal/services/price-service/patterns/sort_by_value_test.go`

- [ ] **Step 1: เขียน test ที่ต้องแดงก่อน**

สร้าง `internal/services/price-service/patterns/sort_by_value_test.go`:

```go
package patterns

import (
	"testing"

	"prime-erp-core/internal/models"
)

// sgk ย่อการสร้าง SubGroupKey ในเทสต์
func sgk(groupCode, valueName string, value float64, has bool) models.PriceListSubGroupKeyResponse {
	return models.PriceListSubGroupKeyResponse{
		GroupCode:   groupCode,
		ValueCode:   groupCode + "_" + valueName,
		ValueName:   valueName,
		ValueNumber: value,
		HasValue:    has,
	}
}

func sub(keys ...models.PriceListSubGroupKeyResponse) models.PriceListSubGroupResponse {
	return models.PriceListSubGroupResponse{SubGroupKeys: keys}
}

func TestValueOfGroup(t *testing.T) {
	keys := []models.PriceListSubGroupKeyResponse{
		sgk("PG06", "1.2", 1.2, true),
		sgk("PG05", "4' x 8'", 32, true),
	}

	if v, ok := valueOfGroup(keys, "PG06"); v != 1.2 || !ok {
		t.Fatalf("PG06 ต้องได้ (1.2, true) แต่ได้ (%v, %v)", v, ok)
	}
	if v, ok := valueOfGroup(keys, "PG99"); v != 0 || ok {
		t.Fatalf("กลุ่มที่ไม่มี ต้องได้ (0, false) แต่ได้ (%v, %v)", v, ok)
	}
}

func TestCmpGroupValue(t *testing.T) {
	cases := []struct {
		name string
		a    models.PriceListSubGroupKeyResponse
		b    models.PriceListSubGroupKeyResponse
		want int
	}{
		{"ตัวเลขน้อยกว่ามาก่อน", sgk("PG06", "9", 9, true), sgk("PG06", "10", 10, true), -1},
		{"string sort เคยให้ 100 มาก่อน 12 ต้องกลับด้าน", sgk("PG06", "100", 100, true), sgk("PG06", "12", 12, true), 1},
		{"ค่าเท่ากัน tie-break ด้วยชื่อ", sgk("PG05", "100x300", 30000, true), sgk("PG05", "150x200", 30000, true), -1},
		{"ฝ่ายที่มีค่ามาก่อนฝ่ายที่ resolve ไม่ได้", sgk("PG05", "ESP", 0, false), sgk("PG05", "4' x 8'", 32, true), 1},
		{"ไม่มีค่าทั้งคู่ ใช้ string compare", sgk("PG05", "AAA", 0, false), sgk("PG05", "BBB", 0, false), -1},
		{"value = 0 จริง ถือว่ามีค่า มาก่อนตัวที่ resolve ไม่ได้", sgk("PG08", "ศูนย์", 0, true), sgk("PG08", "ไม่รู้", 0, false), -1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := cmpGroupValue([]models.PriceListSubGroupKeyResponse{c.a}, []models.PriceListSubGroupKeyResponse{c.b}, c.a.GroupCode)
			if got != c.want {
				t.Fatalf("cmpGroupValue = %d ต้องเป็น %d", got, c.want)
			}
		})
	}
}

func TestCmpSubGroupKeys_StopsAtFirstDifference(t *testing.T) {
	a := []models.PriceListSubGroupKeyResponse{sgk("PG06", "10", 10, true), sgk("PG05", "5' x 20'", 100, true)}
	b := []models.PriceListSubGroupKeyResponse{sgk("PG06", "12", 12, true), sgk("PG05", "4' x 8'", 32, true)}

	if got := cmpSubGroupKeys(a, b, "PG06", "PG05"); got != -1 {
		t.Fatalf("PG06 ต่างกันแล้วต้องจบที่ตัวแรก ได้ %d ต้องเป็น -1", got)
	}

	c := []models.PriceListSubGroupKeyResponse{sgk("PG06", "10", 10, true), sgk("PG05", "5' x 20'", 100, true)}
	d := []models.PriceListSubGroupKeyResponse{sgk("PG06", "10", 10, true), sgk("PG05", "4' x 8'", 32, true)}

	if got := cmpSubGroupKeys(c, d, "PG06", "PG05"); got != 1 {
		t.Fatalf("PG06 เท่ากันต้องไปเทียบ PG05 ต่อ ได้ %d ต้องเป็น 1", got)
	}

	if got := cmpSubGroupKeys(c, c, "PG06", "PG05"); got != 0 {
		t.Fatalf("เหมือนกันทุกแกนต้องได้ 0 ได้ %d", got)
	}
}

func TestSortSubGroupsByValue(t *testing.T) {
	// ลำดับตั้งต้นคือผลของ lexicographic sort ที่เป็นบั๊กอยู่ตอนนี้
	subs := []models.PriceListSubGroupResponse{
		sub(sgk("PG06", "1.2", 1.2, true)),
		sub(sgk("PG06", "10", 10, true)),
		sub(sgk("PG06", "100", 100, true)),
		sub(sgk("PG06", "12", 12, true)),
		sub(sgk("PG06", "15", 15, true)),
		sub(sgk("PG06", "2.3", 2.3, true)),
	}

	SortSubGroupsByValue(subs, "PG06")

	want := []string{"1.2", "2.3", "10", "12", "15", "100"}
	for i, w := range want {
		got := subs[i].SubGroupKeys[0].ValueName
		if got != w {
			t.Fatalf("ตำแหน่ง %d ได้ %q ต้องเป็น %q — ลำดับทั้งหมด %v", i, got, w, valueNames(subs))
		}
	}
}

func TestSortSubGroupsByValue_IsStableAndDeterministic(t *testing.T) {
	build := func() []models.PriceListSubGroupResponse {
		return []models.PriceListSubGroupResponse{
			sub(sgk("PG05", "150x200", 30000, true)),
			sub(sgk("PG05", "100x300", 30000, true)),
			sub(sgk("PG05", "75x75", 5625, true)),
		}
	}

	var first []string
	for round := 0; round < 50; round++ {
		subs := build()
		SortSubGroupsByValue(subs, "PG05")
		got := valueNames(subs)
		if round == 0 {
			first = got
			continue
		}
		for i := range got {
			if got[i] != first[i] {
				t.Fatalf("รอบที่ %d ได้ %v ต่างจากรอบแรก %v — ลำดับไม่นิ่ง", round, got, first)
			}
		}
	}

	want := []string{"75x75", "100x300", "150x200"}
	for i, w := range want {
		if first[i] != w {
			t.Fatalf("ตำแหน่ง %d ได้ %q ต้องเป็น %q", i, first[i], w)
		}
	}
}

func valueNames(subs []models.PriceListSubGroupResponse) []string {
	out := make([]string, 0, len(subs))
	for _, s := range subs {
		if len(s.SubGroupKeys) > 0 {
			out = append(out, s.SubGroupKeys[0].ValueName)
		}
	}
	return out
}

func TestOrderedUnique(t *testing.T) {
	rows := []AGGridRowData{
		{"row_group_value": "1.2"},
		{"row_group_value": "1.2"},
		{"row_group_value": "10"},
		{"row_group_value": ""},
		{"row_group_value": "100"},
		{"row_group_value": "10"},
	}

	got := orderedUnique(rows, "row_group_value")
	want := []string{"1.2", "10", "100"}

	if len(got) != len(want) {
		t.Fatalf("ได้ %v ต้องเป็น %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ตำแหน่ง %d ได้ %q ต้องเป็น %q — ทั้งหมด %v", i, got[i], want[i], got)
		}
	}
}

func TestOrderedUniqueBy(t *testing.T) {
	subs := []models.PriceListSubGroupResponse{
		sub(sgk("PG02", "เหล็กแผ่น", 6, true)),
		sub(sgk("PG02", "เหล็กแผ่น", 6, true)),
		sub(sgk("PG02", "แผ่นลาย", 9, true)),
	}

	got := orderedUniqueBy(subs, func(s models.PriceListSubGroupResponse) string {
		return s.SubGroupKeys[0].ValueName
	})
	want := []string{"เหล็กแผ่น", "แผ่นลาย"}

	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("ได้ %v ต้องเป็น %v", got, want)
	}
}
```

- [ ] **Step 2: รัน test ให้เห็นว่าแดง**

```bash
go test ./internal/services/price-service/patterns/ -run 'TestValueOfGroup|TestCmpGroupValue|TestCmpSubGroupKeys|TestSortSubGroupsByValue|TestOrderedUnique' -v
```

Expected: FAIL — `undefined: valueOfGroup` และเพื่อน ๆ

- [ ] **Step 3: เขียน implementation**

สร้าง `internal/services/price-service/patterns/sort_by_value.go`:

```go
package patterns

import (
	"fmt"
	"sort"
	"strings"

	"prime-erp-core/internal/models"
)

// valueOfGroup คืนค่าตัวเลขของ groupCode ใน subgroup นี้
// bool ตัวที่สองคือ "resolve ค่าได้ไหม" ไม่ใช่ "ไม่ใช่ศูนย์"
func valueOfGroup(sgks []models.PriceListSubGroupKeyResponse, groupCode string) (float64, bool) {
	for _, k := range sgks {
		if k.GroupCode == groupCode {
			return k.ValueNumber, k.HasValue
		}
	}
	return 0, false
}

// nameOfGroup คืนชื่อที่ผู้ใช้เห็นของ groupCode ใช้เป็น tie-break ให้ลำดับนิ่ง
func nameOfGroup(sgks []models.PriceListSubGroupKeyResponse, groupCode string) string {
	for _, k := range sgks {
		if k.GroupCode == groupCode {
			return k.ValueName
		}
	}
	return ""
}

// cmpGroupValue เทียบ subgroup สองตัวที่ groupCode เดียว คืน -1 / 0 / 1
//
// กติกา:
//  1. มีค่าทั้งคู่และไม่เท่ากัน -> เทียบตัวเลขจากน้อยไปมาก
//  2. มีค่าฝ่ายเดียว -> ฝ่ายที่มีค่ามาก่อน ตัวที่ resolve ไม่ได้ไปท้ายเสมอ
//     ห้ามปล่อยให้ค่า 0 ของตัวที่ resolve ไม่ได้ไปกองอยู่หน้าสุด
//  3. ไม่มีค่าทั้งคู่ -> เทียบ ValueName แบบ string ตามพฤติกรรมเดิม
//  4. ค่าเท่ากัน -> tie-break ด้วย ValueName เพื่อให้ลำดับนิ่ง เช่น 100x300 กับ
//     150x200 ที่ value = 30,000.00 เท่ากัน
func cmpGroupValue(a, b []models.PriceListSubGroupKeyResponse, groupCode string) int {
	va, hasA := valueOfGroup(a, groupCode)
	vb, hasB := valueOfGroup(b, groupCode)

	switch {
	case hasA && hasB:
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
	case hasA && !hasB:
		return -1
	case !hasA && hasB:
		return 1
	}

	return strings.Compare(nameOfGroup(a, groupCode), nameOfGroup(b, groupCode))
}

// cmpSubGroupKeys ไล่ groupCodes จากซ้ายไปขวา เจอตัวแรกที่ไม่เท่ากันแล้วจบ
func cmpSubGroupKeys(a, b []models.PriceListSubGroupKeyResponse, groupCodes ...string) int {
	for _, code := range groupCodes {
		if code == "" {
			continue
		}
		if c := cmpGroupValue(a, b, code); c != 0 {
			return c
		}
	}
	return 0
}

// SortSubGroupsByValue เรียง subGroups in-place ตามค่าของ groupCodes ที่ให้มา
//
// ต้องใช้ sort.SliceStable เท่านั้น ห้าม sort.Slice — shared.go บันทึกไว้ว่า
// ลำดับ relative ของ record ที่มี sg.ID เดียวกันต้องคงเดิม ไม่งั้น "record แรก"
// จะไม่ใช่ inventoryWeights[0] อีกต่อไป
func SortSubGroupsByValue(sgs []models.PriceListSubGroupResponse, groupCodes ...string) {
	sort.SliceStable(sgs, func(i, j int) bool {
		return cmpSubGroupKeys(sgs[i].SubGroupKeys, sgs[j].SubGroupKeys, groupCodes...) < 0
	})
}

// orderedUnique เก็บค่าของ field จาก rows ตามลำดับที่เจอครั้งแรก ไม่ซ้ำ และข้ามค่าว่าง
//
// ใช้แทน sort.Strings(mapKeys) ได้เมื่อ rows ถูกสร้างจาก subGroups ที่เรียงแล้ว
// การเรียง key ที่ประกอบเสร็จแล้วทำไม่ได้ เพราะ buildCompositeKeyBy ข้ามค่าว่าง
// ตอน join และ columnKey ยังผ่าน sanitizeIdentifier มาอีกชั้น จึงแยกกลับไม่ได้
func orderedUnique(rows []AGGridRowData, field string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(rows))

	for _, row := range rows {
		v, ok := row[field]
		if !ok {
			continue
		}
		key := strings.TrimSpace(fmt.Sprintf("%v", v))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}

	return out
}

// orderedUniqueBy เวอร์ชันที่ดึง key ด้วยฟังก์ชัน ใช้กับ subGroups โดยตรง
func orderedUniqueBy[T any](items []T, keyOf func(T) string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(items))

	for _, item := range items {
		key := strings.TrimSpace(keyOf(item))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}

	return out
}
```

- [ ] **Step 4: รัน test ให้ผ่าน**

```bash
go test ./internal/services/price-service/patterns/ -run 'TestValueOfGroup|TestCmpGroupValue|TestCmpSubGroupKeys|TestSortSubGroupsByValue|TestOrderedUnique' -v
```

Expected: PASS ทุก test

- [ ] **Step 5: Commit**

```bash
git add internal/services/price-service/patterns/sort_by_value.go internal/services/price-service/patterns/sort_by_value_test.go
git commit -m "feat: เพิ่ม helper เรียง subGroups ด้วย group_item.value

เรียงที่ต้นทางจาก SubGroupKeys ที่มี ValueNumber แทนการเรียง key ที่ประกอบ
เสร็จแล้ว ซึ่ง reverse-map ไม่ได้เพราะ buildCompositeKeyBy ข้ามค่าว่างตอน join
และ columnKey ผ่าน sanitizeIdentifier มาอีกชั้น

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 6: helper เพิ่มเติม — `splitGroupCodes` และ `newValueByCode`

**Files:**
- Modify: `internal/services/price-service/patterns/sort_by_value.go`
- Test: `internal/services/price-service/patterns/sort_by_value_test.go`

บริบท: แกนของแต่ละ pattern อ่านจาก `pattern.Grouping.Rows` / `.ColumnGroups` / `.Tabs` ซึ่งเป็น string เช่น `"PG03|PG08|PG05"` ส่วน `buildColumnGroupsRecursive` มองเห็นแค่ hierarchy key ที่ `composeHierarchyKey(code, label)` ประกอบไว้ — `splitHierarchyKey` แยก `code` (item_code เช่น `PG05_22`) กลับมาได้ตรง ๆ ไม่ผ่าน sanitize จึง lookup ด้วย code ได้อย่างปลอดภัย

- [ ] **Step 1: เขียน test ที่ต้องแดงก่อน**

เพิ่มท้าย `internal/services/price-service/patterns/sort_by_value_test.go`:

```go
func TestSplitGroupCodes(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"PG03|PG08|PG05", []string{"PG03", "PG08", "PG05"}},
		{" PG06 | PG05 ", []string{"PG06", "PG05"}},
		{"PG06", []string{"PG06"}},
		{"", nil},
		{"||", nil},
	}

	for _, c := range cases {
		got := splitGroupCodes(c.in)
		if len(got) != len(c.want) {
			t.Fatalf("splitGroupCodes(%q) = %v ต้องเป็น %v", c.in, got, c.want)
		}
		for i := range c.want {
			if got[i] != c.want[i] {
				t.Fatalf("splitGroupCodes(%q)[%d] = %q ต้องเป็น %q", c.in, i, got[i], c.want[i])
			}
		}
	}
}

func TestNewValueByCode(t *testing.T) {
	subs := []models.PriceListSubGroupResponse{
		sub(sgk("PG05", "4' x 8'", 32, true), sgk("PG06", "1.2", 1.2, true)),
		sub(sgk("PG05", "1250x8'", 10000, true)),
		sub(models.PriceListSubGroupKeyResponse{GroupCode: "PG05", ValueCode: "PG05_40", ValueName: "ESP"}),
	}

	idx := newValueByCode(subs)

	if v, ok := idx["PG05_4' x 8'"]; !ok || v.value != 32 || !v.has {
		t.Fatalf("PG05_4' x 8' ต้องได้ (32, true) แต่ได้ %+v ok=%v", v, ok)
	}
	if v, ok := idx["PG05_1250x8'"]; !ok || v.value != 10000 {
		t.Fatalf("PG05_1250x8' ต้องได้ 10000 แต่ได้ %+v ok=%v", v, ok)
	}
	if v, ok := idx["PG05_40"]; !ok || v.has {
		t.Fatalf("PG05_40 ต้อง resolve ไม่ได้ แต่ได้ %+v ok=%v", v, ok)
	}
	if _, ok := idx["ไม่มีจริง"]; ok {
		t.Fatal("code ที่ไม่มีต้องไม่อยู่ใน index")
	}
}

func TestValueByCodeLess(t *testing.T) {
	idx := valueByCode{
		"PG05_22": {value: 10000, has: true},
		"PG05_3":  {value: 32, has: true},
		"PG05_40": {value: 0, has: false},
	}

	if !idx.Less("PG05_3", "4' x 8'", "PG05_22", "1250x8'") {
		t.Fatal("32 ต้องมาก่อน 10000")
	}
	if idx.Less("PG05_22", "1250x8'", "PG05_3", "4' x 8'") {
		t.Fatal("10000 ต้องไม่มาก่อน 32")
	}
	if !idx.Less("PG05_3", "4' x 8'", "PG05_40", "ESP") {
		t.Fatal("ตัวที่มีค่าต้องมาก่อนตัวที่ resolve ไม่ได้")
	}
	if !idx.Less("missing", "AAA", "missing2", "BBB") {
		t.Fatal("ไม่มีค่าทั้งคู่ ต้อง fallback เป็น string compare ของ label")
	}
}
```

- [ ] **Step 2: รัน test ให้เห็นว่าแดง**

```bash
go test ./internal/services/price-service/patterns/ -run 'TestSplitGroupCodes|TestNewValueByCode|TestValueByCodeLess' -v
```

Expected: FAIL — `undefined: splitGroupCodes`, `undefined: newValueByCode`, `undefined: valueByCode`

- [ ] **Step 3: เขียน implementation**

เพิ่มท้าย `internal/services/price-service/patterns/sort_by_value.go`:

```go
// splitGroupCodes แยกสตริงแกนของ pattern เช่น "PG03|PG08|PG05" เป็น slice
// ตัดช่องว่างและข้ามค่าว่าง คืน nil เมื่อไม่เหลืออะไร
func splitGroupCodes(s string) []string {
	parts := strings.Split(s, "|")
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

// codeValue คือค่าตัวเลขของ item หนึ่งตัว พร้อมธงว่า resolve ได้ไหม
type codeValue struct {
	value float64
	has   bool
}

// valueByCode คือ index จาก item_code ไปหาค่าตัวเลข
type valueByCode map[string]codeValue

// newValueByCode สร้าง index จาก SubGroupKeys ทั้งหมดใน subGroups
//
// key คือ ValueCode (item_code) ซึ่งไม่ผ่าน sanitizeIdentifier และไม่ถูก join
// เป็น composite จึง lookup กลับได้อย่างปลอดภัย ต่างจาก columnKey / row_group_value
// ที่ประกอบมาแล้วและแยกกลับไม่ได้
func newValueByCode(sgs []models.PriceListSubGroupResponse) valueByCode {
	idx := valueByCode{}

	for _, sg := range sgs {
		for _, k := range sg.SubGroupKeys {
			if k.ValueCode == "" {
				continue
			}
			if existing, ok := idx[k.ValueCode]; ok && existing.has {
				continue
			}
			idx[k.ValueCode] = codeValue{value: k.ValueNumber, has: k.HasValue}
		}
	}

	return idx
}

// Less เทียบ item สองตัวด้วย code เป็นหลัก ถ้า resolve ไม่ได้ทั้งคู่จึง fallback
// ไปเทียบ label แบบ string ตามพฤติกรรมเดิม
func (idx valueByCode) Less(codeA, labelA, codeB, labelB string) bool {
	a, hasA := idx[codeA]
	b, hasB := idx[codeB]

	resolvedA := hasA && a.has
	resolvedB := hasB && b.has

	switch {
	case resolvedA && resolvedB:
		if a.value != b.value {
			return a.value < b.value
		}
	case resolvedA && !resolvedB:
		return true
	case !resolvedA && resolvedB:
		return false
	}

	return labelA < labelB
}
```

- [ ] **Step 4: รัน test ให้ผ่าน**

```bash
go test ./internal/services/price-service/patterns/ -run 'TestSplitGroupCodes|TestNewValueByCode|TestValueByCodeLess' -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/services/price-service/patterns/sort_by_value.go internal/services/price-service/patterns/sort_by_value_test.go
git commit -m "feat: เพิ่ม splitGroupCodes และ valueByCode index

valueByCode ใช้กับ buildColumnGroupsRecursive ที่มองเห็นแค่ hierarchy key
ซึ่ง splitHierarchyKey แยก item_code กลับมาได้ตรง ๆ จึง lookup ปลอดภัย

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 7: `shared.go` — 4 จุด

**Files:**
- Modify: `internal/services/price-service/patterns/shared.go:875-905` (`buildSingleLevelColumns`)
- Modify: `internal/services/price-service/patterns/shared.go:998-1016` (`buildColumnGroupsRecursive`)
- Modify: `internal/services/price-service/patterns/shared.go:1083-1086` (caller ของ recursive)
- Modify: `internal/services/price-service/patterns/shared.go:2211-2245` (`buildProductGroup2ColumnGroupsWithCode`)
- Modify: `internal/services/price-service/patterns/shared.go:2283-2315` (`buildDirectRowsWithProductGroup2WithCode`)

- [ ] **Step 1: แก้ `buildSingleLevelColumns` (บรรทัด ~898-905)**

เดิม:

```go
	// Sort keys to ensure consistent column order by label
	sortedKeys := make([]string, 0, len(uniqueValues))
	for key := range uniqueValues {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return uniqueValues[sortedKeys[i]].Label < uniqueValues[sortedKeys[j]].Label
	})
```

ใหม่:

```go
	// เรียงคอลัมน์ด้วยค่าตัวเลขจาก group_item.value ไม่ใช่ label
	// เทียบ label แบบ string จะได้ "1250x8'" < "4' x 8'" < "4'x1500" ซึ่งผิด
	idx := newValueByCode(subGroups)
	sortedKeys := make([]string, 0, len(uniqueValues))
	for key := range uniqueValues {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		a, b := uniqueValues[sortedKeys[i]], uniqueValues[sortedKeys[j]]
		return idx.Less(a.Code, a.Label, b.Code, b.Label)
	})
```

- [ ] **Step 2: แก้ `buildColumnGroupsRecursive` ให้รับ index**

เดิม (บรรทัด 997-1012):

```go
// buildColumnGroupsRecursive recursively builds ColumnDef structures from hierarchy
func buildColumnGroupsRecursive(
	hierarchy map[string]interface{},
	pattern *PatternConfig,
	levelIndex int,
	labelPath []string,
	codePath []string,
) []ColumnDef {
	columns := []ColumnDef{}

	// Get sorted keys for current level
	keys := make([]string, 0, len(hierarchy))
	for key := range hierarchy {
		keys = append(keys, key)
	}
	sort.Strings(keys)
```

ใหม่:

```go
// buildColumnGroupsRecursive recursively builds ColumnDef structures from hierarchy
//
// idx ใช้เรียงคอลัมน์ด้วยค่าตัวเลขจาก group_item.value — hierarchy key ถูกประกอบ
// ด้วย composeHierarchyKey(code, label) จึง splitHierarchyKey แยก item_code
// กลับมา lookup ได้ตรง ๆ
func buildColumnGroupsRecursive(
	hierarchy map[string]interface{},
	pattern *PatternConfig,
	levelIndex int,
	labelPath []string,
	codePath []string,
	idx valueByCode,
) []ColumnDef {
	columns := []ColumnDef{}

	keys := make([]string, 0, len(hierarchy))
	for key := range hierarchy {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		codeI, labelI := splitHierarchyKey(keys[i])
		codeJ, labelJ := splitHierarchyKey(keys[j])
		return idx.Less(codeI, labelI, codeJ, labelJ)
	})
```

- [ ] **Step 3: แก้ call site ทั้งสองจุด**

บรรทัด ~1067 เดิม:

```go
					Children:      buildColumnGroupsRecursive(nestedMap, pattern, levelIndex+1, currentLabelPath, currentCodePath),
```

ใหม่:

```go
					Children:      buildColumnGroupsRecursive(nestedMap, pattern, levelIndex+1, currentLabelPath, currentCodePath, idx),
```

บรรทัด ~1083-1086 เดิม:

```go
	hierarchy := buildHierarchyMap(subGroups, pattern.ColumnLevels)

	columns := buildColumnGroupsRecursive(hierarchy, pattern, 0, []string{}, []string{})
```

ใหม่:

```go
	hierarchy := buildHierarchyMap(subGroups, pattern.ColumnLevels)

	columns := buildColumnGroupsRecursive(hierarchy, pattern, 0, []string{}, []string{}, newValueByCode(subGroups))
```

- [ ] **Step 4: แก้ `buildProductGroup2ColumnGroupsWithCode` (บรรทัด ~2234-2241)**

เดิม:

```go
	// Sort keys to ensure consistent column order by label
	sortedKeys := make([]string, 0, len(uniqueValues))
	for key := range uniqueValues {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return uniqueValues[sortedKeys[i]].Label < uniqueValues[sortedKeys[j]].Label
	})
```

ใหม่:

```go
	// เรียงคอลัมน์ด้วยค่าตัวเลขจาก group_item.value ไม่ใช่ label
	idx := newValueByCode(subGroups)
	sortedKeys := make([]string, 0, len(uniqueValues))
	for key := range uniqueValues {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		a, b := uniqueValues[sortedKeys[i]], uniqueValues[sortedKeys[j]]
		return idx.Less(a.Code, a.Label, b.Code, b.Label)
	})
```

- [ ] **Step 5: แก้ `buildDirectRowsWithProductGroup2WithCode` (บรรทัด ~2306-2312)**

เดิม:

```go
	pg2Entries := make([]pg2Entry, 0, len(pg2Map))
	for code, label := range pg2Map {
		pg2Entries = append(pg2Entries, pg2Entry{Code: code, Label: label})
	}
	sort.Slice(pg2Entries, func(i, j int) bool {
		return pg2Entries[i].Label < pg2Entries[j].Label
	})
```

ใหม่:

```go
	idxPG2 := newValueByCode(subGroups)
	pg2Entries := make([]pg2Entry, 0, len(pg2Map))
	for code, label := range pg2Map {
		pg2Entries = append(pg2Entries, pg2Entry{Code: code, Label: label})
	}
	sort.Slice(pg2Entries, func(i, j int) bool {
		a, b := pg2Entries[i], pg2Entries[j]
		return idxPG2.Less(a.Code, a.Label, b.Code, b.Label)
	})
```

- [ ] **Step 6: ยืนยันว่า build และ test เดิมผ่าน**

```bash
go build ./... && go test ./internal/services/price-service/... 2>&1 | tail -20
```

Expected: build ผ่าน test ผ่าน

- [ ] **Step 7: Commit**

```bash
git add internal/services/price-service/patterns/shared.go
git commit -m "fix: เรียงคอลัมน์ใน shared.go ด้วย group_item.value แทน label

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 8: helper `sortLabelsByValue` สำหรับลำดับ tab

**Files:**
- Modify: `internal/services/price-service/patterns/sort_by_value.go`
- Test: `internal/services/price-service/patterns/sort_by_value_test.go`

บริบท: ลำดับ tab เรียงจาก **label** (เช่น `"เหล็กแผ่น"`, `"แผ่นลาย"`) ไม่ใช่ code จึงต้องมี index ที่ key ด้วย `ValueName` — ปลอดภัยเพราะจำกัดอยู่ใน groupCode เดียว ชื่อ item ภายในกลุ่มเดียวกันไม่ซ้ำ

- [ ] **Step 1: เขียน test ที่ต้องแดงก่อน**

เพิ่มท้าย `internal/services/price-service/patterns/sort_by_value_test.go`:

```go
func TestSortLabelsByValue(t *testing.T) {
	subs := []models.PriceListSubGroupResponse{
		sub(sgk("PG02", "เหล็กแผ่นตัด SIZE", 20, true)),
		sub(sgk("PG02", "เหล็กแผ่น", 6, true)),
		sub(sgk("PG02", "แผ่นลาย", 9, true)),
		sub(sgk("PG02", "เหล็กแผ่น special", 19, true)),
	}

	labels := []string{"เหล็กแผ่น", "เหล็กแผ่นตัด SIZE", "แผ่นลาย", "เหล็กแผ่น special"}
	sortLabelsByValue(labels, subs, "PG02")

	want := []string{"เหล็กแผ่น", "แผ่นลาย", "เหล็กแผ่น special", "เหล็กแผ่นตัด SIZE"}
	for i := range want {
		if labels[i] != want[i] {
			t.Fatalf("ตำแหน่ง %d ได้ %q ต้องเป็น %q — ทั้งหมด %v", i, labels[i], want[i], labels)
		}
	}
}

func TestSortLabelsByValue_UnknownGoesLast(t *testing.T) {
	subs := []models.PriceListSubGroupResponse{
		sub(sgk("PG02", "แผ่นลาย", 9, true)),
		sub(models.PriceListSubGroupKeyResponse{GroupCode: "PG02", ValueCode: "PG02_X", ValueName: "ไม่รู้จัก"}),
	}

	labels := []string{"ไม่รู้จัก", "แผ่นลาย"}
	sortLabelsByValue(labels, subs, "PG02")

	if labels[0] != "แผ่นลาย" || labels[1] != "ไม่รู้จัก" {
		t.Fatalf("ตัวที่ resolve ไม่ได้ต้องไปท้าย แต่ได้ %v", labels)
	}
}
```

- [ ] **Step 2: รัน test ให้เห็นว่าแดง**

```bash
go test ./internal/services/price-service/patterns/ -run TestSortLabelsByValue -v
```

Expected: FAIL — `undefined: sortLabelsByValue`

- [ ] **Step 3: เขียน implementation**

เพิ่มท้าย `internal/services/price-service/patterns/sort_by_value.go`:

```go
// newValueByName สร้าง index จาก ValueName ไปหาค่าตัวเลข เฉพาะ groupCode เดียว
//
// จำกัดอยู่กลุ่มเดียวจึงไม่ชนกัน — ถ้าทำ index ข้ามกลุ่มจะชน เช่น PG05 มี item
// ชื่อ "100" (value 100) และ PG06 ก็มี item ชื่อ "100" (value 100) เหมือนกัน
func newValueByName(sgs []models.PriceListSubGroupResponse, groupCode string) valueByCode {
	idx := valueByCode{}

	for _, sg := range sgs {
		for _, k := range sg.SubGroupKeys {
			if k.GroupCode != groupCode || k.ValueName == "" {
				continue
			}
			if existing, ok := idx[k.ValueName]; ok && existing.has {
				continue
			}
			idx[k.ValueName] = codeValue{value: k.ValueNumber, has: k.HasValue}
		}
	}

	return idx
}

// sortLabelsByValue เรียง label in-place ด้วยค่าตัวเลขของ groupCode ที่ให้มา
// label ที่ resolve ไม่ได้จะไปท้ายเสมอ และเรียงกันเองด้วย string compare
func sortLabelsByValue(labels []string, sgs []models.PriceListSubGroupResponse, groupCode string) {
	idx := newValueByName(sgs, groupCode)
	sort.SliceStable(labels, func(i, j int) bool {
		return idx.Less(labels[i], labels[i], labels[j], labels[j])
	})
}
```

- [ ] **Step 4: รัน test ให้ผ่าน**

```bash
go test ./internal/services/price-service/patterns/ -run TestSortLabelsByValue -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/services/price-service/patterns/sort_by_value.go internal/services/price-service/patterns/sort_by_value_test.go
git commit -m "feat: เพิ่ม sortLabelsByValue สำหรับเรียงลำดับ tab

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 9: `pattern_group_1_item_1.go` — 4 จุด

**Files:**
- Modify: `internal/services/price-service/patterns/pattern_group_1_item_1.go`

จุดที่แก้: บรรทัด 40 (ลำดับ tab), 49-50 (เรียง subGroups ต้นทาง), 100 (ลำดับแถว), 131 (ลำดับคอลัมน์ตอน merge), 244 (sort ซ้ำหลัง merge), 298 (ลำดับ tab สุดท้าย)

จุดที่ **ห้ามแตะ**: บรรทัด 137 `sort.SliceStable(rowsForCol, ...)` ซึ่งเรียงด้วย `<colKey>_row_number` ไม่ใช่ค่าของ product group

- [ ] **Step 1: แก้ลำดับ tab ต้นทาง (บรรทัด 35-40)**

เดิม:

```go
		// Sort productGroup2 keys to ensure consistent iteration order
		productGroup2Keys := make([]string, 0, len(productGroup2Map))
		for pg2 := range productGroup2Map {
			productGroup2Keys = append(productGroup2Keys, pg2)
		}
		sort.Strings(productGroup2Keys)
```

ใหม่:

```go
		// เรียงลำดับ tab ด้วยค่าตัวเลขของ PRODUCT_GROUP2 จาก group_item.value
		// ไม่ใช่ชื่อ tab แบบ string
		productGroup2Keys := make([]string, 0, len(productGroup2Map))
		allSubGroupsForTabs := make([]models.PriceListSubGroupResponse, 0)
		for pg2, sgs := range productGroup2Map {
			productGroup2Keys = append(productGroup2Keys, pg2)
			allSubGroupsForTabs = append(allSubGroupsForTabs, sgs...)
		}
		sort.Strings(productGroup2Keys)
		sortLabelsByValue(productGroup2Keys, allSubGroupsForTabs, productGroup2CodeFromConfig(config))
```

> `sort.Strings` ตัวเดิมยังคงไว้ก่อน `sortLabelsByValue` เพื่อให้ลำดับตั้งต้นนิ่ง — `sortLabelsByValue` เป็น stable sort จึงต้องมีลำดับตั้งต้นที่แน่นอน ไม่งั้น label ที่ resolve ไม่ได้จะสลับตำแหน่งกันเองทุกครั้งที่รัน

- [ ] **Step 2: เพิ่ม helper `productGroup2CodeFromConfig`**

เพิ่มท้าย `internal/services/price-service/patterns/sort_by_value.go`:

```go
// productGroup2CodeFromConfig อ่าน group code ของแกน tab จาก config
// คืน "PG02" เป็นค่าเริ่มต้นเมื่อ config ไม่ได้ระบุ
func productGroup2CodeFromConfig(config *PriceTableConfiguration) string {
	if config == nil {
		return "PG02"
	}
	for _, p := range config.Patterns {
		if codes := splitGroupCodes(p.Grouping.Tabs); len(codes) > 0 {
			return codes[0]
		}
	}
	return "PG02"
}
```

- [ ] **Step 3: เรียง subGroups ที่ต้นทาง (บรรทัด 42-50)**

เดิม:

```go
		for _, productGroup2 := range productGroup2Keys {
			subGroups := productGroup2Map[productGroup2]

			pattern := selectPatternForCategory(config, productGroup2)
			if pattern == nil {
				continue
			}

			columns := buildDynamicColumns(pattern, subGroups)
			rowData := buildDynamicRows(config, pattern, subGroups)
```

ใหม่:

```go
		for _, productGroup2 := range productGroup2Keys {
			subGroups := productGroup2Map[productGroup2]

			pattern := selectPatternForCategory(config, productGroup2)
			if pattern == nil {
				continue
			}

			rowCodes := splitGroupCodes(pattern.Grouping.Rows)
			colCodes := splitGroupCodes(pattern.Grouping.ColumnGroups)

			// คอลัมน์กับแถวเรียงคนละแกน จึงต้องใช้ subGroups คนละชุด
			// ถ้าใช้ชุดเดียวกัน ลำดับคอลัมน์จะกลายเป็นลำดับที่เจอตอนไล่แถว
			colSorted := append([]models.PriceListSubGroupResponse(nil), subGroups...)
			SortSubGroupsByValue(colSorted, colCodes...)
			columns := buildDynamicColumns(pattern, colSorted)

			SortSubGroupsByValue(subGroups, append(append([]string{}, rowCodes...), colCodes...)...)
			rowData := buildDynamicRows(config, pattern, subGroups)
```

- [ ] **Step 4: แก้ลำดับแถว (บรรทัด ~95-100)**

เดิม:

```go
			// Sort row_group_value keys for deterministic row order
			rowGroupKeys := make([]string, 0, len(rowsByRowGroup))
			for k := range rowsByRowGroup {
				rowGroupKeys = append(rowGroupKeys, k)
			}
			sort.Strings(rowGroupKeys)
```

ใหม่:

```go
			// rowData ถูกสร้างจาก subGroups ที่เรียงด้วย group_item.value แล้ว
			// จึงเก็บ key ตามลำดับที่เจอครั้งแรกแทนการ sort ตัว key ที่ประกอบเสร็จแล้ว
			// (key ประกอบด้วย strings.Join(mergeKeyParts, "|") ซึ่งข้ามค่าว่าง แยกกลับไม่ได้)
			rowGroupKeys := orderedUnique(rowData, "row_group_value")
			if len(rowGroupKeys) != len(rowsByRowGroup) {
				// rowGroupValue ในลูปด้านบนอาจไม่ใช่ row_group_value ตรง ๆ
				// เก็บ key ที่ตกหล่นต่อท้ายเพื่อไม่ให้ข้อมูลหาย
				seen := map[string]bool{}
				for _, k := range rowGroupKeys {
					seen[k] = true
				}
				extra := make([]string, 0)
				for k := range rowsByRowGroup {
					if !seen[k] {
						extra = append(extra, k)
					}
				}
				sort.Strings(extra)
				rowGroupKeys = append(rowGroupKeys, extra...)
			}
			filtered := make([]string, 0, len(rowGroupKeys))
			for _, k := range rowGroupKeys {
				if _, ok := rowsByRowGroup[k]; ok {
					filtered = append(filtered, k)
				}
			}
			rowGroupKeys = filtered
```

- [ ] **Step 5: แก้ลำดับคอลัมน์ตอน merge (บรรทัด ~126-131)**

เดิม:

```go
				// Sort column keys for deterministic merge order
				columnKeys := make([]string, 0, len(columnsByKey))
				for k := range columnsByKey {
					columnKeys = append(columnKeys, k)
				}
				sort.Strings(columnKeys)
```

ใหม่:

```go
				// groupRows สืบทอดลำดับมาจาก subGroups ที่เรียงแล้ว
				columnKeys := orderedUnique(groupRows, "column_group_key")
```

- [ ] **Step 6: ลบ sort ซ้ำหลัง merge (บรรทัด ~243-248)**

เดิม:

```go
			// Sort final merged rows by row_group_value to keep the previous behavior
			sort.SliceStable(mergedRows, func(i, j int) bool {
				rowGroupI := fmt.Sprintf("%v", mergedRows[i]["row_group_value"])
				rowGroupJ := fmt.Sprintf("%v", mergedRows[j]["row_group_value"])
				return rowGroupI < rowGroupJ
			})
```

ใหม่ (ลบทิ้ง แล้วใส่คอมเมนต์แทน):

```go
			// ไม่ต้อง sort ซ้ำ — mergedRows ถูกสร้างโดยไล่ rowGroupKeys ที่เรียงด้วย
			// group_item.value มาแล้ว การ sort ด้วย row_group_value แบบ string
			// จะทำลายลำดับที่ถูกต้อง
```

- [ ] **Step 7: แก้ลำดับ tab สุดท้าย (บรรทัด ~297-303)**

เดิม:

```go
	// Sort tabs by pattern order (patternIdx), then by productGroup2 name for same pattern
	sort.Slice(tabsWithOrder, func(i, j int) bool {
		if tabsWithOrder[i].patternIdx != tabsWithOrder[j].patternIdx {
			return tabsWithOrder[i].patternIdx < tabsWithOrder[j].patternIdx
		}
		return tabsWithOrder[i].productGroup2 < tabsWithOrder[j].productGroup2
	})
```

ใหม่:

```go
	// เรียง tab ตาม pattern ก่อน แล้วจึงเรียงด้วยค่าตัวเลขของ PRODUCT_GROUP2
	// ภายใน pattern เดียวกัน — เทียบชื่อ tab แบบ string ให้ลำดับที่ไม่สื่ออะไร
	tabOrderIdx := map[string]int{}
	for i, label := range tabDisplayOrder {
		tabOrderIdx[label] = i
	}
	sort.SliceStable(tabsWithOrder, func(i, j int) bool {
		if tabsWithOrder[i].patternIdx != tabsWithOrder[j].patternIdx {
			return tabsWithOrder[i].patternIdx < tabsWithOrder[j].patternIdx
		}
		oi, okI := tabOrderIdx[tabsWithOrder[i].productGroup2]
		oj, okJ := tabOrderIdx[tabsWithOrder[j].productGroup2]
		if okI && okJ && oi != oj {
			return oi < oj
		}
		if okI != okJ {
			return okI
		}
		return tabsWithOrder[i].productGroup2 < tabsWithOrder[j].productGroup2
	})
```

- [ ] **Step 8: ประกาศ `tabDisplayOrder` สะสมข้าม groupKey**

เพิ่มก่อนลูป `for groupKey, productGroup2Map := range groupedData {` (บรรทัด ~27):

```go
	// tabDisplayOrder เก็บลำดับ tab ที่เรียงด้วย group_item.value ไว้ใช้ตอนเรียง
	// tabsWithOrder ตอนท้าย เพราะตรงนั้นมองเห็นแค่ชื่อ tab ไม่เห็น subGroups แล้ว
	tabDisplayOrder := []string{}
```

และในลูป หลัง `sortLabelsByValue(...)` ของ Step 1 เพิ่ม:

```go
		tabDisplayOrder = append(tabDisplayOrder, productGroup2Keys...)
```

- [ ] **Step 9: ยืนยันว่า build และ test เดิมผ่าน**

```bash
go build ./... && go test ./internal/services/price-service/... 2>&1 | tail -20
```

Expected: build ผ่าน test ผ่าน

- [ ] **Step 10: Commit**

```bash
git add internal/services/price-service/patterns/pattern_group_1_item_1.go internal/services/price-service/patterns/sort_by_value.go
git commit -m "fix: pattern 1 เรียงแถว คอลัมน์ และ tab ด้วย group_item.value

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 10: `pattern_group_1_item_3.go` — ลำดับ tab

**Files:**
- Modify: `internal/services/price-service/patterns/pattern_group_1_item_3.go:47-52`

- [ ] **Step 1: แก้จุด sort**

เดิม:

```go
	// Sort productGroup4 keys for consistent tab order
	productGroup4Keys := make([]string, 0, len(groupedByProductGroup4))
	for pg4 := range groupedByProductGroup4 {
		productGroup4Keys = append(productGroup4Keys, pg4)
	}
	sort.Strings(productGroup4Keys)
```

ใหม่:

```go
	// เรียงลำดับ tab ด้วยค่าตัวเลขของ PRODUCT_GROUP4 จาก group_item.value
	// sort.Strings ยังต้องมีก่อน เพื่อให้ลำดับตั้งต้นนิ่ง — sortLabelsByValue เป็น
	// stable sort ถ้าลำดับตั้งต้นสุ่ม label ที่ resolve ไม่ได้จะสลับกันเองทุกครั้ง
	productGroup4Code := getGroupCodeFromConfig(config, pattern, "productGroup4", "PRODUCT_GROUP4")
	productGroup4Keys := make([]string, 0, len(groupedByProductGroup4))
	allSubGroupsForTabs := make([]models.PriceListSubGroupResponse, 0)
	for pg4, sgs := range groupedByProductGroup4 {
		productGroup4Keys = append(productGroup4Keys, pg4)
		allSubGroupsForTabs = append(allSubGroupsForTabs, sgs...)
	}
	sort.Strings(productGroup4Keys)
	sortLabelsByValue(productGroup4Keys, allSubGroupsForTabs, productGroup4Code)
```

- [ ] **Step 2: เรียง subGroups ของแต่ละ tab ที่ต้นทาง**

หา loop `for _, productGroup4 := range productGroup4Keys {` (บรรทัด ~59) แล้วเพิ่มทันทีหลัง `subGroups := groupedByProductGroup4[productGroup4]`:

```go
		SortSubGroupsByValue(subGroups,
			append(append([]string{}, splitGroupCodes(pattern.Grouping.Rows)...),
				splitGroupCodes(pattern.Grouping.ColumnGroups)...)...)
```

- [ ] **Step 3: ยืนยันว่า build และ test เดิมผ่าน**

```bash
go build ./... && go test ./internal/services/price-service/... 2>&1 | tail -20
```

Expected: build ผ่าน test ผ่าน

- [ ] **Step 4: Commit**

```bash
git add internal/services/price-service/patterns/pattern_group_1_item_3.go
git commit -m "fix: pattern 3 เรียง tab และ subGroups ด้วย group_item.value

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 11: `pattern_group_1_item_4.go` — ลำดับ tab + เรียง subGroups

**Files:**
- Modify: `internal/services/price-service/patterns/pattern_group_1_item_4.go:58-85`

- [ ] **Step 1: แก้ลำดับ tab (บรรทัด ~58-66)**

เดิม:

```go
	remaining := make([]string, 0)
	for key := range groupedByProductGroup2 {
		if !seen[key] {
			remaining = append(remaining, key)
		}
	}
	sort.Strings(remaining)
	tabOrder = append(tabOrder, remaining...)
```

ใหม่:

```go
	remaining := make([]string, 0)
	allSubGroupsForTabs := make([]models.PriceListSubGroupResponse, 0)
	for key, sgs := range groupedByProductGroup2 {
		allSubGroupsForTabs = append(allSubGroupsForTabs, sgs...)
		if !seen[key] {
			remaining = append(remaining, key)
		}
	}
	// sort.Strings ก่อนเพื่อให้ลำดับตั้งต้นนิ่ง แล้วจึงเรียงด้วย group_item.value
	sort.Strings(remaining)
	sortLabelsByValue(remaining, allSubGroupsForTabs,
		getGroupCodeFromConfig(config, pattern, "productGroup2", "PRODUCT_GROUP2"))
	tabOrder = append(tabOrder, remaining...)
```

- [ ] **Step 2: แก้ comparator ของ subGroups (บรรทัด ~76-86)**

เดิม:

```go
		productGroup4Code := getGroupCodeFromConfig(config, pattern, "productGroup4", "PRODUCT_GROUP4")
		productGroup6Code := getGroupCodeFromConfig(config, pattern, "productGroup6", "PRODUCT_GROUP6")
		sort.SliceStable(subGroups, func(i, j int) bool {
			sizeI := getValueNameByGroupCode(subGroups[i].SubGroupKeys, productGroup4Code)
			sizeJ := getValueNameByGroupCode(subGroups[j].SubGroupKeys, productGroup4Code)
			if sizeI == sizeJ {
				thicknessI := getValueNameByGroupCode(subGroups[i].SubGroupKeys, productGroup6Code)
				thicknessJ := getValueNameByGroupCode(subGroups[j].SubGroupKeys, productGroup6Code)
				return thicknessI < thicknessJ
			}
			return sizeI < sizeJ
		})
```

ใหม่:

```go
		productGroup4Code := getGroupCodeFromConfig(config, pattern, "productGroup4", "PRODUCT_GROUP4")
		productGroup6Code := getGroupCodeFromConfig(config, pattern, "productGroup6", "PRODUCT_GROUP6")
		// เรียงด้วยค่าตัวเลขจาก group_item.value — เทียบ ValueName แบบ string
		// ให้ 1.2, 1.4, 1.9, 10, 100, 12 ซึ่งผิด
		SortSubGroupsByValue(subGroups, productGroup4Code, productGroup6Code)
```

- [ ] **Step 3: ยืนยันว่า build และ test เดิมผ่าน**

```bash
go build ./... && go test ./internal/services/price-service/... 2>&1 | tail -20
```

Expected: build ผ่าน test ผ่าน

- [ ] **Step 4: Commit**

```bash
git add internal/services/price-service/patterns/pattern_group_1_item_4.go
git commit -m "fix: pattern 4 เรียง tab และ subGroups ด้วย group_item.value

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 12: `pattern_group_1_item_5.go` — คอลัมน์ + ลำดับ tab

**Files:**
- Modify: `internal/services/price-service/patterns/pattern_group_1_item_5.go:121-129`
- Modify: `internal/services/price-service/patterns/pattern_group_1_item_5.go:206-213`

- [ ] **Step 1: แก้ลำดับคอลัมน์ (บรรทัด ~121-129)**

เดิม:

```go
	// Sort keys to ensure consistent column order by label
	sortedKeys := make([]string, 0, len(uniqueValues))
	for key := range uniqueValues {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return uniqueValues[sortedKeys[i]].Label < uniqueValues[sortedKeys[j]].Label
	})
```

ใหม่:

```go
	// เรียงคอลัมน์ด้วยค่าตัวเลขจาก group_item.value ไม่ใช่ label
	idx := newValueByCode(subGroups)
	sortedKeys := make([]string, 0, len(uniqueValues))
	for key := range uniqueValues {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		a, b := uniqueValues[sortedKeys[i]], uniqueValues[sortedKeys[j]]
		return idx.Less(a.Code, a.Label, b.Code, b.Label)
	})
```

> ถ้าฟังก์ชันนี้ไม่มีพารามิเตอร์ชื่อ `subGroups` ให้ใช้ชื่อ slice ของ subgroup ที่ฟังก์ชันรับเข้ามาแทน — `newValueByCode` รับ `[]models.PriceListSubGroupResponse`

- [ ] **Step 2: แก้ลำดับ tab (บรรทัด ~206-213)**

เดิม:

```go
	remaining := make([]string, 0)
	for label := range groupedData {
		if !seen[label] {
			remaining = append(remaining, label)
		}
	}
	sort.Strings(remaining)

	return append(tabOrder, remaining...)
```

ใหม่:

```go
	remaining := make([]string, 0)
	allSubGroupsForTabs := make([]models.PriceListSubGroupResponse, 0)
	for label, sgs := range groupedData {
		allSubGroupsForTabs = append(allSubGroupsForTabs, sgs...)
		if !seen[label] {
			remaining = append(remaining, label)
		}
	}
	// sort.Strings ก่อนเพื่อให้ลำดับตั้งต้นนิ่ง แล้วจึงเรียงด้วย group_item.value
	sort.Strings(remaining)
	sortLabelsByValue(remaining, allSubGroupsForTabs, tabGroupCode)

	return append(tabOrder, remaining...)
```

เพิ่มพารามิเตอร์ `tabGroupCode string` ให้ฟังก์ชันนี้ และที่ call site ส่ง
`getGroupCodeFromConfig(config, pattern, "productGroup2", "PRODUCT_GROUP2")` เข้าไป

- [ ] **Step 3: ยืนยันว่า build และ test เดิมผ่าน**

```bash
go build ./... && go test ./internal/services/price-service/... 2>&1 | tail -20
```

Expected: build ผ่าน test ผ่าน

- [ ] **Step 4: Commit**

```bash
git add internal/services/price-service/patterns/pattern_group_1_item_5.go
git commit -m "fix: pattern 5 เรียงคอลัมน์และ tab ด้วย group_item.value

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 13: `pattern_group_1_item_6.go` — เรียง subGroups

**Files:**
- Modify: `internal/services/price-service/patterns/pattern_group_1_item_6.go:58-69`

- [ ] **Step 1: แก้ comparator**

เดิม:

```go
		productGroup4Code := getGroupCodeFromConfig(config, pattern, "productGroup4", "PRODUCT_GROUP4")
		productGroup7Code := getGroupCodeFromConfig(config, pattern, "productGroup7", "PRODUCT_GROUP7")
		sort.SliceStable(subGroups, func(i, j int) bool {
			group4I := getValueNameByGroupCode(subGroups[i].SubGroupKeys, productGroup4Code)
			group4J := getValueNameByGroupCode(subGroups[j].SubGroupKeys, productGroup4Code)
			if group4I == group4J {
				group7I := getValueNameByGroupCode(subGroups[i].SubGroupKeys, productGroup7Code)
				group7J := getValueNameByGroupCode(subGroups[j].SubGroupKeys, productGroup7Code)
				return group7I < group7J
			}
			return group4I < group4J
		})
```

ใหม่:

```go
		productGroup4Code := getGroupCodeFromConfig(config, pattern, "productGroup4", "PRODUCT_GROUP4")
		productGroup7Code := getGroupCodeFromConfig(config, pattern, "productGroup7", "PRODUCT_GROUP7")
		// เรียงด้วยค่าตัวเลขจาก group_item.value แทนการเทียบ ValueName แบบ string
		SortSubGroupsByValue(subGroups, productGroup4Code, productGroup7Code)
```

- [ ] **Step 2: ยืนยันว่า build และ test เดิมผ่าน**

```bash
go build ./... && go test ./internal/services/price-service/... 2>&1 | tail -20
```

Expected: build ผ่าน test ผ่าน

- [ ] **Step 3: Commit**

```bash
git add internal/services/price-service/patterns/pattern_group_1_item_6.go
git commit -m "fix: pattern 6 เรียง subGroups ด้วย group_item.value

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 14: `pattern_group_1_item_7.go` — ลำดับ tab + แถว

**Files:**
- Modify: `internal/services/price-service/patterns/pattern_group_1_item_7.go:59-66`
- Modify: `internal/services/price-service/patterns/pattern_group_1_item_7.go:134-138`

- [ ] **Step 1: แก้ลำดับ tab (บรรทัด ~59-66)**

เดิม:

```go
	remaining := make([]string, 0)
	for key := range groupedByProductGroup2 {
		if !seen[key] {
			remaining = append(remaining, key)
		}
	}
	sort.Strings(remaining)
	tabOrder = append(tabOrder, remaining...)
```

ใหม่:

```go
	remaining := make([]string, 0)
	allSubGroupsForTabs := make([]models.PriceListSubGroupResponse, 0)
	for key, sgs := range groupedByProductGroup2 {
		allSubGroupsForTabs = append(allSubGroupsForTabs, sgs...)
		if !seen[key] {
			remaining = append(remaining, key)
		}
	}
	// sort.Strings ก่อนเพื่อให้ลำดับตั้งต้นนิ่ง แล้วจึงเรียงด้วย group_item.value
	sort.Strings(remaining)
	sortLabelsByValue(remaining, allSubGroupsForTabs,
		getGroupCodeFromConfig(config, pattern, "productGroup2", "PRODUCT_GROUP2"))
	tabOrder = append(tabOrder, remaining...)
```

- [ ] **Step 2: เรียง subGroups ที่ต้นทาง**

หา `subGroups := groupedByProductGroup2[tabLabel]` (บรรทัด ~70) แล้วเพิ่มทันทีหลังบล็อก `if len(subGroups) == 0 { continue }`:

```go
		SortSubGroupsByValue(subGroups,
			append(append([]string{}, splitGroupCodes(pattern.Grouping.Rows)...),
				splitGroupCodes(pattern.Grouping.ColumnGroups)...)...)
```

- [ ] **Step 3: ลบ sort หลัง merge (บรรทัด ~134-138)**

เดิม:

```go
		sort.SliceStable(mergedRows, func(i, j int) bool {
			thicknessI := fmt.Sprintf("%v", mergedRows[i]["product_group_6"])
			thicknessJ := fmt.Sprintf("%v", mergedRows[j]["product_group_6"])
			return thicknessI < thicknessJ
		})
```

ใหม่ (ลบทิ้ง แล้วใส่คอมเมนต์แทน):

```go
		// ไม่ต้อง sort ซ้ำ — mergedRowMap ถูกไล่เก็บจาก rowData ที่มาจาก subGroups
		// ที่เรียงด้วย group_item.value แล้ว การ sort ด้วย product_group_6 แบบ string
		// จะทำลายลำดับที่ถูกต้อง
```

> ตรวจก่อนว่า `mergedRows` ถูกสร้างจากการวน `mergedRowMap` โดยตรงหรือไม่ ถ้าใช่ต้องเปลี่ยนให้วนตาม slice ของ key ที่ได้จาก `orderedUnique(rowData, "row_group_value")` เพราะการวน `map` ใน Go สุ่มลำดับ

- [ ] **Step 4: ยืนยันว่า build และ test เดิมผ่าน**

```bash
go build ./... && go test ./internal/services/price-service/... 2>&1 | tail -20
```

Expected: build ผ่าน test ผ่าน

- [ ] **Step 5: Commit**

```bash
git add internal/services/price-service/patterns/pattern_group_1_item_7.go
git commit -m "fix: pattern 7 เรียง tab และแถวด้วย group_item.value

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 15: `pattern_group_1_item_8.go` — ลำดับ tab + แถว

**Files:**
- Modify: `internal/services/price-service/patterns/pattern_group_1_item_8.go:59-66`
- Modify: `internal/services/price-service/patterns/pattern_group_1_item_8.go:79-88`

- [ ] **Step 1: แก้ลำดับ tab (บรรทัด ~59-66)**

เดิม:

```go
	remaining := make([]string, 0)
	for key := range groupedByProductGroup1 {
		if !seen[key] {
			remaining = append(remaining, key)
		}
	}
	sort.Strings(remaining)
	tabOrder = append(tabOrder, remaining...)
```

ใหม่:

```go
	remaining := make([]string, 0)
	allSubGroupsForTabs := make([]models.PriceListSubGroupResponse, 0)
	for key, sgs := range groupedByProductGroup1 {
		allSubGroupsForTabs = append(allSubGroupsForTabs, sgs...)
		if !seen[key] {
			remaining = append(remaining, key)
		}
	}
	// sort.Strings ก่อนเพื่อให้ลำดับตั้งต้นนิ่ง แล้วจึงเรียงด้วย group_item.value
	sort.Strings(remaining)
	sortLabelsByValue(remaining, allSubGroupsForTabs,
		getGroupCodeFromConfig(config, pattern, "productGroup1", "PRODUCT_GROUP1"))
	tabOrder = append(tabOrder, remaining...)
```

- [ ] **Step 2: แก้การเรียงแถว (บรรทัด ~76-88)**

เดิม:

```go
		rows := buildDirectRows(config, pattern, subGroups)

		sort.SliceStable(rows, func(i, j int) bool {
			thicknessI := fmt.Sprintf("%v", rows[i]["product_group_6"])
			thicknessJ := fmt.Sprintf("%v", rows[j]["product_group_6"])
			if thicknessI == thicknessJ {
				shipI := fmt.Sprintf("%v", rows[i]["ship_no"])
				shipJ := fmt.Sprintf("%v", rows[j]["ship_no"])
				return shipI < shipJ
			}
			return thicknessI < thicknessJ
		})
```

ใหม่:

```go
		// เรียงที่ต้นทางด้วย group_item.value แล้ว buildDirectRows จะผลิตแถวตามลำดับนั้น
		productGroup6Code := getGroupCodeFromConfig(config, pattern, "productGroup6", "PRODUCT_GROUP6")
		SortSubGroupsByValue(subGroups, productGroup6Code)

		rows := buildDirectRows(config, pattern, subGroups)

		// ship_no ไม่ได้มาจาก product group จึงยัง tie-break ด้วย string ตามเดิม
		// ใช้ SliceStable เพื่อไม่ทำลายลำดับ product_group_6 ที่เรียงมาแล้ว
		sort.SliceStable(rows, func(i, j int) bool {
			thicknessI := fmt.Sprintf("%v", rows[i]["product_group_6"])
			thicknessJ := fmt.Sprintf("%v", rows[j]["product_group_6"])
			if thicknessI != thicknessJ {
				return false
			}
			shipI := fmt.Sprintf("%v", rows[i]["ship_no"])
			shipJ := fmt.Sprintf("%v", rows[j]["ship_no"])
			return shipI < shipJ
		})
```

- [ ] **Step 3: ยืนยันว่า build และ test เดิมผ่าน**

```bash
go build ./... && go test ./internal/services/price-service/... 2>&1 | tail -20
```

Expected: build ผ่าน test ผ่าน

- [ ] **Step 4: Commit**

```bash
git add internal/services/price-service/patterns/pattern_group_1_item_8.go
git commit -m "fix: pattern 8 เรียง tab และแถวด้วย group_item.value

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 16: pattern 9, 10, 12, 13 — เรียง `allSubGroups` แล้วลบ sort หลัง merge

**Files:**
- Modify: `internal/services/price-service/patterns/pattern_group_1_item_9.go:41-60`
- Modify: `internal/services/price-service/patterns/pattern_group_1_item_10.go:41-60`
- Modify: `internal/services/price-service/patterns/pattern_group_1_item_12.go:41-50`
- Modify: `internal/services/price-service/patterns/pattern_group_1_item_13.go:41-62`

บริบทสำคัญ: `mergeGroup1Item9Rows` (`pattern_group_1_item_9.go:104-144`) **รักษาลำดับที่เจอครั้งแรกอยู่แล้ว** (เก็บ `order []string` แล้ววนตามนั้น) ดังนั้นถ้า `allSubGroups` เรียงถูก แถวที่ออกมาก็เรียงถูกตาม `sort.SliceStable` ที่ตามหลัง merge จึงเป็นตัวที่ทำลายลำดับ ต้องลบทิ้ง

- [ ] **Step 1: แก้ `pattern_group_1_item_9.go` (บรรทัด ~41-60)**

เดิม:

```go
	columns := buildDynamicColumns(pattern, allSubGroups)
	rowData := buildDynamicRows(config, pattern, allSubGroups)
	mergedRows := mergeGroup1Item9Rows(rowData)

	sort.SliceStable(mergedRows, func(i, j int) bool {
		productGroupI := fmt.Sprintf("%v", mergedRows[i]["product_group_2"])
		productGroupJ := fmt.Sprintf("%v", mergedRows[j]["product_group_2"])
		if productGroupI == productGroupJ {
			lengthI := fmt.Sprintf("%v", mergedRows[i]["product_group_7"])
			lengthJ := fmt.Sprintf("%v", mergedRows[j]["product_group_7"])
			if lengthI == lengthJ {
				sizeI := fmt.Sprintf("%v", mergedRows[i]["product_group_6"])
				sizeJ := fmt.Sprintf("%v", mergedRows[j]["product_group_6"])
				return sizeI < sizeJ
			}
			return lengthI < lengthJ
		}
		return productGroupI < productGroupJ
	})
```

ใหม่:

```go
	productGroup2Code := getGroupCodeFromConfig(config, pattern, "productGroup2", "PRODUCT_GROUP2")
	productGroup7Code := getGroupCodeFromConfig(config, pattern, "productGroup7", "PRODUCT_GROUP7")
	productGroup6Code := getGroupCodeFromConfig(config, pattern, "productGroup6", "PRODUCT_GROUP6")

	// คอลัมน์เรียงคนละแกนกับแถว จึงต้องใช้ subGroups คนละชุด
	colSorted := append([]models.PriceListSubGroupResponse(nil), allSubGroups...)
	SortSubGroupsByValue(colSorted, splitGroupCodes(pattern.Grouping.ColumnGroups)...)
	columns := buildDynamicColumns(pattern, colSorted)

	// เรียงที่ต้นทางด้วย group_item.value แล้ว mergeGroup1Item9Rows จะรักษาลำดับ
	// ที่เจอครั้งแรกไว้ให้เอง จึงไม่ต้อง sort ซ้ำหลัง merge
	SortSubGroupsByValue(allSubGroups, productGroup2Code, productGroup7Code, productGroup6Code)
	rowData := buildDynamicRows(config, pattern, allSubGroups)
	mergedRows := mergeGroup1Item9Rows(rowData)
```

- [ ] **Step 2: แก้ `pattern_group_1_item_10.go` (บรรทัด ~41-60)**

เดิม:

```go
	columns := buildDynamicColumns(pattern, allSubGroups)
	rows := buildDynamicRows(config, pattern, allSubGroups)
	mergedRows := mergeGroup1Item9Rows(rows)

	sort.SliceStable(mergedRows, func(i, j int) bool {
		itemI := fmt.Sprintf("%v", mergedRows[i]["product_group_4"])
		itemJ := fmt.Sprintf("%v", mergedRows[j]["product_group_4"])
		if itemI == itemJ {
			lengthI := fmt.Sprintf("%v", mergedRows[i]["product_group_7"])
			lengthJ := fmt.Sprintf("%v", mergedRows[j]["product_group_7"])
			if lengthI == lengthJ {
				// total_weight (คอลัมน์ Weight-spec) เป็นตัวเลข ต้องเทียบเป็น float
				// ไม่ใช่ string ไม่งั้น "12.5" จะมาก่อน "9"
				weightI, _ := toFloat64(mergedRows[i]["total_weight"])
				weightJ, _ := toFloat64(mergedRows[j]["total_weight"])
				return weightI < weightJ
			}
			return lengthI < lengthJ
		}
		return itemI < itemJ
	})
```

ใหม่:

```go
	productGroup4Code := getGroupCodeFromConfig(config, pattern, "productGroup4", "PRODUCT_GROUP4")
	productGroup7Code := getGroupCodeFromConfig(config, pattern, "productGroup7", "PRODUCT_GROUP7")

	// คอลัมน์เรียงคนละแกนกับแถว จึงต้องใช้ subGroups คนละชุด
	colSorted := append([]models.PriceListSubGroupResponse(nil), allSubGroups...)
	SortSubGroupsByValue(colSorted, splitGroupCodes(pattern.Grouping.ColumnGroups)...)
	columns := buildDynamicColumns(pattern, colSorted)

	SortSubGroupsByValue(allSubGroups, productGroup4Code, productGroup7Code)
	rows := buildDynamicRows(config, pattern, allSubGroups)
	mergedRows := mergeGroup1Item9Rows(rows)

	// total_weight ไม่ได้มาจาก product group จึงยัง tie-break ด้วย float ตามเดิม
	// ใช้ SliceStable และ return false เมื่อแกน product group ต่างกัน เพื่อไม่ทำลาย
	// ลำดับที่เรียงมาแล้วจากต้นทาง
	sort.SliceStable(mergedRows, func(i, j int) bool {
		itemI := fmt.Sprintf("%v", mergedRows[i]["product_group_4"])
		itemJ := fmt.Sprintf("%v", mergedRows[j]["product_group_4"])
		lengthI := fmt.Sprintf("%v", mergedRows[i]["product_group_7"])
		lengthJ := fmt.Sprintf("%v", mergedRows[j]["product_group_7"])
		if itemI != itemJ || lengthI != lengthJ {
			return false
		}
		weightI, _ := toFloat64(mergedRows[i]["total_weight"])
		weightJ, _ := toFloat64(mergedRows[j]["total_weight"])
		return weightI < weightJ
	})
```

- [ ] **Step 3: แก้ `pattern_group_1_item_12.go` (บรรทัด ~41-50)**

เดิม:

```go
	columns := buildDynamicColumns(pattern, allSubGroups)
	rowData := buildDynamicRows(config, pattern, allSubGroups)
	mergedRows := mergeGroup1Item9Rows(rowData)

	sort.SliceStable(mergedRows, func(i, j int) bool {
		rowGroupI := fmt.Sprintf("%v", mergedRows[i]["row_group_value"])
		rowGroupJ := fmt.Sprintf("%v", mergedRows[j]["row_group_value"])
		return rowGroupI < rowGroupJ
	})
```

ใหม่:

```go
	// คอลัมน์เรียงคนละแกนกับแถว จึงต้องใช้ subGroups คนละชุด
	colSorted := append([]models.PriceListSubGroupResponse(nil), allSubGroups...)
	SortSubGroupsByValue(colSorted, splitGroupCodes(pattern.Grouping.ColumnGroups)...)
	columns := buildDynamicColumns(pattern, colSorted)

	// เรียงที่ต้นทางด้วย group_item.value — row_group_value เป็น composite ที่
	// ข้ามค่าว่างตอน join จึงแยกกลับไปเทียบเป็นตัวเลขไม่ได้
	SortSubGroupsByValue(allSubGroups, splitGroupCodes(pattern.Grouping.Rows)...)
	rowData := buildDynamicRows(config, pattern, allSubGroups)
	mergedRows := mergeGroup1Item9Rows(rowData)
```

- [ ] **Step 4: แก้ `pattern_group_1_item_13.go` (บรรทัด ~41-62)**

เดิม:

```go
	productGroup1Code := getGroupCodeFromConfig(config, pattern, "productGroup1", "PRODUCT_GROUP1")
	productGroup7Code := getGroupCodeFromConfig(config, pattern, "productGroup7", "PRODUCT_GROUP7")
	productGroup4Code := getGroupCodeFromConfig(config, pattern, "productGroup4", "PRODUCT_GROUP4")
	productGroup3Code := getGroupCodeFromConfig(config, pattern, "productGroup3", "PRODUCT_GROUP3")
	sort.SliceStable(allSubGroups, func(i, j int) bool {
		pg1I := getValueNameByGroupCode(allSubGroups[i].SubGroupKeys, productGroup1Code)
		pg1J := getValueNameByGroupCode(allSubGroups[j].SubGroupKeys, productGroup1Code)
		if pg1I == pg1J {
			lengthI := getValueNameByGroupCode(allSubGroups[i].SubGroupKeys, productGroup7Code)
			lengthJ := getValueNameByGroupCode(allSubGroups[j].SubGroupKeys, productGroup7Code)
			if lengthI == lengthJ {
				size4I := getValueNameByGroupCode(allSubGroups[i].SubGroupKeys, productGroup4Code)
				size4J := getValueNameByGroupCode(allSubGroups[j].SubGroupKeys, productGroup4Code)
				if size4I == size4J {
					size3I := getValueNameByGroupCode(allSubGroups[i].SubGroupKeys, productGroup3Code)
					size3J := getValueNameByGroupCode(allSubGroups[j].SubGroupKeys, productGroup3Code)
					return size3I < size3J
				}
				return size4I < size4J
			}
			return lengthI < lengthJ
		}
		return pg1I < pg1J
	})
```

ใหม่:

```go
	productGroup1Code := getGroupCodeFromConfig(config, pattern, "productGroup1", "PRODUCT_GROUP1")
	productGroup7Code := getGroupCodeFromConfig(config, pattern, "productGroup7", "PRODUCT_GROUP7")
	productGroup4Code := getGroupCodeFromConfig(config, pattern, "productGroup4", "PRODUCT_GROUP4")
	productGroup3Code := getGroupCodeFromConfig(config, pattern, "productGroup3", "PRODUCT_GROUP3")
	// เรียงด้วยค่าตัวเลขจาก group_item.value แทนการเทียบ ValueName แบบ string
	SortSubGroupsByValue(allSubGroups, productGroup1Code, productGroup7Code, productGroup4Code, productGroup3Code)
```

- [ ] **Step 5: ยืนยันว่า build และ test เดิมผ่าน**

```bash
go build ./... && go test ./internal/services/price-service/... 2>&1 | tail -20
```

Expected: build ผ่าน test ผ่าน

หาก compiler แจ้งว่า `sort` หรือ `fmt` ไม่ถูกใช้แล้วในไฟล์ไหน ให้ลบ import นั้นออก — เป็น orphan ที่เกิดจากการแก้ครั้งนี้เอง

- [ ] **Step 6: Commit**

```bash
git add internal/services/price-service/patterns/pattern_group_1_item_9.go internal/services/price-service/patterns/pattern_group_1_item_10.go internal/services/price-service/patterns/pattern_group_1_item_12.go internal/services/price-service/patterns/pattern_group_1_item_13.go
git commit -m "fix: pattern 9, 10, 12, 13 เรียง subGroups ด้วย group_item.value

mergeGroup1Item9Rows รักษาลำดับที่เจอครั้งแรกอยู่แล้ว จึงลบ sort หลัง merge
ที่เทียบ string ทิ้ง เหลือไว้เฉพาะ tie-break ด้วย total_weight ที่ไม่ได้มาจาก
product group

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 17: `pattern_group_1_item_11.go` — composite sort

**Files:**
- Modify: `internal/services/price-service/patterns/pattern_group_1_item_11.go:51-76`

- [ ] **Step 1: แก้ comparator**

เดิม:

```go
	// Sort subgroups by "หนา x ยาว" (PRODUCT_GROUP6 x PRODUCT_GROUP7) for row spanning
	productGroup6Code := getGroupCodeFromConfig(config, pattern, "productGroup6", "PRODUCT_GROUP6")
	productGroup7Code := getGroupCodeFromConfig(config, pattern, "productGroup7", "PRODUCT_GROUP7")
	productGroup5Code := getGroupCodeFromConfig(config, pattern, "productGroup5", "PRODUCT_GROUP5")
	productGroup3Code := getGroupCodeFromConfig(config, pattern, "productGroup3", "PRODUCT_GROUP3")
	sort.SliceStable(allSubGroups, func(i, j int) bool {
		// ต้องใช้ compositeMappingValue เหมือนตอนสร้างแถว ไม่งั้นเรียงตามค่าที่
		// ไม่ตรงกับที่แสดง
		thicknessLength := []string{productGroup6Code, productGroup7Code}
		compositeI := compositeMappingValue(allSubGroups[i].SubGroupKeys, thicknessLength, "_x_")
		compositeJ := compositeMappingValue(allSubGroups[j].SubGroupKeys, thicknessLength, "_x_")

		if compositeI == compositeJ {
			// If same "หนา x ยาว", sort by "ขนาด" (PRODUCT_GROUP5 + PRODUCT_GROUP3)
			pg5I := getValueNameByGroupCode(allSubGroups[i].SubGroupKeys, productGroup5Code)
			pg3I := getValueNameByGroupCode(allSubGroups[i].SubGroupKeys, productGroup3Code)
			pg5J := getValueNameByGroupCode(allSubGroups[j].SubGroupKeys, productGroup5Code)
			pg3J := getValueNameByGroupCode(allSubGroups[j].SubGroupKeys, productGroup3Code)

			sizeI := pg5I + pg3I
			sizeJ := pg5J + pg3J
			return sizeI < sizeJ
		}
		return compositeI < compositeJ
	})
```

ใหม่:

```go
	// Sort subgroups by "หนา x ยาว" (PRODUCT_GROUP6 x PRODUCT_GROUP7) for row spanning
	productGroup6Code := getGroupCodeFromConfig(config, pattern, "productGroup6", "PRODUCT_GROUP6")
	productGroup7Code := getGroupCodeFromConfig(config, pattern, "productGroup7", "PRODUCT_GROUP7")
	productGroup5Code := getGroupCodeFromConfig(config, pattern, "productGroup5", "PRODUCT_GROUP5")
	productGroup3Code := getGroupCodeFromConfig(config, pattern, "productGroup3", "PRODUCT_GROUP3")

	// เรียงทีละแกนด้วยค่าตัวเลขจาก group_item.value แทนการประกอบ composite
	// แล้วเทียบเป็น string — compositeMappingValue ข้ามค่าว่างตอน join จึงเทียบ
	// กลับเป็นตัวเลขไม่ได้ แต่การไล่ทีละแกนให้ผลลัพธ์เดียวกันและถูกต้องกว่า
	//
	// ลำดับแกนต้องตรงกับตอนประกอบ composite เป๊ะ ๆ (PG6 -> PG7 -> PG5 -> PG3)
	// ไม่งั้น row spanning จะไม่ตรงกับค่าที่แสดง
	SortSubGroupsByValue(allSubGroups, productGroup6Code, productGroup7Code, productGroup5Code, productGroup3Code)
```

- [ ] **Step 2: ยืนยันว่า build และ test เดิมผ่าน**

```bash
go build ./... && go test ./internal/services/price-service/... 2>&1 | tail -20
```

Expected: build ผ่าน test ผ่าน

- [ ] **Step 3: Commit**

```bash
git add internal/services/price-service/patterns/pattern_group_1_item_11.go
git commit -m "fix: pattern 11 เรียง subGroups ทีละแกนด้วย group_item.value

compositeMappingValue ข้ามค่าว่างตอน join จึงเทียบกลับเป็นตัวเลขไม่ได้
การไล่ทีละแกนตามลำดับเดิม (PG6, PG7, PG5, PG3) ให้ผลเดียวกันและถูกต้องกว่า

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 18: Golden order test จากข้อมูลจริง UAT

**Files:**
- Create: `internal/services/price-service/patterns/sort_golden_test.go`

ค่าทั้งหมดในเทสต์นี้ดึงจาก UAT จริง (`thaimetal-wms-uat`, site `TMI_WH`, `GROUP_1_ITEM_1`) ผ่าน `POST /api/erp/group/GetGroupMaster` และ `POST /api/erp/price/GetPriceDetail`

เทสต์นี้เรียก helper ตรง ๆ ไม่ผ่าน `LoadConfiguration` จึงไม่ผูกกับไฟล์ config ของ group ใด — ยืนยันกติกาการเรียงล้วน ๆ ด้วยค่าจริงจาก production data

- [ ] **Step 1: เขียน golden test**

สร้าง `internal/services/price-service/patterns/sort_golden_test.go`:

```go
package patterns

import (
	"fmt"
	"testing"

	"prime-erp-core/internal/models"
)

// ค่าจริงของ PG06 "ความหนา" จาก group_item ใน UAT
var uatPG06 = map[string]float64{
	"1.2": 1.20, "1.4": 1.40, "1.9": 1.90, "2.3": 2.30, "2.5": 2.50,
	"2.9": 2.90, "3.8": 3.80, "4.3": 4.30, "4.8": 4.80, "5.5": 5.50,
	"5.8": 5.80, "8": 8.00, "9": 9.00, "10": 10.00, "12": 12.00,
	"15": 15.00, "16": 16.00, "19": 19.00, "20": 20.00, "22": 22.00,
	"25": 25.00, "27": 27.00, "30": 30.00, "32": 32.00, "38": 38.00,
	"40": 40.00, "45": 45.00, "50": 50.00, "100": 100.00,
}

// ค่าจริงของ PG05 "ขนาดหน้ากว้าง" จาก group_item ใน UAT
// value คือผลคูณสองด้าน (พื้นที่) ไม่ใช่ความกว้าง
var uatPG05 = map[string]float64{
	"4' x 8'": 32.00, "5' x 10'": 50.00, "5' x 20'": 100.00,
	"75x75": 5625.00, "4'x1500": 6000.00, "4'x2400": 9600.00,
	"1250x8'": 10000.00, "100x100": 10000.00, "125x125": 15625.00,
	"40x520": 20800.00, "150x150": 22500.00, "150x175": 26250.00,
	"100x300": 30000.00, "150x200": 30000.00, "65x500": 32500.00,
	"200x200": 40000.00, "200x220": 44000.00, "200x300": 60000.00,
	"250x250": 62500.00, "300x300": 90000.00, "5'x5700": 28500.00,
}

// ค่าจริงของ PG03 "เกรด/รูปแบบ" จาก group_item ใน UAT
var uatPG03 = map[string]float64{"SS400": 10.00, "LT": 21.00, "T": 8.00, "S4": 33.00}

// ค่าจริงของ PG02 "หมวดย่อย" จาก group_item ใน UAT
var uatPG02 = map[string]float64{
	"เหล็กแผ่น": 6.00, "แผ่นลาย": 9.00,
	"เหล็กแผ่น special": 19.00, "เหล็กแผ่นตัด SIZE": 20.00,
}

func uatKey(groupCode, name string, table map[string]float64) models.PriceListSubGroupKeyResponse {
	v, ok := table[name]
	return models.PriceListSubGroupKeyResponse{
		GroupCode:   groupCode,
		ValueCode:   fmt.Sprintf("%s_%s", groupCode, name),
		ValueName:   name,
		ValueNumber: v,
		HasValue:    ok,
	}
}

func uatSubGroup(id, pg02, pg05, pg06 string) models.PriceListSubGroupResponse {
	return models.PriceListSubGroupResponse{
		ID:           id,
		SubgroupCode: id,
		SubGroupKeys: []models.PriceListSubGroupKeyResponse{
			uatKey("PG02", pg02, uatPG02),
			uatKey("PG05", pg05, uatPG05),
			uatKey("PG06", pg06, uatPG06),
		},
	}
}

// บั๊กที่ business รายงาน: แถว "แผ่น mm." แสดง 1.2, 1.4, 1.9, 10, 100, 12, 15
// เพราะ sort.Strings เทียบเป็น string
func TestGolden_RowOrder_ThicknessIsNumeric(t *testing.T) {
	// ลำดับตั้งต้นคือผลของ lexicographic sort ที่เป็นบั๊กอยู่ตอนนี้
	input := []string{
		"1.2", "1.4", "1.9", "10", "100", "12", "15", "16", "19",
		"2.3", "2.5", "2.9", "20", "22", "25", "27", "3.8", "30",
		"32", "38", "4.3", "4.8", "40", "45", "5.5", "5.8", "50", "8", "9",
	}

	subs := make([]models.PriceListSubGroupResponse, 0, len(input))
	for i, name := range input {
		subs = append(subs, uatSubGroup(fmt.Sprintf("sg-%d", i), "เหล็กแผ่น", "4' x 8'", name))
	}

	SortSubGroupsByValue(subs, "PG06")

	want := []string{
		"1.2", "1.4", "1.9", "2.3", "2.5", "2.9", "3.8", "4.3", "4.8",
		"5.5", "5.8", "8", "9", "10", "12", "15", "16", "19", "20",
		"22", "25", "27", "30", "32", "38", "40", "45", "50", "100",
	}

	for i := range want {
		got := subs[i].SubGroupKeys[2].ValueName
		if got != want[i] {
			t.Fatalf("แถวที่ %d ได้ %q ต้องเป็น %q", i, got, want[i])
		}
	}
}

// หัวคอลัมน์ tab "เหล็กแผ่น" ปัจจุบันเรียง 1250x8', 4' x 8', 4'x1500, ...
// ต้องเรียงด้วย group_item.value
func TestGolden_ColumnOrder_SteelSheetTab(t *testing.T) {
	input := []string{"1250x8'", "4' x 8'", "4'x1500", "4'x2400", "5' x 10'", "5' x 20'", "5'x5700"}

	subs := make([]models.PriceListSubGroupResponse, 0, len(input))
	for i, name := range input {
		subs = append(subs, uatSubGroup(fmt.Sprintf("sg-%d", i), "เหล็กแผ่น", name, "1.2"))
	}

	SortSubGroupsByValue(subs, "PG05")

	want := []string{"4' x 8'", "5' x 10'", "5' x 20'", "4'x1500", "4'x2400", "1250x8'", "5'x5700"}
	for i := range want {
		got := subs[i].SubGroupKeys[1].ValueName
		if got != want[i] {
			t.Fatalf("คอลัมน์ที่ %d ได้ %q ต้องเป็น %q", i, got, want[i])
		}
	}
}

// หัวคอลัมน์ tab "เหล็กแผ่นตัด SIZE"
func TestGolden_ColumnOrder_CutSizeTab(t *testing.T) {
	input := []string{
		"100x100", "100x300", "125x125", "150x150", "150x175", "150x200",
		"200x200", "200x220", "200x300", "250x250", "300x300", "40x520",
		"65x500", "75x75",
	}

	subs := make([]models.PriceListSubGroupResponse, 0, len(input))
	for i, name := range input {
		subs = append(subs, uatSubGroup(fmt.Sprintf("sg-%d", i), "เหล็กแผ่นตัด SIZE", name, "10"))
	}

	SortSubGroupsByValue(subs, "PG05")

	got := make([]string, 0, len(subs))
	for _, s := range subs {
		got = append(got, s.SubGroupKeys[1].ValueName)
	}

	expected := []string{
		"75x75", "100x100", "125x125", "40x520", "150x150", "150x175",
		"100x300", "150x200", "65x500", "200x200", "200x220", "200x300",
		"250x250", "300x300",
	}

	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("คอลัมน์ที่ %d ได้ %q ต้องเป็น %q — ทั้งหมด %v", i, got[i], expected[i], got)
		}
	}
}

// tab "เหล็กแผ่น special" เรียงด้วยเกรด ไม่ใช่ตัวเลขในชื่อ
func TestGolden_ColumnOrder_SpecialTabUsesGrade(t *testing.T) {
	subs := []models.PriceListSubGroupResponse{
		{ID: "sg-lt", SubGroupKeys: []models.PriceListSubGroupKeyResponse{uatKey("PG03", "LT", uatPG03)}},
		{ID: "sg-ss", SubGroupKeys: []models.PriceListSubGroupKeyResponse{uatKey("PG03", "SS400", uatPG03)}},
	}

	SortSubGroupsByValue(subs, "PG03")

	if subs[0].SubGroupKeys[0].ValueName != "SS400" || subs[1].SubGroupKeys[0].ValueName != "LT" {
		t.Fatalf("ต้องได้ SS400 แล้ว LT แต่ได้ %q, %q",
			subs[0].SubGroupKeys[0].ValueName, subs[1].SubGroupKeys[0].ValueName)
	}
}

// ลำดับ tab ต้องเรียงด้วยค่าของ PG02
func TestGolden_TabOrder(t *testing.T) {
	labels := []string{"เหล็กแผ่น", "เหล็กแผ่นตัด SIZE", "แผ่นลาย", "เหล็กแผ่น special"}

	subs := make([]models.PriceListSubGroupResponse, 0, len(labels))
	for i, l := range labels {
		subs = append(subs, uatSubGroup(fmt.Sprintf("sg-%d", i), l, "4' x 8'", "10"))
	}

	sortLabelsByValue(labels, subs, "PG02")

	want := []string{"เหล็กแผ่น", "แผ่นลาย", "เหล็กแผ่น special", "เหล็กแผ่นตัด SIZE"}
	for i := range want {
		if labels[i] != want[i] {
			t.Fatalf("tab ที่ %d ได้ %q ต้องเป็น %q — ทั้งหมด %v", i, labels[i], want[i], labels)
		}
	}
}
```

- [ ] **Step 2: รัน golden test**

```bash
go test ./internal/services/price-service/patterns/ -run TestGolden -v
```

Expected: PASS ทุก test — ถ้าแดง แปลว่า Task 5-17 ยังไม่ครบ

- [ ] **Step 3: Commit**

```bash
git add internal/services/price-service/patterns/sort_golden_test.go
git commit -m "test: golden order จากข้อมูลจริง UAT

ค่าทั้งหมดดึงจาก thaimetal-wms-uat site TMI_WH group GROUP_1_ITEM_1
ครอบคลุมเคสที่ natural sort แก้ไม่ได้: ขนาดปนหน่วย (4' x 8' กับ 4'x1500),
เกรดเหล็กที่ไม่ใช่ตัวเลข (SS400, LT) และ tie-break ที่ value เท่ากัน
(100x300 กับ 150x200 = 30,000.00)

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 19: Integration test บน testcontainers

**Files:**
- Create: `internal/services/price-service/pricelist_sort_integration_test.go`

- [ ] **Step 1: ดูรูปแบบ integration test ที่มีอยู่**

```bash
head -60 internal/services/price-service/weight_spec_integration_test.go
head -40 internal/repositories/priceList/repository_test.go
```

Expected: เห็น build tag `//go:build integration`, การตั้ง container, และ schema ที่ใช้ (`repository_test.go:96` และ `:162` มี `seq integer`)

- [ ] **Step 2: เขียน integration test**

สร้าง `internal/services/price-service/pricelist_sort_integration_test.go` โดยใช้ helper ตั้ง container ตัวเดียวกับ `weight_spec_integration_test.go` และ seed ข้อมูลนี้:

```go
//go:build integration

package priceService

import (
	"testing"

	"prime-erp-core/internal/models"
)

// seed group/group_item ด้วยค่าจริงจาก UAT แล้วยืนยันว่า parseGroupItemValue
// อ่านค่าจาก varchar ที่มี thousand separator ได้ถูกต้อง
func TestIntegration_GroupItemValueFeedsSortOrder(t *testing.T) {
	groupItemMap := seedGroupItemsForSortTest(t)

	cases := []struct {
		code    string
		wantVal float64
		wantHas bool
	}{
		{"PG05_22", 10000, true}, // 1250x8'  value = "10,000.00"
		{"PG05_3", 32, true},     // 4' x 8'  value = "32.00"
		{"PG06_4", 1.2, true},    // 1.2      value = "1.20"
		{"PG06_78", 100, true},   // 100      value = "100.00"
		{"PG03_10", 10, true},    // SS400    value = "10.00"
		{"PG03_21", 21, true},    // LT       value = "21.00"
		{"PG05_40", 0, false},    // ESP      value = ""
	}

	for _, c := range cases {
		gotVal, gotHas := parseGroupItemValue(groupItemMap, c.code)
		if gotVal != c.wantVal || gotHas != c.wantHas {
			t.Fatalf("%s ได้ (%v, %v) ต้องเป็น (%v, %v)", c.code, gotVal, gotHas, c.wantVal, c.wantHas)
		}
	}
}

// seedGroupItemsForSortTest สร้างตาราง group/group_item บน container แล้ว INSERT
// ค่าจริงจาก UAT คืน map แบบเดียวกับที่ getGroupAndItemMappings คืน
func seedGroupItemsForSortTest(t *testing.T) map[string]models.GetGroupItemResponse {
	t.Helper()

	// ใช้ helper ตั้ง container ตัวเดียวกับ weight_spec_integration_test.go
	db := newTestDB(t)

	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS "group" (
			id uuid PRIMARY KEY,
			group_code varchar(255),
			group_name varchar(255),
			value varchar(255),
			value_int double precision,
			seq integer
		);
		CREATE TABLE IF NOT EXISTS group_item (
			id uuid PRIMARY KEY,
			item_code varchar(255),
			group_id uuid,
			item_name varchar(255),
			value varchar(255),
			value_int double precision
		);
	`).Error; err != nil {
		t.Fatalf("สร้างตารางไม่สำเร็จ: %v", err)
	}

	rows := []struct {
		code  string
		name  string
		value string
	}{
		{"PG05_22", "1250x8'", "10,000.00"},
		{"PG05_3", "4' x 8'", "32.00"},
		{"PG05_40", "ESP", ""},
		{"PG06_4", "1.2", "1.20"},
		{"PG06_78", "100", "100.00"},
		{"PG03_10", "SS400", "10.00"},
		{"PG03_21", "LT", "21.00"},
	}

	out := map[string]models.GetGroupItemResponse{}
	for _, r := range rows {
		if err := db.Exec(
			`INSERT INTO group_item (id, item_code, group_id, item_name, value, value_int)
			 VALUES (gen_random_uuid(), ?, gen_random_uuid(), ?, ?, 0)`,
			r.code, r.name, r.value,
		).Error; err != nil {
			t.Fatalf("insert %s ไม่สำเร็จ: %v", r.code, err)
		}
		out[r.code] = models.GetGroupItemResponse{ItemCode: r.code, ItemName: r.name, Value: r.value}
	}

	return out
}
```

> `newTestDB(t)` เป็นชื่อสมมติ — ต้องเปลี่ยนเป็นชื่อ helper ตั้ง container ที่เห็นจริงใน Step 1 ห้ามสร้าง helper ตั้ง container ขึ้นใหม่ซ้ำ ถ้า `weight_spec_integration_test.go` ไม่มี helper แบบนี้ ให้ใช้รูปแบบเดียวกับ `internal/repositories/priceList/repository_test.go` ที่ตั้ง container แล้ว AutoMigrate

- [ ] **Step 3: รัน integration test**

```bash
make test-integration-pricelist
```

Expected: `PASS` — ต้องมี Docker รันอยู่

- [ ] **Step 4: Commit**

```bash
git add internal/services/price-service/pricelist_sort_integration_test.go
git commit -m "test: integration ยืนยันว่าอ่าน group_item.value จาก DB ได้ถูกต้อง

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk"
```

---

## Task 20: ตรวจครบก่อนปิดงาน

**Files:** ไม่มีไฟล์ใหม่

- [ ] **Step 1: build + vet + fmt**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core
go build ./... && gofmt -l . && go vet ./internal/services/price-service/... ./internal/models/...
```

Expected: ไม่มี output ทั้งสามคำสั่ง

> `go vet ./...` ทั้ง repo มี error ค้างอยู่ก่อนแล้ว 12 จุด (`errors.New` ที่ไม่ใช้ผลลัพธ์ ใน customer/authentication/interface/invoice/credit/cronjob service) ซึ่งไม่เกี่ยวกับงานนี้ จึง vet เฉพาะ package ที่แก้

- [ ] **Step 2: รัน test ทั้งหมด**

```bash
make test
```

Expected: ทุก package `ok` — ต้องไม่มี `FAIL`

- [ ] **Step 3: วัด coverage ของ package ที่แก้ (CLAUDE.md กำหนด ≥ 80%)**

```bash
go test ./internal/services/price-service/... -coverprofile=/tmp/cover.out
go tool cover -func=/tmp/cover.out | tail -1
```

Expected: `total:` ≥ 80.0%

ถ้าต่ำกว่า ให้ดูว่าฟังก์ชันไหนยังไม่ถูกครอบ:

```bash
go tool cover -func=/tmp/cover.out | grep -E 'sort_by_value|group_item_value' 
```

แล้วเพิ่มเทสต์ให้ฟังก์ชันที่ยังเป็น `0.0%` จนครบ

- [ ] **Step 4: ตรวจว่าไม่เหลือ string compare ของ product group**

```bash
grep -rn "getValueNameByGroupCode" internal/services/price-service/patterns/*.go | grep -v _test | grep -n "<"
grep -rn "sort.Strings" internal/services/price-service/patterns/*.go | grep -v _test
```

Expected: ไม่มีผลลัพธ์ที่เป็นการเทียบค่าของ product group เหลืออยู่ — `sort.Strings` ที่ยังเหลือได้เฉพาะจุดที่ใช้ทำ "ลำดับตั้งต้นให้นิ่งก่อน `sortLabelsByValue`" เท่านั้น

- [ ] **Step 5: ตรวจว่าไม่มี placeholder หลงเหลือ**

```bash
grep -rn "TODO\|FIXME\|t.Skip\|\.only(" internal/services/price-service/ --include=*.go
```

Expected: ไม่มีผลลัพธ์ที่เกิดจากงานนี้

- [ ] **Step 6: ยืนยันด้วยตาบน UAT**

deploy branch นี้ขึ้น environment ทดสอบ แล้วเปิด
`/customize/price-list/detail/e51e533c-5143-4048-9a4f-07d79ab2be12?groupCode=GROUP_1_ITEM_1`
ตรวจทั้ง 4 tab:

| ตรวจ | ต้องเห็น |
|------|----------|
| แถว tab "เหล็กแผ่น" | `1.2, 1.4, 1.9, 2.3, ... 8, 9, 10, 12, 15, ... 50, 100` |
| หัวคอลัมน์ tab "เหล็กแผ่น" | `4' x 8'`, `5' x 10'`, `5' x 20'`, `4'x1500`, `4'x2400`, `1250x8'`, `5'x5700` |
| หัวคอลัมน์ tab "ตัด SIZE" | `75x75` มาก่อน `300x300` และ `40x520` อยู่ระหว่าง `125x125` กับ `150x150` |
| ลำดับ tab | `เหล็กแผ่น`, `แผ่นลาย`, `เหล็กแผ่น special`, `เหล็กแผ่นตัด SIZE` |

- [ ] **Step 7: เปิด PR เข้า Develop**

```bash
git push -u origin feat/pricelist-sort-by-group-value
gh pr create --base Develop --title "fix: เรียงลำดับ price list detail ด้วย group_item.value" --body "$(cat <<'BODY'
## ปัญหา
หน้า Price List Detail เรียงทุกแกนด้วย lexicographic string sort ทำให้แถว "แผ่น mm."
แสดง `1.2, 1.4, 1.9, 10, 100, 12` และหัวคอลัมน์ขนาดแสดง `1250x8', 4' x 8', 4'x1500`
แต่ละ tab ก็เรียงไม่เหมือนกันเพราะโลจิกถูกคัดลอกซ้ำ ~23 จุดใน 13 pattern

## การแก้
ใช้ `group_item.value` ที่เก็บค่าตัวเลขไว้ครบอยู่แล้วเป็นแหล่งความจริงของลำดับ
ครอบคลุมทั้งขนาดที่ปนหน่วย (`4' x 8'` = 32 กับ `4'x1500` = 6,000) และเกรดเหล็ก
ที่ไม่ใช่ตัวเลข (`SS400` = 10, `LT` = 21) ซึ่ง natural sort แก้ไม่ได้

เรียง `subGroups` ที่ต้นทางจาก `SubGroupKeys` โดยตรง ไม่แปลงชื่อกลับเป็นตัวเลข
เพราะ `buildCompositeKeyBy` ข้ามค่าว่างตอน join และ `columnKey` ผ่าน
`sanitizeIdentifier` มาอีกชั้น จึง reverse ไม่ได้จริง

## ผลข้างเคียงที่ตั้งใจ
- ลำดับ tab เปลี่ยนเป็น `เหล็กแผ่น(6) → แผ่นลาย(9) → special(19) → ตัด SIZE(20)`
- `PG05.value` เป็นพื้นที่ (ผลคูณสองด้าน) กลุ่มหน่วยฟุตจึงมาก่อนกลุ่มมิลลิเมตร
  และ `40x520` (20,800) แทรกระหว่าง `125x125` (15,625) กับ `150x150` (22,500)
  ถ้าลำดับไหนไม่ถูกใจ business แก้ที่ master ได้โดยไม่ต้องแตะโค้ด

## ไม่ได้แก้
frontend, `value_int`, `SyncGroupMaster` และจุด sort ที่ใช้ `row_number` /
`total_weight` / `ship_no` ซึ่งไม่ได้มาจาก product group

Spec: `docs/superpowers/specs/2026-09-11-pricelist-sort-by-group-value-design.md`
Plan: `docs/superpowers/plans/2026-09-11-pricelist-sort-by-group-value.md`

🤖 Generated with [Claude Code](https://claude.com/claude-code)

https://claude.ai/code/session_017tAQ8sPrGH4Wq7MksZPrLk
BODY
)"
```

Expected: PR ถูกสร้างเข้า `Develop` — **ห้าม merge เข้า Develop เอง** ตาม CLAUDE.md

- [ ] **Step 8: อัปเดต knowledge graph**

```bash
cd /home/chonlatee/Desktop/Work/prime-wms && graphify update .
```

Expected: graph อัปเดตสำเร็จ
