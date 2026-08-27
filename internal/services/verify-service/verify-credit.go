package verifyService

import (
	"encoding/json"
	"errors"
	"fmt"
	"prime-erp-core/internal/db"
	creditService "prime-erp-core/internal/services/credit-service"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type VerifyCreditRequest struct {
	Customers []VerifyCreditCustomer `json:"customers"`
}

type VerifyCreditResponse struct {
	Customers []VerifyCreditCustomer `json:"customers"`
}

type VerifyCreditCustomer struct {
	CustomerCode string  `json:"customer_code"`
	NeedAmount   float64 `json:"need_amount"`

	//Result
	CreditCalculation VerifyCreditCalculation `json:"credit_calculation"`
	IsPass            bool                    `json:"is_pass"`
}

type VerifyCreditCalculation struct {
	Subject          string  `json:"subject"`
	CreditLimit      float64 `json:"credit_limit"`
	ExtraCreditLimit float64 `json:"extra_credit_limit"`
	RemainDeposit    float64 `json:"remain_deposit"`
	UsedCredit       float64 `json:"used_credit"`
	RemainCredit     float64 `json:"remain_credit"`
	NeedAmount       float64 `json:"need_amount"`
	FinalCredit      float64 `json:"final_credit"`
}

func VerifyCredit(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := VerifyCreditRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	sqlx, err := db.ConnectSqlx(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer sqlx.Close()

	return VerifyCreditLogic(sqlx, req)
}

func VerifyCreditLogic(sqlx *sqlx.DB, req VerifyCreditRequest) (*VerifyCreditResponse, error) {
	res := VerifyCreditResponse{}

	customerStrs := []string{}
	customerStrsCheck := map[string]bool{}
	for _, customer := range req.Customers {
		if _, ok := customerStrsCheck[customer.CustomerCode]; !ok {
			customerStrs = append(customerStrs, customer.CustomerCode)
			customerStrsCheck[customer.CustomerCode] = true
		}
	}

	if len(customerStrs) == 0 {
		return nil, fmt.Errorf(`required customer`)
	}

	creditReq := creditService.GetCreditRequest{}
	creditReq.CustomerCodes = customerStrs
	creditCustomer, err := creditService.GetCreditCurrent(sqlx, creditReq)
	if err != nil {
		return nil, err
	}

	for _, rCustomer := range req.Customers {
		for _, credit := range creditCustomer.CreditCustomers {
			if credit.CustomerCode == rCustomer.CustomerCode {
				rCustomer.CreditCalculation = VerifyCreditCalculation{
					Subject:          "credit",
					CreditLimit:      credit.Credit,
					ExtraCreditLimit: credit.Extra,
					RemainDeposit:    credit.RemainDeposit,
					UsedCredit:       credit.Used,
					RemainCredit:     credit.Balance,
					NeedAmount:       rCustomer.NeedAmount,
					FinalCredit:      credit.Balance - rCustomer.NeedAmount,
				}

				if rCustomer.CreditCalculation.FinalCredit >= 0 {
					rCustomer.IsPass = true
				} else {
					rCustomer.IsPass = false
				}

				break
			}
		}

		res.Customers = append(res.Customers, rCustomer)
	}

	return &res, nil
}
