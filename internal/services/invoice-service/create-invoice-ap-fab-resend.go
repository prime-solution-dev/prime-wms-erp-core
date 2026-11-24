package invoiceService

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	models "prime-erp-core/internal/models"
	repositoryInvoice "prime-erp-core/internal/repositories/invoice"
	interfaceService "prime-erp-core/internal/services/interface-service"
	prePurchaseService "prime-erp-core/internal/services/pre-purchase-service"
	purchaseService "prime-erp-core/internal/services/purchase-service"

	"github.com/google/uuid"
)

// AP-FAB (GRA ของ FABRICATION) ถูกบันทึกก่อนแล้วค่อยยิง hook INVOICE/AP/CREATE เข้า TRCloud
// ถ้า hook ล้ม request ตอบ error แต่ GRA อยู่ใน DB แล้ว (external_id ว่าง) — production-core retry
// จะโดนกันซ้ำ (document_ref|document_ref_item) ข้ามไป ใบนั้นเลยไม่เคยเข้า TRCloud
// resendUnsentAPFab ส่ง GRA ใบเดิม (เลขเดิม, item เดิม) ซ้ำแบบเดียวกับตอนสร้าง แล้วเก็บ external_id
// ไม่มีแถว hook_config = ไม่มี TRCloud ให้ส่ง (external_id ว่างเป็นเรื่องปกติ) → ไม่ทำอะไร

// แยกไว้ให้เทสแทนได้
var (
	apFabGetHookConfig  = interfaceService.GetHookConfig
	apFabLoadInvoices   = repositoryInvoice.GetInvoicesWithItemsByIDs
	apFabGetSuppliers   = prePurchaseService.GetSupplierByCode
	apFabGetProducts    = purchaseService.GetProductByCode
	apFabHookInterface  = interfaceService.HookInterface
	apFabSaveExternalID = func(id uuid.UUID, externalID string) error {
		_, err := repositoryInvoice.UpdateInvoice([]models.Invoice{{ID: id, ExternalID: externalID}}, []models.InvoiceItem{})
		return err
	}
)

func resendUnsentAPFab(ctx context.Context, unsent map[uuid.UUID]models.Invoice) error {
	if len(unsent) == 0 {
		return nil
	}
	hookConfig, err := apFabGetHookConfig(ctx, map[string]interface{}{
		"module":    []string{"INVOICE"},
		"topic":     []string{"AP"},
		"sub_topic": []string{"CREATE"},
	})
	if err != nil {
		return err
	}
	if len(hookConfig) == 0 {
		return nil
	}
	urlHook := ""
	for _, hookConfigValue := range hookConfig {
		urlHook = hookConfigValue.HookUrl
	}

	ids := make([]uuid.UUID, 0, len(unsent))
	for id := range unsent {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	invoices, err := apFabLoadInvoices(ids)
	if err != nil {
		return errors.New("failed to load AP-FAB invoices to resend: " + err.Error())
	}

	// party_external_id ไม่ได้เก็บใน DB (gorm "-") — เติมจาก supplier master เหมือนตอนสร้าง
	supplierReq := models.GetSupplierListRequest{}
	for _, inv := range invoices {
		supplierReq.SupplierCodes = append(supplierReq.SupplierCodes, inv.PartyCode)
	}
	mapSupplier, err := apFabGetSuppliers(ctx, supplierReq)
	if err != nil {
		return errors.New("failed to get supplier list: " + err.Error())
	}

	failed := []string{}
	for _, inv := range invoices {
		if strings.TrimSpace(inv.ExternalID) != "" {
			continue // ส่งสำเร็จไปแล้ว (มี request อื่นส่งก่อน)
		}
		if supplier, ok := mapSupplier[inv.PartyCode]; ok {
			inv.PartyExternalID = supplier.ExternalID
		}
		// สินค้าค่าบริการ (PROD_SERVICE) แทน FG แบบเดียวกับตอนสร้าง
		mapProduct, err := apFabGetProducts(ctx, models.GetProductRequest{
			ProductType: []string{"PROD_SERVICE"},
			SiteCode:    []string{inv.SiteCode},
			CompanyCode: []string{inv.CompanyCode},
		})
		if err != nil {
			return errors.New("failed to get product list: " + err.Error())
		}
		for _, product := range mapProduct {
			if inv.DocumentRefType == "FABRICATION" {
				for it := range inv.InvoiceItem {
					inv.InvoiceItem[it].ProductCode = product.ProductCode
					inv.InvoiceItem[it].ProductName = product.ProductName
				}
			}
			break
		}

		res, err := apFabHookInterface(ctx, interfaceService.HookInterfaceRequest{
			RequestData: []models.Invoice{inv},
			UrlHook:     urlHook,
		})
		if err != nil {
			failed = append(failed, inv.InvoiceCode+": "+err.Error())
			continue
		}
		m, _ := res.(map[string]interface{})
		externalID, _ := m["id"].(string)
		if strings.TrimSpace(externalID) == "" {
			failed = append(failed, inv.InvoiceCode+": TRCloud did not return an id")
			continue
		}
		if err := apFabSaveExternalID(inv.ID, externalID); err != nil {
			failed = append(failed, inv.InvoiceCode+": save external_id: "+err.Error())
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("resend AP-FAB to interface failed: %s", strings.Join(failed, "; "))
	}
	return nil
}
