package priceService

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	externalService "prime-erp-core/external/warehouse-service"
	"prime-erp-core/internal/models"
	"prime-erp-core/internal/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func calcSubGroup(needRecalculate bool) models.PriceListSubGroup {
	return models.PriceListSubGroup{
		ID:                        uuid.New(),
		SubGroupCode:              "SUB001",
		ExtraPriceUnit:            2,
		ExtraPriceWeight:          5,
		TotalNetPriceUnit:         90,
		TotalNetPriceWeight:       45,
		BeforeTotalNetPriceUnit:   80,
		BeforeTotalNetPriceWeight: 40,
		NeedRecalculate:           needRecalculate,
		PriceListGroup:            models.PriceListGroup{PriceUnit: 100, PriceWeight: 50},
	}
}

func calcFormula(code, uom string) models.PriceListSubGroupFormulasMap {
	return models.PriceListSubGroupFormulasMap{
		ID:                    uuid.New(),
		PriceListSubGroupCode: "SUB001",
		PriceListFormulasCode: code,
		IsDefault:             uom == "kg",
		PriceListFormulas: models.PriceListFormulas{
			ID:          uuid.New(),
			FormulaCode: code,
			Name:        code,
			Uom:         uom,
			FormulaType: "calculation",
			Expression:  "base_price + extra",
			Params:      json.RawMessage(`{}`),
			Rounding:    2,
		},
	}
}

func stubGetCalculatedDeps(t testing.TB, subGroups []models.PriceListSubGroup) {
	t.Helper()
	origByIDs := getPriceListSubGroupsByIDsFunc
	origByGroupCodes := getPriceListSubGroupsByGroupCodesFunc
	origFormulas := getPriceListSubGroupFormulasMapBySubGroupCodesFunc
	origItems := loadGroupItemValueIntsFunc
	origInventory := getInventoryWeightByKeyFunc
	t.Cleanup(func() {
		getPriceListSubGroupsByIDsFunc = origByIDs
		getPriceListSubGroupsByGroupCodesFunc = origByGroupCodes
		getPriceListSubGroupFormulasMapBySubGroupCodesFunc = origFormulas
		loadGroupItemValueIntsFunc = origItems
		getInventoryWeightByKeyFunc = origInventory
	})

	getPriceListSubGroupsByIDsFunc = func([]uuid.UUID) ([]models.PriceListSubGroup, error) {
		return subGroups, nil
	}
	getPriceListSubGroupsByGroupCodesFunc = func([]string) ([]models.PriceListSubGroup, error) {
		return subGroups, nil
	}
	getPriceListSubGroupFormulasMapBySubGroupCodesFunc = func([]string) (map[string][]models.PriceListSubGroupFormulasMap, error) {
		return map[string][]models.PriceListSubGroupFormulasMap{
			"SUB001": {calcFormula("F_KG", "kg"), calcFormula("F_PCS", "pcs")},
		}, nil
	}
	loadGroupItemValueIntsFunc = func([]models.PriceListSubGroup) (groupItemValueInts, error) {
		return groupItemValueInts{}, nil
	}
	getInventoryWeightByKeyFunc = func(context.Context, string, []string, []externalService.InventoryByProductCodeKeyValue) ([]externalService.InventoryByProductCodeResponse, error) {
		return nil, nil
	}
}

func getCalculated(t *testing.T, payload map[string]interface{}) map[string]models.GetCalculatedPriceListSubGroupItem {
	t.Helper()
	body, _ := json.Marshal(payload)
	resp, err := GetCalculatedPriceListSubGroup(context.Background(), string(body))
	require.NoError(t, err)
	result, ok := resp.(models.GetCalculatedPriceListSubGroupResponse)
	require.True(t, ok, "unexpected response type %T", resp)
	require.True(t, result.Success)

	items := make(map[string]models.GetCalculatedPriceListSubGroupItem, len(result.Data))
	for _, item := range result.Data {
		items[item.SubGroupID] = item
	}
	return items
}

func TestGetCalculatedPriceListSubGroup_RespectsRecalculateFlag(t *testing.T) {
	flagged := calcSubGroup(true)
	unflagged := calcSubGroup(false)
	stubGetCalculatedDeps(t, []models.PriceListSubGroup{flagged, unflagged})

	items := getCalculated(t, map[string]interface{}{
		"subgroup_ids": []string{flagged.ID.String(), unflagged.ID.String()},
	})
	require.Len(t, items, 2)

	// flag = true: คำนวณใหม่ตามสูตร before = after เดิมใน DB
	got := items[flagged.ID.String()]
	// สูตร base_price + extra: kg = 50 + 5, pcs = 100 + 2
	assert.Equal(t, 55.0, got.TotalNetPriceWeight, "flag=true ต้องคำนวณใหม่")
	assert.Equal(t, 102.0, got.TotalNetPriceUnit, "flag=true ต้องคำนวณใหม่")
	assert.Equal(t, 45.0, got.BeforeTotalNetPriceWeight)
	assert.Equal(t, 90.0, got.BeforeTotalNetPriceUnit)
	assert.Equal(t, "kg", got.DefaultUom)

	// flag = false: ไม่คำนวณ before = after = after เดิมใน DB
	got = items[unflagged.ID.String()]
	assert.Equal(t, models.GetCalculatedPriceListSubGroupItem{
		SubGroupID:                unflagged.ID.String(),
		TotalNetPriceUnit:         90,
		TotalNetPriceWeight:       45,
		ExtraPriceUnit:            2,
		ExtraPriceWeight:          5,
		BeforeTotalNetPriceUnit:   90,
		BeforeTotalNetPriceWeight: 45,
	}, got)
}

