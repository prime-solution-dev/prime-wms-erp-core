package priceService

import (
	"strings"
	"testing"
)

// The detail tables reshuffle (and GetPriceDetail picks a random group) when the
// group + sub group query has no ORDER BY, because Postgres is free to change row
// order after any UPDATE.
func TestBuildGroupSubGroupQueryIsOrdered(t *testing.T) {
	query := buildGroupSubGroupQuery(` and plg.group_code in ('GROUP_1_ITEM_5') `)

	idx := strings.Index(query, "ORDER BY")
	if idx == -1 {
		t.Fatal("query has no ORDER BY: detail rows would reshuffle after an update")
	}

	if strings.Index(query, "WHERE") > idx {
		t.Fatal("ORDER BY must come after WHERE")
	}

	order := query[idx:]
	for _, col := range []string{"plg.group_code", "plsg.subgroup_key", "plsg.id"} {
		if !strings.Contains(order, col) {
			t.Errorf("ORDER BY is missing %s, order is not fully deterministic: %q", col, order)
		}
	}

	if !strings.Contains(query, "and plg.group_code in ('GROUP_1_ITEM_5')") {
		t.Error("condition was not interpolated into the query")
	}
}
