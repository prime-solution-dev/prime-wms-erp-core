package xService

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	goodsReceiveService "prime-erp-core/external/goods-receive-service"
	externalProductService "prime-erp-core/external/product-service"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GetPurchaseItemRemainRequest struct {
	CompanyCode      string            `json:"company_code"`
	SiteCode         string            `json:"site_code"`
	PurchaseCodes    []string          `json:"purchase_codes"`
	SupplierCodes    []string          `json:"supplier_codes"`
	Status           []string          `json:"status"`
	StatusApprove    []string          `json:"status_approve"`
	StattusPayment   []string          `json:"stattus_payment"`
	ProductCodes     []string          `json:"product_codes"`
	NotPurchaseItems []NotPurchaseItem `json:"not_purchase_items,omitempty"`
	Page             *int              `json:"page"`
	PageSize         *int              `json:"limit"`
	PurchaseCodeLike string            `json:"purchase_code_like,omitempty"`
	ProductCodeLike  string            `json:"product_code_like,omitempty"`
	ProductNameLike  string            `json:"product_name_like,omitempty"`
}

type NotPurchaseItem struct {
	PurchaseCode string `json:"purchase_code"`
	PurchaseItem string `json:"purchase_item"`
}

type GetPurchaseItemRemainResponse struct {
	ResponseCode int                                   `json:"response_code"`
	Message      string                                `json:"message"`
	TotalPage    *int                                  `json:"total_page"`
	PageSize     *int                                  `json:"page_size"`
	Page         *int                                  `json:"page"`
	Total        *int                                  `json:"total"`
	Daatas       []GetPurchaseItemRemainResponseResult `json:"daatas"`
}

type GetPurchaseItemRemainResponseResult struct {
	PurchaseID           string  `json:"purchase_id"`
	PurchaseCode         string  `json:"purchase_code"`
	PurchaseType         string  `json:"purchase_type"`
	SupplierCode         string  `json:"supplier_code"`
	SupplierName         string  `json:"supplier_name"`
	Status               string  `json:"status"`
	StatusApprove        string  `json:"status_approve"`
	StatusPayment        string  `json:"status_payment"`
	DeliveryDate         string  `json:"delivery_date"`
	ID                   string  `json:"id"`
	PurchaseItem         string  `json:"purchase_item"`
	DocRefItem           string  `json:"doc_ref_item"`
	ProductCode          string  `json:"product_code"`
	ProductDesc          string  `json:"product_desc"`
	ProductName          string  `json:"product_name"`
	ProductGroupOneCode  string  `json:"product_group_one_code"`
	ProductGroupOneName  string  `json:"product_group_one_name"`
	Qty                  float64 `json:"qty"`
	RemainQty            float64 `json:"remain_qty"`
	PurchaseQty          float64 `json:"purchase_qty"`
	Unit                 string  `json:"unit"`
	PurchaseUnit         string  `json:"purchase_unit"`
	PurchaseUnitType     string  `json:"purchase_unit_type"`
	PriceUnit            float64 `json:"price_unit"`
	TotalDiscount        float64 `json:"total_discount"`
	TotalAmount          float64 `json:"total_amount"`
	UnitUom              string  `json:"unit_uom"`
	TotalCost            float64 `json:"total_cost"`
	TotalDiscountPercent float64 `json:"total_discount_percent"`
	DiscountType         string  `json:"discount_type"`
	TotalVat             float64 `json:"total_vat"`
	SubtotalExclVat      float64 `json:"subtotal_excl_vat"`
	WeightUnit           float64 `json:"weight_unit"`
	ActualWeightUnit     float64 `json:"actual_weight_unit"`
	TotalWeight          float64 `json:"total_weight"`
	StatusItem           string  `json:"status_item"`
	StatusPaymentItem    string  `json:"status_payment_item"`
	Remark               string  `json:"remark"`
	CreateDtm            string  `json:"create_dtm"`
	CreateBy             string  `json:"create_by"`
	UpdateDtm            string  `json:"update_dtm"`
	UpdateBy             string  `json:"update_by"`
}

type documentData struct {
	DocumentCode       string
	DocumentItem       string
	DocumentRef        string
	DocumentRefItem    string
	DocumentSource     string
	DocumentSourceItem string
	Qty                float64
	UnitCode           string
}

func GetPurchaseItemRemainRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := GetPurchaseItemRemainRequest{}
	if strings.TrimSpace(jsonPayload) != "" {
		if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
			return nil, fmt.Errorf("invalid JSON payload: %w", err)
		}
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	return GetPurchaseItemRemain(ctx, gormx, req)
}

