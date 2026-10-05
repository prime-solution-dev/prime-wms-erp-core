package utils

import "strings"

// API_LOG_EXCLUDE คือรายการ path prefix (คั่นด้วย comma เช่น "/credit/,/health") ที่ไม่ต้องบันทึก
// ลง service log — erp-core เป็นคนอ่านเอง servicelog library ไม่รู้จักค่านี้
//
// เหตุผลที่ต้องมี: cron ของ repo นี้ (credit-request.go, credit-request copy.go, credit-extra.go)
// ยิงทุกนาทีตลอดเวลา ถ้าไม่กันไว้จะเขียน MongoDB รัวๆ จนกลบ log ของคนใช้งานจริง
//
// ตัวที่อ่าน env แล้วเรียกสองฟังก์ชันนี้คือ internal/middleware.RequestLogMiddleware (ฝั่ง inbound)
// ซึ่งถือ const ชื่อ env ไว้เอง ไฟล์นี้รับแค่ค่าดิบมาแตกกับเทียบ ไม่แตะ os.Getenv เลย เพื่อให้
// เทสได้ตรงๆ โดยไม่ต้องสลับ env
//
// เดิมสองฟังก์ชันนี้อยู่ใน internal/utils/outbound-log-transport.go ซึ่งถูกใช้ร่วมกันระหว่าง inbound
// กับฝั่ง outbound (prime-service-x/apilog) ตอนนี้ apilog ถูกถอดออกจาก repo แล้ว (นโยบาย: ใช้
// servicelog ตัวเดียว) จึงย้ายมาไว้ไฟล์ของตัวเอง ไม่ให้การลบ apilog พา exclude หายไปด้วย
//
// รองรับเฉพาะ prefix match — ไม่ทำ INCLUDE / EXCLUDE_METHODS เพราะฝั่ง inbound ไม่ได้ใช้

// ParseAPILogExcludePrefixes แตกค่าดิบจาก API_LOG_EXCLUDE เป็น slice ของ prefix
// ค่าว่างหรือมีแต่ comma คืน nil (= ไม่ยกเว้นอะไรเลย)
func ParseAPILogExcludePrefixes(raw string) []string {
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	prefixes := make([]string, 0, len(parts))

	for _, part := range parts {
		prefix := strings.TrimSpace(part)
		if prefix == "" {
			continue
		}

		prefixes = append(prefixes, prefix)
	}

	return prefixes
}

// IsExcludedPath เช็คว่า path ตรงกับ prefix ที่ยกเว้นไว้ตัวใดตัวหนึ่งหรือไม่
func IsExcludedPath(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}
