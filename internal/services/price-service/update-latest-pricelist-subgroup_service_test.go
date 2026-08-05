package priceService

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"prime-erp-core/internal/models"
	"prime-erp-core/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUpdateLatestPriceListSubGroup_ValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		payload    interface{}
		assertFunc func(*testing.T, error)
	}{
		{
			name:    "missing subgroup ids",
			payload: map[string]interface{}{
				// default update_type=subgroup
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				_, ok := err.(*utils.BindingError)
				assert.True(t, ok)
			},
		},
		{
			name: "invalid uuid",
			payload: map[string]interface{}{
				// default update_type=subgroup
				"subgroup_ids": []string{"not-a-uuid"},
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				if bindingErr, ok := err.(*utils.BindingError); ok {
					assert.Contains(t, bindingErr.Message, "valid UUID")
				} else {
					t.Fatalf("expected binding error, got %T", err)
				}
			},
		},
		{
			name: "group update missing group_codes",
			payload: map[string]interface{}{
				"update_type": "group",
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				_, ok := err.(*utils.BindingError)
				assert.True(t, ok)
			},
		},
		{
			name: "invalid update_type",
			payload: map[string]interface{}{
				"update_type": "unknown",
			},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				_, ok := err.(*utils.BindingError)
				assert.True(t, ok)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			body, _ := json.Marshal(tc.payload)
			req := httptest.NewRequest(http.MethodPost, "/price/SubGroup/UpdateLatest", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			_, err := UpdateLatestPriceListSubGroup(c)
			tc.assertFunc(t, err)
		})
	}
}

func TestUpdateLatestPriceListSubGroup_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	subGroupID := uuid.New()

	// Mock GetPriceListSubGroupsByIDs to return empty result
	originalGetByIDs := getPriceListSubGroupsByIDsFunc
	getPriceListSubGroupsByIDsFunc = func([]uuid.UUID) ([]models.PriceListSubGroup, error) {
		return []models.PriceListSubGroup{}, nil
	}
	defer func() { getPriceListSubGroupsByIDsFunc = originalGetByIDs }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := map[string]interface{}{
		"subgroup_ids": []string{subGroupID.String()},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/price/SubGroup/UpdateLatest", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	_, err := UpdateLatestPriceListSubGroup(c)
	assert.Error(t, err)
	_, ok := err.(*utils.BindingError)
	assert.True(t, ok, "expected binding error")
}

