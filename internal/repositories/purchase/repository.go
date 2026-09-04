package purchaseRepository

import (
	"fmt"
	"math"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Create
func CreatePurchase(gormx *gorm.DB, purchases []models.Purchase) error {
	return gormx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&purchases).Error; err != nil {
			return err
		}
		return nil
	})
}

func applyItemsProductGroupOneNameLike(query, gormx *gorm.DB, value string) *gorm.DB {
	if value == "" {
		return query
	}

	likePattern := "%" + value + "%"
	sub := gormx.Model(&models.PurchaseItem{}).
		Select("1").
		Where("purchase.id = purchase_item.purchase_id").
		Where("(product_group_code ILIKE ? OR product_group_name ILIKE ?)", likePattern, likePattern)

	return query.Where("EXISTS (?)", sub)
}

// Get
// Get
func GetPurchaseList(
	purchaseCodes []string,
	supplierCodes []string,
	statusApprove []string,
	statusPayment []string,
	statusPaymentIncomplete bool,
	status []string,
	productCodes []string,
	purchaseType []string,
	docRef []string,
	tradingRef []string,
	companyCode string,
	siteCode string,
	page int,
	pageSize int,
	purchaseCodeLike string,
	docRefLike string,
	supplierCodeLike string,
	supplierNameLike string,
	itemsProductCodeLike string,
	itemsProductDescLike string,
	itemsProductGroupOneNameLike string,
	startCreateDate *time.Time,
	endCreateDate *time.Time,
	usedStatus []string,
) ([]models.Purchase, int, int, int, int, error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	defer db.CloseGORM(gormx)

	var purchases []models.Purchase
	var totalRecords int64

	// กัน page <= 0
	if page <= 0 {
		page = 1
	}

	// Build base query
	query := gormx.Model(&models.Purchase{}).
		Where("company_code = ? AND site_code = ?", companyCode, siteCode)

	if len(purchaseCodes) > 0 {
		query = query.Where("purchase_code IN ?", purchaseCodes)
	}

	if len(supplierCodes) > 0 {
		query = query.Where("supplier_code IN ?", supplierCodes)
	}

	if len(statusApprove) > 0 {
		query = query.Where("status_approve IN ?", statusApprove)
	}

	if len(statusPayment) > 0 {
		query = query.Where("status_payment IN ?", statusPayment)
	}

	if statusPaymentIncomplete {
		query = query.Where("(status_payment != ? OR status_payment IS NULL)", "COMPLETED")
	}

	if len(status) > 0 {
		query = query.Where("status IN ?", status)
	}

	// Partial ถูกตัดสินจาก used_status ไม่ใช่ status_approve (ดู
	// convertStatusToStatusWording ฝั่ง web) จึงต้องกรองแยกออกมา
	if len(usedStatus) > 0 {
		query = query.Where("used_status IN ?", usedStatus)
	}

	if purchaseCodeLike != "" {
		query = query.Where("purchase_code ILIKE ?", "%"+purchaseCodeLike+"%")
	}

	if docRefLike != "" {
		query = query.Where("doc_ref ILIKE ?", "%"+docRefLike+"%")
	}

	if supplierCodeLike != "" {
		query = query.Where("supplier_code ILIKE ?", "%"+supplierCodeLike+"%")
	}

	if supplierNameLike != "" {
		query = query.Where("supplier_name ILIKE ?", "%"+supplierNameLike+"%")
	}

	if itemsProductCodeLike != "" {
		sub := gormx.Model(&models.PurchaseItem{}).
			Select("1").
			Where("purchase.id = purchase_item.purchase_id").
			Where("product_code ILIKE ?", "%"+itemsProductCodeLike+"%")
		query = query.Where("EXISTS (?)", sub)
	}

	if itemsProductDescLike != "" {
		sub := gormx.Model(&models.PurchaseItem{}).
			Select("1").
			Where("purchase.id = purchase_item.purchase_id").
			Where("product_desc ILIKE ?", "%"+itemsProductDescLike+"%")
		query = query.Where("EXISTS (?)", sub)
	}

	query = applyItemsProductGroupOneNameLike(query, gormx, itemsProductGroupOneNameLike)

	if startCreateDate != nil {
		query = query.Where("create_dtm >= ?", startCreateDate.Format("2006-01-02"))
	}
	if endCreateDate != nil {
		query = query.Where("create_dtm <= ?", endCreateDate.Format("2006-01-02"))
	}

	if len(productCodes) > 0 {
		sub := gormx.Model(&models.PurchaseItem{}).
			Select("1").
			Where("purchase.id = purchase_item.purchase_id").
			Where("product_code IN ?", productCodes)
		query = query.Where("EXISTS (?)", sub)
	}

	if len(purchaseType) > 0 {
		query = query.Where("purchase_type IN ?", purchaseType)
	}

	if len(docRef) > 0 {
		query = query.Where("doc_ref IN ?", docRef)
	}

	if len(tradingRef) > 0 {
		query = query.Where("trading_ref IN ?", tradingRef)
	}

	// Count total records (no preload needed)
	if err := query.Count(&totalRecords).Error; err != nil {
		return nil, 0, 0, 0, 0, err
	}

	applyPaging := pageSize > 0

	// Order clause (เหมือนเดิม)
	orderClause := `
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
    `

	q := query.Preload("PurchaseItems").Order(orderClause).Debug()

	if applyPaging {
		offset := (page - 1) * pageSize
		q = q.Limit(pageSize).Offset(offset)
	}

	if err := q.Find(&purchases).Error; err != nil {
		return nil, 0, 0, 0, 0, err
	}

	// totalPages
	totalPages := 0
	if totalRecords > 0 {
		if !applyPaging {
			totalPages = 1
			// pageSize=0 หมายถึง "all"
			pageSize = int(totalRecords)
			page = 1
		} else {
			totalPages = int(math.Ceil(float64(totalRecords) / float64(pageSize)))
		}
	}

	return purchases, int(totalRecords), page, pageSize, totalPages, nil
}

