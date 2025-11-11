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
		utils.ProcessRequest(c, groupService.GetGroup)
	})

	//price
	price := ctx.Group("/price")

	price.POST("/GetPriceListGroup", func(c *gin.Context) {
		utils.ProcessRequest(c, priceService.GetPriceListGroup)
	})
	price.POST("/GetPaymentTerm", func(c *gin.Context) {
		utils.ProcessRequest(c, priceService.GetPaymentTerm)
	})
	price.POST("/GetComparePrice", func(c *gin.Context) {
		utils.ProcessRequest(c, priceService.GetComparePrice)
	})
	price.POST("/GetPriceList", func(c *gin.Context) {
		utils.ProcessRequest(c, priceService.GetPriceList)
	}) // for Base Price and price list feature
	price.POST("/CreatePriceListGroupBase", func(c *gin.Context) {
		utils.ProcessRequest(c, priceService.CreatePriceListBase)
	})
	price.POST("/UpdatePriceListGroupBase", func(c *gin.Context) {
		utils.ProcessRequest(c, priceService.UpdatePriceListBase)
	})
	price.POST("/UpdatePriceListSubGroup", func(c *gin.Context) {
		utils.ProcessRequestWithBinding(c, priceService.UpdatePriceListSubGroup)
	})
	price.POST("/DeletePriceListGroupBase", func(c *gin.Context) {
		utils.ProcessRequest(c, priceService.DeletePriceListBase)
	})
	price.POST("/GetPriceTable", func(c *gin.Context) {
		utils.ProcessRequest(c, priceService.GetPriceTable)
	})
	price.POST("/SubGroup/Latest", func(c *gin.Context) {
		utils.ProcessRequestWithBinding(c, priceService.GetLatestPriceListSubGroup)
	})
	// config extra get[3] create[2] update delete
	// extra create update delete [4]

	//quotation
	quotation := ctx.Group("/quotation")

	quotation.POST("/GetQuotation", func(c *gin.Context) {
		utils.ProcessRequest(c, quotationService.GetQuotation)
	})
	quotation.POST("/CreateQuotation", func(c *gin.Context) {
		utils.ProcessRequest(c, quotationService.CreateQuotation)
	})
	quotation.POST("/UpdateQuotation", func(c *gin.Context) {
		utils.ProcessRequest(c, quotationService.UpdateQuotation)
	})
	quotation.POST("/EditQuotation", func(c *gin.Context) {
		utils.ProcessRequest(c, quotationService.EditQuotation)
	})

	quotation.POST("/RequestApproveQuotation", func(c *gin.Context) {
		utils.ProcessRequest(c, quotationService.RequestApproveQuotation)
	})
	quotation.POST("/UpdateStatusApproveQuotation", func(c *gin.Context) {
		utils.ProcessRequest(c, quotationService.UpdateStatusApproveQuotation)
	})
	//invoice
	invoice := ctx.Group("/invoice")
	invoice.POST("/GetInvoice", func(c *gin.Context) {
		utils.ProcessRequest(c, invoiceService.GetInvoice)
	})
	invoice.POST("/CreateInvoice", func(c *gin.Context) {
		utils.ProcessRequest(c, invoiceService.CreateInvoice)
	})
	invoice.POST("/UpdateInvoice", func(c *gin.Context) {
		utils.ProcessRequest(c, invoiceService.UpdateInvoice)
	})
	invoice.POST("/CreateInvoiceAP", func(c *gin.Context) {
		utils.ProcessRequest(c, invoiceService.CreateInvoiceAP)
	})
	invoice.POST("/UpdateInvoiceAP", func(c *gin.Context) {
		utils.ProcessRequest(c, invoiceService.UpdateInvoiceAP)
	})
	invoice.POST("/CreateInvoiceAR", func(c *gin.Context) {
		utils.ProcessRequest(c, invoiceService.CreateInvoiceAR)
	})
	invoice.POST("/UpdateInvoiceAR", func(c *gin.Context) {
		utils.ProcessRequest(c, invoiceService.UpdateInvoiceAR)
	})
	invoice.POST("/CreateInvoiceCN", func(c *gin.Context) {
		utils.ProcessRequest(c, invoiceService.CreateInvoiceCN)
	})
	invoice.POST("/UpdateInvoiceCN", func(c *gin.Context) {
		utils.ProcessRequest(c, invoiceService.UpdateInvoiceCN)
	})
	//payment
	payment := ctx.Group("/payment")
	payment.POST("/GetPayment", func(c *gin.Context) {
		utils.ProcessRequest(c, paymentService.GetPayment)
	})
	payment.POST("/CreatePayment", func(c *gin.Context) {
		utils.ProcessRequest(c, paymentService.CreatePayment)
	})

	//sale
	sale := ctx.Group("/sale")
	sale.POST("/CreateSale", func(c *gin.Context) {
		utils.ProcessRequest(c, saleService.CreateSale)
	})
	sale.POST("/UpdateSaleStatusPayment", func(c *gin.Context) {
		utils.ProcessRequest(c, saleService.UpdateSaleStatusPayment)
	})

	sale.POST("/EditSale", func(c *gin.Context) {
		utils.ProcessRequest(c, saleService.EditSale)
	})
	sale.POST("/GetSale", func(c *gin.Context) {
		utils.ProcessRequest(c, saleService.GetSale)
	})
	sale.POST("/UpdateSale", func(c *gin.Context) {
		utils.ProcessRequest(c, saleService.UpdateSale)
	})
	sale.POST("/RequestApproveSale", func(c *gin.Context) {
		utils.ProcessRequest(c, saleService.RequestApproveSale)
	})
	sale.POST("/UpdateStatusApproveSale", func(c *gin.Context) {
		utils.ProcessRequest(c, saleService.UpdateStatusApproveSale)
	})

	sale.POST("/GetSalePack", func(c *gin.Context) {
		utils.ProcessRequest(c, saleService.GetSalePack)
	})
	//delivery
	delivery := ctx.Group("/delivery")
	delivery.POST("/CreateDelivery", func(c *gin.Context) {
		utils.ProcessRequest(c, deliveryService.CreateDelivery)
	})
	delivery.POST("/GetDelivery", func(c *gin.Context) {
		utils.ProcessRequest(c, deliveryService.GetDelivery)
	})
	delivery.POST("/UpdateDelivery", func(c *gin.Context) {
		utils.ProcessRequest(c, deliveryService.UpdateDelivery)
	})
	delivery.POST("/UpdateStatusDelivery", func(c *gin.Context) {
		utils.ProcessRequest(c, deliveryService.UpdateStatusDelivery)
	})
	/* 	delivery.POST("/GetDeliverySO", func(c *gin.Context) {
	   		utils.ProcessRequest(c, deliveryService.GetDeliverySO)
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
		utils.ProcessRequest(c, approvalService.GetApproval)
	})
	approval.POST("/CreateApproval", func(c *gin.Context) {
		utils.ProcessRequest(c, approvalService.CreateApproval)
	})
	approval.POST("/UpdateApproval", func(c *gin.Context) {
		utils.ProcessRequest(c, approvalService.UpdateApproval)
	})
	//credit
	credit := ctx.Group("/credit")
	credit.POST("/GetCreditCurrent", func(c *gin.Context) {
		utils.ProcessRequest(c, creditService.GetCreditCurrentAPI)
	})
	credit.POST("/GetCreditRequest", func(c *gin.Context) {
		utils.ProcessRequest(c, creditService.GetCreditRequests)
	})
	credit.POST("/CreateCreditRequest", func(c *gin.Context) {
		utils.ProcessRequest(c, creditService.CreateCreditRequest)
	})
	credit.POST("/UpdateCreditRequest", func(c *gin.Context) {
		utils.ProcessRequest(c, creditService.UpdateCreditRequest)
	})
	credit.POST("/GetCredit", func(c *gin.Context) {
		utils.ProcessRequest(c, creditService.GetCredit)
	})
	credit.POST("/CreateCredit", func(c *gin.Context) {
		utils.ProcessRequest(c, creditService.CreateCredit)
	})
	credit.POST("/GetHistory", func(c *gin.Context) {
		utils.ProcessRequest(c, creditService.GetHistory)
	})
	credit.POST("/GetSummaryCredit", func(c *gin.Context) {
		utils.ProcessRequest(c, creditService.GetSummaryCredit)
	})
	credit.POST("/GetTransaction", func(c *gin.Context) {
		utils.ProcessRequest(c, creditService.GetTransaction)
	})
	credit.POST("/CreateCreditTransaction", func(c *gin.Context) {
		utils.ProcessRequest(c, creditService.CreateCreditTransaction)
	})
	credit.POST("/DeleteCreditExtra", func(c *gin.Context) {
		utils.ProcessRequest(c, creditService.DeleteCreditExtra)
	})

	//summaryService
	summary := ctx.Group("/summary")
	summary.POST("/GetConsumend", func(c *gin.Context) {
		utils.ProcessRequest(c, summaryService.GetConsumend)
	})

	summary.POST("/GetOutStandingSo", func(c *gin.Context) {
		utils.ProcessRequest(c, summaryService.GetOutStandingSo)
	})

	//unit
	unit := ctx.Group("/unit")
	unit.POST("/GetAllUnit", func(c *gin.Context) {
		utils.ProcessRequest(c, unitService.GetAllUnit)
	})

	purchase := ctx.Group("/purchase")
	//pre-purchase
	purchase.POST("/CreatePOBigLot", func(c *gin.Context) {
		utils.ProcessRequest(c, prePurchaseService.CreatePOBigLot)
	})
	purchase.POST("/GetPOBigLot", func(c *gin.Context) {
		utils.ProcessRequest(c, prePurchaseService.GetPOBigLot)
	})
	purchase.POST("/UpdatePOBigLot", func(c *gin.Context) {
		utils.ProcessRequest(c, prePurchaseService.UpdatePOBigLot)
	})
	purchase.POST("/UpdateStatusApprovePOBigLot", func(c *gin.Context) {
		utils.ProcessRequest(c, prePurchaseService.UpdateStatusApprovePOBigLot)
	})

	//purchase
	purchase.POST("/CreatePO", func(c *gin.Context) {
		utils.ProcessRequest(c, purchaseService.CreatePO)
	})
	purchase.POST("/GetPO", func(c *gin.Context) {
		utils.ProcessRequest(c, purchaseService.GetPO)
	})
	purchase.POST("/GetPOItemForGR", func(c *gin.Context) {
		utils.ProcessRequest(c, purchaseService.GetPOItem)
	})
	purchase.POST("/UpdatePO", func(c *gin.Context) {
		utils.ProcessRequest(c, purchaseService.UpdatePO)
	})
	purchase.POST("/UpdateStatusApprovePO", func(c *gin.Context) {
		utils.ProcessRequest(c, purchaseService.UpdateStatusApprovePO)
	})
	purchase.POST("/CompleteStatusPaymentPO", func(c *gin.Context) {
		utils.ProcessRequest(c, purchaseService.CompleteStatusPaymentPO)
	})
	purchase.POST("/CompletePO", func(c *gin.Context) {
		utils.ProcessRequest(c, purchaseService.CompletePO)
	})
	purchase.POST("/CompletePOItem", func(c *gin.Context) {
		utils.ProcessRequest(c, purchaseService.CompletePOItem)
	})

	///cronjob
	cronjob := ctx.Group("/cronjob")
	cronjob.POST("/credit-request", func(c *gin.Context) {
		utils.ProcessRequest(c, CronjobService.CreditRequestEffectiveDtm)
	})
	//email alert
	emailAlert := ctx.Group("/emailAlert")
	emailAlert.POST("/SendEmailAlertForNewBrand", func(c *gin.Context) {
		utils.ProcessRequest(c, emailservice.SendEmailAlertForNewBrand)
	})

}
