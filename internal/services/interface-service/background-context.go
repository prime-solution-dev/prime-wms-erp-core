package interfaceService

import "context"

// backgroundContext ใช้กับงานที่ปล่อยไว้ใน goroutine แล้วไม่รอ
// เก็บ user/token ไว้ แต่ไม่ถูกยกเลิกตอน handler คืนค่า
func backgroundContext(ctx context.Context) context.Context {
	return context.WithoutCancel(ctx)
}
