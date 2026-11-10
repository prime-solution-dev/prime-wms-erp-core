package verifyService

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	creditService "prime-erp-core/internal/services/credit-service"
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

func VerifyCredit(ctx context.Context, jsonPayload string) (interface{}, error) {
	req := VerifyCreditRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	return VerifyCreditLogic(ctx, req)
}

// VerifyCreditLogic ใช้ยอดจาก creditService.GetSummaryCredit (ตัวเดียวกับหน้า Customer Credit)
// เพื่อให้ตัวเลขตอน validate ตรงกับหน้าจอ
// GetSummaryCredit รับได้ทีละลูกค้า จึงเรียกแยกรายลูกค้า
func VerifyCreditLogic(ctx context.Context, req VerifyCreditRequest) (*VerifyCreditResponse, error) {
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

	summaryMap := map[string]creditService.ResultGetSummaryCredit{}
	for _, customerCode := range customerStrs {
		payload, err := json.Marshal(map[string][]string{
			"customer_code": {customerCode},
		})
		if err != nil {
			return nil, err
		}

		summaryRes, err := creditService.GetSummaryCredit(ctx, string(payload))
		if err != nil {
			return nil, err
		}

		summaryMap[customerCode] = summaryRes.(creditService.ResultGetSummaryCredit)
	}

	for _, rCustomer := range req.Customers {
		summary := summaryMap[rCustomer.CustomerCode]

		// GetSummaryCredit หักมัดจำรวมไว้ใน consumed_credit แล้ว จึงไม่แยก remain_deposit
		rCustomer.CreditCalculation = VerifyCreditCalculation{
			Subject:          "credit",
			CreditLimit:      summary.CreditLimit,
			ExtraCreditLimit: summary.IncreaseCreditLimit,
			RemainDeposit:    0,
			UsedCredit:       summary.ConsumedCredit,
			RemainCredit:     summary.BalanceCreditLimit,
			NeedAmount:       rCustomer.NeedAmount,
			FinalCredit:      summary.BalanceCreditLimit - rCustomer.NeedAmount,
		}

		rCustomer.IsPass = rCustomer.CreditCalculation.FinalCredit >= 0

		res.Customers = append(res.Customers, rCustomer)
	}

	return &res, nil
}
