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
	ProductCode   string   `json:"product_code"`
	TodayStockQty float64  `json:"today_stock_qty"`
	TodayAtpQty   float64  `json:"today_atp_qty"`
	TotalAtpQty   float64  `json:"total_atp_qty"`
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
		return GetInventoryAtpResponse{}, errors.New("Error marshaling struct to JSON: " + err.Error())
	}

	req, err := http.NewRequest("POST", config.GET_INVENTORY_ATP_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		return GetInventoryAtpResponse{}, errors.New("Error creating request: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	// timeout กันปลายทางค้างแล้วลาก request ของเราค้างตาม (default ของ http.Client คือไม่มี timeout)
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return GetInventoryAtpResponse{}, errors.New("Error sending request: " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GetInventoryAtpResponse{}, errors.New("Error reading response body: " + err.Error())
	}

	// เดิมข้ามการเช็ค status ทำให้ปลายทางตอบ 500 แล้วเราคืน struct เปล่ากับ error = nil
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			Error string `json:"error"`
		}
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Error != "" {
			return GetInventoryAtpResponse{}, errors.New(errResp.Error)
		}
		return GetInventoryAtpResponse{}, fmt.Errorf("API error status %d: %s", resp.StatusCode, string(body))
	}

	var dataRes GetInventoryAtpResponse
	if err = json.Unmarshal(body, &dataRes); err != nil {
		return GetInventoryAtpResponse{}, errors.New("Error parsing response: " + err.Error())
	}

	return dataRes, nil
}
