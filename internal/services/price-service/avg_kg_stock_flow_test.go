package priceService

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prime-erp-core/config"
	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

// เทสชุดนี้พิสูจน์บั๊กราคา 3 ตัวที่แก้ไปด้วยกัน
//
//  1. avgKgStock ต้องอ่าน AvgProduct (ค่าเฉลี่ยระดับ site รวมทุก batch) ไม่ใช่ AvgWeight
//     ซึ่งเป็นค่าของ batch ที่หยิบมาจาก index 0
//  2. ตัวแปร pcs / kg ในสูตร price_calc หมายถึง "ราคาต่อชิ้น" และ "ราคาต่อกิโล"
//     ไม่ใช่จำนวนสต็อก (TotalQty / TotalWeight) — ต้องอ่านจากค่า running
//     totalNetPriceUnit / totalNetPriceWeight
//  3. คู่สูตร kg → pcs เป็น pipeline จึงต้องประเมินตามลำดับ dependency
//     ไม่ใช่ลำดับ create_dtm ที่ repository คืนมา
//
// ไม่พึ่ง DB: repository ถูก stub ผ่าน package-level function var และ inventory service
// ถูกชี้ไปที่ httptest server ผ่าน config.GET_INVENTORY_BY_KEY_ENDPOINT
//
// แก้ package var config.GET_INVENTORY_BY_KEY_ENDPOINT ตรง ๆ จึงไม่ parallel-safe
// ห้ามใส่ t.Parallel()

