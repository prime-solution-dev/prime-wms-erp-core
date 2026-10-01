package utils

import "net/http"

// NewOutboundLogTransport ไม่ log อะไรแล้ว — คืน http.DefaultTransport เฉยๆ
//
// เดิมฟังก์ชันนี้ห่อ prime-service-x/apilog เพื่อบันทึก request ที่ erp-core ยิงออกไปหา service อื่น
// ลง MongoDB แต่นโยบายของทีมคือ **ใช้ servicelog ตัวเดียว** (apilog เป็นของอีกทีม) apilog จึงถูก
// ถอดออกจาก repo นี้ทั้งหมด — ดูรั้วที่ internal/guard/no_apilog_anywhere_test.go
//
// ที่คงชื่อฟังก์ชันไว้ ไม่ได้ลบทิ้ง เพราะมันถูกแปะอยู่ที่ Transport ของ http.Client 33 จุด การลบ
// ต้องแก้ 33 ไฟล์พร้อมกันใน repo ที่มีคนอื่น merge อยู่ทุกวัน เสี่ยงชนมากกว่าผลที่ได้ ตัวนี้จึงเป็น
// shim ไว้ให้ call site ไม่ต้องขยับ แล้วค่อยกวาดชื่อทีหลังเป็นงานแยก
//
// ถ้าจะกวาด: ตัดช่อง Transport ออกจาก http.Client ทั้ง 33 จุด (http.Client ใช้
// http.DefaultTransport เองเมื่อ Transport เป็น nil — พฤติกรรมเหมือนเดิมเป๊ะ) แล้วลบไฟล์นี้
//
// พารามิเตอร์ source (เช่น "order", "warehouse") ไม่ถูกใช้แล้ว แต่คงไว้ให้ call site ไม่ต้องแก้
func NewOutboundLogTransport(_ string) http.RoundTripper {
	return http.DefaultTransport
}
