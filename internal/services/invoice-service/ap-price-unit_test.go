package invoiceService

import "testing"

func TestCalculateAPPriceUnit(t *testing.T) {
	for _, tt := range []struct {
		name, poUnit, interfaceUnit string
		price, qty, weight, want    float64
		wantErr                     bool
	}{
		{"same pieces", "Pcs", "Pcs", 100, 10, 25, 100, false},
		{"same kilograms", "Kg", "Kg", 100, 10, 25, 100, false},
		{"pieces to kilograms", "Pcs", "Kg", 100, 10, 25, 40, false},
		{"kilograms to pieces", "Kg", "Pcs", 100, 10, 25, 250, false},
		{"normalize units", " kg ", "KG", 100, 0, 0, 100, false},
		{"zero weight", "Pcs", "Kg", 100, 10, 0, 0, true},
		{"zero qty", "Kg", "Pcs", 100, 0, 25, 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calculateAPPriceUnit(tt.poUnit, tt.interfaceUnit, tt.price, tt.qty, tt.weight)
			if (err != nil) != tt.wantErr || got != tt.want {
				t.Fatalf("got (%v, %v), want (%v, error=%v)", got, err, tt.want, tt.wantErr)
			}
		})
	}
}
