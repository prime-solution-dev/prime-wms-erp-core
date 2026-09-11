package invoiceService

import (
	"encoding/json"
	"errors"
	"math"
	models "prime-erp-core/internal/models"
	repositoryInvoice "prime-erp-core/internal/repositories/invoice"
	customerService "prime-erp-core/internal/services/customer-service"
	interfaceService "prime-erp-core/internal/services/interface-service"
	purchaseService "prime-erp-core/internal/services/purchase-service"
	"strings"

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
	prefix := "CN"
	if req[0].RefPaymentMethod == "CASH" {
		prefix = "CC"
	}
	if req[0].RefPaymentMethod == "CREDIT" {
		prefix = "CN"
	}
	configCodeValue := "RUNNING_CN"
	count := len(req)
	invoiceCodes, err := GenerateInvoiceCodes(ctx, count, prefix, configCodeValue)
	if err != nil {
		return nil, errors.New("failed to generate invoice codes: " + err.Error())
	}
	productCodes := []string{}
	for i := range req {
		req[i].InvoiceCode = invoiceCodes[i]
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
		for it := range req[i].InvoiceItem {
			productCodes = append(productCodes, req[i].InvoiceItem[it].ProductCode)
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

	if req[0].Status == "TEMP" {
		return createInvoiceReturn, nil
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

		productReq := models.GetProductRequest{
			ProductCode: productCodes,
			SiteCode:    []string{req[0].SiteCode},
			CompanyCode: []string{req[0].CompanyCode},
		}
		mapProductInterface, errGetProductInterface := purchaseService.GetProductInterface(productReq)
		if errGetProductInterface != nil {
			return nil, errors.New("failed to get product interface: " + errGetProductInterface.Error())
		}
		reqHook := req
		for i := range reqHook {
			for it := range reqHook[i].InvoiceItem {
				mapProductInterface, exists := mapProductInterface[reqHook[i].InvoiceItem[it].ProductCode]
				if exists {
					priceUnit, _ := calculateAPPriceUnit(
						reqHook[i].InvoiceItem[it].UnitUom, mapProductInterface.UnitInterface,
						reqHook[i].InvoiceItem[it].PriceUnit, reqHook[i].InvoiceItem[it].Qty, reqHook[i].InvoiceItem[it].TotalWeight,
					)
					reqHook[i].InvoiceItem[it].PriceUnit = math.Round(priceUnit*100) / 100
					reqHook[i].InvoiceItem[it].UnitUom = mapProductInterface.UnitInterface
				}
				reqHook[i].InvoiceItem[it].ProductDesc = strings.ReplaceAll(
					reqHook[i].InvoiceItem[it].ProductDesc,
					"\\",
					"",
				)
			}
		}

		requestDataCreateHook := interfaceService.HookInterfaceRequest{
			RequestData: reqHook,
			UrlHook:     urlHook,
		}
		HookInterfaceValue, err := interfaceService.HookInterface(requestDataCreateHook)
		if err != nil {
			return nil, err
		}
		if HookInterfaceValue != nil {
			externalID := HookInterfaceValue.(map[string]interface{})
			str, _ := externalID["id"].(string)

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
