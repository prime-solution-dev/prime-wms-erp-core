# แผน implementation แก้ issue Price List 6 ชุด

> **สำหรับ agentic worker:** REQUIRED SUB-SKILL: ใช้ `superpowers:subagent-driven-development`
> (แนะนำ) หรือ `superpowers:executing-plans` ในการทำแผนนี้ทีละ task
> ทุกขั้นใช้ checkbox (`- [ ]`) เพื่อติดตามความคืบหน้า

**Goal:** แก้บั๊ก Price List 8 จุดที่ยืนยันแล้ว แยกเป็น 6 PR ที่ merge ได้อิสระ

**Architecture:** ทั้ง 6 ชุดแก้ที่รากในจุดที่ caller ทุกตัววิ่งผ่าน ไม่ปะที่ปลายทาง
สามชุด (A, E-4c, F) เป็นการแก้ config หรือบรรทัดเดียว · สองชุด (C, D) ต้องแยกฟังก์ชันออกมา
ให้ทดสอบได้ก่อนแก้พฤติกรรม · ชุด B เป็นการเพิ่ม sort ให้ผลลัพธ์ deterministic

**Tech Stack:** Go 1.22+ (Gin, GORM, sqlx, testcontainers), Vue 3 + TypeScript (Vitest, AG Grid)

**Spec:** `docs/superpowers/specs/2026-09-11-pricelist-issues-triage-design.md`

---

## ข้อบังคับที่ใช้กับทุก task

- **ห้าม merge หรือแก้ `Develop` ตรง ๆ** ต้องแตก branch ใหม่จาก `origin/Develop` เสมอ
- **ห้ามยุ่ง branch**: `Crossmax-uat`, `shi-sit`, `Pacifica-uat`, `Pacifica-main`,
  `Thaimetal-uat`, `shi-main`
- **ห้าม `git add -A` หรือ `git add .`** ต้องระบุไฟล์ทีละตัว —
  `prime-wms-warehouse-core/cmd/.env` มีสถานะ Modified ค้างอยู่ ถูก track ใน git
  และมี credential ข้างใน **ห้าม commit ห้าม checkout ทับ**
- **ห้ามรัน write ลง database** SELECT ได้เท่านั้น
- **ห้ามรัน `gofmt -w` ทั้งไฟล์** repo มี pre-existing violation อยู่หลายไฟล์
  (`get-price-export-table.go`, `shared_test.go`) จะทำให้ diff บวม ·
  จัดรูปแบบเฉพาะบรรทัดที่แก้
- **ห้ามแทรกคอมเมนต์กลาง struct literal** เพราะ gofmt จะจัดแนว field ใหม่ทั้งบล็อก
- **ห้ามแก้ `CLAUDE.md` ที่ root**
- polyrepo: ต้อง `cd` เข้า service ก่อนรันคำสั่ง go
- coverage ≥ 80% ต่อ package ที่แตะ

### คำสั่งรัน test

```bash
# erp-core
cd prime-wms-erp-core
go test ./...                                              # unit ทั้งหมด
make test-integration-pricelist                            # integration (testcontainers postgres:16, ~6 วิ)
go test -v ./internal/services/price-service/patterns -run TestName

# warehouse-core
cd prime-wms-warehouse-core
go test ./internal/services/inventory-service/

# web
cd prime-wms-web
npm run test                                               # vitest run
```

---

## โครงไฟล์

| PR | ไฟล์ที่แก้ | ไฟล์ test |
|----|-----------|----------|
| A | `internal/services/price-service/patterns/configs/GROUP_1_ITEM_9_PATTERN.json` | `internal/services/price-service/patterns/group_1_item_9_row_key_test.go` (สร้างใหม่) |
| B | `prime-wms-warehouse-core/internal/services/inventory-service/get-inventory-weight-by-key.go`<br>`internal/services/price-service/get-price-detail.go` | `prime-wms-warehouse-core/internal/services/inventory-service/get-inventory-weight-order_test.go` (สร้างใหม่)<br>`internal/services/price-service/get-price-detail-order_test.go` (สร้างใหม่) |
| C | `internal/repositories/priceList/repository.go` | `internal/repositories/priceList/subgroup_update_map_test.go` (สร้างใหม่) |
| D | `internal/utils/request-handler.go`<br>`internal/services/price-service/update-pricelist.go`<br>`prime-wms-web/src/views/price-list/Extra.vue` | `internal/utils/request-handler_test.go` (สร้างใหม่)<br>`internal/services/price-service/update-extras_validation_test.go` (สร้างใหม่)<br>`prime-wms-web/src/views/price-list/extraValidation.spec.ts` (สร้างใหม่) |
| E | `prime-wms-web/src/utils/helper/priceListNumberFormat.ts`<br>`internal/services/price-service/get-price-export-table.go`<br>`internal/services/price-service/build-pricelist-detail-tab.go`<br>config JSON 9 ไฟล์ | `prime-wms-web/src/utils/helper/priceListNumberFormat.spec.ts` (สร้างใหม่)<br>`internal/services/price-service/build-pricelist-detail-tab_test.go` (แก้ + เพิ่ม) |
| F | `internal/services/price-service/upload-pricelist.go` | `internal/services/price-service/upload-pricelist_parse_test.go` (แก้ + เพิ่ม) |

---

# PR A — `GROUP_1_ITEM_9` row key ขาด `PG06`

**branch:** `fix/pricelist-item9-row-key` (มีอยู่แล้ว มี commit เอกสาร 6 ตัว)

ความเสียหาย: 80 subgroup ยุบเหลือ 4 แถว ข้อมูลหาย 76 แถว — หนักสุดในชุดนี้

### Task A1: เขียน test ที่ fail ก่อนแก้

**Files:**
- Create: `internal/services/price-service/patterns/group_1_item_9_row_key_test.go`

- [ ] **Step 1: เขียน test ที่ fail**

```go
package patterns

import (
	"testing"

	"prime-erp-core/internal/models"
)

// subgroup ที่ต่างกันแค่ PG06 ต้องเป็นคนละแถว
// PG06 เป็น pinned fixedColumn ระดับแถวของ GROUP_1_ITEM_9 แต่ไม่ได้อยู่ใน grouping.rows
// จึงถูกยุบเข้าแถวเดียวกันแบบ last-write-wins ทำให้ข้อมูลหาย
func TestGroup1Item9_SubGroupsDifferingOnlyByPG06_StayInSeparateRows(t *testing.T) {
	cfg, err := LoadConfiguration("GROUP_1_ITEM_9")
	if err != nil {
		t.Fatalf("โหลด config ไม่ได้: %v", err)
	}
	if len(cfg.Patterns) == 0 {
		t.Fatal("config ไม่มี pattern เลย")
	}
	pattern := &cfg.Patterns[0]

	subGroups := []models.PriceListSubGroupResponse{
		item9SubGroup("sg-1", "PG06_1", "ความยาว 6 เมตร", 101),
		item9SubGroup("sg-2", "PG06_2", "ความยาว 9 เมตร", 202),
	}

	rows := buildDynamicRows(cfg, pattern, subGroups)
	merged := mergeGroup1Item9Rows(rows)

	if len(merged) != 2 {
		t.Fatalf("ต้องได้ 2 แถว (คนละ PG06) แต่ได้ %d แถว — ข้อมูลถูกยุบทับกัน", len(merged))
	}

	// ยืนยันว่าราคาของทั้งสอง subgroup ยังอยู่ ไม่ถูกทับ
	seen := map[float64]bool{}
	for _, row := range merged {
		for key, value := range row {
			if v, ok := value.(float64); ok && (v == 101 || v == 202) {
				seen[v] = true
				_ = key
			}
		}
	}
	if !seen[101] || !seen[202] {
		t.Fatalf("ราคาของ subgroup ถูกทับหาย: เห็น %v ต้องเห็นทั้ง 101 และ 202", seen)
	}
}

// item9SubGroup สร้าง subgroup ที่มี key ครบตามที่ GROUP_1_ITEM_9 ต้องใช้
// PG02 กับ PG07 เหมือนกันทุกตัว ต่างกันแค่ PG06 เพื่อแยกให้ชัดว่า PG06 คือมิติที่หายไป
// buildCompositeKey อ่านจาก ValueName ส่วน buildCompositeCodeKey อ่านจาก ValueCode
// จึงต้องตั้งทั้งสอง field
func item9SubGroup(id, pg06Code, pg06Name string, priceWeight float64) models.PriceListSubGroupResponse {
	return models.PriceListSubGroupResponse{
		ID:                  id,
		SubgroupCode:        id,
		TotalNetPriceWeight: priceWeight,
		SubGroupKeys: []models.PriceListSubGroupKeyResponse{
			{GroupCode: "PG02", ValueCode: "PG02_10", ValueName: "หมวดเหล็กเส้น", Seq: 2},
			{GroupCode: "PG07", ValueCode: "PG07_8", ValueName: "ขนาด 12 มม.", Seq: 7},
			{GroupCode: "PG06", ValueCode: pg06Code, ValueName: pg06Name, Seq: 6},
			{GroupCode: "PG03", ValueCode: "PG03_1", ValueName: "เกรด SD40", Seq: 3},
		},
	}
}
```

- [ ] **Step 2: รัน test ให้เห็นว่า fail**

Run:
```bash
cd prime-wms-erp-core
go test -v ./internal/services/price-service/patterns -run TestGroup1Item9_SubGroupsDifferingOnlyByPG06
```
Expected: FAIL — `ต้องได้ 2 แถว (คนละ PG06) แต่ได้ 1 แถว — ข้อมูลถูกยุบทับกัน`

ถ้า test **ผ่าน** ตั้งแต่ต้น ให้หยุดและรายงาน เพราะแปลว่าสมมติฐานผิด ห้ามแก้ config

- [ ] **Step 3: Commit test ก่อนแก้**

```bash
cd prime-wms-erp-core
git add internal/services/price-service/patterns/group_1_item_9_row_key_test.go
git commit -m "test: ตรึงว่า GROUP_1_ITEM_9 ต้องไม่ยุบ subgroup ที่ต่างกันแค่ PG06"
```

### Task A2: แก้ config

**Files:**
- Modify: `internal/services/price-service/patterns/configs/GROUP_1_ITEM_9_PATTERN.json`

- [ ] **Step 1: เพิ่ม PG06 เข้า row key**

เปลี่ยนจาก

```json
      "grouping": {
        "tabs": "",
        "rows": "PG02|PG07",
        "columnGroups": "PG03"
      },
```

เป็น

```json
      "grouping": {
        "tabs": "",
        "rows": "PG02|PG07|PG06",
        "columnGroups": "PG03"
      },
```

แก้บรรทัดเดียว ไม่ต้องแตะ Go เลย เพราะ `shared.go:1124` ตั้ง
`row["row_group_value"] = rowKey` และ `shared.go:1131-1133` เขียนคอลัมน์จาก `rowFields`
ทั้งคู่อ่านจาก `pattern.Grouping.Rows` ตัวเดียวกัน การแก้จุดนี้จึงคุมทั้ง merge key
และการป้อนข้อมูลคอลัมน์ PG06 ที่ตอนนี้ว่างอยู่

- [ ] **Step 2: รัน test ให้ผ่าน**

Run:
```bash
cd prime-wms-erp-core
go test -v ./internal/services/price-service/patterns -run TestGroup1Item9_SubGroupsDifferingOnlyByPG06
```
Expected: PASS

- [ ] **Step 3: รัน test ทั้ง package กัน regression**

Run:
```bash
cd prime-wms-erp-core
go test ./internal/services/price-service/...
```
Expected: ok ทุก package

ถ้ามี test อื่นพัง ให้อ่านว่า test นั้นตรึงพฤติกรรม "ยุบแถว" ไว้หรือไม่ ถ้าใช่ต้องแก้ test
นั้นพร้อมอธิบายเหตุผลใน commit message ห้ามลบ test ทิ้งเฉย ๆ

- [ ] **Step 4: ยืนยันกับข้อมูลจริง (SELECT เท่านั้น)**

Run query ข้อ 5 ของ
`docs/superpowers/reports/2026-09-11-pricelist-term-percent-convention.sql`

Expected: `subgroup ทั้งหมด = 80` · `row key = PG02|PG07|PG06 → 21` ·
`cell ที่ทับกัน หลังแก้ = 0`

- [ ] **Step 5: Commit**

```bash
cd prime-wms-erp-core
git add internal/services/price-service/patterns/configs/GROUP_1_ITEM_9_PATTERN.json
git commit -m "fix: เพิ่ม PG06 เข้า row key ของ GROUP_1_ITEM_9

PG06 เป็น pinned fixedColumn ระดับแถวแต่ไม่ได้อยู่ใน grouping.rows
ทำให้ subgroup ที่ต่างกันแค่ PG06 ถูกยุบเข้าแถวเดียวกันแบบ last-write-wins
80 subgroup ยุบเหลือ 4 แถว มี 12 cell ที่ทับกัน หลังแก้ได้ 21 แถวและไม่มี cell ทับกัน
คอลัมน์ PG06 ที่ pinned ไว้ก็จะมีข้อมูลป้อนด้วย เพราะ shared.go เขียนเฉพาะ field ใน rowFields"
```

