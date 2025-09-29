package creditService

import (
	"encoding/json"
	"errors"
	depositService "prime-erp-core/internal/services/deposit-service"
	summaryService "prime-erp-core/internal/services/summary-credit"
	"strings"
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

	requestDataGetDeposit := map[string][]string{
		"customer_code": req.CustomerCode,
	}
	jsonBytesCustomerCode, err := json.Marshal(requestDataGetDeposit)
	if err != nil {
		return nil, err
	}

	credit, errApproval := GetCredit(ctx, string(jsonBytesCustomerCode))
	if errApproval != nil {
		return nil, errApproval
	}
	resultCredit := credit.(ResultCredit).Credit

	creditLimit := 0.00
	increaseCreditLimit := 0.00
	remainDeposit := 0.00

	for _, creditValue := range resultCredit {
		creditLimit += creditValue.Amount
		for _, creditExtraValue := range creditValue.CreditExtra {
			if creditValue.EffectiveDtm == nil {
				increaseCreditLimit += creditExtraValue.Amount
			} else {
				if creditValue.EffectiveDtm.After(time.Now()) || creditValue.EffectiveDtm.Equal(time.Now()) {
					increaseCreditLimit += creditExtraValue.Amount
				}
			}

		}
	}

	getDepositRes, errGetDeposit := depositService.GetDeposit(ctx, string(jsonBytesCustomerCode))
	if errGetDeposit != nil {
		return nil, errGetDeposit
	}
	getDeposit := getDepositRes.(depositService.ResultDeposit).Deposit
	for _, depositValue := range getDeposit {
		remainDeposit += depositValue.AmountRemain
	}

	requestDataGetConsumend := map[string]interface{}{
		"customer_code": strings.Join(req.CustomerCode, ""),
		"paid_invoice":  true,
	}
	jsonBytesGetConsumend, err := json.Marshal(requestDataGetConsumend)
	if err != nil {
		return nil, err
	}

	paidInvoice, errApproval := summaryService.GetConsumend(ctx, string(jsonBytesGetConsumend))
	if errApproval != nil {
		return nil, errApproval
	}
	resultGetPaidInvoice := paidInvoice.(summaryService.ResultGetPaidInvoices)

	resultSummaryCredit := ResultGetSummaryCredit{
		CreditLimit:         creditLimit,
		IncreaseCreditLimit: increaseCreditLimit,
		TotalCreditLimit:    creditLimit + increaseCreditLimit,
		ConsumedCredit:      remainDeposit - (resultGetPaidInvoice.TotalAmount + resultGetPaidInvoice.PaidInvoice),
		BalanceCreditLimit:  (creditLimit + increaseCreditLimit) - (remainDeposit - (resultGetPaidInvoice.TotalAmount + resultGetPaidInvoice.PaidInvoice)),
	}

	return resultSummaryCredit, nil
}
