package invoiceService

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	models "prime-erp-core/internal/models"
	systemConfigRepository "prime-erp-core/internal/repositories/systemConfig"
	interfaceService "prime-erp-core/internal/services/interface-service"
	prePurchaseService "prime-erp-core/internal/services/pre-purchase-service"
	purchaseService "prime-erp-core/internal/services/purchase-service"
	"strconv"

	"github.com/gin-gonic/gin"
)

func UpdateInvoiceAP(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Invoice

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}
	poNumber := []string{}
	companyCode := ""
	siteCode := ""
	supplierReq := models.GetSupplierListRequest{}
	for _, invoice := range req {
		for _, invoiceItem := range invoice.InvoiceItem {
			poNumber = append(poNumber, invoiceItem.DocumentRef)
			companyCode = invoice.CompanyCode
			siteCode = invoice.SiteCode
		}
		supplierReq.SupplierCodes = append(supplierReq.SupplierCodes, invoice.PartyCode)
	}
	requestDataGetPO := map[string]interface{}{
		"purchase_codes": poNumber,
		"company_code":   companyCode,
		"site_code":      siteCode,
	}

	jsonBytesGetPO, err := json.Marshal(requestDataGetPO)
	if err != nil {
		errors.New("Error marshalling data :")
	}
	po, errGetPO := purchaseService.GetPO(ctx, string(jsonBytesGetPO))
	if errGetPO != nil {
		return nil, errGetPO
	}
	poMap := map[string]POData{}
	for _, poValue := range po.(models.GetPurchaseResponse).DataList {
		for _, poItemsValue := range poValue.Items {
			keyConvert := fmt.Sprintf("%s|%s", poValue.PurchaseCode, poItemsValue.PurchaseItem)
			poMap[keyConvert] = POData{
				QTY:    poItemsValue.Qty,
				Weight: poItemsValue.TotalWeight,
			}
		}
	}

	topicCodes := []string{"INVOICE"}
	configCodes := []string{"AP"}

	invoiceConfigs, err := systemConfigRepository.GetSystemConfig(topicCodes, configCodes)
	if err != nil {
		return nil, err
	}
	invoiceConfigsMap := make(map[string]models.SystemConfig)
	tolerance := 0.0
	for _, invoiceConfigsValue := range invoiceConfigs {
		invoiceConfigsMap[fmt.Sprintf("%s|%s", invoiceConfigsValue.TopicCode, invoiceConfigsValue.ConfigCode)] = invoiceConfigsValue
		floatVal, err := strconv.ParseFloat(invoiceConfigsValue.Value, 64)
		if err != nil {
			log.Fatalf("Invalid float value: %v", err)
		}
		tolerance = floatVal
	}

	mapSupplier, errGetSupplierByCode := prePurchaseService.GetSupplierByCode(supplierReq)
	if errGetSupplierByCode != nil {
		return nil, errors.New("failed to get supplier list: " + errGetSupplierByCode.Error())
	}

	requestDataGetInvoice := map[string]interface{}{
		"status":                    []string{"COMPLETED"},
		"invoice_item_document_ref": poNumber,
	}

	jsonBytesGetInvoice, err := json.Marshal(requestDataGetInvoice)
	if err != nil {
		return nil, err
	}
	getInvoice, errCreateInvoice := GetInvoice(ctx, string(jsonBytesGetInvoice))
	if errCreateInvoice != nil {
		return nil, errCreateInvoice
	}
	resultInvoice := getInvoice.(ResultInvoice).Invoice
	resultInvoiceMap := map[string]POData{}
	for _, resultInvoiceValue := range resultInvoice {
		for _, resultInvoiceItemValue := range resultInvoiceValue.InvoiceItem {
			invoiceItemMapResult, exist := resultInvoiceMap[resultInvoiceItemValue.DocumentRefItem]
			if exist {
				resultInvoiceMap[resultInvoiceItemValue.DocumentRefItem] = POData{
					QTY:    invoiceItemMapResult.QTY + resultInvoiceItemValue.Qty,
					Weight: invoiceItemMapResult.Weight + resultInvoiceItemValue.Weight,
				}
			} else {
				resultInvoiceMap[resultInvoiceItemValue.DocumentRefItem] = POData{
					QTY:    resultInvoiceItemValue.Qty,
					Weight: resultInvoiceItemValue.Weight,
				}
			}
		}
	}

	toleranceErrorResponse := ToleranceErrorResponse{}
	completePOItem := []models.PurchaseItemUsed{}
	partialPOItem := []string{}
	for i, invoice := range req {
		if supplier, ok := mapSupplier[req[i].PartyCode]; ok {
			req[i].PartyName = supplier.SupplierName
			req[i].PartyBranch = supplier.Branch
			req[i].PartyAddress = supplier.Address
			req[i].PartyEmail = supplier.Email
			req[i].PartyTel = supplier.Phone
			req[i].PartyTaxID = supplier.TaxID
			req[i].PartyExternalID = supplier.ExternalID
		}
		for it, invoiceItem := range invoice.InvoiceItem {
			keyConvert := fmt.Sprintf("%s|%s", invoiceItem.DocumentRef, invoiceItem.PurchaseItem)
			poQTYMapResult, exist := poMap[keyConvert]
			if exist {
				poQTY := poQTYMapResult.QTY + (poQTYMapResult.QTY * tolerance / 100)
				invoiceItemMapResult, existInvoice := resultInvoiceMap[invoiceItem.DocumentRefItem]
				if existInvoice {
					invoiceItem.Qty += invoiceItemMapResult.QTY
					invoiceItem.Weight += invoiceItemMapResult.Weight
				}
				if invoiceItem.Qty > poQTY {

					toleranceErrorResponse.ToleranceError = append(toleranceErrorResponse.ToleranceError, ToleranceErrorItem{
						Index:   it,
						Message: "เกินจำนวนสูงสุด : " + strconv.FormatFloat(poQTY, 'f', -1, 64),
						Status:  "error",
						Type:    "qty",
					})

				}
				/* if invoiceItem.Qty == poQTYMapResult.QTY {

				} */
				completePOItem = append(completePOItem, models.PurchaseItemUsed{
					PurchaseItemCode: invoiceItem.DocumentRefItem,
					QTY:              invoiceItem.Qty,
				})
				if invoiceItem.Qty < poQTY {
					partialPOItem = append(partialPOItem, invoiceItem.DocumentRefItem)
				}
				if invoiceItem.Weight > 0 {
					if invoiceItem.Weight > poQTYMapResult.Weight {
						toleranceErrorResponse.ToleranceError = append(toleranceErrorResponse.ToleranceError, ToleranceErrorItem{
							Index:   it,
							Message: "เกินน้ำหนักสูงสุด :  " + strconv.FormatFloat(poQTYMapResult.Weight, 'f', -1, 64),
							Status:  "error",
							Type:    "weight",
						})
					}
				}
			} /*  else {

				toleranceErrorResponse.ToleranceError = append(toleranceErrorResponse.ToleranceError, ToleranceErrorItem{
					Index:   it,
					Message: "ไม่มี PO นี้ในระบบ",
					Status:  "error",
					Type:    "po",
				})
			} */
		}
	}
	if len(toleranceErrorResponse.ToleranceError) == 0 {

		if len(completePOItem) > 0 {
			requestDataGetPO := map[string]interface{}{
				"used_type":          "GR",
				"purchase_item_used": completePOItem,
			}

			jsonBytesGetPO, err := json.Marshal(requestDataGetPO)
			if err != nil {
				errors.New("Error marshalling data :")
			}
			_, errCompletePOItem := purchaseService.CompletePOItem(ctx, string(jsonBytesGetPO))
			if errCompletePOItem != nil {
				return nil, errCompletePOItem
			}
		}

		jsonBytesCreateInvoice, err := json.Marshal(req)
		if err != nil {
			return nil, err
		}
		createInvoiceReturn, errCreateInvoice := UpdateInvoice(ctx, string(jsonBytesCreateInvoice))
		if errCreateInvoice != nil {
			return nil, errCreateInvoice
		}
		requestData := map[string]interface{}{
			"module":    []string{"INVOICE"},
			"topic":     []string{"AP"},
			"sub_topic": []string{"UPDATE"},
		}

		hookConfig, err := interfaceService.GetHookConfig(requestData)
		if err != nil {
			return nil, err
		}
		if len(hookConfig) > 0 {
			urlProduct := ""
			for _, hookConfigValue := range hookConfig {
				urlProduct = hookConfigValue.HookUrl
			}

			requestDataCreateHook := interfaceService.HookInterfaceRequest{
				RequestData: req,
				UrlHook:     urlProduct,
			}
			_, err := interfaceService.HookInterface(requestDataCreateHook)
			if err != nil {
				return nil, err
			}
		}

		return createInvoiceReturn, nil
	}

	return toleranceErrorResponse, nil
}