- [ ] **Step 6: เปิด PR**

```bash
cd prime-wms-erp-core
git push -u origin fix/pricelist-item9-row-key
gh pr create --base Develop --title "fix: GROUP_1_ITEM_9 ยุบ subgroup ที่ต่างกันแค่ PG06 ทำข้อมูลหาย 76 จาก 80 แถว"
```

---

# PR B — ลำดับผลลัพธ์ไม่นิ่งเพราะวน map

**branch (warehouse-core):** `fix/pricelist-deterministic-order`
**branch (erp-core):** `fix/pricelist-deterministic-order`

แยก 2 PR เพราะคนละ repo · ฝั่ง web ไม่ต้องแตะ เพราะ `BasePriceTable.vue:341-353`
filter อย่างเดียวไม่ sort เอง ให้ backend เป็นแหล่งความจริงเดียว

### Task B1: warehouse-core — sort `keyValueGroups`

**Files:**
- Create: `prime-wms-warehouse-core/internal/services/inventory-service/get-inventory-weight-order_test.go`
- Modify: `prime-wms-warehouse-core/internal/services/inventory-service/get-inventory-weight-by-key.go:113`

- [ ] **Step 1: แตก branch**

```bash
cd prime-wms-warehouse-core
git status --short --branch          # ต้องเห็น cmd/.env เป็น M — ปล่อยไว้ ห้ามแตะ
git fetch origin
git checkout -b fix/pricelist-deterministic-order origin/Develop
```

- [ ] **Step 2: เขียน test ที่ fail**

`keyValueGroups` ประกอบจาก `for id, items := range idGroups` ซึ่งวน map จึงสุ่มลำดับ
test นี้เรียกซ้ำหลายรอบและเทียบว่าลำดับเหมือนกันทุกรอบ

```go
package inventoryService

import (
	"strings"
	"testing"
)

// การวน map ใน Go สุ่มลำดับ ฉะนั้นลำดับของ keyValueGroups ต้องถูก sort ให้นิ่ง
// ไม่งั้นผลที่ส่งกลับไปหน้าจอจะเรียงต่างกันทุกครั้งที่เรียก
func TestBuildKeyValueGroups_IsDeterministic(t *testing.T) {
	items := []KeyValueItem{
		{ID: "id-3", GroupCode: "PG02", GroupValue: "PG02_3", Seq: 2},
		{ID: "id-1", GroupCode: "PG02", GroupValue: "PG02_1", Seq: 2},
		{ID: "id-2", GroupCode: "PG02", GroupValue: "PG02_2", Seq: 2},
		{ID: "id-1", GroupCode: "PG01", GroupValue: "PG01_1", Seq: 1},
		{ID: "id-2", GroupCode: "PG01", GroupValue: "PG01_2", Seq: 1},
		{ID: "id-3", GroupCode: "PG01", GroupValue: "PG01_3", Seq: 1},
	}

	var first string
	for round := 0; round < 50; round++ {
		groups := buildKeyValueGroups(items)

		ids := make([]string, 0, len(groups))
		for _, g := range groups {
			ids = append(ids, g.ID)
		}
		got := strings.Join(ids, ",")

		if round == 0 {
			first = got
			continue
		}
		if got != first {
			t.Fatalf("รอบที่ %d ได้ลำดับ %q ต่างจากรอบแรก %q — ลำดับไม่นิ่ง", round, got, first)
		}
	}

	if first != "id-1,id-2,id-3" {
		t.Fatalf("ต้อง sort ตาม ID ได้ %q ต้องเป็น \"id-1,id-2,id-3\"", first)
	}
}
```

- [ ] **Step 3: รัน test ให้เห็นว่า fail (compile error)**

Run:
```bash
cd prime-wms-warehouse-core
go test ./internal/services/inventory-service/ -run TestBuildKeyValueGroups_IsDeterministic
```
Expected: FAIL — `undefined: buildKeyValueGroups` และ `undefined: KeyValueGroup`

ตอนนี้ `KeyValueGroup` ประกาศเป็น type ข้างในฟังก์ชัน จึงมองจาก test ไม่เห็น
ขั้นต่อไปต้องยกออกมาระดับ package

- [ ] **Step 4: ยก `KeyValueGroup` และตรรกะการ group ออกมาเป็นฟังก์ชันระดับ package**

ใน `get-inventory-weight-by-key.go` ลบการประกาศ type ที่อยู่ข้างในฟังก์ชัน

```go
	// Group key_value items by id
	type KeyValueGroup struct {
		ID             string
		Items          []KeyValueItem
		GroupCodeKeys  string
		GroupValueKeys string
	}
```

แล้วย้ายไปไว้ระดับ package (วางไว้ก่อนฟังก์ชันที่ใช้) พร้อมฟังก์ชันใหม่

```go
// KeyValueGroup คือ key_value ที่จัดกลุ่มตาม ID แล้ว
type KeyValueGroup struct {
	ID             string
	Items          []KeyValueItem
	GroupCodeKeys  string
	GroupValueKeys string
}

// buildKeyValueGroups จัดกลุ่ม key_value ตาม ID แล้วคืนผลที่เรียงตาม ID
//
// การวน map ใน Go สุ่มลำดับ ถ้าไม่ sort ชั้นนอก ลำดับที่ส่งกลับไปหน้าจอจะเปลี่ยน
// ทุกครั้งที่เรียก ผู้ใช้จึงเห็นรายการสลับที่หลังกดบันทึกแล้วโหลดใหม่
func buildKeyValueGroups(keyValue []KeyValueItem) []KeyValueGroup {
	idGroups := make(map[string][]KeyValueItem)
	for _, kv := range keyValue {
		idGroups[kv.ID] = append(idGroups[kv.ID], kv)
	}

	ids := make([]string, 0, len(idGroups))
	for id := range idGroups {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	groups := make([]KeyValueGroup, 0, len(ids))
	for _, id := range ids {
		items := idGroups[id]

		sortedItems := make([]KeyValueItem, len(items))
		copy(sortedItems, items)
		sort.Slice(sortedItems, func(i, j int) bool {
			return sortedItems[i].Seq < sortedItems[j].Seq
		})

		var groupCodes []string
		var groupValues []string
		for _, item := range sortedItems {
			groupCodes = append(groupCodes, item.GroupCode)
			groupValues = append(groupValues, item.GroupValue)
		}

		groups = append(groups, KeyValueGroup{
			ID:             id,
			Items:          sortedItems,
			GroupCodeKeys:  strings.Join(groupCodes, "|"),
			GroupValueKeys: strings.Join(groupValues, "|"),
		})
	}

	return groups
}
```

แล้วในฟังก์ชันเดิม แทนบล็อกที่ประกอบ `idGroups` และวน `for id, items := range idGroups`
ทั้งหมดด้วยบรรทัดเดียว

```go
	keyValueGroups := buildKeyValueGroups(req.KeyValue)
```

`sort` และ `strings` ถูก import อยู่แล้วในไฟล์นี้ (บรรทัด 8 และ 9) ไม่ต้องเพิ่ม

- [ ] **Step 5: รัน test ให้ผ่าน**

Run:
```bash
cd prime-wms-warehouse-core
go test ./internal/services/inventory-service/ -run TestBuildKeyValueGroups_IsDeterministic -v
```
Expected: PASS

- [ ] **Step 6: รัน test ทั้ง package และ build**

Run:
```bash
cd prime-wms-warehouse-core
go build ./... && go test ./internal/services/inventory-service/
```
Expected: build ผ่าน · test ok

- [ ] **Step 7: Commit (ระบุไฟล์ทีละตัว ห้าม add cmd/.env)**

```bash
cd prime-wms-warehouse-core
git add internal/services/inventory-service/get-inventory-weight-by-key.go
git add internal/services/inventory-service/get-inventory-weight-order_test.go
git status --short          # ยืนยันว่า cmd/.env ยังเป็น M ไม่ได้ staged
git commit -m "fix: sort keyValueGroups ให้ลำดับนิ่ง

keyValueGroups ประกอบจากการวน map idGroups ซึ่ง Go สุ่มลำดับ ทำให้ผลที่ส่งกลับ
เรียงต่างกันทุกครั้งที่เรียก ผู้ใช้เห็นรายการสลับที่หลังกดบันทึกแล้วโหลดใหม่
ยก KeyValueGroup ออกมาระดับ package พร้อมฟังก์ชัน buildKeyValueGroups เพื่อให้ทดสอบได้"
```

- [ ] **Step 8: เปิด PR**

```bash
cd prime-wms-warehouse-core
git push -u origin fix/pricelist-deterministic-order
gh pr create --base Develop --title "fix: ทำลำดับ keyValueGroups ให้ deterministic"
```

### Task B2: erp-core — sort `companyCodes` และ `siteCodes`

**Files:**
- Create: `internal/services/price-service/get-price-detail-order_test.go`
- Modify: `internal/services/price-service/get-price-detail.go:262-270` และ import block

- [ ] **Step 1: แตก branch**

```bash
cd prime-wms-erp-core
git fetch origin
git checkout -b fix/pricelist-deterministic-order origin/Develop
```

- [ ] **Step 2: เขียน test ที่ fail**

`companyCodeSet` และ `siteCodeSet` ถูกแปลงเป็น slice ด้วยการวน map โดยไม่ sort
และโค้ดใช้ `companyCodes[0]` เป็นค่าที่ส่งไป inventory service
ฉะนั้นถ้ามีหลาย company ค่าที่เลือกจะเปลี่ยนไปเรื่อย ๆ

```go
package priceService

import (
	"strings"
	"testing"
)

// companyCodeSet และ siteCodeSet ถูกแปลงเป็น slice ด้วยการวน map ซึ่ง Go สุ่มลำดับ
// get-price-detail ใช้ companyCodes[0] เป็นค่าที่ส่งไป inventory service
// ถ้าไม่ sort ค่าที่ถูกเลือกจะเปลี่ยนทุกครั้งที่เรียก
func TestSortedSetKeys_IsDeterministic(t *testing.T) {
	set := map[string]bool{
		"C003": true,
		"C001": true,
		"C002": true,
	}

	var first string
	for round := 0; round < 50; round++ {
		got := strings.Join(sortedSetKeys(set), ",")
		if round == 0 {
			first = got
			continue
		}
		if got != first {
			t.Fatalf("รอบที่ %d ได้ %q ต่างจากรอบแรก %q — ลำดับไม่นิ่ง", round, got, first)
		}
	}

	if first != "C001,C002,C003" {
		t.Fatalf("ต้อง sort ได้ %q ต้องเป็น \"C001,C002,C003\"", first)
	}
}

func TestSortedSetKeys_Empty(t *testing.T) {
	if got := sortedSetKeys(map[string]bool{}); len(got) != 0 {
		t.Fatalf("set ว่างต้องได้ slice ว่าง แต่ได้ %v", got)
	}
}
```

- [ ] **Step 3: รัน test ให้เห็นว่า fail**

Run:
```bash
cd prime-wms-erp-core
go test ./internal/services/price-service/ -run TestSortedSetKeys
```
Expected: FAIL — `undefined: sortedSetKeys`

- [ ] **Step 4: เพิ่ม helper และเรียกใช้**

เพิ่ม `"sort"` เข้า import block ของ `get-price-detail.go` (ตอนนี้ยังไม่มี)

```go
import (
	"encoding/json"
	"fmt"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	groupService "prime-erp-core/internal/services/group-service"
	priceDomain "prime-erp-core/internal/services/price-service/domain"
	pricePatterns "prime-erp-core/internal/services/price-service/patterns"
	"sort"
	"time"

	externalService "prime-erp-core/external/warehouse-service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)
```

เพิ่มฟังก์ชัน helper ท้ายไฟล์

```go
// sortedSetKeys คืน key ของ set ที่เรียงแล้ว
//
// การวน map ใน Go สุ่มลำดับ ผู้เรียกใช้ผลนี้เลือก element ตัวแรกไปส่งต่อ
// ถ้าไม่ sort ค่าที่ถูกเลือกจะเปลี่ยนทุกครั้งที่เรียก
func sortedSetKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
```

แทนบล็อกเดิม (บรรทัด 262-270)

```go
		// Convert sets to slices
		companyCodes := []string{}
		for code := range companyCodeSet {
			companyCodes = append(companyCodes, code)
		}
		siteCodes := []string{}
		for code := range siteCodeSet {
			siteCodes = append(siteCodes, code)
		}
```

ด้วย

```go
		// Convert sets to slices — sort เพื่อให้ companyCodes[0] และลำดับ siteCodes นิ่ง
		companyCodes := sortedSetKeys(companyCodeSet)
		siteCodes := sortedSetKeys(siteCodeSet)
```

- [ ] **Step 5: รัน test ให้ผ่าน**

