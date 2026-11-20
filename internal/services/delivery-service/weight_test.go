package deliveryService

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"prime-erp-core/config"
	"testing"
)

func TestRoundWeight(t *testing.T) {
	cases := []struct {
		v     float64
		digit int
		want  float64
	}{
		{133.105, 2, 133.11},
		{60.154365, 3, 60.154},
		{60.1545, 3, 60.155},
		{60.154365, 2, 60.15},
		{-133.105, 2, -133.11},
		{0, 3, 0},
	}
	for _, c := range cases {
		if got := roundWeight(c.v, c.digit); got != c.want {
			t.Errorf("roundWeight(%v, %d) = %v, want %v", c.v, c.digit, got, c.want)
		}
	}
}

func TestGetWeightDigits(t *testing.T) {
	calls := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			CompanyCode []string `json:"company_code"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		code := req.CompanyCode[0]
		calls[code]++
		switch code {
		case "TMI":
			w.Write([]byte(`{"company":[{"company_code":"TMI","company_config":{"weight_digit":3}}]}`))
		case "NO_CONFIG":
			w.Write([]byte(`{"company":[{"company_code":"NO_CONFIG","company_config":{"weight_digit":null}}]}`))
		case "DOWN":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.Write([]byte(`{"company":[]}`))
		}
	}))
	defer server.Close()

	original := config.GET_COMPANY_ENDPOINT
	config.GET_COMPANY_ENDPOINT = server.URL
	defer func() { config.GET_COMPANY_ENDPOINT = original }()

	digits, err := getWeightDigits(context.Background(), []string{"TMI", "TMI", "NO_CONFIG", "UNKNOWN", ""})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"TMI": 3, "NO_CONFIG": 2, "UNKNOWN": 2, "": 2}
	for code, digit := range want {
		if digits[code] != digit {
			t.Errorf("digit of %q = %d, want %d", code, digits[code], digit)
		}
	}
	if calls["TMI"] != 1 {
		t.Errorf("TMI called %d times, want 1", calls["TMI"])
	}
	if calls[""] != 0 {
		t.Errorf("empty company code must not call the API")
	}

	if _, err := getWeightDigits(context.Background(), []string{"DOWN"}); err == nil {
		t.Error("want error when the API fails")
	}
}
