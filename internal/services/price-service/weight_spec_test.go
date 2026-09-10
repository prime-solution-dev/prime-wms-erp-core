package priceService

import "testing"

// weight_spec ที่หาไม่ได้จะเป็น 0 ซึ่งถ้าปล่อยเข้าสูตรจะทำให้ราคาเป็น 0
// หรือ division by zero จึงต้อง fallback เป็น 1.0 เฉพาะตอนคำนวณ
// ส่วนการแสดงผลยังต้องโชว์ 0 ตามเดิมเพื่อให้เห็นว่า master data ขาด
func TestWeightSpecForFormula(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{name: "ค่าปกติผ่านไปตรง ๆ", in: 12.5, want: 12.5},
		{name: "0 กลายเป็น 1.0", in: 0, want: 1.0},
		{name: "ค่าติดลบกลายเป็น 1.0", in: -3.2, want: 1.0},
		{name: "ค่าน้อยมากแต่เป็นบวกผ่านไปตรง ๆ", in: 0.001, want: 0.001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := weightSpecForFormula(tt.in); got != tt.want {
				t.Fatalf("weightSpecForFormula(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
