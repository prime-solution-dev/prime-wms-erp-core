package priceService

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// getPriceLastUpdated คืนเวลาที่ราคาถูกแก้ล่าสุด
// คืน nil เมื่อไม่มีแถวที่ตรงเงื่อนไข ผู้เรียกจะปล่อยหัวเรื่อง Last Updated ให้ว่าง
//
// กรองแค่ CompanyCode / SiteCodes / GroupCodes เท่านั้น ซึ่งเป็นทุกฟิลด์ที่ caller
// ปัจจุบัน (document-core GetPriceExportTableRequest) ส่งมาได้ ยังไม่ครอบคลุม
// EffectiveDateFrom/To และ SubGroupCodes ที่ getGroupSubGroup ใช้กรองข้อมูล export จริง
// ถ้าวันหนึ่งมี caller ส่งฟิลด์เหล่านั้นมา ต้องมาเพิ่มเงื่อนไขที่ query นี้ด้วย
// ไม่งั้น Last Updated จะมาจากข้อมูลที่กว้างกว่าที่ export จริง
//
// ใช้ bind parameter แทนการต่อสตริงแบบ getGroupSubGroup เพราะเป็นโค้ดใหม่
// และ cardinality() ทำให้ตัวกรองที่ไม่ได้ส่งมาไม่มีผลกับเงื่อนไข
func getPriceLastUpdated(sqlxDB *sqlx.DB, req GetPriceListGroupRequest) (*time.Time, error) {
	const query = `
		SELECT MAX(GREATEST(g.update_dtm, COALESCE(sg.update_dtm, g.update_dtm)))
		FROM price_list_group g
		LEFT JOIN price_list_sub_group sg ON sg.price_list_group_id = g.id
		WHERE ($1 = '' OR g.company_code = $1)
		  AND (COALESCE(cardinality($2::text[]), 0) = 0 OR g.site_code = ANY($2))
		  AND (COALESCE(cardinality($3::text[]), 0) = 0 OR g.group_code = ANY($3))`

	var result sql.NullTime
	err := sqlxDB.Get(&result, query,
		req.CompanyCode,
		pq.Array(req.SiteCodes),
		pq.Array(req.GroupCodes),
	)
	if err != nil {
		return nil, err
	}
	if !result.Valid {
		return nil, nil
	}

	t := result.Time
	return &t, nil
}
