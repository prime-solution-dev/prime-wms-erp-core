package externalService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"prime-erp-core/config"

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
		return CancelOrderResponse{}, errors.New("Error marshaling struct to JSON:" + err.Error())
	}
	req, err := http.NewRequest("POST", config.CANCEL_ORDER_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		return CancelOrderResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return CancelOrderResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	var dataRes CancelOrderResponse
	err = json.Unmarshal(body, &dataRes)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	return dataRes, nil
}
