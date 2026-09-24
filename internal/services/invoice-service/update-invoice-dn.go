package invoiceService

import (
	"encoding/json"
	"errors"
	"math"
	models "prime-erp-core/internal/models"
	repositoryInvoice "prime-erp-core/internal/repositories/invoice"
	interfaceService "prime-erp-core/internal/services/interface-service"
	purchaseService "prime-erp-core/internal/services/purchase-service"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func UpdateInvoiceDN(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Invoice

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	jsonBytesCreateInvoice, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	if len(req) == 0 {
		return nil, errors.New("invoice is required")
	}
	ids := make([]uuid.UUID, 0, len(req))
	seen := make(map[uuid.UUID]bool, len(req))
	for _, invoice := range req {
		if invoice.ID == uuid.Nil || seen[invoice.ID] {
			return nil, errors.New("invoice IDs must be non-empty and unique")
		}
		seen[invoice.ID] = true
		ids = append(ids, invoice.ID)
	}
	getPayload, err := json.Marshal(GetInvoiceRequest{ID: ids})
	if err != nil {
		return nil, err
	}
	value, err := GetInvoice(ctx, string(getPayload))
	if err != nil {
		return nil, err
	}

	stored, ok := value.(ResultInvoice)
	if !ok {
		return nil, errors.New("invalid GetInvoice response")
	}
	statuses := make(map[uuid.UUID]string, len(stored.Invoice))
	for _, invoice := range stored.Invoice {
		statuses[invoice.ID] = invoice.Status
	}
	tempIDs := make([]uuid.UUID, 0, len(req))
	for _, invoice := range req {
		status, exists := statuses[invoice.ID]
		if !exists {
			return nil, errors.New("invoice not found: " + invoice.ID.String())
		}
		if strings.EqualFold(status, "TEMP") {
			tempIDs = append(tempIDs, invoice.ID)
		}
	}
	if len(tempIDs) == len(req) {
		if err := repositoryInvoice.DeleteInvoice(tempIDs); err != nil {
			return nil, err
		}
		return CreateInvoiceDN(ctx, jsonPayload)
	}

	requestData := map[string]interface{}{
		"module":    []string{"INVOICE"},
		"topic":     []string{"DN"},
		"sub_topic": []string{"UPDATE"},
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
		reqHook := req
		productCodes := []string{}
		for i := range reqHook {
			for it := range reqHook[i].InvoiceItem {
				productCodes = append(productCodes, reqHook[i].InvoiceItem[it].ProductCode)
			}
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
			}
		}

		requestDataCreateHook := interfaceService.HookInterfaceRequest{
			RequestData: req,
			UrlHook:     urlHook,
		}
		_, err := interfaceService.HookInterface(requestDataCreateHook)
		if err != nil {
			return nil, err
		}
	}

	createInvoiceReturn, errCreateInvoice := UpdateInvoice(ctx, string(jsonBytesCreateInvoice))
	if errCreateInvoice != nil {
		return nil, errCreateInvoice
	}

	return createInvoiceReturn, nil

}
