package invoiceService

import (
	"encoding/json"
	"errors"
	"fmt"
	models "prime-erp-core/internal/models"
	purchaseService "prime-erp-core/internal/services/purchase-service"

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
	for _, invoice := range req {
		poNumber = append(poNumber, invoice.DocumentRef)
	}
	requestDataGetPO := map[string][]string{
		"purchase_codes": poNumber,
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

	for _, invoice := range req {
		for _, invoiceItem := range invoice.InvoiceItem {
			keyConvert := fmt.Sprintf("%s|%s", invoiceItem.DocumentRef, invoiceItem.PurchaseItem)
			poQTYMapResult, exist := poMap[keyConvert]
			if exist {
				if invoiceItem.Qty > poQTYMapResult.QTY {
					return nil, errors.New("Invoice Qty over PO Qty : " + invoiceItem.InvoiceCode + " Item : " + invoiceItem.InvoiceItem)
				}
				if invoiceItem.TotalWeight > 0 {
					if invoiceItem.TotalWeight > poQTYMapResult.Weight {
						return nil, errors.New("Invoice Weight over PO Weight : " + invoiceItem.InvoiceCode + " Item : " + invoiceItem.InvoiceItem)
					}
				}
			}
		}
	}

	return nil, nil
}
