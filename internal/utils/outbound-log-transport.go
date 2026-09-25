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

// apiLogOutboundEnabledEnv คือ explicit opt-in ของ outbound logging (apilog) เท่านั้น — แยกจาก
// API_LOG_ENABLED (inbound/servicelog) โดยตั้งใจ ค่า default (unset/อะไรก็ตามที่ไม่ใช่ "true"
// เป๊ะๆ) คือ "ปิด"
//
// เหตุผล — code review เจอ Critical 2 ข้อใน apilog (prime-service-x@v0.1.3/apilog/transport.go)
// ที่แก้จาก erp-core ไม่ได้ (อยู่ใน library, ห้ามแก้ vendored code / ห้ามเพิ่ม dependency):
//
//  1. NO REDACTION — RoundTrip เก็บ request body (บรรทัด ~32-37) และ response body (~61-64) เป็น
//     string ดิบเข้า entry.ReqBody/RespBody มีแค่ truncateBody() ตัดความยาว "ไม่มี" การ redact
//     เหมือนฝั่ง inbound (safeLogBody/redactLogValue ใน request-log-middleware.go) erp-core เรียก
//     service อื่นด้วยชื่อลูกค้า/เบอร์โทร/ที่อยู่/เลขผู้เสียภาษีอยู่ในทุก call แทบทั้งหมด ถ้าเปิด
//     logging เส้นนี้ข้อมูลพวกนี้จะลง Mongo เป็น plain text (header ไม่มีปัญหา — maskHeaders มา
//     mask Authorization ให้แล้ว)
//  2. NO TIMEOUT / SWALLOWED ERROR — บรรทัด ~57 และ ~70 เรียก store.Write(req.Context(), entry)
//     ตรงๆ ไม่มี timeout ของตัวเอง (ใช้ context ของผู้เรียกล้วนๆ) และไม่เช็ค error ที่ได้กลับมาเลย
//     ถ้า Mongo ช้าหรือต่อไม่ได้ request ทั้งเส้นจะค้างรอ Mongo driver server-selection (~30s)
//     โดยไม่มีอะไรขึ้น log ให้รู้เลยว่าทำไมช้า
//
// จนกว่า library จะแก้สองข้อนี้ (เพิ่ม redaction ก่อน store หรือ hook ให้ erp-core redact เองได้,
// และให้ store.Write มี timeout ของตัวเองที่ log error เมื่อ Mongo ช้า/ล้ม) ห้ามเปิด flag นี้ใน
// Vault ของ production เด็ดขาด — ตั้งใจให้ default ปิดแม้ API_LOG_ENABLED=true (inbound ยังทำงาน
// ตามปกติ ไม่เกี่ยวกัน) เพื่อไม่ให้ใครเปิด outbound logging ได้โดยไม่ตั้งใจ
const apiLogOutboundEnabledEnv = "API_LOG_OUTBOUND_ENABLED"

// OutboundLogEnabled อ่าน API_LOG_OUTBOUND_ENABLED สดจาก env ทุกครั้งที่เรียก (เหมือน
// API_LOG_EXCLUDE ด้านบน) — ตั้งใจให้ strict เท่ากับ "true" เท่านั้น (ไม่ใช้ strconv.ParseBool ที่
// รับ "1"/"T"/ฯลฯ ด้วย) เพราะนี่คือ flag ที่ต้อง "เปิดเอง" อย่างตั้งใจเท่านั้น ค่าอื่นทุกค่ารวมถึง
// unset ต้องตีความว่าปิดเสมอ ไม่ใช่ error ไปทางเปิดโดยไม่ได้ตั้งใจ
//
// ใช้ทั้งจาก NewOutboundLogTransport ด้านล่าง (gate ตอนสร้าง transport) และจาก
// internal/middleware.RequestLogMiddleware (gate ตอน apilog.Init) เพื่อให้สองจุดอ่านค่าเดียวกัน
// ด้วยตรรกะเดียวกันเสมอ
func OutboundLogEnabled() bool {
	return os.Getenv(apiLogOutboundEnabledEnv) == "true"
}

// newLoggingTransportFunc คือ signature ของ apilog.NewLoggingTransport แยกเป็น seam เพื่อให้เทส
// พิสูจน์ได้ว่า apilog ไม่ถูกแตะเลยตอน flag ปิด (และถูกใช้จริงตอน flag เปิด) โดยไม่ต้องพึ่ง MongoDB
// จริง — เหตุผลเดียวกับ APILogInitiator ใน api-log.go
type newLoggingTransportFunc func(source string) http.RoundTripper

// activeNewLoggingTransport ปลายทางจริงที่ NewOutboundLogTransport เรียกตอน flag เปิด ค่าเริ่มต้น
// คือของจริง (apilog.NewLoggingTransport) เทสค่อยสลับด้วย SetNewLoggingTransportForTest
var activeNewLoggingTransport newLoggingTransportFunc = func(source string) http.RoundTripper {
	return apilog.NewLoggingTransport(source)
}

// SetNewLoggingTransportForTest สลับปลายทางของ apilog.NewLoggingTransport เป็นของปลอมสำหรับเทส
// คืนฟังก์ชันไว้เรียก (defer/t.Cleanup) เพื่อสลับกลับเป็นของจริงเสมอ — เรียกได้จาก _test.go เท่านั้น
// และไม่ปลอดภัยกับเทสที่รันแบบ parallel เพราะ activeNewLoggingTransport เป็น package-level var
func SetNewLoggingTransportForTest(fn func(source string) http.RoundTripper) func() {
	previous := activeNewLoggingTransport
	activeNewLoggingTransport = fn

	return func() {
		activeNewLoggingTransport = previous
	}
}

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
// ถ้า OutboundLogEnabled() คืน false (default — ดูคอมเมนต์บน apiLogOutboundEnabledEnv ว่าทำไม)
// ฟังก์ชันนี้คืน http.DefaultTransport ตรงๆ โดย "ไม่เรียก" apilog.NewLoggingTransport เลยแม้แต่
// ครั้งเดียว (ไม่ใช่แค่ไม่ log — ไม่มีการสร้าง object ของ apilog ขึ้นมาด้วยซ้ำ) เพื่อไม่ให้มีทางที่
// body ของ request/response จะถูกอ่านหรือเก็บโดย apilog ได้เลย ต่อให้โค้ดของ apilog เปลี่ยนไปในอนาคต
//
// เปิดจริงเมื่อ OutboundLogEnabled() คืน true เท่านั้น — path ที่ตรงกับ API_LOG_EXCLUDE ยังคงไม่ถูก
// ส่งผ่าน apilog เหมือนเดิม (ดูคอมเมนต์บนไฟล์นี้ว่าทำไมต้องกรองเอง) ส่วนที่เหลือถูกส่งผ่าน
// apilog.NewLoggingTransport(source) (ผ่าน seam activeNewLoggingTransport) ตามปกติ ซึ่งเป็น no-op
// เงียบๆ เองอยู่แล้วถ้า apilog ยังไม่ได้ Init (เช่น Init ล้มเหลว) เพราะเช็ค store == nil ก่อนทำ
// อะไรทั้งนั้น
func NewOutboundLogTransport(source string) http.RoundTripper {
	if !OutboundLogEnabled() {
		return http.DefaultTransport
	}

	return newOutboundLogTransport(activeNewLoggingTransport(source), http.DefaultTransport)
}
