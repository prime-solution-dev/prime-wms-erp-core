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
	approvalService "prime-erp-core/internal/services/approval-service"
	systemConfigService "prime-erp-core/internal/services/system-config"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func MapBigLotRequestToPrePurchaseItemsModel(reqItems models.CreatePOBigLotItemRequest, prePurchaseID uuid.UUID, user string, now time.Time, preItem string) models.PrePurchaseItem {
	return models.PrePurchaseItem{
		ID:                   uuid.New(),
		PrePurchaseID:        prePurchaseID,
		PreItem:              preItem,
		HierarchyType:        reqItems.ProductGroupType,
		HierarchyCode:        reqItems.ProductGroupCode,
		DocRefItem:           reqItems.DocRefItem,
		Qty:                  reqItems.Qty,
		Unit:                 reqItems.Unit,
		PurchaseQty:          reqItems.PurchaseQty,
		PurchaseUnit:         reqItems.PurchaseUnit,
		PurchaseUnitType:     reqItems.PurchaseUnitType,
		PriceUnit:            reqItems.PriceUnit,
		TotalDiscount:        reqItems.TotalDiscount,
		TotalAmount:          reqItems.TotalAmount,
		UnitUom:              reqItems.UnitUom,
		TotalCost:            reqItems.TotalCost,
		TotalDiscountPercent: reqItems.TotalDiscountPercent,
		DiscountType:         reqItems.DiscountType,
		TotalVat:             reqItems.TotalVat,
		SubtotalExclVat:      reqItems.SubtotalExclVat,
		WeightUnit:           reqItems.WeightUnit,
		TotalWeight:          reqItems.TotalWeight,
		Status:               reqItems.Status,
		Remark:               reqItems.Remark,
		CreateBy:             user,
		CreateDtm:            now,
		UpdateBy:             user,
		UpdateDtm:            now,
	}
}

func MapBigLotRequestToPrePurchaseModel(req models.CreatePOBigLotRequest) models.PrePurchase {
	user := `system` // TODO: get from ctx
	now := time.Now().UTC()

	prePurchase := models.PrePurchase{
		PurchaseType:    "LOT",
		CompanyCode:     req.CompanyCode,
		SiteCode:        req.SiteCode,
		DocRefType:      "",
		SupplierCode:    req.SupplierCode,
		SupplierName:    req.SupplierName,
		SupplierAddress: req.SupplierAddress,
		SupplierPhone:   req.SupplierPhone,
		SupplierEmail:   req.SupplierEmail,
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
		CreditTerm:      req.CreditTerm,
		CreateBy:        user,
		CreateDtm:       now,
		UpdateBy:        user,
		UpdateDtm:       now,
	}

	return prePurchase
}

func MapPrePurchaseItemsModelToBigLotItemsResponse(prePurchaseItems []models.PrePurchaseItem) ([]models.GetPOBigLotItemResponse, float64) {
	items := []models.GetPOBigLotItemResponse{}
	var sumSubTotalExclDiscountExclVat float64

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
			UnitUom:              item.UnitUom,
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

		sumSubTotalExclDiscountExclVat += item.TotalCost
	}

	return items, sumSubTotalExclDiscountExclVat
}

func MapPrePurchasesModelToBigLotsResponse(prePurchases models.PrePurchase) models.GetPOBigLotResponse {
	items, sumSubTotalExclDiscountExclVat := MapPrePurchaseItemsModelToBigLotItemsResponse(prePurchases.PrePurchaseItems)

	return models.GetPOBigLotResponse{
		ID:                          prePurchases.ID.String(),
		PrePurchaseCode:             prePurchases.PrePurchaseCode,
		PurchaseType:                prePurchases.PurchaseType,
		CompanyCode:                 prePurchases.CompanyCode,
		SiteCode:                    prePurchases.SiteCode,
		SupplierCode:                prePurchases.SupplierCode,
		SupplierName:                prePurchases.SupplierName,
		SupplierAddress:             prePurchases.SupplierAddress,
		SupplierPhone:               prePurchases.SupplierPhone,
		SupplierEmail:               prePurchases.SupplierEmail,
		DeliveryAddress:             prePurchases.DeliveryAddress,
		Status:                      prePurchases.Status,
		TotalAmount:                 prePurchases.TotalAmount,
		TotalWeight:                 prePurchases.TotalWeight,
		TotalDiscount:               prePurchases.TotalDiscount,
		TotalVat:                    prePurchases.TotalVat,
		SubtotalExclDiscountExclVat: sumSubTotalExclDiscountExclVat,
		SubtotalExclVat:             prePurchases.SubtotalExclVat,
		IsApproved:                  prePurchases.IsApproved,
		StatusApprove:               prePurchases.StatusApprove,
		Remark:                      prePurchases.Remark,
		CreditTerm:                  prePurchases.CreditTerm,
		CreateBy:                    prePurchases.CreateBy,
		CreateDtm:                   prePurchases.CreateDtm.Format(time.RFC3339),
		UpdateBy:                    prePurchases.UpdateBy,
		UpdateDtm:                   prePurchases.UpdateDtm.Format(time.RFC3339),
		PrePurchaseItems:            items,
	}
}

func MapUpdatePOBigLotRequestToPrePurchaseItem(reqItem models.UpdatePOBigLotItemRequest, user string, now time.Time, prePurchaseCode string) models.PrePurchaseItem {
	preItem := ""
	if reqItem.PreItem == nil {
		preItem = fmt.Sprintf("%s-%s", prePurchaseCode, time.Now().Format("150405"))
	} else {
		preItem = *reqItem.PreItem
	}

	return models.PrePurchaseItem{
		ID:                   *reqItem.ID,
		PreItem:              preItem,
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
		UnitUom:              reqItem.UnitUom,
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
		CreditTerm:      req.CreditTerm,
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
			MDItemCode:    "CTM-CTM3",
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

// Running code actions
func GeneratePrePurchaseCodes(ctx *gin.Context, count int) ([]string, error) {
	if count <= 0 {
		return []string{}, nil // No pre-purchase to generate codes for
	}

	configCode := "RUNNING_PB"

	getReq := systemConfigService.GetRunningSystemConfigRequest{
		ConfigCode: configCode,
		Count:      count,
	}

	reqJSON, err := json.Marshal(getReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get request: %v", err)
	}

	prePurchaseCodeResponse, err := systemConfigService.GetRunningSystemConfig(ctx, string(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to generate pre-purchase order codes: %v", err)
	}

	updateReq := systemConfigService.UpdateRunningSystemConfigRequest{
		ConfigCode: configCode,
		Count:      count,
	}

	reqUpdateJSON, err := json.Marshal(updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update request: %v", err)
	}

	_, err = systemConfigService.UpdateRunningSystemConfig(ctx, string(reqUpdateJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to update running config: %v", err)
	}

	prePurchaseCodeResult, ok := prePurchaseCodeResponse.(systemConfigService.GetRunningSystemConfigResponse)
	if !ok || len(prePurchaseCodeResult.Data) != count {
		return nil, errors.New("failed to get correct number of pre-purchase order codes from system config")
	}

	return prePurchaseCodeResult.Data, nil
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