func GetPurchaseItemRemain(ctx *gin.Context, gormx *gorm.DB, req GetPurchaseItemRemainRequest) (*GetPurchaseItemRemainResponse, error) {
	res := GetPurchaseItemRemainResponse{
		ResponseCode: 200,
		Message:      "success",
		Daatas:       []GetPurchaseItemRemainResponseResult{},
	}

	if strings.TrimSpace(req.CompanyCode) == "" {
		return nil, fmt.Errorf("company_code is required")
	}
	if strings.TrimSpace(req.SiteCode) == "" {
		return nil, fmt.Errorf("site_code is required")
	}

	productSet := makeStringSetTrimUpper(req.ProductCodes)

	poCodes := []string{}
	poCodesCheck := map[string]bool{}
	poItems := []string{}
	poItemsCheck := map[string]bool{}

	poMap, err := getPurchase(gormx, req)
	if err != nil {
		return nil, err
	}

	if len(poMap) == 0 {
		page, pageSize, total, totalPages := buildEmptyPagination(req.Page, req.PageSize)
		res.Page = &page
		res.PageSize = &pageSize
		res.Total = &total
		res.TotalPage = &totalPages
		return &res, nil
	}

	for _, po := range poMap {
		poCode := strings.TrimSpace(po.PurchaseCode)
		if poCode == "" {
			continue
		}
		if _, ok := poCodesCheck[poCode]; !ok {
			poCodesCheck[poCode] = true
			poCodes = append(poCodes, poCode)
		}

		for _, poi := range po.Items {
			if len(productSet) > 0 && !inSetTrimUpper(productSet, poi.ProductCode) {
				continue
			}
			poItem := strings.TrimSpace(poi.PurchaseItem)
			if poItem == "" {
				continue
			}
			if _, ok := poItemsCheck[poItem]; !ok {
				poItemsCheck[poItem] = true
				poItems = append(poItems, poItem)
			}
		}
	}

	if len(productSet) > 0 && len(poItems) == 0 {
		page, pageSize, total, totalPages := buildEmptyPagination(req.Page, req.PageSize)
		res.Page = &page
		res.PageSize = &pageSize
		res.Total = &total
		res.TotalPage = &totalPages
		return &res, nil
	}

	ibDocMap, err := getInbound(poCodes, poItems)
	if err != nil {
		return nil, err
	}

	ibCodes := []string{}
	ibCodesCheck := map[string]bool{}
	ibItems := []string{}
	ibItemsCheck := map[string]bool{}

	ibDocMapPo := map[string]documentData{}
	grDocMap := map[string]documentData{}
	grRemainMapPo := map[string]documentData{}
	ignoreSourceAp := map[string]bool{}
	ibToPO := map[string][2]string{}

	if len(ibDocMap) > 0 {
		for _, ib := range ibDocMap {
			ibCode := strings.TrimSpace(ib.DocumentCode)
			ibItem := strings.TrimSpace(ib.DocumentItem)

			if ibCode != "" {
				if _, ok := ibCodesCheck[ibCode]; !ok {
					ibCodesCheck[ibCode] = true
					ibCodes = append(ibCodes, ibCode)
				}
			}
			if ibItem != "" {
				if _, ok := ibItemsCheck[ibItem]; !ok {
					ibItemsCheck[ibItem] = true
					ibItems = append(ibItems, ibItem)
				}
			}

			poCode := strings.TrimSpace(ib.DocumentRef)
			poItem := strings.TrimSpace(ib.DocumentRefItem)
			if ibCode != "" && ibItem != "" && poCode != "" && poItem != "" {
				key := fmt.Sprintf("%s|%s", ibCode, ibItem)
				ibToPO[key] = [2]string{poCode, poItem}
			}
		}

		ibDocMapPo, err = ComputeSumInbound(ibDocMap)
		if err != nil {
			return nil, err
		}

		grDocMap, err = getGoodsReceive(ibCodes, ibItems)
		if err != nil {
			return nil, err
		}
	}

	apDocMap, err := getInvoiceAp(gormx, req, poCodes, poItems)
	if err != nil {
		return nil, err
	}

	if len(grDocMap) > 0 {
		grRemainMapPo, ignoreSourceAp, err = ComputeReceiveRemain(grDocMap, apDocMap, ibToPO)
		if err != nil {
			return nil, err
		}
	}

	apDocMapPo, err := ComputeApRemain(apDocMap, ignoreSourceAp)
	if err != nil {
		return nil, err
	}

	remainMap := ComputePurchaseRemainQty(poMap, ibDocMapPo, apDocMapPo, grRemainMapPo, productSet)

	results, err := ConvertToResponse(poMap, remainMap, productSet, nil)
	if err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].PurchaseCode == results[j].PurchaseCode {
			return results[i].PurchaseItem < results[j].PurchaseItem
		}
		return results[i].PurchaseCode < results[j].PurchaseCode
	})

	paged, page, pageSize, total, totalPages := paginateResults(results, req.Page, req.PageSize)
	productMasterMap, err := getProductMasterMap(req, paged)
	if err != nil {
		return nil, err
	}
	enrichProductMaster(paged, productMasterMap)

	res.Daatas = paged
	res.Page = &page
	res.PageSize = &pageSize
	res.Total = &total
	res.TotalPage = &totalPages

	return &res, nil
}

func ComputeSumInbound(ibDocMap map[string]documentData) (map[string]documentData, error) {
	ibDocMapPo := map[string]documentData{}

	for _, ib := range ibDocMap {
		poCode := strings.TrimSpace(ib.DocumentRef)
		poItem := strings.TrimSpace(ib.DocumentRefItem)
		if poCode == "" || poItem == "" {
			continue
		}

		key := fmt.Sprintf("%s|%s", poCode, poItem)
		doc, ok := ibDocMapPo[key]
		if !ok {
			doc = documentData{
				DocumentCode: poCode,
				DocumentItem: poItem,
				Qty:          0,
			}
		}
		doc.Qty += ib.Qty
		ibDocMapPo[key] = doc
	}

	return ibDocMapPo, nil
}

