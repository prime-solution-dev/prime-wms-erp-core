package purchaseService

import (
	"encoding/json"
	"errors"
	"prime-erp-core/internal/models"
	purchaseRepository "prime-erp-core/internal/repositories/purchase"
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
		mappedPurchase.CreateBy = "system"
		mappedPurchase.CreateDtm = time.Now().UTC()

		purchaseItems := []models.PurchaseItem{}
		for _, item := range p.Items {
			mappedItem := MapPurchaseItemFormRequestToPurchaseItemModel(item, purchaseCodes[i])
			mappedItem.ID = uuid.New()
			mappedItem.PurchaseID = mappedPurchase.ID
			purchaseItems = append(purchaseItems, mappedItem)
		}

		mappedPurchase.PurchaseItems = purchaseItems

		purchase = append(purchase, mappedPurchase)
	}

	// Create purchase
	if err := purchaseRepository.CreatePurchase(purchase); err != nil {
		return nil, errors.New("failed to create purchase: " + err.Error())
	}

	// Create purchase approval
	if err := CreatePurchaseApproval(ctx, purchase); err != nil {
		return nil, errors.New("failed to create purchase approval: " + err.Error())
	}

	return nil, nil
}
