package interfaceService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/google/uuid"
)

type HookConfig struct {
	ID         uuid.UUID `json:"id"`
	Module     string    `json:"module"`
	Topic      string    `json:"topic"`
	SubTopic   string    `json:"sub_topic"`
	HookMethod string    `json:"hook_method"`
	HookUrl    string    `json:"hook_url"`
	Header     string    `json:"header"`
	Body       string    `json:"body"`
}

func GetHookConfig(requestData map[string]interface{}) ([]HookConfig, error) {

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		errors.New("Error marshalling data :")
	}
	fmt.Println(string(jsonData))

	reqHttp, err := http.NewRequest("POST", os.Getenv("base_url_document")+"/interface/get-hook-config", bytes.NewBuffer(jsonData))
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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Response Status:", err)
	}
	var hookConfig []HookConfig
	err = json.Unmarshal(body, &hookConfig)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	fmt.Println("Response Status:", resp.Status)

	return hookConfig, nil

}
