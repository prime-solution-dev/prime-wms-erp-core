package config

// API Configuration Constants
const (
	// WMS_API_BASE_URL is the base URL for WMS API endpoints
	WMS_API_BASE_URL = "https://wms-uat.prime-lab.cc/api"

	// API Endpoints
	GET_INVENTORY_ATP_ENDPOINT = WMS_API_BASE_URL + "/warehouse/Inventory/GetInventoryAtp"
	CREATE_ORDER_ENDPOINT      = WMS_API_BASE_URL + "/order/Order/CreateOrders"
	CANCEL_ORDER_ENDPOINT      = WMS_API_BASE_URL + "/order/Order/CancelOrders"
	GET_PACK_SO_ENDPOINT       = WMS_API_BASE_URL + "/packing/packing/get-packing-so"
)

// HTTP Configuration
const (
	CONTENT_TYPE_JSON = "application/json"
)
