package purchaseRepository

import "testing"

func TestEvaluatePOLineComplete(t *testing.T) {
	cases := []struct {
		name     string
		base     float64
		received float64
		tol      float64
		active   bool
		want     bool
	}{
		// PC 100, 5% active -> min 95
		{"pc 5% exactly at min", 100, 95, 5, true, true},
		{"pc 5% just below min", 100, 94.9999, 5, true, false},
		{"pc 5% above min", 100, 96, 5, true, true},
		{"pc 5% far below", 100, 50, 5, true, false},

		// KG 3800, 5% active -> min 3610 (the FBB20046C steel case)
		{"kg 5% at min", 3800, 3610, 5, true, true},
		{"kg 5% just below", 3800, 3609.99, 5, true, false},
		{"kg 5% full weight", 3800, 3800, 5, true, true},

		// active=false => 0% exact (tol value ignored)
		{"inactive => exact pass", 100, 100, 5, false, true},
		{"inactive => exact fail one short", 100, 99, 5, false, false},
		{"inactive => over still pass", 100, 101, 5, false, true},

		// active but value 0 => exact
		{"zero tol => exact pass", 100, 100, 0, true, true},
		{"zero tol => exact fail", 100, 99.9999, 0, true, false},

		// base <= 0 can never complete (guards divide-by / free close)
		{"zero base never complete", 0, 100, 5, true, false},
		{"negative base never complete", -5, 100, 5, true, false},

		// float drift: SUM(weight) lands a hair under an exact base -> round4 saves it
		{"float drift exact pass", 3800, 3799.99999999, 0, true, true},

		// over-issue: lower-bound only, no upper cap here
		{"massive over-issue passes", 100, 500, 5, true, true},

		// tolerance clamps
		{"tol over 100 clamps to 100 => min 0", 100, 0, 150, true, true},
		{"negative tol clamps to 0 => exact", 100, 100, -20, true, true},
		{"negative tol exact fails short", 100, 99, -20, true, false},

		// fractional percent (no whole-number assumption)
		{"fractional pct 2.5 at min", 100, 97.5, 2.5, true, true},
		{"fractional pct 2.5 just below", 100, 97.4999, 2.5, true, false},

		// received zero with active tol never completes (base>0)
		{"active tol received zero", 100, 0, 10, true, false},

		// clean 100% tolerance -> min 0 (not via clamp)
		{"clean 100 pct min zero", 100, 0, 100, true, true},

		// credit-note direction: negative received stays open
		{"negative received", 100, -5, 5, true, false},

		// round4 boundary around min=95
		{"round4 just under min", 100, 94.99994, 5, true, false},
		{"round4 rounds up to min", 100, 94.99996, 5, true, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := evaluatePOLineComplete(c.base, c.received, c.tol, c.active)
			if got != c.want {
				t.Fatalf("evaluatePOLineComplete(base=%v received=%v tol=%v active=%v) = %v, want %v",
					c.base, c.received, c.tol, c.active, got, c.want)
			}
		})
	}
}

func TestRound4(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{1.234567, 1.2346},
		{3799.99999999, 3800},
		{0, 0},
		{95.00004, 95},
		{95.00005, 95.0001},
	}
	for _, c := range cases {
		if got := round4(c.in); got != c.want {
			t.Fatalf("round4(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestNormKey(t *testing.T) {
	cases := []struct {
		code string
		item string
		want string
	}{
		{"PO202609-0066", "1", "PO202609-0066|1"},
		{"  po1  ", " 2 ", "PO1|2"},
		{"pO-x", "a", "PO-X|A"},
	}
	for _, c := range cases {
		if got := normKey(c.code, c.item); got != c.want {
			t.Fatalf("normKey(%q,%q) = %q, want %q", c.code, c.item, got, c.want)
		}
	}
}
