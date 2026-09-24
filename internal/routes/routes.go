package routes

import (
	"prime-erp-core/internal/utils"

	approvalService "prime-erp-core/internal/services/approval-service"
	creditService "prime-erp-core/internal/services/credit-service"
	CronjobService "prime-erp-core/internal/services/cronjob-service"
	depositService "prime-erp-core/internal/services/deposit-service"
	emailservice "prime-erp-core/internal/services/email-service"
	groupService "prime-erp-core/internal/services/group-service"
	invoiceService "prime-erp-core/internal/services/invoice-service"
	paymentService "prime-erp-core/internal/services/payment-service"
	prePurchaseService "prime-erp-core/internal/services/pre-purchase-service"
	priceService "prime-erp-core/internal/services/price-service"
	purchaseService "prime-erp-core/internal/services/purchase-service"
	systemConfigService "prime-erp-core/internal/services/system-config"
	xService "prime-erp-core/internal/services/x-service"

	deliveryService "prime-erp-core/internal/services/delivery-service"
	quotationService "prime-erp-core/internal/services/quotation-service"
	saleService "prime-erp-core/internal/services/sale-service"
	summaryService "prime-erp-core/internal/services/summary-credit"
	timeService "prime-erp-core/internal/services/time-service"
	unitService "prime-erp-core/internal/services/unit-service"
	verifyService "prime-erp-core/internal/services/verify-service"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(ctx *gin.Engine) {
	//group
	group := ctx.Group("/group")

	group.POST("/GetGroupMaster", func(c *gin.Context) {
		utils.ProcessContextRequest(c, groupService.GetGroup)
	})
	group.POST("/SyncGroupMaster", func(c *gin.Context) {
		utils.ProcessRequest(c, groupService.SyncGroupMaster)
	})

	//price
	price := ctx.Group("/price")

	price.POST("/GetPriceListGroup", func(c *gin.Context) {
		utils.ProcessContextRequest(c, priceService.GetPriceListGroup)
	})
	price.POST("/GetPaymentTerm", func(c *gin.Context) {
		utils.ProcessContextRequest(c, priceService.GetPaymentTerm)
	})
	price.POST("/GetComparePrice", func(c *gin.Context) {
		utils.ProcessContextRequest(c, priceService.GetComparePrice)
	})
	price.POST("/GetPriceList", func(c *gin.Context) {
		utils.ProcessContextRequest(c, priceService.GetPriceList)
	}) // for Base Price and price list feature
	price.POST("/CreatePriceListGroupBase", func(c *gin.Context) {
		utils.ProcessContextRequest(c, priceService.CreatePriceListBase)
	})
	price.POST("/UpdatePriceListGroupBase", func(c *gin.Context) {
		utils.ProcessContextRequest(c, priceService.UpdatePriceListBase)
	})
	price.POST("/UpdatePriceListSubGroup", func(c *gin.Context) {
		utils.ProcessContextRequest(c, priceService.UpdatePriceListSubGroup)
	})
	price.POST("/DeletePriceListGroupBase", func(c *gin.Context) {
		utils.ProcessContextRequest(c, priceService.DeletePriceListBase)
	})
	price.POST("/GetPriceDetail", func(c *gin.Context) {
		utils.ProcessContextRequest(c, priceService.GetPriceDetail)
	})
	price.POST("/GetPriceExportTable", func(c *gin.Context) {
		utils.ProcessContextRequest(c, priceService.GetPriceExportTable)
	})
	price.POST("/SubGroup/UpdateLatest", func(c *gin.Context) {
		utils.ProcessContextRequest(c, priceService.UpdateLatestPriceListSubGroup)
	})
	price.POST("/SubGroup/GetCalculated", func(c *gin.Context) {
		utils.ProcessContextRequest(c, priceService.GetCalculatedPriceListSubGroup)
	})
	price.POST("/UpdatePriceListExtra", func(c *gin.Context) {
		utils.ProcessContextRequest(c, priceService.UpdateExtras)
	})
	price.POST("/UploadPriceList", func(c *gin.Context) {
		utils.ProcessContextRequestMultipart(c, priceService.UploadPricelistMultipart)
	})
	price.POST("/UploadPriceListTemplate", func(c *gin.Context) {
		utils.ProcessContextRequestMultipart(c, priceService.UploadPricelistTemplateMultipart)
	})
	// config extra get[3] create[2] update delete
	// extra create update delete [4]

	config := ctx.Group("/config")
	config.POST("/GetSystemConfig", func(c *gin.Context) {
		utils.ProcessRequest(c, systemConfigService.GetSystemConfig)
	})

	//quotation
	quotation := ctx.Group("/quotation")

	quotation.POST("/GetQuotation", func(c *gin.Context) {
		utils.ProcessContextRequest(c, quotationService.GetQuotation)
	})
	quotation.POST("/CreateQuotation", func(c *gin.Context) {
		utils.ProcessContextRequest(c, quotationService.CreateQuotation)
	})
	quotation.POST("/UpdateQuotation", func(c *gin.Context) {
		utils.ProcessContextRequest(c, quotationService.UpdateQuotation)
	})
	quotation.POST("/EditQuotation", func(c *gin.Context) {
		utils.ProcessContextRequest(c, quotationService.EditQuotation)
	})
	quotation.POST("/CancelQuotation", func(c *gin.Context) {
		utils.ProcessContextRequest(c, quotationService.CancelQuotation)
	})
	quotation.POST("/ReviseQuotation", func(c *gin.Context) {
		utils.ProcessContextRequest(c, quotationService.ReviseQuotation)
	})

	quotation.POST("/RequestApproveQuotation", func(c *gin.Context) {
		utils.ProcessContextRequest(c, quotationService.RequestApproveQuotation)
	})
	quotation.POST("/UpdateStatusApproveQuotation", func(c *gin.Context) {
		utils.ProcessContextRequest(c, quotationService.UpdateStatusApproveQuotation)
	})
	//invoice
	invoice := ctx.Group("/invoice")
	invoice.POST("/GetInvoice", func(c *gin.Context) {
		utils.ProcessContextRequest(c, invoiceService.GetInvoice)
	})
	invoice.POST("/CreateInvoice", func(c *gin.Context) {
		utils.ProcessContextRequest(c, invoiceService.CreateInvoice)
	})
	invoice.POST("/UpdateInvoice", func(c *gin.Context) {
		utils.ProcessContextRequest(c, invoiceService.UpdateInvoice)
	})
	invoice.POST("/CreateInvoiceAP", func(c *gin.Context) {
		utils.ProcessContextRequest(c, invoiceService.CreateInvoiceAP)
	})
	invoice.POST("/UpdateInvoiceAP", func(c *gin.Context) {
		utils.ProcessContextRequest(c, invoiceService.UpdateInvoiceAP)
	})
	invoice.POST("/CreateInvoiceAR", func(c *gin.Context) {
		utils.ProcessContextRequest(c, invoiceService.CreateInvoiceAR)
	})
	invoice.POST("/UpdateInvoiceAR", func(c *gin.Context) {
		utils.ProcessContextRequest(c, invoiceService.UpdateInvoiceAR)
	})
	invoice.POST("/CreateInvoiceCN", func(c *gin.Context) {
		utils.ProcessContextRequest(c, invoiceService.CreateInvoiceCN)
	})
	invoice.POST("/UpdateInvoiceCN", func(c *gin.Context) {
		utils.ProcessContextRequest(c, invoiceService.UpdateInvoiceCN)
	})
	invoice.POST("/CreateInvoiceDN", func(c *gin.Context) {
		utils.ProcessContextRequest(c, invoiceService.CreateInvoiceDN)
	})
	invoice.POST("/UpdateInvoiceDN", func(c *gin.Context) {
		utils.ProcessContextRequest(c, invoiceService.UpdateInvoiceDN)
	})
	//payment
	payment := ctx.Group("/payment")
	payment.POST("/GetPayment", func(c *gin.Context) {
		utils.ProcessRequest(c, paymentService.GetPayment)
	})
	payment.POST("/CreatePayment", func(c *gin.Context) {
		utils.ProcessRequest(c, paymentService.CreatePayment)
	})
	payment.POST("/DeletePayment", func(c *gin.Context) {
		utils.ProcessRequest(c, paymentService.DeletePayment)
	})

	//sale
	sale := ctx.Group("/sale")
	sale.POST("/CreateSale", func(c *gin.Context) {
		utils.ProcessContextRequest(c, saleService.CreateSale)
	})
	sale.POST("/UpdateSaleStatusPayment", func(c *gin.Context) {
		utils.ProcessContextRequest(c, saleService.UpdateSaleStatusPayment)
	})
	sale.POST("/UpdateStatusSale", func(c *gin.Context) {
		utils.ProcessContextRequest(c, saleService.UpdateStatusSale)
	})

	sale.POST("/EditSale", func(c *gin.Context) {
		utils.ProcessContextRequest(c, saleService.EditSale)
	})
	sale.POST("/GetSale", func(c *gin.Context) {
		utils.ProcessContextRequest(c, saleService.GetSale)
	})
	sale.POST("/UpdateSale", func(c *gin.Context) {
		utils.ProcessContextRequest(c, saleService.UpdateSale)
	})
	sale.POST("/RequestApproveSale", func(c *gin.Context) {
		utils.ProcessContextRequest(c, saleService.RequestApproveSale)
	})
	sale.POST("/UpdateStatusApproveSale", func(c *gin.Context) {
		utils.ProcessContextRequest(c, saleService.UpdateStatusApproveSale)
	})

	sale.POST("/GetSalePack", func(c *gin.Context) {
		utils.ProcessContextRequest(c, saleService.GetSalePack)
	})

	sale.POST("/ValidateSaleOrder", func(c *gin.Context) {
		utils.ProcessContextRequest(c, saleService.ValidateSale)
	})

	sale.POST("/UpdateSaleItemStatus", func(c *gin.Context) {
		utils.ProcessContextRequest(c, saleService.UpdateSaleItemStatus)
	})
	//delivery
	delivery := ctx.Group("/delivery")
	delivery.POST("/CreateDelivery", func(c *gin.Context) {
		utils.ProcessContextRequest(c, deliveryService.CreateDelivery)
	})
	delivery.POST("/GetDelivery", func(c *gin.Context) {
		utils.ProcessContextRequest(c, deliveryService.GetDelivery)
	})
	delivery.POST("/UpdateDelivery", func(c *gin.Context) {
		utils.ProcessContextRequest(c, deliveryService.UpdateDelivery)
	})
	delivery.POST("/UpdateStatusDelivery", func(c *gin.Context) {
		utils.ProcessContextRequest(c, deliveryService.UpdateStatusDelivery)
	})
	delivery.POST("/GetDeliveryCO", func(c *gin.Context) {
		utils.ProcessContextRequest(c, deliveryService.GetDeliveryCO)
	})
	/* 	delivery.POST("/GetDeliverySO", func(c *gin.Context) {
	   		utils.ProcessContextRequest(c, deliveryService.GetDeliverySO)
	   	})
	*/
	//time
	time := ctx.Group("/time")
	time.POST("/GetTime", func(c *gin.Context) {
		utils.ProcessRequest(c, timeService.GetTime)
	})
	//deposit
	deposit := ctx.Group("/deposit")
	deposit.POST("/GetDeposit", func(c *gin.Context) {
		utils.ProcessRequest(c, depositService.GetDeposit)
	})
	deposit.POST("/CreateDepost", func(c *gin.Context) {
		utils.ProcessRequest(c, depositService.CreateDepost)
	})

	//approval
	approval := ctx.Group("/approval")
	approval.POST("/VerifyApprove", func(c *gin.Context) {
		utils.ProcessRequest(c, verifyService.VerifyApprove)
	})
	approval.POST("/GetApproval", func(c *gin.Context) {
		utils.ProcessContextRequest(c, approvalService.GetApproval)
	})
	approval.POST("/CreateApproval", func(c *gin.Context) {
		utils.ProcessContextRequest(c, approvalService.CreateApproval)
	})
	approval.POST("/UpdateApproval", func(c *gin.Context) {
		utils.ProcessContextRequest(c, approvalService.UpdateApproval)
	})
	approval.POST("/CheckAutoApprovalRest", func(c *gin.Context) {
		utils.ProcessRequest(c, approvalService.CheckAutoApprovalRest)
	})

	//credit
	credit := ctx.Group("/credit")
	credit.POST("/GetCreditCurrent", func(c *gin.Context) {
		utils.ProcessContextRequest(c, creditService.GetCreditCurrentAPI)
	})
	credit.POST("/GetCreditRequest", func(c *gin.Context) {
		utils.ProcessContextRequest(c, creditService.GetCreditRequests)
	})
	credit.POST("/GetCreditRequestCronjob", func(c *gin.Context) {
		utils.ProcessContextRequest(c, creditService.GetCreditRequestCronjob)
	})
	credit.POST("/GetCustomerCredit", func(c *gin.Context) {
		utils.ProcessContextRequest(c, creditService.GetCustomerCreditRest)
	})

	credit.POST("/CreateCreditRequest", func(c *gin.Context) {
		utils.ProcessContextRequest(c, creditService.CreateCreditRequest)
	})
	credit.POST("/UpdateCreditRequest", func(c *gin.Context) {
		utils.ProcessContextRequest(c, creditService.UpdateCreditRequest)
	})
	credit.POST("/GetCredit", func(c *gin.Context) {
		utils.ProcessContextRequest(c, creditService.GetCredit)
	})
	credit.POST("/CreateCredit", func(c *gin.Context) {
		utils.ProcessContextRequest(c, creditService.CreateCredit)
	})
	credit.POST("/GetHistory", func(c *gin.Context) {
		utils.ProcessContextRequest(c, creditService.GetHistory)
	})
	credit.POST("/GetSummaryCredit", func(c *gin.Context) {
		utils.ProcessContextRequest(c, creditService.GetSummaryCredit)
	})
	credit.POST("/GetTransaction", func(c *gin.Context) {
		utils.ProcessContextRequest(c, creditService.GetTransaction)
	})
	credit.POST("/CreateCreditTransaction", func(c *gin.Context) {
		utils.ProcessContextRequest(c, creditService.CreateCreditTransaction)
	})
	credit.POST("/DeleteCreditExtra", func(c *gin.Context) {
		utils.ProcessContextRequest(c, creditService.DeleteCreditExtra)
	})

	//summaryService
	summary := ctx.Group("/summary")
	summary.POST("/GetConsumend", func(c *gin.Context) {
		utils.ProcessContextRequest(c, summaryService.GetConsumend)
	})

	summary.POST("/GetOutStandingSo", func(c *gin.Context) {
		utils.ProcessContextRequest(c, summaryService.GetOutStandingSo)
	})

	//unit
	unit := ctx.Group("/unit")
	unit.POST("/GetAllUnit", func(c *gin.Context) {
		utils.ProcessRequest(c, unitService.GetAllUnit)
	})

	purchase := ctx.Group("/purchase")
	//pre-purchase
	purchase.POST("/CreatePOBigLot", func(c *gin.Context) {
		utils.ProcessContextRequest(c, prePurchaseService.CreatePOBigLot)
	})
	purchase.POST("/GetPOBigLot", func(c *gin.Context) {
		utils.ProcessContextRequest(c, prePurchaseService.GetPOBigLot)
	})
	purchase.POST("/UpdatePOBigLot", func(c *gin.Context) {
		utils.ProcessContextRequest(c, prePurchaseService.UpdatePOBigLot)
	})
	purchase.POST("/UpdateStatusApprovePOBigLot", func(c *gin.Context) {
		utils.ProcessContextRequest(c, prePurchaseService.UpdateStatusApprovePOBigLot)
	})
	purchase.POST("/CompletePOBigLot", func(c *gin.Context) {
		utils.ProcessContextRequest(c, prePurchaseService.CompletePOBigLot)
	})
	purchase.POST("/CancelPOBigLot", func(c *gin.Context) {
		utils.ProcessContextRequest(c, prePurchaseService.CancelPOBigLot)
	})
	purchase.POST("/GetPurchaseItemRemain", func(c *gin.Context) {
		utils.ProcessRequest(c, xService.GetPurchaseItemRemainRest)
	})
	purchase.POST("/ValidateAPOverPurchase", func(c *gin.Context) {
		utils.ProcessContextRequest(c, xService.ValidateAPOverPurchaseRest)
	})

	//purchase
	purchase.POST("/CreatePO", func(c *gin.Context) {
		utils.ProcessContextRequest(c, purchaseService.CreatePO)
	})
	purchase.POST("/GetPO", func(c *gin.Context) {
		utils.ProcessContextRequest(c, purchaseService.GetPO)
	})
	purchase.POST("/GetPOItemForGR", func(c *gin.Context) {
		utils.ProcessContextRequest(c, purchaseService.GetPOItem)
	})
	purchase.POST("/UpdatePO", func(c *gin.Context) {
		utils.ProcessContextRequest(c, purchaseService.UpdatePO)
	})
	purchase.POST("/UpdateStatusApprovePO", func(c *gin.Context) {
		utils.ProcessContextRequest(c, purchaseService.UpdateStatusApprovePO)
	})
	purchase.POST("/CompleteStatusPaymentPO", func(c *gin.Context) {
		utils.ProcessContextRequest(c, purchaseService.CompleteStatusPaymentPO)
	})
	purchase.POST("/CompletePO", func(c *gin.Context) {
		utils.ProcessContextRequest(c, purchaseService.CompletePO)
	})
	purchase.POST("/CancelPO", func(c *gin.Context) {
		utils.ProcessContextRequest(c, purchaseService.CancelPO)
	})
	purchase.POST("/CompletePOItem", func(c *gin.Context) {
		utils.ProcessContextRequest(c, purchaseService.CompletePOItem)
	})

	///cronjob
	cronjob := ctx.Group("/cronjob")
	cronjob.POST("/credit-request", func(c *gin.Context) {
		utils.ProcessContextRequest(c, CronjobService.GetKernalManual)
	})
	//email alert
	emailAlert := ctx.Group("/emailAlert")
	emailAlert.POST("/SendEmailAlertForNewBrand", func(c *gin.Context) {
		utils.ProcessRequest(c, emailservice.SendEmailAlertForNewBrand)
	})

}
