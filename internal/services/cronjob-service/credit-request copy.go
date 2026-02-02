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
	"time"

	creditService "prime-erp-core/internal/services/credit-service"
)

func CreditRequestEffectiveDtmPending() (interface{}, error) {

	url := os.Getenv("base_url_erp") + "/credit/GetCreditRequestCronjob"
	requestData := map[string]interface{}{
		"request_type": []string{"EXTRA"},
		"status":       []string{"PENDING"},
	}
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		errors.New("Error marshalling data :")
	}
	reqHttp, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
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

	creditTransaction := []models.CreditTransaction{}
	creditRequestUpdate := []models.CreditRequest{}
	for _, creditRequestValue := range creditRequest.CreditRequest {
		if creditRequestValue.ExpireDtm != nil {
			exp := creditRequestValue.ExpireDtm
			loc, _ := time.LoadLocation("Asia/Bangkok")
			expLocal := exp.In(loc)
			nowLocal := time.Now().In(loc)
			if expLocal.Before(nowLocal) {
				creditTransaction = append(creditTransaction, models.CreditTransaction{
					TransactionCode: creditRequestValue.RequestCode,
					TransactionType: creditRequestValue.RequestType,
					Amount:          creditRequestValue.Amount,
					AdjustAmount:    0,
					EffectiveDtm:    creditRequestValue.EffectiveDtm,
					ExpireDtm:       creditRequestValue.ExpireDtm,
					IsApprove:       false,
					Status:          "EXPIRED",
					Reason:          "",
				})
				creditRequestUpdate = append(creditRequestUpdate, models.CreditRequest{
					ID:                           creditRequestValue.ID,
					Status:                       "CANCELLED",
					RequestCode:                  creditRequestValue.RequestCode,
					CustomerCode:                 creditRequestValue.CustomerCode,
					CustomerName:                 creditRequestValue.CustomerName,
					TemporaryIncreaseCreditLimit: creditRequestValue.TemporaryIncreaseCreditLimit,
					ConsumedCredit:               creditRequestValue.ConsumedCredit,
					BalanceCreditLimit:           creditRequestValue.BalanceCreditLimit,
					CustomeStatus:                creditRequestValue.CustomeStatus,
					Amount:                       creditRequestValue.Amount,
					RequestType:                  creditRequestValue.RequestType,
					IsApprove:                    creditRequestValue.IsApprove,
					Reason:                       creditRequestValue.Reason,
					EffectiveDtm:                 creditRequestValue.EffectiveDtm,
					ExpireDtm:                    creditRequestValue.ExpireDtm,
					RequestDate:                  creditRequestValue.RequestDate,
					ActionDate:                   creditRequestValue.ActionDate,
					IsAction:                     creditRequestValue.IsAction,
				})

			}
		}
	}
	if len(creditRequestUpdate) > 0 {
		jsonBytesUpdateCreditRequest, err := json.Marshal(creditRequestUpdate)
		if err != nil {
			return nil, err
		}
		urlUpdateCreditRequest := os.Getenv("base_url_erp") + "/credit/UpdateCreditRequest"
		reqUpdateCreditRequest, err := http.NewRequest("POST", urlUpdateCreditRequest, bytes.NewBuffer(jsonBytesUpdateCreditRequest))
		if err != nil {
			return nil, errors.New("Error parsing DateTo: " + err.Error())
		}

		reqUpdateCreditRequest.Header.Set("Content-Type", "application/json")

		// Create a client and execute the request
		clientUpdateCreditRequest := &http.Client{}
		respUpdateCreditRequest, errUpdateCreditRequest := clientUpdateCreditRequest.Do(reqUpdateCreditRequest)
		if errUpdateCreditRequest != nil {
			return nil, errors.New("Error parsing DateTo: " + errUpdateCreditRequest.Error())
		}
		defer respUpdateCreditRequest.Body.Close()

		bodyUpdateCreditRequest, err := io.ReadAll(respUpdateCreditRequest.Body)
		if err != nil {
			return nil, err
		}
		var convertUpdateCreditRequest interface{}
		err = json.Unmarshal(bodyUpdateCreditRequest, &convertUpdateCreditRequest)
		if err != nil {
			return nil, err
		}

	}
	if len(creditTransaction) > 0 {

		jsonBytesCreditTransaction, err := json.Marshal(creditTransaction)
		if err != nil {
			return nil, err
		}
		urlCreateCreditTransaction := os.Getenv("base_url_erp") + "/credit/CreateCreditTransaction"
		reqCreateCreditTransaction, err := http.NewRequest("POST", urlCreateCreditTransaction, bytes.NewBuffer(jsonBytesCreditTransaction))
		if err != nil {
			return nil, errors.New("Error parsing DateTo: " + err.Error())
		}

		reqCreateCreditTransaction.Header.Set("Content-Type", "application/json")

		// Create a client and execute the request
		clientCreateCreditTransaction := &http.Client{}
		respCreateCreditTransaction, errCreateCreditTransaction := clientCreateCreditTransaction.Do(reqCreateCreditTransaction)
		if errCreateCreditTransaction != nil {
			return nil, errors.New("Error parsing DateTo: " + errCreateCreditTransaction.Error())
		}
		defer respCreateCreditTransaction.Body.Close()

		bodyCreateCreditTransaction, err := io.ReadAll(respCreateCreditTransaction.Body)
		if err != nil {
			return nil, err
		}
		var convertCreateCreditTransaction interface{}
		err = json.Unmarshal(bodyCreateCreditTransaction, &convertCreateCreditTransaction)
		if err != nil {
			return nil, err
		}

	}
	return nil, nil

}
