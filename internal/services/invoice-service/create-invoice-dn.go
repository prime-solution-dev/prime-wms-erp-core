package invoiceService

import (
	"encoding/json"
	"errors"
	"math"
	models "prime-erp-core/internal/models"
	customerService "prime-erp-core/internal/services/customer-service"
	interfaceService "prime-erp-core/internal/services/interface-service"
	purchaseService "prime-erp-core/internal/services/purchase-service"
	"slices"

	"github.com/gin-gonic/gin"
)

func CreateInvoiceDN(ctx *gin.Context, jsonPayload string) (interface{}, error) {

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
	prefix := "DN"
	configCodeValue := "RUNNING_DN"
	count := 0
	for i := range req {
		if req[i].InvoiceCode == "" {
			count++
		}
	}
	var invoiceCodes []string
	if count > 0 {
		invoiceCodes, err = GenerateInvoiceCodes(ctx, count, prefix, configCodeValue)
		if err != nil {
			return nil, errors.New("failed to generate invoice codes: " + err.Error())
		}
	}
	codeIndex := 0
	productCodes := []string{}
	for i := range req {
		if req[i].InvoiceCode == "" {
			req[i].InvoiceCode = invoiceCodes[codeIndex]
			codeIndex++
		}
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
		}
		for it := range req[i].InvoiceItem {
			productCodes = append(productCodes, req[i].InvoiceItem[it].ProductCode)
		}
	}

	requestData := map[string]interface{}{
		"module":    []string{"INVOICE"},
		"topic":     []string{"DN"},
		"sub_topic": []string{"CREATE"},
	}

	hookConfig, err := interfaceService.GetHookConfig(requestData)
	if err != nil {
		return nil, err
	}
	if len(hookConfig) > 0 && req[0].Status != "TEMP" {
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

		productDebitReq := models.GetProductRequest{
			ProductType: []string{"DEBIT_NOTE"},
			SiteCode:    []string{req[0].SiteCode},
			CompanyCode: []string{req[0].CompanyCode},
		}

		mapProduct, errmapProduct := purchaseService.GetProductByCode(productDebitReq)
		if errmapProduct != nil {
			return nil, errors.New("failed to get product list: " + errmapProduct.Error())
		}
		firstProduct := models.GetProductsDetailComponent{}
		hasProduct := false
		for _, product := range mapProduct {
			firstProduct = product
			hasProduct = true
			break
		}

		reqHook := slices.Clone(req)
		for i := range reqHook {
			reqHook[i].InvoiceItem = slices.Clone(req[i].InvoiceItem)
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
				if hasProduct {
					reqHook[i].InvoiceItem[it].ProductCode = firstProduct.ProductCode
					reqHook[i].InvoiceItem[it].ProductName = firstProduct.ProductName
				}
			}
		}

		requestDataCreateHook := interfaceService.HookInterfaceRequest{
			RequestData: reqHook,
			UrlHook:     urlHook,
		}
		HookInterfaceValue, err := interfaceService.HookInterface(requestDataCreateHook)
		if err != nil {
			if req[0].Status == "COMPLETED" {
				req[0].Status = "TEMP"
				jsonBytesCreateInvoice, err := json.Marshal(req)
				if err != nil {
					return nil, err
				}
				_, errCreateInvoice := CreateInvoice(ctx, string(jsonBytesCreateInvoice))
				if errCreateInvoice != nil {
					return nil, errCreateInvoice
				}
			}
			return nil, err
		}
		if HookInterfaceValue != nil {
			externalID := HookInterfaceValue.(map[string]interface{})
			str, _ := externalID["id"].(string)
			req[0].ExternalID = str
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
