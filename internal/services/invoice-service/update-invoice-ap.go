package invoiceService

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	models "prime-erp-core/internal/models"
	interfaceService "prime-erp-core/internal/services/interface-service"
	prePurchaseService "prime-erp-core/internal/services/pre-purchase-service"
	purchaseService "prime-erp-core/internal/services/purchase-service"
	xService "prime-erp-core/internal/services/x-service"
	"slices"
	"strings"

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
				QTY:          poItemsValue.Qty,
				Weight:       poItemsValue.TotalWeight,
				PurchaseUnit: poItemsValue.PurchaseUnit,
			}
		}
	}

	validateRequest := xService.ValidateAPOverPurchaseRequest{}
	for _, invoice := range req {
		for _, invoiceItem := range invoice.InvoiceItem {
			key := fmt.Sprintf("%s|%s", invoiceItem.DocumentRef, invoiceItem.DocumentRefItem)
			validateUnit := ""
			if poItem, ok := poMap[key]; ok {
				switch strings.ToUpper(strings.TrimSpace(poItem.PurchaseUnit)) {
				case "KG":
					validateUnit = "WEIGHT"
				default:
					validateUnit = "UNIT"
				}
				validateRequest.Datas = append(validateRequest.Datas, xService.ValidateAPOverPurchaseRequestData{
					PurchaseCode: invoiceItem.DocumentRef,
					PurchaseItem: invoiceItem.DocumentRefItem,
					Qty:          invoiceItem.Qty,
					TotalWeight:  invoiceItem.Weight,
					ValidateUnit: validateUnit,
				})
			}

		}
	}
	toleranceErrorResponse := ToleranceErrorResponse{}
	if len(validateRequest.Datas) > 0 {
		validateRequest.InvoiceCode = req[0].InvoiceCode
		validatePayload, err := json.Marshal(validateRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal AP over-purchase validation request: %w", err)
		}
		fmt.Println(string(validatePayload))
		validateResult, err := xService.ValidateAPOverPurchaseRest(ctx, string(validatePayload))
		if err != nil {
			return nil, err
		}
		validateResponse, ok := validateResult.(*xService.ValidateAPOverPurchaseResponse)
		if !ok {
			return nil, errors.New("invalid AP over-purchase validation response")
		}
		for _, validation := range validateResponse.Datas {
			if validation.Status == "ERROR" {
				errorType := strings.ToLower(validation.ValidateUnit)
				if validation.ValidateUnit == "UNIT" {
					errorType = "qty"
				}
				toleranceErrorResponse.ToleranceError = append(toleranceErrorResponse.ToleranceError, ToleranceErrorItem{
					Index:   validation.Index,
					Message: validation.Message,
					Status:  "error",
					Type:    errorType,
				})
			}
		}
		if len(toleranceErrorResponse.ToleranceError) > 0 {
			return toleranceErrorResponse, nil
		}
	}
	mapSupplier, errGetSupplierByCode := prePurchaseService.GetSupplierByCode(supplierReq)
	if errGetSupplierByCode != nil {
		return nil, errors.New("failed to get supplier list: " + errGetSupplierByCode.Error())
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
		for it := range invoice.InvoiceItem {
			req[i].InvoiceItem[it].PriceUnit = round2(req[i].InvoiceItem[it].PriceUnit)
			req[i].InvoiceItem[it].Qty = round2(req[i].InvoiceItem[it].Qty)
			req[i].InvoiceItem[it].TotalVat = round2(req[i].InvoiceItem[it].TotalVat)
			req[i].InvoiceItem[it].TotalDiscount = round2(req[i].InvoiceItem[it].TotalDiscount)
		}
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

		productReq := models.GetProductRequest{
			ProductType: []string{"PROD_SERVICE"},
			SiteCode:    []string{siteCode},
			CompanyCode: []string{companyCode},
		}

		mapProduct, errmapProduct := purchaseService.GetProductByCode(productReq)
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
		// Keep hook-only product substitutions separate from the persisted request.
		hookReq := slices.Clone(req)
		if hasProduct {
			for r := range hookReq {
				if hookReq[r].DocumentRefType == "FABRICATION" {
					hookReq[r].InvoiceItem = slices.Clone(hookReq[r].InvoiceItem)
					for it := range hookReq[r].InvoiceItem {
						hookReq[r].InvoiceItem[it].ProductCode = firstProduct.ProductCode
						hookReq[r].InvoiceItem[it].ProductName = firstProduct.ProductName
					}
				}
			}
		}

		requestDataCreateHook := interfaceService.HookInterfaceRequest{
			RequestData: hookReq,
			UrlHook:     urlProduct,
		}
		_, err := interfaceService.HookInterface(requestDataCreateHook)
		if err != nil {
			return nil, err
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

	// Auto-close PO from AP: after the GRA is persisted, reconcile every referenced
	// PO from the cumulative COMPLETED-AP state (product-master tolerance per unit_uom).
	// The invoice is already saved here; a reconcile failure must NOT fail the request
	// (a 5xx after save would invite a duplicate GRA on retry). Log and continue — the
	// next GRA on this PO, or a manual reconcile, self-heals.
	if err := reconcilePOAfterAPSave(req); err != nil {
		log.Printf("UpdateInvoiceAP: reconcilePOAfterAPSave failed (invoice saved, PO not closed): %v", err)
	}

	return createInvoiceReturn, nil
}
