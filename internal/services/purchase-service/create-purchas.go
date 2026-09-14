package purchaseService

import (
	"encoding/json"
	"errors"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	purchaseRepository "prime-erp-core/internal/repositories/purchase"
	approvalService "prime-erp-core/internal/services/approval-service"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreatePO(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := models.CreatePurchaseRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	count := len(req.Purchases)
	purchaseCodes, err := GeneratePurchaseCodes(ctx, count)
	if err != nil {
		return nil, errors.New("failed to generate purchase order codes: " + err.Error())
	}

	userCode := ""
	if ctx != nil {
		userCode = ctx.GetString("user")
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	purchase := []models.Purchase{}
	for i, p := range req.Purchases {
		mappedPurchase := MapPurchaseFormRequestToPurchaseModel(p)

		mappedPurchase.PurchaseCode = purchaseCodes[i]
		mappedPurchase.CompanyCode = req.CompanyCode
		mappedPurchase.SiteCode = req.SiteCode
		mappedPurchase.SupplierCode = *p.SupplierCode
		mappedPurchase.SupplierName = *p.SupplierName
		mappedPurchase.SupplierAddress = *p.SupplierAddress
		mappedPurchase.SupplierPhone = *p.SupplierPhone
		mappedPurchase.SupplierEmail = *p.SupplierEmail
		mappedPurchase.ID = uuid.New()

		docRefType := ""
		if p.DocRefType != nil {
			docRefType = *p.DocRefType
		}

		docRef := ""
		if p.DocRef != nil {
			docRef = *p.DocRef
		}

		mappedPurchase.DocRefType = &docRefType
		mappedPurchase.DocRef = &docRef
		mappedPurchase.TradingRef = p.TradingRef
		mappedPurchase.CreateBy = userCode
		mappedPurchase.UpdateBy = userCode
		mappedPurchase.CreateDtm = time.Now().UTC()
		mappedPurchase.UpdateDtm = time.Now().UTC()

		purchaseItems := []models.PurchaseItem{}
		for idx, item := range p.Items {
			mappedItem := MapPurchaseItemFormRequestToPurchaseItemModel(item, idx+1)
			mappedItem.ID = uuid.New()
			mappedItem.PurchaseID = mappedPurchase.ID
			purchaseItems = append(purchaseItems, mappedItem)
		}

		mappedPurchase.PurchaseItems = purchaseItems

		if p.Status == "PENDING" {
			// Check auto approval for PENDING status
			autoApprovalReq := approvalService.CheckAutoApprovalRequest{
				RequestUserCode: userCode,
				ModuleCode:      "CUSTOMIZE",
				TopicCode:       "CUSTOMIZE",
				MdItemCode:      "CTM-CTM3",
				CondRangeMin:    p.TotalAmount,
				// TODO: Add module_code, topic_code, md_item_code
			}

			autoApprovalRes, err := approvalService.CheckAutoApproval(gormx, autoApprovalReq, userCode)
			if err != nil {
				return nil, err
			}

			if autoApprovalRes.IsAutoApproved {
				mappedPurchase.IsApproved = true
				mappedPurchase.StatusApprove = "COMPLETED"
			} else {
				mappedPurchase.IsApproved = false
				mappedPurchase.StatusApprove = "PROCESS"
			}
		}

		purchase = append(purchase, mappedPurchase)
	}

	// Create purchase
	if err := purchaseRepository.CreatePurchase(gormx, purchase); err != nil {
		return nil, errors.New("failed to create purchase: " + err.Error())
	}

	// Create purchase approval
	if err := CreatePurchaseApproval(ctx, purchase); err != nil {
		return nil, errors.New("failed to create purchase approval: " + err.Error())
	}

	return purchaseCodes, nil
}
