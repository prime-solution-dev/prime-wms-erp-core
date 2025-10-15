package purchaseService

import (
	"encoding/json"
	"errors"
	"fmt"
	"prime-erp-core/internal/models"
	purchaseRepository "prime-erp-core/internal/repositories/purchase"
	systemConfigRepository "prime-erp-core/internal/repositories/systemConfig"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreatePO(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := models.CreatePurchaseRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	// Get purchase code config
	purchaseConfig, err := GetPurchaseCodeConfig()
	if err != nil {
		return nil, errors.New("failed to get purchase code config: " + err.Error())
	}

	purchaseConfigMap := make(map[string]models.SystemConfig)
	for _, config := range purchaseConfig {
		purchaseConfigMap[config.ConfigCode] = config
	}

	NMLno := 0
	TRDno := 0
	PREno := 0

	purchase := []models.Purchase{}
	for _, p := range req.Purchases {
		mappedPurchase := MapPurchaseFormRequestToPurchaseModel(p)

		if p.SupplierCode == nil {
			return nil, errors.New("supplier code is required")
		}

		// Generate purchase code
		config, ok := purchaseConfigMap[p.PurchaseType]
		if !ok {
			return nil, errors.New("failed to get purchase config: config not found for purchase type " + p.PurchaseType)
		}

		lastNo, err := ConvertConfigToLatestPurchaseNumber(config.Value)
		if err != nil {
			return nil, errors.New("failed to convert config to latest purchase number: " + err.Error())
		}

		count := 0

		switch p.PurchaseType {
		case "NML":
			count = NMLno
			NMLno++
		case "TRD":
			count = TRDno
			TRDno++
		case "PRE":
			count = PREno
			PREno++
		}

		saveValue := fmt.Sprintf(
			"%s-%04d",
			time.Now().Format("20060102"),
			lastNo+count+1,
		)

		purchaseCode := fmt.Sprintf(
			"%s-%s%s",
			config.TopicCode,
			config.ConfigCode,
			saveValue,
		)

		config.Value = saveValue
		purchaseConfigMap[p.PurchaseType] = config

		mappedPurchase.PurchaseCode = purchaseCode
		mappedPurchase.CompanyCode = req.CompanyCode
		mappedPurchase.SiteCode = req.SiteCode
		mappedPurchase.SupplierCode = *p.SupplierCode
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

		purchaseItems := []models.PurchaseItem{}
		for _, item := range p.Items {
			mappedItem := MapPurchaseItemFormRequestToPurchaseItemModel(item)
			mappedItem.ID = uuid.New()
			mappedItem.PurchaseID = mappedPurchase.ID
			mappedItem.CreateBy = "system"
			mappedItem.CreateDtm = time.Now().UTC()
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

	// Update purchase code config
	updateSystemConfigReq := []models.SystemConfig{}
	for _, config := range purchaseConfigMap {
		updateSystemConfigReq = append(updateSystemConfigReq, config)
	}

	if err := systemConfigRepository.UpdateSystemConfig(updateSystemConfigReq); err != nil {
		return nil, errors.New("failed to update system config: " + err.Error())
	}

	return nil, nil
}
