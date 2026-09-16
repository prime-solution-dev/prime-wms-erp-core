package interfaceService

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type HookInterfaceRequest struct {
	RequestData interface{} `json:"request_data"`
	UrlHook     string      `json:"url_hook"`
}

func HookInterface(requestData HookInterfaceRequest) (interface{}, error) {
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("hook interface: encode request: %w", err)
	}
	reqHttp, err := http.NewRequest("POST", os.Getenv("base_url_document")+"/interface/hook-interface", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("hook interface: create request: %w", err)
	}
	reqHttp.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(reqHttp)
	if err != nil {
		return nil, fmt.Errorf("hook interface: send request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("hook interface (%s): read response: %w", resp.Status, err)
	}
	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("hook interface (%s): invalid JSON response: %w", resp.Status, err)
	}
	resultMap, _ := result.(map[string]interface{})
	message, _ := resultMap["message"].(string)
	message = strings.TrimSpace(message)
	if message == "" {
		message, _ = resultMap["error"].(string)
		message = strings.TrimSpace(message)
	}
	responseError := func(reason string) (interface{}, error) {
		if message != "" {
			return nil, fmt.Errorf("hook interface (%s): %s: %s", resp.Status, reason, message)
		}
		return nil, fmt.Errorf("hook interface (%s): %s", resp.Status, reason)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return responseError("request failed")
	}
	if resultMap == nil {
		return responseError("expected a JSON object")
	}
	if id, ok := resultMap["id"].(string); ok && strings.TrimSpace(id) != "" {
		return resultMap, nil
	}
	httpStatus, ok := resultMap["HTTP"].(string)
	if !ok || strings.TrimSpace(httpStatus) == "" {
		return responseError("missing or invalid response fields: expected a non-empty id or HTTP string")
	}
	if httpStatus == "200 Success" {
		return message, nil
	}
	return responseError(fmt.Sprintf("upstream HTTP status %q", httpStatus))
}