Run:
```bash
cd prime-wms-erp-core
go test -v ./internal/services/price-service/ -run TestSortedSetKeys
```
Expected: PASS ทั้งสอง test

- [ ] **Step 6: build และรัน test ทั้ง package**

Run:
```bash
cd prime-wms-erp-core
go build ./... && go test ./internal/services/price-service/...
```
Expected: build ผ่าน · test ok

- [ ] **Step 7: Commit และเปิด PR**

```bash
cd prime-wms-erp-core
git add internal/services/price-service/get-price-detail.go
git add internal/services/price-service/get-price-detail-order_test.go
git commit -m "fix: sort companyCodes และ siteCodes ใน get-price-detail

ทั้งสอง slice ถูกประกอบจากการวน map ซึ่ง Go สุ่มลำดับ และโค้ดใช้ companyCodes[0]
เป็นค่าที่ส่งไป inventory service ค่าที่ถูกเลือกจึงเปลี่ยนทุกครั้งที่เรียก
เพิ่ม helper sortedSetKeys เพื่อให้ทดสอบได้และใช้ซ้ำได้"
git push -u origin fix/pricelist-deterministic-order
gh pr create --base Develop --title "fix: ทำลำดับ companyCodes และ siteCodes ให้ deterministic"
```

---

# PR C — `before_*` ถูกเลื่อนทุกครั้งแม้ค่าไม่เปลี่ยน

**branch:** `fix/pricelist-before-price-snapshot`

ความเสียหาย: 1,257 จาก 1,350 subgroup มี `before = after` แล้ว (93%)

`repository.go:490-516` เขียน `before_*` ถูกอยู่แล้ว แต่ไม่มีเงื่อนไขว่าค่าต้องเปลี่ยนจริง
และ `update-latest-pricelist-subgroup.go:321-324` ส่ง pointer ของ `TotalNetPrice*`
ทุก subgroup ทุกครั้ง ประกอบกับ `update_type = "group"` ที่ดึงทั้งกลุ่มมาคำนวณ
จึงทำให้ snapshot ถูกทับเป็นวงกว้าง

แก้ที่ `repository.go` ไม่ใช่ที่ `update-latest` เพราะ repository เป็นจุดที่ caller
ทุกตัววิ่งผ่าน แก้ที่นั่นครอบทั้ง `update-latest` และ `update-pricelist-subgroup` ในทีเดียว

### Task C1: แยกตรรกะ `updateMap` ออกมาเป็นฟังก์ชันบริสุทธิ์ (refactor ไม่เปลี่ยนพฤติกรรม)

**Files:**
- Create: `internal/repositories/priceList/subgroup_update_map_test.go`
- Modify: `internal/repositories/priceList/repository.go:480-520`

- [ ] **Step 1: แตก branch**

```bash
cd prime-wms-erp-core
git fetch origin
git checkout -b fix/pricelist-before-price-snapshot origin/Develop
```

- [ ] **Step 2: เขียน characterization test ที่ตรึงพฤติกรรม*ปัจจุบัน* ไว้ก่อน**

test นี้ต้องผ่านทันทีหลัง refactor เพื่อพิสูจน์ว่า refactor ไม่เปลี่ยนพฤติกรรม
ยังไม่ใช่ test ของบั๊ก

```go
package priceList

import (
	"testing"

	"prime-erp-core/internal/models"
)

func floatPtr(v float64) *float64 { return &v }

// ตรึงพฤติกรรมเดิม: เมื่อค่าใหม่ต่างจากค่าเดิม ต้องเลื่อน before_* มาเก็บค่าเดิม
func TestBuildSubGroupUpdateMap_ShiftsBeforeWhenValueChanges(t *testing.T) {
	old := models.PriceListSubGroup{
		TotalNetPriceWeight: 18.50,
		TotalNetPriceUnit:   1200,
	}
	item := models.UpdatePriceListSubGroupItem{
		TotalNetPriceWeight: floatPtr(19.75),
		TotalNetPriceUnit:   floatPtr(1300),
	}

	got := buildSubGroupUpdateMap(old, item)

	if got["total_net_price_weight"] != 19.75 {
		t.Fatalf("total_net_price_weight = %v, want 19.75", got["total_net_price_weight"])
	}
	if got["before_total_net_price_weight"] != 18.50 {
		t.Fatalf("before_total_net_price_weight = %v, want 18.50", got["before_total_net_price_weight"])
	}
	if got["total_net_price_unit"] != float64(1300) {
		t.Fatalf("total_net_price_unit = %v, want 1300", got["total_net_price_unit"])
	}
	if got["before_total_net_price_unit"] != float64(1200) {
		t.Fatalf("before_total_net_price_unit = %v, want 1200", got["before_total_net_price_unit"])
	}
}

// ตรึงพฤติกรรมเดิม: field ที่ req ไม่ได้ส่งมา (nil) ต้องไม่ปรากฏใน updateMap เลย
func TestBuildSubGroupUpdateMap_SkipsNilFields(t *testing.T) {
	old := models.PriceListSubGroup{PriceWeight: 10, TotalNetPriceWeight: 20}
	item := models.UpdatePriceListSubGroupItem{TotalNetPriceWeight: floatPtr(21)}

	got := buildSubGroupUpdateMap(old, item)

	if _, exists := got["price_weight"]; exists {
		t.Fatal("price_weight ไม่ควรอยู่ใน updateMap เพราะ req ไม่ได้ส่งมา")
	}
	if _, exists := got["before_price_weight"]; exists {
		t.Fatal("before_price_weight ไม่ควรอยู่ใน updateMap เพราะ req ไม่ได้ส่งมา")
	}
}
```

- [ ] **Step 3: รัน test ให้เห็นว่า fail**

Run:
```bash
cd prime-wms-erp-core
go test ./internal/repositories/priceList/ -run TestBuildSubGroupUpdateMap
```
Expected: FAIL — `undefined: buildSubGroupUpdateMap`

- [ ] **Step 4: แยกฟังก์ชันออกมา (คัดลอกตรรกะเดิมแบบคำต่อคำ ห้ามเปลี่ยนพฤติกรรม)**

เพิ่มฟังก์ชันใหม่ใน `repository.go`

```go
// buildSubGroupUpdateMap ประกอบ map สำหรับ UPDATE จากค่าเดิมและค่าที่ request ส่งมา
//
// field ที่ request ไม่ได้ส่งมา (nil) จะไม่ถูกใส่ใน map เลย เพื่อไม่ให้ GORM
// เขียนทับด้วย zero value
func buildSubGroupUpdateMap(oldSubGroup models.PriceListSubGroup, req models.UpdatePriceListSubGroupItem) map[string]interface{} {
	updateMap := make(map[string]interface{})

	if req.PriceUnit != nil {
		updateMap["before_price_unit"] = oldSubGroup.PriceUnit
		updateMap["price_unit"] = *req.PriceUnit
	}
	if req.ExtraPriceUnit != nil {
		updateMap["before_extra_price_unit"] = oldSubGroup.ExtraPriceUnit
		updateMap["extra_price_unit"] = *req.ExtraPriceUnit
	}
	if req.TotalNetPriceUnit != nil {
		updateMap["before_total_net_price_unit"] = oldSubGroup.TotalNetPriceUnit
		updateMap["total_net_price_unit"] = *req.TotalNetPriceUnit
	}
	if req.PriceWeight != nil {
		updateMap["before_price_weight"] = oldSubGroup.PriceWeight
		updateMap["price_weight"] = *req.PriceWeight
	}
	if req.ExtraPriceWeight != nil {
		updateMap["before_extra_price_weight"] = oldSubGroup.ExtraPriceWeight
		updateMap["extra_price_weight"] = *req.ExtraPriceWeight
	}
	if req.TermPriceWeight != nil {
		updateMap["before_term_price_weight"] = oldSubGroup.TermPriceWeight
		updateMap["term_price_weight"] = *req.TermPriceWeight
	}
	if req.TotalNetPriceWeight != nil {
		updateMap["before_total_net_price_weight"] = oldSubGroup.TotalNetPriceWeight
		updateMap["total_net_price_weight"] = *req.TotalNetPriceWeight
	}

	return updateMap
}
```

ใน `UpdatePriceListSubGroups` แทนบล็อกที่ประกอบ `updateMap` ทั้งหมด (ตั้งแต่
`updateMap := make(map[string]interface{})` ถึงบรรทัดสุดท้ายของ
`if req.TotalNetPriceWeight != nil { ... }`) ด้วยบรรทัดเดียว

```go
			updateMap := buildSubGroupUpdateMap(oldSubGroup, req)
```

**หมายเหตุ**: ชื่อตัวแปรในฟังก์ชันเดิมอาจไม่ใช่ `req` ตรง ๆ (เป็น element ของ
`reqs.Changes`) ให้ใช้ชื่อตัวแปรที่มีอยู่จริงในบริบทนั้น อ่านโค้ดรอบ ๆ ก่อนแก้

ส่วนที่เหลือของฟังก์ชัน (การอ่าน `oldSubGroup`, การสร้าง `historyRecord`,
การเรียก `tx.Model(...).Updates(updateMap)`) ห้ามแตะ

- [ ] **Step 5: รัน test ให้ผ่าน และยืนยันว่าไม่มี regression**

Run:
```bash
cd prime-wms-erp-core
go test -v ./internal/repositories/priceList/ -run TestBuildSubGroupUpdateMap
go build ./... && go test ./...
```
Expected: test ใหม่ PASS · build ผ่าน · test เดิมทั้งหมด ok

- [ ] **Step 6: Commit refactor แยกจากการแก้บั๊ก**

```bash
cd prime-wms-erp-core
git add internal/repositories/priceList/repository.go
git add internal/repositories/priceList/subgroup_update_map_test.go
git commit -m "refactor: แยก buildSubGroupUpdateMap ออกมาให้ทดสอบได้

ยกตรรกะการประกอบ updateMap ออกจาก UpdatePriceListSubGroups แบบคำต่อคำ
ไม่เปลี่ยนพฤติกรรม เพื่อให้เขียน unit test เงื่อนไขการเลื่อน before_* ได้
โดยไม่ต้องมี database"
```

### Task C2: แก้บั๊ก — เลื่อน `before_*` เฉพาะเมื่อค่าเปลี่ยนจริง

**Files:**
- Modify: `internal/repositories/priceList/subgroup_update_map_test.go`
- Modify: `internal/repositories/priceList/repository.go`

- [ ] **Step 1: เขียน test ของบั๊กที่ fail**

เพิ่มท้ายไฟล์ test

```go
// บั๊กจริง: update-latest ส่ง TotalNetPrice* ทุก subgroup ทุกครั้งแม้ราคาไม่เปลี่ยน
// ถ้าเลื่อน before_* โดยไม่ดูว่าค่าเปลี่ยนจริงไหม snapshot จะถูกทับจนเท่ากับค่าปัจจุบัน
// ข้อมูลจริงเมื่อ 2026-09-11 มี 1,257 จาก 1,350 subgroup ที่ before = after ไปแล้ว
func TestBuildSubGroupUpdateMap_DoesNotShiftBeforeWhenValueUnchanged(t *testing.T) {
	old := models.PriceListSubGroup{
		TotalNetPriceWeight: 18.50,
		TotalNetPriceUnit:   1200,
		PriceWeight:         17,
		ExtraPriceWeight:    1.5,
		TermPriceWeight:     0.55,
		PriceUnit:           1100,
		ExtraPriceUnit:      100,
	}
	item := models.UpdatePriceListSubGroupItem{
		TotalNetPriceWeight: floatPtr(18.50),
		TotalNetPriceUnit:   floatPtr(1200),
		PriceWeight:         floatPtr(17),
		ExtraPriceWeight:    floatPtr(1.5),
		TermPriceWeight:     floatPtr(0.55),
		PriceUnit:           floatPtr(1100),
		ExtraPriceUnit:      floatPtr(100),
	}

	got := buildSubGroupUpdateMap(old, item)

	for key := range got {
		if len(key) >= 7 && key[:7] == "before_" {
			t.Errorf("ค่าไม่เปลี่ยนแต่ยังเลื่อน %s = %v — snapshot จะถูกทับ", key, got[key])
		}
	}
}

// เลื่อนเฉพาะ field ที่ค่าเปลี่ยนจริง field อื่นในคำขอเดียวกันต้องไม่ถูกเลื่อน
func TestBuildSubGroupUpdateMap_ShiftsOnlyChangedFields(t *testing.T) {
	old := models.PriceListSubGroup{
		TotalNetPriceWeight: 18.50,
		TotalNetPriceUnit:   1200,
	}
	item := models.UpdatePriceListSubGroupItem{
		TotalNetPriceWeight: floatPtr(19.75), // เปลี่ยน
		TotalNetPriceUnit:   floatPtr(1200),  // ไม่เปลี่ยน
	}

	got := buildSubGroupUpdateMap(old, item)

	if got["before_total_net_price_weight"] != 18.50 {
		t.Errorf("field ที่เปลี่ยนต้องเลื่อน before: ได้ %v", got["before_total_net_price_weight"])
	}
	if _, exists := got["before_total_net_price_unit"]; exists {
		t.Error("field ที่ไม่เปลี่ยนต้องไม่เลื่อน before")
	}
	// ค่าปัจจุบันยังต้องถูกเขียนทั้งสอง field
	if got["total_net_price_weight"] != 19.75 || got["total_net_price_unit"] != float64(1200) {
		t.Errorf("ค่าปัจจุบันต้องถูกเขียนครบ: %v", got)
	}
}

// before_term_price_unit ไม่มีผู้เขียนเลย ทั้งที่ before_term_price_weight มี
func TestBuildSubGroupUpdateMap_CoversTermPriceUnit(t *testing.T) {
	old := models.PriceListSubGroup{BeforeTermPriceUnit: 0, TermPriceUnit: 5}
	item := models.UpdatePriceListSubGroupItem{TermPriceUnit: floatPtr(7)}

	got := buildSubGroupUpdateMap(old, item)

	if got["term_price_unit"] != float64(7) {
		t.Fatalf("term_price_unit = %v, want 7", got["term_price_unit"])
	}
	if got["before_term_price_unit"] != float64(5) {
		t.Fatalf("before_term_price_unit = %v, want 5", got["before_term_price_unit"])
	}
}
```

