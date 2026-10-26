package purchaseRepository

import (
	"fmt"

	"prime-erp-core/internal/utils"

	"gorm.io/gorm"
)

// PurchaseListOptions คือตัวกรอง Multi Select และการเรียงของ GetPurchaseList
// ค่าว่างทั้งหมด = พฤติกรรมเดิม
type PurchaseListOptions struct {
	PurchaseCode []utils.MultiSelectTerm
	DocRef       []utils.MultiSelectTerm
	SupplierCode []utils.MultiSelectTerm
	SupplierName []utils.MultiSelectTerm

	// ระดับ item (purchase_item) ใช้ EXISTS จึงไม่เกิด PO ซ้ำ
	ItemsProductCode  []utils.MultiSelectTerm
	ItemsProductDesc  []utils.MultiSelectTerm
	ItemsProductGroup []utils.MultiSelectTerm

	SortBy  string
	SortDir string
}

const purchaseItemExists = "EXISTS (SELECT 1 FROM purchase_item pit WHERE pit.purchase_id = purchase.id AND %s)"

// item แรกของ PO: หน้าจอแสดง item เดียว หรือ "Multi" ถ้ามีหลายรายการ
func firstPurchaseItemColumn(column string) string {
	return "(SELECT pit." + column + " FROM purchase_item pit WHERE pit.purchase_id = purchase.id " +
		"ORDER BY pit.purchase_item, pit.id LIMIT 1)"
}

// whitelist ของ sort_by → expression ใน SQL (ห้ามรับชื่อคอลัมน์จาก request ตรงๆ)
var purchaseSortColumns = map[string]string{
	"purchase_code":                "purchase.purchase_code",
	"doc_ref":                      "purchase.doc_ref",
	"supplier_code":                "purchase.supplier_code",
	"supplier_name":                "purchase.supplier_name",
	"items_product_code":           firstPurchaseItemColumn("product_code"),
	"items_product_desc":           firstPurchaseItemColumn("product_desc"),
	"items_product_group_one_name": firstPurchaseItemColumn("product_group_name"),
}

// applyPurchaseMultiSelect ต่อเงื่อนไขของแต่ละคอลัมน์แบบ AND (ในคอลัมน์เดียวกัน OR กันอยู่ใน helper แล้ว)
func applyPurchaseMultiSelect(query *gorm.DB, opts PurchaseListOptions) *gorm.DB {
	query = utils.ApplyMultiSelectFilter(query, "purchase.purchase_code", opts.PurchaseCode)
	query = utils.ApplyMultiSelectFilter(query, "purchase.doc_ref", opts.DocRef)
	query = utils.ApplyMultiSelectFilter(query, "purchase.supplier_code", opts.SupplierCode)
	// คอลัมน์ชื่อ: ค้นที่ supplier_name แต่ _in / _not เป็น supplier_code
	query = utils.ApplyMultiSelectFilter(query, "purchase.supplier_code", opts.SupplierName)
	query = applyPurchaseItemMultiSelect(query, "pit.product_code", opts.ItemsProductCode)
	// คอลัมน์ชื่อสินค้า / กลุ่มสินค้า: _in / _not เป็นรหัส
	query = applyPurchaseItemMultiSelect(query, "pit.product_code", opts.ItemsProductDesc)
	query = applyPurchaseItemMultiSelect(query, "pit.product_group_code", opts.ItemsProductGroup)
	return query
}

func applyPurchaseItemMultiSelect(query *gorm.DB, codeColumn string, terms []utils.MultiSelectTerm) *gorm.DB {
	cond, args := utils.MultiSelectCondition(codeColumn, terms)
	if cond == "" {
		return query
	}
	return query.Where(fmt.Sprintf(purchaseItemExists, cond), args...)
}

// purchaseOrderClause คืน ORDER BY ของรายการ PO
// sort_by ว่าง/ไม่อยู่ใน whitelist = เรียงแบบเดิม และใส่ id เป็นตัวเรียงสำรองเสมอ
func purchaseOrderClause(defaultOrder string, sortBy string, sortDir string) string {
	column, ok := purchaseSortColumns[sortBy]
	if !ok {
		return defaultOrder + ", purchase.id"
	}
	dir := "ASC"
	if sortDir == "desc" || sortDir == "DESC" {
		dir = "DESC"
	}
	return column + " " + dir + " NULLS LAST, purchase.id"
}
