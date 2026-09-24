package utils

import (
	"context"
	"io"
	"net/http"
	"strings"

	"prime-erp-core/internal/requestcontext"
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
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	if token, ok := requestcontext.GetToken(ctx); ok && strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}

	return req, nil
}
