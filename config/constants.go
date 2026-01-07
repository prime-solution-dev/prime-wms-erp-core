package config

import "os"

// GetBaseURL returns the base URL from environment variable with fallback
func GetBaseURL() string {
	if baseURL := os.Getenv("base_url"); baseURL != "" {
		return baseURL
	}
	return ""
}

// API Endpoint variables
var (
	GET_INVENTORY_ATP_ENDPOINT        string
	CREATE_ORDER_ENDPOINT             string
	UPDATE_ORDER_BY_DELIVERY_ENDPOINT string
	CANCEL_ORDER_ENDPOINT             string
	GET_PACK_SO_ENDPOINT              string
)

// Initialize sets the endpoint values (call after loading .env)
func Initialize() {
	GET_INVENTORY_ATP_ENDPOINT = GetBaseURL() + "/warehouse/Inventory/GetInventoryAtp"
	CREATE_ORDER_ENDPOINT = GetBaseURL() + "/order/Order/CreateOrders"
	UPDATE_ORDER_BY_DELIVERY_ENDPOINT = GetBaseURL() + "/order/Order/UpdateOrderByDelivery"
	CANCEL_ORDER_ENDPOINT = GetBaseURL() + "/order/Order/CancelOrders"
	GET_PACK_SO_ENDPOINT = GetBaseURL() + "/packing/packing/get-packing-so"
}

// HTTP Configuration
const (
	CONTENT_TYPE_JSON = "application/json"
)