func ComputeReceiveRemain(
	grDocMap map[string]documentData,
	apDocMap map[string]documentData,
	ibToPO map[string][2]string,
) (map[string]documentData, map[string]bool, error) {

	rs := map[string]documentData{}
	ignoreSourceAp := map[string]bool{}

	apBy4Key := map[string]float64{}
	for _, ap := range apDocMap {
		poCode := strings.TrimSpace(ap.DocumentRef)
		poItem := strings.TrimSpace(ap.DocumentRefItem)
		grCode := strings.TrimSpace(ap.DocumentSource)
		grItem := strings.TrimSpace(ap.DocumentSourceItem)
		if poCode == "" || poItem == "" || grCode == "" || grItem == "" {
			continue
		}
		k4 := fmt.Sprintf("%s|%s|%s|%s", poCode, poItem, grCode, grItem)
		apBy4Key[k4] += ap.Qty
	}

	for _, gr := range grDocMap {
		grCode := strings.TrimSpace(gr.DocumentCode)
		grItem := strings.TrimSpace(gr.DocumentItem)
		ibCode := strings.TrimSpace(gr.DocumentRef)
		ibItem := strings.TrimSpace(gr.DocumentRefItem)

		if grCode == "" || grItem == "" || ibCode == "" || ibItem == "" {
			continue
		}

		poPair, ok := ibToPO[fmt.Sprintf("%s|%s", ibCode, ibItem)]
		if !ok {
			continue
		}

		poCode := strings.TrimSpace(poPair[0])
		poItem := strings.TrimSpace(poPair[1])
		if poCode == "" || poItem == "" {
			continue
		}

		k4 := fmt.Sprintf("%s|%s|%s|%s", poCode, poItem, grCode, grItem)
		apQty, hasAP := apBy4Key[k4]

		diff := gr.Qty - apQty
		if diff < 0 {
			diff = 0
		}

		poKey := fmt.Sprintf("%s|%s", poCode, poItem)
		doc, ok := rs[poKey]
		if !ok {
			doc = documentData{
				DocumentCode: poCode,
				DocumentItem: poItem,
				Qty:          0,
			}
		}
		doc.Qty += diff
		rs[poKey] = doc

		if hasAP && apQty > 0 {
			ignoreSourceAp[k4] = true
		}
	}

	return rs, ignoreSourceAp, nil
}

func ComputeApRemain(apDocMap map[string]documentData, ignoreSourceAp map[string]bool) (map[string]documentData, error) {
	apDocMapPo := map[string]documentData{}

	for _, ap := range apDocMap {
		poCode := strings.TrimSpace(ap.DocumentRef)
		poItem := strings.TrimSpace(ap.DocumentRefItem)
		grCode := strings.TrimSpace(ap.DocumentSource)
		grItem := strings.TrimSpace(ap.DocumentSourceItem)

		if poCode == "" || poItem == "" {
			continue
		}

		ignoreKey := fmt.Sprintf("%s|%s|%s|%s", poCode, poItem, grCode, grItem)
		if _, ok := ignoreSourceAp[ignoreKey]; ok {
			continue
		}

		key := fmt.Sprintf("%s|%s", poCode, poItem)
		doc, ok := apDocMapPo[key]
		if !ok {
			doc = documentData{
				DocumentCode: poCode,
				DocumentItem: poItem,
				Qty:          0,
			}
		}
		doc.Qty += ap.Qty
		apDocMapPo[key] = doc
	}

	return apDocMapPo, nil
}

func ComputePurchaseRemainQty(
	poMap map[string]models.PurchaseResponse,
	ibDocMapPo map[string]documentData,
	apDocMapPo map[string]documentData,
	grRemainMapPo map[string]documentData,
	productSet map[string]bool,
) map[string]float64 {

	remainMap := map[string]float64{}

	for poCode, po := range poMap {
		for _, it := range po.Items {
			if len(productSet) > 0 && !inSetTrimUpper(productSet, it.ProductCode) {
				continue
			}

			poItem := strings.TrimSpace(it.PurchaseItem)
			if poItem == "" {
				continue
			}

			key := fmt.Sprintf("%s|%s", strings.TrimSpace(poCode), poItem)

			poQty := it.Qty

			ibQty := 0.0
			if d, ok := ibDocMapPo[key]; ok {
				ibQty = d.Qty
			}

			apQty := 0.0
			if d, ok := apDocMapPo[key]; ok {
				apQty = d.Qty
			}

			grDiff := 0.0
			if d, ok := grRemainMapPo[key]; ok {
				grDiff = d.Qty
			}

			remain := poQty - apQty - ibQty + grDiff
			if remain < 0 {
				remain = 0
			}

			remainMap[key] = remain
		}
	}

	return remainMap
}

