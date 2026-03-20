package externalService

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"prime-erp-core/config"
	"prime-erp-core/internal/models"
)

// InventoryByProductCodeRequest represents the request structure for inventory service
type InventoryByProductCodeRequest struct {
	CompanyCode []string                         `json:"company_code"`
	SiteCode    []string                         `json:"site_code"`
	KeyValue    []InventoryByProductCodeKeyValue `json:"key_value"`
}

// InventoryByProductCodeKeyValue represents a key-value pair in the inventory service request
type InventoryByProductCodeKeyValue struct {
	ID         string `json:"id"`
	GroupCode  string `json:"group_code"`
	GroupValue string `json:"group_value"`
	Seq        int    `json:"seq"`
}

// InventoryByProductCodeResponse represents the response structure from inventory service
type InventoryByProductCodeResponse struct {
	ID              string                           `json:"id"`
	GroupCodeKeys   string                           `json:"group_code_keys"`
	GroupValueKeys  string                           `json:"group_value_keys"`
	ProductCode     string                           `json:"product_code"`
	SupplierCode    string                           `json:"supplier_code"`
	SupplierName    string                           `json:"supplier_name"`
	InventoryWeight []models.InventoryWeightResponse `json:"inventory_weight"`
}

// GetInventoryWeightByKey calls the external inventory service to get inventory weight data
func GetInventoryWeightByKey(companyCode string, siteCodes []string, keyValues []InventoryByProductCodeKeyValue) ([]InventoryByProductCodeResponse, error) {
	// Build request body
	reqBody := InventoryByProductCodeRequest{
		CompanyCode: []string{companyCode},
		SiteCode:    siteCodes,
		KeyValue:    keyValues,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal inventory request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", config.GET_INVENTORY_BY_KEY_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("inventory service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var inventoryResponse []InventoryByProductCodeResponse
	if err := json.Unmarshal(body, &inventoryResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal inventory response: %w", err)
	}

	return inventoryResponse, nil
}