func TestUpdateLatestPriceListSubGroup_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := uuid.New()
	subGroupID := uuid.New()
	now := time.Now()

	subGroup := models.PriceListSubGroup{
		ID:                  subGroupID,
		PriceListGroupID:    groupID,
		SubGroupCode:        "SUB001",
		SubgroupKey:         "SUB",
		IsTrading:           true,
		PriceUnit:           100.0,
		PriceWeight:         50.0,
		ExtraPriceUnit:      10.0,
		ExtraPriceWeight:    5.0,
		TotalNetPriceUnit:   90.0,
		TotalNetPriceWeight: 45.0,
		CreateBy:            "tester",
		CreateDtm:           &now,
		UpdateBy:            "tester",
		UpdateDtm:           &now,
		PriceListGroup: models.PriceListGroup{
			ID:          groupID,
			PriceUnit:   100.0,
			PriceWeight: 50.0,
		},
		PriceListSubGroupKeys: []models.PriceListSubGroupKey{
			{
				ID:         uuid.New(),
				SubGroupID: subGroupID,
				Code:       "CODE",
				Value:      "VALUE",
				Seq:        1,
			},
		},
	}

	// Mock GetPriceListSubGroupsByIDs
	originalGetByIDs := getPriceListSubGroupsByIDsFunc
	getPriceListSubGroupsByIDsFunc = func([]uuid.UUID) ([]models.PriceListSubGroup, error) {
		return []models.PriceListSubGroup{subGroup}, nil
	}
	defer func() { getPriceListSubGroupsByIDsFunc = originalGetByIDs }()

	// Mock GetPriceListSubGroupFormulasMapBySubGroupCodes to return empty (no formulas)
	originalGetFormulas := getPriceListSubGroupFormulasMapBySubGroupCodesFunc
	getPriceListSubGroupFormulasMapBySubGroupCodesFunc = func([]string) (map[string][]models.PriceListSubGroupFormulasMap, error) {
		return map[string][]models.PriceListSubGroupFormulasMap{}, nil
	}
	defer func() { getPriceListSubGroupFormulasMapBySubGroupCodesFunc = originalGetFormulas }()

	// Mock UpdatePriceListSubGroups
	updateCalled := false
	var updateRequest models.UpdatePriceListSubGroupRequest
	originalUpdate := updateLatestSubGroupFunc
	updateLatestSubGroupFunc = func(req models.UpdatePriceListSubGroupRequest) error {
		updateCalled = true
		updateRequest = req
		return nil
	}
	defer func() { updateLatestSubGroupFunc = originalUpdate }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := map[string]interface{}{
		// default update_type=subgroup
		"subgroup_ids": []string{subGroupID.String()},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/price/SubGroup/UpdateLatest", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	resp, err := UpdateLatestPriceListSubGroup(c)
	assert.NoError(t, err)

	// Verify success response
	result, ok := resp.(map[string]interface{})
	if assert.True(t, ok, "expected map[string]interface{}") {
		assert.True(t, result["success"].(bool))
		assert.Contains(t, result["message"].(string), "updated successfully")
	}

	// Verify update was called
	assert.True(t, updateCalled, "UpdatePriceListSubGroups should have been called")
	assert.Len(t, updateRequest.Changes, 1)
	assert.Equal(t, subGroupID, updateRequest.Changes[0].SubGroupID)
	assert.NotNil(t, updateRequest.Changes[0].TotalNetPriceUnit)
	assert.NotNil(t, updateRequest.Changes[0].TotalNetPriceWeight)
	assert.NotNil(t, updateRequest.Changes[0].ExtraPriceWeight)
}

