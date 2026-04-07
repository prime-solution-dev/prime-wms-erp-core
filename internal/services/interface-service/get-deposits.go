package interfaceService

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

func GetDeposits(extelnalID string, urlHook string) ([]HookConfig, error) {

	form := url.Values{}
	form.Add("id", extelnalID)

	reqHttp, err := http.NewRequest("POST", urlHook, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, errors.New("Error parsing DateTo: " + err.Error())
	}
	reqHttp.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
