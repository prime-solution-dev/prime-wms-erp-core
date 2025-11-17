package config

import "os"

// GetBaseURL returns the base URL from environment variable with fallback
func GetBaseURL() string {
	if baseURL := os.Getenv("base_url"); baseURL != "" {
		return baseURL
	}
	return ""
}

var (
	GET_INVENTORY_ATP_ENDPOINT string
	CREATE_ORDER_ENDPOINT      string
	CANCEL_ORDER_ENDPOINT      string
)

func Initialize() {
	GET_INVENTORY_ATP_ENDPOINT = GetBaseURL() + "/warehouse/Inventory/GetInventoryAtp"
	CREATE_ORDER_ENDPOINT = GetBaseURL() + "/order/Order/CreateOrders"
	CANCEL_ORDER_ENDPOINT = GetBaseURL() + "/order/Order/CancelOrders"
}

// HTTP Configuration
const (
	CONTENT_TYPE_JSON = "application/json"
)
