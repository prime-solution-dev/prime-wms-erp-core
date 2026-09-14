package deliveryService

import "math"

// roundWeight ปัดน้ำหนัก (KG) เป็น 2 ตำแหน่งแบบ half-up ให้ตรงกับที่หน้าบ้านบันทึก
// ลงเอกสาร (quotation -> sale order -> DBS -> CO) เจ้าของงานเคาะ 2026-09-03 ว่า
// ทุกเอกสารเก็บ 2 ตำแหน่งแล้วคัดลอกต่อ ไม่ให้แต่ละชั้นปัดกันเอง
//
// บวก 1e-9 ก่อนปัด กัน 133.105*100 = 13310.499999 จาก floating point แล้วปัดลงผิด
func roundWeight(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Copysign(math.Floor(math.Abs(v)*100+0.5+1e-9), v) / 100
}
