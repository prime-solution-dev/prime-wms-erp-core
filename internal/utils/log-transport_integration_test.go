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
// ไม่เพี้ยน — ยิงของจริงผ่าน httptest server ไม่ได้ปลอม RoundTripper
//
// NewOutboundLogTransport เหลือเป็น shim ที่คืน http.DefaultTransport แล้ว (ดู log-transport.go)
// เทสนี้จึงเป็นตัวยืนยันว่า shim โปร่งใสจริง ไม่ได้ไปแตะ body หรือ status ระหว่างทาง ถ้าวันไหนมีคน
// เอา logging transport กลับมาแปะที่นี่ เทสนี้คือด่านแรกที่ต้องยังผ่าน
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