- [ ] **Step 2: รัน test ให้เห็นว่า fail**

Run:
```bash
cd prime-wms-erp-core
go test -v ./internal/repositories/priceList/ -run TestBuildSubGroupUpdateMap
```
Expected:
- `TestBuildSubGroupUpdateMap_DoesNotShiftBeforeWhenValueUnchanged` FAIL —
  รายงานทุก key `before_*` ที่ถูกเลื่อนทั้งที่ค่าไม่เปลี่ยน
- `TestBuildSubGroupUpdateMap_ShiftsOnlyChangedFields` FAIL
- `TestBuildSubGroupUpdateMap_CoversTermPriceUnit` FAIL (compile error ถ้า
  `models.PriceListSubGroup` ยังไม่มี field `TermPriceUnit` หรือ
  `UpdatePriceListSubGroupItem` ไม่มี field นี้)

**ถ้า compile error เพราะไม่มี field `TermPriceUnit`**: ให้ตรวจ schema ก่อนว่าตาราง
`price_list_sub_group` มีคอลัมน์ `term_price_unit` จริงหรือไม่ ด้วย SELECT

```bash
psql "$DSN" -c "\d price_list_sub_group" | grep term_price
```

ถ้าไม่มีคอลัมน์นี้ในตาราง ให้**ลบ test ตัวนี้ทิ้ง** และบันทึกไว้ใน PR description ว่า
`before_term_price_unit` ในโครงสร้าง Go เป็น field ที่ตารางไม่มี จึงไม่มีอะไรต้องแก้
ห้ามสร้างคอลัมน์ใหม่ในงานนี้

- [ ] **Step 3: แก้ `buildSubGroupUpdateMap` ให้เลื่อนเฉพาะเมื่อค่าเปลี่ยน**

```go
// buildSubGroupUpdateMap ประกอบ map สำหรับ UPDATE จากค่าเดิมและค่าที่ request ส่งมา
//
// field ที่ request ไม่ได้ส่งมา (nil) จะไม่ถูกใส่ใน map เลย เพื่อไม่ให้ GORM
// เขียนทับด้วย zero value
//
// before_* จะถูกเลื่อน **เฉพาะเมื่อค่าเปลี่ยนจริง** เพราะ update-latest ส่ง
// TotalNetPrice* ทุก subgroup ทุกครั้งแม้ราคาที่คำนวณได้เท่าเดิม และรองรับ
// update_type = "group" ที่ดึงทั้งกลุ่มมาคำนวณ ถ้าเลื่อนทุกครั้ง snapshot จะถูกทับ
// จนเท่ากับค่าปัจจุบันและค่า "ก่อนแก้ครั้งล่าสุด" จะหายไป
func buildSubGroupUpdateMap(oldSubGroup models.PriceListSubGroup, req models.UpdatePriceListSubGroupItem) map[string]interface{} {
	updateMap := make(map[string]interface{})

	apply := func(column string, newValue *float64, oldValue float64) {
		if newValue == nil {
			return
		}
		if *newValue != oldValue {
			updateMap["before_"+column] = oldValue
		}
		updateMap[column] = *newValue
	}

	apply("price_unit", req.PriceUnit, oldSubGroup.PriceUnit)
	apply("extra_price_unit", req.ExtraPriceUnit, oldSubGroup.ExtraPriceUnit)
	apply("total_net_price_unit", req.TotalNetPriceUnit, oldSubGroup.TotalNetPriceUnit)
	apply("price_weight", req.PriceWeight, oldSubGroup.PriceWeight)
	apply("extra_price_weight", req.ExtraPriceWeight, oldSubGroup.ExtraPriceWeight)
	apply("term_price_weight", req.TermPriceWeight, oldSubGroup.TermPriceWeight)
	apply("total_net_price_weight", req.TotalNetPriceWeight, oldSubGroup.TotalNetPriceWeight)

	return updateMap
}
```

ถ้า Step 2 ยืนยันว่าคอลัมน์ `term_price_unit` มีจริงในตาราง ให้เพิ่มบรรทัด

```go
	apply("term_price_unit", req.TermPriceUnit, oldSubGroup.TermPriceUnit)
```

- [ ] **Step 4: รัน test ให้ผ่านทั้งหมด**

Run:
```bash
cd prime-wms-erp-core
go test -v ./internal/repositories/priceList/ -run TestBuildSubGroupUpdateMap
```
Expected: PASS ทุกตัว รวม characterization test จาก Task C1

- [ ] **Step 5: รัน test และ build ทั้ง repo**

Run:
```bash
cd prime-wms-erp-core
go build ./... && go test ./...
```
Expected: build ผ่าน · test ok ทุก package

- [ ] **Step 6: รัน integration test**

Run:
```bash
cd prime-wms-erp-core
make test-integration-pricelist
```
Expected: ok — ใช้ testcontainers postgres:16 `TestMain` ตั้ง DSN ให้เอง
ต้องมีแค่ docker daemon

- [ ] **Step 7: Commit**

```bash
cd prime-wms-erp-core
git add internal/repositories/priceList/repository.go
git add internal/repositories/priceList/subgroup_update_map_test.go
git commit -m "fix: เลื่อน before_* เฉพาะเมื่อราคาเปลี่ยนจริง

update-latest-pricelist-subgroup.go:321-324 ส่ง pointer ของ TotalNetPrice*
ทุก subgroup ทุกครั้งแม้ราคาที่คำนวณได้เท่าเดิม และรองรับ update_type=group
ที่ดึง subgroup ทั้งกลุ่มมาคำนวณ ฝั่ง web ก็เรียกแบบ fire-and-forget หลังแก้ Extra
ฉะนั้นการเลื่อน before_* โดยไม่ดูว่าค่าเปลี่ยนจริงไหม ทำให้ snapshot ถูกทับ
จนเท่ากับค่าปัจจุบัน ผู้ใช้กด Reset แล้วเห็นราคา before เปลี่ยน

ข้อมูลจริงเมื่อ 2026-09-11: 1,257 จาก 1,350 subgroup มี before = after แล้ว

แก้ที่ repository เพราะเป็นจุดที่ caller ทุกตัววิ่งผ่าน แก้ที่ update-latest
จะครอบแค่ caller เดียว"
```

- [ ] **Step 8: เปิด PR**

```bash
cd prime-wms-erp-core
git push -u origin fix/pricelist-before-price-snapshot
gh pr create --base Develop --title "fix: before_* ถูกทับทุกครั้งที่คำนวณใหม่แม้ราคาไม่เปลี่ยน"
```

ใน PR description ต้องระบุว่า **การแก้นี้ไม่ย้อนไปซ่อม 1,257 แถวที่ `before` หายไปแล้ว**
ค่าเดิมสูญไปใน `price_list_sub_group` แต่ยังกู้ได้จาก `price_list_sub_group_history`
ซึ่ง `repository.go` เขียนไว้ทุกครั้ง ถ้าธุรกิจต้องการกู้ ให้แยกเป็นงานต่างหาก
พร้อมรายงานก่อนเหมือน issue 1

---

# PR D — Extra Price List ไม่ตรวจค่าว่าง

**branch (erp-core):** `fix/pricelist-extra-validation`
**branch (web):** `fix/pricelist-extra-validation`

`binding:"required"` **ใช้ไม่ได้** เพราะ route `/UpdatePriceListExtra`
(`internal/routes/routes.go:81-82`) ใช้ `utils.ProcessRequest` ซึ่งอ่าน raw body แล้วให้
service ทำ `json.Unmarshal` เอง ไม่ผ่าน `ShouldBindJSON` จึงต้อง validate ด้วยโค้ดตรง ๆ

### Task D1: ให้ `ProcessRequest` แปลง `*BindingError` เป็น HTTP 400

**Files:**
- Create: `internal/utils/request-handler_test.go`
- Modify: `internal/utils/request-handler.go:11-29`

- [ ] **Step 1: แตก branch**

```bash
cd prime-wms-erp-core
git fetch origin
git checkout -b fix/pricelist-extra-validation origin/Develop
```

- [ ] **Step 2: เขียน test ที่ fail**

```go
package utils

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// ProcessRequest แปลง error ทุกชนิดเป็น 500 ทำให้แยก "ผู้ใช้ส่งข้อมูลไม่ครบ"
// กับ "ระบบพัง" ไม่ออก BindingError ต้องได้ 400 เหมือนที่ ProcessRequestWithBinding ทำ
func TestProcessRequest_BindingErrorReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString(`{}`))

	ProcessRequest(c, func(*gin.Context, string) (interface{}, error) {
		return nil, &BindingError{Message: "extra_key is required"}
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("extra_key is required")) {
		t.Fatalf("response ต้องมีข้อความของ BindingError: %s", w.Body.String())
	}
}

// error ชนิดอื่นต้องยังได้ 500 เหมือนเดิม
func TestProcessRequest_OtherErrorStillReturns500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString(`{}`))

	ProcessRequest(c, func(*gin.Context, string) (interface{}, error) {
		return nil, errors.New("database is down")
	})

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

// กรณีปกติต้องได้ 200 พร้อม payload
func TestProcessRequest_SuccessReturns200(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString(`{"a":1}`))

	ProcessRequest(c, func(_ *gin.Context, payload string) (interface{}, error) {
		return map[string]string{"echo": payload}, nil
	})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`{"a":1}`)) {
		t.Fatalf("payload ต้องถูกส่งต่อให้ service: %s", w.Body.String())
	}
}
```

- [ ] **Step 3: รัน test ให้เห็นว่า fail**

Run:
```bash
cd prime-wms-erp-core
go test -v ./internal/utils/ -run TestProcessRequest
```
Expected: `TestProcessRequest_BindingErrorReturns400` FAIL — `status = 500, want 400`
ส่วนอีกสองตัว PASS

- [ ] **Step 4: แก้ `ProcessRequest`**

ใน `internal/utils/request-handler.go` เปลี่ยนบล็อกจัดการ error

```go
	// เรียกใช้ service function ที่ส่งเข้ามา
	response, err := serviceFunc(c, string(jsonData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
```

เป็น

```go
	// เรียกใช้ service function ที่ส่งเข้ามา
	response, err := serviceFunc(c, string(jsonData))
	if err != nil {
		// แยก "ผู้ใช้ส่งข้อมูลไม่ครบ" ออกจาก "ระบบพัง" ให้ตรงกับ
		// ProcessRequestWithBinding ไม่งั้น validation error จะกลายเป็น 500
		if bindingErr, ok := err.(*BindingError); ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Validation failed",
				"details": bindingErr.Message,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
```

- [ ] **Step 5: รัน test ให้ผ่าน**

Run:
```bash
cd prime-wms-erp-core
go test -v ./internal/utils/ -run TestProcessRequest
go test ./...
```
Expected: PASS ทั้งสาม · test ทั้ง repo ok

- [ ] **Step 6: Commit**

```bash
cd prime-wms-erp-core
git add internal/utils/request-handler.go internal/utils/request-handler_test.go
git commit -m "fix: ProcessRequest แปลง BindingError เป็น HTTP 400

เดิม error ทุกชนิดกลายเป็น 500 ทำให้แยก 'ผู้ใช้ส่งข้อมูลไม่ครบ' กับ 'ระบบพัง' ไม่ออก
ทำให้ตรงกับ ProcessRequestWithBinding ที่แปลงอยู่แล้ว
เป็นเงื่อนไขที่ต้องมีก่อนเพิ่ม validation ให้ UpdateExtras"
```

### Task D2: เพิ่ม validation ใน `UpdateExtras`

**Files:**
- Create: `internal/services/price-service/update-extras_validation_test.go`
- Modify: `internal/services/price-service/update-pricelist.go` (`UpdateExtras` ราวบรรทัด 164)

- [ ] **Step 1: เขียน test ที่ fail**

