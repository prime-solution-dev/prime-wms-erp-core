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

func TestGetLatestPriceListSubGroup_ValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		payload    interface{}
		assertFunc func(*testing.T, error)
	}{
		{
			name:    "missing subgroup id",
			payload: map[string]interface{}{},
			assertFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				_, ok := err.(*utils.BindingError)
				assert.True(t, ok)
			},
		},
		{
			name: "invalid uuid",
			payload: map[string]interface{}{
				"subgroup_id": "not-a-uuid",
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			body, _ := json.Marshal(tc.payload)
			req := httptest.NewRequest(http.MethodPost, "/price/SubGroup/Latest", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			_, err := GetLatestPriceListSubGroup(c)
			tc.assertFunc(t, err)
		})
	}
}

func TestGetLatestPriceListSubGroup_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	original := getLatestSubGroupFunc
	getLatestSubGroupFunc = func(uuid.UUID) (*models.PriceListSubGroup, error) {
		return nil, nil
	}
	defer func() { getLatestSubGroupFunc = original }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := map[string]interface{}{
		"subgroup_id": uuid.New().String(),
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/price/SubGroup/Latest", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	_, err := GetLatestPriceListSubGroup(c)
	assert.Error(t, err)
	_, ok := err.(*utils.BindingError)
	assert.True(t, ok, "expected binding error")
}

func TestGetLatestPriceListSubGroup_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := uuid.New()
	subGroupID := uuid.New()
	now := time.Now()

	expected := &models.PriceListSubGroup{
		ID:               subGroupID,
		PriceListGroupID: groupID,
		SubgroupKey:      "SUB",
		IsTrading:        true,
		PriceUnit:        100.0,
		CreateBy:         "tester",
		CreateDtm:        &now,
		UpdateBy:         "tester",
		UpdateDtm:        &now,
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

	original := getLatestSubGroupFunc
	getLatestSubGroupFunc = func(uuid.UUID) (*models.PriceListSubGroup, error) {
		return expected, nil
	}
	defer func() { getLatestSubGroupFunc = original }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := map[string]interface{}{
		"subgroup_id": subGroupID.String(),
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/price/SubGroup/Latest", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	resp, err := GetLatestPriceListSubGroup(c)
	assert.NoError(t, err)

	result, ok := resp.(*models.PriceListSubGroup)
	if assert.True(t, ok, "expected *models.PriceListSubGroup") {
		assert.Equal(t, expected.ID, result.ID)
		assert.Equal(t, expected.SubgroupKey, result.SubgroupKey)
		assert.Equal(t, expected.PriceUnit, result.PriceUnit)
		assert.Len(t, result.PriceListSubGroupKeys, 1)
		assert.Equal(t, expected.PriceListSubGroupKeys[0].Code, result.PriceListSubGroupKeys[0].Code)
	}
}
