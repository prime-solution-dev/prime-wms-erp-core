package prePurchaseService

import (
	"encoding/json"
	"testing"
	"time"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

// เอกสาร PDF ใบสั่งซื้อ Big lot อ่านชื่อกลุ่มสินค้าจาก product_group_name
// ถ้า mapper ไม่ส่งค่านี้ ช่อง "รายการ" ในเอกสารจะว่าง
func TestMapPrePurchaseItemsModelToBigLotItemsResponse_ProductGroupName(t *testing.T) {
	items := []models.PrePurchaseItem{
		{
			ID:            uuid.New(),
			PrePurchaseID: uuid.New(),
			PreItem:       "PB202609-0012-001",
			HierarchyType: "เหล็กแบนตัด",
			HierarchyCode: "G1-001",
			CreateDtm:     time.Now(),
			UpdateDtm:     time.Now(),
		},
	}

	got, _ := MapPrePurchaseItemsModelToBigLotItemsResponse(items)

	if len(got) != 1 {
		t.Fatalf("got %d items, want 1", len(got))
	}
	if got[0].ProductGroupName != "เหล็กแบนตัด" {
		t.Errorf("ProductGroupName = %q, want %q", got[0].ProductGroupName, "เหล็กแบนตัด")
	}
	if got[0].ProductGroupCode != "G1-001" {
		t.Errorf("ProductGroupCode = %q, want %q", got[0].ProductGroupCode, "G1-001")
	}
	if got[0].ProductGroupType != "เหล็กแบนตัด" {
		t.Errorf("ProductGroupType = %q, want %q", got[0].ProductGroupType, "เหล็กแบนตัด")
	}
}

// ชื่อกลุ่มว่างต้องไม่ถูกแทนด้วยค่าอื่น — คงพฤติกรรมตรงไปตรงมาไว้
func TestMapPrePurchaseItemsModelToBigLotItemsResponse_BlankGroupName(t *testing.T) {
	items := []models.PrePurchaseItem{{
		ID:            uuid.New(),
		PrePurchaseID: uuid.New(),
		HierarchyCode: "G1-002",
		CreateDtm:     time.Now(),
		UpdateDtm:     time.Now(),
	}}

	got, _ := MapPrePurchaseItemsModelToBigLotItemsResponse(items)

	if got[0].ProductGroupName != "" {
		t.Errorf("ProductGroupName = %q, want empty", got[0].ProductGroupName)
	}
}

// ยอดรวมที่ mapper คืนมาต้องเป็นผลรวม TotalCost ของทุกรายการ
func TestMapPrePurchaseItemsModelToBigLotItemsResponse_SumTotalCost(t *testing.T) {
	items := []models.PrePurchaseItem{
		{ID: uuid.New(), PrePurchaseID: uuid.New(), TotalCost: 1500.50, CreateDtm: time.Now(), UpdateDtm: time.Now()},
		{ID: uuid.New(), PrePurchaseID: uuid.New(), TotalCost: 2499.50, CreateDtm: time.Now(), UpdateDtm: time.Now()},
	}

	got, sum := MapPrePurchaseItemsModelToBigLotItemsResponse(items)

	if len(got) != 2 {
		t.Fatalf("got %d items, want 2", len(got))
	}
	if sum != 4000 {
		t.Errorf("sum = %v, want 4000", sum)
	}
}

// ไม่มีรายการต้องคืน slice ว่าง ไม่ใช่ nil และผลรวมเป็น 0
func TestMapPrePurchaseItemsModelToBigLotItemsResponse_NoItems(t *testing.T) {
	got, sum := MapPrePurchaseItemsModelToBigLotItemsResponse(nil)

	if got == nil {
		t.Error("got nil slice, want empty slice")
	}
	if len(got) != 0 {
		t.Errorf("got %d items, want 0", len(got))
	}
	if sum != 0 {
		t.Errorf("sum = %v, want 0", sum)
	}
}

func TestBigLotNoteRequestAndResponse(t *testing.T) {
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
			var req models.CreatePOBigLotRequest
			if err := json.Unmarshal([]byte(tt.payload), &req); err != nil {
				t.Fatal(err)
			}
			purchase := MapBigLotRequestToPrePurchaseModel(req)
			var updateReq models.UpdatePOBigLotRequest
			if err := json.Unmarshal([]byte(tt.payload), &updateReq); err != nil {
				t.Fatal(err)
			}
			update := MapUpdatePOBigLotRequestToPrePurchase(updateReq)
			if (update.Note == nil) != tt.wantNil {
				t.Fatalf("update Note nil = %v, want %v", update.Note == nil, tt.wantNil)
			}
			if update.Note != nil && *update.Note != tt.want {
				t.Fatalf("update Note = %q, want %q", *update.Note, tt.want)
			}
			if update.Remark != "original remark" {
				t.Fatal("update note mapping changed remark")
			}
			if (purchase.Note == nil) != tt.wantNil {
				t.Fatalf("Note nil = %v, want %v", purchase.Note == nil, tt.wantNil)
			}
			if purchase.Note != nil && *purchase.Note != tt.want {
				t.Fatalf("Note = %q, want %q", *purchase.Note, tt.want)
			}
			response := MapPrePurchasesModelToBigLotsResponse(purchase)
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
