package purchaseService

import (
	"encoding/json"
	"errors"
	"fmt"
	"prime-erp-core/internal/models"
	approvalService "prime-erp-core/internal/services/approval-service"
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
		PrePurchaseCode: req.PrePurchaseCode,
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

func MapPrePurchaseItemsModelToBigLotItemsResponse(prePurchaseItems []models.PrePurchaseItem) []models.GetPOBigLotItemResponse {
	items := make([]models.GetPOBigLotItemResponse, 0, len(prePurchaseItems))

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
