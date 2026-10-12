package purchaseService

import (
	"encoding/json"
	"strings"
	"testing"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestPurchaseNoteRequestAndResponse(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		want    string
		wantNil bool
	}{
		{"text", `{"note":"  หมายเหตุ\nบรรทัดสอง  ","remark":"original remark"}`, "  หมายเหตุ\nบรรทัดสอง  ", false},
		{"empty", `{"note":"","remark":"original remark"}`, "", false},
		{"omitted", `{"remark":"original remark"}`, "", true},
		{"null", `{"note":null,"remark":"original remark"}`, "", true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var req models.PurchaseFormRequest
			if err := json.Unmarshal([]byte(tt.payload), &req); err != nil {
				t.Fatal(err)
			}
			purchase := MapPurchaseFormRequestToPurchaseModel(req)
			if (purchase.Note == nil) != tt.wantNil {
				t.Fatalf("Note nil = %v, want %v", purchase.Note == nil, tt.wantNil)
			}
			if purchase.Note != nil && *purchase.Note != tt.want {
				t.Fatalf("Note = %q, want %q", *purchase.Note, tt.want)
			}
			response := MapPurchaseModelToPurchaseResponse(purchase)
			if response.Note != tt.want {
				t.Fatalf("response Note = %q, want %q", response.Note, tt.want)
			}
			if purchase.Remark != "original remark" || response.Remark != "original remark" {
				t.Fatal("note mapping changed remark")
			}
			payload, err := json.Marshal(response)
			if err != nil {
				t.Fatal(err)
			}
			var wire map[string]interface{}
			if err := json.Unmarshal(payload, &wire); err != nil {
				t.Fatal(err)
			}
			if value, ok := wire["note"].(string); !ok || value != tt.want {
				t.Fatalf("JSON note = %#v, want string %q", wire["note"], tt.want)
			}
		})
	}
}

func TestPONotePersistenceSQL(t *testing.T) {
	gormx, err := gorm.Open(postgres.New(postgres.Config{
		DSN: "host=localhost user=test dbname=test sslmode=disable",
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}

	for _, modelType := range []struct {
		name     string
		newModel func(*string) interface{}
	}{
		{"PO", func(note *string) interface{} { return &models.Purchase{ID: uuid.New(), Note: note} }},
		{"BigLot", func(note *string) interface{} { return &models.PrePurchase{ID: uuid.New(), Note: note} }},
	} {
		for _, tt := range []struct {
			name string
			note *string
			want string
		}{
			{name: "omitted"},
			{name: "empty", note: noteSQLString("")},
			{name: "text", note: noteSQLString("  ข้อความ\nสองบรรทัด  "), want: "  ข้อความ\nสองบรรทัด  "},
		} {
			t.Run(modelType.name+"/"+tt.name, func(t *testing.T) {
				created := gormx.Create(modelType.newModel(tt.note))
				if created.Error != nil {
					t.Fatal(created.Error)
				}
				values := created.Statement.Clauses["VALUES"].Expression.(clause.Values)
				found := false
				for i, column := range values.Columns {
					if column.Name != "note" {
						continue
					}
					found = true
					value := values.Values[0][i]
					if pointer, ok := value.(*string); ok {
						value = *pointer
					}
					if value != tt.want {
						t.Fatalf("INSERT note = %#v, want %q", value, tt.want)
					}
				}
				if !found {
					t.Fatal("INSERT is missing note")
				}

				model := modelType.newModel(tt.note)
				updated := gormx.Model(model).Updates(model)
				if updated.Error != nil {
					t.Fatal(updated.Error)
				}
				writesNote := strings.Contains(updated.Statement.SQL.String(), `"note"=`)
				if writesNote != (tt.note != nil) {
					t.Fatalf("unexpected note update: %s", updated.Statement.SQL.String())
				}
				if writesNote {
					value, ok := updated.Statement.Vars[0].(*string)
					if !ok || value == nil || *value != tt.want {
						t.Fatalf("UPDATE note = %#v, want %q", updated.Statement.Vars[0], tt.want)
					}
				}
			})
		}
	}
}

func noteSQLString(value string) *string { return &value }
