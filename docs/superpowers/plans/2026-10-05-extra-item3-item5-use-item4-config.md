# Extra GROUP_1_ITEM_3 / ITEM_5 ใช้ config ของ GROUP_1_ITEM_4 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** ให้หน้า extra ของ GROUP_1_ITEM_3 และ GROUP_1_ITEM_5 ทำงาน/แสดงผลเหมือน GROUP_1_ITEM_4 โดยคัดลอก `config_json` ของ ITEM_4 ไปทับ

**Architecture:** หน้า extra (`prime-wms-web/src/views/price-list/Extra.vue`) สร้างคอลัมน์จาก `price_list_extra_config.config_json` ล้วน ๆ backend ส่งค่าตรง ๆ จึงแก้แค่ข้อมูลด้วย SQL migration หนึ่งไฟล์ (รันมือต่อ environment ตาม convention ของ `migrations/`) และพิสูจน์ด้วย integration test ที่รันไฟล์ migration จริงบน Postgres testcontainer

**Tech Stack:** PostgreSQL, Go 1.25, GORM, testify, testcontainers-go v0.42.0

**Spec:** `docs/superpowers/specs/2026-10-05-extra-item3-item5-use-item4-config-design.md`

**Working dir:** `/home/chonlatee/Desktop/Work/prime-wms/prime-wms-erp-core-extra-item4` (worktree, branch `feature/extra-item3-item5-use-item4-config` แตกจาก `Develop`) — ห้ามทำงานใน branch อื่น

---

## ข้อเท็จจริงที่ยืนยันแล้ว

- Model: `internal/models/pricelist.go:102-113` → ตาราง `price_list_extra_config` (`id uuid`, `group_code`, `is_active`, `config_json`, `create_dtm`, `create_by`, `update_dtm`, `update_by`)
- ชนิดคอลัมน์ `config_json` บน DB จริงไม่พบใน repo → migration คัดลอกค่าคอลัมน์ต่อคอลัมน์ (ใช้ได้ทั้ง text/json/jsonb)
- `migrations/` ไม่มี `.down.sql` — convention คือสร้าง backup table ในไฟล์เดียวกันแล้วห่อ `BEGIN/COMMIT` (ดู `migrations/2026-09-14-price-list-extra-operator-normalize.sql`)
- Integration test ของ package `internal/repositories/priceList` มี `TestMain` สตาร์ท `postgres:16` และตั้ง env `database_gorm_url_prime_erp` ให้แล้ว (`repository_test.go:24-67`); `createSchema()` ไม่มีตาราง `price_list_extra_config` → test ใหม่สร้างเอง
- import path ของ module คือ `prime-erp-core`
- Frontend: `convertExtraConfigToColumns` เช็ก `'PRODUCT_GROUP4'` แบบ literal ไม่มี mapping จาก `PG04` — ไม่กระทบงานนี้เพราะคัดลอกค่าของ ITEM_4 ตรงตัวอักษร ITEM_3/5 จะได้ผลเหมือน ITEM_4 เสมอ ไม่มีโค้ด frontend เปลี่ยน จึงไม่เพิ่ม frontend test (case PRODUCT_GROUP4 มี test อยู่แล้วใน `priceListExtra.spec.ts`)

## ไฟล์

- Create: `migrations/2026-10-05-extra-item3-item5-use-item4-config.sql` — คัดลอก config + backup + rollback snippet
- Create: `internal/repositories/priceList/extra_config_migration_test.go` — integration test รันไฟล์ migration จริง
- Modify: `docs/superpowers/specs/2026-10-05-extra-item3-item5-use-item4-config-design.md` — เปลี่ยน `.down.sql` เป็น backup table และตัด frontend test ให้ตรง convention

---

### Task 1: Integration test (red)

**Files:**
- Create: `internal/repositories/priceList/extra_config_migration_test.go`

- [ ] **Step 1: เขียน test**

