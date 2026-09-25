package utils

import (
	"errors"
	"net/http"
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
