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

	// ห้ามแปะ utils.NewOutboundLogTransport (หรือ apilog.NewLoggingTransport ตรงๆ) ที่ client นี้
	// เด็ดขาด — urlHook มาจากผู้เรียก (พารามิเตอร์ของฟังก์ชัน) ไม่ใช่ปลายทางที่เรากำหนดเอง อาจเป็น
	// third party ภายนอกก็ได้ body ของ request/response นี้ต้องไม่มีทางหลุดลง MongoDB ของเรา (ดู
	// internal/guard/no_apilog_on_deposit_clients_test.go ที่กันไว้ไม่ให้ใครเผลอเติมทีหลัง)
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
