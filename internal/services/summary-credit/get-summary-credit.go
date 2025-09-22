package creditService

import (
	"encoding/json"
	"errors"
	creditService "prime-erp-core/internal/services/credit-service"
	depositService "prime-erp-core/internal/services/deposit-service"
	paymentService "prime-erp-core/internal/services/payment-service"
	saleService "prime-erp-core/internal/services/sale-service"
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

	var req creditService.GetApprovalRequest

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

	credit, errApproval := creditService.GetCredit(ctx, string(jsonBytesCustomerCode))
	if errApproval != nil {
		return nil, errApproval
	}
	resultCredit := credit.(creditService.ResultCredit).Credit

	creditLimit := 0.00
	increaseCreditLimit := 0.00
	remainDeposit := 0.00
	totalAmount := 0.00
	for _, creditValue := range resultCredit {
		creditLimit += creditValue.Amount
		for _, creditExtraValue := range creditValue.CreditExtra {
			if creditValue.EffectiveDtm.After(time.Now()) || creditValue.EffectiveDtm.Equal(time.Now()) {
				increaseCreditLimit += creditExtraValue.Amount
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

	requestDataGetSale := map[string]interface{}{
		"customer_code":  req.CustomerCode,
		"status":         []string{"PENDING", "COMPLETED"},
		"status_payment": []string{"PENDING"},
		"is_approved":    []bool{true},
	}

	jsonBytesSale, err := json.Marshal(requestDataGetSale)
	if err != nil {
		return nil, err
	}

	sale, errGetSale := saleService.GetSale(ctx, string(jsonBytesSale))
	if errGetSale != nil {
		return nil, errGetSale
	}
	resultSale := sale.(saleService.ResultSale).Sale
	for _, saleValue := range resultSale {
		totalAmount += saleValue.TotalAmount
	}

	paymentle, errGetPayment := paymentService.GetPayment(ctx, string(jsonBytesSale))
	if errGetPayment != nil {
		return nil, errGetPayment
	}
	resultPayment := paymentle.(paymentService.ResultPayment).Payment
	for _, paymentValue := range resultPayment {
		totalAmount += paymentValue.Amount
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