func GetPurchaseListByGRFilter(
	supplierCodes []string,
	purchaseCodes []string,
	purchaseItemCodes []string,
	statusApprove []string,
	purchaseItemStatus []string,
	productCodes []string,
	notItems []models.ExceptPurchaseAndPurchaseItemRequest,
	companyCode string,
	siteCode string,
	page int,
	pageSize int,
) ([]models.Purchase, int, int, int, int, error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	defer db.CloseGORM(gormx)

	var purchases []models.Purchase
	var totalRecords int64

	// Build base query
	baseQuery := gormx.Model(&models.Purchase{}).
		Where("company_code = ? AND site_code = ?", companyCode, siteCode)

	if len(purchaseItemStatus) > 0 {
		sub := gormx.Model(&models.PurchaseItem{}).
			Select("1").
			Where("purchase.id = purchase_item.purchase_id").
			Where("status IN ?", purchaseItemStatus)

		baseQuery = baseQuery.Where("EXISTS (?)", sub)
	}

	if len(purchaseItemCodes) > 0 {
		sub := gormx.Model(&models.PurchaseItem{}).
			Select("1").
			Where("purchase.id = purchase_item.purchase_id").
			Where("purchase_item IN ?", purchaseItemCodes)

		baseQuery = baseQuery.Where("EXISTS (?)", sub)
	}

	if len(purchaseCodes) > 0 {
		sub := gormx.Model(&models.PurchaseItem{}).
			Select("1").
			Where("purchase.id = purchase_item.purchase_id").
			Where("purchase.purchase_code IN ?", purchaseCodes)

		baseQuery = baseQuery.Where("EXISTS (?)", sub)
	}

	if len(notItems) > 0 {
		for _, notItem := range notItems {
			subQuery := gormx.Model(&models.PurchaseItem{}).
				Select("1").
				Where("purchase.id = purchase_item.purchase_id").
				Where("purchase_item.purchase_item IN ?", notItem.PurchaseItemCodes)

			baseQuery = baseQuery.Where("NOT EXISTS (?)", subQuery)
		}
	}

	if len(supplierCodes) > 0 {
		baseQuery = baseQuery.Where("supplier_code IN ?", supplierCodes)
	}

	if len(statusApprove) > 0 {
		baseQuery = baseQuery.Where("status_approve IN ?", statusApprove)
	}

	if len(productCodes) > 0 {
		sub := gormx.Model(&models.PurchaseItem{}).
			Select("1").
			Where("purchase.id = purchase_item.purchase_id").
			Where("product_code IN ?", productCodes)

		baseQuery = baseQuery.Where("EXISTS (?)", sub)
	}

	// Count total records (no preload needed)
	if err := baseQuery.Count(&totalRecords).Error; err != nil {
		return nil, 0, 0, 0, 0, err
	}

	if pageSize == 0 {
		pageSize = int(totalRecords)
	}

	// Apply pagination
	offset := (page - 1) * pageSize
	if err := baseQuery.
		Preload("PurchaseItems").
		Limit(pageSize).
		Offset(offset).
		Find(&purchases).Error; err != nil {
		return nil, 0, 0, 0, 0, err
	}

	totalPages := 0
	if totalRecords > 0 {
		totalPages = int(math.Ceil(float64(totalRecords) / float64(pageSize)))
	}

	if page == 0 {
		page = 1
	}

	return purchases, int(totalRecords), page, pageSize, totalPages, nil
}

