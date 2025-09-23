package authenticationService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type GetRequesterRequest struct {
	MDItemCode  []string `json:"md_item_code"`
	ActionCode  []string `json:"action_code"`
	RequesterID []string `json:"requester_id"`
}
type Requester struct {
	RequesterType string
	RequesterID   string
	RequesterCode string
}

func GetRequester(requestData map[string]interface{}) ([]Requester, error) {

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		errors.New("Error marshalling data :")
	}

	reqHttp, err := http.NewRequest("POST", os.Getenv("base_url_authorization")+"/author/get-requester", bytes.NewBuffer(jsonData))
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
	var requesters []Requester
	err = json.Unmarshal(body, &requesters)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	fmt.Println("Response Status:", resp.Status)

	return requesters, nil

}
