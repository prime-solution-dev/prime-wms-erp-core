package deliveryService

import (
	"testing"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func updateDoc(code string, externalID string, isDraft bool, qtys ...float64) DeliveryDocumentUpdate {
	doc := DeliveryDocumentUpdate{
		Delivery: models.Delivery{
			ID:           uuid.New(),
			DeliveryCode: code,
			ExternalID:   externalID,
		},
		IsDraft: isDraft,
	}

	for _, qty := range qtys {
		doc.Items = append(doc.Items, DeliveryItemDocumentUpdate{
			DeliveryItem: models.DeliveryItem{ID: uuid.New(), Qty: qty},
		})
	}

	return doc
}

// ปลายทางต้องได้ทั้งเลขใบจองและ external id ของใบที่ถูกแก้
func TestBuildUpdateHookRequestKeepsDeliveryCodeAndExternalID(t *testing.T) {
	hookReq := buildUpdateHookRequest([]DeliveryDocumentUpdate{
		updateDoc("DBS2609-0001", "TRC-001", false, 5),
	})

	if len(hookReq) != 1 {
		t.Fatalf("จำนวนใบที่ส่งเข้า hook = %d, ต้องการ 1", len(hookReq))
	}

	if hookReq[0].DeliveryCode != "DBS2609-0001" {
		t.Fatalf("delivery_code = %q", hookReq[0].DeliveryCode)
	}

	if hookReq[0].ExternalID != "TRC-001" {
		t.Fatalf("external_id = %q", hookReq[0].ExternalID)
	}
}

// ใบร่างยังไม่ผูกของจริง ปลายทางไม่ต้องรู้
func TestBuildUpdateHookRequestSkipsDraft(t *testing.T) {
	hookReq := buildUpdateHookRequest([]DeliveryDocumentUpdate{
		updateDoc("DBS2609-0001", "TRC-001", true, 5),
		updateDoc("DBS2609-0002", "TRC-002", false, 3),
	})

	if len(hookReq) != 1 || hookReq[0].DeliveryCode != "DBS2609-0002" {
		t.Fatalf("hookReq = %+v", hookReq)
	}
}

// TRCloud ไม่เอาบรรทัดที่จอง 0 ชิ้น และใบที่ไม่เหลือบรรทัดเลยต้องไม่ถูกส่งไป
func TestBuildUpdateHookRequestDropsZeroQtyItemsAndEmptyDocs(t *testing.T) {
	input := []DeliveryDocumentUpdate{
		updateDoc("DBS2609-0001", "TRC-001", false, 0, 2, 0),
		updateDoc("DBS2609-0002", "TRC-002", false, 0),
	}

	hookReq := buildUpdateHookRequest(input)

	if len(hookReq) != 1 {
		t.Fatalf("จำนวนใบ = %d, ต้องการ 1 (ใบที่เหลือแต่บรรทัด qty 0 ต้องถูกตัด)", len(hookReq))
	}

	if len(hookReq[0].Items) != 1 || hookReq[0].Items[0].DeliveryItem.Qty != 2 {
		t.Fatalf("items = %+v", hookReq[0].Items)
	}

	// ของที่ส่งเข้ามาต้องไม่ถูกแก้ ฝั่งที่เขียนลง DB ยังต้องได้บรรทัดครบเหมือนเดิม
	if len(input[0].Items) != 3 {
		t.Fatalf("payload ต้นฉบับถูกแก้: เหลือ %d บรรทัด", len(input[0].Items))
	}
}
