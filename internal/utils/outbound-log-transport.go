package utils

import (
	"net/http"
	"os"
	"strings"

	"github.com/prime-solution-dev/prime-service-x/apilog"
)

// apiLogExcludeEnv คือ list เดียวกับที่ RequestLogMiddleware ใช้กรอง inbound (API_LOG_EXCLUDE)
// — คั่นด้วย comma เช่น "/api/erp/credit/,/api/erp/emailAlert/"
//
// เหตุผลที่ต้องมีไฟล์นี้: apilog (github.com/prime-solution-dev/prime-service-x/apilog)
// เวอร์ชัน v0.1.3 มีฟังก์ชัน shouldLog() ที่อ่าน API_LOG_EXCLUDE/INCLUDE/EXCLUDE_METHODS ก็จริง
// แต่ shouldLog ถูกเรียกจาก LoggingMiddleware (inbound, gin) เท่านั้น — LoggingTransport.RoundTrip
// (ฝั่ง outbound ที่ erp-core ใช้จริง) ไม่เคยเรียก shouldLog เลย ต่อให้ตั้ง API_LOG_EXCLUDE ไว้ที่
// Vault เส้น outbound ก็ยังถูกบันทึกลง Mongo ทุกเส้นอยู่ดี (ดูได้จาก
// $(go env GOMODCACHE)/github.com/prime-solution-dev/prime-service-x@v0.1.3/apilog/transport.go
// — RoundTrip เช็คแค่ store == nil) ที่อันตรายที่สุดคือ cron (credit-request.go,
// credit-request copy.go, credit-extra.go) ที่ยิงทุกนาที ถ้าไม่มีตัวกรองนี้จะเขียน Mongo รัวๆ
// ทั้งที่ Vault ตั้ง exclude ไว้แล้ว
//
// NewOutboundLogTransport จึงเป็นชั้นชดเชย (compensating layer) ที่ห่อ
// apilog.NewLoggingTransport ไว้อีกชั้น แล้วเช็ค path เองก่อนว่าตรงกับ API_LOG_EXCLUDE หรือไม่ —
// ถ้าตรง ให้ยิงตรงผ่าน http.DefaultTransport (ไม่แตะ apilog เลย ไม่มีทางหลุดลง Mongo) ถ้าไม่ตรง
// ค่อยให้ apilog.NewLoggingTransport เป็นคนจัดการตามปกติ ไม่ได้แก้ library (go.mod ห้ามเพิ่ม
// dependency ใหม่และห้ามแก้ vendored code) — วันไหน apilog รุ่นถัดไปแก้ RoundTrip ให้เรียก
// shouldLog(req.URL.Path, req.Method) เองแล้ว ไฟล์นี้ก็ลบทิ้งได้ เปลี่ยนไปเรียก
// apilog.NewLoggingTransport(source) ตรงๆ แทน
//
// รองรับเฉพาะ path-prefix exclude (API_LOG_EXCLUDE) เหมือนฝั่ง inbound ของ repo นี้ — ไม่ทำ
// INCLUDE/EXCLUDE_METHODS ให้ครบตาม apilog เพราะฝั่ง inbound เองก็ไม่ได้ใช้สองตัวนั้น (ดู
// RequestLogMiddleware) การทำให้ outbound ทำมากกว่า inbound จะสร้างพฤติกรรมที่ไม่มีใครขอ
const apiLogExcludeEnv = "API_LOG_EXCLUDE"

// ParseAPILogExcludePrefixes แตก comma-separated prefix list เป็น slice — export ออกมาให้ทั้ง
// internal/middleware (inbound) และไฟล์นี้ (outbound) ใช้ตัวเดียวกัน ไม่ต้องคนละชุด
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

// outboundLogTransport เลือกระหว่าง logged (apilog) กับ direct (http.DefaultTransport) ตอน
// RoundTrip จริง ไม่ใช่ตอนสร้าง client — อ่าน API_LOG_EXCLUDE จาก env สดทุกครั้งที่ยิง request
// (ไม่ cache ไว้ตอนสร้าง) เพราะ client ส่วนใหญ่ในโค้ดนี้ถูกสร้างใหม่ทุกครั้งที่ฟังก์ชันถูกเรียกอยู่
// แล้ว การ cache จะไม่ได้ช่วยอะไรและอาจทำให้เทสที่สลับ env กลางคันงงว่าทำไมค่าไม่อัปเดต
type outboundLogTransport struct {
	logged http.RoundTripper
	direct http.RoundTripper
}

func (t *outboundLogTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	prefixes := ParseAPILogExcludePrefixes(os.Getenv(apiLogExcludeEnv))

	if IsExcludedPath(req.URL.Path, prefixes) {
		return t.direct.RoundTrip(req)
	}

	return t.logged.RoundTrip(req)
}

// newOutboundLogTransport คือ constructor ภายในที่รับ RoundTripper ปลอมได้ ใช้จากเทสในไฟล์นี้
// (package utils เอง ไม่ใช่ utils_test) เพื่อพิสูจน์ได้ว่า path ที่ถูก exclude ไปทาง direct จริง
// โดยไม่ต้องพึ่ง MongoDB — ของจริงมาจาก NewOutboundLogTransport ด้านล่างเท่านั้น
func newOutboundLogTransport(logged, direct http.RoundTripper) http.RoundTripper {
	return &outboundLogTransport{logged: logged, direct: direct}
}

// NewOutboundLogTransport คืน http.RoundTripper สำหรับแปะที่ Transport ของ http.Client ที่ยิง
// ออกไปหา service อื่น — source บอกว่าเรากำลังเรียกใคร (เช่น "order", "warehouse", "erp") ต้อง
// ใช้ชื่อเดียวกันทุกจุดที่เรียก target เดียวกัน เพื่อให้ query จาก Mongo กรองตาม source ได้
//
// path ที่ตรงกับ API_LOG_EXCLUDE จะไม่ถูกส่งผ่าน apilog เลย (ดูคอมเมนต์บนไฟล์นี้ว่าทำไมต้องกรอง
// เอง) — ส่วนที่เหลือถูกส่งผ่าน apilog.NewLoggingTransport(source) ตามปกติ ซึ่งเป็น no-op เงียบๆ
// เองอยู่แล้วถ้า apilog ยังไม่ได้ Init (เช่น ปิด logging ไว้ หรือ Init ล้มเหลว) เพราะเช็ค
// store == nil ก่อนทำอะไรทั้งนั้น
func NewOutboundLogTransport(source string) http.RoundTripper {
	return newOutboundLogTransport(apilog.NewLoggingTransport(source), http.DefaultTransport)
}
