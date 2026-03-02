package xService

import (
	"encoding/json"
	"fmt"
	goodsReceiveService "prime-erp-core/external/goods-receive-service"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	invoiceService "prime-erp-core/internal/services/invoice-service"
	purchaseService "prime-erp-core/internal/services/purchase-service"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GetPurchaseItemRemainRequest struct {
	CompanyCode    string   `json:"company_code"`
	SiteCode       string   `json:"site_code"`
	PurchaseCodes  []string `json:"purchase_codes"`
	SupplierCodes  []string `json:"supplier_codes"`
	StatusApprove  []string `json:"status_approve"`
	StattusPayment []string `json:"stattus_payment"`
	ProductCodes   []string `json:"product_codes"` // optional
	Page           *int     `json:"page"`
	PageSize       *int     `json:"limit"`
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
	res := GetPurchaseItemRemainResponse{}

	// validatetion
	if strings.TrimSpace(req.CompanyCode) == "" {
		return nil, fmt.Errorf("company_code is required")
	}
	if strings.TrimSpace(req.SiteCode) == "" {
		return nil, fmt.Errorf("site_code is required")
	}

	// Optional filters (empty = no filter)
	productSet := makeStringSetTrimUpper(req.ProductCodes)

	poCodes := []string{}
	poCodesCheck := map[string]bool{}
	poItems := []string{}
	poItemsCheck := map[string]bool{}

	// fetch purchase (PENDING) - allow upstream filter if supported (cheap)
	poMap, err := getPurchase(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(poMap) == 0 {
		return &res, nil
	}

	// Build PO codes, PO items (filtered by product if provided)
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
			// If product filter present, only keep those PO items
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

	// If product filter was provided but no matching PO items -> early return empty
	if len(productSet) > 0 && len(poItems) == 0 {
		res.Daatas = []GetPurchaseItemRemainResponseResult{}
		return &res, nil
	}

	// fetch IB (PENDING) by filtered poCodes + poItems
	ibDocMap, err := getInbound(poCodes, poItems)
	if err != nil {
		return nil, err
	}

	// prepare maps
	ibCodes := []string{}
	ibCodesCheck := map[string]bool{}
	ibItems := []string{}
	ibItemsCheck := map[string]bool{}

	ibDocMapPo := map[string]documentData{}    // po|item -> ibQty
	grDocMap := map[string]documentData{}      // grCode|grItem -> grUseQty + ref IB
	grRemainMapPo := map[string]documentData{} // po|item -> (grUseQty - apMatchedBy4Key) clamp>=0
	ignoreSourceAp := map[string]bool{}        // po|item|gr|grItem -> true when AP matched by 4-key
	ibToPO := map[string][2]string{}           // ibCode|ibItem -> [poCode, poItem]

	if len(ibDocMap) > 0 {
		// build IB lists + build IB->PO map
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

		// sum IB by PO|POItem
		ibDocMapPo, err = ComputeSumInbound(ibDocMap)
		if err != nil {
			return nil, err
		}

		// fetch GR (COMPLETED)
		grDocMap, err = getGoodsReceive(ibCodes, ibItems)
		if err != nil {
			return nil, err
		}
	}

	// fetch AP (PENDING/COMPLETED) by poCodes (optionally filter by poItems locally)
	apDocMap, err := getInvoiceAp(ctx, poCodes, poItems)
	if err != nil {
		return nil, err
	}

	// compute GRDiff per PO|POItem and ignoreSourceAp
	if len(grDocMap) > 0 {
		grRemainMapPo, ignoreSourceAp, err = ComputeReceiveRemain(grDocMap, apDocMap, ibToPO)
		if err != nil {
			return nil, err
		}
	}

	// sum AP by PO|POItem excluding ignored 4-key
	apDocMapPo, err := ComputeApRemain(apDocMap, ignoreSourceAp)
	if err != nil {
		return nil, err
	}

	// compute remainQty map (skip non-matching product items when filter exists)
	remainMap := ComputePurchaseRemainQty(poMap, ibDocMapPo, apDocMapPo, grRemainMapPo, productSet)

	// build response (filter product again for safety)
	results, err := ConvertToResponse(poMap, remainMap, productSet)
	if err != nil {
		return nil, err
	}
	res.Daatas = results

	return &res, nil
}

func ComputeSumInbound(ibDocMap map[string]documentData) (map[string]documentData, error) {
	ibDocMapPo := map[string]documentData{} // key -> poCode|poItem

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
	ibToPO map[string][2]string, // ibCode|ibItem -> poCode|poItem
) (map[string]documentData, map[string]bool, error) {

	rs := map[string]documentData{}     // key -> poCode|poItem (sum diff)
	ignoreSourceAp := map[string]bool{} // key -> po|item|gr|grItem

	// sum AP by 4-key (po|item|gr|grItem)
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

	// walk GR, resolve po|item via IB->PO
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
			diff = 0 // clamp: GR completed usually already goes to AP by confirm qty
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

		// mark ignore only when AP matched by 4-key and has qty
		if hasAP && apQty > 0 {
			ignoreSourceAp[k4] = true
		}
	}

	return rs, ignoreSourceAp, nil
}

func ComputeApRemain(apDocMap map[string]documentData, ignoreSourceAp map[string]bool) (map[string]documentData, error) {
	apDocMapPo := map[string]documentData{} // key -> poCode|poItem

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
	productSet map[string]bool, // optional
) map[string]float64 {

	remainMap := map[string]float64{} // key -> poCode|poItem

	for poCode, po := range poMap {
		for _, it := range po.Items {

			// If product filter provided, compute only for those items
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
	productSet map[string]bool, // optional
) ([]GetPurchaseItemRemainResponseResult, error) {

	rs := []GetPurchaseItemRemainResponseResult{}

	for _, po := range poMap {
		for _, it := range po.Items {

			// If product filter provided, only return those items
			if len(productSet) > 0 && !inSetTrimUpper(productSet, it.ProductCode) {
				continue
			}

			key := fmt.Sprintf("%s|%s", strings.TrimSpace(po.PurchaseCode), strings.TrimSpace(it.PurchaseItem))
			remain := remainMap[key] // default 0 if not found

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
				ProductDesc:          it.ProductDesc,
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

func getPurchase(ctx *gin.Context, req GetPurchaseItemRemainRequest) (map[string]models.PurchaseResponse, error) {
	rs := map[string]models.PurchaseResponse{} // map -> purchaseCode

	reqPo := models.GetPurchaseRequest{
		PurchaseCodes: req.PurchaseCodes,
		SupplierCodes: req.SupplierCodes,
		Status:        []string{`PENDING`},
		StatusApprove: req.StatusApprove,
		StatusPayment: req.StattusPayment,
		CompanyCode:   strings.TrimSpace(req.CompanyCode),
		SiteCode:      strings.TrimSpace(req.SiteCode),
		ProductCodes:  req.ProductCodes,
	}

	jsonBytes, err := json.Marshal(reqPo)
	if err != nil {
		return rs, err
	}

	resPoInf, err := purchaseService.GetPO(ctx, string(jsonBytes))
	if err != nil {
		return rs, err
	}

	resPo, ok := resPoInf.(models.GetPurchaseResponse)
	if !ok {
		if p, ok2 := resPoInf.(*models.GetPurchaseResponse); ok2 && p != nil {
			resPo = *p
		} else {
			return rs, fmt.Errorf("GetPO returned %T, expected models.GetPurchaseResponse", resPoInf)
		}
	}

	for _, po := range resPo.DataList {
		rs[po.PurchaseCode] = po
	}

	return rs, nil
}

func getInbound(poCodes []string, poItems []string) (map[string]documentData, error) {
	rs := map[string]documentData{} // inboundCode|inboundItem -> doc

	reqIb := goodsReceiveService.InboundFilter{
		InboundItemDocumentRef:     poCodes,
		InboundItemDocumentRefItem: poItems,
		Status:                     []string{`PENDING`},
	}

	resIb, err := goodsReceiveService.GetInbounds(reqIb)
	if err != nil {
		return rs, err
	}

	for _, ib := range resIb.InboundRes {
		for _, ibi := range ib.InboundItemRes {
			ibCode := ib.InboundCode
			ibItem := ibi.InboundItem

			poCode := ibi.DocumentRef
			poItem := ibi.DocumentRefItem

			qty := ibi.Qty
			unitCode := ibi.UnitCode

			key := fmt.Sprintf("%s|%s", ibCode, ibItem)
			doc, ok := rs[key]
			if !ok {
				doc = documentData{
					DocumentCode:    ibCode,
					DocumentItem:    ibItem,
					DocumentRef:     poCode,
					DocumentRefItem: poItem,
					Qty:             0,
					UnitCode:        unitCode,
				}
			}
			doc.Qty += qty
			rs[key] = doc
		}
	}

	return rs, nil
}

func getGoodsReceive(ibCodes []string, ibItems []string) (map[string]documentData, error) {
	rs := map[string]documentData{} // grCode|grItem -> doc (ref = IB)

	reqGr := goodsReceiveService.GoodsReceiveFilter{
		ReferenceNo:     ibCodes,
		DocumentRefItem: ibItems,
		Status:          []string{`COMPLETED`},
	}

	resGr, err := goodsReceiveService.GetGoodsReceives(reqGr)
	if err != nil {
		return rs, err
	}

	for _, gr := range resGr.GoodsReceive {
		for _, gri := range gr.GoodsReceiveItem {
			grCode := gr.ReceiveCode
			grItem := gri.ReceiveItem
			ibCode := gr.DocumentRef
			ibItem := gri.DocumentRefItem
			unitCode := gri.UnitCode

			key := fmt.Sprintf("%s|%s", grCode, grItem)
			doc, ok := rs[key]
			if !ok {
				doc = documentData{
					DocumentCode:    grCode,
					DocumentItem:    grItem,
					DocumentRef:     ibCode,
					DocumentRefItem: ibItem,
					Qty:             0,
					UnitCode:        unitCode,
				}
			}

			// Use min(GR Qty, GR Confirm Qty) to avoid confirm > base qty
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

func getInvoiceAp(ctx *gin.Context, poCodes []string, poItems []string) (map[string]documentData, error) {
	rs := map[string]documentData{} // invoiceCode|invoiceItem -> doc

	poItemSet := makeStringSetTrimUpper(poItems)
	poCodeSet := makeStringSetTrimUpper(poCodes)

	reqAp := invoiceService.GetInvoiceRequest{
		InvoiceItemDocRef: poCodes,
		Status:            []string{`PENDING`, `COMPLETED`},
	}

	jsonBytes, err := json.Marshal(reqAp)
	if err != nil {
		return rs, err
	}

	resApInf, err := invoiceService.GetInvoice(ctx, string(jsonBytes))
	if err != nil {
		return rs, err
	}

	resAp, ok := resApInf.(invoiceService.ResultInvoice)
	if !ok {
		if p, ok2 := resApInf.(*invoiceService.ResultInvoice); ok2 && p != nil {
			resAp = *p
		} else {
			return rs, fmt.Errorf("GetInvoice returned %T, expected invoiceService.ResultInvoice", resApInf)
		}
	}

	for _, ap := range resAp.Invoice {
		for _, api := range ap.InvoiceItem {
			invCode := strings.TrimSpace(ap.InvoiceCode)
			invItem := strings.TrimSpace(api.InvoiceItem)
			if invCode == "" || invItem == "" {
				continue
			}

			poCode := strings.TrimSpace(api.DocumentRef)
			poItem := strings.TrimSpace(api.DocumentRefItem)

			// Safety: ensure only requested POs (even if invoice service returns extra)
			if len(poCodeSet) > 0 && !inSetTrimUpper(poCodeSet, poCode) {
				continue
			}
			// Optional: if caller filtered PO items, keep only those items
			if len(poItemSet) > 0 && !inSetTrimUpper(poItemSet, poItem) {
				continue
			}

			grCode := strings.TrimSpace(api.SourceCode)
			grItem := strings.TrimSpace(api.SourceItem)

			qty := api.Qty
			unitCode := api.UnitCode

			key := fmt.Sprintf("%s|%s", invCode, invItem)
			doc, ok := rs[key]
			if !ok {
				doc = documentData{
					DocumentCode:       invCode,
					DocumentItem:       invItem,
					DocumentRef:        poCode,
					DocumentRefItem:    poItem,
					DocumentSource:     grCode,
					DocumentSourceItem: grItem,
					Qty:                0,
					UnitCode:           unitCode,
				}
			}

			doc.Qty += qty
			rs[key] = doc
		}
	}

	return rs, nil
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
		return true // no filter
	}
	k := strings.ToUpper(strings.TrimSpace(v))
	if k == "" {
		return false
	}
	return m[k]
}
