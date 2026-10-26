package priceService

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	externalService "prime-erp-core/external/warehouse-service"
	"prime-erp-core/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixture จำลอง GROUP_1_ITEM_6 บน prod: group ไม่มีราคาฐาน ผู้ใช้กรอกราคา kg ของ subgroup เอง
// สูตร kg = base_price+extra (default) และ pcs = kg*weight_spec
const item6Code = "GROUP_1_ITEM_6_SUB"

var (
	kgBaseFormula  = item6Formula("KG", "kg", "base_price+extra", `{"required":["base_price","extra"]}`, true)
	pcsFromKg      = item6Formula("PCS", "pcs", "kg*weight_spec", `{"required":["kg","weight_spec"]}`, false)
	kgFromPcs      = item6Formula("KG_PCS", "kg", "pcs/weight_spec", `{"required":["pcs","weight_spec"]}`, false)
	pcsFromBase    = item6Formula("PCS_BASE", "pcs", "(base_price+extra)*weight_spec", `{"required":["base_price","extra","weight_spec"]}`, false)
	pcsFromAvg     = item6Formula("PCS_AVG", "pcs", "kg*avg_kg_stock", `{"required":["kg","avg_kg_stock"]}`, false)
	pcsBasePlusExt = item6Formula("PCS_BE", "pcs", "base_price+extra", `{"required":["base_price","extra"]}`, false)
)

func item6SubGroup(basePriceWeight float64) models.PriceListSubGroup {
	id := uuid.New()
	return models.PriceListSubGroup{
		ID:                  id,
		SubGroupCode:        item6Code,
		TotalNetPriceWeight: 4150,
		TotalNetPriceUnit:   1,
		NeedRecalculate:     true,
		PriceListGroup: models.PriceListGroup{
			PriceWeight: basePriceWeight,
			// rule ที่ key ตรงกับ subgroup: calculateExtraForSubGroup จะคืน 7
			PriceListGroupExtras: []models.PriceListGroupExtra{{
				ValueInt:                7,
				PriceListGroupExtraKeys: []models.PriceListGroupExtraKey{{Code: "PRODUCT", Value: "P001"}},
			}},
		},
		PriceListSubGroupKeys: []models.PriceListSubGroupKey{{SubGroupID: id, Code: "PRODUCT", Value: "P001", Seq: 1}},
	}
}

func item6Formula(code, uom, expression, params string, isDefault bool) models.PriceListSubGroupFormulasMap {
	return models.PriceListSubGroupFormulasMap{
		PriceListSubGroupCode: item6Code,
		PriceListFormulasCode: code,
		IsDefault:             isDefault,
		PriceListFormulas: models.PriceListFormulas{
			FormulaCode: code,
			Uom:         uom,
			FormulaType: "price_calc",
			Expression:  expression,
			Params:      json.RawMessage(params),
			Rounding:    2,
		},
	}
}

// stockStub คือผล warehouse ที่เทสแก้ weight_spec / avg / error ได้ และนับจำนวนครั้งที่ถูกเรียก
type stockStub struct {
	weightSpec float64
	avgProduct float64
	err        error
	calls      int
}

