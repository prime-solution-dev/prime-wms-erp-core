package purchaseService

import (
	"testing"

	"prime-erp-core/internal/models"
)

func ptr(s string) *string { return &s }

func TestNextPurchaseItemSeq(t *testing.T) {
	cases := []struct {
		name  string
		items []models.PurchaseItemFormRequest
		want  int
	}{
		{"empty", nil, 1},
		{"all new", []models.PurchaseItemFormRequest{{}, {}}, 1},
		{"contiguous existing", []models.PurchaseItemFormRequest{
			{PurchaseItem: ptr("1")}, {PurchaseItem: ptr("2")}, {PurchaseItem: ptr("3")},
		}, 4},
		{"gap keeps max", []models.PurchaseItemFormRequest{
			{PurchaseItem: ptr("1")}, {PurchaseItem: ptr("7")},
		}, 8},
		{"legacy non-numeric ignored", []models.PurchaseItemFormRequest{
			{PurchaseItem: ptr("PO202609-0011-1756789")}, {PurchaseItem: ptr("2")},
		}, 3},
		{"blank ignored", []models.PurchaseItemFormRequest{
			{PurchaseItem: ptr("  ")}, {PurchaseItem: ptr("5")},
		}, 6},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NextPurchaseItemSeq(c.items); got != c.want {
				t.Fatalf("NextPurchaseItemSeq() = %d, want %d", got, c.want)
			}
		})
	}
}

func TestMapPurchaseItemFormRequestToPurchaseItemModel_PurchaseItem(t *testing.T) {
	t.Run("generates the given seq when request has none", func(t *testing.T) {
		got := MapPurchaseItemFormRequestToPurchaseItemModel(models.PurchaseItemFormRequest{}, 3)
		if got.PurchaseItem != "3" {
			t.Fatalf("PurchaseItem = %q, want %q", got.PurchaseItem, "3")
		}
	})

	t.Run("keeps the purchase_item the request carries", func(t *testing.T) {
		got := MapPurchaseItemFormRequestToPurchaseItemModel(
			models.PurchaseItemFormRequest{PurchaseItem: ptr("2")}, 9)
		if got.PurchaseItem != "2" {
			t.Fatalf("PurchaseItem = %q, want %q", got.PurchaseItem, "2")
		}
	})

	t.Run("treats a blank purchase_item as absent", func(t *testing.T) {
		got := MapPurchaseItemFormRequestToPurchaseItemModel(
			models.PurchaseItemFormRequest{PurchaseItem: ptr("")}, 4)
		if got.PurchaseItem != "4" {
			t.Fatalf("PurchaseItem = %q, want %q", got.PurchaseItem, "4")
		}
	})
}

// mirrors the CreatePO loop: every line gets 1..N in payload order
func TestCreatePOItemNumberingIsContiguous(t *testing.T) {
	items := make([]models.PurchaseItemFormRequest, 5)
	for idx, item := range items {
		got := MapPurchaseItemFormRequestToPurchaseItemModel(item, idx+1)
		want := []string{"1", "2", "3", "4", "5"}[idx]
		if got.PurchaseItem != want {
			t.Fatalf("item %d: PurchaseItem = %q, want %q", idx, got.PurchaseItem, want)
		}
	}
}

// mirrors the UpdatePO loop: existing lines keep their number, new ones continue
func TestUpdatePOItemNumberingContinuesAfterExisting(t *testing.T) {
	items := []models.PurchaseItemFormRequest{
		{PurchaseItem: ptr("1")},
		{PurchaseItem: ptr("2")},
		{}, // newly added line
		{}, // newly added line
	}

	seq := NextPurchaseItemSeq(items)
	got := []string{}
	for _, item := range items {
		mapped := MapPurchaseItemFormRequestToPurchaseItemModel(item, seq)
		if item.PurchaseItem == nil || *item.PurchaseItem == "" {
			seq++
		}
		got = append(got, mapped.PurchaseItem)
	}

	want := []string{"1", "2", "3", "4"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
