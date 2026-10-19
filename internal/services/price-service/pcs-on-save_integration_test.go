//go:build integration

package priceService

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// save ราคา kg จากหน้า detail ของ group ที่ไม่มีราคาฐาน (GROUP_1_ITEM_6) ต้องได้ pcs = kg x weight_spec
// ลง DB จริง โดย kg ที่กรอกและ extra ไม่ถูกแตะ
func TestUpdatePriceListSubGroupIntegration_KgSaveComputesPcs(t *testing.T) {
	ensureSqlxEnvForTest(t)
	gormx := openTestDB(t)
	truncateAll(t, gormx)

	groupID, subGroupID := uuid.New(), uuid.New()
	seed := []struct {
		sql  string
		args []interface{}
	}{
		{`INSERT INTO price_list_group (id, company_code, site_code, group_code, price_unit, price_weight)
			VALUES (?, 'C1', 'S1', 'GROUP_1_ITEM_6', 0, 0)`, []interface{}{groupID}},
		{`INSERT INTO price_list_sub_group (id, price_list_group_id, subgroup_code, subgroup_key,
			total_net_price_unit, total_net_price_weight, extra_price_unit, extra_price_weight, need_recalculate)
			VALUES (?, ?, 'G16_SUB', 'G16_SUB', 0, 0, 3, 3, true)`, []interface{}{subGroupID, groupID}},
		{`INSERT INTO price_list_sub_group_key (id, sub_group_id, seq, code, value) VALUES (?, ?, 1, 'PRODUCT', 'P001')`,
			[]interface{}{uuid.New(), subGroupID}},
		{`INSERT INTO price_list_formulas (id, formula_code, name, uom, formula_type, expression, params, rounding, create_dtm)
			VALUES (?, 'G16_KG', 'kg', 'kg', 'price_calc', 'base_price+extra', '{"required":["base_price","extra"]}', 2, now()),
			       (?, 'G16_PCS', 'pcs', 'pcs', 'price_calc', 'kg*weight_spec', '{"required":["kg","weight_spec"]}', 2, now())`,
			[]interface{}{uuid.New(), uuid.New()}},
		{`INSERT INTO price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
			VALUES (?, 'G16_SUB', 'G16_KG', true, now()), (?, 'G16_SUB', 'G16_PCS', false, now())`,
			[]interface{}{uuid.New(), uuid.New()}},
	}
	for _, s := range seed {
		require.NoError(t, gormx.Exec(s.sql, s.args...).Error, s.sql)
	}

	srv := fakeWarehouseServer(t, `[{"id": "`+subGroupID.String()+`", "product_code": "P001", "weight_spec": 148, "inventory_weight": []}]`)
	defer srv.Close()
	pointWarehouseEndpointAt(t, srv.URL)

	_, err := UpdatePriceListSubGroup(context.Background(),
		`{"changes":[{"subgroup_id":"`+subGroupID.String()+`","total_net_price_weight":4150}]}`)
	require.NoError(t, err)

	var row struct {
		TotalNetPriceUnit   float64
		TotalNetPriceWeight float64
		ExtraPriceUnit      float64
		ExtraPriceWeight    float64
		NeedRecalculate     bool
	}
	require.NoError(t, gormx.Raw(`SELECT total_net_price_unit, total_net_price_weight, extra_price_unit,
		extra_price_weight, need_recalculate FROM price_list_sub_group WHERE id = ?`, subGroupID).Scan(&row).Error)
	require.Equal(t, 4150.0, row.TotalNetPriceWeight)
	require.Equal(t, 614200.0, row.TotalNetPriceUnit, "4150 x 148")
	require.Equal(t, 3.0, row.ExtraPriceUnit)
	require.Equal(t, 3.0, row.ExtraPriceWeight)
	require.False(t, row.NeedRecalculate)
	require.Equal(t, int64(1), count(t, gormx, "price_list_sub_group_history"), "save ครั้งเดียว = history แถวเดียว")

	// cascade/UpdateLatest ภายหลังต้องไม่ทับ kg ด้วย 0+extra
	_, err = UpdateLatestPriceListSubGroup(context.Background(), `{"subgroup_ids":["`+subGroupID.String()+`"]}`)
	require.NoError(t, err)
	require.NoError(t, gormx.Raw(`SELECT total_net_price_unit, total_net_price_weight, extra_price_unit,
		extra_price_weight, need_recalculate FROM price_list_sub_group WHERE id = ?`, subGroupID).Scan(&row).Error)
	require.Equal(t, 4150.0, row.TotalNetPriceWeight)
	require.Equal(t, 614200.0, row.TotalNetPriceUnit)
}
