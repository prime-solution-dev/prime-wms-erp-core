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
