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
}
