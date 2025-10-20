package externalService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"prime-erp-core/config"
	"time"

	"github.com/google/uuid"
)

type CreateOrderRequest struct {
	Orders []CreateOrderDetail `json:"orders"`
}

type CreateOrderDetail struct {
	Action              string                  `json:"action"`
	OrderID             uuid.UUID               `json:"order_id"`
	OrderCode           string                  `json:"order_code"`
	OrderType           string                  `json:"order_type"`
	OrderDate           time.Time               `json:"order_date"`
	TenantID            *uuid.UUID              `json:"tenant_id"`
	CustomerCode        string                  `json:"customer_code"`
	SoldToCode          string                  `json:"sold_to_code"`
	ShipToCode          string                  `json:"ship_to_code"`
	BillToCode          string                  `json:"bill_to_code"`
	TransportZone       string                  `json:"transport_zone"`
	InterfaceQty        float64                 `json:"interface_qty"`
	InterfaceUnitCode   string                  `json:"interface_unit_code"`
	Qty                 float64                 `json:"qty"`
	UnitCode            string                  `json:"unit_code"`
	EstimatePickingDate *time.Time              `json:"estimate_picking_date"`
	DeliveryDate        *time.Time              `json:"delivery_date"`
	SubmitDate          *time.Time              `json:"submit_date"`
	Status              string                  `json:"status"`
	DocumentRefType     string                  `json:"document_ref_type"`
	DocumentRef         string                  `json:"document_ref"`
	Remark              string                  `json:"remark"`
	OrderItem           []CreateOrderItemDetail `json:"order_item"`
}

type CreateOrderItemDetail struct {
	OrderItem         string  `json:"order_item"`
	DocumentRefItem   string  `json:"document_ref_item"`
	ProductCode       string  `json:"product_code"`
	ProductType       string  `json:"product_type"`
	InterfaceOrderQty float64 `json:"interface_order_qty"`
	Qty               float64 `json:"qty"`
	UnitCode          string  `json:"unit_code"`
	IsFocGwp          bool    `gorm:"type:boolean" json:"is_foc_gwp"`
	WarehouseCode     string  `json:"warehouse_code"`
	BatchNo           string  `json:"batch_no"`
	SerialCode        string  `json:"serial_code"`
	SaleUnitCode      string  `json:"sale_unit_code"`
	SaleMethod        string  `json:"sale_method"`
	Weight            float64 `json:"weight"`
	WeightUnit        float64 `json:"weight_unit"`
	Remark            string  `json:"remark"`
	Status            string  `json:"status"`
}

type CreateOrderResponse struct {
	Status    string   `json:"status"`
	Message   string   `json:"message"`
	OrderCode []string `json:"order_code"`
}

func CreateOrder(jsonPayload CreateOrderRequest) (CreateOrderResponse, error) {

	jsonData, err := json.Marshal(jsonPayload)
	if err != nil {
		return CreateOrderResponse{}, errors.New("Error marshaling struct to JSON:" + err.Error())
	}
	req, err := http.NewRequest("POST", config.CREATE_ORDER_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		return CreateOrderResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return CreateOrderResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	var dataRes CreateOrderResponse
	err = json.Unmarshal(body, &dataRes)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	return dataRes, nil
}
