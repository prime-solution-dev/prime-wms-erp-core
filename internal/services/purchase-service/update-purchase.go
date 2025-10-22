package purchaseService

import (
	"encoding/json"
	"errors"
	"prime-erp-core/internal/models"
	purchaseRepository "prime-erp-core/internal/repositories/purchase"

	"github.com/gin-gonic/gin"
)

func UpdatePO(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := []models.PurchaseFormRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	purchases := []models.Purchase{}
	for _, r := range req {
		purchase := MapPurchaseFormRequestToPurchaseModel(r)

		if r.ID == nil || r.PurchaseCode == nil {
			return nil, errors.New("purchase ID and code are required for update")
		}

		purchase.ID = *r.ID
		purchase.PurchaseCode = *r.PurchaseCode

		// Map purchase items
		reqItems := []models.PurchaseItem{}
		for _, item := range r.Items {
			purchaseItem := MapPurchaseItemFormRequestToPurchaseItemModel(item, purchase.PurchaseCode)

			if item.ID == nil || item.PurchaseID == nil {
				return nil, errors.New("purchase item ID and purchase ID are required for update")
			}

			reqItems = append(reqItems, purchaseItem)
		}

		purchase.PurchaseItems = reqItems

		purchases = append(purchases, purchase)
	}

	if err := purchaseRepository.UpdatePurchase(purchases); err != nil {
		return nil, errors.New("fail to update purchase: " + err.Error())
	}

	return nil, nil
}

func UpdateStatusApprovePO(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := []models.UpdateStatusApprovePurchaseRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	// Update approval status
	if err := UpdatePOToApproval(ctx, req); err != nil {
		return nil, err
	}

	if err := purchaseRepository.UpdatePurchaseStatusApprove(req); err != nil {
		return nil, errors.New("failed to update purchase status approve: " + err.Error())
	}

	return nil, nil
}

func CompleteStatusPaymentPO(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := models.CompleteStatusPaymentPurchaseRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	if err := purchaseRepository.CompletePOPayment(req.PurchaseCodes, req.PurchaseItems); err != nil {
		return nil, errors.New("failed to complete PO payment: " + err.Error())
	}

	return nil, nil
}

func CompletePO(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := models.CompletePurchaseRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	if err := purchaseRepository.CompletePO(req.PurchaseCodes); err != nil {
		return nil, errors.New("failed to complete PO: " + err.Error())
	}

	return nil, nil
}
