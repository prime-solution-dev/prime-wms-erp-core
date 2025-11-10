package invoiceService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryInvoice "prime-erp-core/internal/repositories/invoice"
	prePurchaseService "prime-erp-core/internal/services/pre-purchase-service"
	purchaseService "prime-erp-core/internal/services/purchase-service"

	"github.com/gin-gonic/gin"
)

func SaleAutoStatusPayment(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetInvoiceRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	invoice, totalPages, totalRecords, errDeposit := repositoryInvoice.GetInvoicePreload(req.ID, req.InvoiceCode, req.InvoiceType, req.CustomerCode, req.Status, req.DocRef, req.Page, req.PageSize)
	if errDeposit != nil {
		return nil, errDeposit
	}
	supplierReq := models.GetSupplierListRequest{}
	productCodes := []string{}
	for _, invoiceValue := range invoice {
		supplierReq.SupplierCodes = append(supplierReq.SupplierCodes, invoiceValue.PartyCode)
		for _, invoiceItemValue := range invoiceValue.InvoiceItem {
			productCodes = append(productCodes, invoiceItemValue.ProductCode)
		}
	}
	mapSupplier, err := prePurchaseService.GetSupplierByCode(supplierReq)
	if err != nil {
		return nil, errors.New("failed to get supplier list: " + err.Error())
	}

	productReq := models.GetProductRequest{
		ProductCode: productCodes,
		SiteCode:    []string{req.SiteCode},
		CompanyCode: []string{req.CompanyCode},
	}

	mapProduct, err := purchaseService.GetProductByCode(productReq)
	if err != nil {
		return nil, errors.New("failed to get product list: " + err.Error())
	}

	// Get Product Group One

	for i := range invoice {
		if supplier, ok := mapSupplier[invoice[i].PartyCode]; ok {
			invoice[i].PartyName = supplier.SupplierName
		}
		for j := range invoice[i].InvoiceItem {
			if productDetail, ok := mapProduct[invoice[i].InvoiceItem[j].ProductCode]; ok {
				invoice[i].InvoiceItem[j].ProductName = productDetail.ProductName
				if len(productDetail.ProductGroups) > 0 {
					for _, productGroups := range productDetail.ProductGroups {
						if productGroups.GroupCode == "PRODUCT_GROUP1" {
							invoice[i].InvoiceItem[j].ProductGroup = productGroups.GroupValue
						}
					}

				}

			}
		}
	}

	resultInvoice := ResultInvoice{
		Total:      totalRecords,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		Invoice:    invoice,
	}

	return resultInvoice, nil
}
