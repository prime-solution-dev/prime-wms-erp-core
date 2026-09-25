package utils

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func fakeOutboundResponse() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
}

// path ที่ตรงกับ API_LOG_EXCLUDE ต้องไม่ผ่าน apilog เลย (ยิงตรงผ่าน direct transport) — เหตุผลอยู่
// ในคอมเมนต์บนไฟล์ outbound-log-transport.go: apilog v0.1.3 ไม่กรอง path เองตอน outbound
// (LoggingTransport.RoundTrip ไม่เคยเรียก shouldLog) ต่อให้ตั้ง API_LOG_EXCLUDE ไว้ apilog เองก็ยัง
// log ทุกเส้นอยู่ดีถ้าไม่มีชั้นกรองนี้
func TestOutboundLogTransportRoutesExcludedPathDirect(t *testing.T) {
	t.Setenv("API_LOG_EXCLUDE", "/api/erp/credit/,/api/erp/emailAlert/")

	loggedHit := false
	directHit := false

	logged := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		loggedHit = true
		return fakeOutboundResponse(), nil
	})
	direct := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		directHit = true
		return fakeOutboundResponse(), nil
	})

	transport := newOutboundLogTransport(logged, direct)

	req, err := http.NewRequest(http.MethodPost, "https://example.local/api/erp/credit/GetCredit", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}

	if loggedHit {
		t.Fatalf("path ที่ถูก exclude ต้องไม่ผ่าน apilog transport")
	}
	if !directHit {
		t.Fatalf("path ที่ถูก exclude ต้องยิงผ่าน direct transport")
	}
}

// path ที่ไม่ตรง exclude ต้องผ่าน apilog ตามปกติ
func TestOutboundLogTransportRoutesNonExcludedPathThroughLogged(t *testing.T) {
	t.Setenv("API_LOG_EXCLUDE", "/api/erp/credit/,/api/erp/emailAlert/")

	loggedHit := false
	directHit := false

	logged := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		loggedHit = true
		return fakeOutboundResponse(), nil
	})
	direct := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		directHit = true
		return fakeOutboundResponse(), nil
	})

	transport := newOutboundLogTransport(logged, direct)

	req, err := http.NewRequest(http.MethodPost, "https://example.local/order/Order/CreateOrders", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}

	if !loggedHit {
		t.Fatalf("path ที่ไม่ถูก exclude ต้องผ่าน apilog transport")
	}
	if directHit {
		t.Fatalf("path ที่ไม่ถูก exclude ต้องไม่ยิงผ่าน direct transport")
	}
}

// =========================================================
// OUTBOUND LOGGING GATE (API_LOG_OUTBOUND_ENABLED)
//
// Code review พบ Critical 2 ข้อที่อยู่ใน library เอง (prime-service-x/apilog v0.1.3,
// ดูคอมเมนต์เต็มบน OutboundLogEnabled ใน outbound-log-transport.go):
//   1. transport.go เก็บ request/response body เป็น string ดิบ ไม่ redact เลย (มีแค่ truncate)
//   2. transport.go เรียก store.Write(req.Context(), entry) โดยไม่มี timeout ของตัวเอง และทิ้ง
//      error ที่ได้กลับมา
// ตราบใดที่สองข้อนี้ยังไม่ถูกแก้ที่ library ต้นทาง outbound logging (apilog) ต้องปิดอยู่โดย
// default เสมอ ห้ามเปิดเองจาก API_LOG_ENABLED (ตัวนั้นฝั่ง inbound/servicelog ใช้คนละตัว) — เทส
// ด้านล่างพิสูจน์ผ่าน seam (SetNewLoggingTransportForTest) ว่า apilog ไม่ถูกแตะเลยตอนปิด และถูก
// ใช้จริงตอนเปิด ไม่ใช่เช็คจากการอ่าน source
// =========================================================

// ไม่ได้ตั้ง API_LOG_OUTBOUND_ENABLED ไว้เลย (ค่า default) → NewOutboundLogTransport ต้องไม่แตะ
// apilog เลย (seam ต้องไม่ถูกเรียก) แต่ request ยังต้องสำเร็จตามปกติผ่าน direct transport — ใช้
// httptest server จริง (ไม่ใช่ host ปลอม) เพราะตอนปิด transport ที่ได้คือ http.DefaultTransport
// ตรงๆ ซึ่งจะยิงเครือข่ายจริง
func TestNewOutboundLogTransportDisabledByDefaultNeverTouchesApilog(t *testing.T) {
	t.Setenv("API_LOG_OUTBOUND_ENABLED", "")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	seamCalled := false
	restore := SetNewLoggingTransportForTest(func(source string) http.RoundTripper {
		seamCalled = true
		return roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return fakeOutboundResponse(), nil
		})
	})
	defer restore()

	transport := NewOutboundLogTransport("test-source")

	req, err := http.NewRequest(http.MethodPost, server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, ต้องการ %d", resp.StatusCode, http.StatusOK)
	}

	if seamCalled {
		t.Fatalf("apilog seam ถูกเรียกทั้งที่ API_LOG_OUTBOUND_ENABLED ปิดอยู่ (default) — ต้องไม่แตะ apilog เลย")
	}
}

// ตั้ง API_LOG_OUTBOUND_ENABLED=true ชัดเจน → NewOutboundLogTransport ต้องกลับไปใช้เส้นทาง apilog
// เหมือนเดิม (ผ่าน seam) ไม่งั้น flag นี้จะกลายเป็น dead code ที่เปิดยังไงก็ไม่มีผล
func TestNewOutboundLogTransportExplicitlyEnabledUsesApilogSeam(t *testing.T) {
	t.Setenv("API_LOG_OUTBOUND_ENABLED", "true")

	seamCalled := false
	restore := SetNewLoggingTransportForTest(func(source string) http.RoundTripper {
		seamCalled = true
		if source != "test-source" {
			t.Fatalf("seam ได้ source = %q, ต้องการ test-source", source)
		}
		return roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return fakeOutboundResponse(), nil
		})
	})
	defer restore()

	transport := NewOutboundLogTransport("test-source")

	req, err := http.NewRequest(http.MethodPost, "https://example.local/order/Order/CreateOrders", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	if !seamCalled {
		t.Fatalf("apilog seam ต้องถูกเรียกตอน API_LOG_OUTBOUND_ENABLED=true")
	}
}

// ไม่ได้ตั้ง API_LOG_EXCLUDE ไว้เลย (ค่าว่าง) ทุก path ต้องผ่าน apilog ตามปกติ ไม่ exclude อะไร
func TestOutboundLogTransportNoExcludeConfiguredLogsEverything(t *testing.T) {
	t.Setenv("API_LOG_EXCLUDE", "")

	loggedHit := false
	logged := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		loggedHit = true
		return fakeOutboundResponse(), nil
	})
	direct := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("direct transport ไม่ควรถูกเรียกตอนไม่มี exclude")
	})

	transport := newOutboundLogTransport(logged, direct)

	req, err := http.NewRequest(http.MethodPost, "https://example.local/order/Order/CreateOrders", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}

	if !loggedHit {
		t.Fatalf("ไม่มี exclude ต้อง log ทุก path")
	}
}
