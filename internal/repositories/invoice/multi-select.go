package saleRepository

import (
	"fmt"
	"prime-erp-core/internal/utils"

	"gorm.io/gorm"
)

// InvoiceListOptions คือตัวกรอง Multi Select และการเรียงของ GetInvoicePreload
// ค่าว่างทั้งหมด = พฤติกรรมเดิม
type InvoiceListOptions struct {
	InvoiceCode  []utils.MultiSelectTerm
	InvoiceRef   []utils.MultiSelectTerm
	Packing      []utils.MultiSelectTerm
	SalesOrder   []utils.MultiSelectTerm
	CustomerCode []utils.MultiSelectTerm
	CustomerName []utils.MultiSelectTerm

	SortBy  string
	SortDir string
}

// packing / sales order อยู่ใน invoice_item ใช้ EXISTS แทน join จะได้ไม่เกิดแถว invoice ซ้ำ
const invoiceItemExists = "EXISTS (SELECT 1 FROM invoice_item ii WHERE ii.invoice_id = invoice.id AND %s)"

// item แรกตามที่หน้าจอแสดง: เลขที่ item น้อยสุดก่อน, item ที่ไม่ใช่ตัวเลขไปท้าย
func firstInvoiceItemColumn(column string) string {
	return "(SELECT ii." + column + " FROM invoice_item ii WHERE ii.invoice_id = invoice.id " +
		"ORDER BY (CASE WHEN ii.invoice_item ~ '^[0-9]+$' THEN 0 ELSE 1 END), " +
		"(CASE WHEN ii.invoice_item ~ '^[0-9]+$' THEN ii.invoice_item::numeric END), ii.invoice_item, ii.id LIMIT 1)"
}

// whitelist ของ sort_by → expression ใน SQL (ห้ามรับชื่อคอลัมน์จาก request ตรงๆ)
var invoiceSortColumns = map[string]string{
	"invoice_code":  "invoice.invoice_code",
	"invoice_ref":   "invoice.document_ref",
	"customer_code": "invoice.party_code",
	"customer_name": "invoice.party_name",
	"packing":       firstInvoiceItemColumn("source_code"),
	"sales_order":   firstInvoiceItemColumn("document_ref"),
}

// applyMultiSelect ต่อเงื่อนไขของแต่ละคอลัมน์แบบ AND (ในคอลัมน์เดียวกัน OR กันอยู่ใน helper แล้ว)
func applyMultiSelect(query *gorm.DB, opts InvoiceListOptions) *gorm.DB {
	query = utils.ApplyMultiSelectFilter(query, "invoice.invoice_code", opts.InvoiceCode)
	query = utils.ApplyMultiSelectFilter(query, "invoice.document_ref", opts.InvoiceRef)
	query = utils.ApplyMultiSelectFilter(query, "invoice.party_code", opts.CustomerCode)
	// คอลัมน์ชื่อ: ค้นที่ party_name แต่ _in / _not เป็น party_code
	query = utils.ApplyMultiSelectFilter(query, "invoice.party_code", opts.CustomerName)
	query = applyItemMultiSelect(query, "ii.source_code", opts.Packing)
	query = applyItemMultiSelect(query, "ii.document_ref", opts.SalesOrder)
	return query
}

func applyItemMultiSelect(query *gorm.DB, itemColumn string, terms []utils.MultiSelectTerm) *gorm.DB {
	cond, args := utils.MultiSelectCondition(itemColumn, terms)
	if cond == "" {
		return query
	}
	return query.Where(fmt.Sprintf(invoiceItemExists, cond), args...)
}

// invoiceOrderClause คืน ORDER BY ของรายการ invoice
// sort_by ว่าง/ไม่อยู่ใน whitelist = เรียงแบบเดิม (update_dtm desc) และใส่ id เป็นตัวเรียงสำรองเสมอ
func invoiceOrderClause(sortBy string, sortDir string) string {
	column, ok := invoiceSortColumns[sortBy]
	if !ok {
		return "update_dtm desc, id"
	}
	dir := "ASC"
	if sortDir == "desc" || sortDir == "DESC" {
		dir = "DESC"
	}
	return column + " " + dir + " NULLS LAST, id"
}
