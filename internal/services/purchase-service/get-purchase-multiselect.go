package purchaseService

import (
	"prime-erp-core/internal/models"
	purchaseRepository "prime-erp-core/internal/repositories/purchase"
	"prime-erp-core/internal/utils"
)

// purchaseListOptions แปลง *_likes ของ request เป็น term ของ helper
// Column = คอลัมน์ที่ค้น ILIKE ส่วน _in / _not เป็นรหัสของคอลัมน์ (กำหนดใน repository)
func purchaseListOptions(req models.GetPurchaseRequest) purchaseRepository.PurchaseListOptions {
	opts := purchaseRepository.PurchaseListOptions{SortBy: req.SortBy, SortDir: req.SortDir}
	for _, t := range req.PurchaseCodeLikes {
		opts.PurchaseCode = append(opts.PurchaseCode, utils.MultiSelectTerm{Column: "purchase.purchase_code", Like: t.Like})
	}
	for _, t := range req.DocRefLikes {
		opts.DocRef = append(opts.DocRef, utils.MultiSelectTerm{Column: "purchase.doc_ref", Like: t.Like})
	}
	for _, t := range req.SupplierCodeLikes {
		opts.SupplierCode = append(opts.SupplierCode, utils.MultiSelectTerm{Column: "purchase.supplier_code", Like: t.Like, In: t.In, Not: t.Not})
	}
	for _, t := range req.SupplierNameLikes {
		opts.SupplierName = append(opts.SupplierName, utils.MultiSelectTerm{Column: "purchase.supplier_name", Like: t.Like, In: t.In, Not: t.Not})
	}
	for _, t := range req.ItemsProductCodeLikes {
		opts.ItemsProductCode = append(opts.ItemsProductCode, utils.MultiSelectTerm{Column: "pit.product_code", Like: t.Like, In: t.In, Not: t.Not})
	}
	for _, t := range req.ItemsProductDescLikes {
		opts.ItemsProductDesc = append(opts.ItemsProductDesc, utils.MultiSelectTerm{Column: "pit.product_desc", Like: t.Like, In: t.In, Not: t.Not})
	}
	for _, t := range req.ItemsProductGroupOneNameLikes {
		opts.ItemsProductGroup = append(opts.ItemsProductGroup, utils.MultiSelectTerm{Column: "pit.product_group_name", Like: t.Like, In: t.In, Not: t.Not})
	}
	return opts
}