```go
package priceService

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prime-erp-core/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// payload ที่ครบถ้วน ใช้เป็นฐานแล้วเอาแต่ละ field ออกทีละตัว
func validExtraPayload() map[string]interface{} {
	return map[string]interface{}{
		"price_list_group_id": uuid.New().String(),
		"extra_key":           "PG06_1",
		"condition_code":      "PG06",
		"operator":            "<=",
		"value_int":           1.0,
		"length_extra_key":    1,
		"cond_range_min":      0.0,
		"cond_range_max":      45.0,
		"price_list_group_extra_keys": []map[string]interface{}{
			{"code": "PG06", "value": "PG06_1", "seq": 1},
		},
	}
}

func TestUpdateExtras_RejectsEmptyFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name  string
		mutate func(m map[string]interface{})
	}{
		{"price_list_group_id ว่าง", func(m map[string]interface{}) {
			m["price_list_group_id"] = uuid.Nil.String()
		}},
		{"extra_key ว่าง", func(m map[string]interface{}) { m["extra_key"] = "" }},
		{"extra_key มีแต่ช่องว่าง", func(m map[string]interface{}) { m["extra_key"] = "   " }},
		{"condition_code ว่าง", func(m map[string]interface{}) { m["condition_code"] = "" }},
		{"operator ว่าง", func(m map[string]interface{}) { m["operator"] = "" }},
		{"operator ไม่รู้จัก", func(m map[string]interface{}) { m["operator"] = "~~" }},
		{"cond_range_min มากกว่า max", func(m map[string]interface{}) {
			m["cond_range_min"] = 50.0
			m["cond_range_max"] = 10.0
		}},
		{"extra_keys เป็น slice ว่าง", func(m map[string]interface{}) {
			m["price_list_group_extra_keys"] = []map[string]interface{}{}
		}},
		{"extra_keys มี code ว่าง", func(m map[string]interface{}) {
			m["price_list_group_extra_keys"] = []map[string]interface{}{
				{"code": "", "value": "PG06_1", "seq": 1},
			}
		}},
		{"extra_keys มี value ว่าง", func(m map[string]interface{}) {
			m["price_list_group_extra_keys"] = []map[string]interface{}{
				{"code": "PG06", "value": "", "seq": 1},
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := validExtraPayload()
			tt.mutate(payload)
			body, _ := json.Marshal([]map[string]interface{}{payload})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/price/UpdatePriceListExtra", bytes.NewReader(body))

			_, err := UpdateExtras(c, string(body))
			if err == nil {
				t.Fatal("ต้องได้ error แต่ผ่าน validation ไปได้")
			}
			if _, ok := err.(*utils.BindingError); !ok {
				t.Fatalf("ต้องเป็น *utils.BindingError เพื่อให้ได้ HTTP 400 แต่ได้ %T: %v", err, err)
			}
		})
	}
}
```

**หมายเหตุสำคัญ**: `UpdateExtras` เรียก `priceListRepository.UpdateExtra(extras)` ตรง ๆ
ซึ่งต้องมี DB ฉะนั้น test นี้พิสูจน์ได้เฉพาะกรณีที่ validation **ปฏิเสธ** ก่อนถึง
repository เท่านั้น — ซึ่งเป็นสิ่งที่ต้องการพอดี ห้ามเขียน test กรณี "ผ่าน validation"
ในไฟล์นี้ เพราะจะไปชน DB

ถ้าต้องการ test กรณีผ่านด้วย ให้ทำแบบ `update-latest-pricelist-subgroup.go:22`
คือแยก `priceListRepository.UpdateExtra` ออกเป็น package-level function var

```go
var updateExtraFunc = priceListRepository.UpdateExtra
```

แล้ว swap ใน test ตาม pattern ของ
`TestUpdatePriceListSubGroup_Validation_MissingID`
(`update-pricelist-subgroup_service_test.go`) — ทำเฉพาะถ้าจำเป็น อย่าเพิ่มโดยไม่ใช้

- [ ] **Step 2: รัน test ให้เห็นว่า fail**

Run:
```bash
cd prime-wms-erp-core
go test -v ./internal/services/price-service/ -run TestUpdateExtras_RejectsEmptyFields
```
Expected: FAIL ทุก subtest — `ต้องได้ error แต่ผ่าน validation ไปได้`
(บาง subtest อาจ panic เพราะไปถึง repository ที่ไม่มี DB — ก็ถือว่า fail เช่นกัน
และจะหายไปเมื่อ validation ทำงาน)

- [ ] **Step 3: เพิ่มฟังก์ชัน validate**

เพิ่มใน `update-pricelist.go` ก่อน `UpdateExtras`

```go
// validExtraOperators คือ operator ที่ getEffectiveRange รู้จัก
// operator อื่นจะตกไป default เงียบ ๆ ทำให้การตรวจ overlap ไม่ตรงกับที่ตั้งใจ
var validExtraOperators = map[string]bool{
	"<=": true,
	">=": true,
	"=":  true,
	"<>": true,
}

// validateExtras ตรวจว่าข้อมูล extra ที่ส่งมาไม่มี field ว่างที่จำเป็น
//
// route /UpdatePriceListExtra ใช้ utils.ProcessRequest ซึ่งไม่ผ่าน ShouldBindJSON
// ฉะนั้น binding:"required" tag ใช้ไม่ได้ ต้องตรวจด้วยโค้ดตรง ๆ
func validateExtras(extras []models.UpdatePriceListExtraRequest) error {
	if len(extras) == 0 {
		return &utils.BindingError{Message: "ต้องส่งข้อมูล extra มาอย่างน้อย 1 รายการ"}
	}

	for i, e := range extras {
		if e.PriceListGroupID == uuid.Nil {
			return &utils.BindingError{
				Message: fmt.Sprintf("รายการที่ %d: price_list_group_id ห้ามว่าง", i+1),
			}
		}
		if strings.TrimSpace(e.ExtraKey) == "" {
			return &utils.BindingError{
				Message: fmt.Sprintf("รายการที่ %d: extra_key ห้ามว่าง", i+1),
			}
		}
		if strings.TrimSpace(e.ConditionCode) == "" {
			return &utils.BindingError{
				Message: fmt.Sprintf("รายการที่ %d: condition_code ห้ามว่าง", i+1),
			}
		}
		if !validExtraOperators[strings.TrimSpace(e.Operator)] {
			return &utils.BindingError{
				Message: fmt.Sprintf("รายการที่ %d: operator %q ไม่ถูกต้อง ต้องเป็น <=, >=, = หรือ <>", i+1, e.Operator),
			}
		}
		if e.CondRangeMin > e.CondRangeMax {
			return &utils.BindingError{
				Message: fmt.Sprintf("รายการที่ %d: cond_range_min (%v) ต้องไม่มากกว่า cond_range_max (%v)", i+1, e.CondRangeMin, e.CondRangeMax),
			}
		}
		if len(e.PriceListGroupExtraKeys) == 0 {
			return &utils.BindingError{
				Message: fmt.Sprintf("รายการที่ %d: price_list_group_extra_keys ห้ามว่าง", i+1),
			}
		}
		for j, k := range e.PriceListGroupExtraKeys {
			if strings.TrimSpace(k.Code) == "" {
				return &utils.BindingError{
					Message: fmt.Sprintf("รายการที่ %d key ที่ %d: code ห้ามว่าง", i+1, j+1),
				}
			}
			if strings.TrimSpace(k.Value) == "" {
				return &utils.BindingError{
					Message: fmt.Sprintf("รายการที่ %d key ที่ %d: value ห้ามว่าง", i+1, j+1),
				}
			}
		}
	}

	return nil
}
```

แล้วใน `UpdateExtras` แทรกการเรียกหลัง `json.Unmarshal` ก่อน
`checkForOverlappingConditions`

```go
func UpdateExtras(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := []models.UpdatePriceListExtraRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, &utils.BindingError{Message: "payload ไม่ใช่ JSON ที่ถูกต้อง: " + err.Error()}
	}

	if err := validateExtras(req); err != nil {
		return nil, err
	}

	// Validate for overlapping conditions
	if err := checkForOverlappingConditions(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
```

ตรวจว่า import มี `strings`, `fmt`, `github.com/google/uuid` และ
`prime-erp-core/internal/utils` ครบ ถ้าขาดให้เพิ่ม

- [ ] **Step 4: รัน test ให้ผ่าน**

Run:
```bash
cd prime-wms-erp-core
go test -v ./internal/services/price-service/ -run TestUpdateExtras_RejectsEmptyFields
```
Expected: PASS ทุก subtest

- [ ] **Step 5: build และรัน test ทั้ง repo**

Run:
```bash
cd prime-wms-erp-core
go build ./... && go test ./...
```
Expected: build ผ่าน · test ok

- [ ] **Step 6: Commit และเปิด PR**

```bash
cd prime-wms-erp-core
git add internal/services/price-service/update-pricelist.go
git add internal/services/price-service/update-extras_validation_test.go
git commit -m "fix: ตรวจค่าว่างใน UpdateExtras

route /UpdatePriceListExtra ใช้ ProcessRequest ซึ่งไม่ผ่าน ShouldBindJSON
ฉะนั้น binding:\"required\" tag ใช้ไม่ได้ ต้องตรวจด้วยโค้ดตรง ๆ
ตรวจ price_list_group_id, extra_key, condition_code, operator, ช่วง min/max
และ price_list_group_extra_keys พร้อมระบุ index ในข้อความ

เพิ่มการตรวจ operator ให้เป็นค่าที่ getEffectiveRange รู้จักด้วย
เพราะ operator ที่ไม่รู้จักเดิมตกไป default เงียบ ๆ ทำให้ตรวจ overlap ไม่ตรงเจตนา"
git push -u origin fix/pricelist-extra-validation
gh pr create --base Develop --title "fix: Extra Price List ไม่ตรวจค่าว่างตอน New และ Edit"
```

### Task D3: validation ฝั่ง web

**Files:**
- Create: `prime-wms-web/src/views/price-list/extraValidation.ts`
- Create: `prime-wms-web/src/views/price-list/extraValidation.spec.ts`
- Modify: `prime-wms-web/src/views/price-list/Extra.vue`

- [ ] **Step 1: แตก branch**

```bash
cd prime-wms-web
git fetch origin
git checkout -b fix/pricelist-extra-validation origin/Develop
```

- [ ] **Step 2: เขียน test ที่ fail**

`formState` เป็น `PriceListExtra[]` (array) ไม่ใช่ object เดียว จึงใช้
`a-form :rules` กับมันตรง ๆ ไม่ได้ — แยกตรรกะออกเป็นฟังก์ชันบริสุทธิ์แล้วทดสอบ

```typescript
import { describe, expect, it } from 'vitest';

import { validateExtras } from './extraValidation';
import { OPERATOR_ENUM } from '@/types/price-list/price-list.model';
import type { PriceListExtra } from '@/types/price-list/price-list.model';

const validExtra = (): PriceListExtra => ({
  id: 'e1',
  priceListGroupId: 'g1',
  extraKey: 'PG06_1',
  conditionCode: 'PG06',
  valueInt: 1,
  lengthExtraKey: 1,
  operator: OPERATOR_ENUM.BETWEEN,
  condRangeMin: 0,
  condRangeMax: 45,
  priceListGroupExtraKeys: {
    PG06: { valueCode: 'PG06_1', valueName: 'ความยาว 6 เมตร' },
  },
} as PriceListExtra);

describe('validateExtras', () => {
  it('ข้อมูลครบต้องผ่าน', () => {
    expect(validateExtras([validExtra()])).toEqual([]);
  });

  it('ปฏิเสธเมื่อไม่มีรายการเลย', () => {
    expect(validateExtras([]).length).toBeGreaterThan(0);
  });

  it('ปฏิเสธ conditionCode ว่าง', () => {
    const e = validExtra();
    e.conditionCode = '';
    expect(validateExtras([e]).length).toBeGreaterThan(0);
  });

  it('ปฏิเสธ conditionCode ที่มีแต่ช่องว่าง', () => {
    const e = validExtra();
    e.conditionCode = '   ';
    expect(validateExtras([e]).length).toBeGreaterThan(0);
  });

  it('ปฏิเสธ operator ว่าง', () => {
    const e = validExtra();
    e.operator = '' as OPERATOR_ENUM;
    expect(validateExtras([e]).length).toBeGreaterThan(0);
  });

  it('ปฏิเสธเมื่อ condRangeMin มากกว่า condRangeMax', () => {
    const e = validExtra();
    e.condRangeMin = 50;
    e.condRangeMax = 10;
    expect(validateExtras([e]).length).toBeGreaterThan(0);
  });

  it('ปฏิเสธเมื่อ priceListGroupExtraKeys ว่าง', () => {
    const e = validExtra();
    e.priceListGroupExtraKeys = {};
    expect(validateExtras([e]).length).toBeGreaterThan(0);
  });

  it('ปฏิเสธเมื่อ key ตัวใดมี valueCode ว่าง', () => {
    const e = validExtra();
    e.priceListGroupExtraKeys = {
      PG06: { valueCode: '', valueName: '' },
    } as PriceListExtra['priceListGroupExtraKeys'];
    expect(validateExtras([e]).length).toBeGreaterThan(0);
  });

  it('ข้อความ error ต้องระบุแถวเพื่อให้ผู้ใช้หาได้', () => {
    const bad = validExtra();
    bad.conditionCode = '';
    const messages = validateExtras([validExtra(), bad]);
    expect(messages.join(' ')).toContain('2');
  });
});
```

