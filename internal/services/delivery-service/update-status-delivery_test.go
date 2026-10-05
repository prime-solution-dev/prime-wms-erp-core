package deliveryService

import (
	"encoding/json"
	"testing"

	"prime-erp-core/internal/models"
)

func deliveryMapOf(pairs map[string]string) map[string]models.Delivery {
	deliveryOf := map[string]models.Delivery{}
	for deliveryCode, status := range pairs {
		deliveryOf[deliveryCode] = models.Delivery{
			DeliveryCode: deliveryCode,
			Status:       status,
			DocumentRef:  "SO-" + deliveryCode,
		}
	}

	return deliveryOf
}

// ใบที่อยู่ COMPLETED อยู่แล้วต้องไม่หายไปจากรายชื่อ เพราะเส้นปิด SO ต้องนับมันด้วย
//
// outbound ใบเดียวยืนยันได้หลาย CO และแต่ละ CO ยิง hook ของตัวเอง hook ตัวแรกพลิก DBS
// เป็น COMPLETED แล้วรันปิด SO ตอนที่บรรทัดของ CO ตัวที่สองยังเปิดอยู่ พอ hook ตัวที่สอง
// มาถึง DBS ก็ COMPLETED ไปแล้ว ถ้าตัดออก จะไม่มีรอบไหนเลยที่ข้อมูลครบ
func TestPartitionDeliveriesByStatusKeepsDeliveriesAlreadyAtTarget(t *testing.T) {
	deliveryOf := deliveryMapOf(map[string]string{
		"DBS-NEW":  "PENDING",
		"DBS-DONE": "COMPLETED",
	})

	toUpdate, alreadyAtStatus, err := partitionDeliveriesByStatus(
		[]string{"DBS-NEW", "DBS-DONE"}, deliveryOf, "COMPLETED")
	if err != nil {
		t.Fatalf("partitionDeliveriesByStatus error = %v, want nil", err)
	}

	if len(toUpdate) != 1 || toUpdate[0] != "DBS-NEW" {
		t.Errorf("toUpdate = %v, want [DBS-NEW]", toUpdate)
	}

	if len(alreadyAtStatus) != 1 || alreadyAtStatus[0] != "DBS-DONE" {
		t.Errorf("alreadyAtStatus = %v, want [DBS-DONE] — ใบนี้ต้องยังถูกส่งไปให้เส้นปิด SO", alreadyAtStatus)
	}
}

// hook ตัวที่สองของ outbound เดียวกัน: ทุกใบ COMPLETED ไปแล้ว ไม่มีอะไรต้องอัปเดต
// แต่ยังต้องมีรายชื่อใบให้ปิด SO ไม่งั้นเส้นปิดจะถูกข้ามทั้งรอบ (ตาม early return เดิม)
func TestPartitionDeliveriesByStatusReturnsCandidatesWhenNothingToUpdate(t *testing.T) {
	deliveryOf := deliveryMapOf(map[string]string{
		"DBS-A": "COMPLETED",
		"DBS-B": "COMPLETED",
	})

	toUpdate, alreadyAtStatus, err := partitionDeliveriesByStatus(
		[]string{"DBS-A", "DBS-B"}, deliveryOf, "COMPLETED")
	if err != nil {
		t.Fatalf("partitionDeliveriesByStatus error = %v, want nil", err)
	}

	if len(toUpdate) != 0 {
		t.Errorf("toUpdate = %v, want ว่าง", toUpdate)
	}

	if len(alreadyAtStatus) != 2 {
		t.Fatalf("alreadyAtStatus = %v, want 2 ใบ", alreadyAtStatus)
	}
}

func TestPartitionDeliveriesByStatusRejectsCanceledDelivery(t *testing.T) {
	deliveryOf := deliveryMapOf(map[string]string{"DBS-X": "CANCELED"})

	if _, _, err := partitionDeliveriesByStatus([]string{"DBS-X"}, deliveryOf, "COMPLETED"); err == nil {
		t.Error("ใบที่ยกเลิกแล้วต้องคืน error ไม่ใช่ปล่อยผ่าน")
	}
}

func TestPartitionDeliveriesByStatusRejectsUnknownDelivery(t *testing.T) {
	if _, _, err := partitionDeliveriesByStatus([]string{"DBS-MISSING"}, map[string]models.Delivery{}, "COMPLETED"); err == nil {
		t.Error("ใบที่หาไม่เจอต้องคืน error")
	}
}

// ยกเลิก: ใบที่ CANCELED อยู่แล้วต้องเข้ากอง alreadyAtStatus ไม่ใช่ error
func TestPartitionDeliveriesByStatusTreatsCancelTargetAsIdempotent(t *testing.T) {
	deliveryOf := deliveryMapOf(map[string]string{"DBS-X": "CANCELED"})

	toUpdate, alreadyAtStatus, err := partitionDeliveriesByStatus([]string{"DBS-X"}, deliveryOf, "CANCELED")
	if err != nil {
		t.Fatalf("partitionDeliveriesByStatus error = %v, want nil", err)
	}

	if len(toUpdate) != 0 || len(alreadyAtStatus) != 1 {
		t.Errorf("toUpdate = %v, alreadyAtStatus = %v, want [] และ [DBS-X]", toUpdate, alreadyAtStatus)
	}
}

// hook ยกเลิกต้องได้ payload หน้าตา {"delivery_codes":"<เลขใบ>","status":"CANCELED","external_id":"<id>"} ทีละใบ
func TestBuildCancelHookRequestsShape(t *testing.T) {
	deliveryOf := map[string]models.Delivery{
		"DBS202610-0004": {DeliveryCode: "DBS202610-0004", ExternalID: "1"},
		"DBS202610-0005": {DeliveryCode: "DBS202610-0005", ExternalID: "2"},
	}

	hookReqs := buildCancelHookRequests(deliveryOf, []string{"DBS202610-0004"})

	if len(hookReqs) != 1 {
		t.Fatalf("จำนวน payload = %d, ต้องการ 1 (ส่งเฉพาะใบที่เพิ่งยกเลิก)", len(hookReqs))
	}

	got, err := json.Marshal(hookReqs[0])
	if err != nil {
		t.Fatal(err)
	}

	want := `{"delivery_codes":"DBS202610-0004","status":"CANCELED","external_id":"1"}`
	if string(got) != want {
		t.Fatalf("payload = %s, ต้องการ %s", got, want)
	}
}
