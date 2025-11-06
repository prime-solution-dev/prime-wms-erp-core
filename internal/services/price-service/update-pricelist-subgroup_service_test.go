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
)

func TestUpdatePriceListSubGroup_Validation_MissingID(t *testing.T) {
    gin.SetMode(gin.TestMode)

    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)

    payload := map[string]interface{}{
        "price_unit": 100.0,
    }
    body, _ := json.Marshal(payload)
    req := httptest.NewRequest(http.MethodPost, "/price/UpdatePriceListSubGroup", bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")
    c.Request = req

    _, err := UpdatePriceListSubGroup(c)
    if err == nil {
        t.Fatalf("expected validation error, got nil")
    }
    if _, ok := err.(*utils.BindingError); !ok {
        t.Fatalf("expected *utils.BindingError, got %T", err)
    }
}

func TestUpdatePriceListSubGroup_Validation_NegativePrice(t *testing.T) {
    gin.SetMode(gin.TestMode)

    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)

    payload := map[string]interface{}{
        "id":         uuid.New().String(),
        "price_unit": -5,
    }
    body, _ := json.Marshal(payload)
    req := httptest.NewRequest(http.MethodPost, "/price/UpdatePriceListSubGroup", bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")
    c.Request = req

    _, err := UpdatePriceListSubGroup(c)
    if err == nil {
        t.Fatalf("expected validation error for negative price, got nil")
    }
    if _, ok := err.(*utils.BindingError); !ok {
        t.Fatalf("expected *utils.BindingError, got %T", err)
    }
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
        "id":         id.String(),
        "price_unit": 123.45,
        "udf_json":   map[string]interface{}{"highlight": true},
    }
    body, _ := json.Marshal(payload)
    req := httptest.NewRequest(http.MethodPost, "/price/UpdatePriceListSubGroup", bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")
    c.Request = req

    resp, err := UpdatePriceListSubGroup(c)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !called {
        t.Fatalf("repository function was not called")
    }
    if captured.ID != id {
        t.Fatalf("expected ID %v, got %v", id, captured.ID)
    }
    if captured.PriceUnit == nil || *captured.PriceUnit != 123.45 {
        t.Fatalf("expected price_unit 123.45, got %v", captured.PriceUnit)
    }

    // verify response shape
    m, ok := resp.(map[string]interface{})
    if !ok {
        t.Fatalf("expected map response, got %T", resp)
    }
    if m["success"] != true {
        t.Fatalf("expected success=true, got %v", m["success"])
    }
}


