# Price List Detail — เรียงลำดับด้วย `group_item.value`

วันที่: 2026-09-11
Branch: `feat/pricelist-sort-by-group-value` (แตกจาก `origin/Develop`)
ขอบเขต: `prime-wms-erp-core` เท่านั้น — ไม่แตะ `prime-wms-web`

## ปัญหา

หน้า Price List Detail (`/customize/price-list/detail/:id`) เรียงค่าทุกแกนด้วย
lexicographic string sort ทำให้ค่าตัวเลขเรียงผิด และแต่ละ tab / แต่ละ product group
เรียงไม่เหมือนกัน

ค่าจริงจาก UAT (`GROUP_1_ITEM_1`, site TMI_WH):

| แกน | ค่าที่ API ส่งมาตอนนี้ |
|-----|------------------------|
| แถว tab "เหล็กแผ่น" (`pg_06`) | `1.2, 1.4, 1.9, 10, 100, 12, 15, 16, 19, 2.3, 2.5, ...` |
| แถว tab "เหล็กแผ่นตัด SIZE" | `10, 100, 12, 15, 2, 20, 6, 8, 9` |
| หัวคอลัมน์ tab "เหล็กแผ่น" | `1250x8'`, `4' x 8'`, `4'x1500`, `4'x2400`, `5' x 10'`, `5' x 20'`, `5'x5700` |
| หัวคอลัมน์ tab "ตัด SIZE" | `100x100 ... 300x300, 40x520, 65x500, 75x75` |

Frontend ไม่ sort อะไรเลย — แสดงตามลำดับที่ API ส่งมาทั้งหมด ดังนั้นแก้ที่ backend
ที่เดียวจบ

## แหล่งความจริงของลำดับ: `group_item.value`

ตาราง `group_item` เก็บค่าตัวเลขของทุก item ไว้แล้ว ครอบคลุมแม้แต่ค่าที่ไม่ใช่ตัวเลข

| Group | ตัวอย่าง |
|-------|----------|
| `PG06` ความหนา (แกนแถว) | `1.2`→`1.20`, `10`→`10.00`, `100`→`100.00` (78 items) |
| `PG05` ขนาดหน้ากว้าง (แกนคอลัมน์) | `4' x 8'`→`32.00`, `5' x 10'`→`50.00`, `4'x1500`→`6,000.00`, `1250x8'`→`10,000.00`, `5'x5700`→`28,500.00`, `75x75`→`5,625.00`, `300x300`→`90,000.00` (40 items) |
| `PG03` เกรด (แกนคอลัมน์ tab special) | `SS400`→`10.00`, `LT`→`21.00` (39 items) |
| `PG02` หมวดย่อย (แกน tab) | `เหล็กแผ่น`→`6.00`, `แผ่นลาย`→`9.00`, `เหล็กแผ่น special`→`19.00`, `เหล็กแผ่นตัด SIZE`→`20.00` |

ข้อจำกัดที่ต้องรับมือ:

- `value` เป็น `varchar` มี thousand separator (`"10,000.00"`) → ต้องตัด `,` ก่อน `ParseFloat`
- `value_int` (`float64`) มีคอลัมน์อยู่แต่เป็น `0` ทั้งตาราง — `SyncGroupMaster` ไม่ populate
  **spec นี้ไม่ใช้ `value_int` และไม่แก้ `SyncGroupMaster`**
- `PG05.value` คือ **ผลคูณสองด้าน (พื้นที่)** ไม่ใช่ความกว้าง จึงเกิดผลข้างเคียงที่ยอมรับแล้ว:
  กลุ่มหน่วยฟุต (32–100) มาก่อนกลุ่มมิลลิเมตรทั้งหมด (6,000+) และ `40x520` (20,800)
  แทรกระหว่าง `125x125` (15,625) กับ `150x150` (22,500)
  ถ้าลำดับไหนไม่ถูกใจ business แก้ที่ master ได้เลย ไม่ต้องแก้โค้ด
- `price_list_sub_group_key.seq` **ใช้แทนไม่ได้** — `upload-pricelist.go:1476` กำหนด
  `Seq = i + 1` คือลำดับของ `PG0x` ในคีย์ (PG01→1, PG05→5) ไม่ใช่ลำดับของค่า

## ทำไมต้องมี ValueIndex

จุดที่ sort อยู่ ~23 จุดใน 13 pattern + `shared.go` เทียบค่าอยู่ 3 รูปแบบ

| รูปแบบ | เทียบจาก | ตัวอย่าง |
|--------|----------|----------|
| ① struct | `SubGroupKeys` ครบ | `pattern_group_1_item_4.go:76`, `_6.go:60`, `shared.go:903`, `:2239`, `:2310` |
| ② row map | เหลือแค่ชื่อ เช่น `rows[i]["product_group_6"]` | `_8.go:79`, `_10.go:45`, `_9/_11/_12/_13`, `_7.go:134`, `_1.go:244` |
| ③ map key | เหลือแค่ string | `_1.go:40,100,131`, `_1.go:298`, `shared.go:1012`, `_3.go:52`, `_4.go:65`, `_5.go:212`, `_7.go:65`, `_8.go:65` |