// pointInventoryEndpointAtFake ตั้ง fake inventory service ที่คืน body ที่กำหนด
// และชี้ endpoint ไปที่นั่นตลอดอายุเทส
func pointInventoryEndpointAtFake(t *testing.T, body string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(body)); err != nil {
			t.Errorf("write fake inventory response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	original := config.GET_INVENTORY_BY_KEY_ENDPOINT
	config.GET_INVENTORY_BY_KEY_ENDPOINT = srv.URL
	t.Cleanup(func() { config.GET_INVENTORY_BY_KEY_ENDPOINT = original })
}

// subGroupForAvgKgStockTest สร้าง subgroup ที่ราคาต่อกิโลคำนวณได้ 50+5 = 55
// โดย TotalNetPriceWeight ที่ค้างใน DB เป็น 45 ซึ่งตั้งใจให้ต่างจาก 55
// เพื่อแยกแยะได้ว่าสูตร pcs อ่านค่าใหม่หรือค่าเก่า
func subGroupForAvgKgStockTest(id uuid.UUID) models.PriceListSubGroup {
	groupID := uuid.New()
	return models.PriceListSubGroup{
		ID:                  id,
		PriceListGroupID:    groupID,
		SubGroupCode:        "SUB_AVG",
		SubgroupKey:         "SUB_AVG",
		IsTrading:           true,
		PriceUnit:           100.0,
		PriceWeight:         50.0,
		ExtraPriceUnit:      10.0,
		ExtraPriceWeight:    5.0,
		TotalNetPriceUnit:   90.0,
		TotalNetPriceWeight: 45.0,
		PriceListGroup: models.PriceListGroup{
			ID:          groupID,
			PriceUnit:   100.0,
			PriceWeight: 50.0,
			CompanyCode: "C01",
			SiteCode:    "S01",
		},
		PriceListSubGroupKeys: []models.PriceListSubGroupKey{
			{ID: uuid.New(), SubGroupID: id, Code: "CODE", Value: "VALUE", Seq: 1},
		},
	}
}

// kgThenPcsFormulas คืนคู่สูตร pipeline: kg = base_price + extra แล้ว pcs = kg * avg_kg_stock
func kgThenPcsFormulas() (kg models.PriceListSubGroupFormulasMap, pcs models.PriceListSubGroupFormulasMap) {
	kg = models.PriceListSubGroupFormulasMap{
		ID:                    uuid.New(),
		PriceListSubGroupCode: "SUB_AVG",
		PriceListFormulasCode: "F_KG",
		PriceListFormulas: models.PriceListFormulas{
			ID:          uuid.New(),
			FormulaCode: "F_KG",
			Name:        "kg = [Base Price] + [Extra]",
			Uom:         "kg",
			FormulaType: "price_calc",
			Expression:  "base_price + extra",
			Params:      json.RawMessage(`{"required":["base_price","extra"]}`),
			Rounding:    2,
		},
	}
	pcs = models.PriceListSubGroupFormulasMap{
		ID:                    uuid.New(),
		PriceListSubGroupCode: "SUB_AVG",
		PriceListFormulasCode: "F_PCS",
		PriceListFormulas: models.PriceListFormulas{
			ID:          uuid.New(),
			FormulaCode: "F_PCS",
			Name:        "Pcs = [kg] x [Avg. kg stock]",
			Uom:         "pcs",
			FormulaType: "price_calc",
			Expression:  "kg * avg_kg_stock",
			Params:      json.RawMessage(`{"required":["kg","avg_kg_stock"]}`),
			Rounding:    2,
		},
	}
	return kg, pcs
}

// runUpdateLatestForAvgKgStock stub repository แล้วรัน flow จริง คืนค่าที่จะถูกบันทึกลง DB
func runUpdateLatestForAvgKgStock(
	t *testing.T,
	subGroupID uuid.UUID,
	formulas []models.PriceListSubGroupFormulasMap,
) models.UpdatePriceListSubGroupItem {
	t.Helper()

	subGroup := subGroupForAvgKgStockTest(subGroupID)

	originalGetByIDs := getPriceListSubGroupsByIDsFunc
	getPriceListSubGroupsByIDsFunc = func([]uuid.UUID) ([]models.PriceListSubGroup, error) {
		return []models.PriceListSubGroup{subGroup}, nil
	}
	defer func() { getPriceListSubGroupsByIDsFunc = originalGetByIDs }()

	originalGetFormulas := getPriceListSubGroupFormulasMapBySubGroupCodesFunc
	getPriceListSubGroupFormulasMapBySubGroupCodesFunc = func([]string) (map[string][]models.PriceListSubGroupFormulasMap, error) {
		return map[string][]models.PriceListSubGroupFormulasMap{
			"SUB_AVG": formulas,
		}, nil
	}
	defer func() { getPriceListSubGroupFormulasMapBySubGroupCodesFunc = originalGetFormulas }()

	var saved models.UpdatePriceListSubGroupRequest
	originalUpdate := updateLatestSubGroupFunc
	updateLatestSubGroupFunc = func(req models.UpdatePriceListSubGroupRequest) error {
		saved = req
		return nil
	}
	defer func() { updateLatestSubGroupFunc = originalUpdate }()

	if _, err := RunUpdateLatestPriceListSubGroup(models.UpdateLatestPriceListSubGroupRequest{
		SubGroupIDs: []string{subGroupID.String()},
	}); err != nil {
		t.Fatalf("RunUpdateLatestPriceListSubGroup: %v", err)
	}

	if len(saved.Changes) != 1 {
		t.Fatalf("expected 1 change saved, got %d", len(saved.Changes))
	}
	return saved.Changes[0]
}

// assertWeightAndUnit ตรวจค่าที่บันทึก พร้อมบอกความหมายของค่าที่ผิดแต่ละแบบ
func assertWeightAndUnit(t *testing.T, change models.UpdatePriceListSubGroupItem, wantWeight, wantUnit float64) {
	t.Helper()

	if change.TotalNetPriceWeight == nil || change.TotalNetPriceUnit == nil {
		t.Fatal("TotalNetPriceWeight / TotalNetPriceUnit ต้องไม่เป็น nil")
	}
	if got := *change.TotalNetPriceWeight; got != wantWeight {
		t.Errorf("total_net_price_weight = %v, want %v (= PriceWeight 50 + ExtraPriceWeight 5)", got, wantWeight)
	}
	if got := *change.TotalNetPriceUnit; got != wantUnit {
		t.Errorf(`total_net_price_unit = %v, want %v
  %v  = ถูกต้อง: สูตร pcs อ่านค่า running total_net_price_weight ที่สูตร kg เพิ่งคำนวณ
  45 = ผิด: อ่าน total_net_price_weight ค่าเก่าใน DB (ลำดับประเมินสูตรผิด - บั๊ก C)
  0  = ผิด: อ่าน inventoryWeight[0].TotalWeight ซึ่งเป็นน้ำหนักสต็อก (บั๊ก B เดิม)`,
			got, wantUnit, wantUnit)
	}
}

// TestUpdateLatestUsesRunningWeightPriceInPcsFormula — สูตร pcs ต้องอ่านราคาต่อกิโลที่
// สูตร kg เพิ่งคำนวณ (55) ไม่ใช่ค่าเก่าใน DB (45) และไม่ใช่น้ำหนักสต็อก (0)
func TestUpdateLatestUsesRunningWeightPriceInPcsFormula(t *testing.T) {
	pointInventoryEndpointAtFake(t, `[]`)

	kg, pcs := kgThenPcsFormulas()
	change := runUpdateLatestForAvgKgStock(t, uuid.New(), []models.PriceListSubGroupFormulasMap{kg, pcs})

	// ไม่มี inventory → avgKgStock ใช้ fallback 1.0 → pcs = 55 * 1.0
	assertWeightAndUnit(t, change, 55.0, 55.0)
}

// TestUpdateLatestCorrectWhenFormulaOrderReversed — จำลอง 522 subgroup ที่ create_dtm ของ
// สูตร pcs ใหม่กว่า จึงถูกส่งมาก่อน kg ต้องได้ผลเท่าเดิมจากการคำนวณครั้งเดียว
func TestUpdateLatestCorrectWhenFormulaOrderReversed(t *testing.T) {
	pointInventoryEndpointAtFake(t, `[]`)

	kg, pcs := kgThenPcsFormulas()
	change := runUpdateLatestForAvgKgStock(t, uuid.New(), []models.PriceListSubGroupFormulasMap{pcs, kg})

	assertWeightAndUnit(t, change, 55.0, 55.0)
}

// TestUpdateLatestIsDeterministicAcrossRuns — input เดียวกันต้องให้ผลเท่ากันทุกครั้ง
// (ลำดับสูตรมาจาก map ซึ่งไม่มีลำดับรับประกัน การเรียงต้องทำให้ผลคงที่)
func TestUpdateLatestIsDeterministicAcrossRuns(t *testing.T) {
	pointInventoryEndpointAtFake(t, `[]`)

	kg, pcs := kgThenPcsFormulas()
	subGroupID := uuid.New()

	for i := 0; i < 20; i++ {
		change := runUpdateLatestForAvgKgStock(t, subGroupID, []models.PriceListSubGroupFormulasMap{pcs, kg})
		if *change.TotalNetPriceUnit != 55.0 || *change.TotalNetPriceWeight != 55.0 {
			t.Fatalf("run %d: unit=%v weight=%v, want 55/55 — ผลลัพธ์ไม่คงที่",
				i, *change.TotalNetPriceUnit, *change.TotalNetPriceWeight)
		}
	}
}

// TestUpdateLatestUsesAvgProductNotAvgWeight — พิสูจน์บั๊ก A: avg_kg_stock ต้องมาจาก
// AvgProduct (ระดับ site) ไม่ใช่ AvgWeight (ระดับ batch) ที่ต่างกันชัดเจนใน fixture นี้
func TestUpdateLatestUsesAvgProductNotAvgWeight(t *testing.T) {
	subGroupID := uuid.New()

	// inventory_weight[0]: AvgWeight 32.0 (batch) vs AvgProduct 2.0 (site)
	pointInventoryEndpointAtFake(t, `[{
		"id": "`+subGroupID.String()+`",
		"product_code": "P001",
		"supplier_code": "SUP01",
		"weight_spec": 3.0,
		"inventory_weight": [{
			"product_code": "P001",
			"batch_no": "B001",
			"avg_product": 2.0,
			"avg_batch": 32.0,
			"avg_weight": 32.0,
			"total_qty": 7.0,
			"total_weight": 224.0
		}]
	}]`)

	kg, pcs := kgThenPcsFormulas()
	change := runUpdateLatestForAvgKgStock(t, subGroupID, []models.PriceListSubGroupFormulasMap{kg, pcs})

	if change.TotalNetPriceUnit == nil {
		t.Fatal("TotalNetPriceUnit ต้องไม่เป็น nil")
	}
	// pcs = total_net_price_weight (55) * avg_kg_stock
	// 110 = 55 * 2.0 (AvgProduct, ถูกต้อง) · 1760 = 55 * 32.0 (AvgWeight, บั๊ก A)
	if got := *change.TotalNetPriceUnit; got != 110.0 {
		t.Errorf(`total_net_price_unit = %v, want 110
  110  = ถูกต้อง: avg_kg_stock อ่าน AvgProduct = 2.0 (ค่าเฉลี่ยระดับ site)
  1760 = ผิด: อ่าน AvgWeight = 32.0 (ค่าเฉลี่ยระดับ batch - บั๊ก A)
  7168 = ผิด: อ่านน้ำหนักสต็อก total_weight = 224 เข้าสูตร`, got)
	}
}
