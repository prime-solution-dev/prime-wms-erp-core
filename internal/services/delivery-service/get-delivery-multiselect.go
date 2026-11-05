package deliveryService

import (
	"prime-erp-core/internal/utils"

	"gorm.io/gorm"
)

// Multi Select ที่หัวคอลัมน์ของ GetDelivery (1 entry = 1 คำค้น)
// *_in / *_not เป็นรหัสของคอลัมน์เสมอ ยกเว้นชื่อลูกค้าที่ส่งเป็น customer_code
type DeliveryCodeLikeTerm struct {
	Like string   `json:"delivery_code_like"`
	In   []string `json:"delivery_code_in"`
	Not  []string `json:"delivery_code_not"`
}

type DocumentRefLikeTerm struct {
	Like string   `json:"document_ref_like"`
	In   []string `json:"document_ref_in"`
	Not  []string `json:"document_ref_not"`
}

type CustomerCodeLikeTerm struct {
	Like string   `json:"customer_code_like"`
	In   []string `json:"customer_code_in"`
	Not  []string `json:"customer_code_not"`
}

// ชื่อลูกค้าอยู่ใน sale order (sale.customer_name)
type CustomerNameLikeTerm struct {
	Like string   `json:"customer_name_like"`
	In   []string `json:"customer_code_in"`
	Not  []string `json:"customer_code_not"`
}

// ค้นด้วย ILIKE อย่างเดียว ไม่มี _in / _not
type ShipToAddressLikeTerm struct {
	Like string `json:"ship_to_address_like"`
}

type LicensePlateLikeTerm struct {
	Like string   `json:"license_plate_like"`
	In   []string `json:"license_plate_in"`
	Not  []string `json:"license_plate_not"`
}

type ContactNameLikeTerm struct {
	Like string   `json:"contact_name_like"`
	In   []string `json:"contact_name_in"`
	Not  []string `json:"contact_name_not"`
}

// whitelist ของ sort_by → expression ใน SQL (ห้ามรับชื่อคอลัมน์จาก request ตรงๆ)
// customer_name มาจาก sale ที่ LEFT JOIN อยู่แล้วทั้ง query และ countQuery
var deliverySortColumns = map[string]string{
	"delivery_code":   "delivery_booking.delivery_code",
	"document_ref":    "delivery_booking.document_ref",
	"customer_code":   "delivery_booking.customer_code",
	"customer_name":   "sale.customer_name",
	"ship_to_address": "delivery_booking.ship_to_address",
	"license_plate":   "delivery_booking.license_plate",
	"contact_name":    "delivery_booking.contact_name",
}

// applyDeliveryMultiSelect ต่อเงื่อนไขของแต่ละคอลัมน์แบบ AND (ในคอลัมน์เดียวกัน OR กันอยู่ใน helper แล้ว)
// ต้องเรียกกับทั้ง query และ countQuery เพื่อให้ total ตรงกับแถวที่ได้จริง
func applyDeliveryMultiSelect(query *gorm.DB, req GetDeliveryRequest) *gorm.DB {
	var terms []utils.MultiSelectTerm

	terms = terms[:0]
	for _, t := range req.DeliveryCodeLikes {
		terms = append(terms, utils.MultiSelectTerm{Column: "delivery_booking.delivery_code", Like: t.Like, In: t.In, Not: t.Not})
	}
	query = utils.ApplyMultiSelectFilter(query, "delivery_booking.delivery_code", terms)

	terms = nil
	for _, t := range req.DocumentRefLikes {
		terms = append(terms, utils.MultiSelectTerm{Column: "delivery_booking.document_ref", Like: t.Like, In: t.In, Not: t.Not})
	}
	query = utils.ApplyMultiSelectFilter(query, "delivery_booking.document_ref", terms)

	terms = nil
	for _, t := range req.CustomerCodeLikes {
		terms = append(terms, utils.MultiSelectTerm{Column: "delivery_booking.customer_code", Like: t.Like, In: t.In, Not: t.Not})
	}
	query = utils.ApplyMultiSelectFilter(query, "delivery_booking.customer_code", terms)

	// คอลัมน์ชื่อ: ค้นที่ sale.customer_name แต่ _in / _not เป็น customer_code
	terms = nil
	for _, t := range req.CustomerNameLikes {
		terms = append(terms, utils.MultiSelectTerm{Column: "sale.customer_name", Like: t.Like, In: t.In, Not: t.Not})
	}
	query = utils.ApplyMultiSelectFilter(query, "delivery_booking.customer_code", terms)

	terms = nil
	for _, t := range req.ShipToAddressLikes {
		terms = append(terms, utils.MultiSelectTerm{Column: "delivery_booking.ship_to_address", Like: t.Like})
	}
	query = utils.ApplyMultiSelectFilter(query, "delivery_booking.ship_to_address", terms)

	terms = nil
	for _, t := range req.LicensePlateLikes {
		terms = append(terms, utils.MultiSelectTerm{Column: "delivery_booking.license_plate", Like: t.Like, In: t.In, Not: t.Not})
	}
	query = utils.ApplyMultiSelectFilter(query, "delivery_booking.license_plate", terms)

	terms = nil
	for _, t := range req.ContactNameLikes {
		terms = append(terms, utils.MultiSelectTerm{Column: "delivery_booking.contact_name", Like: t.Like, In: t.In, Not: t.Not})
	}
	return utils.ApplyMultiSelectFilter(query, "delivery_booking.contact_name", terms)
}

// deliveryOrderClause คืน ORDER BY ของรายการ delivery
// sort_by ว่าง/ไม่อยู่ใน whitelist = เรียงแบบเดิม (update_date desc) และใส่ id เป็นตัวเรียงสำรองเสมอ
func deliveryOrderClause(sortBy string, sortDir string) string {
	column, ok := deliverySortColumns[sortBy]
	if !ok {
		return "delivery_booking.update_date DESC, delivery_booking.id"
	}
	dir := "ASC"
	if sortDir == "desc" || sortDir == "DESC" {
		dir = "DESC"
	}
	return column + " " + dir + " NULLS LAST, delivery_booking.id"
}