รูปแบบ ② และ ③ ไม่มี struct ให้อ่าน จึงต้องมีทางแปลง **ชื่อ → ตัวเลข** กลับ
`ValueIndex` คือตัวกลางตัวนั้น — สร้างครั้งเดียวต่อ request จาก `SubGroupKeys` ของข้อมูลชุดนั้น

## การเปลี่ยนแปลง

### 1. `internal/models/pricelist.go` — `PriceListSubGroupKeyResponse` (บรรทัด 236-244)

```go
ValueNumber float64 `json:"value_number"`
HasValue    bool    `json:"-"`   // แยก "value = 0 จริง" (เช่น PG08_5) ออกจาก "ไม่เจอใน group_item"
```

`HasValue` ไม่ส่งออก JSON — เป็นข้อมูลภายในสำหรับ comparator เท่านั้น

### 2. `internal/services/price-service/get-price-detail.go`

`getGroupAndItemMappings()` (บรรทัด 22) โหลด `groupItemMap[item_code] → GetGroupItemResponse`
ซึ่งมี `Value` อยู่แล้ว จึงเติมค่าได้ที่จุดประกอบ key ทั้ง 2 จุด (บรรทัด ~201-210 และ ~255-262)
โดยไม่ต้องยิง query เพิ่ม

```go
valueNumber, hasValue := parseGroupItemValue(groupItemMap, sgk.Value)
// ... ใส่ลง PriceListSubGroupKeyResponse
```

เพิ่ม helper ในไฟล์เดียวกัน:

```go
// parseGroupItemValue อ่าน group_item.value ของ item_code ที่ให้มาแล้วแปลงเป็นตัวเลข
// value เก็บเป็น varchar มี thousand separator เช่น "10,000.00" จึงต้องตัด "," ก่อน
// คืน (0, false) เมื่อไม่มี record หรือ value ว่าง/parse ไม่ได้ ซึ่งต่างจาก (0, true)
// ที่หมายถึง value = "0" จริง
func parseGroupItemValue(m map[string]models.GetGroupItemResponse, code string) (float64, bool)
```

`get-price-export-table.go:128` ประกอบ `PriceListSubGroupKeyResponse` เหมือนกัน ต้องเติมด้วย
เพื่อให้ export เรียงตรงกับหน้าจอ

### 3. `internal/services/price-service/patterns/value_index.go` (ไฟล์ใหม่)

```go
// ValueIndex แปลงชื่อ item (เช่น "4' x 8'") กลับเป็นค่าตัวเลขจาก group_item.value
// ต้องสร้างใหม่ทุก request ห้ามทำเป็น package-level var — Gin รับ request พร้อมกันได้
type ValueIndex struct {
    byGroup map[string]map[string]groupValue // groupCode -> valueName -> ค่า
}

func NewValueIndex(data []models.GetPriceListResponse) *ValueIndex

// Less เทียบชื่อสองตัวในกลุ่มเดียวกัน
func (vi *ValueIndex) Less(groupCode, nameA, nameB string) bool

// LessKeys เทียบ composite key ที่คั่นด้วย "|" ทีละ segment ตาม groupCodes ที่ให้มา
func (vi *ValueIndex) LessKeys(groupCodes []string, keyA, keyB string) bool
```

กติกาของ `Less` เรียงตามลำดับ:

1. มีค่าทั้งคู่ และไม่เท่ากัน → เทียบตัวเลขจากน้อยไปมาก
2. มีค่าฝ่ายเดียว → **ฝ่ายที่มีค่ามาก่อน** (ตัวที่ resolve ไม่ได้ไปท้ายเสมอ ไม่กองอยู่หน้าสุดเพราะถูกมองเป็น 0)
3. ไม่มีค่าทั้งคู่ → เทียบ string ตามเดิม
4. ค่าเท่ากัน → tie-break ด้วยชื่อ เพื่อให้ deterministic
   (เช่น `150x200` กับ `100x300` ที่ value = 30,000.00 เท่ากัน)

### 4. แทนที่จุด sort ทั้งหมด

แต่ละ handler เพิ่ม `vi := NewValueIndex(priceListData)` หนึ่งบรรทัดบนสุด แล้วเปลี่ยน
comparator ทุกจุดให้เรียก `vi.Less(...)` / `vi.LessKeys(...)` แทน `sort.Strings` และ
`a < b` บน string

