package prePurchaseRepository

import (
	"math"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"strings"
	"time"

	"gorm.io/gorm"
)

// bigLotRemainingEpsilon = เผื่อ floating point ตอนเทียบ remaining <= 0
const bigLotRemainingEpsilon = 0.0000001

// pieceUnitCodes = หน่วยที่นับเป็น "ชิ้น" — โควตาบรรทัดนี้หักด้วย qty (ไม่ใช่ total_weight)
// ให้ตรงกับ isPieceCode ฝั่ง web (purchase-calc.ts): unit master เก็บ PCS, ของเก่ายังมี PC
var pieceUnitCodes = map[string]bool{"PCS": true, "PC": true}

func isPieceUnit(code string) bool {
	return pieceUnitCodes[strings.ToUpper(strings.TrimSpace(code))]
}

func sameUnitCode(a, b string) bool {
	return a != "" && b != "" && strings.EqualFold(a, b)
}

// bigLotLineState = บรรทัดโควตาของ Big lot 1 บรรทัด พร้อม remaining ที่หักไปเรื่อยๆ
type bigLotLineState struct {
	item      models.PrePurchaseItem
	remaining float64
}

// fullyConsumedPrePurchaseCodes คืน pre_purchase_code ของ Big lot ที่โควตารวมทุกบรรทัด
// ถูกใช้จนหมด (remaining รวม <= epsilon) เพื่อคัดออกจาก picker
func fullyConsumedPrePurchaseCodes(gormx *gorm.DB, candidates []models.PrePurchase) ([]string, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	codes := make([]string, 0, len(candidates))
	for _, c := range candidates {
		codes = append(codes, c.PrePurchaseCode)
	}

	// ดึง PO ที่ ref Big lot เหล่านี้ (purchase_type=PRE, doc_ref=pre_purchase_code)
	// นับสถานะ COMPLETED/PENDING/TEMP ให้ตรงกับ getPrePurchaseRemaining ฝั่ง web —
	// TEMP (Draft) จองโควตาไว้แล้ว ถ้าไม่นับ โควตาคงเหลือจะสูงเกินจริง
	var consumers []models.Purchase
	if err := gormx.
		Where("purchase_type = ?", "PRE").
		Where("doc_ref IN ?", codes).
		Where("status IN ?", []string{"COMPLETED", "PENDING", "TEMP"}).
		Preload("PurchaseItems").
		Find(&consumers).Error; err != nil {
		return nil, err
	}

	consumersByBigLot := map[string][]models.Purchase{}
	for _, po := range consumers {
		if po.DocRef == nil {
			continue
		}
		consumersByBigLot[*po.DocRef] = append(consumersByBigLot[*po.DocRef], po)
	}

	excluded := []string{}
	for _, bigLot := range candidates {
		if bigLotTotalRemaining(bigLot, consumersByBigLot[bigLot.PrePurchaseCode]) <= bigLotRemainingEpsilon {
			excluded = append(excluded, bigLot.PrePurchaseCode)
		}
	}

	return excluded, nil
}

// bigLotTotalRemaining = ผลรวม remaining ของทุกบรรทัดใน Big lot ใบเดียว
// จำลอง logic ฝั่ง web (getPrePurchaseRemaining + findBigLotLine + consumedBigLotQty)
// แบบบรรทัดต่อบรรทัด: 1 กลุ่มสินค้ามีได้หลายบรรทัด (คนละหน่วย) แต่ละบรรทัดถือโควตาของตัวเอง
// ห้ามยุบรวมเป็นโควตาเดียว ไม่งั้นจะถูกหักซ้ำจนติดลบ
func bigLotTotalRemaining(bigLot models.PrePurchase, consumers []models.Purchase) float64 {
	lines := make([]*bigLotLineState, 0, len(bigLot.PrePurchaseItems))
	linesByGroup := map[string][]*bigLotLineState{}
	for _, item := range bigLot.PrePurchaseItems {
		ls := &bigLotLineState{item: item, remaining: item.PurchaseQty}
		lines = append(lines, ls)
		linesByGroup[item.HierarchyCode] = append(linesByGroup[item.HierarchyCode], ls)
	}

	for _, po := range consumers {
		for _, poItem := range po.PurchaseItems {
			matched := matchBigLotLine(linesByGroup[poItem.ProductGroupCode], poItem)
			if matched == nil {
				continue
			}
			matched.remaining -= consumedBigLotQty(matched.item.PurchaseUnit, poItem)
		}
	}

	var total float64
	for _, ls := range lines {
		total += ls.remaining
	}
	return total
}

