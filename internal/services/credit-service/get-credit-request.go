package creditService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryCredit "prime-erp-core/internal/repositories/credit"
	customerService "prime-erp-core/internal/services/customer-service"
	depositService "prime-erp-core/internal/services/deposit-service"
	summaryService "prime-erp-core/internal/services/summary-credit"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetCreditReq struct {
	ID           []uuid.UUID `json:"id"`
	CustomerCode []string    `json:"customer_code"`
	Page         int         `json:"page"`
	PageSize     int         `json:"page_size"`
}
type ResultCreditRequest struct {
	Total         int                    `json:"total"`
	Page          int                    `json:"page"`
	PageSize      int                    `json:"page_size"`
	TotalPages    int                    `json:"total_pages"`
	CreditRequest []models.CreditRequest `json:"credit_request"`
}

func GetCreditRequests(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetCreditReq

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	credit, totalPages, totalRecords, errApproval := repositoryCredit.GetCreditRequestPreload(req.ID, req.CustomerCode, req.Page, req.PageSize)
	if errApproval != nil {
		return nil, errApproval
	}
	customerCode := []string{}

	for _, creditValue := range credit {
		customerCode = append(customerCode, creditValue.CustomerCode)
	}

	requestData := map[string]interface{}{
		"customer_code": customerCode,
	}

	customers, err := customerService.GetCustomers(requestData)
	if err != nil {
		return nil, err
	}

	convertCustomerMap := map[string]customerService.GetCustomerResponse{}
	for _, customer := range customers.Customers {
		convertCustomerMap[customer.CustomerCode] = customer
	}

	jsonBytesGetCredit, err := json.Marshal(requestData)
	if err != nil {
		return nil, err
	}

	/* 	GetCreditRes, errGetCredit := GetCredit(ctx, string(jsonBytesGetCredit))
	   	if errGetCredit != nil {
	   		return nil, errGetCredit
	   	}
	   	creditExtraValueMap := map[string]float64{}

	   	for _, creditValue := range GetCreditRes.(ResultCredit).Credit {
	   		for _, creditExtraValue := range creditValue.CreditExtra {

	   			if creditExtraValue.EffectiveDtm == nil {
	   				creditExtraItemMap, exist := creditExtraValueMap[creditValue.CustomerCode]
	   				if exist {
	   					creditExtraValueMap[creditValue.CustomerCode] = creditExtraItemMap + creditExtraValue.Amount
	   				} else {
	   					creditExtraValueMap[creditValue.CustomerCode] = creditExtraValue.Amount
	   				}
	   			} else {
	   				if (creditExtraValue.EffectiveDtm.After(time.Now()) || creditExtraValue.EffectiveDtm.Equal(time.Now())) && (creditExtraValue.ExpireDtm.After(time.Now()) || creditExtraValue.ExpireDtm.Equal(time.Now())) {
	   					creditExtraItemMap, exist := creditExtraValueMap[creditValue.CustomerCode]
	   					if exist {
	   						creditExtraValueMap[creditValue.CustomerCode] = creditExtraItemMap + creditExtraValue.Amount
	   					} else {
	   						creditExtraValueMap[creditValue.CustomerCode] = creditExtraValue.Amount
	   					}
	   				}
	   			}

	   		}
	   	} */

	getDepositRes, errGetDeposit := depositService.GetDeposit(ctx, string(jsonBytesGetCredit))
	if errGetDeposit != nil {
		return nil, errGetDeposit
	}
	getDeposit := getDepositRes.(depositService.ResultDeposit).Deposit
	remainDepositMap := map[string]float64{}
	for _, depositValue := range getDeposit {
		remainDepositItemMap, exist := remainDepositMap[depositValue.CustomerCode]
		if exist {
			remainDepositMap[depositValue.CustomerCode] = remainDepositItemMap + depositValue.AmountRemain
		} else {
			remainDepositMap[depositValue.CustomerCode] = depositValue.AmountRemain
		}
	}

	for i := range credit {

		requestDataGetConsumend := map[string]interface{}{
			"customer_code": credit[i].CustomerCode,
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

		conMapCustomer, exist := convertCustomerMap[credit[i].CustomerCode]
		if exist {
			credit[i].CustomerName = conMapCustomer.CustomerName
			credit[i].CustomeStatus = conMapCustomer.ActiveFlg
		}

		conMapremainDeposit, exist := remainDepositMap[credit[i].CustomerCode]
		if exist {
			credit[i].ConsumedCredit = conMapremainDeposit - (resultGetPaidInvoice.TotalAmount - resultGetPaidInvoice.SumInvoiceTotalAmountDN +
				resultGetPaidInvoice.SumInvoiceTotalAmountCN + resultGetPaidInvoice.SumPaymentTotalAmountAR + resultGetPaidInvoice.SumPaymentTotalAmountDN)

		}
		credit[i].BalanceCreditLimit = (credit[i].Amount + credit[i].TemporaryIncreaseCreditLimit) - credit[i].ConsumedCredit

	}

	resultApproval := ResultCreditRequest{
		Total:         totalRecords,
		Page:          req.Page,
		PageSize:      req.PageSize,
		TotalPages:    totalPages,
		CreditRequest: credit,
	}

	return resultApproval, nil
}
func GetCreditRequestCronjob(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetCreditReq

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	credit, totalPages, totalRecords, errApproval := repositoryCredit.GetCreditRequest(req.ID, req.CustomerCode, req.Page, req.PageSize)
	if errApproval != nil {
		return nil, errApproval
	}

	resultApproval := ResultCreditRequest{
		Total:         totalRecords,
		Page:          req.Page,
		PageSize:      req.PageSize,
		TotalPages:    totalPages,
		CreditRequest: credit,
	}

	return resultApproval, nil
}
