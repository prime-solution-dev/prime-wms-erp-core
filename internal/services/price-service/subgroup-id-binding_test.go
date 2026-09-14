package priceService

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prime-erp-core/internal/models"

	"github.com/gin-gonic/gin"
)

// Some price_list_sub_group rows carry deterministic UUIDv5 ids, so a uuid4-only
// binding rejected the whole batch and the detail pages never loaded their
// calculated prices. Any UUID version must bind.
func TestSubGroupIDsBindAnyUUIDVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := map[string]string{
		"v4": "430f49e2-8840-44bc-9a5c-6926763ecc4b",
		"v5": "2ef73090-2583-586a-b3c7-8c0dd91e3d4a",
	}

	for name, id := range cases {
		t.Run(name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]any{"subgroup_ids": []string{id}})
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
			ctx.Request.Header.Set("Content-Type", "application/json")

			var req models.UpdateLatestPriceListSubGroupRequest
			if err := ctx.ShouldBindJSON(&req); err != nil {
				t.Fatalf("binding %s uuid failed: %v", name, err)
			}
			if len(req.SubGroupIDs) != 1 || req.SubGroupIDs[0] != id {
				t.Fatalf("got %v, want [%s]", req.SubGroupIDs, id)
			}
		})
	}

	t.Run("rejects garbage", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"subgroup_ids": []string{"not-a-uuid"}})
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		ctx.Request.Header.Set("Content-Type", "application/json")

		var req models.UpdateLatestPriceListSubGroupRequest
		if err := ctx.ShouldBindJSON(&req); err == nil {
			t.Fatal("expected a binding error for a non-UUID id")
		}
	})
}
