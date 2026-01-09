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
)

type UpdateOrderByDeliveryRequest struct {
	DocumentRef      string                            `json:"document_ref"`
	DeliveryMethod   string                            `json:"delivery_method"`
	BookingDate      *time.Time                        `json:"booking_date"`
	DeliveryTimeCode string                            `json:"delivery_time_code"`
	Tel              string                            `json:"tel"`
	LicensePlate     string                            `json:"license_plate"`
	ContactName      string                            `json:"contact_name"`
	Remark           string                            `json:"remark"`
	OrderItem        []UpdateOrderByDeliveryItemDetail `json:"order_item"`
}

type UpdateOrderByDeliveryItemDetail struct {
	OrderItem            string     `json:"order_item"`
	DocumentRefItem      string     `json:"document_ref_item"`
	ProductCode          string     `json:"product_code"`
	ProductType          string     `json:"product_type"`
	InterfaceOrderQty    float64    `json:"interface_order_qty"`
	Qty                  float64    `json:"qty"`
	UnitCode             string     `json:"unit_code"`
	IsFocGwp             bool       `gorm:"type:boolean" json:"is_foc_gwp"`
	WarehouseCode        string     `json:"warehouse_code"`
	BatchNo              string     `json:"batch_no"`
	SerialCode           string     `json:"serial_code"`
	SaleUnitCode         string     `json:"sale_unit_code"`
	SaleMethod           string     `json:"sale_method"`
	InterfaceOrderWeight float64    `json:"interface_order_weight"`
	Weight               float64    `json:"weight"`
	WeightUnit           float64    `json:"weight_unit"`
	MfgDate              *time.Time `json:"mfg_date"`
	ExpDate              *time.Time `json:"exp_date"`
	LocationCode         string     `json:"location_code"`
	StorageType          string     `json:"storage_type"`
	Remark               string     `json:"remark"`
	Status               string     `json:"status"`
}

type UpdateOrderByDeliveryResponse struct {
	Status            string `json:"status"`
	Message           string `json:"message"`
	OrderCode         string `json:"order_code"`
	DocumentRef       string `json:"document_ref"`
	OrderItemsCreated int    `json:"order_items_created"`
}

func UpdateOrderByDelivery(jsonPayload UpdateOrderByDeliveryRequest) (UpdateOrderByDeliveryResponse, error) {

	jsonData, err := json.Marshal(jsonPayload)
	if err != nil {
		return UpdateOrderByDeliveryResponse{}, errors.New("Error marshaling struct to JSON:" + err.Error())
	}
	req, err := http.NewRequest("POST", config.UPDATE_ORDER_BY_DELIVERY_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		return UpdateOrderByDeliveryResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return UpdateOrderByDeliveryResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	var dataRes UpdateOrderByDeliveryResponse
	err = json.Unmarshal(body, &dataRes)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	return dataRes, nil
}
