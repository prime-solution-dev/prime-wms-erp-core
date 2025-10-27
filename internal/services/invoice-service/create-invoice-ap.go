package invoiceService

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	models "prime-erp-core/internal/models"
	repositoryInvoice "prime-erp-core/internal/repositories/invoice"
	systemConfigRepository "prime-erp-core/internal/repositories/systemConfig"
	interfaceService "prime-erp-core/internal/services/interface-service"
	prePurchaseService "prime-erp-core/internal/services/pre-purchase-service"
	purchaseService "prime-erp-core/internal/services/purchase-service"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type POData struct {
	QTY    float64
	Weight float64
}

type ToleranceErrorItem struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
	Status  string `json:"status"`
	Type    string `json:"type"`
}

type ToleranceErrorResponse struct {
	ToleranceError []ToleranceErrorItem `json:"tolerance_error"`
}

func CreateInvoiceAP(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Invoice

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}
	poNumber := []string{}
	companyCode := ""
	siteCode := ""
	supplierReq := models.GetSupplierListRequest{}
	productCodes := []string{}

	for _, invoice := range req {
		for _, invoiceItem := range invoice.InvoiceItem {
			poNumber = append(poNumber, invoiceItem.DocumentRef)
			companyCode = invoice.CompanyCode
			siteCode = invoice.SiteCode
			productCodes = append(productCodes, invoiceItem.ProductCode)
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
	toleranceErrorResponse := ToleranceErrorResponse{}

	mapSupplier, errGetSupplierByCode := prePurchaseService.GetSupplierByCode(supplierReq)
	if errGetSupplierByCode != nil {
		return nil, errors.New("failed to get supplier list: " + errGetSupplierByCode.Error())
	}

	productReq := models.GetProductRequest{
		ProductCode: productCodes,
		SiteCode:    []string{siteCode},
		CompanyCode: []string{companyCode},
	}

	mapProduct, errmapProduct := purchaseService.GetProductByCode(productReq)
	if errmapProduct != nil {
		return nil, errors.New("failed to get product list: " + errmapProduct.Error())
	}

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
			req[i].InvoiceItem[it].ProductName = mapProduct[req[i].InvoiceItem[it].ProductCode].ProductName
			keyConvert := fmt.Sprintf("%s|%s", invoiceItem.DocumentRef, invoiceItem.PurchaseItem)
			poQTYMapResult, exist := poMap[keyConvert]
			if exist {
				poQTY := poQTYMapResult.QTY + (poQTYMapResult.QTY * tolerance / 100)
				if invoiceItem.Qty > poQTY {

					toleranceErrorResponse.ToleranceError = append(toleranceErrorResponse.ToleranceError, ToleranceErrorItem{
						Index:   it,
						Message: "เกินจำนวนสูงสุด : " + strconv.FormatFloat(poQTY, 'f', -1, 64),
						Status:  "error",
						Type:    "qty",
					})

				}
				if invoiceItem.Weight > 0 {
					if invoiceItem.Weight > poQTYMapResult.Weight {
						toleranceErrorResponse.ToleranceError = append(toleranceErrorResponse.ToleranceError, ToleranceErrorItem{
							Index:   it,
							Message: "เกินน้ำหนักสูงสุด : " + strconv.FormatFloat(poQTYMapResult.Weight, 'f', -1, 64),
							Status:  "error",
							Type:    "weight",
						})
					}
				}
			} /*  else {

				toleranceErrorResponse.ToleranceError = append(toleranceErrorResponse.ToleranceError, ToleranceErrorItem{
					Index:   i,
					Message: "ไม่มี PO นี้ในระบบ",
					Status:  "error",
					Type:    "po",
				})
			} */
		}

	}
	if len(toleranceErrorResponse.ToleranceError) == 0 {
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
			"topic":     []string{"AP"},
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

	return toleranceErrorResponse, nil
}