```go
package priceListRepository

import (
	"os"
	"testing"

	"prime-erp-core/internal/db"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	extraConfigMigrationFile = "../../../migrations/2026-10-05-extra-item3-item5-use-item4-config.sql"
	extraConfigBackupTable   = "price_list_extra_config_backup_20261005"

	extraConfigItem4 = `[{"group_code":"PG01","is_condition":false,"priority":1},{"group_code":"PG02","is_condition":false,"priority":2},{"group_code":"PG04","is_condition":false,"priority":3},{"group_code":"PG06","is_condition":true,"priority":4}]`
	extraConfigPG03  = `[{"group_code":"PG01","is_condition":false,"priority":1},{"group_code":"PG02","is_condition":false,"priority":2},{"group_code":"PG03","is_condition":false,"priority":3},{"group_code":"PG06","is_condition":true,"priority":4}]`
)

func resetExtraConfig(t *testing.T, gormx *gorm.DB, rows map[string]string) {
	t.Helper()
	require.NoError(t, gormx.Exec(`CREATE TABLE IF NOT EXISTS price_list_extra_config (
		id uuid PRIMARY KEY,
		group_code text,
		is_active boolean,
		config_json jsonb,
		create_dtm timestamptz,
		create_by text,
		update_dtm timestamptz,
		update_by text
	)`).Error)
	require.NoError(t, gormx.Exec(`DROP TABLE IF EXISTS `+extraConfigBackupTable).Error)
	require.NoError(t, gormx.Exec(`TRUNCATE price_list_extra_config`).Error)
	for code, cfg := range rows {
		require.NoError(t, gormx.Exec(`INSERT INTO price_list_extra_config
			(id, group_code, is_active, config_json, create_dtm, create_by, update_dtm, update_by)
			VALUES (?, ?, true, ?::jsonb, now(), 'seed', now(), 'seed')`, uuid.New(), code, cfg).Error)
	}
}

func runExtraConfigMigration(t *testing.T, gormx *gorm.DB) {
	t.Helper()
	sql, err := os.ReadFile(extraConfigMigrationFile)
	require.NoError(t, err)
	require.NoError(t, gormx.Exec(string(sql)).Error)
}

func extraConfigOf(t *testing.T, gormx *gorm.DB, table, groupCode string) string {
	t.Helper()
	var cfg string
	require.NoError(t, gormx.Raw(`SELECT config_json::text FROM `+table+` WHERE group_code = ?`, groupCode).Scan(&cfg).Error)
	return cfg
}

func TestMigration_ExtraItem3Item5UseItem4Config(t *testing.T) {
	gormx, err := db.ConnectGORM("prime_erp")
	require.NoError(t, err)
	defer db.CloseGORM(gormx)

	seed := map[string]string{
		"GROUP_1_ITEM_1": extraConfigPG03,
		"GROUP_1_ITEM_3": extraConfigPG03,
		"GROUP_1_ITEM_4": extraConfigItem4,
		"GROUP_1_ITEM_5": extraConfigPG03,
	}

	t.Run("ITEM_3 และ ITEM_5 ได้ config เท่ากับ ITEM_4 ส่วนแถวอื่นไม่เปลี่ยน", func(t *testing.T) {
		resetExtraConfig(t, gormx, seed)
		runExtraConfigMigration(t, gormx)

		item4 := extraConfigOf(t, gormx, "price_list_extra_config", "GROUP_1_ITEM_4")
		assert.JSONEq(t, extraConfigItem4, item4)
		assert.Equal(t, item4, extraConfigOf(t, gormx, "price_list_extra_config", "GROUP_1_ITEM_3"))
		assert.Equal(t, item4, extraConfigOf(t, gormx, "price_list_extra_config", "GROUP_1_ITEM_5"))
		assert.JSONEq(t, extraConfigPG03, extraConfigOf(t, gormx, "price_list_extra_config", "GROUP_1_ITEM_1"))

		var updateBy string
		require.NoError(t, gormx.Raw(`SELECT update_by FROM price_list_extra_config WHERE group_code = 'GROUP_1_ITEM_3'`).Scan(&updateBy).Error)
		assert.Equal(t, "system", updateBy)
	})

	t.Run("backup เก็บค่าเดิมของ ITEM_3 และ ITEM_5 เท่านั้น", func(t *testing.T) {
		resetExtraConfig(t, gormx, seed)
		runExtraConfigMigration(t, gormx)

		assert.JSONEq(t, extraConfigPG03, extraConfigOf(t, gormx, extraConfigBackupTable, "GROUP_1_ITEM_3"))
		assert.JSONEq(t, extraConfigPG03, extraConfigOf(t, gormx, extraConfigBackupTable, "GROUP_1_ITEM_5"))

		var n int64
		require.NoError(t, gormx.Raw(`SELECT count(*) FROM `+extraConfigBackupTable).Scan(&n).Error)
		assert.EqualValues(t, 2, n)
	})

	t.Run("รันซ้ำได้ และ backup ยังเป็นค่าก่อนรันครั้งแรก", func(t *testing.T) {
		resetExtraConfig(t, gormx, seed)
		runExtraConfigMigration(t, gormx)
		runExtraConfigMigration(t, gormx)

		item4 := extraConfigOf(t, gormx, "price_list_extra_config", "GROUP_1_ITEM_4")
		assert.Equal(t, item4, extraConfigOf(t, gormx, "price_list_extra_config", "GROUP_1_ITEM_3"))
		assert.JSONEq(t, extraConfigPG03, extraConfigOf(t, gormx, extraConfigBackupTable, "GROUP_1_ITEM_3"))
	})

	t.Run("ไม่มีแถว ITEM_4 → ไม่แก้อะไร", func(t *testing.T) {
		resetExtraConfig(t, gormx, map[string]string{
			"GROUP_1_ITEM_3": extraConfigPG03,
			"GROUP_1_ITEM_5": extraConfigPG03,
		})
		runExtraConfigMigration(t, gormx)

		assert.JSONEq(t, extraConfigPG03, extraConfigOf(t, gormx, "price_list_extra_config", "GROUP_1_ITEM_3"))
		assert.JSONEq(t, extraConfigPG03, extraConfigOf(t, gormx, "price_list_extra_config", "GROUP_1_ITEM_5"))
	})

	t.Run("rollback snippet คืนค่าเดิม", func(t *testing.T) {
		resetExtraConfig(t, gormx, seed)
		runExtraConfigMigration(t, gormx)

		require.NoError(t, gormx.Exec(`UPDATE price_list_extra_config t
			SET config_json = b.config_json, update_by = b.update_by, update_dtm = b.update_dtm
			FROM `+extraConfigBackupTable+` b
			WHERE t.id = b.id`).Error)

		assert.JSONEq(t, extraConfigPG03, extraConfigOf(t, gormx, "price_list_extra_config", "GROUP_1_ITEM_3"))
		assert.JSONEq(t, extraConfigPG03, extraConfigOf(t, gormx, "price_list_extra_config", "GROUP_1_ITEM_5"))
	})
}
```

