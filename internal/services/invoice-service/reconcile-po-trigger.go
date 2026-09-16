package invoiceService

import (
	"errors"
	models "prime-erp-core/internal/models"
	purchaseService "prime-erp-core/internal/services/purchase-service"
	"strings"
)

// reconcilePOAfterAPSave closes / updates the POs referenced by a just-saved AP
// invoice (GRA), driven from persisted state.
//
// Only COMPLETED (submitted) invoices trigger a close — a PENDING draft is a
// no-op, because completion is judged against the cumulative COMPLETED-AP sum in
// the DB (which excludes drafts). It gathers the distinct PO codes (document_ref)
// referenced by COMPLETED invoice lines, fetches the product master (for the
// per-unit tolerance), and hands off to the reconcile routine. PO codes that are
// not real POs are ignored by the reconcile routine. Idempotent — safe to call on
// every create/update of an AP invoice.
func reconcilePOAfterAPSave(req []models.Invoice) error {
	// Process each COMPLETED invoice independently so the product master is always
	// fetched with THAT invoice's own company/site (a batch spanning companies must
	// not resolve one invoice's products under another's scope).
	for _, invoice := range req {
		if !strings.EqualFold(strings.TrimSpace(invoice.Status), "COMPLETED") {
			continue
		}

		poCodeSeen := map[string]struct{}{}
		poCodes := []string{}
		productSeen := map[string]struct{}{}
		productCodes := []string{}
		for _, item := range invoice.InvoiceItem {
			poCode := strings.TrimSpace(item.DocumentRef)
			if poCode != "" {
				if _, ok := poCodeSeen[poCode]; !ok {
					poCodeSeen[poCode] = struct{}{}
					poCodes = append(poCodes, poCode)
				}
			}
			if item.ProductCode != "" {
				if _, ok := productSeen[item.ProductCode]; !ok {
					productSeen[item.ProductCode] = struct{}{}
					productCodes = append(productCodes, item.ProductCode)
				}
			}
		}

		if len(poCodes) == 0 {
			continue
		}

		productMap, err := purchaseService.GetProductByCode(models.GetProductRequest{
			ProductCode: productCodes,
			SiteCode:    []string{invoice.SiteCode},
			CompanyCode: []string{invoice.CompanyCode},
		})
		if err != nil {
			return errors.New("failed to get product master for PO reconcile: " + err.Error())
		}

		if err := purchaseService.ReconcilePOFromAP(poCodes, productMap); err != nil {
			return err
		}
	}

	return nil
}
