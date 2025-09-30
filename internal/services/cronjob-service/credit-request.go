package CronjobService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"prime-erp-core/internal/models"
	"strings"
	"time"

	creditService "prime-erp-core/internal/services/credit-service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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
	var creditRequest creditService.ResultCreditRequest
	err = json.Unmarshal(body, &creditRequest)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	fmt.Println("Response Status:", resp.Status)
	credit := []models.Credit{}
	creditRequestForAlert := []models.CreditRequest{}
	for _, creditRequestValue := range creditRequest.CreditRequest {
		if creditRequestValue.EffectiveDtm.After(time.Now()) || creditRequestValue.EffectiveDtm.Equal(time.Now()) {
			creditExtra := []models.CreditExtra{}
			CreditID := uuid.New()
			if creditRequestValue.RequestType == "EXTRA" {
				creditExtra = append(creditExtra, models.CreditExtra{
					ID:       uuid.New(),
					CreditID: CreditID,
					//ExtraType:    "",
					Amount:       creditRequestValue.Amount,
					EffectiveDtm: creditRequestValue.EffectiveDtm,
					ExpireDtm:    creditRequestValue.ExpireDtm,
					DocRef:       creditRequestValue.RequestCode,
					//ApproveDate:  "",
				})
			}
			credit = append(credit, models.Credit{
				ID:           CreditID,
				CustomerCode: creditRequestValue.CustomerCode,
				Amount:       creditRequestValue.Amount,
				EffectiveDtm: creditRequestValue.EffectiveDtm,
				IsActive:     true,
				DocRef:       creditRequestValue.RequestCode,
				//ApproveDate:        "",
				AlertBalanceCredit: false,
				CreditExtra:        creditExtra,
			})
		}
		if creditRequestValue.BalanceCreditLimit < 0 {
			creditRequestForAlert = append(creditRequestForAlert, creditRequestValue)
		}
	}
	if len(credit) > 0 {
		jsonBytesCredit, err := json.Marshal(credit)
		if err != nil {
			return nil, err
		}
		urlCreateCredit := os.Getenv("base_url_erp") + "/credit/CreateCredit"
		reqCreateCredit, err := http.NewRequest("POST", urlCreateCredit, bytes.NewBuffer(jsonBytesCredit))
		if err != nil {
			return nil, errors.New("Error parsing DateTo: " + err.Error())
		}

		reqCreateCredit.Header.Set("Content-Type", "application/json")

		// Create a client and execute the request
		clientCreateCredit := &http.Client{}
		respCreateCredit, errCreateCredit := clientCreateCredit.Do(reqCreateCredit)
		if errCreateCredit != nil {
			return nil, errors.New("Error parsing DateTo: " + errCreateCredit.Error())
		}
		defer respCreateCredit.Body.Close()

		bodyCreateCredit, err := io.ReadAll(respCreateCredit.Body)
		if err != nil {
			return nil, err
		}
		var convertCreateCredit interface{}
		err = json.Unmarshal(bodyCreateCredit, &convertCreateCredit)
		if err != nil {
			return nil, err
		}
	}
	if len(creditRequestForAlert) > 0 {
		jsonBytesEmailAlert, err := json.Marshal(creditRequestForAlert)
		if err != nil {
			return nil, err
		}
		urlEmailAlert := os.Getenv("base_url_erp") + "/emailAlert/SendEmailAlertForNewBrand"
		reqEmailAlert, err := http.NewRequest("POST", urlEmailAlert, bytes.NewBuffer(jsonBytesEmailAlert))
		if err != nil {
			return nil, errors.New("Error parsing DateTo: " + err.Error())
		}

		reqEmailAlert.Header.Set("Content-Type", "application/json")

		// Create a client and execute the request
		clientEmailAlert := &http.Client{}
		respEmailAlert, errEmailAlert := clientEmailAlert.Do(reqEmailAlert)
		if errEmailAlert != nil {
			return nil, errors.New("Error parsing DateTo: " + errEmailAlert.Error())
		}
		defer respEmailAlert.Body.Close()

		bodyEmailAlert, err := io.ReadAll(respEmailAlert.Body)
		if err != nil {
			return nil, err
		}
		var convertEmailAlert interface{}
		err = json.Unmarshal(bodyEmailAlert, &convertEmailAlert)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil

}
