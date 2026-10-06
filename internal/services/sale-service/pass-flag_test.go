package saleService

import "testing"

func TestNormalizePassFlagKeepsOnlyExplicitN(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"N", "N"},
		{"n", "N"},
		{" N ", "N"},
		{"Y", "Y"},
		{"y", "Y"},
		{"", "Y"},
		{"TRUE", "Y"},
	}

	for _, tc := range cases {
		if got := normalizePassFlag(tc.in); got != tc.want {
			t.Errorf("normalizePassFlag(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