// Update
func UpdatePurchase(purchases []models.Purchase) (err error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		for _, purchase := range purchases {
			// Update purchase
			if err := tx.Model(&models.Purchase{}).
				Where("id = ?", purchase.ID).
				Updates(purchase).Error; err != nil {
				return err
			}

			// Delete old items
			if result := tx.Where("purchase_id = ?", purchase.ID).Delete(&models.PurchaseItem{}); result.Error != nil {
				return result.Error
			}

			// Insert new items
			for _, item := range purchase.PurchaseItems {
				item.PurchaseID = purchase.ID // Ensure foreign key is set
				if err := tx.Create(&item).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func UpdatePurchaseStatusApprove(purchases []models.UpdateStatusApprovePurchaseRequest) (err error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		for _, purchase := range purchases {
			updates := map[string]interface{}{
				"status_approve": purchase.StatusApprove,
				"is_approved":    purchase.IsApproved,
				"update_dtm":     time.Now().UTC(),
			}
			// sync lifecycle status ตามตาราง: Approved(COMPLETED)→status COMPLETED,
			// Reject→status CANCELLED (ให้ filter/display สถานะตรงกัน). PROCESS/REVIEW
			// ไม่แตะ status (คง PENDING). รับของยัง gate ที่ status_approve ไม่ใช่ status
			switch purchase.StatusApprove {
			case "COMPLETED":
				updates["status"] = "COMPLETED"
			case "REJECT":
				updates["status"] = "CANCELLED"
			}
			if result := tx.Model(&models.Purchase{}).
				Where("id = ?", purchase.ID).
				Updates(updates); result.Error != nil {
				err = result.Error
			}
		}
		return nil
	})
}

func CompletePOPayment(purchaseCodes []string, purchaseItems []string) (err error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		// purchase_item is a per-purchase running number (1, 2, 3, ...), so it is
		// only unique within a purchase. Always scope the update by purchase_id.
		if len(purchaseCodes) > 0 {
			q := tx.Model(&models.PurchaseItem{}).
				Where("purchase_id IN (?)", tx.Model(&models.Purchase{}).
					Select("id").
					Where("purchase_code IN ?", purchaseCodes))

			if len(purchaseItems) > 0 {
				q = q.Where("purchase_item IN ?", purchaseItems)
			}

			if result := q.Updates(map[string]interface{}{
				"status_payment": "COMPLETED",
				"update_dtm":     time.Now().UTC(),
			}); result.Error != nil {
				return result.Error
			}
		}

		var purchaseAutoCompleteCodes []string
		subQueryCompletePurchaseItem := tx.Model(&models.PurchaseItem{}).
			Select("purchase_id").
			Where("status_payment = ?", "COMPLETED")

		if err := tx.Model(&models.Purchase{}).
			Where("id IN (?)", subQueryCompletePurchaseItem).
			Pluck("purchase_code", &purchaseAutoCompleteCodes).Error; err != nil {
			return err
		}

		purchaseCodes = append(purchaseCodes, purchaseAutoCompleteCodes...)

		for _, code := range purchaseCodes {
			var totalItems int64
			var completedItems int64

			queryPurchaseID := tx.Model(&models.Purchase{}).Select("id").Where("purchase_code = ?", code)

			if err := tx.Model(&models.PurchaseItem{}).Where("purchase_id = (?)", queryPurchaseID).Count(&totalItems).Error; err != nil {
				return err
			}

			if err := tx.Model(&models.PurchaseItem{}).Where("purchase_id = (?) AND status_payment = ?", queryPurchaseID, "COMPLETED").Count(&completedItems).Error; err != nil {
				return err
			}

			status := "PENDING"
			if totalItems == 0 {
				status = "COMPLETED"
			}

			if totalItems > 0 && totalItems == completedItems {
				status = "COMPLETED"
			}

			if err := tx.Model(&models.Purchase{}).
				Where("purchase_code = ?", code).
				Updates(map[string]interface{}{
					"status_payment": status,
					"update_dtm":     time.Now().UTC(),
				}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func CompletePO(purchaseCodes []string) (err error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Purchase{}).
			Where("purchase_code IN ?", purchaseCodes).
			Updates(map[string]interface{}{
				"status":     "COMPLETED",
				"update_dtm": time.Now().UTC(),
			}).Error; err != nil {
			return err
		}
		return nil
	})
}

func CompletePOItem(usedType string, purchaseItemUsed []models.PurchaseItemUsed) error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	poCodes := []string{}
	poCodesCheck := map[string]bool{}
	poItems := []string{}
	poItemsCheck := map[string]bool{}
	reqMap := map[string]models.PurchaseItemUsed{}

	for _, piu := range purchaseItemUsed {
		purchaseCode := piu.PurchaseCode
		purchaseItem := piu.PurchaseItemCode

		if purchaseCode != "" {
			if _, ok := poCodesCheck[purchaseCode]; !ok {
				poCodesCheck[purchaseCode] = true
				poCodes = append(poCodes, purchaseCode)
			}
		}

		if purchaseItem != "" {
			if _, ok := poItemsCheck[purchaseItem]; !ok {
				poItemsCheck[purchaseItem] = true
				poItems = append(poItems, purchaseItem)
			}
		}

		reqKey := fmt.Sprintf("%s|%s", purchaseCode, purchaseItem)
		reqMap[reqKey] = piu
	}

	if len(poCodes) == 0 || len(poItems) == 0 {
		return nil
	}

	var purchases []models.Purchase

	q := gormx.
		Model(&models.Purchase{}).
		Select("purchase.*").
		Joins("JOIN purchase_item pi ON pi.purchase_id = purchase.id").
		Where("purchase.purchase_code IN ?", poCodes).
		Where("pi.purchase_item IN ?", poItems).
		Distinct().
		Preload("PurchaseItems")

	if err := q.Find(&purchases).Error; err != nil {
		return err
	}

	updateCompPoItemsSet := map[string]struct{}{} // purchase_item_id ที่จะ set COMPLETED
	updatePoPartialSet := map[string]struct{}{}   // purchase_id ที่จะ set PARTIAL
	updatePoCompSet := map[string]struct{}{}      // purchase_id ที่จะ set COMPLETED
	updatePoPendingSet := map[string]struct{}{}   // purchase_id ที่จะ set PENDING (optional)

	for pIdx := range purchases {
		p := &purchases[pIdx]

		itemsCount := len(p.PurchaseItems)
		if itemsCount == 0 {
			continue
		}

		// count completed in po
		completedCount := 0
		for i := range p.PurchaseItems {
			if p.PurchaseItems[i].Status == "COMPLETED" {
				completedCount++
			}
		}

		for piIdx := range p.PurchaseItems {
			pi := &p.PurchaseItems[piIdx]

			purchaseCode := p.PurchaseCode
			purchaseItem := pi.PurchaseItem
			purchaseItemID := pi.ID

			poQty := pi.Qty
			poWeight := pi.TotalWeight
			poUnit := pi.PurchaseUnit

			rKey := fmt.Sprintf("%s|%s", purchaseCode, purchaseItem)
			r, ok := reqMap[rKey]
			if !ok {
				continue
			}

			// tolerance
			tolPct := float64(r.Tolerance)
			if tolPct <= 0 {
				return fmt.Errorf("PurchaseCode: %s, PurchaseItem: %s tolerance is zero", purchaseCode, purchaseItem)
			}

			reqQty := r.QTY
			reqWeight := r.Weight

			if pi.Status == "COMPLETED" {
				continue
			}

			isPass := false

			if strings.EqualFold(poUnit, "KG") {
				base := poWeight
				input := reqWeight

				if base == 0 {
					return fmt.Errorf("PurchaseCode: %s, PurchaseItem: %s base weight is zero", purchaseCode, purchaseItem)
				}

				min := base * (1 - tolPct/100.0)
				//max := base * (1 + tolPct/100.0)

				//if input >= min && input <= max {
				if input >= min {
					isPass = true
				}
			} else {
				base := poQty
				input := reqQty

				if base == 0 {
					return fmt.Errorf("PurchaseCode: %s, PurchaseItem: %s base qty is zero", purchaseCode, purchaseItem)
				}

				min := base * (1 - tolPct/100.0)
				//max := base * (1 + tolPct/100.0)

				//if input >= min && input <= max {
				if input >= min {
					isPass = true
				}
			}

			if isPass {
				pi.Status = "COMPLETED"
				updateCompPoItemsSet[purchaseItemID.String()] = struct{}{}
				completedCount++
			}
		}

		if completedCount < itemsCount {
			updatePoPartialSet[p.ID.String()] = struct{}{}
		} else if completedCount == itemsCount {
			updatePoCompSet[p.ID.String()] = struct{}{}
		}
	}

	// execute
	if len(updateCompPoItemsSet) == 0 &&
		len(updatePoPartialSet) == 0 &&
		len(updatePoCompSet) == 0 &&
		len(updatePoPendingSet) == 0 {
		return nil
	}

	now := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), time.Now().Hour(), time.Now().Minute(), time.Now().Second(), time.Now().Nanosecond(), time.UTC)

	tx := gormx.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// update purchase_item -> COMPLETED
	if len(updateCompPoItemsSet) > 0 {
		itemIDs := make([]string, 0, len(updateCompPoItemsSet))
		for id := range updateCompPoItemsSet {
			itemIDs = append(itemIDs, id)
		}

		if err := tx.Model(&models.PurchaseItem{}).
			Where("id IN ?", itemIDs).
			Updates(map[string]any{
				"status":     "COMPLETED",
				"update_dtm": now,
			}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// update purchase -> PENDING
	if len(updatePoPendingSet) > 0 {
		poIDs := make([]string, 0, len(updatePoPendingSet))
		for id := range updatePoPendingSet {
			poIDs = append(poIDs, id)
		}

		if err := tx.Model(&models.Purchase{}).
			Where("id IN ?", poIDs).
			Updates(map[string]any{
				"used_type":   usedType,
				"used_status": "PENDING",
				"update_dtm":  now,
			}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// update purchase -> PARTIAL
	if len(updatePoPartialSet) > 0 {
		poIDs := make([]string, 0, len(updatePoPartialSet))
		for id := range updatePoPartialSet {
			poIDs = append(poIDs, id)
		}

		if err := tx.Model(&models.Purchase{}).
			Where("id IN ?", poIDs).
			Updates(map[string]any{
				"used_type":   usedType,
				"used_status": "PARTIAL",
				"update_dtm":  now,
			}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// update purchase -> COMPLETED
	if len(updatePoCompSet) > 0 {
		poIDs := make([]string, 0, len(updatePoCompSet))
		for id := range updatePoCompSet {
			poIDs = append(poIDs, id)
		}

		if err := tx.Model(&models.Purchase{}).
			Where("id IN ?", poIDs).
			Updates(map[string]any{
				"status":      "COMPLETED",
				"used_type":   usedType,
				"used_status": "COMPLETED",
				"update_dtm":  now,
			}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// func CompletePOItem(usedType string, purchaseItemUsed []models.PurchaseItemUsed) error {
//  gormx, err := db.ConnectGORM("prime_erp")
//  if err != nil {
//      return err
//  }
//  defer db.CloseGORM(gormx)

//  purchaseItemCodes := []string{}
//  for _, usedPurchase := range purchaseItemUsed {
//      purchaseItemCodes = append(purchaseItemCodes, usedPurchase.PurchaseItemCode)
//  }

//  return gormx.Transaction(func(tx *gorm.DB) error {
//      for _, poItemUsed := range purchaseItemUsed {
//          if err := tx.Model(&models.PurchaseItem{}).
//              Where("purchase_item = ? AND purchase_qty = ?", poItemUsed.PurchaseItemCode, poItemUsed.QTY).
//              Updates(map[string]interface{}{
//                  "status":     "COMPLETED",
//                  "update_dtm": time.Now().UTC(),
//              }).Error; err != nil {
//              return err
//          }

//          subQuery := tx.Model(&models.PurchaseItem{}).
//              Select("purchase_id").
//              Where("purchase_item = ?", poItemUsed.PurchaseItemCode)

//          if err := tx.Model(&models.Purchase{}).
//              Where("id IN (?)", subQuery).
//              Updates(map[string]interface{}{
//                  "used_type":   usedType,
//                  "used_status": "PARTIAL",
//                  "update_dtm":  time.Now().UTC(),
//              }).Error; err != nil {
//              return err
//          }

//          var completedCount int64
//          var itemsCount int64

//          if err := tx.Model(&models.PurchaseItem{}).
//              Where("purchase_id = (?) AND status = ?", subQuery, "COMPLETED").
//              Count(&completedCount).Error; err != nil {
//              return err
//          }

//          if err := tx.Model(&models.PurchaseItem{}).
//              Where("purchase_id = (?)", subQuery).
//              Count(&itemsCount).Error; err != nil {
//              return err
//          }

//          if completedCount == itemsCount {
//              if err := tx.Model(&models.Purchase{}).
//                  Where("id = (?)", subQuery).
//                  Updates(map[string]interface{}{
//                      "status":      "COMPLETED",
//                      "used_type":   usedType,
//                      "used_status": "COMPLETED",
//                      "update_dtm":  time.Now().UTC(),
//                  }).Error; err != nil {
//                  return err
//              }
//          }
//      }

//      return nil
//  })
// }