func ConvertToResponse(
	poMap map[string]models.PurchaseResponse,
	remainMap map[string]float64,
	productSet map[string]bool,
	productMasterMap map[string]externalProductService.GetProductsComponent,
) ([]GetPurchaseItemRemainResponseResult, error) {

	rs := []GetPurchaseItemRemainResponseResult{}

	for _, po := range poMap {
		for _, it := range po.Items {
			if len(productSet) > 0 && !inSetTrimUpper(productSet, it.ProductCode) {
				continue
			}

			key := fmt.Sprintf("%s|%s", strings.TrimSpace(po.PurchaseCode), strings.TrimSpace(it.PurchaseItem))
			remain := remainMap[key]
			if remain <= 0 {
				continue
			}

			productDesc := it.ProductDesc
			productName := ""

			if pm, ok := productMasterMap[strings.ToUpper(strings.TrimSpace(it.ProductCode))]; ok {
				productName = strings.TrimSpace(pm.ProductName)
				if productName != "" {
					productDesc = productName
				}
			}

			actualWeightUnit := 0.0
			if it.TotalWeight > 0 {
				actualWeightUnit = it.Qty / it.TotalWeight
			}

			r := GetPurchaseItemRemainResponseResult{
				PurchaseID:           po.ID,
				PurchaseCode:         po.PurchaseCode,
				PurchaseType:         po.PurchaseType,
				SupplierCode:         po.SupplierCode,
				SupplierName:         po.SupplierName,
				Status:               po.Status,
				StatusApprove:        po.StatusApprove,
				StatusPayment:        po.StatusPayment,
				DeliveryDate:         po.DeliveryDate,
				ID:                   it.ID,
				PurchaseItem:         it.PurchaseItem,
				DocRefItem:           it.DocRefItem,
				ProductCode:          it.ProductCode,
				ProductDesc:          productDesc,
				ProductName:          productName,
				ProductGroupOneCode:  it.ProductGroupOneCode,
				ProductGroupOneName:  it.ProductGroupOneName,
				Qty:                  it.Qty,
				RemainQty:            remain,
				PurchaseQty:          it.PurchaseQty,
				Unit:                 it.Unit,
				PurchaseUnit:         it.PurchaseUnit,
				PurchaseUnitType:     it.PurchaseUnitType,
				PriceUnit:            it.PriceUnit,
				TotalDiscount:        it.TotalDiscount,
				TotalAmount:          it.TotalAmount,
				UnitUom:              it.UnitUom,
				TotalCost:            it.TotalCost,
				TotalDiscountPercent: it.TotalDiscountPercent,
				DiscountType:         it.DiscountType,
				TotalVat:             it.TotalVat,
				SubtotalExclVat:      it.SubtotalExclVat,
				WeightUnit:           it.WeightUnit,
				ActualWeightUnit:     actualWeightUnit,
				TotalWeight:          it.TotalWeight,
				StatusItem:           it.Status,
				StatusPaymentItem:    it.StatusPayment,
				Remark:               it.Remark,
				CreateDtm:            it.CreateDtm,
				CreateBy:             it.CreateBy,
				UpdateDtm:            it.UpdateDtm,
				UpdateBy:             it.UpdateBy,
			}

			rs = append(rs, r)
		}
	}

	return rs, nil
}