- [ ] **Step 3: รัน test ให้เห็นว่า fail**

Run:
```bash
cd prime-wms-web
npm run test -- extraValidation
```
Expected: FAIL — หาโมดูล `./extraValidation` ไม่เจอ

**ก่อนเขียน implementation ต้องอ่านของจริงก่อน**: ตรวจ `PriceListExtra` และ
`OPERATOR_ENUM` ที่ `src/types/price-list/price-list.model.ts` ว่าชื่อ field และ path
import ตรงกับที่ test เขียนไว้หรือไม่ (`priceListGroupExtraKeys` เป็น
`Record<string, PriceListGroupExtraKey>` และ `PriceListGroupExtraKey` มี field อะไร)
ถ้าไม่ตรงให้แก้ **test** ให้ตรงกับของจริง ห้ามแก้ type ให้ตรงกับ test

- [ ] **Step 4: เขียน implementation**

```typescript
import { OPERATOR_ENUM } from '@/types/price-list/price-list.model';
import type { PriceListExtra } from '@/types/price-list/price-list.model';

const isBlank = (v?: string): boolean => !v || v.trim() === '';

/**
 * ตรวจข้อมูล Extra Price List ก่อนส่งขึ้น backend
 *
 * formState เป็น array ของ PriceListExtra จึงใช้ `a-form :rules` กับมันตรง ๆ ไม่ได้
 * ตรวจชุดเดียวกับ validateExtras ฝั่ง Go เพื่อให้ผู้ใช้เห็น error ทันทีไม่ต้องรอ 400
 * แต่ backend ยังต้องตรวจซ้ำเพราะเป็น trust boundary
 *
 * คืน array ของข้อความ error ถ้าว่างแปลว่าผ่าน
 */
export const validateExtras = (extras: PriceListExtra[]): string[] => {
  const errors: string[] = [];

  if (extras.length === 0) {
    errors.push('ต้องมีข้อมูล Extra อย่างน้อย 1 รายการ');
    return errors;
  }

  extras.forEach((e, index) => {
    const row = index + 1;

    if (isBlank(e.priceListGroupId)) {
      errors.push(`แถวที่ ${row}: ไม่พบ price list group`);
    }
    if (isBlank(e.extraKey)) {
      errors.push(`แถวที่ ${row}: extra key ห้ามว่าง`);
    }
    if (isBlank(e.conditionCode)) {
      errors.push(`แถวที่ ${row}: condition ห้ามว่าง`);
    }
    if (isBlank(e.operator)) {
      errors.push(`แถวที่ ${row}: operator ห้ามว่าง`);
    }
    if (
      e.operator === OPERATOR_ENUM.BETWEEN &&
      Number(e.condRangeMin) > Number(e.condRangeMax)
    ) {
      errors.push(
        `แถวที่ ${row}: ค่าเริ่มต้น (${e.condRangeMin}) ต้องไม่มากกว่าค่าสิ้นสุด (${e.condRangeMax})`
      );
    }

    const keys = Object.entries(e.priceListGroupExtraKeys || {});
    if (keys.length === 0) {
      errors.push(`แถวที่ ${row}: ต้องเลือกกลุ่มสินค้าอย่างน้อย 1 รายการ`);
    }
    keys.forEach(([code, value]) => {
      if (isBlank(value?.valueCode)) {
        errors.push(`แถวที่ ${row}: ยังไม่ได้เลือกค่าของ ${code}`);
      }
    });
  });

  return errors;
};
```

- [ ] **Step 5: รัน test ให้ผ่าน**

Run:
```bash
cd prime-wms-web
npm run test -- extraValidation
```
Expected: PASS ทุก case

- [ ] **Step 6: ต่อเข้ากับฟอร์ม**

ใน `Extra.vue` ลบ `rules` ที่ไม่ได้ใช้งาน (มีแค่ key `pass` ซึ่งไม่ตรงกับ field ไหนของ
`formState`) และ `:rules="rules"` บน `<a-form>` แล้วตรวจใน `onUpdate` แทน

```typescript
import { message } from 'ant-design-vue';

import { validateExtras } from './extraValidation';

const onUpdate = async () => {
  const errors = validateExtras(formState.value);
  if (errors.length > 0) {
    errors.slice(0, 3).forEach((e) => message.error(e));
    return;
  }

  await callUpdatePriceListExtra(formState.value);
  await loadPriceListData();

  // Update latest price list sub group (fire-and-forget)
  if (groupCode.value) {
    await updateLatestPriceListSubGroup([groupCode.value]).catch((error) => {
      console.error('Error updating latest price list sub group:', error);
    });
  }

  mode.value = MODEPAGE.VIEW;
};
```

แสดงแค่ 3 ข้อความแรกเพื่อไม่ให้ toast ท่วมจอ ผู้ใช้แก้แล้วกดซ้ำจะเห็นชุดถัดไป

ตรวจว่า `message` ถูก import จาก `ant-design-vue` อยู่แล้วในไฟล์นี้หรือไม่
ถ้ามีวิธีแสดง error แบบอื่นที่ใช้กันอยู่ในโปรเจกต์ (เช่น composable กลาง) ให้ใช้ตัวนั้น
แทนเพื่อให้เข้ากับของเดิม

- [ ] **Step 7: build และรัน test ทั้งหมด**

Run:
```bash
cd prime-wms-web
npm run test && npm run build
```
Expected: test ผ่าน · `vue-tsc` ไม่มี type error · build ผ่าน

- [ ] **Step 8: Commit และเปิด PR**

```bash
cd prime-wms-web
git add src/views/price-list/extraValidation.ts
git add src/views/price-list/extraValidation.spec.ts
git add src/views/price-list/Extra.vue
git commit -m "fix: ตรวจค่าว่างในฟอร์ม Extra Price List ก่อนส่ง

rules เดิมมีแค่ key pass ซึ่งไม่ตรงกับ field ไหนของ formState จึงไม่ validate อะไรเลย
formState เป็น array ของ PriceListExtra จึงใช้ a-form :rules กับมันตรง ๆ ไม่ได้
แยกเป็นฟังก์ชัน validateExtras ที่ทดสอบได้ และตรวจใน onUpdate ก่อนเรียก API
ตรวจชุดเดียวกับฝั่ง Go แต่ backend ยังตรวจซ้ำเพราะเป็น trust boundary"
git push -u origin fix/pricelist-extra-validation
gh pr create --base Develop --title "fix: ฟอร์ม Extra Price List ไม่ตรวจค่าว่าง"
```

---

# PR E — แสดงผล: comma, fallback, center

**branch (erp-core):** `fix/pricelist-display-fixes`
**branch (web):** `fix/pricelist-display-fixes`

### Task E1: web — เพิ่ม suffix ตัวเลขที่ขาด

**Files:**
- Create: `prime-wms-web/src/utils/helper/priceListNumberFormat.spec.ts`
- Modify: `prime-wms-web/src/utils/helper/priceListNumberFormat.ts`

- [ ] **Step 1: แตก branch**

```bash
cd prime-wms-web
git fetch origin
git checkout -b fix/pricelist-display-fixes origin/Develop
```

- [ ] **Step 2: ไล่หา field ตัวเลขที่ยังไม่อยู่ในลิสต์ (SELECT/grep เท่านั้น)**

Run:
```bash
cd prime-wms-erp-core
python3 -c "
import json, glob, os, re
have = {'price_unit','price_weight','total_net_price_unit','total_net_price_weight',
        'extra_price_unit','extra_price_weight','extra_thb','avg_weight','avg_kg_stock',
        'avg_weight_ton','total_weight','market_weight','stock','stock_quantity',
        'quantity','ton'}
fields=set()
for f in glob.glob('internal/services/price-service/patterns/configs/*.json'):
    c=json.load(open(f))
    for p in c.get('patterns',[]):
        for key in ('fixedColumns','columns'):
            for col in p.get(key,[]):
                fld = col.get('dataMapping') or col.get('field')
                if fld: fields.add(fld)
missing = sorted(x for x in fields
                 if not any(x==s or x.endswith('_'+s) for s in have))
print('field ที่ยังไม่ match suffix ใดเลย:')
for m in missing: print('  ', m)
"
```

บันทึกผลไว้ แล้วคัดเฉพาะ field ที่เป็น**ตัวเลขเงินหรือน้ำหนัก** ตามที่ผู้ใช้สั่ง
field ที่เป็น boolean (`inactive`, `is_highlight`), ข้อความ (`remark`, `PG01`–`PG10`),
หรือคอลัมน์ลำดับ (`#`) **ไม่ต้องใส่**

- [ ] **Step 3: เขียน test ที่ fail**

```typescript
import { describe, expect, it } from 'vitest';

import {
  isNumericPriceListField,
  applyNumberFormatting,
  formatPriceListNumber,
} from './priceListNumberFormat';

describe('isNumericPriceListField', () => {
  it('จับ field เปล่า', () => {
    expect(isNumericPriceListField('total_net_price_weight')).toBe(true);
  });

  it('จับ field ที่มี prefix กลุ่ม', () => {
    expect(isNumericPriceListField('pg09_2_total_net_price_weight')).toBe(true);
    expect(isNumericPriceListField('pg09_2_avg_weight')).toBe(true);
  });

  it('จับคอลัมน์ before ผ่าน suffix ที่มีอยู่แล้ว', () => {
    expect(isNumericPriceListField('before_total_net_price_weight')).toBe(true);
  });

  it('จับ line_bundle ซึ่งเป็นคอลัมน์ตัวเลขที่เคยหลุดไป', () => {
    expect(isNumericPriceListField('line_bundle')).toBe(true);
    expect(isNumericPriceListField('pg09_2_line_bundle')).toBe(true);
  });

  it('ไม่จับคอลัมน์ที่ไม่ใช่ตัวเลข', () => {
    expect(isNumericPriceListField('remark')).toBe(false);
    expect(isNumericPriceListField('inactive')).toBe(false);
    expect(isNumericPriceListField('is_highlight')).toBe(false);
    expect(isNumericPriceListField('PG01')).toBe(false);
  });

  it('ไม่จับ undefined', () => {
    expect(isNumericPriceListField(undefined)).toBe(false);
  });
});

describe('applyNumberFormatting', () => {
  it('ใส่ valueFormatter ให้คอลัมน์ตัวเลขและใส่ comma', () => {
    const [col] = applyNumberFormatting([{ field: 'total_net_price_weight' }]);
    expect(typeof col.valueFormatter).toBe('function');
    expect(col.valueFormatter({ value: 1234567.5 })).toBe('1,234,567.50');
  });

  it('ไม่แตะคอลัมน์ที่มี valueFormatter ของตัวเองอยู่แล้ว', () => {
    const own = () => 'x';
    const [col] = applyNumberFormatting([
      { field: 'total_net_price_weight', valueFormatter: own },
    ]);
    expect(col.valueFormatter).toBe(own);
  });

  it('ลงไปถึง children ของ column group', () => {
    const [group] = applyNumberFormatting([
      { headerName: 'กลุ่ม', children: [{ field: 'line_bundle' }] },
    ]);
    expect(typeof group.children[0].valueFormatter).toBe('function');
  });

  it('ค่าว่างต้องได้ช่องว่าง ไม่ใช่ NaN', () => {
    const [col] = applyNumberFormatting([{ field: 'avg_weight' }]);
    expect(col.valueFormatter({ value: null })).toBe('');
    expect(col.valueFormatter({ value: undefined })).toBe('');
    expect(col.valueFormatter({ value: '' })).toBe('');
  });
});

describe('formatPriceListNumber', () => {
  it('ใส่ comma', () => {
    expect(formatPriceListNumber(1234567.5)).toBe('1,234,567.50');
  });

  it('ค่าว่างได้ช่องว่าง', () => {
    expect(formatPriceListNumber(null)).toBe('');
  });
});
```

- [ ] **Step 4: รัน test ให้เห็นว่า fail**

Run:
```bash
cd prime-wms-web
npm run test -- priceListNumberFormat
```
Expected: FAIL ที่ case `line_bundle` ทั้งสองบรรทัด และ case `children` ·
case อื่นควรผ่านเพราะกลไกเดิมทำงานอยู่แล้ว

ถ้า case `1,234,567.50` fail ให้ตรวจ `formatMoney` ที่
`src/utils/helper/number.ts:110-111` ว่า `formatDecimal(v, 2, placeholder)`
ใส่ comma จริงหรือไม่ — ถ้าไม่ใส่ ปัญหาอยู่ที่ `formatDecimal` ไม่ใช่ที่ลิสต์ suffix
ให้หยุดและรายงาน เพราะขอบเขตการแก้จะเปลี่ยน

