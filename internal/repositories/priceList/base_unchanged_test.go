package priceListRepository

import (
	"testing"

	"prime-erp-core/internal/models"
)

func TestIsBasePriceUnchanged(t *testing.T) {
	old := models.PriceListGroup{PriceUnit: 1, PriceWeight: 100}
	oldTerms := []models.PriceListGroupTerm{
		{TermCode: "T30", PdcPercent: 2, DuePercent: 3},
		{TermCode: "T60", PdcPercent: 4, DuePercent: 5},
	}
	same := func() models.PriceListGroup {
		return models.PriceListGroup{
			PriceUnit:   1,
			PriceWeight: 100,
			// ลำดับ term ใน payload ไม่จำเป็นต้องตรงกับ DB
			PriceListGroupTerms: []models.PriceListGroupTerm{
				{TermCode: "T60", PdcPercent: 4, DuePercent: 5},
				{TermCode: "T30", PdcPercent: 2, DuePercent: 3},
			},
		}
	}

	cases := []struct {
		name   string
		mutate func(g *models.PriceListGroup)
		want   bool
	}{
		{"identical", func(g *models.PriceListGroup) {}, true},
		{"float noise below 2 decimals", func(g *models.PriceListGroup) { g.PriceWeight = 100.0000001 }, true},
		{"price weight changed", func(g *models.PriceListGroup) { g.PriceWeight = 100.01 }, false},
		{"price unit changed", func(g *models.PriceListGroup) { g.PriceUnit = 2 }, false},
		{"pdc percent changed", func(g *models.PriceListGroup) { g.PriceListGroupTerms[1].PdcPercent = 2.5 }, false},
		{"due percent changed", func(g *models.PriceListGroup) { g.PriceListGroupTerms[0].DuePercent = 6 }, false},
		{"term added", func(g *models.PriceListGroup) {
			g.PriceListGroupTerms = append(g.PriceListGroupTerms, models.PriceListGroupTerm{TermCode: "T90"})
		}, false},
		{"term removed", func(g *models.PriceListGroup) { g.PriceListGroupTerms = g.PriceListGroupTerms[:1] }, false},
		{"term code replaced", func(g *models.PriceListGroup) { g.PriceListGroupTerms[0].TermCode = "T90" }, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := same()
			tc.mutate(&g)
			if got := isBasePriceUnchanged(old, oldTerms, g); got != tc.want {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}
}
