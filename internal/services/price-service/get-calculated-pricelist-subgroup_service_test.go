package priceService

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

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

func stubGetCalculatedDeps(t *testing.T, subGroups []models.PriceListSubGroup) {
	t.Helper()
	origByIDs := getPriceListSubGroupsByIDsFunc
	origByGroupCodes := getPriceListSubGroupsByGroupCodesFunc
	origFormulas := getPriceListSubGroupFormulasMapBySubGroupCodesFunc
	origItems := loadGroupItemValueIntsFunc
	t.Cleanup(func() {
		getPriceListSubGroupsByIDsFunc = origByIDs
		getPriceListSubGroupsByGroupCodesFunc = origByGroupCodes
		getPriceListSubGroupFormulasMapBySubGroupCodesFunc = origFormulas
		loadGroupItemValueIntsFunc = origItems
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
