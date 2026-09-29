package deliveryService

import (
	"context"
	"testing"

	"prime-erp-core/internal/requestcontext"
)

// เส้นที่ทำงานหลัง commit ต้องเก็บ user/token ไว้ แต่ไม่ตายตาม caller ที่ตัดสายไปแล้ว
func TestPostCommitContextSurvivesCallerCancel(t *testing.T) {
	ctx := requestcontext.WithUser(context.Background(), "somchai")
	ctx = requestcontext.WithToken(ctx, "Bearer token-test")

	canceled, cancel := context.WithCancel(ctx)
	cancel()

	postCommit := postCommitContext(canceled)

	if err := postCommit.Err(); err != nil {
		t.Fatalf("post-commit context ถูกยกเลิกไปด้วย: %v", err)
	}

	if user, _ := requestcontext.GetUser(postCommit); user != "somchai" {
		t.Fatalf("user หาย: %q", user)
	}

	if token, _ := requestcontext.GetToken(postCommit); token != "Bearer token-test" {
		t.Fatalf("token หาย: %q", token)
	}
}
