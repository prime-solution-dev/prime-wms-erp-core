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

func TestGetCustomerForwardsCallerToken(t *testing.T) {
	gotAuth := ""

	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
	}))
	defer downstream.Close()

	original := config.GET_CUSTOMER_MASTER_ENDPOINT
	config.GET_CUSTOMER_MASTER_ENDPOINT = downstream.URL
	defer func() { config.GET_CUSTOMER_MASTER_ENDPOINT = original }()

	ctx := requestcontext.WithToken(context.Background(), "Bearer token-test")

	if _, err := GetCustomer(ctx, GetCustomerRequest{}); err != nil {
		t.Fatalf("GetCustomer: %q", err)
	}

	if gotAuth != "Bearer token-test" {
		t.Fatalf("downstream got Authorization = %q", gotAuth)
	}
}
