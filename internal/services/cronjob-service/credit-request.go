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

	"github.com/google/uuid"
)

func CreditRequestEffectiveDtm() (interface{}, error) {

	url := os.Getenv("base_url_erp") + "/credit/GetCreditRequestCronjob"
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
	creditTransaction := []models.CreditTransaction{}
	creditRequestUpdate := []models.CreditRequest{}
	for _, creditRequestValue := range creditRequest.CreditRequest {
		if creditRequestValue.ExpireDtm != nil {
			now := time.Now()
			exp := creditRequestValue.ExpireDtm
			if !exp.Before(now) {
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
		if creditRequestValue.EffectiveDtm != nil {
			fmt.Println("Now:", time.Now().Format(time.RFC3339))
			fmt.Println("Effective:", creditRequestValue.EffectiveDtm)
			now := time.Now()
			eff := creditRequestValue.EffectiveDtm
			if !eff.Before(now) {
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
				creditRequestUpdate = append(creditRequestUpdate, models.CreditRequest{
					ID:       creditRequestValue.ID,
					IsAction: true,
				})
			}
		}

		if creditRequestValue.BalanceCreditLimit < 0 {
			creditRequestForAlert = append(creditRequestForAlert, creditRequestValue)
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