func TestUpdateLatestPriceListSubGroup_GroupSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := uuid.New()
	subGroupID1 := uuid.New()
	subGroupID2 := uuid.New()
	now := time.Now()

	subGroup1 := models.PriceListSubGroup{
		ID:                  subGroupID1,
		PriceListGroupID:    groupID,
		SubGroupCode:        "SUBG1",
		SubgroupKey:         "KEY1",
		IsTrading:           true,
		PriceUnit:           100.0,
		PriceWeight:         50.0,
		ExtraPriceUnit:      10.0,
		ExtraPriceWeight:    5.0,
		TotalNetPriceUnit:   90.0,
		TotalNetPriceWeight: 45.0,
		CreateBy:            "tester",
		CreateDtm:           &now,
		UpdateBy:            "tester",
		UpdateDtm:           &now,
		PriceListGroup: models.PriceListGroup{
			ID:          groupID,
			GroupCode:   "GRP001",
			PriceUnit:   100.0,
			PriceWeight: 50.0,
		},
		PriceListSubGroupKeys: []models.PriceListSubGroupKey{
			{
				ID:         uuid.New(),
				SubGroupID: subGroupID1,
				Code:       "CODE1",
				Value:      "VAL1",
				Seq:        1,
			},
		},
	}

	subGroup2 := models.PriceListSubGroup{
		ID:                  subGroupID2,
		PriceListGroupID:    groupID,
		SubGroupCode:        "SUBG2",
		SubgroupKey:         "KEY2",
		IsTrading:           true,
		PriceUnit:           120.0,
		PriceWeight:         60.0,
		ExtraPriceUnit:      12.0,
		ExtraPriceWeight:    6.0,
		TotalNetPriceUnit:   110.0,
		TotalNetPriceWeight: 55.0,
		CreateBy:            "tester",
		CreateDtm:           &now,
		UpdateBy:            "tester",
		UpdateDtm:           &now,
		PriceListGroup: models.PriceListGroup{
			ID:          groupID,
			GroupCode:   "GRP001",
			PriceUnit:   120.0,
			PriceWeight: 60.0,
		},
		PriceListSubGroupKeys: []models.PriceListSubGroupKey{
			{
				ID:         uuid.New(),
				SubGroupID: subGroupID2,
				Code:       "CODE2",
				Value:      "VAL2",
				Seq:        1,
			},
		},
	}

	// Mock GetPriceListSubGroupsByGroupCodes to return two subgroups
	originalGetByGroupCodes := getPriceListSubGroupsByGroupCodesFunc
	getPriceListSubGroupsByGroupCodesFunc = func(groupCodes []string) ([]models.PriceListSubGroup, error) {
		return []models.PriceListSubGroup{subGroup1, subGroup2}, nil
	}
	defer func() { getPriceListSubGroupsByGroupCodesFunc = originalGetByGroupCodes }()

	// Mock GetPriceListSubGroupFormulasMapBySubGroupCodes to return empty (no formulas)
	originalGetFormulas := getPriceListSubGroupFormulasMapBySubGroupCodesFunc
	getPriceListSubGroupFormulasMapBySubGroupCodesFunc = func([]string) (map[string][]models.PriceListSubGroupFormulasMap, error) {
		return map[string][]models.PriceListSubGroupFormulasMap{}, nil
	}
	defer func() { getPriceListSubGroupFormulasMapBySubGroupCodesFunc = originalGetFormulas }()

	// Mock UpdatePriceListSubGroups to capture request
	updateCalled := false
	var updateRequest models.UpdatePriceListSubGroupRequest
	originalUpdate := updateLatestSubGroupFunc
	updateLatestSubGroupFunc = func(req models.UpdatePriceListSubGroupRequest) error {
		updateCalled = true
		updateRequest = req
		return nil
	}
	defer func() { updateLatestSubGroupFunc = originalUpdate }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := map[string]interface{}{
		"update_type": "group",
		"group_codes": []string{"GRP001"},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/price/SubGroup/UpdateLatest", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	resp, err := UpdateLatestPriceListSubGroup(c)
	assert.NoError(t, err)

	// Verify success response
	result, ok := resp.(map[string]interface{})
	if assert.True(t, ok, "expected map[string]interface{}") {
		assert.True(t, result["success"].(bool))
		assert.Contains(t, result["message"].(string), "updated successfully")
	}

	// Verify update was called for both subgroups
	assert.True(t, updateCalled, "UpdatePriceListSubGroups should have been called")
	assert.Len(t, updateRequest.Changes, 2)

	// Ensure both subgroup IDs are included in changes
	found1, found2 := false, false
	for _, ch := range updateRequest.Changes {
		if ch.SubGroupID == subGroupID1 {
			found1 = true
		}
		if ch.SubGroupID == subGroupID2 {
			found2 = true
		}
		// All calculated fields should be set
		assert.NotNil(t, ch.TotalNetPriceUnit)
		assert.NotNil(t, ch.TotalNetPriceWeight)
		assert.NotNil(t, ch.ExtraPriceWeight)
	}
	assert.True(t, found1, "expected subgroup 1 in changes")
	assert.True(t, found2, "expected subgroup 2 in changes")
}

