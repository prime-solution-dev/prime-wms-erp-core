package invoiceService

import (
	"github.com/google/uuid"
	"prime-erp-core/internal/models"
	"testing"
)

func TestAssignInvoiceItemNumbers(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	old := []models.InvoiceItem{{ID: a, InvoiceItem: "1"}, {ID: b, InvoiceItem: "2"}, {ID: c, InvoiceItem: "10"}}
	tests := []struct {
		name  string
		input []models.InvoiceItem
		want  []string
		fail  bool
	}{
		{"preserve stored numbers", []models.InvoiceItem{{ID: a, InvoiceItem: "999"}, {ID: b}, {ID: c}}, []string{"1", "2", "10"}, false},
		{"remove middle and add", []models.InvoiceItem{{ID: a}, {ID: c}, {}}, []string{"1", "10", "11"}, false},
		{"remove maximum and add twice", []models.InvoiceItem{{ID: a}, {}, {}}, []string{"1", "11", "12"}, false},
		{"remove all", nil, nil, false},
		{"foreign ID", []models.InvoiceItem{{ID: uuid.New()}}, nil, true},
		{"duplicate ID", []models.InvoiceItem{{ID: a}, {ID: a}}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := assignInvoiceItemNumbers(old, tt.input)
			if tt.fail {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatal("unexpected item count")
			}
			for i, item := range got {
				if item.InvoiceItem != tt.want[i] {
					t.Fatalf("item %d: got %s want %s", i, item.InvoiceItem, tt.want[i])
				}
			}
		})
	}
	t.Run("empty invoice starts at one", func(t *testing.T) {
		got, err := assignInvoiceItemNumbers(nil, []models.InvoiceItem{{}})
		if err != nil {
			t.Fatal(err)
		}
		if got[0].InvoiceItem != "1" {
			t.Fatal(got)
		}
	})
}
