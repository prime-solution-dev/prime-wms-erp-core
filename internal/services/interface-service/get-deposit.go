package interfaceService

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func GetDeposit(extelnalID string) ([]interface{}, error) {

	timestamp := time.Now().Unix()
	encryptHead := "d6d413"
	raw := fmt.Sprintf("%st%d", encryptHead, timestamp)
	hash := md5.Sum([]byte(raw))
	hashString := hex.EncodeToString(hash[:])
	depositReq := map[string]interface{}{
		"company_id": "2",
		"passkey":    "d590bc05f00bef126f43911aacdc7f71",
		"timestamp":  timestamp,
		"securekey":  hashString,
		"acc_code":   "2334000",
		"contact_id": extelnalID,
	}
	jsonData, err := json.Marshal(depositReq)
	if err != nil {
		log.Fatal("Error marshalling JSON:", err)
	}
	formValues := url.Values{}
	formValues.Add("json", string(jsonData))
	var respBodyValue interface{}

	url := "https://tmi.trcloud.co/application/api-connector2/end-point/contact/deposit.php"

	req, err := http.NewRequest("POST", url, strings.NewReader(formValues.Encode()))
	if err != nil {
		panic(fmt.Sprintf("Failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "PRIME")

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("Request error: %v", err))
	}
	defer resp.Body.Close()

	// อ่าน response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("Failed to read response: %v", err))
	}
	err = json.Unmarshal(respBody, &respBodyValue)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	depositMap, _ := respBodyValue.(map[string]interface{})
	depositMapResult, _ := depositMap["result"].([]interface{})

	return depositMapResult, nil

}
