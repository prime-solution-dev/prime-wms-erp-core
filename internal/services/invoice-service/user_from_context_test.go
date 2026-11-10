package invoiceService

import (
	"context"
	"testing"

	"prime-erp-core/internal/requestcontext"
)

// ยืนยันว่า package นี้อ่าน user จาก context ไม่ใช่จาก gin
func TestUserFromContext(t *testing.T) {
	ctx := requestcontext.WithUser(context.Background(), "somchai")

	if got := requestcontext.GetUserOrDefault(ctx); got != "somchai" {
		t.Fatalf("user = %q", got)
	}

	if got := requestcontext.GetUserOrDefault(context.Background()); got != requestcontext.DefaultUser {
		t.Fatalf("fallback = %q", got)
	}
}