// stubItem6Deps stub getter ทุกตัว ไม่ส่ง formulas = คู่ item6 (ใส่ pcs ก่อน sortFormulasByDependency ต้องเรียงให้)
func stubItem6Deps(t *testing.T, stored models.PriceListSubGroup, formulas ...models.PriceListSubGroupFormulasMap) *stockStub {
	t.Helper()
	origByIDs := getPriceListSubGroupsByIDsFunc
	origByGroup := getPriceListSubGroupsByGroupCodesFunc
	origFormulas := getPriceListSubGroupFormulasMapBySubGroupCodesFunc
	origItems := loadGroupItemValueIntsFunc
	origInventory := getInventoryWeightByKeyFunc
	t.Cleanup(func() {
		getPriceListSubGroupsByIDsFunc = origByIDs
		getPriceListSubGroupsByGroupCodesFunc = origByGroup
		getPriceListSubGroupFormulasMapBySubGroupCodesFunc = origFormulas
		loadGroupItemValueIntsFunc = origItems
		getInventoryWeightByKeyFunc = origInventory
	})

	if len(formulas) == 0 {
		formulas = []models.PriceListSubGroupFormulasMap{pcsFromKg, kgBaseFormula}
	}
	getPriceListSubGroupsByIDsFunc = func([]uuid.UUID) ([]models.PriceListSubGroup, error) {
		return []models.PriceListSubGroup{stored}, nil
	}
	getPriceListSubGroupsByGroupCodesFunc = func([]string) ([]models.PriceListSubGroup, error) {
		return []models.PriceListSubGroup{stored}, nil
	}
	getPriceListSubGroupFormulasMapBySubGroupCodesFunc = func([]string) (map[string][]models.PriceListSubGroupFormulasMap, error) {
		return map[string][]models.PriceListSubGroupFormulasMap{item6Code: formulas}, nil
	}
	loadGroupItemValueIntsFunc = func([]models.PriceListSubGroup) (groupItemValueInts, error) {
		return groupItemValueInts{}, nil
	}
	stock := &stockStub{weightSpec: 148}
	getInventoryWeightByKeyFunc = func(_ context.Context, _ string, _ []string, _ []externalService.InventoryByProductCodeKeyValue) ([]externalService.InventoryByProductCodeResponse, error) {
		stock.calls++
		if stock.err != nil {
			return nil, stock.err
		}
		resp := externalService.InventoryByProductCodeResponse{ID: stored.ID.String(), WeightSpec: stock.weightSpec}
		if stock.avgProduct > 0 {
			resp.InventoryWeight = []models.InventoryWeightResponse{{AvgProduct: stock.avgProduct}}
		}
		return []externalService.InventoryByProductCodeResponse{resp}, nil
	}
	return stock
}

func captureUpdateLatest(t *testing.T) *[]models.UpdatePriceListSubGroupRequest {
	t.Helper()
	calls := []models.UpdatePriceListSubGroupRequest{}
	orig := updateLatestSubGroupFunc
	t.Cleanup(func() { updateLatestSubGroupFunc = orig })
	updateLatestSubGroupFunc = func(req models.UpdatePriceListSubGroupRequest) error {
		calls = append(calls, req)
		return nil
	}
	return &calls
}

func captureSave(t *testing.T) *[]models.UpdatePriceListSubGroupRequest {
	t.Helper()
	calls := []models.UpdatePriceListSubGroupRequest{}
	orig := updateSubGroupFunc
	t.Cleanup(func() { updateSubGroupFunc = orig })
	updateSubGroupFunc = func(req models.UpdatePriceListSubGroupRequest) error {
		calls = append(calls, req)
		return nil
	}
	return &calls
}

func save(t *testing.T, change map[string]interface{}) error {
	t.Helper()
	body, err := json.Marshal(map[string]interface{}{"changes": []map[string]interface{}{change}})
	require.NoError(t, err)
	_, err = UpdatePriceListSubGroup(context.Background(), string(body))
	return err
}

// saveOnce save แล้วคืน change เดียวที่ถูกเขียน (ต้องเขียนครั้งเดียวเสมอ)
func saveOnce(t *testing.T, change map[string]interface{}) models.UpdatePriceListSubGroupItem {
	t.Helper()
	calls := captureSave(t)
	require.NoError(t, save(t, change))
	require.Len(t, *calls, 1, "save ต้องเขียน DB ครั้งเดียว")
	assert.True(t, (*calls)[0].ClearRecalculate)
	require.Len(t, (*calls)[0].Changes, 1)
	return (*calls)[0].Changes[0]
}

func TestSkipFormulaWithoutBasePrice(t *testing.T) {
	assert.True(t, skipFormulaWithoutBasePrice(kgBaseFormula.PriceListFormulas, 0))
	assert.False(t, skipFormulaWithoutBasePrice(kgBaseFormula.PriceListFormulas, 4000))
	assert.False(t, skipFormulaWithoutBasePrice(pcsFromKg.PriceListFormulas, 0))
	assert.False(t, skipFormulaWithoutBasePrice(models.PriceListFormulas{}, 0), "ไม่มี params ต้องคำนวณตามเดิม")
}

