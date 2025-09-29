package CronjobService

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"prime-erp-core/internal/models"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ResultCreditRequest struct {
	Total         int                    `json:"total"`
	Page          int                    `json:"page"`
	PageSize      int                    `json:"page_size"`
	TotalPages    int                    `json:"total_pages"`
	CreditRequest []models.CreditRequest `json:"credit_request"`
}

func CreditRequestEffectiveDtm(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	url := os.Getenv("base_url_erp") + "/credit/GetCreditRequest"
	bodyNewRequest := strings.NewReader(`{}`)
	reqHttp, err := http.NewRequest("POST", url, bodyNewRequest)
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
	var creditRequest ResultCreditRequest
	err = json.Unmarshal(body, &creditRequest)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	fmt.Println("Response Status:", resp.Status)

	for _, creditRequestValue := range creditRequest.CreditRequest {

		if creditRequestValue.EffectiveDtm.After(time.Now()) || creditRequestValue.EffectiveDtm.Equal(time.Now()) {
			fmt.Println("effective_date is today or in the future")
		}

	}

	return creditRequest, nil

}
