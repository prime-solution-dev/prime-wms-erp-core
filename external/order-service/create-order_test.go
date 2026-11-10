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

func TestCreateOrderForwardsCallerToken(t *testing.T) {
	gotAuth := ""

	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}))
	defer downstream.Close()

	original := config.CREATE_ORDER_ENDPOINT
	config.CREATE_ORDER_ENDPOINT = downstream.URL
	defer func() { config.CREATE_ORDER_ENDPOINT = original }()

	ctx := requestcontext.WithToken(context.Background(), "Bearer token-test")

	if _, err := CreateOrder(ctx, CreateOrderRequest{}); err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	if gotAuth != "Bearer token-test" {
		t.Fatalf("downstream got Authorization = %q", gotAuth)
	}
}
