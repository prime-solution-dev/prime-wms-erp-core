package repositoryCredit

import (
	"prime-erp-core/internal/utils"

	"gorm.io/gorm"
)

// CreditRequestListOptions คือตัวกรอง Multi Select และการเรียงของ GetCreditRequestPreload
// ค่าว่างทั้งหมด = พฤติกรรมเดิม
type CreditRequestListOptions struct {
	// customer_code_likes ค้นที่ credit_request.customer_code ตรงๆ
	CustomerCode []utils.MultiSelectTerm
	// customer_name_likes: ชื่อไม่อยู่ในตารางนี้ service resolve จาก master เป็นรหัสมาให้
	// ไม่ว่าง = AND customer_code IN รหัสเหล่านี้ (ว่าง = ไม่กรองด้วยชื่อ)
	CustomerNameCodes []string

	// SortBy รองรับ customer_code (customer_name เรียงใน service หลัง resolve ชื่อ)
	SortBy  string
	SortDir string
}

// applyCreditRequestMultiSelect ต่อด้วย AND เสมอ ไม่ปนกับ scope customer_code เดิม
func applyCreditRequestMultiSelect(query *gorm.DB, opts CreditRequestListOptions) *gorm.DB {
	query = utils.ApplyMultiSelectFilter(query, "customer_code", opts.CustomerCode)
	if len(opts.CustomerNameCodes) > 0 {
		query = query.Where("customer_code IN ?", opts.CustomerNameCodes)
	}
	return query
}

// creditRequestOrderClause เรียงจาก subquery ที่ group ตาม customer_code แล้ว
// จึงใช้ customer_code (unique ต่อแถว) เป็นตัวเรียงสำรองแทน id
func creditRequestOrderClause(sortBy string, sortDir string) string {
	if sortBy != "customer_code" {
		return "sub.is_approve DESC, sub.customer_code"
	}
	if sortDir == "desc" || sortDir == "DESC" {
		return "sub.customer_code DESC"
	}
	return "sub.customer_code ASC"
}
