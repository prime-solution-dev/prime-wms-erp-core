package interfaceService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

type HookInterfaceRequest struct {
	RequestData interface{} `json:"request_data"`
	UrlHook     string      `json:"url_hook"`
}

func HookInterface(requestData HookInterfaceRequest) (interface{}, error) {

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		errors.New("error marshalling data :")
	}
	fmt.Println(string(jsonData))
	reqHttp, err := http.NewRequest("POST", os.Getenv("base_url_document")+"/interface/hook-interface", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, errors.New("Error parsing DateTo: " + err.Error())
	}

	reqHttp.Header.Set("Content-Type", "application/json")

	// Create a client and execute the request
	client := &http.Client{}
	resp, err := client.Do(reqHttp)
	if err != nil {
		return nil, errors.New("Error parsing DateTo : " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Response Status:", err)
	}
	var products interface{}
	err = json.Unmarshal(body, &products)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	productMap, _ := products.(map[string]interface{})

	if productMap != nil {
		if idVal, exists := productMap["id"]; exists {
			if idStr, ok := idVal.(string); ok && idStr != "" {
				return productMap, nil
			}
		} else {
			return nil, errors.New(productMap["message"].(string))
		}
	}

	return products, nil

}
