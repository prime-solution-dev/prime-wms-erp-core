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

type CancelOrderRequest struct {
	OrderID     []uuid.UUID `json:"order_id"`
	DocumentRef []string    `json:"document_ref"`
}

type CancelOrderResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func CancelOrder(jsonPayload CancelOrderRequest) (CancelOrderResponse, error) {

	jsonData, err := json.Marshal(jsonPayload)
	if err != nil {
		return CancelOrderResponse{}, errors.New("Error marshaling struct to JSON: " + err.Error())
	}

	req, err := http.NewRequest("POST", config.CANCEL_ORDER_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		return CancelOrderResponse{}, errors.New("Error creating request: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	// timeout กันปลายทางค้างแล้วลาก request ของเราค้างตาม (default ของ http.Client คือไม่มี timeout)
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return CancelOrderResponse{}, errors.New("Error sending request: " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return CancelOrderResponse{}, errors.New("Error reading response body: " + err.Error())
	}

	// เดิมข้ามการเช็ค status ทำให้ปลายทางตอบ 500 แล้วเราคืน struct เปล่ากับ error = nil
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			Error string `json:"error"`
		}
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Error != "" {
			return CancelOrderResponse{}, errors.New(errResp.Error)
		}
		return CancelOrderResponse{}, fmt.Errorf("API error status %d: %s", resp.StatusCode, string(body))
	}

	var dataRes CancelOrderResponse
	if err = json.Unmarshal(body, &dataRes); err != nil {
		return CancelOrderResponse{}, errors.New("Error parsing response: " + err.Error())
	}

	return dataRes, nil
}