func TestGetCalculatedPriceListSubGroup_GroupModeRespectsFlag(t *testing.T) {
	unflagged := calcSubGroup(false)
	stubGetCalculatedDeps(t, []models.PriceListSubGroup{unflagged})

	items := getCalculated(t, map[string]interface{}{
		"update_type": "group",
		"group_codes": []string{"GROUP_1_ITEM_1"},
	})

	got := items[unflagged.ID.String()]
	assert.Equal(t, 45.0, got.TotalNetPriceWeight)
	assert.Equal(t, 45.0, got.BeforeTotalNetPriceWeight)
}

func TestGetCalculatedPriceListSubGroup_Validation(t *testing.T) {
	stubGetCalculatedDeps(t, nil)

	cases := map[string]map[string]interface{}{
		"missing subgroup_ids": {},
		"invalid uuid":         {"subgroup_ids": []string{"not-a-uuid"}},
		"missing group_codes":  {"update_type": "group"},
		"subgroup not found":   {"subgroup_ids": []string{uuid.NewString()}},
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			body, _ := json.Marshal(payload)
			_, err := GetCalculatedPriceListSubGroup(context.Background(), string(body))
			var bindingErr *utils.BindingError
			assert.ErrorAs(t, err, &bindingErr)
		})
	}
}

func TestGetCalculatedPriceListSubGroup_DependencyErrors(t *testing.T) {
	boom := errors.New("boom")
	sg := calcSubGroup(true)

	cases := map[string]struct {
		payload  map[string]interface{}
		sabotage func()
	}{
		"fetch by ids": {
			map[string]interface{}{"subgroup_ids": []string{sg.ID.String()}},
			func() {
				getPriceListSubGroupsByIDsFunc = func([]uuid.UUID) ([]models.PriceListSubGroup, error) { return nil, boom }
			},
		},
		"fetch by group codes": {
			map[string]interface{}{"update_type": "group", "group_codes": []string{"G"}},
			func() {
				getPriceListSubGroupsByGroupCodesFunc = func([]string) ([]models.PriceListSubGroup, error) { return nil, boom }
			},
		},
		"formulas": {
			map[string]interface{}{"subgroup_ids": []string{sg.ID.String()}},
			func() {
				getPriceListSubGroupFormulasMapBySubGroupCodesFunc = func([]string) (map[string][]models.PriceListSubGroupFormulasMap, error) {
					return nil, boom
				}
			},
		},
		"group item values": {
			map[string]interface{}{"subgroup_ids": []string{sg.ID.String()}},
			func() {
				loadGroupItemValueIntsFunc = func([]models.PriceListSubGroup) (groupItemValueInts, error) { return nil, boom }
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			stubGetCalculatedDeps(t, []models.PriceListSubGroup{sg})
			tc.sabotage()
			body, _ := json.Marshal(tc.payload)
			_, err := GetCalculatedPriceListSubGroup(context.Background(), string(body))
			assert.ErrorIs(t, err, boom)
		})
	}
}

func TestGetCalculatedPriceListSubGroup_GroupModeNoSubGroups(t *testing.T) {
	stubGetCalculatedDeps(t, nil)
	body, _ := json.Marshal(map[string]interface{}{"update_type": "group", "group_codes": []string{"G"}})
	_, err := GetCalculatedPriceListSubGroup(context.Background(), string(body))
	var bindingErr *utils.BindingError
	assert.ErrorAs(t, err, &bindingErr)
}

func TestGetCalculatedPriceListSubGroup_InvalidJSON(t *testing.T) {
	_, err := GetCalculatedPriceListSubGroup(context.Background(), "{")
	var bindingErr *utils.BindingError
	assert.ErrorAs(t, err, &bindingErr)
}

func TestGetCalculatedPriceListSubGroup_DefaultInputFormulaKeepsValues(t *testing.T) {
	sg := calcSubGroup(true)
	stubGetCalculatedDeps(t, []models.PriceListSubGroup{sg})
	input := calcFormula("F_IN", "kg")
	input.PriceListFormulas.FormulaType = "input"
	getPriceListSubGroupFormulasMapBySubGroupCodesFunc = func([]string) (map[string][]models.PriceListSubGroupFormulasMap, error) {
		return map[string][]models.PriceListSubGroupFormulasMap{"SUB001": {input}}, nil
	}

	got := getCalculated(t, map[string]interface{}{"subgroup_ids": []string{sg.ID.String()}})[sg.ID.String()]
	assert.Equal(t, 90.0, got.TotalNetPriceUnit)
	assert.Equal(t, 45.0, got.TotalNetPriceWeight)
}