func getPurchase(gormx *gorm.DB, req GetPurchaseItemRemainRequest) (map[string]models.PurchaseResponse, error) {
	rs := map[string]models.PurchaseResponse{}

	company := strings.TrimSpace(req.CompanyCode)
	site := strings.TrimSpace(req.SiteCode)

	poSet := makeStringSetTrimUpper(req.PurchaseCodes)
	suppSet := makeStringSetTrimUpper(req.SupplierCodes)
	prodSet := makeStringSetTrimUpper(req.ProductCodes)
	approveSet := makeStringSetTrimUpper(req.StatusApprove)
	paySet := makeStringSetTrimUpper(req.StattusPayment)
	notPairs := normalizeNotPurchasePairs(req.NotPurchaseItems)
	statusSet := makeStringSetTrimUpper(req.Status)
	productNameSet := map[string]bool{}

	if strings.TrimSpace(req.ProductNameLike) != "" {
		var err error
		productNameSet, err = getProductCodesByNameLike(req)
		if err != nil {
			return rs, err
		}
		if len(productNameSet) == 0 {
			return rs, nil
		}

		if len(prodSet) > 0 {
			prodSet = intersectStringSets(prodSet, productNameSet)
			if len(prodSet) == 0 {
				return rs, nil
			}
		} else {
			prodSet = productNameSet
		}
	}

	q := gormx.Model(&models.Purchase{}).
		Where("company_code = ? AND site_code = ?", company, site)

	if len(statusSet) > 0 {
		q = q.Where("UPPER(LTRIM(RTRIM(status))) IN ?", setToSlice(statusSet))
	} else {
		q = q.Where("status = ?", "PENDING")
	}

	if len(poSet) > 0 {
		q = q.Where("UPPER(LTRIM(RTRIM(purchase_code))) IN ?", setToSlice(poSet))
	}
	if len(suppSet) > 0 {
		q = q.Where("UPPER(LTRIM(RTRIM(supplier_code))) IN ?", setToSlice(suppSet))
	}
	if len(approveSet) > 0 {
		q = q.Where("UPPER(LTRIM(RTRIM(status_approve))) IN ?", setToSlice(approveSet))
	}
	if len(paySet) > 0 {
		q = q.Where(
			"COALESCE(NULLIF(UPPER(LTRIM(RTRIM(status_payment))), ''), 'PENDING') IN ?",
			setToSlice(paySet),
		)
	}

	if strings.TrimSpace(req.PurchaseCodeLike) != "" {
		q = q.Where("purchase_code ILIKE ?", "%"+strings.TrimSpace(req.PurchaseCodeLike)+"%")
	}
	if strings.TrimSpace(req.ProductCodeLike) != "" {
		q = q.Where(`
			EXISTS (
				SELECT 1
				FROM purchase_item pi
				WHERE pi.purchase_id = purchase.id
				  AND pi.product_code ILIKE ?
			)
		`, "%"+strings.TrimSpace(req.ProductCodeLike)+"%")
	}
	if len(prodSet) > 0 {
		q = q.Where(`
			EXISTS (
				SELECT 1
				FROM purchase_item pi
				WHERE pi.purchase_id = purchase.id
				  AND UPPER(LTRIM(RTRIM(pi.product_code))) IN ?
			)
		`, setToSlice(prodSet))
	}

	if len(notPairs) > 0 {
		valuesSQL, args := buildValuesPairsSQL(notPairs)
		q = q.Where(fmt.Sprintf(`
			EXISTS (
				SELECT 1
				FROM purchase_item pi
				WHERE pi.purchase_id = purchase.id
				  AND NOT EXISTS (
						SELECT 1
						FROM (VALUES %s) v(po_code, po_item)
						WHERE v.po_code = UPPER(LTRIM(RTRIM(purchase.purchase_code)))
						  AND v.po_item = UPPER(LTRIM(RTRIM(pi.purchase_item)))
				  )
			)
		`, valuesSQL), args...)
	}

	var purchases []models.Purchase
	if err := q.
		Select([]string{
			"id", "purchase_code", "purchase_type",
			"company_code", "site_code",
			"supplier_code", "supplier_name",
			"delivery_date",
			"status", "status_approve", "status_payment",
			"total_amount", "total_weight", "total_discount", "total_vat", "subtotal_excl_vat",
			"remark",
			"create_by", "create_dtm", "update_by", "update_dtm",
		}).
		Order("purchase_code ASC").
		Find(&purchases).Error; err != nil {
		return rs, err
	}

	if len(purchases) == 0 {
		return rs, nil
	}

	ids := make([]string, 0, len(purchases))
	purchaseByID := map[string]models.Purchase{}
	for _, p := range purchases {
		idStr := p.ID.String()
		ids = append(ids, idStr)
		purchaseByID[idStr] = p
	}

	itemQ := gormx.Model(&models.PurchaseItem{}).
		Joins("JOIN purchase p ON p.id = purchase_item.purchase_id").
		Where("purchase_item.purchase_id IN ?", ids)

	if len(prodSet) > 0 {
		itemQ = itemQ.Where("UPPER(LTRIM(RTRIM(purchase_item.product_code))) IN ?", setToSlice(prodSet))
	}
	if strings.TrimSpace(req.ProductCodeLike) != "" {
		itemQ = itemQ.Where("purchase_item.product_code ILIKE ?", "%"+strings.TrimSpace(req.ProductCodeLike)+"%")
	}

	if len(notPairs) > 0 {
		valuesSQL, args := buildValuesPairsSQL(notPairs)
		itemQ = itemQ.Where(fmt.Sprintf(`
			NOT EXISTS (
				SELECT 1
				FROM (VALUES %s) v(po_code, po_item)
				WHERE v.po_code = UPPER(LTRIM(RTRIM(p.purchase_code)))
				  AND v.po_item = UPPER(LTRIM(RTRIM(purchase_item.purchase_item)))
			)
		`, valuesSQL), args...)
	}

	var items []models.PurchaseItem
	if err := itemQ.
		Select([]string{
			"purchase_item.id", "purchase_item.purchase_id", "purchase_item.purchase_item",
			"purchase_item.product_code", "purchase_item.product_desc",
			"purchase_item.product_group_code", "purchase_item.product_group_name",
			"purchase_item.doc_ref_item",
			"purchase_item.qty", "purchase_item.unit",
			"purchase_item.purchase_qty", "purchase_item.purchase_unit", "purchase_item.purchase_unit_type",
			"purchase_item.price_unit",
			"purchase_item.total_discount", "purchase_item.total_amount",
			"purchase_item.unit_uom", "purchase_item.total_cost",
			"purchase_item.total_discount_percent", "purchase_item.discount_type",
			"purchase_item.total_vat", "purchase_item.subtotal_excl_vat",
			"purchase_item.weight_unit", "purchase_item.total_weight",
			"purchase_item.status", "purchase_item.status_payment",
			"purchase_item.remark",
			"purchase_item.create_by", "purchase_item.create_dtm", "purchase_item.update_by", "purchase_item.update_dtm",
		}).
		Order("purchase_item.purchase_id ASC, purchase_item.purchase_item ASC").
		Find(&items).Error; err != nil {
		return rs, err
	}

	tmp := map[string]*models.PurchaseResponse{}

	for _, it := range items {
		pid := it.PurchaseID.String()
		p, ok := purchaseByID[pid]
		if !ok {
			continue
		}

		code := strings.TrimSpace(p.PurchaseCode)
		if code == "" {
			continue
		}

		pr, ok := tmp[code]
		if !ok {
			base := toPurchaseResponse(p)
			tmp[code] = &base
			pr = tmp[code]
		}

		pr.Items = append(pr.Items, toPurchaseItemResponse(it, p))
	}

	for code, pr := range tmp {
		if pr == nil || len(pr.Items) == 0 {
			continue
		}
		rs[code] = *pr
	}

	return rs, nil
}

