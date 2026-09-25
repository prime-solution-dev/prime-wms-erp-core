package priceService

import (
	"testing"

	externalProductService "prime-erp-core/external/product-service"
	"prime-erp-core/internal/models"
)

func TestProductKey_SortsBySeqAndDropsInactive(t *testing.T) {
	p := externalProductService.GetProductsComponent{
		ProductGroup: []models.ProductGroup{
			{GroupCode: "PG02", GroupValue: "PG02_19", Seq: 2, ActiveFlg: true},
			{GroupCode: "PG09", GroupValue: "PG09_1", Seq: 3, ActiveFlg: false},
			{GroupCode: "PG01", GroupValue: "PG01_3", Seq: 1, ActiveFlg: true},
		},
	}
	got := productKey(p)
	want := "PG01|PG02#PG01_3|PG02_19"
	if got != want {
		t.Fatalf("productKey = %q, want %q", got, want)
	}
}

func TestSubGroupKey_SortsBySeqAndSkipsEmptyCode(t *testing.T) {
	sg := SubGroup{GroupKeys: []GroupKey{
		{Code: "PG02", Value: "PG02_19", Seq: 2},
		{Code: "", Value: "junk", Seq: 0},
		{Code: "PG01", Value: "PG01_3", Seq: 1},
	}}
	got := subGroupKey(sg)
	want := "PG01|PG02#PG01_3|PG02_19"
	if got != want {
		t.Fatalf("subGroupKey = %q, want %q", got, want)
	}
}

func TestProductKey_EmptyWhenNoActiveGroups(t *testing.T) {
	if got := productKey(externalProductService.GetProductsComponent{}); got != "" {
		t.Fatalf("want empty key, got %q", got)
	}
}
