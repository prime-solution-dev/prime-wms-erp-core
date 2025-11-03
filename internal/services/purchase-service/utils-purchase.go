package purchaseService

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
	prePurchaseService "prime-erp-core/internal/services/pre-purchase-service"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func MapPurchaseItemFormRequestToPurchaseItemModel(req models.PurchaseItemFormRequest, purchaseCode string) models.PurchaseItem {
	now := time.Now().UTC()

	id := uuid.New()
	if req.ID != nil {
		id = *req.ID
	}

	createBy := "system"
	if req.CreateBy != nil {
		createBy = *req.CreateBy
	}

	createDtm := now
	if req.CreateDtm != nil {
		createDtm = *req.CreateDtm
	}

	purchaseItem := fmt.Sprintf("%s-%s", purchaseCode, time.Now().Format("150405"))
	if req.PurchaseItem != nil && purchaseCode != "" {
		purchaseItem = *req.PurchaseItem
	}

	docRefItem := ""
	if req.DocRefItem != nil {
		docRefItem = *req.DocRefItem
	}

	return models.PurchaseItem{
		ID:                   id,
		PurchaseItem:         purchaseItem,
		DocRefItem:           docRefItem,
		ProductCode:          req.ProductCode,
		Qty:                  req.Qty,
		Unit:                 req.Unit,
		PurchaseQty:          req.PurchaseQty,
		PurchaseUnit:         req.PurchaseUnit,
		PurchaseUnitType:     req.PurchaseUnitType,
		PriceUnit:            req.PriceUnit,
		TotalDiscount:        req.TotalDiscount,
		TotalAmount:          req.TotalAmount,
		UnitUom:              req.UnitUom,
		TotalCost:            req.TotalCost,
		TotalDiscountPercent: req.TotalDiscountPercent,
		DiscountType:         req.DiscountType,
		TotalVat:             req.TotalVat,
		SubtotalExclVat:      req.SubtotalExclVat,
		WeightUnit:           req.WeightUnit,
		TotalWeight:          req.TotalWeight,
		Status:               req.Status,
		Remark:               req.Remark,
		CreateBy:             createBy,
		CreateDtm:            createDtm,
		UpdateDtm:            now,
		UpdateBy:             "system",
	}
}

func MapPurchaseFormRequestToPurchaseModel(req models.PurchaseFormRequest) models.Purchase {
	now := time.Now().UTC()
	deliveryDate := &time.Time{}
	if req.DeliveryDate != nil {
		utcDate := req.DeliveryDate.UTC()
		deliveryDate = &utcDate
	}

	return models.Purchase{
		PurchaseType:    req.PurchaseType,
		DeliveryDate:    deliveryDate,
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
		UpdateBy:        "system",
		UpdateDtm:       now,
	}
}

func MapPurchaseItemModelToPurchaseItemResponse(item models.PurchaseItem) models.PurchaseItemResponse {
	return models.PurchaseItemResponse{
		ID:                   item.ID.String(),
		PurchaseID:           item.PurchaseID.String(),
		PurchaseItem:         item.PurchaseItem,
		DocRefItem:           item.DocRefItem,
		ProductCode:          item.ProductCode,
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
		TotalVat:             item.TotalVat,
		SubtotalExclVat:      item.SubtotalExclVat,
		WeightUnit:           item.WeightUnit,
		TotalWeight:          item.TotalWeight,
		Status:               item.Status,
		StatusPayment:        item.StatusPayment,
		Remark:               item.Remark,
		CreateDtm:            item.CreateDtm.Format(time.RFC3339),
		CreateBy:             item.CreateBy,
		UpdateDtm:            item.UpdateDtm.Format(time.RFC3339),
		UpdateBy:             item.UpdateBy,
	}
}

func MapPurchaseModelToPurchaseResponse(purchase models.Purchase) models.PurchaseResponse {
	docRefType := ""
	if purchase.DocRefType != nil {
		docRefType = *purchase.DocRefType
	}

	docRef := ""
	if purchase.DocRef != nil {
		docRef = *purchase.DocRef
	}

	return models.PurchaseResponse{
		ID:              purchase.ID.String(),
		PurchaseCode:    purchase.PurchaseCode,
		PurchaseType:    purchase.PurchaseType,
		CompanyCode:     purchase.CompanyCode,
		SiteCode:        purchase.SiteCode,
		DocRefType:      &docRefType,
		DocRef:          &docRef,
		SupplierCode:    purchase.SupplierCode,
		DeliveryDate:    purchase.DeliveryDate.Format(time.RFC3339),
		DeliveryAddress: purchase.DeliveryAddress,
		Status:          purchase.Status,
		TotalAmount:     purchase.TotalAmount,
		TotalWeight:     purchase.TotalWeight,
		TotalDiscount:   purchase.TotalDiscount,
		TotalVat:        purchase.TotalVat,
		SubtotalExclVat: purchase.SubtotalExclVat,
		IsApproved:      purchase.IsApproved,
		StatusApprove:   purchase.StatusApprove,
		StatusPayment:   purchase.StatusPayment,
		Remark:          purchase.Remark,
		CreateBy:        purchase.CreateBy,
		CreateDtm:       purchase.CreateDtm.Format(time.RFC3339),
		UpdateBy:        purchase.UpdateBy,
		UpdateDtm:       purchase.UpdateDtm.Format(time.RFC3339),
	}
}