func getInbound(poCodes []string, poItems []string) (map[string]documentData, error) {
	rs := map[string]documentData{}

	reqIb := goodsReceiveService.InboundFilter{
		InboundItemDocumentRef:     poCodes,
		InboundItemDocumentRefItem: poItems,
		Status:                     []string{"PENDING"},
	}

	resIb, err := goodsReceiveService.GetInbounds(reqIb)
	if err != nil {
		return rs, err
	}

	for _, ib := range resIb.InboundRes {
		for _, ibi := range ib.InboundItemRes {
			ibCode := strings.TrimSpace(ib.InboundCode)
			ibItem := strings.TrimSpace(ibi.InboundItem)
			poCode := strings.TrimSpace(ibi.DocumentRef)
			poItem := strings.TrimSpace(ibi.DocumentRefItem)

			if ibCode == "" || ibItem == "" || poCode == "" || poItem == "" {
				continue
			}

			key := fmt.Sprintf("%s|%s", ibCode, ibItem)
			doc, ok := rs[key]
			if !ok {
				doc = documentData{
					DocumentCode:    ibCode,
					DocumentItem:    ibItem,
					DocumentRef:     poCode,
					DocumentRefItem: poItem,
					Qty:             0,
					UnitCode:        strings.TrimSpace(ibi.UnitCode),
				}
			}
			doc.Qty += ibi.Qty
			rs[key] = doc
		}
	}

	return rs, nil
}

func getGoodsReceive(ibCodes []string, ibItems []string) (map[string]documentData, error) {
	rs := map[string]documentData{}

	reqGr := goodsReceiveService.GoodsReceiveFilter{
		ReferenceNo:     ibCodes,
		DocumentRefItem: ibItems,
		Status:          []string{"COMPLETED"},
	}

	resGr, err := goodsReceiveService.GetGoodsReceives(reqGr)
	if err != nil {
		return rs, err
	}

	for _, gr := range resGr.GoodsReceive {
		for _, gri := range gr.GoodsReceiveItem {
			grCode := strings.TrimSpace(gr.ReceiveCode)
			grItem := strings.TrimSpace(gri.ReceiveItem)
			ibCode := strings.TrimSpace(gr.DocumentRef)
			ibItem := strings.TrimSpace(gri.DocumentRefItem)

			if grCode == "" || grItem == "" || ibCode == "" || ibItem == "" {
				continue
			}

			key := fmt.Sprintf("%s|%s", grCode, grItem)
			doc, ok := rs[key]
			if !ok {
				doc = documentData{
					DocumentCode:    grCode,
					DocumentItem:    grItem,
					DocumentRef:     ibCode,
					DocumentRefItem: ibItem,
					Qty:             0,
					UnitCode:        strings.TrimSpace(gri.UnitCode),
				}
			}

			grQty := gri.Qty
			confirmQty := 0.0
			for _, cf := range gri.GoodsReceiveConfirm {
				confirmQty += cf.Qty
			}

			useQty := grQty
			if confirmQty < useQty {
				useQty = confirmQty
			}
			if useQty < 0 {
				useQty = 0
			}

			doc.Qty += useQty
			rs[key] = doc
		}
	}

	return rs, nil
}

func getInvoiceAp(gormx *gorm.DB, req GetPurchaseItemRemainRequest, poCodes []string, poItems []string) (map[string]documentData, error) {
	rs := map[string]documentData{}

	company := strings.TrimSpace(req.CompanyCode)
	site := strings.TrimSpace(req.SiteCode)
	if company == "" || site == "" {
		return rs, fmt.Errorf("company_code/site_code is required")
	}

	poCodesNorm := normalizeTrimUpperSlice(poCodes)
	if len(poCodesNorm) == 0 {
		return rs, nil
	}
	poItemsNorm := normalizeTrimUpperSlice(poItems)

	poCodeSet := makeStringSetTrimUpper(poCodesNorm)
	poItemSet := makeStringSetTrimUpper(poItemsNorm)

	const chunkSize = 500

	type apRow struct {
		InvoiceCode     string  `gorm:"column:invoice_code"`
		InvoiceItem     string  `gorm:"column:invoice_item"`
		DocumentRef     string  `gorm:"column:document_ref"`
		DocumentRefItem string  `gorm:"column:document_ref_item"`
		SourceCode      string  `gorm:"column:source_code"`
		SourceItem      string  `gorm:"column:source_item"`
		Qty             float64 `gorm:"column:qty"`
		UnitCode        string  `gorm:"column:unit_code"`
	}

	queryChunk := func(codeChunk []string) ([]apRow, error) {
		q := gormx.
			Table("invoice_item ii").
			Joins("JOIN invoice i ON i.id = ii.invoice_id").
			Where("i.company_code = ? AND i.site_code = ?", company, site).
			Where("i.status IN ?", []string{"PENDING", "COMPLETED"}).
			Where("UPPER(LTRIM(RTRIM(ii.document_ref))) IN ?", codeChunk)

		if len(poItemsNorm) > 0 {
			q = q.Where("UPPER(LTRIM(RTRIM(ii.document_ref_item))) IN ?", poItemsNorm)
		}

		q = q.Select(`
			i.invoice_code        AS invoice_code,
			ii.invoice_item       AS invoice_item,
			ii.document_ref       AS document_ref,
			ii.document_ref_item  AS document_ref_item,
			ii.source_code        AS source_code,
			ii.source_item        AS source_item,
			ii.qty                AS qty,
			ii.unit_code          AS unit_code
		`)

		rows := []apRow{}
		if err := q.Scan(&rows).Error; err != nil {
			return nil, err
		}
		return rows, nil
	}

	for i := 0; i < len(poCodesNorm); i += chunkSize {
		j := i + chunkSize
		if j > len(poCodesNorm) {
			j = len(poCodesNorm)
		}
		chunk := poCodesNorm[i:j]

		rows, err := queryChunk(chunk)
		if err != nil {
			return rs, err
		}

		for _, r := range rows {
			invCode := strings.TrimSpace(r.InvoiceCode)
			invItem := strings.TrimSpace(r.InvoiceItem)
			if invCode == "" || invItem == "" || r.Qty == 0 {
				continue
			}

			poCode := strings.TrimSpace(r.DocumentRef)
			poItem := strings.TrimSpace(r.DocumentRefItem)

			if len(poCodeSet) > 0 && !inSetTrimUpper(poCodeSet, poCode) {
				continue
			}
			if len(poItemSet) > 0 && !inSetTrimUpper(poItemSet, poItem) {
				continue
			}

			key := invCode + "|" + invItem
			doc := rs[key]
			if doc.DocumentCode == "" {
				doc = documentData{
					DocumentCode:       invCode,
					DocumentItem:       invItem,
					DocumentRef:        poCode,
					DocumentRefItem:    poItem,
					DocumentSource:     strings.TrimSpace(r.SourceCode),
					DocumentSourceItem: strings.TrimSpace(r.SourceItem),
					Qty:                0,
					UnitCode:           strings.TrimSpace(r.UnitCode),
				}
			}
			doc.Qty += r.Qty
			if strings.TrimSpace(doc.UnitCode) == "" && strings.TrimSpace(r.UnitCode) != "" {
				doc.UnitCode = strings.TrimSpace(r.UnitCode)
			}
			rs[key] = doc
		}
	}

	return rs, nil
}