func TestGetCalculatedPriceListSubGroup_InvalidExpressionReturnsError(t *testing.T) {
	sg := calcSubGroup(true)
	stubGetCalculatedDeps(t, []models.PriceListSubGroup{sg})
	bad := calcFormula("F_KG", "kg")
	bad.PriceListFormulas.Expression = "((("
	getPriceListSubGroupFormulasMapBySubGroupCodesFunc = func([]string) (map[string][]models.PriceListSubGroupFormulasMap, error) {
		return map[string][]models.PriceListSubGroupFormulasMap{"SUB001": {bad}}, nil
	}

	body, _ := json.Marshal(map[string]interface{}{"subgroup_ids": []string{sg.ID.String()}})
	_, err := GetCalculatedPriceListSubGroup(context.Background(), string(body))
	assert.Error(t, err)
}

// benchSubGroups สร้าง subgroup จำลองขนาด n โดยมี formula code วน 20 แบบ และ key 3 ตัวต่อ subgroup
func benchSubGroups(n int, flagged bool) []models.PriceListSubGroup {
	subGroups := make([]models.PriceListSubGroup, n)
	for i := range subGroups {
		id := uuid.New()
		subGroups[i] = models.PriceListSubGroup{
			ID:                  id,
			SubGroupCode:        fmt.Sprintf("SUB_%02d", i%20),
			ExtraPriceUnit:      2,
			ExtraPriceWeight:    5,
			TotalNetPriceUnit:   90,
			TotalNetPriceWeight: 45,
			NeedRecalculate:     flagged,
			PriceListGroup:      models.PriceListGroup{CompanyCode: "C1", SiteCode: "S1", PriceUnit: 100, PriceWeight: 50},
			PriceListSubGroupKeys: []models.PriceListSubGroupKey{
				{SubGroupID: id, Code: "SIZE", Value: fmt.Sprint(i), Seq: 1},
				{SubGroupID: id, Code: "GRADE", Value: "A", Seq: 2},
				{SubGroupID: id, Code: "LENGTH", Value: "6M", Seq: 3},
			},
		}
	}
	return subGroups
}

// BenchmarkGetCalculatedPriceListSubGroup จำลอง latency ของ DB (5ms ต่อ query) และ
// warehouse-core (20ms ต่อ call) แล้วนับจำนวนครั้งที่เรียก loader แต่ละตัวต่อ op
func BenchmarkGetCalculatedPriceListSubGroup(b *testing.B) {
	const dbLatency = 5 * time.Millisecond
	const inventoryLatency = 20 * time.Millisecond

	for _, n := range []int{50, 500, 1200} {
		for _, flagged := range []bool{true, false} {
			name := fmt.Sprintf("N=%d/flagged=%t", n, flagged)
			b.Run(name, func(b *testing.B) {
				subGroups := benchSubGroups(n, flagged)
				ids := make([]string, n)
				for i, sg := range subGroups {
					ids[i] = sg.ID.String()
				}
				body, _ := json.Marshal(map[string]interface{}{"subgroup_ids": ids})

				stubGetCalculatedDeps(b, subGroups)
				var formulaCalls, groupItemCalls, inventoryCalls int
				getPriceListSubGroupFormulasMapBySubGroupCodesFunc = func(codes []string) (map[string][]models.PriceListSubGroupFormulasMap, error) {
					formulaCalls++
					time.Sleep(dbLatency)
					kg, pcs := kgThenPcsFormulas()
					kg.IsDefault = true
					out := make(map[string][]models.PriceListSubGroupFormulasMap, len(codes))
					for _, code := range codes {
						// pcs อ้าง kg และมาก่อนตามลำดับ create_dtm จึงต้องผ่าน sortFormulasByDependency จริง
						out[code] = []models.PriceListSubGroupFormulasMap{pcs, kg}
					}
					return out, nil
				}
				loadGroupItemValueIntsFunc = func([]models.PriceListSubGroup) (groupItemValueInts, error) {
					groupItemCalls++
					time.Sleep(dbLatency)
					return groupItemValueInts{}, nil
				}
				getInventoryWeightByKeyFunc = func(_ context.Context, _ string, _ []string, keyValues []externalService.InventoryByProductCodeKeyValue) ([]externalService.InventoryByProductCodeResponse, error) {
					inventoryCalls++
					time.Sleep(inventoryLatency)
					return nil, nil
				}

				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, err := GetCalculatedPriceListSubGroup(context.Background(), string(body)); err != nil {
						b.Fatal(err)
					}
				}
				b.StopTimer()
				b.ReportMetric(float64(formulaCalls)/float64(b.N), "formulaCalls/op")
				b.ReportMetric(float64(groupItemCalls)/float64(b.N), "groupItemCalls/op")
				b.ReportMetric(float64(inventoryCalls)/float64(b.N), "inventoryCalls/op")
			})
		}
	}
}
