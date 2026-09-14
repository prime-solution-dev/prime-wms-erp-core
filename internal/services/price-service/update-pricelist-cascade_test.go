package priceService

import (
	"errors"
	"testing"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func withCascadeStubs(t *testing.T, codes func([]uuid.UUID) ([]string, error), run func(models.UpdateLatestPriceListSubGroupRequest) (interface{}, error)) {
	t.Helper()
	origCodes, origRun := getPriceListGroupCodesByIDsFunc, runUpdateLatestSubGroupFunc
	getPriceListGroupCodesByIDsFunc, runUpdateLatestSubGroupFunc = codes, run
	t.Cleanup(func() {
		getPriceListGroupCodesByIDsFunc, runUpdateLatestSubGroupFunc = origCodes, origRun
	})
}

// A base price update must recalculate the sub groups, otherwise the detail
// pages keep showing the old "after" price until someone manually triggers
// /price/SubGroup/UpdateLatest.
func TestCascadeBasePriceToSubGroups(t *testing.T) {
	idA, idB := uuid.New(), uuid.New()

	var gotIDs []uuid.UUID
	var gotReq models.UpdateLatestPriceListSubGroupRequest
	calls := 0

	withCascadeStubs(t,
		func(ids []uuid.UUID) ([]string, error) {
			gotIDs = ids
			return []string{"GROUP_1_ITEM_4", "GROUP_1_ITEM_9"}, nil
		},
		func(req models.UpdateLatestPriceListSubGroupRequest) (interface{}, error) {
			calls++
			gotReq = req
			return nil, nil
		})

	err := cascadeBasePriceToSubGroups([]models.PriceListGroup{{ID: idA}, {ID: idB}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("want 1 recalculation call, got %d", calls)
	}
	if len(gotIDs) != 2 || gotIDs[0] != idA || gotIDs[1] != idB {
		t.Fatalf("want ids %v, got %v", []uuid.UUID{idA, idB}, gotIDs)
	}
	if gotReq.UpdateType != "group" {
		t.Fatalf(`want update type "group", got %q`, gotReq.UpdateType)
	}
	if len(gotReq.GroupCodes) != 2 || gotReq.GroupCodes[0] != "GROUP_1_ITEM_4" {
		t.Fatalf("want the resolved group codes, got %v", gotReq.GroupCodes)
	}
}

func TestCascadeBasePriceToSubGroups_NoGroupsIsNoOp(t *testing.T) {
	calls := 0
	withCascadeStubs(t,
		func([]uuid.UUID) ([]string, error) { return nil, nil },
		func(models.UpdateLatestPriceListSubGroupRequest) (interface{}, error) {
			calls++
			return nil, nil
		})

	if err := cascadeBasePriceToSubGroups(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 0 {
		t.Fatalf("want no recalculation when nothing resolved, got %d calls", calls)
	}
}

func TestCascadeBasePriceToSubGroups_PropagatesErrors(t *testing.T) {
	lookupErr := errors.New("lookup boom")
	withCascadeStubs(t,
		func([]uuid.UUID) ([]string, error) { return nil, lookupErr },
		func(models.UpdateLatestPriceListSubGroupRequest) (interface{}, error) { return nil, nil })

	if err := cascadeBasePriceToSubGroups([]models.PriceListGroup{{ID: uuid.New()}}); !errors.Is(err, lookupErr) {
		t.Fatalf("want the lookup error wrapped, got %v", err)
	}

	recalcErr := errors.New("recalc boom")
	withCascadeStubs(t,
		func([]uuid.UUID) ([]string, error) { return []string{"GROUP_1_ITEM_4"}, nil },
		func(models.UpdateLatestPriceListSubGroupRequest) (interface{}, error) { return nil, recalcErr })

	if err := cascadeBasePriceToSubGroups([]models.PriceListGroup{{ID: uuid.New()}}); !errors.Is(err, recalcErr) {
		t.Fatalf("want the recalculation error wrapped, got %v", err)
	}
}
