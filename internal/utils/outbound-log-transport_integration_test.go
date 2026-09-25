package utils_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"prime-erp-core/internal/utils"
)

// client ที่แปะ utils.NewOutboundLogTransport ต้องส่ง body ออกไปไม่เพี้ยน และเอา response กลับมา
// ไม่เพี้ยน
//
// เทสนี้ไม่ได้เรียก utils.InitAPILog ก่อน apilog.store จึงเป็น nil ตาม default ของ process เทส
// (apilog เป็น package-level var, ไม่มี seam ให้ปลอม MongoDB store จากนอก package) พฤติกรรมที่เทส
// ได้คือ passthrough ล้วนๆ — ไม่ได้ครอบคลุม path ที่ apilog อ่าน/re-buffer body จริงตอน store != nil
// (โค้ดส่วนนั้นมีเทสของ library เองแล้วที่ apilog/transport_test.go)
func TestOutboundLogTransportPassesRequestAndResponseUnchanged(t *testing.T) {
	var gotBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"result":"ok"}`))
	}))
	defer server.Close()

	client := &http.Client{Transport: utils.NewOutboundLogTransport("test-source")}

	req, err := http.NewRequest(http.MethodPost, server.URL, bytes.NewBufferString(`{"hello":"world"}`))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	if gotBody != `{"hello":"world"}` {
		t.Fatalf("ปลายทางได้ body = %q, ต้องการ body เดิมไม่เพี้ยน", gotBody)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, ต้องการ %d", resp.StatusCode, http.StatusCreated)
	}

	if string(respBody) != `{"result":"ok"}` {
		t.Fatalf("response body = %q, ต้องการ response เดิมไม่เพี้ยน", string(respBody))
	}
}