- [ ] **Step 5: เพิ่ม suffix**

ใน `NUMERIC_SUFFIXES` เพิ่ม `line_bundle` และ field ที่ Step 2 คัดมาได้
เรียงต่อท้ายกลุ่มที่เกี่ยวข้องกัน

```typescript
const NUMERIC_SUFFIXES = [
  'price_unit',
  'price_weight',
  'total_net_price_unit',
  'total_net_price_weight',
  'extra_price_unit',
  'extra_price_weight',
  'extra_thb',
  'avg_weight',
  'avg_kg_stock',
  'avg_weight_ton',
  'total_weight',
  'market_weight',
  'stock',
  'stock_quantity',
  'quantity',
  'ton',
  'line_bundle',
];
```

- [ ] **Step 6: รัน test ให้ผ่าน**

Run:
```bash
cd prime-wms-web
npm run test -- priceListNumberFormat
```
Expected: PASS ทุก case

- [ ] **Step 7: Commit**

```bash
cd prime-wms-web
git add src/utils/helper/priceListNumberFormat.ts
git add src/utils/helper/priceListNumberFormat.spec.ts
git commit -m "fix: ใส่ comma คั่นหลักพันให้คอลัมน์ตัวเลขที่ตกหล่น

กลไก suffix match ทำงานถูกอยู่แล้วและครอบคอลัมน์ที่มี prefix กลุ่มกับ before_ ได้
แต่ลิสต์ NUMERIC_SUFFIXES ขาด line_bundle
เพิ่ม test ตรึงพฤติกรรมของทั้ง isNumericPriceListField และ applyNumberFormatting
รวมถึงการลงไปถึง children ของ column group และการจัดการค่าว่าง"
```

### Task E2: erp-core — แยก "ไม่มี record" จาก "มี record แต่ชื่อว่าง"

**Files:**
- Modify: `internal/services/price-service/get-price-export-table.go:90-101`
- Modify: `internal/services/price-service/build-pricelist-detail-tab.go:152-154, 186-188`
- Modify: `internal/services/price-service/build-pricelist-detail-tab_test.go`

- [ ] **Step 1: แตก branch**

```bash
cd prime-wms-erp-core
git fetch origin
git checkout -b fix/pricelist-display-fixes origin/Develop
```

- [ ] **Step 2: เขียน test ที่ fail**

เพิ่มท้าย `build-pricelist-detail-tab_test.go`

```go
// กฎที่ธุรกิจยืนยัน: ถ้ามี record ใน DB แต่ชื่อว่าง = ว่างจริง ห้าม fallback ไป code
// ตอนนี้ itemNameByCode คืน string เดี่ยว จึงแยกไม่ออกจากกรณี "ไม่มี record"
func TestBuildPricelistDetailTab_EmptyNameOnExistingRecordStaysEmpty(t *testing.T) {
	groups, groupNameByCode, _ := detailTestFixtures()

	// มี record ของ PG01_3 อยู่จริงแต่ชื่อว่างโดยตั้งใจ
	itemNameByCode := func(code string) (string, bool) {
		if code == "PG01_3" {
			return "", true
		}
		return "", false
	}

	tab := buildPricelistDetailTab(groups, groupNameByCode, itemNameByCode, nil, nil, nil)

	if got := tab.Rows[0]["PG01"]; got != "" {
		t.Fatalf("มี record แต่ชื่อว่าง ต้องได้เซลล์ว่าง แต่ได้ %v", got)
	}
}

// กรณีหา record ไม่เจอ ต้อง fallback ไป code ดิบเหมือนเดิม
func TestBuildPricelistDetailTab_MissingRecordStillFallsBackToCode(t *testing.T) {
	groups, groupNameByCode, _ := detailTestFixtures()

	itemNameByCode := func(string) (string, bool) { return "", false }

	tab := buildPricelistDetailTab(groups, groupNameByCode, itemNameByCode, nil, nil, nil)

	if got := tab.Rows[0]["PG01"]; got != "PG01_3" {
		t.Fatalf("ไม่มี record ต้อง fallback เป็น code ดิบ แต่ได้ %v", got)
	}
}
```

และแก้ `TestBuildPricelistDetailTab_FallbackToRawCode` ที่มีอยู่ (บรรทัด 277-295)
ให้ตรงกับ signature ใหม่ — เปลี่ยน

```go
	none := func(string) string { return "" }
	tab := buildPricelistDetailTab(groups, none, none, nil, nil, nil)
```

เป็น

```go
	noneName := func(string) string { return "" }
	noneItem := func(string) (string, bool) { return "", false }
	tab := buildPricelistDetailTab(groups, noneName, noneItem, nil, nil, nil)
```

**ต้องอ่านโค้ดจริงก่อน**: `buildPricelistDetailTab` รับ `groupNameByCode` และ
`itemNameByCode` เป็น parameter สองตัว ตรวจว่าตัวไหนเป็นตัวไหนและมี caller อื่น
อีกกี่ที่ ด้วย

```bash
cd prime-wms-erp-core
grep -rn "buildPricelistDetailTab(\|itemNameByCode\|groupNameByCode" --include=*.go internal/
```

แก้ให้ครบทุก caller ก่อนคาดหวังว่าจะ compile ผ่าน

- [ ] **Step 3: รัน test ให้เห็นว่า fail**

Run:
```bash
cd prime-wms-erp-core
go test ./internal/services/price-service/ -run TestBuildPricelistDetailTab
```
Expected: FAIL — compile error เพราะ signature ของ `itemNameByCode` ยังเป็น
`func(string) string`

- [ ] **Step 4: เปลี่ยน signature และเงื่อนไข fallback**

ใน `get-price-export-table.go:90-101` เปลี่ยน closure ให้คืน `ok` ที่มีอยู่แล้วออกมา

```go
	groupNameByCode := func(code string) string {
		if g, ok := groupMap[code]; ok {
			return g.GroupName
		}
		return ""
	}
	// คืน ok ออกมาด้วย เพื่อให้ผู้เรียกแยก "ไม่มี record" จาก "มี record แต่ชื่อว่าง" ได้
	// ธุรกิจยืนยันว่าชื่อว่างโดยตั้งใจต้องแสดงว่าง ไม่ fallback ไป code
	itemNameByCode := func(code string) (string, bool) {
		it, ok := groupItemMap[code]
		if !ok {
			return "", false
		}
		return it.ItemName, true
	}
```

ใน `build-pricelist-detail-tab.go` แก้ signature ของฟังก์ชันให้รับ
`itemNameByCode func(string) (string, bool)` แล้วเปลี่ยนบรรทัด 186-188

```go
		for _, k := range sg.GroupKeys {
			if k.Code == "" {
				continue
			}
			name, found := itemNameByCode(k.Value)
			if !found {
				name = k.Value
			}
			row[k.Code] = name
			row[k.Code+groupCodeColumnSuffix] = k.Value
		}
```

จุดที่ประกอบ header ของคอลัมน์ (`cols` ราวบรรทัด 115-125) ก็ใช้ชื่อจากแหล่งเดียวกัน
ให้ไล่ดูว่าใช้ `itemNameByCode` หรือไม่ ถ้าใช้ ต้องใช้เงื่อนไข `found` เดียวกัน

**ระดับ group** (บรรทัด 152-154) ใช้ `g.GroupName` ตรงจาก struct ไม่ได้ผ่าน closure
จึงแยก "ไม่มี record" ไม่ได้อยู่แล้ว — **ไม่ต้องแก้ในงานนี้** และให้บันทึกไว้ใน
PR description ว่าเป็นข้อจำกัดที่เหลืออยู่ เพราะการแก้ต้องเปลี่ยนที่แหล่งข้อมูล
ซึ่งอยู่นอกขอบเขต

- [ ] **Step 5: รัน test ให้ผ่าน**

Run:
```bash
cd prime-wms-erp-core
go test -v ./internal/services/price-service/ -run TestBuildPricelistDetailTab
```
Expected: PASS ทุกตัว รวม `TestBuildPricelistDetailTab_FallbackToRawCode` ที่แก้แล้ว

- [ ] **Step 6: build และรัน test ทั้ง repo**

Run:
```bash
cd prime-wms-erp-core
go build ./... && go test ./...
```
Expected: build ผ่าน · test ok

- [ ] **Step 7: Commit**

```bash
cd prime-wms-erp-core
git add internal/services/price-service/get-price-export-table.go
git add internal/services/price-service/build-pricelist-detail-tab.go
git add internal/services/price-service/build-pricelist-detail-tab_test.go
git commit -m "fix: ไม่ fallback ไป item_code เมื่อ record มีอยู่แต่ชื่อว่าง

itemNameByCode คืน string เดี่ยวจึงแยกไม่ออกระหว่าง 'ไม่มี record' กับ
'มี record แต่ชื่อว่าง' ทั้งสองกรณีได้ empty string เท่ากัน
ธุรกิจยืนยันว่าชื่อว่างโดยตั้งใจต้องแสดงว่าง ไม่ใช่โชว์ code อย่าง PG08_5

เปลี่ยนเป็นคืน (string, bool) โดยใช้ ok ของ map lookup ที่มีอยู่แล้ว
fallback เฉพาะเมื่อหา record ไม่เจอ
แก้ TestBuildPricelistDetailTab_FallbackToRawCode ให้ตรงกับ signature ใหม่
และเพิ่มเคส 'มี record แต่ชื่อว่าง' ที่เดิมไม่มี test ครอบ"
```

### Task E3: erp-core — จัดเซลล์ให้ center

**Files:**
- Modify: config JSON 9 ไฟล์ใน `internal/services/price-service/patterns/configs/`

- [ ] **Step 1: ยืนยันสถานะก่อนแก้**

Run:
```bash
cd prime-wms-erp-core
grep -c '"textAlign": "left"' internal/services/price-service/patterns/configs/*.json | grep -v ':0'
```
Expected: 9 ไฟล์ รวม 81 จุด — `GROUP_1_ITEM_2` (10), `GROUP_1_ITEM_3` (14),
`GROUP_1_ITEM_4` (9), `GROUP_1_ITEM_5` (12), `GROUP_1_ITEM_8` (6),
`GROUP_1_ITEM_9` (6), `GROUP_1_ITEM_10` (6), `GROUP_1_ITEM_11` (5), `PG01_3` (13)

หัวตารางเป็น center อยู่แล้วทั้งที่ `DynamicTable.vue:365-374` (default
`headerClass: 'ag-header-cell-center'`) และ `applyHeaderAlignment()` ที่ fallback
เป็น `'center'` — ที่ไม่ center คือเซลล์ข้อมูลเพราะ config ตั้ง `left` ไว้

- [ ] **Step 2: เปลี่ยน left เป็น center**

Run:
```bash
cd prime-wms-erp-core
sed -i 's/"textAlign": "left"/"textAlign": "center"/g' internal/services/price-service/patterns/configs/*.json
```

- [ ] **Step 3: ยืนยันว่าไม่เหลือ left และ JSON ยังอ่านได้**

Run:
```bash
cd prime-wms-erp-core
grep -rc '"textAlign": "left"' internal/services/price-service/patterns/configs/*.json | grep -v ':0' || echo "ไม่เหลือ left แล้ว"
python3 -c "
import json, glob
for f in sorted(glob.glob('internal/services/price-service/patterns/configs/*.json')):
    json.load(open(f))
print('JSON ทุกไฟล์ parse ผ่าน')
"
```
Expected: `ไม่เหลือ left แล้ว` · `JSON ทุกไฟล์ parse ผ่าน`

- [ ] **Step 4: รัน test ทั้ง repo**

Run:
```bash
cd prime-wms-erp-core
go test ./...
```
Expected: ok — ถ้ามี test ตรึง `textAlign: left` ไว้ ต้องแก้ test นั้นพร้อมอธิบาย
เหตุผลใน commit message

- [ ] **Step 5: Commit และเปิด PR ทั้งสอง repo**

```bash
cd prime-wms-erp-core
git add internal/services/price-service/patterns/configs/
git commit -m "fix: จัดเซลล์ข้อมูลใน price list ให้ center

หัวตารางเป็น center อยู่แล้วทั้งจาก defaultColDef ของ DynamicTable และจาก
applyHeaderAlignment ที่ fallback เป็น center
ที่ไม่ center คือเซลล์ข้อมูลเพราะ config ตั้ง textAlign left ไว้ 81 จุดใน 9 ไฟล์
แก้ที่ config ฝั่ง backend จุดเดียว ไม่ต้องแตะ web"
git push -u origin fix/pricelist-display-fixes
gh pr create --base Develop --title "fix: comma คั่นหลักพัน, fallback ชื่อว่าง, จัดเซลล์ center"

cd prime-wms-web
git push -u origin fix/pricelist-display-fixes
gh pr create --base Develop --title "fix: ใส่ comma คั่นหลักพันให้คอลัมน์ตัวเลขที่ตกหล่น"
```

---

