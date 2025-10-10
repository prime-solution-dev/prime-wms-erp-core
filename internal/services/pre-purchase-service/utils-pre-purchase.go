package prePurchaseService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"prime-erp-core/internal/models"
	systemConfigRepository "prime-erp-core/internal/repositories/systemConfig"
	approvalService "prime-erp-core/internal/services/approval-service"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func MapBigLotRequestToPrePurchaseItemsModel(reqItems []models.CreatePOBigLotItemRequest, prePurchaseID uuid.UUID, user string, now time.Time) []models.PrePurchaseItem {
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
			DiscountType:         item.DiscountType,
			TotalVat:             item.TotalVat,
			SubtotalExclVat:      item.SubtotalExclVat,
			WeightUnit:           item.WeightUnit,
			TotalWeight:          item.TotalWeight,
			Status:               item.Status,
			Remark:               item.Remark,
			CreateBy:             user,
			CreateDtm:            now,
			UpdateBy:             user,
			UpdateDtm:            now,
		})
	}
	return items
}

func MapBigLotRequestToPrePurchaseModel(req models.CreatePOBigLotRequest) models.PrePurchase {
	user := `system` // TODO: get from ctx
	now := time.Now().UTC()

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
		CreateDtm:       now,
		UpdateBy:        user,
		UpdateDtm:       now,
	}

	prePurchase.PrePurchaseItems = MapBigLotRequestToPrePurchaseItemsModel(req.Items, prePurchase.ID, user, now)

	return prePurchase
}

func MapPrePurchaseItemsModelToBigLotItemsResponse(prePurchaseItems []models.PrePurchaseItem) []models.GetPOBigLotItemResponse {
	items := []models.GetPOBigLotItemResponse{}

	for _, item := range prePurchaseItems {
		items = append(items, models.GetPOBigLotItemResponse{
			ID:                   item.ID.String(),
			PrePurchaseID:        item.PrePurchaseID.String(),
			PreItem:              item.PreItem,
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
			DiscountType:         item.DiscountType,
			SubtotalExclVat:      item.SubtotalExclVat,
			TotalVat:             item.TotalVat,
			WeightUnit:           item.WeightUnit,
			TotalWeight:          item.TotalWeight,
			Status:               item.Status,
			Remark:               item.Remark,
			CreateBy:             item.CreateBy,
			CreateDtm:            item.CreateDtm.Format(time.RFC3339),
			UpdateBy:             item.UpdateBy,
			UpdateDtm:            item.UpdateDtm.Format(time.RFC3339),
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
		CreateDtm:        prePurchases.CreateDtm.Format(time.RFC3339),
		UpdateBy:         prePurchases.UpdateBy,
		UpdateDtm:        prePurchases.UpdateDtm.Format(time.RFC3339),
		PrePurchaseItems: MapPrePurchaseItemsModelToBigLotItemsResponse(prePurchases.PrePurchaseItems),
	}
}

func MapUpdatePOBigLotRequestToPrePurchaseItem(req []models.UpdatePOBigLotItemRequest) []models.PrePurchaseItem {
	results := []models.PrePurchaseItem{}
	user := "system"
	now := time.Now().UTC()

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
			DiscountType:         reqItem.DiscountType,
			TotalVat:             reqItem.TotalVat,
			SubtotalExclVat:      reqItem.SubtotalExclVat,
			WeightUnit:           reqItem.WeightUnit,
			TotalWeight:          reqItem.TotalWeight,
			Status:               reqItem.Status,
			Remark:               reqItem.Remark,
			CreateBy:             reqItem.CreateBy,
			CreateDtm:            reqItem.CreateDtm,
			UpdateBy:             user,
			UpdateDtm:            now,
		}

		results = append(results, item)
	}

	return results
}

func MapUpdatePOBigLotRequestToPrePurchase(req models.UpdatePOBigLotRequest) models.PrePurchase {
	user := "system"
	now := time.Now().UTC()

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
		DeliveryAddress: req.DeliveryAddress,
		Remark:          req.Remark,
		UpdateBy:        user,
		UpdateDtm:       now,
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

func GetPOApproval(ctx *gin.Context, POcodes []string) ([]models.Approval, error) {
	approvalReq := approvalService.GetApprovalRequest{
		DocumentCode: POcodes,
		Page:         1,
		PageSize:     len(POcodes),
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

func UpdatePOApproval(ctx *gin.Context, docCodes []string, mappedApprovalReq map[string]models.Approval) error {
	approvalList, err := GetPOApproval(ctx, docCodes)
	if err != nil {
		return errors.New("failed get approvals: " + err.Error())
	}

	updateApprovalReq := []models.Approval{}
	for _, approval := range approvalList {
		if mapped, ok := mappedApprovalReq[approval.DocumentCode]; ok {
			updateApprovalReq = append(updateApprovalReq, models.Approval{
				ID:     approval.ID,
				Status: mapped.Status,
			})
		} else {
			return fmt.Errorf("approval request for document code %s not found", approval.DocumentCode)
		}
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

func UpdateBigLotToApproval(ctx *gin.Context, updateReqs []models.UpdateStatusApprovePOBigLotRequest) error {
	prePurchaseCodes := []string{}
	mapUpdateList := make(map[string]models.Approval)

	for _, req := range updateReqs {
		prePurchaseCodes = append(prePurchaseCodes, req.PrePurchaseCode)
		mapUpdateList[req.PrePurchaseCode] = models.Approval{
			DocumentCode: req.PrePurchaseCode,
			Status:       req.StatusApprove,
		}
	}

	if err := UpdatePOApproval(ctx, prePurchaseCodes, mapUpdateList); err != nil {
		return errors.New("failed update approvals")
	}

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

	t1, err := time.ParseInLocation("20060102", latestDate, time.Local)
	if err != nil {
		return result, err
	}

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

// Supplier actions
func GetSupplierByCode(supplierReq models.GetSupplierListRequest) (map[string]models.Supplier, error) {
	jsonData, err := json.Marshal(supplierReq)
	if err != nil {
		return nil, errors.New("failed to marshal supplier data to JSON: " + err.Error())
	}

	getSuppliers, err := http.NewRequest("POST", os.Getenv("base_url_supplier")+"/get-suppliers", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, errors.New("failed to create HTTP request: " + err.Error())
	}

	getSuppliers.Header.Set("Content-Type", "application/json")

	// Create a client and execute the request
	client := &http.Client{}
	resp, err := client.Do(getSuppliers)
	if err != nil {
		return nil, errors.New("failed to execute HTTP request: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("received non-OK HTTP status: " + resp.Status)
	}

	supplierBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("failed to read response body: " + err.Error())
	}

	supplierResponse := models.GetSupplierListResponse{}
	if err := json.Unmarshal(supplierBody, &supplierResponse); err != nil {
		return nil, errors.New("failed to decode JSON response: " + err.Error())
	}

	mapSupplier := map[string]models.Supplier{}
	for _, suppliers := range supplierResponse.Supplier {
		mapSupplier[suppliers.SupplierCode] = suppliers
	}

	return mapSupplier, nil
}
