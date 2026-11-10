// Package requestcontext เก็บข้อมูลของคนที่ยิง request เข้ามา (user, token, trace id)
// ไว้ใน context.Context เพื่อให้ service ชั้นในและเส้นที่ยิงต่อไป service อื่นหยิบไปใช้ได้
// โดยไม่ต้องพึ่ง *gin.Context
//
// ของที่ middleware เก็บด้วย c.Set() จะอยู่ใน map ของ gin ซึ่ง c.Request.Context()
// มองไม่เห็น การเก็บผ่าน package นี้จึงเป็นทางเดียวที่ service แบบ context.Context จะอ่านเจอ
//
// ไฟล์นี้ต้องเหมือนกันทุก repo
package requestcontext

import (
	"context"
	"log"
)

type contextKey string

const (
	userKey    contextKey = "user"
	tokenKey   contextKey = "authorization"
	traceIDKey contextKey = "trace_id"
)

// DefaultUser ใช้เมื่อหา user ใน context ไม่เจอ เช่นถูกเรียกจากระบบที่ไม่มี token
//
// เป็นค่าว่าง ไม่ใช่ชื่อสมมติ เพราะ "ไม่รู้ว่าใคร" กับ "ระบบเป็นคนทำ" คนละเรื่องกัน
// เส้นที่ระบบเป็นคนทำจริง เช่น cron ต้องใส่ชื่อของตัวเองลง context เอง (ดู CronUser)
const DefaultUser = ""

// User

func WithUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, userKey, user)
}

func GetUser(ctx context.Context) (string, bool) {
	user, ok := ctx.Value(userKey).(string)
	return user, ok
}

// GetUserOrDefault อ่านชื่อ user จาก context ถ้าไม่มีจะคืนค่าว่าง พร้อม log เตือนหนึ่งบรรทัด
// ใช้ตัวนี้กับทุกจุดที่เขียน create_by / update_by
//
// log ไว้ไล่หาเส้นที่ยังส่ง token ต่อไม่ครบระหว่างที่ทยอยแปลงทีละ service
func GetUserOrDefault(ctx context.Context) string {
	if user, ok := GetUser(ctx); ok && user != "" {
		return user
	}

	log.Printf("[WARN] requestcontext: no user in context, writing empty user")

	return DefaultUser
}

// Token — เก็บค่า Authorization header ดิบ (รวมคำว่า "Bearer ") ไว้ส่งต่อให้ service ปลายทาง

func WithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenKey, token)
}

func GetToken(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(tokenKey).(string)
	return token, ok
}

// Trace ID

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

func GetTraceID(ctx context.Context) (string, bool) {
	traceID, ok := ctx.Value(traceIDKey).(string)
	return traceID, ok
}
