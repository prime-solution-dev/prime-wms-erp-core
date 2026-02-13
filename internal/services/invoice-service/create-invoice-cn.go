package invoiceService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryInvoice "prime-erp-core/internal/repositories/invoice"
	customerService "prime-erp-core/internal/services/customer-service"
	interfaceService "prime-erp-core/internal/services/interface-service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreateInvoiceCN(ctx *gin.Context, jsonPayload string) (interface{}, error) {

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

	for i := range req {
		conMapCustomer, exist := convertCustomerMap[req[i].PartyCode]
		if exist {
			req[i].PartyName = conMapCustomer.CustomerName
			for _, soldValue := range conMapCustomer.Billing {
				req[i].PartyBranch = soldValue.BranchID
				req[i].PartyAddress = conMapCustomer.Address
			}
			req[i].PartyEmail = conMapCustomer.Email
			req[i].PartyTel = conMapCustomer.Phone
			req[i].PartyTaxID = conMapCustomer.TaxID
			req[i].PartyExternalID = conMapCustomer.ExternalID
			req[i].PartyBranch = conMapCustomer.BranchName
		}

	}

	jsonBytesCreateInvoice, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	createInvoiceReturn, errCreateInvoice := CreateInvoice(ctx, string(jsonBytesCreateInvoice))
	if errCreateInvoice != nil {
		return nil, errCreateInvoice
	}

	invoiceMap, _ := createInvoiceReturn.(map[string]interface{})
	idInvoice := invoiceMap["id"].([]uuid.UUID)
	requestData := map[string]interface{}{
		"module":    []string{"INVOICE"},
		"topic":     []string{"CN"},
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

			invoiceValue := []models.Invoice{}
			invoiceValue = append(invoiceValue, models.Invoice{
				ID:         idInvoice[0],
				ExternalID: str,
			})

			_, errCreateApproval := repositoryInvoice.UpdateInvoice(invoiceValue, []models.InvoiceItem{})
			if errCreateApproval != nil {
				return nil, errCreateApproval
			}
		}
	}

	return createInvoiceReturn, nil

}
