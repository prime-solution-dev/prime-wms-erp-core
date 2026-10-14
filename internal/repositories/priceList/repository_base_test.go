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

func seedBaseGroup(t *testing.T, gormx *gorm.DB, code string) models.PriceListGroup {
	t.Helper()
	now := time.Now().UTC()
	group := models.PriceListGroup{
		ID:                uuid.New(),
		CompanyCode:       "TEST",
		SiteCode:          "TEST",
		GroupCode:         code,
		PriceUnit:         1,
		PriceWeight:       100,
		BeforePriceWeight: 90,
		Currency:          "THB",
		CreateBy:          "seed",
		CreateDtm:         now,
		UpdateBy:          "seed",
		UpdateDtm:         now,
	}
	require.NoError(t, gormx.Omit("PriceListGroupTerms", "PriceListGroupExtras", "PriceListSubGroups").Create(&group).Error)

	term := models.PriceListGroupTerm{
		ID:               uuid.New(),
		PriceListGroupID: group.ID,
		TermCode:         "T30",
		Pdc:              2,
		PdcPercent:       2,
		Due:              3,
		DuePercent:       3,
		CreateBy:         "seed",
		CreateDtm:        &now,
		UpdateBy:         "seed",
		UpdateDtm:        &now,
	}
	require.NoError(t, gormx.Create(&term).Error)

	group.PriceListGroupTerms = []models.PriceListGroupTerm{term}
	return group
}

// payloadFrom สร้าง payload แบบที่ service ส่งให้ repo เมื่อผู้ใช้ไม่ได้แก้อะไรเลย
func payloadFrom(seed models.PriceListGroup) models.PriceListGroup {
	now := time.Now().UTC()
	terms := make([]models.PriceListGroupTerm, len(seed.PriceListGroupTerms))
	for i, term := range seed.PriceListGroupTerms {
		term.UpdateBy = "editor"
		term.UpdateDtm = &now
		terms[i] = term
	}
	return models.PriceListGroup{
		ID:                  seed.ID,
		PriceUnit:           seed.PriceUnit,
		PriceWeight:         seed.PriceWeight,
		Currency:            seed.Currency,
		UpdateBy:            "editor",
		UpdateDtm:           now,
		PriceListGroupTerms: terms,
	}
}

func historyCount(t *testing.T, gormx *gorm.DB, code string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, gormx.Model(&models.PriceListGroupHistory{}).Where("group_code = ?", code).Count(&n).Error)
	return n
}

func reloadGroup(t *testing.T, gormx *gorm.DB, id uuid.UUID) models.PriceListGroup {
	t.Helper()
	var group models.PriceListGroup
	require.NoError(t, gormx.Preload("PriceListGroupTerms").Where("id = ?", id).First(&group).Error)
	return group
}

func TestUpdatePriceListBase_Integration(t *testing.T) {
	gormx, err := db.ConnectGORM("prime_erp")
	require.NoError(t, err)
	defer db.CloseGORM(gormx)

	t.Run("unchanged group is not touched", func(t *testing.T) {
		seed := seedBaseGroup(t, gormx, "BASE_UNCHANGED")

		changed, err := UpdatePriceListBase([]models.PriceListGroup{payloadFrom(seed)})
		require.NoError(t, err)
		require.Empty(t, changed)

		require.Equal(t, int64(0), historyCount(t, gormx, "BASE_UNCHANGED"))
		got := reloadGroup(t, gormx, seed.ID)
		require.Equal(t, 90.0, got.BeforePriceWeight, "before_price_weight must keep the previous snapshot")
		require.Equal(t, "seed", got.UpdateBy)
		require.Len(t, got.PriceListGroupTerms, 1)
		require.Equal(t, "seed", got.PriceListGroupTerms[0].UpdateBy, "terms must not be deleted and re-inserted")
	})

	t.Run("price change is saved", func(t *testing.T) {
		seed := seedBaseGroup(t, gormx, "BASE_PRICE")
		payload := payloadFrom(seed)
		payload.PriceWeight = 110

		changed, err := UpdatePriceListBase([]models.PriceListGroup{payload})
		require.NoError(t, err)
		require.Len(t, changed, 1)
		require.Equal(t, seed.ID, changed[0].ID)

		require.Equal(t, int64(1), historyCount(t, gormx, "BASE_PRICE"))
		got := reloadGroup(t, gormx, seed.ID)
		require.Equal(t, 110.0, got.PriceWeight)
		require.Equal(t, 100.0, got.BeforePriceWeight)
		require.Equal(t, "editor", got.UpdateBy)
	})

	t.Run("term percent change alone is saved", func(t *testing.T) {
		seed := seedBaseGroup(t, gormx, "BASE_TERM")
		payload := payloadFrom(seed)
		payload.PriceListGroupTerms[0].PdcPercent = 2.5

		changed, err := UpdatePriceListBase([]models.PriceListGroup{payload})
		require.NoError(t, err)
		require.Len(t, changed, 1)

		got := reloadGroup(t, gormx, seed.ID)
		require.Len(t, got.PriceListGroupTerms, 1)
		require.Equal(t, 2.5, got.PriceListGroupTerms[0].PdcPercent)
		require.Equal(t, 100.0, got.PriceWeight)
	})

	t.Run("mixed payload returns only changed groups", func(t *testing.T) {
		untouched := seedBaseGroup(t, gormx, "BASE_MIX_SAME")
		edited := seedBaseGroup(t, gormx, "BASE_MIX_EDIT")
		editedPayload := payloadFrom(edited)
		editedPayload.PriceWeight = 95

		changed, err := UpdatePriceListBase([]models.PriceListGroup{payloadFrom(untouched), editedPayload})
		require.NoError(t, err)
		require.Len(t, changed, 1)
		require.Equal(t, edited.ID, changed[0].ID)

		require.Equal(t, int64(0), historyCount(t, gormx, "BASE_MIX_SAME"))
		require.Equal(t, int64(1), historyCount(t, gormx, "BASE_MIX_EDIT"))
	})

	t.Run("failed term insert rolls back the whole update", func(t *testing.T) {
		seed := seedBaseGroup(t, gormx, "BASE_ROLLBACK")
		other := seedBaseGroup(t, gormx, "BASE_ROLLBACK_OTHER")
		payload := payloadFrom(seed)
		payload.PriceWeight = 150
		// ใช้ term ID ของอีก group → PK ชนตอน insert (delete ลบเฉพาะ term ของ group นี้)
		payload.PriceListGroupTerms[0].ID = other.PriceListGroupTerms[0].ID

		changed, err := UpdatePriceListBase([]models.PriceListGroup{payload})
		require.Error(t, err)
		require.Nil(t, changed)

		require.Equal(t, int64(0), historyCount(t, gormx, "BASE_ROLLBACK"))
		got := reloadGroup(t, gormx, seed.ID)
		require.Equal(t, 100.0, got.PriceWeight)
		require.Equal(t, "seed", got.UpdateBy)
		require.Len(t, got.PriceListGroupTerms, 1, "original term must survive the rollback")
	})
}
