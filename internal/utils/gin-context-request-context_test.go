package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"prime-erp-core/internal/requestcontext"

	"github.com/gin-gonic/gin"
)

// TestUnconvertedCallerMustForwardRequestContext ยืนยันกับดักที่ทำให้ fix round 1 เกิดขึ้น:
// เมื่อ package ที่ยังไม่แปลง (เช่น credit-service, purchase-service, pre-purchase-service)
// เรียกฟังก์ชันที่แปลงเป็น context.Context แล้ว (เช่น approvalService.GetApproval/UpdateApproval/
// CreateApproval) ต้องส่ง ctx.Request.Context() ไม่ใช่ *gin.Context ตรงๆ
//
// *gin.Context implement context.Context ครบทุก method จึง compile ผ่านทั้งสองแบบ แต่ค่าที่ได้
// ไม่เหมือนกัน: gin.Context.Value() เช็คแค่ key ที่เป็น string ก่อน (ของ gin เก็บด้วย c.Set ซึ่งเป็น
// string key) แล้ว fallback ไป Request.Context().Value(key) ก็ต่อเมื่อ engine เปิด
// ContextWithFallback เท่านั้น — requestcontext ใช้ typed key (type contextKey string) ไม่ใช่
// string เปล่า และ repo นี้ (cmd/main.go ใช้ gin.Default()) ไม่ได้เปิด ContextWithFallback ดังนั้น
// การส่ง *gin.Context ตรงๆ ทำให้ requestcontext.GetUserOrDefault คืนค่าว่างเงียบๆ ทุกครั้ง แม้
// middleware จะเก็บ user ไว้ใน c.Request.Context() ถูกต้องแล้วก็ตาม
func TestUnconvertedCallerMustForwardRequestContext(t *testing.T) {
	var gotUserFromBareGinContext string
	var gotUserFromRequestContext string

	// callee จำลอง approvalService.GetApproval/UpdateApproval/CreateApproval ที่แปลงเป็น
	// context.Context แล้ว — ฝั่ง callee ทำถูกอยู่แล้ว (อ่านผ่าน requestcontext.GetUserOrDefault)
	callee := func(ctx context.Context) string {
		return requestcontext.GetUserOrDefault(ctx)
	}

	router := newTestRouter()

	// handler นี้มีรูปร่างเหมือน caller ที่ยังไม่แปลง เช่น
	// credit-service.CreateCreditRequest(ctx *gin.Context, ...) — รับ *gin.Context ตรงๆ
	router.POST("/x", func(c *gin.Context) {
		// วิธีที่เคยเป็นบั๊ก (แก้ไปแล้วใน 6 จุดจริงของ credit/purchase/pre-purchase-service):
		// ส่ง *gin.Context ตรงๆ เข้า callee ที่รับ context.Context — compile ผ่านเพราะ
		// *gin.Context satisfy interface นี้ แต่ callee หา user ไม่เจอ
		gotUserFromBareGinContext = callee(c)

		// วิธีที่ถูกต้อง (แก้แล้วในทุกจุด): ส่ง ctx.Request.Context() ที่ AuthMiddleware
		// ใส่ user ไว้จริง
		gotUserFromRequestContext = callee(c.Request.Context())

		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+tokenOfSomchai)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	// นี่คือกับดัก: ส่ง *gin.Context ตรงๆ ทำให้ callee อ่าน user ไม่เจอ ได้ค่าว่างเงียบๆ
	// (และ [WARN] requestcontext: no user in context จะยิงทั้งที่จริงๆ มี user)
	if gotUserFromBareGinContext != requestcontext.DefaultUser {
		t.Fatalf("ส่ง *gin.Context ตรงๆ ต้องได้ user ว่าง (นี่คือกับดักที่ทำให้ fix round 1 เกิด) ได้ %q แทน", gotUserFromBareGinContext)
	}

	// นี่คือสิ่งที่ต้องเกิดจริงหลัง fix: caller ที่ยังไม่แปลงต้องส่ง ctx.Request.Context()
	// ให้ callee ที่แปลงแล้วเห็น user ที่ถูกต้อง
	if gotUserFromRequestContext != "somchai" {
		t.Fatalf("callee ต้องเห็น user = somchai เมื่อ caller ส่ง ctx.Request.Context() ได้ %q แทน", gotUserFromRequestContext)
	}
}
