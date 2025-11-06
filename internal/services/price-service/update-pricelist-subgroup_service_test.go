package priceService

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prime-erp-core/internal/models"
	"prime-erp-core/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUpdatePriceListSubGroup_Validation_MissingID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Ensure repository is NOT called if validation fails
	original := updateSubGroupFunc
	updateSubGroupFunc = func(req models.UpdatePriceListSubGroupRequest) error {
		t.Fatalf("repository should not be called on validation error")
		return nil
	}
	defer func() { updateSubGroupFunc = original }()

	payload := map[string]interface{}{
		"changes": []map[string]interface{}{{
			"price_unit": 100.0,
		}},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/price/UpdatePriceListSubGroup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	_, err := UpdatePriceListSubGroup(c)
	assert.Error(t, err)
	_, ok := err.(*utils.BindingError)
	assert.True(t, ok, "expected *utils.BindingError, got %T", err)
}

func TestUpdatePriceListSubGroup_Validation_NegativePrice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := map[string]interface{}{
		"changes": []map[string]interface{}{{
			"subgroup_id": uuid.New().String(),
			"price_unit":  -5,
		}},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/price/UpdatePriceListSubGroup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Ensure repository is NOT called if validation fails
	original := updateSubGroupFunc
	updateSubGroupFunc = func(req models.UpdatePriceListSubGroupRequest) error {
		t.Fatalf("repository should not be called on validation error")
		return nil
	}
	defer func() { updateSubGroupFunc = original }()

	_, err := UpdatePriceListSubGroup(c)
	assert.Error(t, err)
	_, ok := err.(*utils.BindingError)
	assert.True(t, ok, "expected *utils.BindingError, got %T", err)
}

func TestUpdatePriceListSubGroup_Success_CallsRepository(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// stub repo function
	called := false
	var captured models.UpdatePriceListSubGroupRequest
	original := updateSubGroupFunc
	updateSubGroupFunc = func(req models.UpdatePriceListSubGroupRequest) error {
		called = true
		captured = req
		return nil
	}
	defer func() { updateSubGroupFunc = original }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	id := uuid.New()
	payload := map[string]interface{}{
		"changes": []map[string]interface{}{{
			"subgroup_id": id.String(),
			"price_unit":  123.45,
			"udf_json":    map[string]interface{}{"highlight": true},
		}},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/price/UpdatePriceListSubGroup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	resp, err := UpdatePriceListSubGroup(c)
	assert.NoError(t, err)
	assert.True(t, called, "repository function was not called")
	if assert.Len(t, captured.Changes, 1) {
		assert.Equal(t, id, captured.Changes[0].SubGroupID)
		if assert.NotNil(t, captured.Changes[0].PriceUnit) {
			assert.Equal(t, 123.45, *captured.Changes[0].PriceUnit)
		}
	}

	// verify response shape
	m, ok := resp.(map[string]interface{})
	assert.True(t, ok, "expected map response, got %T", resp)
	assert.Equal(t, true, m["success"])
}