func getProductCodesByNameLike(req GetPurchaseItemRemainRequest) (map[string]bool, error) {
	rs := map[string]bool{}

	company := strings.TrimSpace(req.CompanyCode)
	site := strings.TrimSpace(req.SiteCode)
	productNameLike := strings.TrimSpace(req.ProductNameLike)
	if company == "" || site == "" || productNameLike == "" {
		return rs, nil
	}

	const pageSize = 1000
	totalPages := 1

	for page := 1; page <= totalPages; page++ {
		productRes, err := externalProductService.GetProduct(externalProductService.GetProductRequest{
			CompanyCode:     []string{company},
			SiteCode:        []string{site},
			ProductNameLike: productNameLike,
			Page:            page,
			PageSize:        pageSize,
		})
		if err != nil {
			return rs, err
		}

		if productRes.TotalPages > totalPages {
			totalPages = productRes.TotalPages
		}

		for _, p := range productRes.Products {
			code := strings.ToUpper(strings.TrimSpace(p.ProductCode))
			if code == "" {
				continue
			}
			rs[code] = true
		}

		if productRes.TotalPages <= 0 {
			break
		}
	}

	return rs, nil
}

func getProductMasterMap(
	req GetPurchaseItemRemainRequest,
	results []GetPurchaseItemRemainResponseResult,
) (map[string]externalProductService.GetProductsComponent, error) {
	rs := map[string]externalProductService.GetProductsComponent{}

	productCodes := []string{}
	seen := map[string]bool{}

	for _, result := range results {
		code := strings.TrimSpace(result.ProductCode)
		if code == "" {
			continue
		}
		key := strings.ToUpper(code)
		if seen[key] {
			continue
		}
		seen[key] = true
		productCodes = append(productCodes, code)
	}

	if len(productCodes) == 0 {
		return rs, nil
	}

	pageSize := len(productCodes)
	if pageSize < 1 {
		pageSize = 50
	}

	productRes, err := externalProductService.GetProduct(externalProductService.GetProductRequest{
		CompanyCode: []string{strings.TrimSpace(req.CompanyCode)},
		SiteCode:    []string{strings.TrimSpace(req.SiteCode)},
		ProductCode: productCodes,
		Page:        1,
		PageSize:    pageSize,
	})
	if err != nil {
		return rs, err
	}

	for _, p := range productRes.Products {
		code := strings.TrimSpace(p.ProductCode)
		if code == "" {
			continue
		}
		rs[strings.ToUpper(code)] = p
	}

	return rs, nil
}

func enrichProductMaster(
	results []GetPurchaseItemRemainResponseResult,
	productMasterMap map[string]externalProductService.GetProductsComponent,
) {
	for i := range results {
		pm, ok := productMasterMap[strings.ToUpper(strings.TrimSpace(results[i].ProductCode))]
		if !ok {
			continue
		}

		productName := strings.TrimSpace(pm.ProductName)
		results[i].ProductName = productName
		if productName != "" {
			results[i].ProductDesc = productName
		}
	}
}

func buildEmptyPagination(pagePtr *int, limitPtr *int) (page int, pageSize int, total int, totalPages int) {
	page = 1
	pageSize = 50
	total = 0
	totalPages = 0

	if pagePtr != nil && *pagePtr > 0 {
		page = *pagePtr
	}
	if limitPtr != nil && *limitPtr == 0 {
		page = 1
		pageSize = 0
		return
	}
	if limitPtr != nil && *limitPtr > 0 {
		pageSize = *limitPtr
	}

	return
}