# PR F — `parsePercent` หารด้วย 100 ผิดมาตรฐาน

**branch:** `fix/pricelist-percent-convention`

**ต้อง backfill ข้อมูลให้เสร็จก่อน deploy PR นี้** ตามที่ผู้ใช้ยืนยัน ดูขั้นตอนใน
`docs/superpowers/reports/2026-09-11-pricelist-term-percent-backfill.sql`
การเขียนโค้ดและเปิด PR ทำได้เลย แต่ห้าม merge ขึ้น environment จริงก่อน backfill

มาตรฐานที่ธุรกิจยืนยัน: `1` = 1% และ `0.1` = 0.1% คือเก็บเป็นจำนวนเปอร์เซ็นต์
และ `baht = price × percent / 100`

### Task F1: แก้ test ที่ตรึงพฤติกรรมผิดไว้ แล้วแก้ production code

**Files:**
- Modify: `internal/services/price-service/upload-pricelist_parse_test.go`
- Modify: `internal/services/price-service/upload-pricelist.go:1195-1204`

- [ ] **Step 1: แตก branch**

```bash
cd prime-wms-erp-core
git fetch origin
git checkout -b fix/pricelist-percent-convention origin/Develop
```

- [ ] **Step 2: แก้ test ที่ตรึงพฤติกรรมผิด และเพิ่มเคสใหม่**

`TestParse_PercentAndDecimalNotTruncated` ตรึงพฤติกรรมผิดไว้ — assert ว่า `"1.0%"`
ต้องได้ `0.01` ซึ่งตามมาตรฐานที่ธุรกิจยืนยันต้องได้ `1`

เปลี่ยนบล็อกคาดหวังจาก

```go
	for i, want := range []struct{ pdc, due float64 }{{0.01, 0.015}, {0.02, 0.03}, {0, 0}} {
```

เป็น

```go
	// มาตรฐานที่ธุรกิจยืนยัน 2026-09-11: 1 = 1% และ 0.1 = 0.1%
	// cell "1.0%" ต้องเก็บเป็น 1 ไม่ใช่ 0.01 · cell ที่ไม่มี % ต่อท้ายเก็บตามที่พิมพ์
	for i, want := range []struct{ pdc, due float64 }{{1, 1.5}, {0.02, 3}, {0, 0}} {
```

แถวที่ 2 ใน sheet คือ `{"0.37", "0.02", "0.56", "3.0%"}` → `pdc_percent` เป็น `"0.02"`
ไม่มี `%` ต่อท้าย จึงเก็บ `0.02` ตามที่พิมพ์ ส่วน `due_percent` เป็น `"3.0%"` → `3`

และเพิ่ม test ใหม่ท้ายไฟล์ ครอบ `parsePercent` ให้ครบทุกรูปแบบ input

```go
// มาตรฐานที่ธุรกิจยืนยัน 2026-09-11: pdc_percent / due_percent เก็บเป็นจำนวนเปอร์เซ็นต์
// (1 = 1%, 0.1 = 0.1%) และ baht = price * percent / 100
//
// เดิม parsePercent หารด้วย 100 เมื่อ cell มี % ต่อท้าย ทำให้ cell "3.0%" ถูกเก็บเป็น 0.03
// ซึ่งอ่านเป็น 0.03% ข้อมูลใน DB จึงเล็กไป 100 เท่า 46 แถวสำหรับ pdc และ 48 แถวสำหรับ due
func TestParse_PercentKeepsWholeNumberConvention(t *testing.T) {
	tests := []struct {
		name       string
		pdcPercent string
		duePercent string
		wantPdc    float64
		wantDue    float64
	}{
		{"หนึ่งเปอร์เซ็นต์", "1.0%", "1.5%", 1, 1.5},
		{"สามเปอร์เซ็นต์", "3.0%", "4.0%", 3, 4},
		{"ต่ำกว่าหนึ่งเปอร์เซ็นต์", "0.1%", "0.5%", 0.1, 0.5},
		{"ไม่มีเครื่องหมายเปอร์เซ็นต์", "2.5", "3.5", 2.5, 3.5},
		{"ศูนย์", "0%", "0", 0, 0},
		{"ว่าง", "", "", 0, 0},
		{"มีช่องว่างรอบ", " 2.0% ", " 2.5% ", 2, 2.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sh := baseSheets()
			sh["price_list_group_term"] = [][]string{
				{"company_code", "site_code", "group_code", "term_code", "pdc", "pdc_percent", "due", "due_percent"},
				{testCompany, testSite, "G1", "T1", "0.19", tt.pdcPercent, "0.28", tt.duePercent},
			}

			req, err := buildCreatePricelistRequestFromExcel(buildXlsx(t, sh))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(req.Terms) != 1 {
				t.Fatalf("terms = %d, want 1", len(req.Terms))
			}
			if got := req.Terms[0].PdcPercent; got != tt.wantPdc {
				t.Errorf("PdcPercent = %v, want %v", got, tt.wantPdc)
			}
			if got := req.Terms[0].DuePercent; got != tt.wantDue {
				t.Errorf("DuePercent = %v, want %v", got, tt.wantDue)
			}
		})
	}
}
```

- [ ] **Step 3: รัน test ให้เห็นว่า fail**

Run:
```bash
cd prime-wms-erp-core
go test -v ./internal/services/price-service/ -run "TestParse_Percent"
```
Expected: FAIL
- `TestParse_PercentKeepsWholeNumberConvention/หนึ่งเปอร์เซ็นต์` —
  `PdcPercent = 0.01, want 1`
- `TestParse_PercentAndDecimalNotTruncated` — ค่าที่ได้ยังเป็นเศษส่วน

- [ ] **Step 4: ตัด `/ 100` ออกและแก้คอมเมนต์ที่อธิบายเจตนาผิด**

ใน `upload-pricelist.go` เปลี่ยน

```go
	// Excel percent cells come back already formatted ("1.0%"), not as "0.01".
	// ponytail: precision follows the sheet's own display format (0.0% here);
	// read the raw cell value if a template ever needs more decimals than it shows.
	parsePercent := func(s string) float64 {
		s = strings.TrimSpace(s)
		if strings.HasSuffix(s, "%") {
			v, _ := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(s, "%")), 64)
			return v / 100
		}
		return parseFloat(s)
	}
```

เป็น

```go
	// Excel percent cells come back already formatted ("1.0%"), not as "0.01".
	//
	// pdc_percent / due_percent เก็บเป็นจำนวนเปอร์เซ็นต์ ไม่ใช่เศษส่วน
	// ธุรกิจยืนยันเมื่อ 2026-09-11 ว่า 1 = 1% และ 0.1 = 0.1% และสูตรคือ
	// baht = price * percent / 100 ซึ่งตรงกับที่ฝั่ง web คำนวณอยู่
	// (BasePriceTable.vue calculateUpdateTerm) ฉะนั้นตัดแค่เครื่องหมาย % ทิ้ง
	// ห้ามหารด้วย 100 ซ้ำ
	//
	// ponytail: precision follows the sheet's own display format (0.0% here);
	// read the raw cell value if a template ever needs more decimals than it shows.
	parsePercent := func(s string) float64 {
		s = strings.TrimSpace(s)
		if strings.HasSuffix(s, "%") {
			return parseFloat(strings.TrimSuffix(s, "%"))
		}
		return parseFloat(s)
	}
```

`parseFloat` ทำ `TrimSpace` ให้อยู่แล้ว จึงไม่ต้อง trim ซ้ำ และไม่ต้องใช้
`strconv` ในบล็อกนี้อีก — ตรวจว่ายังมีที่อื่นในไฟล์ใช้ `strconv` อยู่ ถ้าไม่มีแล้ว
ต้องลบ import ออก ไม่งั้น compile ไม่ผ่าน

Run เพื่อตรวจ:
```bash
cd prime-wms-erp-core
grep -c 'strconv\.' internal/services/price-service/upload-pricelist.go
```

- [ ] **Step 5: รัน test ให้ผ่าน**

Run:
```bash
cd prime-wms-erp-core
go test -v ./internal/services/price-service/ -run "TestParse_Percent"
```
Expected: PASS ทุก subtest

- [ ] **Step 6: build และรัน test ทั้ง repo**

Run:
```bash
cd prime-wms-erp-core
go build ./... && go test ./...
```
Expected: build ผ่าน · test ok

- [ ] **Step 7: รัน integration test**

Run:
```bash
cd prime-wms-erp-core
make test-integration-pricelist
```
Expected: ok — ถ้ามี integration test ที่ assert ค่า percent เป็นเศษส่วน
(เช่น `upload-pricelist_integration_test.go:263` ที่ select `pdc_percent, due_percent`)
ต้องแก้ให้ตรงมาตรฐานใหม่พร้อมอธิบายเหตุผลใน commit message

- [ ] **Step 8: Commit และเปิด PR**

```bash
cd prime-wms-erp-core
git add internal/services/price-service/upload-pricelist.go
git add internal/services/price-service/upload-pricelist_parse_test.go
git commit -m "fix: parsePercent ห้ามหารด้วย 100

ธุรกิจยืนยันเมื่อ 2026-09-11 ว่า pdc_percent / due_percent เก็บเป็นจำนวนเปอร์เซ็นต์
1 = 1% และ 0.1 = 0.1% และสูตรคือ baht = price * percent / 100
ซึ่งตรงกับที่ฝั่ง web คำนวณอยู่ และฝั่ง export เป็น passthrough ล้วนจึงไม่มีส่วนผิด

parsePercent หารด้วย 100 เมื่อ cell ใน Excel มี % ต่อท้าย ทำให้ cell '3.0%'
ถูกเก็บเป็น 0.03 ซึ่งอ่านเป็น 0.03% ข้อมูลจึงเล็กไป 100 เท่า
46 แถวสำหรับ pdc_percent และ 48 แถวสำหรับ due_percent

TestParse_PercentAndDecimalNotTruncated ตรึงพฤติกรรมผิดนี้ไว้ แก้ให้ตรงมาตรฐาน
และเพิ่ม TestParse_PercentKeepsWholeNumberConvention ครอบทุกรูปแบบ input"
git push -u origin fix/pricelist-percent-convention
gh pr create --base Develop --title "fix: parsePercent หารด้วย 100 ทำให้ percent ที่นำเข้าจาก Excel เล็กไป 100 เท่า"
```

ใน PR description ต้องระบุชัดว่า **ห้าม merge ขึ้น environment จริงก่อน backfill ข้อมูล**
ตามลำดับที่ผู้ใช้ยืนยัน และอ้างไฟล์
`docs/superpowers/reports/2026-09-11-pricelist-term-percent-backfill.sql`

---

## ลำดับการทำและการปล่อย

1. **PR A** ก่อน — ข้อมูลหาย 76/80 แถว เสียหายหนักสุด แก้ config บรรทัดเดียว
2. **PR B** — ลำดับไม่นิ่งทำให้ทุก issue อื่นตรวจซ้ำยาก ควรจบก่อนเพื่อให้ผลการทดสอบนิ่ง
3. **PR C** — ข้อมูล `before` เสียหายต่อเนื่องทุกครั้งที่มีการคำนวณ ยิ่งช้ายิ่งเสียหายเพิ่ม
4. **PR D** — งานเดี่ยว ไม่พันกับใคร
5. **PR E** — แสดงผล ไม่กระทบข้อมูล
6. **backfill percent** → ตรวจผล → **PR F** ปล่อยท้ายสุด

PR A–E ไม่ผูกกับ backfill ปล่อยได้ทันที

## สิ่งที่แผนนี้ไม่ทำ

- **กู้ค่า `before_*` ของ 1,257 subgroup** ที่หายไปแล้ว — ยังกู้ได้จาก
  `price_list_sub_group_history` แต่ต้องแยกเป็นงานพร้อมรายงานก่อน
- **backfill `pdc_percent` / `due_percent`** — ส่งมอบเป็นไฟล์ `.sql` ให้ผู้ใช้รันเอง
- **`avg_weight_ton`** — `headerName` บอก "Avg.kg stock (Tons)" แต่ `dataMapping` ชี้
  `avg_weight` ซึ่งเป็น kg และไม่พบการหารด้วย 1000 ที่ใดในระบบ รอคำตอบจากผู้ใช้
- **F5** — 147 subgroup ผูกสูตร uom `pcs` ทั้งคู่ ไม่มีสูตร `kg` รอธุรกิจตัดสิน
- **`cmd/.env` ของ warehouse-core ถูก track และมี credential ใน git history** —
  ควรแยกเป็นงาน (ย้ายไป `examples.env` + `.gitignore` + rotate)
- **fallback `1.0` เมื่อไม่มีสต็อก** — ผู้ใช้ตัดสินใจคงไว้แล้ว
- **fallback ชื่อ group ที่ว่าง** — `g.GroupName` มาจาก struct ตรง ๆ ไม่ผ่าน closure
  จึงแยก "ไม่มี record" ไม่ได้ ต้องแก้ที่แหล่งข้อมูลซึ่งอยู่นอกขอบเขต