// System Config
// purchase_code ex. PO-PRE20250930-0001; value ex. 20250930-0001
func GetPurchaseCodeConfig() ([]models.SystemConfig, error) {
	topicCodes := []string{"PO"}
	configCodes := []string{}

	purchaseConfigs, err := systemConfigRepository.GetSystemConfig(topicCodes, configCodes)
	if err != nil {
		return nil, err
	}

	return purchaseConfigs, nil
}

func ConvertConfigToLatestPurchaseNumber(value string) (int, error) {
	if len(value) < 1 {
		return 0, nil
	}

	result := 0

	valueParts := strings.Split(value, "-")
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

// Approval actions
func CreatePurchaseApproval(ctx *gin.Context, purchases []models.Purchase) error {
	user := `system` // TODO: get from ctx

	approvalReq := []models.Approval{}

	for _, p := range purchases {
		approvalReq = append(approvalReq, models.Approval{
			ApproveTopic:  "PO",
			DocumentType:  p.PurchaseType,
			DocumentCode:  p.PurchaseCode,
			ActionDate:    time.Now(),
			Status:        p.StatusApprove,
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

func UpdatePOToApproval(ctx *gin.Context, updateReqs []models.UpdateStatusApprovePurchaseRequest) error {
	purchaseCodes := []string{}
	mapUpdateList := make(map[string]models.Approval)

	for _, req := range updateReqs {
		purchaseCodes = append(purchaseCodes, req.PurchaseCode)
		mapUpdateList[req.PurchaseCode] = models.Approval{
			DocumentCode: req.PurchaseCode,
			Status:       req.StatusApprove,
		}
	}

	if err := prePurchaseService.UpdatePOApproval(ctx, purchaseCodes, mapUpdateList); err != nil {
		return errors.New("failed update approvals: " + err.Error())
	}

	return nil
}

// Product actions
func GetProductByCode(productReq models.GetProductRequest) (map[string]models.GetProductsDetailComponent, error) {
	jsonData, err := json.Marshal(productReq)
	if err != nil {
		return nil, errors.New("failed to marshal product data to JSON: " + err.Error())
	}

	getProducts, err := http.NewRequest("POST", os.Getenv("base_url_product")+"/Product/GetProductDetail", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, errors.New("failed to create HTTP request: " + err.Error())
	}

	getProducts.Header.Set("Content-Type", "application/json")

	// Create a client and execute the request
	client := &http.Client{}
	resp, err := client.Do(getProducts)
	if err != nil {
		return nil, errors.New("failed to execute HTTP request: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("received non-OK HTTP status: " + resp.Status)
	}

	productBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("failed to read response body: " + err.Error())
	}

	productResponse := models.GetProductsDetailResponse{}
	if err := json.Unmarshal(productBody, &productResponse); err != nil {
		return nil, errors.New("failed to decode JSON response: " + err.Error())
	}

	mapProduct := map[string]models.GetProductsDetailComponent{}
	for _, product := range productResponse.Products {
		groups := []models.ProductGroup{}
		for _, g := range product.ProductGroups {
			if g.Seq == 1 {
				groups = append(groups, g)
			}

		}
		product.ProductGroups = groups

		mapProduct[product.ProductCode] = product
	}

	return mapProduct, nil
}

func GetProductGroup(productGroupReq models.GetGroupRequest) (map[string]string, error) {
	jsonData, err := json.Marshal(productGroupReq)
	if err != nil {
		return nil, errors.New("failed to marshal product group data to JSON: " + err.Error())
	}

	getProductGroups, err := http.NewRequest("POST", os.Getenv("base_url_product")+"/Product/GetGroup", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, errors.New("failed to create HTTP request: " + err.Error())
	}

	getProductGroups.Header.Set("Content-Type", "application/json")

	// Create a client and execute the request
	client := &http.Client{}
	resp, err := client.Do(getProductGroups)
	if err != nil {
		return nil, errors.New("failed to execute HTTP request: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("received non-OK HTTP status: " + resp.Status)
	}

	productGroupBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("failed to read response body: " + err.Error())
	}

	fmt.Println("productGroupBody: ", productGroupBody)

	productGroupResponse := []models.GetGroupResponse{}
	if err := json.Unmarshal(productGroupBody, &productGroupResponse); err != nil {
		return nil, errors.New("failed to decode JSON response: " + err.Error())
	}

	mapProductGroupItem := map[string]string{}
	for _, groupItem := range productGroupResponse[0].Items {
		mapProductGroupItem[groupItem.ItemCode] = groupItem.ItemName
	}

	return mapProductGroupItem, nil
}

// PrePurchase actions
func GetRelatedPrePurchase(ctx *gin.Context, req models.GetPOBigLotListRequest) (map[string]models.GetPOBigLotResponse, error) {
	prePurchaseReqJson, err := json.Marshal(req)
	if err != nil {
		return nil, errors.New("failed to marshal pre purchase request to JSON: " + err.Error())
	}

	prePurchaseRespRaw, err := prePurchaseService.GetPOBigLot(ctx, string(prePurchaseReqJson))
	if err != nil {
		return nil, errors.New("failed to get pre purchase list: " + err.Error())
	}

	var prePurchaseResp models.GetPOBigLotListResponse
	switch v := prePurchaseRespRaw.(type) {
	case models.GetPOBigLotListResponse:
		prePurchaseResp = v
	case *models.GetPOBigLotListResponse:
		prePurchaseResp = *v
	default:
		return nil, errors.New("unexpected pre purchase response type")
	}

	mapPrePurchase := make(map[string]models.GetPOBigLotResponse)
	for _, prePurchase := range prePurchaseResp.BigLotList {
		mapPrePurchase[prePurchase.PrePurchaseCode] = prePurchase
	}

	return mapPrePurchase, nil
}
