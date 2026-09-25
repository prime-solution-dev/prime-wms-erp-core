package guard

import (
	"os"
	"strings"
	"testing"
)

// get-deposit.go และ get-deposits.go คุยกับ third party ภายนอก (https://tmi.trcloud.co/... ใน
// get-deposit.go และ URL ที่ผู้เรียกส่งมาเองใน get-deposits.go) — ดู
// internal/services/interface-service/get-deposit.go, get-deposits.go body ของสอง endpoint นี้
// เป็นบทสนทนากับ vendor ภายนอก ต้องไม่มีทางหลุดลง MongoDB ของเราเอง ห้ามแปะ
// utils.NewOutboundLogTransport (หรือ apilog.NewLoggingTransport ตรงๆ) ที่ client ของสองไฟล์นี้
// เด็ดขาด
//
// เทสนี้เป็นรั้วกันคนแก้ทีหลัง "ทำให้เหมือนที่อื่น" โดยเผลอเติม logging transport เข้าไป — ผ่าน
// อยู่แล้วตั้งแต่ก่อน apilog เข้ามาในระบบ (เพราะยังไม่มีใครอ้างถึง apilog ในสองไฟล์นี้เลย) จะพังก็
// ต่อเมื่อมีคนเพิ่มมันเข้าไปในอนาคต
func TestDepositClientsNeverGetOutboundLoggingTransport(t *testing.T) {
	files := []string{
		"../services/interface-service/get-deposit.go",
		"../services/interface-service/get-deposits.go",
	}

	// เช็คเฉพาะรูปแบบการใช้งานจริง (import/เรียกฟังก์ชัน) ไม่ใช่คำเปล่าๆ — get-deposit.go/
	// get-deposits.go เองมีคอมเมนต์ที่พูดถึงชื่อ apilog/NewOutboundLogTransport ตรงๆ เพื่ออธิบายว่า
	// ทำไมถึงไม่มี ถ้าเช็คแบบ substring เปล่าๆ เทสนี้จะฟ้องคอมเมนต์ของตัวเอง
	bannedPatterns := []string{
		`"github.com/prime-solution-dev/prime-service-x/apilog"`, // import
		"apilog.NewLoggingTransport(",                            // เรียกตรงๆ
		"NewOutboundLogTransport(",                                // ผ่าน utils wrapper
	}

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		text := string(content)

		for _, banned := range bannedPatterns {
			if strings.Contains(text, banned) {
				t.Errorf("%s: เจอ %q — เส้นนี้คุยกับ third party ภายนอก ห้ามแปะ outbound logging transport เด็ดขาด (ดูคอมเมนต์บนเทสนี้)", f, banned)
			}
		}
	}
}
