package goodsReceiveService

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"prime-erp-core/config"
	"prime-erp-core/internal/utils"
	"time"
)

type CompanyFilter struct {
	CompanyCode []string `json:"company_code"`
}

type ResultCompany struct {
	Company []Company `json:"company"`
}

type Company struct {
	CompanyCode   string        `json:"company_code"`
	CompanyConfig CompanyConfig `json:"company_config"`
}

type CompanyConfig struct {
	WeightDigit *int `json:"weight_digit"`
}

func GetCompany(ctx context.Context, jsonPayload CompanyFilter) (ResultCompany, error) {

	jsonData, err := json.Marshal(jsonPayload)
	if err != nil {
		return ResultCompany{}, errors.New("Error marshaling struct to JSON: " + err.Error())
	}

	req, err := utils.NewRequest(ctx, "POST", config.GET_COMPANY_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		return ResultCompany{}, errors.New("Error creating request: " + err.Error())
	}

	client := &http.Client{Timeout: 60 * time.Second, Transport: utils.NewOutboundLogTransport("goods-receive")}
	resp, err := client.Do(req)
	if err != nil {
		return ResultCompany{}, errors.New("Error sending request: " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ResultCompany{}, errors.New("Error reading response body: " + err.Error())
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			Error string `json:"error"`
		}
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Error != "" {
			return ResultCompany{}, errors.New(errResp.Error)
		}
		return ResultCompany{}, fmt.Errorf("API error status %d: %s", resp.StatusCode, string(body))
	}

	var dataRes ResultCompany
	if err = json.Unmarshal(body, &dataRes); err != nil {
		return ResultCompany{}, errors.New("Error parsing response: " + err.Error())
	}

	return dataRes, nil
}
