package saleService

import "testing"

func TestAllSaleItemsClosedCountsCanceledLinesAsDone(t *testing.T) {
	cases := []struct {
		name     string
		statuses []string
		want     bool
	}{
		{"ครบทุกบรรทัด", []string{"COMPLETED", "COMPLETED"}, true},
		{"มีบรรทัดที่ถูกยกเลิกปนอยู่", []string{"COMPLETED", "CANCELED"}, true},
		{"สะกดแบบสองแอล", []string{"COMPLETED", "CANCELLED"}, true},
		{"ยังมีบรรทัดค้าง", []string{"COMPLETED", "PENDING"}, false},
		{"ไม่มีบรรทัดเลย", []string{}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := allSaleItemsClosed(tc.statuses); got != tc.want {
				t.Errorf("allSaleItemsClosed(%v) = %v, want %v", tc.statuses, got, tc.want)
			}
		})
	}
}