// ---- guard ใน update-latest / preview ----

func TestRunUpdateLatest_BasePriceZeroKeepsKgAndComputesPcs(t *testing.T) {
	stored := item6SubGroup(0)
	stubItem6Deps(t, stored)
	calls := captureUpdateLatest(t)

	_, err := RunUpdateLatestPriceListSubGroup(context.Background(), models.UpdateLatestPriceListSubGroupRequest{
		SubGroupIDs: []string{stored.ID.String()},
	})
	require.NoError(t, err)
	change := (*calls)[0].Changes[0]
	assert.Equal(t, 4150.0, *change.TotalNetPriceWeight, "kg ที่กรอกเองต้องไม่ถูกทับด้วย 0+extra")
	assert.Equal(t, 614200.0, *change.TotalNetPriceUnit, "4150 x 148")
	assert.Equal(t, 7.0, *change.ExtraPriceWeight, "เส้นปกติยังคำนวณ extra ตามเดิม")
}

func TestRunUpdateLatest_GroupCascadeBasePriceZeroKeepsKg(t *testing.T) {
	stored := item6SubGroup(0)
	stubItem6Deps(t, stored)
	calls := captureUpdateLatest(t)

	_, err := RunUpdateLatestPriceListSubGroup(context.Background(), models.UpdateLatestPriceListSubGroupRequest{
		UpdateType: "group",
		GroupCodes: []string{"GROUP_1_ITEM_6"},
	})
	require.NoError(t, err)
	change := (*calls)[0].Changes[0]
	assert.Equal(t, 4150.0, *change.TotalNetPriceWeight)
	assert.Equal(t, 614200.0, *change.TotalNetPriceUnit)
}

func TestRunUpdateLatest_BasePriceSetRunsKgFormula(t *testing.T) {
	stored := item6SubGroup(4000)
	stubItem6Deps(t, stored)
	calls := captureUpdateLatest(t)

	_, err := RunUpdateLatestPriceListSubGroup(context.Background(), models.UpdateLatestPriceListSubGroupRequest{
		SubGroupIDs: []string{stored.ID.String()},
	})
	require.NoError(t, err)
	change := (*calls)[0].Changes[0]
	assert.Equal(t, 4007.0, *change.TotalNetPriceWeight, "4000 + extra 7")
	assert.Equal(t, 593036.0, *change.TotalNetPriceUnit, "4007 x 148")
}

// ราคาฐานแยกตาม uom: PriceWeight > 0 แต่ PriceUnit = 0 → kg คำนวณ ส่วน pcs ที่ต้องใช้ base ถูกข้าม
func TestRunUpdateLatest_BasePriceSplitPerUom(t *testing.T) {
	stored := item6SubGroup(4000)
	stubItem6Deps(t, stored, kgBaseFormula, pcsBasePlusExt)
	calls := captureUpdateLatest(t)

	_, err := RunUpdateLatestPriceListSubGroup(context.Background(), models.UpdateLatestPriceListSubGroupRequest{
		SubGroupIDs: []string{stored.ID.String()},
	})
	require.NoError(t, err)
	change := (*calls)[0].Changes[0]
	assert.Equal(t, 4007.0, *change.TotalNetPriceWeight)
	assert.Equal(t, 1.0, *change.TotalNetPriceUnit, "PriceUnit = 0 ต้องคง pcs เดิม")
}

func TestGetCalculated_BasePriceZeroKeepsKgAndComputesPcs(t *testing.T) {
	stored := item6SubGroup(0)
	stubItem6Deps(t, stored)

	body, _ := json.Marshal(map[string]interface{}{"subgroup_ids": []string{stored.ID.String()}})
	resp, err := GetCalculatedPriceListSubGroup(context.Background(), string(body))
	require.NoError(t, err)
	item := resp.(models.GetCalculatedPriceListSubGroupResponse).Data[0]
	assert.Equal(t, 4150.0, item.TotalNetPriceWeight)
	assert.Equal(t, 614200.0, item.TotalNetPriceUnit)
}

