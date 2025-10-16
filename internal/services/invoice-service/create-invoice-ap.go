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

type POData struct {
	QTY    float64
	Weight float64
}

func CreateInvoiceAP(ctx *gin.Context, jsonPayload string) (interface{}, error) {

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
			keyConvert := fmt.Sprintf("%s|%d", poValue.PurchaseCode, poItemsValue.PurchaseItem)
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
	errData := map[string]interface{}{}
	for _, invoice := range req {
		for i, invoiceItem := range invoice.InvoiceItem {
			keyConvert := fmt.Sprintf("%s|%s", invoiceItem.DocumentRef, invoiceItem.PurchaseItem)
			poQTYMapResult, exist := poMap[keyConvert]
			if exist {
				poQTY := poQTYMapResult.QTY + (poQTYMapResult.QTY * tolerance / 100)
				if invoiceItem.Qty > poQTY {
					errData[strconv.Itoa(i)] = map[string]interface{}{
						"index":   i,
						"status":  "error",
						"message": "เกินจำนวนสูงสุด : " + strconv.FormatFloat(poQTY, 'f', -1, 64),
					}

				}
				if invoiceItem.Weight > 0 {
					if invoiceItem.Weight > poQTYMapResult.Weight {
						errData[strconv.Itoa(i)] = map[string]interface{}{
							"index":   i,
							"status":  "error",
							"message": "เกินน้ำหนักสูงสุด : " + strconv.FormatFloat(poQTYMapResult.Weight, 'f', -1, 64),
						}
					}
				}
			}
		}
	}
	if len(errData) == 0 {
		_, errCreateInvoice := CreateInvoice(ctx, jsonPayload)
		if errCreateInvoice != nil {
			return nil, errCreateInvoice
		}
	}

	return errData, nil
}