func TestUpdateLatestPriceListSubGroup_WithFormulas(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := uuid.New()
	subGroupID := uuid.New()
	now := time.Now()

	subGroup := models.PriceListSubGroup{
		ID:                  subGroupID,
		PriceListGroupID:    groupID,
		SubGroupCode:        "SUB001",
		SubgroupKey:         "SUB",
		IsTrading:           true,
		PriceUnit:           100.0,
		PriceWeight:         50.0,
		ExtraPriceUnit:      10.0,
		ExtraPriceWeight:    5.0,
		TotalNetPriceUnit:   90.0,
		TotalNetPriceWeight: 45.0,
		CreateBy:            "tester",
		CreateDtm:           &now,
		UpdateBy:            "tester",
		UpdateDtm:           &now,
		PriceListGroup: models.PriceListGroup{
			ID:          groupID,
			PriceUnit:   100.0,
			PriceWeight: 50.0,
		},
		PriceListSubGroupKeys: []models.PriceListSubGroupKey{
			{
				ID:         uuid.New(),
				SubGroupID: subGroupID,
				Code:       "CODE",
				Value:      "VALUE",
				Seq:        1,
			},
		},
	}

	// Mock GetPriceListSubGroupsByIDs
	originalGetByIDs := getPriceListSubGroupsByIDsFunc
	getPriceListSubGroupsByIDsFunc = func([]uuid.UUID) ([]models.PriceListSubGroup, error) {
		return []models.PriceListSubGroup{subGroup}, nil
	}
	defer func() { getPriceListSubGroupsByIDsFunc = originalGetByIDs }()

	// Mock GetPriceListSubGroupFormulasMapBySubGroupCodes to return a formula
	originalGetFormulas := getPriceListSubGroupFormulasMapBySubGroupCodesFunc
	getPriceListSubGroupFormulasMapBySubGroupCodesFunc = func([]string) (map[string][]models.PriceListSubGroupFormulasMap, error) {
		formulaID := uuid.New()
		return map[string][]models.PriceListSubGroupFormulasMap{
			"SUB001": {
				{
					ID:                    uuid.New(),
					PriceListSubGroupCode: "SUB001",
					PriceListFormulasCode: "FORMULA001",
					IsDefault:             true,
					PriceListFormulas: models.PriceListFormulas{
						ID:          formulaID,
						FormulaCode: "FORMULA001",
						Name:        "Test Formula",
						Uom:         "kg",
						FormulaType: "calculation",
						Expression:  "base_price + extra",
						Params:      json.RawMessage(`{}`),
						Rounding:    2,
					},
				},
			},
		}, nil
	}
	defer func() { getPriceListSubGroupFormulasMapBySubGroupCodesFunc = originalGetFormulas }()

	// Mock UpdatePriceListSubGroups
	updateCalled := false
	var updateRequest models.UpdatePriceListSubGroupRequest
	originalUpdate := updateLatestSubGroupFunc
	updateLatestSubGroupFunc = func(req models.UpdatePriceListSubGroupRequest) error {
		updateCalled = true
		updateRequest = req
		return nil
	}
	defer func() { updateLatestSubGroupFunc = originalUpdate }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := map[string]interface{}{
		// default update_type=subgroup
		"subgroup_ids": []string{subGroupID.String()},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/price/SubGroup/UpdateLatest", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	resp, err := UpdateLatestPriceListSubGroup(c)
	assert.NoError(t, err)

	// Verify success response
	result, ok := resp.(map[string]interface{})
	if assert.True(t, ok, "expected map[string]interface{}") {
		assert.True(t, result["success"].(bool))
	}

	// Verify update was called with calculated values
	assert.True(t, updateCalled, "UpdatePriceListSubGroups should have been called")
	assert.Len(t, updateRequest.Changes, 1)
	assert.Equal(t, subGroupID, updateRequest.Changes[0].SubGroupID)
	assert.NotNil(t, updateRequest.Changes[0].TotalNetPriceUnit)
	assert.NotNil(t, updateRequest.Changes[0].TotalNetPriceWeight)
	assert.NotNil(t, updateRequest.Changes[0].ExtraPriceWeight)

	// Verify calculated values (base_price + extra = 50.0 + 5.0 = 55.0)
	assert.Equal(t, 55.0, *updateRequest.Changes[0].TotalNetPriceWeight)
}

func TestUpdateLatestPriceListSubGroup_GroupCodesNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Mock GetPriceListSubGroupsByGroupCodes to return no subgroups
	originalGetByGroupCodes := getPriceListSubGroupsByGroupCodesFunc
	getPriceListSubGroupsByGroupCodesFunc = func(groupCodes []string) ([]models.PriceListSubGroup, error) {
		return []models.PriceListSubGroup{}, nil
	}
	defer func() { getPriceListSubGroupsByGroupCodesFunc = originalGetByGroupCodes }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := map[string]interface{}{
		"update_type": "group",
		"group_codes": []string{"GRP_UNKNOWN"},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/price/SubGroup/UpdateLatest", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	_, err := UpdateLatestPriceListSubGroup(c)
	if assert.Error(t, err) {
		bindingErr, ok := err.(*utils.BindingError)
		assert.True(t, ok, "expected BindingError")
		assert.Contains(t, bindingErr.Message, "No price list sub groups found for the provided group_codes")
	}
}

func TestUpdateLatestPriceListSubGroup_PriceListGroupExtraKeyLoaded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := uuid.New()
	subGroupID := uuid.New()
	extraID := uuid.New()
	extraKeyID := uuid.New()
	now := time.Now()

	subGroup := models.PriceListSubGroup{
		ID:                  subGroupID,
		PriceListGroupID:    groupID,
		SubGroupCode:        "SUB001",
		SubgroupKey:         "SUB",
		IsTrading:           true,
		PriceUnit:           100.0,
		PriceWeight:         50.0,
		ExtraPriceUnit:      10.0,
		ExtraPriceWeight:    5.0,
		TotalNetPriceUnit:   90.0,
		TotalNetPriceWeight: 45.0,
		CreateBy:            "tester",
		CreateDtm:           &now,
		UpdateBy:            "tester",
		UpdateDtm:           &now,
		PriceListGroup: models.PriceListGroup{
			ID:          groupID,
			PriceUnit:   100.0,
			PriceWeight: 50.0,
			PriceListGroupExtras: []models.PriceListGroupExtra{
				{
					ID:               extraID,
					PriceListGroupID: groupID,
					ExtraKey:         "EXTRA001",
					PriceListGroupExtraKeys: []models.PriceListGroupExtraKey{
						{
							ID:           extraKeyID,
							GroupExtraID: extraID,
							Code:         "KEY_CODE",
							Value:        "KEY_VALUE",
							Seq:          1,
						},
					},
				},
			},
		},
		PriceListSubGroupKeys: []models.PriceListSubGroupKey{},
	}

	// Mock GetPriceListSubGroupsByIDs
	originalGetByIDs := getPriceListSubGroupsByIDsFunc
	getPriceListSubGroupsByIDsFunc = func([]uuid.UUID) ([]models.PriceListSubGroup, error) {
		return []models.PriceListSubGroup{subGroup}, nil
	}
	defer func() { getPriceListSubGroupsByIDsFunc = originalGetByIDs }()

	// Mock GetPriceListSubGroupFormulasMapBySubGroupCodes
	originalGetFormulas := getPriceListSubGroupFormulasMapBySubGroupCodesFunc
	getPriceListSubGroupFormulasMapBySubGroupCodesFunc = func([]string) (map[string][]models.PriceListSubGroupFormulasMap, error) {
		return map[string][]models.PriceListSubGroupFormulasMap{}, nil
	}
	defer func() { getPriceListSubGroupFormulasMapBySubGroupCodesFunc = originalGetFormulas }()

	// Mock UpdatePriceListSubGroups
	originalUpdate := updateLatestSubGroupFunc
	updateLatestSubGroupFunc = func(req models.UpdatePriceListSubGroupRequest) error {
		return nil
	}
	defer func() { updateLatestSubGroupFunc = originalUpdate }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := map[string]interface{}{
		// default update_type=subgroup
		"subgroup_ids": []string{subGroupID.String()},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/price/SubGroup/UpdateLatest", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Verify that PriceListGroupExtraKey is accessible (loaded via preload)
	subGroups, _ := getPriceListSubGroupsByIDsFunc([]uuid.UUID{subGroupID})
	if len(subGroups) > 0 && len(subGroups[0].PriceListGroup.PriceListGroupExtras) > 0 {
		extraKeys := subGroups[0].PriceListGroup.PriceListGroupExtras[0].PriceListGroupExtraKeys
		assert.Len(t, extraKeys, 1, "PriceListGroupExtraKey should be loaded")
		assert.Equal(t, "KEY_CODE", extraKeys[0].Code)
		assert.Equal(t, "KEY_VALUE", extraKeys[0].Value)
	}

	resp, err := UpdateLatestPriceListSubGroup(c)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestUpdateLatestPriceListSubGroup_WithInventoryData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := uuid.New()
	subGroupID := uuid.New()
	now := time.Now()

	subGroup := models.PriceListSubGroup{
		ID:                  subGroupID,
		PriceListGroupID:    groupID,
		SubGroupCode:        "SUB001",
		SubgroupKey:         "SUB",
		IsTrading:           true,
		PriceUnit:           100.0,
		PriceWeight:         50.0,
		ExtraPriceUnit:      10.0,
		ExtraPriceWeight:    5.0,
		TotalNetPriceUnit:   90.0,
		TotalNetPriceWeight: 45.0,
		CreateBy:            "tester",
		CreateDtm:           &now,
		UpdateBy:            "tester",
		UpdateDtm:           &now,
		PriceListGroup: models.PriceListGroup{
			ID:          groupID,
			CompanyCode: "C001",
			SiteCode:    "S001",
			PriceUnit:   100.0,
			PriceWeight: 50.0,
		},
		PriceListSubGroupKeys: []models.PriceListSubGroupKey{
			{
				ID:         uuid.New(),
				SubGroupID: subGroupID,
				Code:       "CODE",
				Value:      "VALUE",
				Seq:        1,
			},
		},
	}

	// Mock GetPriceListSubGroupsByIDs
	originalGetByIDs := getPriceListSubGroupsByIDsFunc
	getPriceListSubGroupsByIDsFunc = func([]uuid.UUID) ([]models.PriceListSubGroup, error) {
		return []models.PriceListSubGroup{subGroup}, nil
	}
	defer func() { getPriceListSubGroupsByIDsFunc = originalGetByIDs }()

	// Mock GetPriceListSubGroupFormulasMapBySubGroupCodes to return a formula that uses avg_kg_stock
	originalGetFormulas := getPriceListSubGroupFormulasMapBySubGroupCodesFunc
	getPriceListSubGroupFormulasMapBySubGroupCodesFunc = func([]string) (map[string][]models.PriceListSubGroupFormulasMap, error) {
		formulaID := uuid.New()
		return map[string][]models.PriceListSubGroupFormulasMap{
			"SUB001": {
				{
					ID:                    uuid.New(),
					PriceListSubGroupCode: "SUB001",
					PriceListFormulasCode: "FORMULA001",
					IsDefault:             false,
					PriceListFormulas: models.PriceListFormulas{
						ID:          formulaID,
						FormulaCode: "FORMULA001",
						Name:        "Test Formula with Inventory",
						Uom:         "kg",
						FormulaType: "calculation",
						// Formula uses avg_kg_stock: base_price + extra + avg_kg_stock
						Expression: "base_price + extra + avg_kg_stock",
						Params:     json.RawMessage(`{}`),
						Rounding:   2,
					},
				},
			},
		}, nil
	}
	defer func() { getPriceListSubGroupFormulasMapBySubGroupCodesFunc = originalGetFormulas }()

	// Mock UpdatePriceListSubGroups
	updateCalled := false
	var updateRequest models.UpdatePriceListSubGroupRequest
	originalUpdate := updateLatestSubGroupFunc
	updateLatestSubGroupFunc = func(req models.UpdatePriceListSubGroupRequest) error {
		updateCalled = true
		updateRequest = req
		return nil
	}
	defer func() { updateLatestSubGroupFunc = originalUpdate }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := map[string]interface{}{
		"subgroup_ids": []string{subGroupID.String()},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/price/SubGroup/UpdateLatest", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	resp, err := UpdateLatestPriceListSubGroup(c)
	assert.NoError(t, err)

	// Verify success response
	result, ok := resp.(map[string]interface{})
	if assert.True(t, ok, "expected map[string]interface{}") {
		assert.True(t, result["success"].(bool))
	}

	// Verify update was called with calculated values
	assert.True(t, updateCalled, "UpdatePriceListSubGroups should have been called")
	assert.Len(t, updateRequest.Changes, 1)
	assert.Equal(t, subGroupID, updateRequest.Changes[0].SubGroupID)
	assert.NotNil(t, updateRequest.Changes[0].TotalNetPriceWeight)

	// Note: In this test, the inventory service call will fail (no mock), so avg_kg_stock=0
	// Formula: base_price + extra + avg_kg_stock = 50.0 + 5.0 + 0.0 = 55.0
	assert.Equal(t, 55.0, *updateRequest.Changes[0].TotalNetPriceWeight)
}
