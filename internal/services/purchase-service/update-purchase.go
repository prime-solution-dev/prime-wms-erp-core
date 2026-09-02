package purchaseService

import (
	"encoding/json"
	"errors"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	purchaseRepository "prime-erp-core/internal/repositories/purchase"
	approvalService "prime-erp-core/internal/services/approval-service"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func UpdatePO(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := []models.PurchaseFormRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
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

	purchases := []models.Purchase{}
	for _, r := range req {
		purchase := MapPurchaseFormRequestToPurchaseModel(r)

		if r.ID == nil || r.PurchaseCode == nil {
			return nil, errors.New("purchase ID and code are required for update")
		}

		purchase.ID = *r.ID
		purchase.PurchaseCode = *r.PurchaseCode
		purchase.UpdateBy = userCode
		purchase.UpdateDtm = time.Now().UTC()

		// Map purchase items: existing lines keep their purchase_item,
		// new lines continue the running number after the highest one.
		reqItems := []models.PurchaseItem{}
		seq := NextPurchaseItemSeq(r.Items)
		for _, item := range r.Items {
			purchaseItem := MapPurchaseItemFormRequestToPurchaseItemModel(item, seq)
			if item.PurchaseItem == nil || strings.TrimSpace(*item.PurchaseItem) == "" {
				seq++
			}
			purchaseItem.PurchaseID = purchase.ID

			reqItems = append(reqItems, purchaseItem)
		}

		purchase.PurchaseItems = reqItems

		if r.Status == "PENDING" {
			// Check auto approval for PENDING status
			autoApprovalReq := approvalService.CheckAutoApprovalRequest{
				RequestUserCode: userCode,
				ModuleCode:      "CUSTOMIZE",
				TopicCode:       "CUSTOMIZE",
				MdItemCode:      "CTM-CTM3",
				CondRangeMin:    r.TotalAmount,
				// TODO: Add module_code, topic_code, md_item_code
			}

			autoApprovalRes, err := approvalService.CheckAutoApproval(gormx, autoApprovalReq, userCode)
			if err != nil {
				return nil, err
			}

			if autoApprovalRes.IsAutoApproved {
				r.IsApproved = true
				r.StatusApprove = "COMPLETED"
			} else {
				r.IsApproved = false
				r.StatusApprove = "PROCESS"
			}
		}

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

func CompletePOItem(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := models.CompletePurchaseItemRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	if err := purchaseRepository.CompletePOItem(req.UsedType, req.PurchaseItemUsed); err != nil {
		return nil, errors.New("failed to complete PO item: " + err.Error())
	}

	return nil, nil
}
