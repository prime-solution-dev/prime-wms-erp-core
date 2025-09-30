package externalService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"prime-erp-core/config"
)

type GetInventoryAtpRequest struct {
	CompanyCodes   []string   `json:"company_codes"`
	SiteCodes      []string   `json:"site_codes"`
	WarehouseCodes []string   `json:"warehouse_codes"`
	ProductCodes   []string   `json:"product_codes"`
	StorageTypes   []string   `json:"storage_types"`
	ToDate         *time.Time `json:"to_date"`
}

type GetInventoryAtpResponse struct {
	ProductAtps []productAtp `json:"product_atps"`
}

type productAtp struct {
	CompanyCode   string   `json:"company_code"`
	SiteCode      string   `json:"site_code"`
	WarehouseCode string   `json:"warehouse_code"`
	ProductCode   string   `json:"product_code"`
	TodayStockQty int      `json:"current_stock_qty"`
	TodayAtpQty   int      `json:"current_atp_qty"`
	TotalAtpQty   int      `json:"atp_qty"`
	DayAtps       []dayAtp `json:"day_atps"`
}

type dayAtp struct {
	Date         time.Time     `json:"date"`
	AtpQty       int           `json:"atp_qty"`
	DocumentAtps []documentAtp `json:"document_atps"`
}

type documentAtp struct {
	Seq             int       `json:"seq"`
	Date            time.Time `json:"date"`
	DocumentType    string    `json:"document_type"`     //e.g., SO, PO, KITTING
	DocumentSubType string    `json:"document_sub_type"` //e.g., KITTING-IN, KITTING-OUT
	DocumentCode    string    `json:"document_code"`
	DocumentDate    time.Time `json:"document_date"`
	Qty             int       `json:"qty"`
	FinishedQty     int       `json:"finished_qty"`
	RemainQty       int       `json:"remain_qty"`
	BalanceQty      int       `json:"balance_qty"`
}

func GetInventoryATP(jsonPayload GetInventoryAtpRequest) (GetInventoryAtpResponse, error) {

	jsonData, err := json.Marshal(jsonPayload)
	if err != nil {
		return GetInventoryAtpResponse{}, errors.New("Error marshaling struct to JSON:" + err.Error())
	}
	req, err := http.NewRequest("POST", config.GET_INVENTORY_ATP_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		return GetInventoryAtpResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return GetInventoryAtpResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	var dataRes GetInventoryAtpResponse
	err = json.Unmarshal(body, &dataRes)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	return dataRes, nil
}
