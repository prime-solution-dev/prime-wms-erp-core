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
	saleRepository "prime-erp-core/internal/repositories/invoice"
	approvalService "prime-erp-core/internal/services/approval-service"
	prePurchaseService "prime-erp-core/internal/services/pre-purchase-service"
	systemConfigService "prime-erp-core/internal/services/system-config"
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

	t := time.Now()
	purchaseItem := fmt.Sprintf("%s-%v", purchaseCode, t.UnixNano())
	if req.PurchaseItem != nil {
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
		ProductDesc:          req.ProductDesc,
		ProductGroupCode:     req.ProductGroupCode,
		ProductGroupName:     req.ProductGroupName,
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
		CreditTerm:      req.CreditTerm,
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
		ProductDesc:          item.ProductDesc,
		ProductGroupOneCode:  item.ProductGroupCode,
		ProductGroupOneName:  item.ProductGroupName,
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
		TradingRef:      purchase.TradingRef,
		SupplierCode:    purchase.SupplierCode,
		SupplierName:    purchase.SupplierName,
		SupplierAddress: purchase.SupplierAddress,
		SupplierPhone:   purchase.SupplierPhone,
		SupplierEmail:   purchase.SupplierEmail,
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
		UsedType:        purchase.UsedType,
		UsedStatus:      purchase.UsedStatus,
		Remark:          purchase.Remark,
		CreditTerm:      purchase.CreditTerm,
		CreateBy:        purchase.CreateBy,
		CreateDtm:       purchase.CreateDtm.Format(time.RFC3339),
		UpdateBy:        purchase.UpdateBy,
		UpdateDtm:       purchase.UpdateDtm.Format(time.RFC3339),
	}
}

// Running code actions
func GeneratePurchaseCodes(ctx *gin.Context, count int) ([]string, error) {
	if count <= 0 {
		return []string{}, nil // No purchases to generate codes for
	}

	configCode := "RUNNING_PO"

	getReq := systemConfigService.GetRunningSystemConfigRequest{
		ConfigCode: configCode,
		Count:      count,
	}

	reqJSON, err := json.Marshal(getReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get request: %v", err)
	}

	purchaseCodeResponse, err := systemConfigService.GetRunningSystemConfig(ctx, string(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to generate purchase order codes: %v", err)
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

	purchaseCodeResult, ok := purchaseCodeResponse.(systemConfigService.GetRunningSystemConfigResponse)
	if !ok || len(purchaseCodeResult.Data) != count {
		return nil, errors.New("failed to get correct number of purchase order codes from system config")
	}

	return purchaseCodeResult.Data, nil
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
	fmt.Println(string(jsonData))

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
func GetProductInterface(productReq models.GetProductRequest) (map[string]models.ProductInterface, error) {
	jsonData, err := json.Marshal(productReq)
	if err != nil {
		return nil, errors.New("failed to marshal product data to JSON: " + err.Error())
	}

	getProducts, err := http.NewRequest("POST", os.Getenv("base_url_product")+"/Product/get-product-interface", bytes.NewBuffer(jsonData))
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

	productResponse := models.ResultProductInterface{}
	if err := json.Unmarshal(productBody, &productResponse); err != nil {
		return nil, errors.New("failed to decode JSON response: " + err.Error())
	}

	mapProduct := map[string]models.ProductInterface{}
	for _, product := range productResponse.ProductInterface {

		mapProduct[product.ProductCode] = product
	}

	return mapProduct, nil
}
func GetMovingAvgCost(productReq models.GetProductRequest) (map[string]models.MovingAvgCost, error) {
	jsonData, err := json.Marshal(productReq)
	if err != nil {
		return nil, errors.New("failed to marshal product data to JSON: " + err.Error())
	}

	getProducts, err := http.NewRequest("POST", os.Getenv("base_url_product")+"/Product/get-moving-avg-cost", bytes.NewBuffer(jsonData))
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

	productResponse := models.ResultMovingAvgCost{}
	if err := json.Unmarshal(productBody, &productResponse); err != nil {
		return nil, errors.New("failed to decode JSON response: " + err.Error())
	}

	mapProduct := map[string]models.MovingAvgCost{}
	for _, product := range productResponse.Unit {

		mapProduct[product.ProductCode] = product
	}

	return mapProduct, nil
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

// Invoice actions
type UsedPOByInvoice struct {
	Qty    float64 `json:"qty"`
	Weight float64 `json:"weight"`
}

func GetUsedQtyAndWeight(companyCode string, siteCode string, purchaseCodes []string, purchaseItemCodes []string) (map[string]UsedPOByInvoice, error) {
	invoices, err := saleRepository.GetInvoiceRelatedByPO(companyCode, siteCode, purchaseCodes, purchaseItemCodes, []string{"AP"}, []string{"PENDING", "COMPLETED"})
	if err != nil {
		return nil, errors.New("failed to get related invoices: " + err.Error())
	}

	usedMap := make(map[string]UsedPOByInvoice)
	for _, invoice := range invoices {
		for _, invoiceItem := range invoice.InvoiceItem {
			if val, exists := usedMap[invoiceItem.DocumentRefItem]; exists {
				val.Qty += invoiceItem.Qty
				val.Weight += invoiceItem.Weight
				usedMap[invoiceItem.DocumentRefItem] = val
			} else {
				usedMap[invoiceItem.DocumentRefItem] = UsedPOByInvoice{
					Qty:    invoiceItem.Qty,
					Weight: invoiceItem.Weight,
				}
			}
		}
	}

	return usedMap, nil
}
