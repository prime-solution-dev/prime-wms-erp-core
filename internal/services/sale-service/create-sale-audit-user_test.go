package saleService

import (
	"context"
	"testing"

	"prime-erp-core/internal/requestcontext"
)

// คนกดคือสมชาย แต่ body ส่งชื่อคนอื่นมา ชื่อที่ลง create_by ต้องเป็นสมชาย
// ไม่งั้นใครยิง API ตรงๆ ก็เขียนชื่อคนอื่นลงฐานข้อมูลได้
func TestAuditUserPrefersTokenOverRequestBody(t *testing.T) {
	ctx := requestcontext.WithUser(context.Background(), "somchai")

	if got := auditUserFrom(ctx, "someone-else"); got != "somchai" {
		t.Fatalf("create_by = %q, ต้องเป็นชื่อคนที่ถือ token", got)
	}
}

// ไม่มี token มาด้วย (เช่นเครื่องมือภายในที่ยังไม่ส่ง) ให้ใช้ค่าจาก body ตามของเดิม
// ดีกว่าเขียนค่าว่างทับชื่อที่เคยมี
func TestAuditUserFallsBackToRequestBodyWithoutToken(t *testing.T) {
	if got := auditUserFrom(context.Background(), "from-body"); got != "from-body" {
		t.Fatalf("create_by = %q, ต้องตกไปใช้ค่าจาก body", got)
	}
}

// ไม่มีทั้ง token และ body ก็ต้องไม่พัง และต้องได้ค่าว่างตามกติกาของโปรเจกต์
func TestAuditUserEmptyWhenNothingKnown(t *testing.T) {
	if got := auditUserFrom(context.Background(), ""); got != "" {
		t.Fatalf("create_by = %q, ต้องเป็นค่าว่าง", got)
	}
}
