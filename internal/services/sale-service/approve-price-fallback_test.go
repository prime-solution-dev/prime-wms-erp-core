package saleService

import "testing"

func TestShouldAdoptQuotationPriceOnlyWhenApproved(t *testing.T) {
	cases := []struct {
		status string
		want   bool
	}{
		{"COMPLETED", true},
		{"REVIEW", false},
		{"REJECT", false},
		{"", false},
	}

	for _, tc := range cases {
		if got := shouldAdoptQuotationPrice(tc.status); got != tc.want {
			t.Errorf("shouldAdoptQuotationPrice(%q) = %v, want %v", tc.status, got, tc.want)
		}
	}
}
