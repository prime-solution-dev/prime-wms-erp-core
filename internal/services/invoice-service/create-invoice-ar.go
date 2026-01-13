package invoiceService

import (
	"encoding/json"
	"errors"
	"fmt"
	models "prime-erp-core/internal/models"
	repositoryDeposit "prime-erp-core/internal/repositories/deposit"
	customerService "prime-erp-core/internal/services/customer-service"
	interfaceService "prime-erp-core/internal/services/interface-service"
	systemConfigService "prime-erp-core/internal/services/system-config"
	"strconv"

	"github.com/gin-gonic/gin"
)

func CreateInvoiceAR(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Invoice

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	customerCode := []string{}

	for _, reqValue := range req {
		customerCode = append(customerCode, reqValue.PartyCode)
	}

	requestDataGetCustomers := map[string]interface{}{
		"customer_code": customerCode,
	}

	customers, err := customerService.GetCustomers(requestDataGetCustomers)
	if err != nil {
		return nil, err
	}

	convertCustomerMap := map[string]customerService.GetCustomerResponse{}
	for _, customer := range customers.Customers {
		convertCustomerMap[customer.CustomerCode] = customer
	}
	prefix := ""
	if req[0].PaymentMethod == "CASH" {
		prefix = "CC"
	}
	if req[0].PaymentMethod == "CREDIT" {
		prefix = "CN"
	}

	count := len(req)
	purchaseCodes, err := GeneratePurchaseCodes(ctx, count, prefix)
	if err != nil {
		return nil, errors.New("failed to generate purchase order codes: " + err.Error())
	}
	fmt.Println(purchaseCodes)

	for i := range req {
		conMapCustomer, exist := convertCustomerMap[req[i].PartyCode]
		if exist {
			req[i].PartyName = conMapCustomer.CustomerName
			for _, soldValue := range conMapCustomer.Sold {
				req[i].PartyBranch = soldValue.BranchID
				req[i].PartyAddress = conMapCustomer.Address
			}
			req[i].PartyEmail = conMapCustomer.Email
			req[i].PartyTel = conMapCustomer.Phone
			req[i].PartyTaxID = conMapCustomer.TaxID
			req[i].PartyExternalID = conMapCustomer.ExternalID
		}
		req[i].InvoiceCode = purchaseCodes[i]
	}

	requestData := map[string]interface{}{
		"module":    []string{"INVOICE"},
		"topic":     []string{"AR"},
		"sub_topic": []string{"CREATE"},
	}

	hookConfig, err := interfaceService.GetHookConfig(requestData)
	if err != nil {
		return nil, err
	}
	if len(hookConfig) > 0 {
		urlHook := ""
		for _, hookConfigValue := range hookConfig {
			urlHook = hookConfigValue.HookUrl
		}

		requestDataCreateHook := interfaceService.HookInterfaceRequest{
			RequestData: req,
			UrlHook:     urlHook,
		}
		HookInterfaceValue, err := interfaceService.HookInterface(requestDataCreateHook)
		if err != nil {
			return nil, err
		}
		if HookInterfaceValue != nil {
			str, _ := HookInterfaceValue.(string)

			/* 	invoiceValue := []models.Invoice{}
			invoiceValue = append(invoiceValue, models.Invoice{
				ID:         idInvoice[0],
				ExternalID: str,
			})

			_, errCreateApproval := repositoryInvoice.UpdateInvoice(invoiceValue, []models.InvoiceItem{})
			if errCreateApproval != nil {
				return nil, errCreateApproval
			} */
			req[0].ExternalID = str
			jsonBytesCreateInvoice, err := json.Marshal(req)
			if err != nil {
				return nil, err
			}

			createInvoiceReturn, errCreateInvoice := CreateInvoice(ctx, string(jsonBytesCreateInvoice))
			if errCreateInvoice != nil {
				return nil, errCreateInvoice
			}

			depositMapResult, err := interfaceService.GetDeposit(str)
			if err != nil {
				return nil, err
			}
			if len(depositMapResult) > 0 {
				var deposit []models.Deposit

				for _, v := range depositMapResult {
					depMap, _ := v.(map[string]interface{})

					totalFloat, err := strconv.ParseFloat(depMap["total"].(string), 64)
					if err != nil {
						totalFloat = 0
					}
					drFloat, err := strconv.ParseFloat(depMap["dr"].(string), 64)
					if err != nil {
						totalFloat = 0
					}
					crFloat, err := strconv.ParseFloat(depMap["cr"].(string), 64)
					if err != nil {
						totalFloat = 0
					}

					deposit = append(deposit, models.Deposit{
						DepositCode:  depMap["anchor"].(string),
						CustomerCode: req[0].PartyCode,
						AmountTotal:  totalFloat,
						AmountUsed:   drFloat,
						AmountRemain: crFloat,
						Status:       "PENDING",
					})
				}
				errDeposit := repositoryDeposit.CreateDeposit(deposit)
				if errDeposit != nil {
					return nil, errDeposit
				}

			}
			return createInvoiceReturn, nil
		}
	} else {
		jsonBytesCreateInvoice, err := json.Marshal(req)
		if err != nil {
			return nil, err
		}

		createInvoiceReturn, errCreateInvoice := CreateInvoice(ctx, string(jsonBytesCreateInvoice))
		if errCreateInvoice != nil {
			return nil, errCreateInvoice
		}
		return createInvoiceReturn, nil
	}
	return nil, nil
}
func GeneratePurchaseCodes(ctx *gin.Context, count int, prefix string) ([]string, error) {
	if count <= 0 {
		return []string{}, nil // No purchases to generate codes for
	}

	configCode := "RUNNING_AR"

	getReq := systemConfigService.GetRunningSystemConfigRequest{
		ConfigCode: configCode,
		Count:      count,
		Prefix:     prefix,
	}

	reqJSON, err := json.Marshal(getReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get request: %v", err)
	}

	purchaseCodeResponse, err := systemConfigService.GetRunningSystemConfigInvoice(ctx, string(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to generate purchase order codes: %v", err)
	}

	updateReq := systemConfigService.UpdateRunningSystemConfigRequest{
		ConfigCode: configCode,
		Count:      count,
	}

	reqUpdateJSON, err := json.Marshal(updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update request: %v", err)
	}

	_, err = systemConfigService.UpdateRunningSystemConfigInvoice(ctx, string(reqUpdateJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to update running config: %v", err)
	}

	purchaseCodeResult, ok := purchaseCodeResponse.(systemConfigService.GetRunningSystemConfigResponse)
	if !ok || len(purchaseCodeResult.Data) != count {
		return nil, errors.New("failed to get correct number of purchase order codes from system config")
	}

	return purchaseCodeResult.Data, nil
}
