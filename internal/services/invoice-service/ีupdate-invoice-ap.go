package invoiceService

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	models "prime-erp-core/internal/models"
	systemConfigRepository "prime-erp-core/internal/repositories/systemConfig"
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
	for _, invoice := range req {
		for _, invoiceItem := range invoice.InvoiceItem {
			poNumber = append(poNumber, invoiceItem.DocumentRef)
			companyCode = invoice.CompanyCode
			siteCode = invoice.SiteCode
		}

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
	for _, invoice := range req {
		for i, invoiceItem := range invoice.InvoiceItem {
			keyConvert := fmt.Sprintf("%s|%s", invoiceItem.DocumentRef, invoiceItem.PurchaseItem)
			poQTYMapResult, exist := poMap[keyConvert]
			if exist {
				poQTY := poQTYMapResult.QTY + (poQTYMapResult.QTY * tolerance / 100)
				if invoiceItem.Qty > poQTY {

					toleranceErrorResponse.ToleranceError = append(toleranceErrorResponse.ToleranceError, ToleranceErrorItem{
						Index:   i,
						Message: "เกินจำนวนสูงสุด : " + strconv.FormatFloat(poQTY, 'f', -1, 64),
						Status:  "error",
						Type:    "qty",
					})

				}
				if invoiceItem.Weight > 0 {
					if invoiceItem.Weight > poQTYMapResult.Weight {
						toleranceErrorResponse.ToleranceError = append(toleranceErrorResponse.ToleranceError, ToleranceErrorItem{
							Index:   i,
							Message: "เกินน้ำหนักสูงสุด : " + strconv.FormatFloat(poQTYMapResult.Weight, 'f', -1, 64),
							Status:  "error",
							Type:    "weight",
						})
					}
				}
			} else {

				toleranceErrorResponse.ToleranceError = append(toleranceErrorResponse.ToleranceError, ToleranceErrorItem{
					Index:   i,
					Message: "ไม่มี PO นี้ในระบบ",
					Status:  "error",
					Type:    "po",
				})
			}
		}
	}
	if len(toleranceErrorResponse.ToleranceError) == 0 {
		createInvoiceReturn, errCreateInvoice := UpdateInvoice(ctx, jsonPayload)
		if errCreateInvoice != nil {
			return nil, errCreateInvoice
		} else {
			return createInvoiceReturn, nil
		}
	}

	return toleranceErrorResponse, nil
}