// matchBigLotLine = หาบรรทัด Big lot ที่ PO แถวนี้หักโควตาออกไป (มิเรอร์ findBigLotLine ฝั่ง web)
// ลำดับ: doc_ref_item (pre_item) ก่อน → หน่วยสั่งซื้อ → บรรทัดแรก (คงพฤติกรรมเดิม)
func matchBigLotLine(lines []*bigLotLineState, poItem models.PurchaseItem) *bigLotLineState {
	if len(lines) < 1 {
		return nil
	}
	if len(lines) == 1 {
		return lines[0]
	}

	var byDocRef []*bigLotLineState
	for _, l := range lines {
		if l.item.PreItem != "" && l.item.PreItem == poItem.DocRefItem {
			byDocRef = append(byDocRef, l)
		}
	}
	if len(byDocRef) == 1 {
		return byDocRef[0]
	}

	var byUnit []*bigLotLineState
	for _, l := range lines {
		if sameUnitCode(l.item.PurchaseUnit, poItem.PurchaseUnit) {
			byUnit = append(byUnit, l)
		}
	}
	if len(byUnit) == 1 {
		return byUnit[0]
	}

	return lines[0]
}

// consumedBigLotQty = ปริมาณที่ PO แถวนี้กินโควตา Big lot ไป วัดตามหน่วยของบรรทัด Big lot
// บรรทัดหน่วยชิ้น → หัก qty (ชิ้น), หน่วยน้ำหนัก → หัก total_weight (kg) (มิเรอร์ consumedBigLotQty ฝั่ง web)
func consumedBigLotQty(bigLotUnit string, poItem models.PurchaseItem) float64 {
	if isPieceUnit(bigLotUnit) {
		return poItem.Qty
	}
	return poItem.TotalWeight
}

// Create
func CreatePOBigLot(prePurchases []models.PrePurchase) error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&prePurchases).Error; err != nil {
			return err
		}
		return nil
	})
}

// Get
func GetPOBigLotList(req models.GetPOBigLotListRequest) ([]models.PrePurchase, int, int, int, int, error) {
	prePurchaseCodes := req.PrePurchaseCodes
	supplierCodes := req.SupplierCodes
	productGroupCodes := req.ProductGroupCodes
	statusApprove := req.StatusApprove
	status := req.Status
	companyCode := req.CompanyCode
	siteCode := req.SiteCode
	page := req.Page
	pageSize := req.PageSize

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	defer db.CloseGORM(gormx)

	var prePurchaseList []models.PrePurchase
	var totalRecords int64

	// Build base query
	query := gormx.Model(&models.PrePurchase{}).
		Where("company_code = ? AND site_code = ?", companyCode, siteCode)

	if len(prePurchaseCodes) > 0 {
		query = query.Where("pre_purchase_code IN ?", prePurchaseCodes)
	}

	if len(supplierCodes) > 0 {
		query = query.Where("supplier_code IN ?", supplierCodes)
	}

	if len(statusApprove) > 0 {
		query = query.Where("status_approve IN ?", statusApprove)
	}

	if len(status) > 0 {
		query = query.Where("status IN ?", status)
	}

	if len(productGroupCodes) > 0 {
		sub := gormx.Model(&models.PrePurchaseItem{}).
			Select("1").
			Where("pre_purchase.id = pre_purchase_item.pre_purchase_id").
			Where("hierarchy_code IN ?", productGroupCodes)

		query = query.Where("EXISTS (?)", sub)
	}

	// partial search จากช่องค้นหาบนหัวตาราง (modal Big lot PO selection)
	if req.PrePurchaseCodeLike != "" {
		query = query.Where("pre_purchase_code ILIKE ?", "%"+req.PrePurchaseCodeLike+"%")
	}

	if req.SupplierCodeLike != "" {
		query = query.Where("supplier_code ILIKE ?", "%"+req.SupplierCodeLike+"%")
	}

	if req.SupplierNameLike != "" {
		query = query.Where("supplier_name ILIKE ?", "%"+req.SupplierNameLike+"%")
	}

	if req.StartCreateDate != nil {
		query = query.Where("create_dtm >= ?", *req.StartCreateDate)
	}

	if req.EndCreateDate != nil {
		query = query.Where("create_dtm <= ?", *req.EndCreateDate)
	}

	if req.ProductGroupCodeLike != "" {
		sub := gormx.Model(&models.PrePurchaseItem{}).
			Select("1").
			Where("pre_purchase.id = pre_purchase_item.pre_purchase_id").
			Where("hierarchy_code ILIKE ?", "%"+req.ProductGroupCodeLike+"%")

		query = query.Where("EXISTS (?)", sub)
	}

	// ชื่อกลุ่มสินค้าอยู่ในคอลัมน์ hierarchy_type ไม่ใช่คอลัมน์ชื่อของตัวเอง —
	// หน้าจอ Big lot ส่ง itemName ลง product_group_type ตอนสร้าง (ดู
	// PrePurchaseItemTable.vue handleSelectProductGroup) และคอลัมน์ Product group
	// ในตารางก็แสดงค่านี้ ผู้ใช้จึงค้นด้วยชื่อที่เห็น ไม่ใช่ code
	if req.ProductGroupNameLike != "" {
		sub := gormx.Model(&models.PrePurchaseItem{}).
			Select("1").
			Where("pre_purchase.id = pre_purchase_item.pre_purchase_id").
			Where("hierarchy_type ILIKE ?", "%"+req.ProductGroupNameLike+"%")

		query = query.Where("EXISTS (?)", sub)
	}

	// คัด Big lot ที่โควตารวมถูกใช้จนหมด (remaining รวม <= 0) ออกก่อนนับ/แบ่งหน้า
	// เพื่อให้ total/pagination ตรง — คำนวณ remaining ที่ service layer (ไม่ใช่ SQL ล้วน)
	// เพราะการจับคู่ PO กับบรรทัด Big lot ต้องข้ามหน่วย Pcs/kg และ 1 กลุ่มมีได้หลายบรรทัด
	if req.OnlyRemaining {
		var candidates []models.PrePurchase
		if err := query.Session(&gorm.Session{}).
			Preload("PrePurchaseItems").
			Find(&candidates).Error; err != nil {
			return nil, 0, 0, 0, 0, err
		}

		excludeCodes, err := fullyConsumedPrePurchaseCodes(gormx, candidates)
		if err != nil {
			return nil, 0, 0, 0, 0, err
		}
		if len(excludeCodes) > 0 {
			query = query.Where("pre_purchase_code NOT IN ?", excludeCodes)
		}
	}

	// Count total records (no preload needed)
	if err := query.Count(&totalRecords).Error; err != nil {
		return nil, 0, 0, 0, 0, err
	}

	if pageSize == 0 {
		pageSize = int(totalRecords)
	}

	// Pagination
	offset := (page - 1) * pageSize
	if err := query.
		Preload("PrePurchaseItems").
		Order(`
        CASE
            WHEN status = 'PENDING' AND status_approve = 'PENDING' THEN 1
            WHEN status = 'PENDING' AND status_approve = 'PROCESS' THEN 2
            WHEN status = 'PENDING' AND status_approve = 'COMPLETED' THEN 3
						WHEN status = 'PENDING' AND status_approve = 'REVIEW' THEN 4
						WHEN status = 'PENDING' AND status_approve = 'REJECT' THEN 5
						WHEN status = 'CANCELLED' THEN 6
						WHEN status = 'COMPLETED' THEN 7
						WHEN status = 'TEMP' THEN 8
						ELSE 9
        END ASC,
				create_dtm DESC
    `).
		Limit(pageSize).
		Offset(offset).
		Find(&prePurchaseList).Error; err != nil {
		return nil, 0, 0, 0, 0, err
	}

	totalPages := int(math.Ceil(float64(totalRecords) / float64(pageSize)))

	if page == 0 {
		page = 1
	}

	return prePurchaseList, int(totalRecords), page, pageSize, totalPages, nil
}

