package priceService

// weightSpecForFormula แปลง weight_spec ให้ปลอดภัยต่อการนำไปเข้าสูตรคำนวณราคา
//
// weight_spec มาจาก product master (น้ำหนักของ unit ที่ flag_base = true) ถ้าหาไม่เจอ
// จะได้ 0 ซึ่งในสูตรมักถูกใช้เป็นตัวคูณหรือตัวหาร ปล่อยไปจะได้ราคา 0 หรือ +Inf
// จึงแทนด้วย 1.0 (ค่าเดียวกับ default เดิมของโค้ด)
//
// ฟังก์ชันนี้ใช้เฉพาะตอนคำนวณ การแสดงผลบนกริดและใน API response ยังต้องโชว์ค่าดิบ
// เพื่อให้ผู้ใช้เห็นว่า master data ของสินค้าตัวนั้นยังไม่ครบ
func weightSpecForFormula(weightSpec float64) float64 {
	if weightSpec <= 0 {
		return 1.0
	}
	return weightSpec
}
