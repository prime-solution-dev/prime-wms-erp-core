package priceService

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func withUpdateBaseStub(t *testing.T, fn func([]models.PriceListGroup) ([]models.PriceListGroup, error)) {
	t.Helper()
	orig := updatePriceListBaseFunc
	updatePriceListBaseFunc = fn
	t.Cleanup(func() { updatePriceListBaseFunc = orig })
}

func basePayload(ids ...uuid.UUID) string {
	s := "["
	for i, id := range ids {
		if i > 0 {
			s += ","
		}
		s += fmt.Sprintf(`{"id":"%s","price_weight":%d}`, id, 100+i)
	}
	return s + "]"
}

func TestUpdatePriceListBase_CascadesOnlyChangedGroups(t *testing.T) {
	idA, idB := uuid.New(), uuid.New()

	var sentToRepo []models.PriceListGroup
	withUpdateBaseStub(t, func(groups []models.PriceListGroup) ([]models.PriceListGroup, error) {
		sentToRepo = groups
		return []models.PriceListGroup{groups[1]}, nil // A ไม่เปลี่ยน, B เปลี่ยน
	})

	var cascadedIDs []uuid.UUID
	calls := 0
	withCascadeStubs(t,
		func(ids []uuid.UUID) ([]string, error) {
			cascadedIDs = ids
			return []string{"GROUP_B"}, nil
		},
		func(context.Context, models.UpdateLatestPriceListSubGroupRequest) (interface{}, error) {
			calls++
			return nil, nil
		})

	if _, err := UpdatePriceListBase(context.Background(), basePayload(idA, idB)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sentToRepo) != 2 {
		t.Fatalf("want every group sent to the repo, got %d", len(sentToRepo))
	}
	if len(cascadedIDs) != 1 || cascadedIDs[0] != idB {
		t.Fatalf("want cascade only for %v, got %v", idB, cascadedIDs)
	}
	if calls != 1 {
		t.Fatalf("want 1 recalculation, got %d", calls)
	}
}

func TestUpdatePriceListBase_NothingChangedSkipsCascade(t *testing.T) {
	withUpdateBaseStub(t, func([]models.PriceListGroup) ([]models.PriceListGroup, error) {
		return []models.PriceListGroup{}, nil
	})

	calls := 0
	withCascadeStubs(t,
		func(ids []uuid.UUID) ([]string, error) {
			if len(ids) > 0 {
				t.Fatalf("want no ids to cascade, got %v", ids)
			}
			return nil, nil
		},
		func(context.Context, models.UpdateLatestPriceListSubGroupRequest) (interface{}, error) {
			calls++
			return nil, nil
		})

	if _, err := UpdatePriceListBase(context.Background(), basePayload(uuid.New())); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 0 {
		t.Fatalf("want no recalculation, got %d", calls)
	}
}

func TestUpdatePriceListBase_RepoErrorStopsBeforeCascade(t *testing.T) {
	repoErr := errors.New("repo boom")
	withUpdateBaseStub(t, func([]models.PriceListGroup) ([]models.PriceListGroup, error) {
		return nil, repoErr
	})

	calls := 0
	withCascadeStubs(t,
		func([]uuid.UUID) ([]string, error) { calls++; return nil, nil },
		func(context.Context, models.UpdateLatestPriceListSubGroupRequest) (interface{}, error) {
			calls++
			return nil, nil
		})

	if _, err := UpdatePriceListBase(context.Background(), basePayload(uuid.New())); !errors.Is(err, repoErr) {
		t.Fatalf("want repo error, got %v", err)
	}
	if calls != 0 {
		t.Fatalf("want cascade untouched on repo error, got %d calls", calls)
	}
}
