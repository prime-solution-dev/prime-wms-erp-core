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
	PurchaseCodes  []string `json:"purchase_codes"`
	SupplierCodes  []string `json:"supplier_codes"`
	StatusApprove  []string `json:"status_approve"`
	StattusPayment []string `json:"stattus_payment"`
	ProductCodes   []string `json:"product_codes"` // TODO: implement filter by product code in PO Item
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

	poCodes := []string{}
	poCosesCheck := map[string]bool{}
	poItems := []string{}
	poItemsCheck := map[string]bool{}

	// fetch purchase (PENDING)
	poMap, err := getPurchase(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(poMap) == 0 {
		return &res, nil
	}

	for _, po := range poMap {
		poCode := po.PurchaseCode
		if _, ok := poCosesCheck[poCode]; !ok {
			poCosesCheck[poCode] = true
			poCodes = append(poCodes, poCode)
		}

		for _, poi := range po.Items {
			poItem := poi.PurchaseItem
			if _, ok := poItemsCheck[poItem]; !ok {
				poItemsCheck[poItem] = true
				poItems = append(poItems, poItem)
			}
		}
	}

	// fetch IB (PENDING)
	ibDocMap, err := getInbound(poCodes, poItems)
	if err != nil {
		return nil, err
	}

	// prepare
	ibCodes := []string{}
	ibCodesCheck := map[string]bool{}
	ibItems := []string{}
	ibItemsCheck := map[string]bool{}

	ibDocMapPo := map[string]documentData{}    // po|item -> ibQty
	grDocMap := map[string]documentData{}      // grCode|grItem -> grConfirmQty + ref IB
	grRemainMapPo := map[string]documentData{} // po|item -> (grConfirm - apMatchedBy4Key)
	ignoreSourceAp := map[string]bool{}        // po|item|gr|grItem -> true when AP matched
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

	// fetch AP
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

	// compute remainQty map
	remainMap := ComputePurchaseRemainQty(poMap, ibDocMapPo, apDocMapPo, grRemainMapPo)

	// build response (Qty = ตั้งต้นจาก PO, RemainQty = computed)
	results, err := ConvertToResponse(poMap, remainMap)
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

	//  sum AP by 4-key (po|item|gr|grItem)
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

		poCode := poPair[0]
		poItem := poPair[1]
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

		//  mark ignore เฉพาะกรณีเจอ AP จริง
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
) map[string]float64 {

	remainMap := map[string]float64{} // key -> poCode|poItem

	for poCode, po := range poMap {
		for _, it := range po.Items {
			poItem := strings.TrimSpace(it.PurchaseItem)
			if poItem == "" {
				continue
			}

			key := fmt.Sprintf("%s|%s", poCode, poItem)

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
) ([]GetPurchaseItemRemainResponseResult, error) {

	rs := []GetPurchaseItemRemainResponseResult{}

	for _, po := range poMap {
		for _, it := range po.Items {
			key := fmt.Sprintf("%s|%s", po.PurchaseCode, strings.TrimSpace(it.PurchaseItem))
			remain := remainMap[key] // default 0 ถ้าไม่เจอ

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
	}

	jsonBytes, err := json.Marshal(reqPo)
	if err != nil {
		return rs, err
	}
	jsonString := string(jsonBytes)

	resPoInf, err := purchaseService.GetPO(ctx, jsonString)
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

			//case if GR Confirm more than GR Item, we only use GR Item as max
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
	_ = poItems

	reqAp := invoiceService.GetInvoiceRequest{
		InvoiceItemDocRef: poCodes,
	}

	jsonBytes, err := json.Marshal(reqAp)
	if err != nil {
		return rs, err
	}
	jsonString := string(jsonBytes)

	resApInf, err := invoiceService.GetInvoice(ctx, jsonString)
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
			invCode := ap.InvoiceCode
			invItem := api.InvoiceItem
			poCode := api.DocumentRef
			poItem := api.DocumentRefItem
			grCode := api.SourceCode
			grItem := api.SourceItem
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