ครอบคลุมทุกแกน **รวมลำดับ tab** (`_1.go:298`, `_1.go:40`, `_3.go:52`, `_4.go:65`,
`_5.go:212`, `_7.go:65`, `_8.go:65`) ตามที่ตกลงไว้ — ลำดับ tab จะเปลี่ยนจาก
`เหล็กแผ่น, เหล็กแผ่นตัด SIZE, แผ่นลาย, เหล็กแผ่น special` เป็น
`เหล็กแผ่น (6), แผ่นลาย (9), เหล็กแผ่น special (19), เหล็กแผ่นตัด SIZE (20)`

จุดที่ **ไม่ต้องแตะ** เพราะไม่ได้เรียงด้วยค่าของ product group:

- `_1.go:137` เรียงด้วย `<colKey>_row_number`
- tie-break ด้วย `total_weight` (`_10.go:45` และ pattern อื่นที่ merge row) ซึ่งแปลงเป็น
  `float64` ถูกต้องอยู่แล้ว
- tie-break ด้วย `ship_no` (`_8.go:79`)

ข้อบังคับที่ต้องรักษาไว้:

- `shared.go:590-591` ระบุว่าการเรียง subgroup ที่มี `sg.ID` เดียวกันต้องใช้
  `sort.SliceStable` เท่านั้น ห้าม `sort.Slice` มิฉะนั้น "record แรก" จะไม่ใช่
  `inventoryWeights[0]` อีกต่อไป — เปลี่ยนแค่ comparator ไม่เปลี่ยนฟังก์ชัน sort
- `get-pricelist.go:320` `ORDER BY plg.group_code, plg.id, plsg.subgroup_key, plsg.id`
  คงไว้ตามเดิม เป็นตัวกัน non-deterministic ระดับ DB ไม่ใช่ลำดับที่ผู้ใช้เห็น

### 5. Frontend

ไม่แก้ `prime-wms-web` ทั้ง `PriceListItemOneDetail.vue` และ `price-api.service.ts`

## Test

ตาม CLAUDE.md ต้องมี unit + integration test และ coverage ไม่ต่ำกว่า 80%

**Unit — `patterns/value_index_test.go`**

- parse `"10,000.00"` → `10000`
- `value = "0"` → `(0, true)` ต่างจากไม่พบ item → `(0, false)`
- `value = ""` และ parse ไม่ได้ → `(0, false)`
- ตัวที่ resolve ไม่ได้ไปท้ายเสมอ
- ค่าเท่ากัน tie-break ด้วยชื่อ และผลลัพธ์ต้องคงที่เมื่อรันซ้ำ
- `LessKeys` กับ composite key `"PG03|PG08|PG05"`

**Golden order — 1 test ต่อ pattern (13 ตัว)**

fixture ตั้งต้นใช้ค่าจริงจาก UAT ที่ดึงมาแล้ว ยืนยันลำดับ row / column / tab ที่คาดหวัง
เคสบังคับ:

- `1.2, 1.4, 1.9, 10, 100, 12, 15` → `1.2, 1.4, 1.9, 10, 12, 15, 100`
- tab "เหล็กแผ่น" คอลัมน์ → `4' x 8'`, `5' x 10'`, `5' x 20'`, `4'x1500`, `4'x2400`, `1250x8'`, `5'x5700`
- tab "special" คอลัมน์ → `SS400`, `LT`
- ลำดับ tab → `เหล็กแผ่น`, `แผ่นลาย`, `เหล็กแผ่น special`, `เหล็กแผ่นตัด SIZE`

**Integration — `make test-integration`**

ต่อ testcontainers seed `group` / `group_item` แล้วยืนยันว่า `GetPriceDetail` คืน
`value_number` ครบทุก `SubGroupKey` และลำดับตรงกับ golden order

**Regression ที่ต้องไม่พัง**

- `get-price-detail-order_test.go` และ `get-pricelist-order_test.go` ที่มีอยู่
- `build-pricelist-detail-tab_test.go` (ลำดับคอลัมน์ product group ของ export ที่ใช้ `Seq` — คนละแกน ต้องไม่กระทบ)

## สิ่งที่ไม่ทำใน spec นี้

- ไม่ populate `value_int` และไม่แก้ `SyncGroupMaster` ฝั่ง `prime-wms-product-core`
- ไม่ลบ `GetGroupItemValueInt()` (`repositories/priceList/repository.go:152`) ที่เป็น dead code —
  เป็นโค้ดเดิมที่ไม่เกี่ยวกับงานนี้ แจ้งไว้เฉย ๆ
- ไม่เพิ่ม UI ให้ business ตั้งลำดับเอง — ลำดับมาจาก `group_item.value` ที่มีหน้าจัดการอยู่แล้ว
- ไม่ทำ natural sort เป็น fallback — ตัวที่ resolve ไม่ได้ใช้ string compare ตามเดิมและไปท้าย
