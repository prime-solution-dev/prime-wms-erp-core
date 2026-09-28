package interfaceService

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestHookInterfaceResponse(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		body        string
		want        interface{}
		errContains string
	}{
		{"id success", 200, `{"id":"external-1"}`, map[string]interface{}{"id": "external-1"}, ""},
		{"message success", 200, `{"HTTP":"200 Success","message":"updated"}`, "updated", ""},
		{"missing HTTP preserves message", 200, `{"message":"invoice rejected"}`, nil, "invoice rejected"},
		{"numeric HTTP preserves message", 200, `{"HTTP":500,"message":"invalid invoice"}`, nil, "invalid invoice"},
		{"error fallback", 200, `{"error":"unknown invoice"}`, nil, "unknown invoice"},
		{"HTTP failure overrides id", 500, `{"id":"external-1","message":"database unavailable"}`, nil, "500 Internal Server Error"},
		{"HTTP failure overrides success", 400, `{"HTTP":"200 Success","message":"bad request"}`, nil, "request failed"},
		{"upstream failure", 200, `{"HTTP":"500 Error","message":"rejected"}`, nil, "rejected"},
		{"upstream failure without message", 200, `{"HTTP":"500 Error"}`, nil, "500 Error"},
		{"invalid JSON", 502, "<html>Bad Gateway</html>", nil, "502 Bad Gateway"},
		{"empty response", 200, "", nil, "invalid JSON response"},
		{"null response", 200, "null", nil, "expected a JSON object"},
		{"array response", 200, "[]", nil, "expected a JSON object"},
		{"invalid id", 200, `{"id":null}`, nil, "missing or invalid response fields"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/interface/hook-interface" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			t.Setenv("base_url_document", server.URL)
			got, err := HookInterface(HookInterfaceRequest{RequestData: map[string]string{"id": "invoice-1"}})
			if tc.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("error = %v, want containing %q", err, tc.errContains)
				}
				if got != nil {
					t.Fatalf("unexpected result on failure: %v", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("result = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestHookInterfaceRejectsUnencodableRequest(t *testing.T) {
	_, err := HookInterface(HookInterfaceRequest{RequestData: make(chan int)})
	if err == nil || !strings.Contains(err.Error(), "encode request") {
		t.Fatalf("unexpected error: %v", err)
	}
}
