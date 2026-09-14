package utils

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// ProcessRequest แปลง error ทุกชนิดเป็น 500 ทำให้แยก "ผู้ใช้ส่งข้อมูลไม่ครบ"
// กับ "ระบบพัง" ไม่ออก BindingError ต้องได้ 400 เหมือนที่ ProcessRequestWithBinding ทำ
func TestProcessRequest_BindingErrorReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString(`{}`))

	ProcessRequest(c, func(*gin.Context, string) (interface{}, error) {
		return nil, &BindingError{Message: "extra_key is required"}
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("extra_key is required")) {
		t.Fatalf("response ต้องมีข้อความของ BindingError: %s", w.Body.String())
	}
}

// error ชนิดอื่นต้องยังได้ 500 เหมือนเดิม
func TestProcessRequest_OtherErrorStillReturns500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString(`{}`))

	ProcessRequest(c, func(*gin.Context, string) (interface{}, error) {
		return nil, errors.New("database is down")
	})

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

// กรณีปกติต้องได้ 200 พร้อม payload ที่ส่งต่อให้ service ครบ
func TestProcessRequest_SuccessReturns200(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString(`{"a":1}`))

	ProcessRequest(c, func(_ *gin.Context, payload string) (interface{}, error) {
		return map[string]string{"echo": payload}, nil
	})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("echo")) || !bytes.Contains(w.Body.Bytes(), []byte(`a`)) {
		t.Fatalf("payload ต้องถูกส่งต่อให้ service: %s", w.Body.String())
	}
}
