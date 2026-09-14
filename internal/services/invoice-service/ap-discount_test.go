package invoiceService

import "testing"

func TestCalculateAPDiscount(t *testing.T) {
	for _, tt := range []struct {
		name                                   string
		before, percent, amount, discount, net float64
	}{
		{"fractional percent", 160000, 0.10, 0, 160, 159840},
		{"fixed amount", 94017, 0, 2000, 2000, 92017},
		{"ten percent", 58050, 10, 0, 5805, 52245},
		{"no discount", 13267852.50, 0, 0, 0, 13267852.50},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateAPDiscount(tt.before, tt.percent, tt.amount)
			if got != tt.discount || tt.before-got != tt.net {
				t.Fatalf("discount=%v net=%v; want discount=%v net=%v", got, tt.before-got, tt.discount, tt.net)
			}
		})
	}
}
