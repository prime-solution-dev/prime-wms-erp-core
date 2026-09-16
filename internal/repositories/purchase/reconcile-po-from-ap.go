package purchaseRepository

import (
	"log"
	"math"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// round4 rounds to 4 decimals to kill float drift before comparing sums/bounds.
// (Weights are 2dp in this system; 4dp is safely beyond business precision.)
func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}

// apSumRow is the cumulative COMPLETED-AP received amount per PO line.
type apSumRow struct {
	PurchaseCode string  `gorm:"column:purchase_code"`
	PurchaseItem string  `gorm:"column:purchase_item"`
	Qty          float64 `gorm:"column:qty"`
	Weight       float64 `gorm:"column:weight"`
}

func normKey(purchaseCode, purchaseItem string) string {
	return strings.ToUpper(strings.TrimSpace(purchaseCode)) + "|" + strings.ToUpper(strings.TrimSpace(purchaseItem))
}

// evaluatePOLineComplete is the pure completion decision for one PO line:
// received (cumulative COMPLETED-AP amount) must reach the ordered base minus the
// tolerance slack. active=false (or tol 0) means 0% == exact match (min = base).
// A non-positive base cannot be judged complete. Rounded to 4dp to avoid float drift.
func evaluatePOLineComplete(base, received, tol float64, active bool) bool {
	if base <= 0 {
		return false
	}
	effTol := 0.0
	if active {
		effTol = tol
	}
	if effTol < 0 {
		effTol = 0
	}
	if effTol > 100 {
		effTol = 100
	}
	min := base * (1 - effTol/100.0)
	return round4(received) >= round4(min)
}