func makeStringSetTrimUpper(xs []string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		v := strings.ToUpper(strings.TrimSpace(x))
		if v == "" {
			continue
		}
		m[v] = true
	}
	return m
}

func inSetTrimUpper(m map[string]bool, v string) bool {
	if len(m) == 0 {
		return true
	}
	k := strings.ToUpper(strings.TrimSpace(v))
	if k == "" {
		return false
	}
	return m[k]
}

func intersectStringSets(a map[string]bool, b map[string]bool) map[string]bool {
	rs := map[string]bool{}
	for k := range a {
		if b[k] {
			rs[k] = true
		}
	}
	return rs
}

func paginateResults(
	all []GetPurchaseItemRemainResponseResult,
	pagePtr *int,
	limitPtr *int,
) (paged []GetPurchaseItemRemainResponseResult, page int, limit int, total int, totalPages int) {
	page = 1
	limit = 50
	if pagePtr != nil && *pagePtr > 0 {
		page = *pagePtr
	}
	if limitPtr != nil && *limitPtr == 0 {
		limit = 0
	}
	if limitPtr != nil && *limitPtr > 0 {
		limit = *limitPtr
	}

	total = len(all)
	if total == 0 {
		return []GetPurchaseItemRemainResponseResult{}, page, limit, 0, 0
	}
	if limit == 0 {
		return all, 1, total, total, 1
	}

	totalPages = int(math.Ceil(float64(total) / float64(limit)))
	start := (page - 1) * limit
	if start >= total {
		return []GetPurchaseItemRemainResponseResult{}, page, limit, total, totalPages
	}
	end := start + limit
	if end > total {
		end = total
	}

	return all[start:end], page, limit, total, totalPages
}

func toPurchaseResponse(p models.Purchase) models.PurchaseResponse {
	delivery := ""
	if p.DeliveryDate != nil && !p.DeliveryDate.IsZero() {
		delivery = p.DeliveryDate.Format("2006-01-02")
	}

	return models.PurchaseResponse{
		ID:            p.ID.String(),
		PurchaseCode:  p.PurchaseCode,
		PurchaseType:  p.PurchaseType,
		SupplierCode:  p.SupplierCode,
		SupplierName:  p.SupplierName,
		Status:        p.Status,
		StatusApprove: p.StatusApprove,
		StatusPayment: p.StatusPayment,
		DeliveryDate:  delivery,
		Items:         []models.PurchaseItemResponse{},
	}
}

func toPurchaseItemResponse(it models.PurchaseItem, p models.Purchase) models.PurchaseItemResponse {
	return models.PurchaseItemResponse{
		ID:                   it.ID.String(),
		PurchaseItem:         it.PurchaseItem,
		DocRefItem:           it.DocRefItem,
		ProductCode:          it.ProductCode,
		ProductDesc:          it.ProductDesc,
		ProductGroupOneCode:  it.ProductGroupCode,
		ProductGroupOneName:  it.ProductGroupName,
		Qty:                  it.Qty,
		PurchaseQty:          it.PurchaseQty,
		Unit:                 it.Unit,
		PurchaseUnit:         it.PurchaseUnit,
		PurchaseUnitType:     it.PurchaseUnitType,
		PriceUnit:            it.PriceUnit,
		TotalDiscount:        it.TotalDiscount,
		TotalAmount:          it.TotalAmount,
		UnitUom:              it.UnitUom,
		TotalCost:            it.TotalCost,
		TotalDiscountPercent: it.TotalDiscountPercent,
		DiscountType:         it.DiscountType,
		TotalVat:             it.TotalVat,
		SubtotalExclVat:      it.SubtotalExclVat,
		WeightUnit:           it.WeightUnit,
		TotalWeight:          it.TotalWeight,
		Status:               it.Status,
		StatusPayment:        it.StatusPayment,
		Remark:               it.Remark,
		CreateBy:             it.CreateBy,
		CreateDtm:            it.CreateDtm.Format(time.RFC3339),
		UpdateBy:             it.UpdateBy,
		UpdateDtm:            it.UpdateDtm.Format(time.RFC3339),
	}
}

func setToSlice(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func normalizeTrimUpperSlice(xs []string) []string {
	out := make([]string, 0, len(xs))
	seen := map[string]bool{}
	for _, x := range xs {
		v := strings.ToUpper(strings.TrimSpace(x))
		if v == "" {
			continue
		}
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

type pair struct {
	PO   string
	Item string
}

func normalizeNotPurchasePairs(xs []NotPurchaseItem) []pair {
	out := make([]pair, 0, len(xs))
	seen := map[string]bool{}

	for _, x := range xs {
		po := strings.ToUpper(strings.TrimSpace(x.PurchaseCode))
		it := strings.ToUpper(strings.TrimSpace(x.PurchaseItem))
		if po == "" || it == "" {
			continue
		}
		k := po + "|" + it
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, pair{PO: po, Item: it})
	}

	return out
}

func buildValuesPairsSQL(ps []pair) (string, []interface{}) {
	if len(ps) == 0 {
		return "", nil
	}

	parts := make([]string, 0, len(ps))
	args := make([]interface{}, 0, len(ps)*2)

	for _, p := range ps {
		parts = append(parts, "(?, ?)")
		args = append(args, p.PO, p.Item)
	}

	return strings.Join(parts, ","), args
}
