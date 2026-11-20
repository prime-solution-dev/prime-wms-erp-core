package customerService

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// json.Marshal ล้ม (ค่า chan) ต้องคืน error ทันที ไม่ยิง request ด้วย body ว่าง
func TestGetCustomers_MarshalError(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()
	t.Setenv("base_url_customer", srv.URL)

	_, err := GetCustomers(context.Background(), map[string]interface{}{"bad": make(chan int)})
	if err == nil {
		t.Fatal("want marshal error, got nil")
	}
	if called {
		t.Fatal("request must not be sent when marshal fails")
	}
}
