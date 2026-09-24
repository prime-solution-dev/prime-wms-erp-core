package CronjobService

import (
	"context"
	"testing"

	"prime-erp-core/internal/requestcontext"
)

// GetKernalManual ต้องส่ง context ที่ยกเลิกไม่ได้เข้า GetKernal
//
// ถ้าใครย้อนกลับไปเรียก GetKernal(ctx) ตรงๆ (ไม่ผ่าน context.WithoutCancel) เทสนี้ต้องพัง:
// พอ caller (เช่น client ปิดการเชื่อมต่อ หรือ gateway timeout) ยกเลิก request context ระหว่างทาง
// งาน credit ที่ยังทำอยู่เบื้องหลัง (UpdateCreditRequest -> CreateCreditTransaction) จะขาดตอน
// ทิ้ง state ครึ่งๆ กลางๆ ไว้
func TestGetKernalManualUsesUncancellableContext(t *testing.T) {
	var got context.Context

	original := runKernal
	runKernal = func(ctx context.Context) {
		got = ctx
	}
	t.Cleanup(func() {
		runKernal = original
	})

	ctx := requestcontext.WithUser(context.Background(), "somchai")
	ctx = requestcontext.WithToken(ctx, "Bearer token-test")

	canceled, cancel := context.WithCancel(ctx)
	cancel()

	if _, err := GetKernalManual(canceled, ""); err != nil {
		t.Fatalf("GetKernalManual คืน error: %v", err)
	}

	if got == nil {
		t.Fatalf("runKernal ไม่ถูกเรียกเลย")
	}

	if err := got.Err(); err != nil {
		t.Fatalf("context ที่ส่งเข้า GetKernal ถูกยกเลิกไปด้วย: %v", err)
	}

	if user, _ := requestcontext.GetUser(got); user != "somchai" {
		t.Fatalf("user หาย: %q", user)
	}

	if token, _ := requestcontext.GetToken(got); token != "Bearer token-test" {
		t.Fatalf("token หาย: %q", token)
	}
}