- [ ] **Step 2: รันให้ fail** (ต้องมี Docker)

Run: `go test -v -tags=integration ./internal/repositories/priceList -run TestMigration_ExtraItem3Item5UseItem4Config`
Expected: FAIL ที่ `os.ReadFile` — `no such file or directory`

### Task 2: Migration (green)

**Files:**
- Create: `migrations/2026-10-05-extra-item3-item5-use-item4-config.sql`

- [ ] **Step 1: เขียน migration**

```sql
-- ให้หน้า extra ของ GROUP_1_ITEM_3 และ GROUP_1_ITEM_5 ใช้เงื่อนไขเดียวกับ GROUP_1_ITEM_4
--
-- หน้า extra (Extra.vue → convertExtraConfigToColumns) สร้างคอลัมน์จาก config_json ล้วน ๆ
-- จึงคัดลอก config_json ของ ITEM_4 มาทับตรงตัวอักษร (เดิม ITEM_3/5 ใช้ PG03 แทน PG04)
-- ถ้าไม่มีแถว ITEM_4 คำสั่ง UPDATE จะไม่แก้อะไร
--
-- ข้อมูล extra เดิมของ ITEM_3/5 (price_list_group_extra) ไม่แตะ — ผู้ใช้เลือกค่าใหม่และบันทึกเองในหน้า extra
--
-- ตรวจก่อนรัน:
--   SELECT group_code, config_json FROM price_list_extra_config
--   WHERE group_code IN ('GROUP_1_ITEM_3', 'GROUP_1_ITEM_4', 'GROUP_1_ITEM_5');
--
-- Rollback:
--   UPDATE price_list_extra_config t
--   SET config_json = b.config_json, update_by = b.update_by, update_dtm = b.update_dtm
--   FROM price_list_extra_config_backup_20261005 b
--   WHERE t.id = b.id;

BEGIN;

CREATE TABLE IF NOT EXISTS price_list_extra_config_backup_20261005 AS
SELECT id, group_code, config_json, update_by, update_dtm
FROM price_list_extra_config
WHERE group_code IN ('GROUP_1_ITEM_3', 'GROUP_1_ITEM_5');

UPDATE price_list_extra_config t
SET config_json = src.config_json,
    update_by = 'system',
    update_dtm = now()
FROM price_list_extra_config src
WHERE src.group_code = 'GROUP_1_ITEM_4'
  AND t.group_code IN ('GROUP_1_ITEM_3', 'GROUP_1_ITEM_5');

COMMIT;
```

