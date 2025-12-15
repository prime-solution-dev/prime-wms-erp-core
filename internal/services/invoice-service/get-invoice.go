package invoiceService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryInvoice "prime-erp-core/internal/repositories/invoice"
	paymentService "prime-erp-core/internal/services/payment-service"
	prePurchaseService "prime-erp-core/internal/services/pre-purchase-service"
	purchaseService "prime-erp-core/internal/services/purchase-service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetInvoiceRequest struct {
	ID                []uuid.UUID `json:"id"`
	InvoiceCode       []string    `json:"invoice_code"`
	InvoiceRef        []string    `json:"invoice_ref"`
	InvoiceType       []string    `json:"invoice_type"`
	CustomerCode      []string    `json:"customer_code"`
	Status            []string    `json:"status"`
	DocRef            []string    `json:"document_ref"`
	InvoiceItemDocRef []string    `json:"invoice_item_document_ref"`
	CompanyCode       string      `json:"company_code"`
	SiteCode          string      `json:"site_code"`
	Page              int         `json:"page"`
	PageSize          int         `json:"page_size"`
}
type ResultInvoice struct {
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
	Invoice    []models.Invoice `json:"invoice"`
}

func GetInvoice(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetInvoiceRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	invoice, totalPages, totalRecords, errDeposit := repositoryInvoice.GetInvoicePreload(req.ID, req.InvoiceCode, req.InvoiceType, req.CustomerCode, req.Status, req.DocRef, req.InvoiceRef, req.InvoiceItemDocRef, req.Page, req.PageSize)
	if errDeposit != nil {
		return nil, errDeposit
	}
	supplierReq := models.GetSupplierListRequest{}
	productCodes := []string{}
	invoiceCode := []string{}
	for _, invoiceValue := range invoice {
		supplierReq.SupplierCodes = append(supplierReq.SupplierCodes, invoiceValue.PartyCode)
		for _, invoiceItemValue := range invoiceValue.InvoiceItem {
			productCodes = append(productCodes, invoiceItemValue.ProductCode)
		}
		invoiceCode = append(invoiceCode, invoiceValue.InvoiceCode)
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

	requestDataGetPayment := map[string]interface{}{
		"invoice_code": invoiceCode,
	}

	jsonBytesPayment, err := json.Marshal(requestDataGetPayment)
	if err != nil {
		return nil, err
	}

	payment, errGetPayment := paymentService.GetPayment(ctx, string(jsonBytesPayment))
	if errGetPayment != nil {
		return nil, errGetPayment
	}
	resultPayment := payment.(paymentService.ResultPayment).Payment
	paymentValueMap := map[string]float64{}

	for _, paymentValue := range resultPayment {
		for _, paymentInvoiceValue := range paymentValue.PaymentInvoice {

			paymentItemMap, exist := paymentValueMap[paymentInvoiceValue.InvoiceCode]
			if exist {
				paymentValueMap[paymentInvoiceValue.InvoiceCode] = paymentItemMap + paymentInvoiceValue.Amount
			} else {
				paymentValueMap[paymentInvoiceValue.InvoiceCode] = paymentInvoiceValue.Amount
			}

		}

	}

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
		paymentItemMap, exist := paymentValueMap[invoice[i].InvoiceCode]
		if exist {
			if invoice[i].TotalAmount == paymentItemMap {
				invoice[i].PaymentStatus = "Paid"
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
