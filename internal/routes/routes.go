package routes

import (
	"prime-erp-core/internal/utils"

	approvalService "prime-erp-core/internal/services/approval-service"
	creditService "prime-erp-core/internal/services/credit-service"
	depositService "prime-erp-core/internal/services/deposit-service"
	invoiceService "prime-erp-core/internal/services/invoice-service"
	paymentService "prime-erp-core/internal/services/payment-service"
	priceService "prime-erp-core/internal/services/price-service"
	purchaseService "prime-erp-core/internal/services/purchase-service"
	quotationService "prime-erp-core/internal/services/quotation-service"
	saleService "prime-erp-core/internal/services/sale-service"
	summaryService "prime-erp-core/internal/services/summary-credit"
	unitService "prime-erp-core/internal/services/unit-service"
	verifyService "prime-erp-core/internal/services/verify-service"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(ctx *gin.Engine) {

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

	//quotation
	quotation := ctx.Group("/quotation")

	quotation.POST("/GetQuotation", func(c *gin.Context) {
		utils.ProcessRequest(c, quotationService.GetQuotation)
	})
	quotation.POST("/CreateQuotation", func(c *gin.Context) {
		utils.ProcessRequest(c, quotationService.CreateQuotation)
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
	//payment
	payment := ctx.Group("/GetPayment")
	payment.POST("/GetPayment", func(c *gin.Context) {
		utils.ProcessRequest(c, paymentService.GetPayment)
	})
	//sale
	sale := ctx.Group("/sale")
	sale.POST("/CreateSale", func(c *gin.Context) {
		utils.ProcessRequest(c, saleService.CreateSale)
	})
	sale.POST("/UpdateSaleStatusPayment", func(c *gin.Context) {
		utils.ProcessRequest(c, saleService.UpdateSaleStatusPayment)
	})

	sale.POST("/GetSale", func(c *gin.Context) {
		utils.ProcessRequest(c, saleService.GetSale)
	})
	//deposit
	deposit := ctx.Group("/deposit")
	deposit.POST("/GetDeposit", func(c *gin.Context) {
		utils.ProcessRequest(c, depositService.GetDeposit)
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
	credit.POST("/GetHistory", func(c *gin.Context) {
		utils.ProcessRequest(c, creditService.GetHistory)
	})

	//summaryService
	summary := ctx.Group("/summary")
	summary.POST("/GetConsumend", func(c *gin.Context) {
		utils.ProcessRequest(c, summaryService.GetConsumend)
	})
	summary.POST("/GetSummaryCredit", func(c *gin.Context) {
		utils.ProcessRequest(c, summaryService.GetSummaryCredit)
	})
	summary.POST("/GetOutStandingSo", func(c *gin.Context) {
		utils.ProcessRequest(c, summaryService.GetOutStandingSo)
	})

	//unit
	unit := ctx.Group("/unit")
	unit.POST("/GetAllUnit", func(c *gin.Context) {
		utils.ProcessRequest(c, unitService.GetAllUnit)
	})

	//purchase
	purchase := ctx.Group("/purchase")
	purchase.POST("/CreatePOBigLot", func(c *gin.Context) {
		utils.ProcessRequest(c, purchaseService.CreatePOBigLot)
	})
	purchase.POST("/GetPOBigLot", func(c *gin.Context) {
		utils.ProcessRequest(c, purchaseService.GetPOBigLot)
	})
}
