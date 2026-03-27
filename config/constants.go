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
	GET_INVENTORY_ATP_ENDPOINT             string
	GET_INVENTORY_BY_PRODUCT_CODE_ENDPOINT string
	CREATE_ORDER_ENDPOINT                  string
	UPDATE_ORDER_BY_DELIVERY_ENDPOINT      string
	CANCEL_ORDER_ENDPOINT                  string
	GET_PACK_SO_ENDPOINT                   string
	GET_INBOUND_ENDPOINT                   string
	GET_GOODS_RECEIVE_ENDPOINT             string
	GET_ORDER_DELIVERY_ENDPOINT            string
	GET_CUSTOMER_MASTER_ENDPOINT           string
	GET_SYSTEM_CONFIG_WAREHOUSE_ENDPOINT   string
	GET_INVENTORY_BY_KEY_ENDPOINT          string
	GET_PRODUCT_ENDPOINT                   string
)

// Initialize sets the endpoint values (call after loading .env)
func Initialize() {
	GET_INVENTORY_ATP_ENDPOINT = GetBaseURL() + "/warehouse/Inventory/GetInventoryAtp"
	GET_INVENTORY_BY_KEY_ENDPOINT = GetBaseURL() + "/warehouse/Inventory/get-inventory-weight-by-key"
	CREATE_ORDER_ENDPOINT = GetBaseURL() + "/order/Order/CreateOrders"
	UPDATE_ORDER_BY_DELIVERY_ENDPOINT = GetBaseURL() + "/order/Order/UpdateOrderByDelivery"
	CANCEL_ORDER_ENDPOINT = GetBaseURL() + "/order/Order/CancelOrders"
	GET_PACK_SO_ENDPOINT = GetBaseURL() + "/packing/packing/get-packing-so"
	GET_INBOUND_ENDPOINT = GetBaseURL() + "/goods-receive/get-inbounds"
	GET_GOODS_RECEIVE_ENDPOINT = GetBaseURL() + "/goods-receive/get-goods-recieves"
	GET_ORDER_DELIVERY_ENDPOINT = GetBaseURL() + "/order/Order/GetOrdersDelivery"
	GET_CUSTOMER_MASTER_ENDPOINT = GetBaseURL() + "/customer/Customer/GetCustomers"
	GET_SYSTEM_CONFIG_WAREHOUSE_ENDPOINT = GetBaseURL() + "/warehouse/Warehouse/get-system-config"
	GET_PRODUCT_ENDPOINT = GetBaseURL() + "/product/Product/GetProduct"
}

// HTTP Configuration
const (
	CONTENT_TYPE_JSON = "application/json"
)
