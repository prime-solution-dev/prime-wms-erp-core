package priceService

import (
	"testing"

	"prime-erp-core/internal/models"
)

// คอลัมน์ Avg. kg stock ต้องอ่าน AvgProduct ซึ่งเป็นค่าเฉลี่ยระดับ site
// ไม่ใช่ AvgWeight ซึ่งเป็นค่าระดับ batch
//
// ค่าทั้งสองตั้งให้ต่างกันชัดเพื่อให้ test แยกแยะได้ว่าอ่าน field ไหน
func TestApplyInventoryFieldsToRowUsesAvgProduct(t *testing.T) {
	row := map[string]interface{}{}
	sg := SubGroup{
		WeightSpec: 12.5,
		InventoryWeight: []models.InventoryWeightResponse{
			{AvgWeight: 32.0, AvgProduct: 2.9969, TotalQty: 10436, SumQty: 10436},
		},
	}

	applyInventoryFieldsToRow(row, sg)

	if row["avg_weight"] != 2.9969 {
		t.Errorf("avg_weight = %v ต้องเป็น 2.9969 (AvgProduct ระดับ site) ไม่ใช่ 32 (AvgWeight ระดับ batch)", row["avg_weight"])
	}
	if row["total_weight"] != 12.5 {
		t.Errorf("total_weight = %v ต้องเป็น 12.5 (Weight-spec จาก product master)", row["total_weight"])
	}
}

// ไม่มีสต็อกต้องเติม avg_weight เป็น 0 ไม่ใช่ปล่อยให้ key หายไป
// เดิม early return ทำให้คอลัมน์นี้ไม่มี key เลย ต่างจากกริดที่คืน 0
// และต่างจาก Pricelist Detail Report ที่ใส่ string ว่าง
func TestApplyInventoryFieldsToRowNoStockFillsZero(t *testing.T) {
	row := map[string]interface{}{}
	sg := SubGroup{WeightSpec: 12.5}

	applyInventoryFieldsToRow(row, sg)

	avg, ok := row["avg_weight"]
	if !ok {
		t.Fatal("ไม่มี key avg_weight — ต้องมีและเป็น 0 เมื่อไม่มีสต็อก")
	}
	if avg != float64(0) {
		t.Errorf("avg_weight = %v (%T) ต้องเป็น float64(0)", avg, avg)
	}
	if row["total_weight"] != 12.5 {
		t.Errorf("total_weight = %v ต้องเป็น 12.5 แม้ไม่มีสต็อก", row["total_weight"])
	}
}
