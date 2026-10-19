package externalService

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prime-erp-core/config"
	"prime-erp-core/internal/requestcontext"
)

// เทสนี้ยืนยันว่า token ของคนที่ยิงเข้ามา ไปโผล่ที่ service ปลายทางจริง
// ไม่ใช่แค่ถูกใส่ไว้ใน context ของเราเอง
func TestCancelOrderForwardsCallerToken(t *testing.T) {
	gotAuth := ""

	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")

		json.NewEncoder(w).Encode(CancelOrderResponse{Status: "success"})
	}))
	defer downstream.Close()

	original := config.CANCEL_ORDER_ENDPOINT
	config.CANCEL_ORDER_ENDPOINT = downstream.URL
	defer func() { config.CANCEL_ORDER_ENDPOINT = original }()

	ctx := requestcontext.WithToken(context.Background(), "Bearer token-ของสมชาย")

	res, err := CancelOrder(ctx, CancelOrderRequest{DocumentRef: []string{"DBS2609-0001"}})
	if err != nil {
		t.Fatalf("CancelOrder: %v", err)
	}

	if res.Status != "success" {
		t.Fatalf("status = %q", res.Status)
	}

	if gotAuth != "Bearer token-ของสมชาย" {
		t.Fatalf("service ปลายทางได้ Authorization = %q ซึ่งแปลว่า user ไม่ได้ตามไป", gotAuth)
	}
}

// CancelOrderByCode ตอบ 500 แบบ {status, message, fail_order} ไม่มี key "error"
// เหตุผลรายใบต้องไปถึงผู้ใช้ ไม่ใช่ body ดิบทั้งก้อน
func TestCancelOrderSurfacesFailOrderReason(t *testing.T) {
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status":"error","message":"cannot cancel 1 order(s)","fail_order":[{"order_code":"CO2610-00001","reason":"cannot cancel: goods have already left the warehouse, use Close instead"}]}`))
	}))
	defer downstream.Close()

	original := config.CANCEL_ORDER_ENDPOINT
	config.CANCEL_ORDER_ENDPOINT = downstream.URL
	defer func() { config.CANCEL_ORDER_ENDPOINT = original }()

	_, err := CancelOrder(context.Background(), CancelOrderRequest{DocumentRef: []string{"DBS2610-0001"}})
	if err == nil {
		t.Fatal("want error")
	}
	want := "cannot cancel 1 order(s): CO2610-00001 cannot cancel: goods have already left the warehouse, use Close instead"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}