// ---- save หน้า detail ----

func TestUpdatePriceListSubGroup_KgEditComputesPcsInSameWrite(t *testing.T) {
	// ราคาฐาน > 0 เพื่อพิสูจน์ว่าสูตร kg ไม่รันเพราะผู้ใช้กรอก kg เอง ไม่ใช่เพราะ guard
	stored := item6SubGroup(9999)
	stored.TotalNetPriceWeight = 0
	stubItem6Deps(t, stored)
	loadGroupItemValueIntsFunc = func([]models.PriceListSubGroup) (groupItemValueInts, error) {
		t.Fatal("save ต้องไม่คำนวณ extra ใหม่")
		return nil, nil
	}

	change := saveOnce(t, map[string]interface{}{"subgroup_id": stored.ID.String(), "total_net_price_weight": 4150})
	assert.Equal(t, 4150.0, *change.TotalNetPriceWeight, "kg ที่กรอกชนะสูตร")
	assert.Equal(t, 614200.0, *change.TotalNetPriceUnit, "pcs คิดจาก kg ใน request 4150 x 148 ไม่ใช่ค่าใน DB")
	assert.Nil(t, change.ExtraPriceWeight)
	assert.Nil(t, change.ExtraPriceUnit)
}

func TestUpdatePriceListSubGroup_PcsEditKeepsPcs(t *testing.T) {
	stored := item6SubGroup(0)
	stock := stubItem6Deps(t, stored)

	change := saveOnce(t, map[string]interface{}{"subgroup_id": stored.ID.String(), "total_net_price_unit": 700000})
	assert.Equal(t, 700000.0, *change.TotalNetPriceUnit)
	assert.Nil(t, change.TotalNetPriceWeight, "kg = base_price+extra ไม่ขึ้นกับ pcs ต้องไม่รัน")
	assert.Zero(t, stock.calls, "ไม่มีสูตรที่ต้องใช้สต็อก ต้องไม่เรียก warehouse")
}

func TestUpdatePriceListSubGroup_PcsEditComputesKg(t *testing.T) {
	stored := item6SubGroup(0)
	stubItem6Deps(t, stored, kgFromPcs, pcsFromKg)

	change := saveOnce(t, map[string]interface{}{"subgroup_id": stored.ID.String(), "total_net_price_unit": 614200})
	assert.Equal(t, 614200.0, *change.TotalNetPriceUnit)
	assert.Equal(t, 4150.0, *change.TotalNetPriceWeight, "614200 / 148")
}

func TestUpdatePriceListSubGroup_KgEditSkipsFormulaNotDependingOnKg(t *testing.T) {
	stored := item6SubGroup(4000)
	stock := stubItem6Deps(t, stored, kgBaseFormula, pcsFromBase)

	change := saveOnce(t, map[string]interface{}{"subgroup_id": stored.ID.String(), "total_net_price_weight": 4150})
	assert.Nil(t, change.TotalNetPriceUnit, "(base_price+extra)*weight_spec ไม่ขึ้นกับ kg ต้องไม่รันตอน save")
	assert.Zero(t, stock.calls)
}

func TestUpdatePriceListSubGroup_MissingWeightSpecSkipsPcs(t *testing.T) {
	stored := item6SubGroup(0)
	stock := stubItem6Deps(t, stored)
	stock.weightSpec = 0

	change := saveOnce(t, map[string]interface{}{"subgroup_id": stored.ID.String(), "total_net_price_weight": 4150})
	assert.Nil(t, change.TotalNetPriceUnit, "ห้ามบันทึก pcs ที่คิดจาก weight_spec fallback 1.0")
}

