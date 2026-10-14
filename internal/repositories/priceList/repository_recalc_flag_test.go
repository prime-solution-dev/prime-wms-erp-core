package priceListRepository

import (
	"testing"
	"time"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedRecalcSubGroup(t *testing.T, gormx *gorm.DB, groupID uuid.UUID, needRecalculate bool) models.PriceListSubGroup {
	t.Helper()
	now := time.Now().UTC()
	subGroup := models.PriceListSubGroup{
		ID:                  uuid.New(),
		PriceListGroupID:    groupID,
		SubgroupKey:         "RECALC_" + uuid.NewString(),
		TotalNetPriceWeight: 45,
		NeedRecalculate:     needRecalculate,
		CreateBy:            "seed",
		CreateDtm:           &now,
		UpdateBy:            "seed",
		UpdateDtm:           &now,
	}
	require.NoError(t, gormx.Omit("PriceListGroup", "PriceListSubGroupKeys").Create(&subGroup).Error)
	return subGroup
}

func needRecalculate(t *testing.T, gormx *gorm.DB, id uuid.UUID) bool {
	t.Helper()
	var subGroup models.PriceListSubGroup
	require.NoError(t, gormx.Where("id = ?", id).First(&subGroup).Error)
	return subGroup.NeedRecalculate
}

func TestUpdatePriceListBase_NeedRecalculate(t *testing.T) {
	gormx, err := db.ConnectGORM("prime_erp")
	require.NoError(t, err)
	defer db.CloseGORM(gormx)

	t.Run("price change flags every subgroup of that group only", func(t *testing.T) {
		seed := seedBaseGroup(t, gormx, "RECALC_CHANGED")
		other := seedBaseGroup(t, gormx, "RECALC_OTHER")
		first := seedRecalcSubGroup(t, gormx, seed.ID, false)
		second := seedRecalcSubGroup(t, gormx, seed.ID, false)
		untouched := seedRecalcSubGroup(t, gormx, other.ID, false)

		payload := payloadFrom(seed)
		payload.PriceWeight = 110
		_, err := UpdatePriceListBase([]models.PriceListGroup{payload, payloadFrom(other)})
		require.NoError(t, err)

		require.True(t, needRecalculate(t, gormx, first.ID))
		require.True(t, needRecalculate(t, gormx, second.ID))
		require.False(t, needRecalculate(t, gormx, untouched.ID), "group ที่ราคาไม่เปลี่ยนต้องไม่ถูก flag")
	})

	t.Run("unchanged group leaves flag as is", func(t *testing.T) {
		seed := seedBaseGroup(t, gormx, "RECALC_UNCHANGED")
		subGroup := seedRecalcSubGroup(t, gormx, seed.ID, false)

		_, err := UpdatePriceListBase([]models.PriceListGroup{payloadFrom(seed)})
		require.NoError(t, err)

		require.False(t, needRecalculate(t, gormx, subGroup.ID))
	})

	t.Run("failed update rolls back the flag", func(t *testing.T) {
		seed := seedBaseGroup(t, gormx, "RECALC_ROLLBACK")
		subGroup := seedRecalcSubGroup(t, gormx, seed.ID, false)

		payload := payloadFrom(seed)
		payload.PriceWeight = 110
		// term ซ้ำ ID เดียวกัน → insert ตัวที่สองชน primary key ภายใน transaction
		payload.PriceListGroupTerms = append(payload.PriceListGroupTerms, payload.PriceListGroupTerms[0])

		_, err := UpdatePriceListBase([]models.PriceListGroup{payload})
		require.Error(t, err)

		require.False(t, needRecalculate(t, gormx, subGroup.ID), "flag ต้อง rollback ไปกับ transaction")
		require.Equal(t, 100.0, reloadGroup(t, gormx, seed.ID).PriceWeight)
	})
}

func TestUpdatePriceListSubGroups_ClearRecalculate(t *testing.T) {
	gormx, err := db.ConnectGORM("prime_erp")
	require.NoError(t, err)
	defer db.CloseGORM(gormx)

	price := 50.0

	t.Run("detail save clears the flag", func(t *testing.T) {
		seed := seedBaseGroup(t, gormx, "RECALC_CLEAR")
		subGroup := seedRecalcSubGroup(t, gormx, seed.ID, true)

		err := UpdatePriceListSubGroups(models.UpdatePriceListSubGroupRequest{
			Changes:          []models.UpdatePriceListSubGroupItem{{SubGroupID: subGroup.ID, TotalNetPriceWeight: &price}},
			ClearRecalculate: true,
		})
		require.NoError(t, err)

		require.False(t, needRecalculate(t, gormx, subGroup.ID))
	})

	t.Run("cascade keeps the flag", func(t *testing.T) {
		seed := seedBaseGroup(t, gormx, "RECALC_CASCADE")
		subGroup := seedRecalcSubGroup(t, gormx, seed.ID, true)

		err := UpdatePriceListSubGroups(models.UpdatePriceListSubGroupRequest{
			Changes: []models.UpdatePriceListSubGroupItem{{SubGroupID: subGroup.ID, TotalNetPriceWeight: &price}},
		})
		require.NoError(t, err)

		require.True(t, needRecalculate(t, gormx, subGroup.ID), "cascade ต้องไม่ล้าง flag")
	})

	t.Run("unknown subgroup returns not found", func(t *testing.T) {
		err := UpdatePriceListSubGroups(models.UpdatePriceListSubGroupRequest{
			Changes:          []models.UpdatePriceListSubGroupItem{{SubGroupID: uuid.New(), TotalNetPriceWeight: &price}},
			ClearRecalculate: true,
		})
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}
