package creditService

import (
	"encoding/json"
	"errors"
	repositoryCredit "prime-erp-core/internal/repositories/credit"
	"time"

	"github.com/gin-gonic/gin"
)

type GetSummaryCreditRequest struct {
	CustomerCode string `json:"customer_code"`
}
type ResultGetSummaryCredit struct {
	CreditLimit         float64 `json:"credit_limit"`
	IncreaseCreditLimit float64 `json:"increase_credit_limit"`
	TotalCreditLimit    float64 `json:"total_credit_limit"`
	ConsumedCredit      float64 `json:"consumed_credit"`
	BalanceCreditLimit  float64 `json:"balance_credit_limit"`
}

func GetSummaryCredit(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetApprovalRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	credit, _, _, errApproval := repositoryCredit.GetCreditPreload(nil, req.CustomerCode, nil, 0, 0)
	if errApproval != nil {
		return nil, errApproval
	}
	creditLimit := 0.00
	increaseCreditLimit := 0.00
	for _, creditValue := range credit {
		creditLimit += creditValue.Amount
		for _, creditExtraValue := range creditValue.CreditExtra {
			if creditValue.EffectiveDtm.After(time.Now()) || creditValue.EffectiveDtm.Equal(time.Now()) {
				increaseCreditLimit += creditExtraValue.Amount
			}
		}
	}

	resultSummaryCredit := ResultGetSummaryCredit{
		CreditLimit:         creditLimit,
		IncreaseCreditLimit: increaseCreditLimit,
		TotalCreditLimit:    creditLimit + increaseCreditLimit,
		ConsumedCredit:      0,
		BalanceCreditLimit:  0,
	}

	return resultSummaryCredit, nil
}