- [ ] **Step 2: รัน test ให้ผ่าน**

Run: `go test -v -tags=integration ./internal/repositories/priceList -run TestMigration_ExtraItem3Item5UseItem4Config`
Expected: PASS ทั้ง 5 subtest

ถ้า fail ด้วย `cannot insert multiple commands into a prepared statement`: driver ไม่ได้ใช้ simple protocol → ใน `runExtraConfigMigration` เปลี่ยนเป็นเปิด `*sql.DB` จาก `gormx.DB()` แล้ว `sqlDB.Exec(string(sql))` และรันใหม่

- [ ] **Step 3: รัน integration test ทั้ง package กัน regression**

Run: `make test-integration`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add migrations/2026-10-05-extra-item3-item5-use-item4-config.sql internal/repositories/priceList/extra_config_migration_test.go
git commit -m "feat(price-list): extra GROUP_1_ITEM_3/5 use GROUP_1_ITEM_4 config"
```

### Task 3: ปรับ spec ให้ตรงกับที่ทำจริง

**Files:**
- Modify: `docs/superpowers/specs/2026-10-05-extra-item3-item5-use-item4-config-design.md`

- [ ] **Step 1:** ในหัวข้อ "การเปลี่ยนแปลง" แทนข้อ 2 (`.down.sql`) ด้วย: "ไม่มี `.down.sql` ตาม convention ของ `migrations/` — สร้าง `price_list_extra_config_backup_20261005` ในไฟล์เดียวกัน และมี rollback snippet ในคอมเมนต์หัวไฟล์" และลบประโยค "ก่อนเขียน migration ต้องยืนยันชื่อคอลัมน์..." (ยืนยันแล้ว)
- [ ] **Step 2:** ในหัวข้อ "การทดสอบ" ข้อ 1 เปลี่ยน "รัน down" เป็น "รัน rollback snippet" และแทนข้อ 2 ด้วย: "ไม่เพิ่ม frontend test — ไม่มีโค้ด frontend เปลี่ยน, case PRODUCT_GROUP4 มี test อยู่แล้วใน `priceListExtra.spec.ts`"
- [ ] **Step 3: Commit**

```bash
git add docs/superpowers/specs/2026-10-05-extra-item3-item5-use-item4-config-design.md
git commit -m "docs: align extra item3/5 spec with migration convention"
```

### Task 4: ตรวจบน Thaimetal UAT (ต้องให้ผู้ใช้อนุมัติก่อนแตะ DB)

- [ ] **Step 1:** ผู้ใช้รัน query "ตรวจก่อนรัน" ในหัวไฟล์ migration บน DB `prime_erp` ของ Thaimetal UAT → ยืนยันว่ามีแถว ITEM_3/4/5 และ ITEM_4 เป็นค่าที่ต้องการ
- [ ] **Step 2:** ผู้ใช้รันไฟล์ migration บน UAT
- [ ] **Step 3:** เปิด 3 หน้าใน Chrome เทียบหัวคอลัมน์ ต้องตรงกันทั้ง 3:
  - `https://thaimetal-wms-uat.prime-lab.cc/customize/price-list/extra/GROUP_1_ITEM_3`
  - `https://thaimetal-wms-uat.prime-lab.cc/customize/price-list/extra/GROUP_1_ITEM_4`
  - `https://thaimetal-wms-uat.prime-lab.cc/customize/price-list/extra/GROUP_1_ITEM_5`
- [ ] **Step 4:** ลองเพิ่ม/แก้ 1 แถวใน ITEM_3 แล้วกด Update → บันทึกสำเร็จ ไม่มี error overlap

## หมายเหตุ

- Coverage 80%: งานนี้ไม่มีโค้ด Go/TS ใหม่ (มีแต่ SQL) — integration test ครอบทุก path ของ migration (copy, backup, รันซ้ำ, ไม่มี ITEM_4, rollback)
- ห้าม push/merge เข้า `Develop` โดยตรง — เปิด PR จาก `feature/extra-item3-item5-use-item4-config`