// Update
func UpdatePOBigLot(gormx *gorm.DB, prePurchases []models.PrePurchase) (err error) {
	tx := gormx.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit().Error
		}
	}()

	for _, prePurchase := range prePurchases {
		// update pre_purchase
		if result := tx.Model(&models.PrePurchase{}).
			Where("id = ?", prePurchase.ID).
			Updates(prePurchase); result.Error != nil {
			err = result.Error
			return
		}

		// delete old items
		if result := tx.Where("pre_purchase_id = ?", prePurchase.ID).
			Delete(&models.PrePurchaseItem{}); result.Error != nil {
			err = result.Error
			return
		}

		for i := range prePurchase.PrePurchaseItems {
			prePurchase.PrePurchaseItems[i].PrePurchaseID = prePurchase.ID
			prePurchase.PrePurchaseItems[i].Status = prePurchase.Status
		}

		// insert new items
		if len(prePurchase.PrePurchaseItems) > 0 {
			if result := tx.Create(&prePurchase.PrePurchaseItems); result.Error != nil {
				err = result.Error
				return
			}
		}
	}

	return
}

func UpdateStatusApprovePOBigLot(prePurchases []models.UpdateStatusApprovePOBigLotRequest) (err error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	tx := gormx.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit().Error
		}
	}()

	for _, prePurchase := range prePurchases {
		updates := map[string]interface{}{
			"status_approve": prePurchase.StatusApprove,
			"is_approved":    prePurchase.IsApproved,
			"update_dtm":     time.Now().UTC(),
		}
		// Approved: คง status=PENDING (approved = PENDING + status_approve=COMPLETED)
		// ให้ Plan GR/รับของยังเห็น PO (เหมือน Normal). Reject→CANCELLED. status เป็น
		// COMPLETED ต่อเมื่อรับของครบ (used_status)
		switch prePurchase.StatusApprove {
		case "REJECT":
			updates["status"] = "CANCELLED"
		}
		// update pre_purchase
		if result := tx.Model(&models.PrePurchase{}).
			Where("id = ?", prePurchase.ID).Updates(updates); result.Error != nil {
			err = result.Error
			return
		}
	}

	return
}
