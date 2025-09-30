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

func CreditExtra(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	url := os.Getenv("base_url_erp") + "/credit/GetCredit"
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
	var creditRequest creditService.ResultCredit
	err = json.Unmarshal(body, &creditRequest)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	fmt.Println("Response Status:", resp.Status)
	creditTransaction := []models.CreditTransaction{}
	creditExtraID := []uuid.UUID{}
	for _, creditValue := range creditRequest.Credit {
		for _, creditExtraValue := range creditValue.CreditExtra {
			if creditExtraValue.ExpireDtm.After(time.Now()) || creditExtraValue.ExpireDtm.Equal(time.Now()) {
				creditTransaction = append(creditTransaction, models.CreditTransaction{
					TransactionCode: creditExtraValue.DocRef,
					TransactionType: "EXTRA",
					Amount:          creditExtraValue.Amount,
					AdjustAmount:    0,
					EffectiveDtm:    creditExtraValue.EffectiveDtm,
					ExpireDtm:       creditExtraValue.ExpireDtm,
					//ForceExpireDtm:  req[i].e,
					//ApproveDate:     "",
					IsApprove: false,
					Status:    "INACTIVE",
					Reason:    "",
				})
			}
			creditExtraID = append(creditExtraID, creditExtraValue.ID)
		}
	}

	jsonBytesDeleteCreditExtra, err := json.Marshal(creditExtraID)
	if err != nil {
		return nil, err
	}
	urlCreateDeleteCreditExtra := os.Getenv("base_url_erp") + "/credit/DeleteCreditExtra"
	reqCreateDeleteCreditExtra, err := http.NewRequest("POST", urlCreateDeleteCreditExtra, bytes.NewBuffer(jsonBytesDeleteCreditExtra))
	if err != nil {
		return nil, errors.New("Error parsing DateTo: " + err.Error())
	}

	reqCreateDeleteCreditExtra.Header.Set("Content-Type", "application/json")

	// Create a client and execute the request
	clientCreateDeleteCreditExtra := &http.Client{}
	respCreateDeleteCreditExtra, errCreateDeleteCreditExtra := clientCreateDeleteCreditExtra.Do(reqCreateDeleteCreditExtra)
	if errCreateDeleteCreditExtra != nil {
		return nil, errors.New("Error parsing DateTo: " + errCreateDeleteCreditExtra.Error())
	}
	defer respCreateDeleteCreditExtra.Body.Close()

	bodyCreateDeleteCreditExtra, err := io.ReadAll(respCreateDeleteCreditExtra.Body)
	if err != nil {
		return nil, err
	}
	var convertCreateDeleteCreditExtra interface{}
	err = json.Unmarshal(bodyCreateDeleteCreditExtra, &convertCreateDeleteCreditExtra)
	if err != nil {
		return nil, err
	}

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

	return nil, nil

}
