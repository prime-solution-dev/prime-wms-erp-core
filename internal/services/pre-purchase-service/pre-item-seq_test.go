package prePurchaseService

import (
	"testing"
	"time"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func preItemPtr(s string) *string { return &s }

func TestBuildPreItemCode(t *testing.T) {
	cases := []struct {
		code string
		seq  int
		want string
	}{
		{"PB202609-0012", 1, "PB202609-0012-001"},
		{"PB202609-0012", 12, "PB202609-0012-012"},
		{"PB202609-0012", 999, "PB202609-0012-999"},
		{"PB202609-0012", 1000, "PB202609-0012-1000"},
	}

	for _, c := range cases {
		if got := BuildPreItemCode(c.code, c.seq); got != c.want {
			t.Fatalf("BuildPreItemCode(%q, %d) = %q, want %q", c.code, c.seq, got, c.want)
		}
	}
}

func TestNextPreItemSeq(t *testing.T) {
	cases := []struct {
		name  string
		items []models.UpdatePOBigLotItemRequest
		want  int
	}{
		{"empty", nil, 1},
		{"all new", []models.UpdatePOBigLotItemRequest{{}, {}}, 1},
		{"blank pre_item ignored", []models.UpdatePOBigLotItemRequest{
			{PreItem: preItemPtr("")}, {PreItem: preItemPtr("   ")},
		}, 1},
		{"contiguous existing", []models.UpdatePOBigLotItemRequest{
			{PreItem: preItemPtr("PB202609-0012-001")},
			{PreItem: preItemPtr("PB202609-0012-002")},
			{PreItem: preItemPtr("PB202609-0012-003")},
		}, 4},
		{"gap keeps max", []models.UpdatePOBigLotItemRequest{
			{PreItem: preItemPtr("PB202609-0012-001")},
			{PreItem: preItemPtr("PB202609-0012-007")},
		}, 8},
		{"legacy timestamp suffix ignored", []models.UpdatePOBigLotItemRequest{
			{PreItem: preItemPtr("PB202609-0012-022646")},
			{PreItem: preItemPtr("PB202609-0012-022646")},
		}, 1},
		{"legacy mixed with new numbering", []models.UpdatePOBigLotItemRequest{
			{PreItem: preItemPtr("PB202609-0012-022646")},
			{PreItem: preItemPtr("PB202609-0012-002")},
		}, 3},
		{"no dash ignored", []models.UpdatePOBigLotItemRequest{
			{PreItem: preItemPtr("PB2026090012001")},
		}, 1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NextPreItemSeq(c.items); got != c.want {
				t.Fatalf("NextPreItemSeq() = %d, want %d", got, c.want)
			}
		})
	}
}

func mapUpdateItem(item models.UpdatePOBigLotItemRequest, seq int) models.PrePurchaseItem {
	id := uuid.New()
	item.ID = &id
	return MapUpdatePOBigLotRequestToPrePurchaseItem(item, "system", time.Now().UTC(), "PB202609-0012", seq)
}

func TestMapUpdatePOBigLotRequestToPrePurchaseItem_PreItem(t *testing.T) {
	t.Run("generates the given seq when request has none", func(t *testing.T) {
		if got := mapUpdateItem(models.UpdatePOBigLotItemRequest{}, 3); got.PreItem != "PB202609-0012-003" {
			t.Fatalf("PreItem = %q, want %q", got.PreItem, "PB202609-0012-003")
		}
	})

	t.Run("keeps the pre_item the request carries", func(t *testing.T) {
		item := models.UpdatePOBigLotItemRequest{PreItem: preItemPtr("PB202609-0012-002")}
		if got := mapUpdateItem(item, 9); got.PreItem != "PB202609-0012-002" {
			t.Fatalf("PreItem = %q, want %q", got.PreItem, "PB202609-0012-002")
		}
	})

	t.Run("treats a blank pre_item as absent", func(t *testing.T) {
		item := models.UpdatePOBigLotItemRequest{PreItem: preItemPtr("")}
		if got := mapUpdateItem(item, 4); got.PreItem != "PB202609-0012-004" {
			t.Fatalf("PreItem = %q, want %q", got.PreItem, "PB202609-0012-004")
		}
	})
}

// mirrors the CreatePOBigLot loop: every line gets 1..N in payload order and no two are equal
func TestCreatePreItemNumberingIsUniqueAndContiguous(t *testing.T) {
	seen := map[string]bool{}
	for idx := range make([]struct{}, 5) {
		got := BuildPreItemCode("PB202609-0012", idx+1)
		want := []string{
			"PB202609-0012-001", "PB202609-0012-002", "PB202609-0012-003",
			"PB202609-0012-004", "PB202609-0012-005",
		}[idx]
		if got != want {
			t.Fatalf("item %d: pre_item = %q, want %q", idx, got, want)
		}
		if seen[got] {
			t.Fatalf("duplicate pre_item %q", got)
		}
		seen[got] = true
	}
}

// mirrors the UpdatePOBigLot loop: existing lines keep their number, new ones continue after the max
func TestUpdatePreItemNumberingContinuesAfterExisting(t *testing.T) {
	items := []models.UpdatePOBigLotItemRequest{
		{PreItem: preItemPtr("PB202609-0012-001")},
		{}, // แทรก item ใหม่กลางรายการ
		{PreItem: preItemPtr("PB202609-0012-003")},
		{}, // item ใหม่ท้ายรายการ
	}

	seq := NextPreItemSeq(items)
	got := []string{}
	seen := map[string]bool{}
	for _, item := range items {
		mapped := mapUpdateItem(item, seq)
		if item.PreItem == nil || *item.PreItem == "" {
			seq++
		}
		if seen[mapped.PreItem] {
			t.Fatalf("duplicate pre_item %q in %v", mapped.PreItem, got)
		}
		seen[mapped.PreItem] = true
		got = append(got, mapped.PreItem)
	}

	want := []string{
		"PB202609-0012-001", "PB202609-0012-004",
		"PB202609-0012-003", "PB202609-0012-005",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
