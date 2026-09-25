package utils

import (
	"context"
	"io"
	"net/http"
	"strings"

	"prime-erp-core/internal/requestcontext"

	"github.com/prime-solution-dev/prime-service-x/apilog"
)

// NewRequest สร้าง http.Request ที่พก context และ token ของคนที่ยิงเข้ามาไปด้วย
//
// context ของ Go ไม่ได้วิ่งข้ามไปกับ HTTP มีแต่ header เท่านั้นที่ข้ามไปถึง service ปลายทาง
// ฟังก์ชันนี้จึงเป็นจุดเดียวที่หยิบ token ออกจาก context มาแปะเป็น Authorization
// ทุกจุดที่ยิงออกต้องเรียกผ่านตัวนี้ ไม่ใช้ http.NewRequest ตรงๆ
//
// ไม่มี token ใน context ก็ไม่แปะ header ตัวไหนที่ต้องยิงในนามของระบบ (เช่น cron)
// ต้องใส่ token ลง context เองตั้งแต่จุดเริ่มงาน ดู requestcontext.WithToken
func NewRequest(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error) {

	// trace id ของเราเอง (requestcontext) กับ transaction id ของ apilog เป็นคนละระบบที่แยกกัน
	// มาตั้งแต่ต้น (apilog.WithTransactionID/TransactionIDFromContext) แต่ต้องเป็นเลขเดียวกัน ไม่งั้น
	// จะมีสอง id วิ่งตามกันไปคนละเส้น ทำให้ไล่ log ข้าม service ไม่ได้ จุดนี้เป็นจุดเดียวที่ทำการ
	// เชื่อมสองระบบนี้เข้าด้วยกัน (ทุก http.Client ที่แปะ utils.NewOutboundLogTransport ยิงผ่าน
	// NewRequest นี้เสมอ — ดู internal/guard/no_legacy_context_test.go ที่ห้าม
	// http.NewRequestWithContext ตรงๆ นอกไฟล์นี้ ยกเว้นสองไฟล์ที่คุย third party ซึ่งก็ไม่ได้แปะ
	// transport ตัวนี้อยู่แล้ว) ต้องทำก่อนสร้าง request เพราะ apilog อ่าน transaction id จาก
	// req.Context() ตอน RoundTrip ไม่ใช่จาก header ที่เราแปะเอง
	if traceID, ok := requestcontext.GetTraceID(ctx); ok && strings.TrimSpace(traceID) != "" {
		ctx = apilog.WithTransactionID(ctx, traceID)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	if token, ok := requestcontext.GetToken(ctx); ok && strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}

	// trace id ไม่ได้บอกว่าใครกด แต่บอกว่า request เดียวกันวิ่งผ่าน service ไหนมาบ้าง
	// ส่งต่อไปด้วยเสมอ ปลายทางจะได้ใช้เลขเดิมแทนการออกเลขใหม่
	if traceID, ok := requestcontext.GetTraceID(ctx); ok && strings.TrimSpace(traceID) != "" {
		req.Header.Set(TraceIDHeader, traceID)
	}

	return req, nil
}
