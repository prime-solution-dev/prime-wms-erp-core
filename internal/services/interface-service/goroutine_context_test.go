package interfaceService

import (
	"context"
	"testing"
	"time"

	"prime-erp-core/internal/requestcontext"
)

// goroutine ที่ยังทำงานต่อหลัง handler คืนค่า ต้องใช้ context ที่ไม่ถูกยกเลิกตาม
func TestBackgroundContextOutlivesRequest(t *testing.T) {
	ctx := requestcontext.WithUser(context.Background(), "somchai")

	requestCtx, done := context.WithCancel(ctx)
	background := backgroundContext(requestCtx)

	done() // handler คืนค่าแล้ว

	select {
	case <-background.Done():
		t.Fatal("งานเบื้องหลังถูกยกเลิกตาม request")
	case <-time.After(10 * time.Millisecond):
	}

	if user, _ := requestcontext.GetUser(background); user != "somchai" {
		t.Fatalf("user หาย: %q", user)
	}
}
