package externalService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"prime-erp-core/config"
)

type GetSystemConfigRequest struct {
	TopicCodes  []string `json:"topic_codes"`
	ConfigCodes []string `json:"config_codes"`
}

type GetSystemConfigResponse struct {
	SystemConfigs []SystemConfigDto `json:"system_configs"`
}

type SystemConfigDto struct {
	TopicCode  string `json:"topic_code"`
	ConfigCode string `json:"config_code"`
	Value      string `json:"value"`
	JSON       string `json:"json,omitempty"`
}

func GetSystemConfig(jsonPayload GetSystemConfigRequest) (GetSystemConfigResponse, error) {

	jsonData, err := json.Marshal(jsonPayload)
	if err != nil {
		return GetSystemConfigResponse{}, errors.New("Error marshaling struct to JSON:" + err.Error())
	}
	req, err := http.NewRequest("POST", config.GET_SYSTEM_CONFIG_WAREHOUSE_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		return GetSystemConfigResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return GetSystemConfigResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	var dataRes GetSystemConfigResponse
	err = json.Unmarshal(body, &dataRes)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	return dataRes, nil
}
