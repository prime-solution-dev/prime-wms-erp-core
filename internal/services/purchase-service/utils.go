package purchaseService

import (
	"encoding/json"
	"errors"
	"fmt"
	"prime-erp-core/internal/models"
	systemConfigRepository "prime-erp-core/internal/repositories/systemConfig"
	approvalService "prime-erp-core/internal/services/approval-service"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func MapBigLotRequestToPrePurchaseItemsModel(reqItems []models.CreatePOBigLotItemRequest, prePurchaseID uuid.UUID, user string) []models.PrePurchaseItem {
	items := make([]models.PrePurchaseItem, 0, len(reqItems))

	for _, item := range reqItems {
		items = append(items, models.PrePurchaseItem{
			ID:                   uuid.New(),
			PrePurchaseID:        prePurchaseID,
			PreItem:              item.PreItem,
			HierarchyType:        item.ProductGroupType,
			HierarchyCode:        item.ProductGroupCode,
			DocRefItem:           item.DocRefItem,
			Qty:                  item.Qty,
			Unit:                 item.Unit,
			PurchaseQty:          item.PurchaseQty,
			PurchaseUnit:         item.PurchaseUnit,
			PurchaseUnitType:     item.PurchaseUnitType,
			PriceUnit:            item.PriceUnit,
			TotalDiscount:        item.TotalDiscount,
			TotalAmount:          item.TotalAmount,
			UnitUOM:              item.UnitUOM,
			TotalCost:            item.TotalCost,
			TotalDiscountPercent: item.TotalDiscountPercent,
			TotalVat:             item.TotalVat,
			SubtotalExclVat:      item.SubtotalExclVat,
			WeightUnit:           item.WeightUnit,
			TotalWeight:          item.TotalWeight,
			Status:               item.Status,
			Remark:               item.Remark,
			CreateBy:             user,
			UpdateBy:             user,
		})
	}
	return items
}

func MapBigLotRequestToPrePurchaseModel(req models.CreatePOBigLotRequest) models.PrePurchase {
	user := `system` // TODO: get from ctx

	prePurchase := models.PrePurchase{
		ID:              uuid.New(),
		PurchaseType:    "LOT",
		CompanyCode:     req.CompanyCode,
		SiteCode:        req.SiteCode,
		DocRefType:      "",
		SupplierCode:    req.SupplierCode,
		DeliveryAddress: req.DeliveryAddress,
		Status:          req.Status,
		TotalAmount:     req.TotalAmount,
		TotalWeight:     req.TotalWeight,
		TotalDiscount:   req.TotalDiscount,
		TotalVat:        req.TotalVat,
		SubtotalExclVat: req.SubtotalExclVat,
		IsApproved:      req.IsApproved,
		StatusApprove:   req.StatusApprove,
		Remark:          req.Remark,
		CreateBy:        user,
		UpdateBy:        user,
	}

	prePurchase.PrePurchaseItems = MapBigLotRequestToPrePurchaseItemsModel(req.Items, prePurchase.ID, user)

	return prePurchase
}

func MapPrePurchaseItemsModelToBigLotItemsResponse(prePurchaseItems []models.PrePurchaseItem) []models.GetPOBigLotItemResponse {
	items := []models.GetPOBigLotItemResponse{}

	for _, item := range prePurchaseItems {
		items = append(items, models.GetPOBigLotItemResponse{
			ID:                   item.ID.String(),
			PrePurchaseID:        item.PrePurchaseID.String(),
			ProductGroupType:     item.HierarchyType,
			ProductGroupCode:     item.HierarchyCode,
			Qty:                  item.Qty,
			Unit:                 item.Unit,
			PurchaseQty:          item.PurchaseQty,
			PurchaseUnit:         item.PurchaseUnit,
			PurchaseUnitType:     item.PurchaseUnitType,
			PriceUnit:            item.PriceUnit,
			TotalDiscount:        item.TotalDiscount,
			TotalAmount:          item.TotalAmount,
			UnitUOM:              item.UnitUOM,
			TotalCost:            item.TotalCost,
			TotalDiscountPercent: item.TotalDiscountPercent,
			TotalVat:             item.TotalVat,
			WeightUnit:           item.WeightUnit,
			TotalWeight:          item.TotalWeight,
			Status:               item.Status,
			Remark:               item.Remark,
			CreateBy:             item.CreateBy,
			CreateDate:           item.CreateDate.Format(time.RFC3339),
			UpdateBy:             item.UpdateBy,
			UpdateDate:           item.UpdateDate.Format(time.RFC3339),
		})
	}

	return items
}

func MapPrePurchasesModelToBigLotsResponse(prePurchases models.PrePurchase) models.GetPOBigLotResponse {
	return models.GetPOBigLotResponse{
		ID:               prePurchases.ID.String(),
		PrePurchaseCode:  prePurchases.PrePurchaseCode,
		PurchaseType:     prePurchases.PurchaseType,
		CompanyCode:      prePurchases.CompanyCode,
		SiteCode:         prePurchases.SiteCode,
		SupplierCode:     prePurchases.SupplierCode,
		DeliveryAddress:  prePurchases.DeliveryAddress,
		Status:           prePurchases.Status,
		TotalAmount:      prePurchases.TotalAmount,
		TotalWeight:      prePurchases.TotalWeight,
		TotalDiscount:    prePurchases.TotalDiscount,
		TotalVat:         prePurchases.TotalVat,
		SubtotalExclVat:  prePurchases.SubtotalExclVat,
		IsApproved:       prePurchases.IsApproved,
		StatusApprove:    prePurchases.StatusApprove,
		Remark:           prePurchases.Remark,
		CreateBy:         prePurchases.CreateBy,
		CreateDate:       prePurchases.CreateDate.Format(time.RFC3339),
		UpdateBy:         prePurchases.UpdateBy,
		UpdateDate:       prePurchases.UpdateDate.Format(time.RFC3339),
		PrePurchaseItems: MapPrePurchaseItemsModelToBigLotItemsResponse(prePurchases.PrePurchaseItems),
	}
}

func MapUpdatePOBigLotRequestToPrePurchaseItem(req []models.UpdatePOBigLotItemRequest) []models.PrePurchaseItem {
	results := []models.PrePurchaseItem{}

	for _, reqItem := range req {
		if reqItem.ID == nil || *reqItem.ID == uuid.Nil {
			newID := uuid.New()
			reqItem.ID = &newID
		}

		item := models.PrePurchaseItem{
			ID:                   *reqItem.ID,
			PrePurchaseID:        reqItem.PrePurchaseID,
			HierarchyType:        reqItem.ProductGroupType,
			HierarchyCode:        reqItem.ProductGroupCode,
			Qty:                  reqItem.Qty,
			Unit:                 reqItem.Unit,
			PurchaseQty:          reqItem.PurchaseQty,
			PurchaseUnit:         reqItem.PurchaseUnit,
			PurchaseUnitType:     reqItem.PurchaseUnitType,
			PriceUnit:            reqItem.PriceUnit,
			TotalDiscount:        reqItem.TotalDiscount,
			TotalAmount:          reqItem.TotalAmount,
			UnitUOM:              reqItem.UnitUOM,
			TotalCost:            reqItem.TotalCost,
			TotalDiscountPercent: reqItem.TotalDiscountPercent,
			TotalVat:             reqItem.TotalVat,
			SubtotalExclVat:      reqItem.SubtotalExclVat,
			WeightUnit:           reqItem.WeightUnit,
			TotalWeight:          reqItem.TotalWeight,
			Status:               reqItem.Status,
			Remark:               reqItem.Remark,
		}

		results = append(results, item)
	}

	return results
}

func MapUpdatePOBigLotRequestToPrePurchase(req models.UpdatePOBigLotRequest) models.PrePurchase {
	return models.PrePurchase{
		ID:              req.ID,
		Status:          req.Status,
		TotalAmount:     req.TotalAmount,
		TotalWeight:     req.TotalWeight,
		TotalDiscount:   req.TotalDiscount,
		TotalVat:        req.TotalVat,
		SubtotalExclVat: req.SubtotalExclVat,
		IsApproved:      req.IsApproved,
		StatusApprove:   req.StatusApprove,
	}
}

// Approval action
func CreateBigLotToApproval(ctx *gin.Context, prePurchase []models.PrePurchase) error {
	user := `system` // TODO: get from ctx

	approvalReq := []models.Approval{}

	for _, pp := range prePurchase {
		approvalReq = append(approvalReq, models.Approval{
			ApproveTopic:  "PPOL",
			DocumentType:  "PPO",
			DocumentCode:  pp.PrePurchaseCode,
			ActionDate:    time.Now(),
			Status:        pp.StatusApprove,
			Remark:        "-",
			CurentStepSeq: 1,
			MDItemCode:    "CTM-CTM1",
			CreateBy:      user,
		})
	}

	approvalReqJson, err := json.Marshal(approvalReq)
	if err != nil {
		return errors.New("failed to marshal JSON from struct: " + err.Error())
	}

	approvalReqString := string(approvalReqJson)

	approvalIDs, err := approvalService.CreateApproval(ctx, approvalReqString)
	if err != nil {
		return err
	}

	fmt.Println("approvalIDs:", approvalIDs)
	return nil
}

func GetBigLotToApproval(ctx *gin.Context, prePurchaseCodes []string) ([]models.Approval, error) {
	approvalReq := approvalService.GetApprovalRequest{
		DocumentCode: prePurchaseCodes,
		Page:         1,
		PageSize:     len(prePurchaseCodes),
	}

	approvalReqJson, err := json.Marshal(approvalReq)
	if err != nil {
		return nil, errors.New("failed to marshal JSON from struct: " + err.Error())
	}

	approvalReqString := string(approvalReqJson)

	resp, err := approvalService.GetApproval(ctx, approvalReqString)
	if err != nil {
		return nil, errors.New("failed to get approval list: " + err.Error())
	}

	approvalResp, ok := resp.(approvalService.ResultApproval)
	if !ok {
		return nil, errors.New("failed to assertion approval type")
	}

	return approvalResp.ApprovalRes, nil
}

func UpdateBigLotToApproval(ctx *gin.Context, updateReqs []models.UpdateStatusApprovePOBigLotRequest) error {
	fmt.Println("KAO")
	prePurchaseCodes := []string{}
	mapUpdateList := make(map[string]models.Approval)

	for _, req := range updateReqs {
		prePurchaseCodes = append(prePurchaseCodes, req.PrePurchaseCode)
		mapUpdateList[req.PrePurchaseCode] = models.Approval{
			DocumentCode: req.PrePurchaseCode,
			Status:       req.StatusApprove,
		}
	}

	approvalList, err := GetBigLotToApproval(ctx, prePurchaseCodes)
	if err != nil {
		return errors.New("failed get approvals")
	}

	updateApprovalReq := []models.Approval{}

	for _, approval := range approvalList {
		updateApprovalReq = append(updateApprovalReq, models.Approval{
			ID:     approval.ID,
			Status: mapUpdateList[approval.DocumentCode].Status,
		})
	}

	approvalReqJson, err := json.Marshal(updateApprovalReq)
	if err != nil {
		return errors.New("failed to marshal JSON from struct: " + err.Error())
	}

	resp, err := approvalService.UpdateApproval(ctx, string(approvalReqJson))
	if err != nil {
		return errors.New("failed to update approval: " + err.Error())
	}

	fmt.Println("updated approval: ", resp)

	return nil
}

// System Config
// pre_purchase_code ex. PPO-LOT20250930-0001; value ex. 20250930-0001
func GetPrePurchaseCodeConfig() (*models.SystemConfig, error) {
	topicCodes := []string{"PPO"}
	configCodes := []string{"LOT"}

	prePurchaseConfigs, err := systemConfigRepository.GetSystemConfig(topicCodes, configCodes)
	if err != nil {
		return nil, err
	}

	prePurchaseConfigMap := make(map[string]models.SystemConfig)

	for _, poRunConfig := range prePurchaseConfigs {
		prePurchaseConfigMap[fmt.Sprintf("%s|%s", poRunConfig.TopicCode, poRunConfig.ConfigCode)] = poRunConfig
	}

	config := prePurchaseConfigMap["PPO|LOT"]
	return &config, nil
}

func ConvertConfigToLatestPrePurchaseNumber(prePurchaseConfig models.SystemConfig) (int, error) {
	result := 0

	if len(prePurchaseConfig.Value) < 1 {
		return result, nil
	}

	valueParts := strings.Split(prePurchaseConfig.Value, "-")
	latestDate := valueParts[0]
	latestNo := valueParts[1]

	t1, _ := time.Parse("2006-01-02", latestDate)
	t2 := time.Now()

	t1Day := time.Date(t1.Year(), t1.Month(), t1.Day(), 0, 0, 0, 0, t1.Location())
	t2Day := time.Date(t2.Year(), t2.Month(), t2.Day(), 0, 0, 0, 0, t2.Location())

	if t1Day.Equal(t2Day) {
		lastNum, err := strconv.Atoi(latestNo)
		if err != nil {
			return 0, err
		}

		result = lastNum
	}

	return result, nil
}