// ReconcilePOFromAP recomputes each PO's completion state from persisted state
// (state-reconcile, not per-invoice delta) and closes lines / headers accordingly.
//
// Rule per PO line (boss spec):
//   - basis = the line's unit_uom: "KG" -> weight, otherwise -> qty (pieces).
//   - tolerance % from the PRODUCT MASTER for that basis:
//     KG  -> gr_weight_tolerance (+ active), else -> gr_tolerance (+ active).
//     If the active flag is false (or value 0) the tolerance is 0% == exact.
//   - base       = PO ordered qty (or total weight for KG).
//   - min        = base * (1 - tol/100).            (tol=0 => min=base, exact)
//   - received   = SUM over ALL already-COMPLETED AP invoice lines for that
//     (PO code, PO item): qty for PC, weight for KG. Includes the GRA just saved.
//   - received >= min (rounded) -> mark purchase_item.status = COMPLETED.
//
// After every line of a PO is processed: all items COMPLETED -> close the header
// (status + used_status = COMPLETED); some but not all -> used_status = PARTIAL.
//
// Monotonic: a line already COMPLETED is never re-opened, and a header already
// COMPLETED is never touched (protects downstream syncs). Safe to run repeatedly
// (idempotent) and concurrently (header is locked FOR UPDATE).
func ReconcilePOFromAP(purchaseCodes []string, productMap map[string]models.GetProductsDetailComponent) error {
	// de-dupe PO codes
	seen := map[string]struct{}{}
	codes := make([]string, 0, len(purchaseCodes))
	for _, c := range purchaseCodes {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		codes = append(codes, c)
	}
	if len(codes) == 0 {
		return nil
	}
	// Lock in a deterministic order so concurrent reconciles over overlapping PO
	// sets cannot deadlock.
	sort.Strings(codes)

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		// Lock + load the PO headers (and their items) in ONE query — no per-PO query
		// loop. codes is sorted so the IN list is deterministic; FOR UPDATE serializes
		// concurrent GRA roll-ups for the same PO. A GRA normally references a single
		// PO, and reconcile is idempotent/self-healing, so the residual multi-PO
		// deadlock window is negligible and not worth an N-query lock loop.
		var purchases []models.Purchase
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("PurchaseItems").
			Where("purchase_code IN ?", codes).
			Order("purchase_code").
			Find(&purchases).Error; err != nil {
			return err
		}
		if len(purchases) == 0 {
			return nil
		}

		// Cumulative COMPLETED-AP received amount per (PO code, PO item).
		var sums []apSumRow
		if err := tx.
			Table("invoice_item ii").
			Select("ii.document_ref AS purchase_code, ii.document_ref_item AS purchase_item, "+
				"COALESCE(SUM(ii.qty),0) AS qty, COALESCE(SUM(ii.weight),0) AS weight").
			Joins("JOIN invoice i ON i.id = ii.invoice_id").
			Where("i.invoice_type IN ?", []string{"AP", "AP-FAB"}).
			Where("i.status = ?", "COMPLETED").
			Where("ii.document_ref IN ?", codes).
			Group("ii.document_ref, ii.document_ref_item").
			Scan(&sums).Error; err != nil {
			return err
		}
		// Accumulate (not overwrite): the SQL GROUP BY keys on the raw
		// document_ref/document_ref_item, while normKey upper-cases + trims. Two group
		// rows that differ only in case/whitespace must be summed, not clobbered, or
		// received would be under-counted and a line that should close would not.
		sumMap := make(map[string]apSumRow, len(sums))
		for _, s := range sums {
			k := normKey(s.PurchaseCode, s.PurchaseItem)
			acc := sumMap[k]
			acc.Qty += s.Qty
			acc.Weight += s.Weight
			sumMap[k] = acc
		}

		now := time.Now().UTC()

		for _, p := range purchases {
			// Header already closed -> monotonic, leave it alone.
			if strings.EqualFold(p.Status, "COMPLETED") {
				continue
			}

			itemsCount := len(p.PurchaseItems)
			if itemsCount == 0 {
				continue
			}

			completedIDs := make([]interface{}, 0, itemsCount)
			completedCount := 0

			for _, pi := range p.PurchaseItems {
				if strings.EqualFold(pi.Status, "COMPLETED") {
					completedCount++
					continue
				}

				prod := productMap[pi.ProductCode]
				s := sumMap[normKey(p.PurchaseCode, pi.PurchaseItem)]

				var base, received, tol float64
				var active bool
				if strings.EqualFold(strings.TrimSpace(pi.UnitUom), "KG") {
					base = pi.TotalWeight
					received = s.Weight
					tol = prod.GRWeightTolerance
					active = prod.GRWeightToleranceActive
				} else {
					base = pi.Qty
					received = s.Qty
					tol = prod.GRTolerance
					active = prod.GRToleranceActive
				}

				// Surface the "can't judge" case instead of silently hanging the PO in
				// PARTIAL (e.g. a KG line whose total_weight is 0).
				if base <= 0 {
					log.Printf("ReconcilePOFromAP: PO %s item %s base is non-positive (%v) for unit_uom=%q; left open",
						p.PurchaseCode, pi.PurchaseItem, base, pi.UnitUom)
					continue
				}

				if evaluatePOLineComplete(base, received, tol, active) {
					completedIDs = append(completedIDs, pi.ID)
					completedCount++
				}
			}

			if len(completedIDs) > 0 {
				if err := tx.Model(&models.PurchaseItem{}).
					Where("id IN ?", completedIDs).
					Updates(map[string]interface{}{
						"status":     "COMPLETED",
						"update_dtm": now,
					}).Error; err != nil {
					return err
				}
			}

			// Header roll-up (header is not yet COMPLETED here).
			if completedCount == itemsCount {
				if err := tx.Model(&models.Purchase{}).
					Where("id = ?", p.ID).
					Updates(map[string]interface{}{
						"status":      "COMPLETED",
						"used_type":   "GR",
						"used_status": "COMPLETED",
						"update_dtm":  now,
					}).Error; err != nil {
					return err
				}
			} else if completedCount > 0 {
				if err := tx.Model(&models.Purchase{}).
					Where("id = ?", p.ID).
					Updates(map[string]interface{}{
						"used_type":   "GR",
						"used_status": "PARTIAL",
						"update_dtm":  now,
					}).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}
