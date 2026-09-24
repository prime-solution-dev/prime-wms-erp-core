package priceService

import (
	"context"
	"encoding/json"
	"testing"

	"prime-erp-core/internal/models"
	"prime-erp-core/internal/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUpdatePriceListSubGroup_Validation_MissingID(t *testing.T) {
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

	_, err := UpdatePriceListSubGroup(context.Background(), string(body))
	assert.Error(t, err)
	_, ok := err.(*utils.BindingError)
	assert.True(t, ok, "expected *utils.BindingError, got %T", err)
}

func TestUpdatePriceListSubGroup_Validation_NegativePrice(t *testing.T) {
	payload := map[string]interface{}{
		"changes": []map[string]interface{}{{
			"subgroup_id": uuid.New().String(),
			"price_unit":  -5,
		}},
	}
	body, _ := json.Marshal(payload)

	// Ensure repository is NOT called if validation fails
	original := updateSubGroupFunc
	updateSubGroupFunc = func(req models.UpdatePriceListSubGroupRequest) error {
		t.Fatalf("repository should not be called on validation error")
		return nil
	}
	defer func() { updateSubGroupFunc = original }()

	_, err := UpdatePriceListSubGroup(context.Background(), string(body))
	assert.Error(t, err)
	_, ok := err.(*utils.BindingError)
	assert.True(t, ok, "expected *utils.BindingError, got %T", err)
}

func TestUpdatePriceListSubGroup_Success_CallsRepository(t *testing.T) {
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

	id := uuid.New()
	payload := map[string]interface{}{
		"changes": []map[string]interface{}{{
			"subgroup_id": id.String(),
			"price_unit":  123.45,
			"udf_json":    map[string]interface{}{"is_highlight": true},
		}},
	}
	body, _ := json.Marshal(payload)

	resp, err := UpdatePriceListSubGroup(context.Background(), string(body))
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