func TestUpdatePriceListSubGroup_AvgKgStock(t *testing.T) {
	stored := item6SubGroup(0)
	stock := stubItem6Deps(t, stored, kgBaseFormula, pcsFromAvg)

	change := saveOnce(t, map[string]interface{}{"subgroup_id": stored.ID.String(), "total_net_price_weight": 4150})
	assert.Nil(t, change.TotalNetPriceUnit, "ไม่มีสต็อก = ไม่มี avg_kg_stock ต้องข้าม")

	stock.avgProduct = 2
	change = saveOnce(t, map[string]interface{}{"subgroup_id": stored.ID.String(), "total_net_price_weight": 4150})
	assert.Equal(t, 8300.0, *change.TotalNetPriceUnit, "4150 x 2")
}

func TestUpdatePriceListSubGroup_InventoryErrorWritesNothing(t *testing.T) {
	stored := item6SubGroup(0)
	stock := stubItem6Deps(t, stored)
	stock.err = errors.New("warehouse down")
	calls := captureSave(t)

	err := save(t, map[string]interface{}{"subgroup_id": stored.ID.String(), "total_net_price_weight": 4150})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to calculate dependent prices")
	assert.Empty(t, *calls, "คำนวณไม่ได้ต้องไม่มีอะไรถูกเขียน")
}

func TestUpdatePriceListSubGroup_FormulaErrorWritesNothing(t *testing.T) {
	stored := item6SubGroup(0)
	stubItem6Deps(t, stored, kgBaseFormula, item6Formula("BAD", "pcs", "kg*", `{"required":["kg"]}`, false))
	calls := captureSave(t)

	require.Error(t, save(t, map[string]interface{}{"subgroup_id": stored.ID.String(), "total_net_price_weight": 4150}))
	assert.Empty(t, *calls)
}

func TestUpdatePriceListSubGroup_DefaultInputFormulaComputesNothing(t *testing.T) {
	stored := item6SubGroup(0)
	input := item6Formula("IN", "kg", "", `{}`, true)
	input.PriceListFormulas.FormulaType = "input"
	stubItem6Deps(t, stored, input, pcsFromKg)

	change := saveOnce(t, map[string]interface{}{"subgroup_id": stored.ID.String(), "total_net_price_weight": 4150})
	assert.Nil(t, change.TotalNetPriceUnit)
}

func TestUpdatePriceListSubGroup_FetchErrorWritesNothing(t *testing.T) {
	stored := item6SubGroup(0)
	stubItem6Deps(t, stored)
	calls := captureSave(t)

	getPriceListSubGroupsByIDsFunc = func([]uuid.UUID) ([]models.PriceListSubGroup, error) {
		return nil, errors.New("db down")
	}
	require.Error(t, save(t, map[string]interface{}{"subgroup_id": stored.ID.String(), "total_net_price_weight": 4150}))

	getPriceListSubGroupsByIDsFunc = func([]uuid.UUID) ([]models.PriceListSubGroup, error) {
		return []models.PriceListSubGroup{stored}, nil
	}
	getPriceListSubGroupFormulasMapBySubGroupCodesFunc = func([]string) (map[string][]models.PriceListSubGroupFormulasMap, error) {
		return nil, errors.New("db down")
	}
	require.Error(t, save(t, map[string]interface{}{"subgroup_id": stored.ID.String(), "total_net_price_weight": 4150}))
	assert.Empty(t, *calls)
}

// backfill: kg ใน request เท่ากับค่าใน DB (เคส prod ที่ pcs ค้าง 0) ต้องยังคำนวณ pcs ให้
func TestUpdatePriceListSubGroup_SameKgBackfillsPcs(t *testing.T) {
	stored := item6SubGroup(0)
	stored.TotalNetPriceUnit = 0
	stubItem6Deps(t, stored)

	change := saveOnce(t, map[string]interface{}{"subgroup_id": stored.ID.String(), "total_net_price_weight": 4150})
	assert.Equal(t, 4150.0, *change.TotalNetPriceWeight)
	require.NotNil(t, change.TotalNetPriceUnit)
	assert.Equal(t, 614200.0, *change.TotalNetPriceUnit, "4150 x 148")
}
