package priceService

import (
	"context"
	"encoding/json"
	"testing"

	"prime-erp-core/internal/models"
	"prime-erp-core/internal/requestcontext"

	"github.com/google/uuid"
)

// UpdateExtras เคยเขียน UpdateBy เป็น "system" ตายตัว (// TODO: get user from auth)
// เปลี่ยนมาอ่านจาก requestcontext.GetUserOrDefault(ctx) แล้ว — เทสนี้ยืนยันว่า user ที่อยู่ใน
// context ไหลไปถึงแถวที่เขียนจริง ไม่ใช่แค่ว่า compile ผ่าน (กับดักที่โปรเจกต์นี้เจอมาแล้ว 2 ครั้ง
// คือ user หายเงียบๆ กลายเป็นค่าว่างหรือค่า hardcode)
func TestUpdateExtras_UserFromContextLandsInUpdateBy(t *testing.T) {
	var captured []models.PriceListGroupExtra
	original := updateExtraFunc
	updateExtraFunc = func(extras []models.PriceListGroupExtra) error {
		captured = extras
		return nil
	}
	defer func() { updateExtraFunc = original }()

	payload := []map[string]interface{}{{
		"price_list_group_id": uuid.New().String(),
		"extra_key":           "PG06_1",
		"condition_code":      "PG06",
		"operator":            "<=",
		"value_int":           1.0,
		"length_extra_key":    1,
		"cond_range_min":      0.0,
		"cond_range_max":      45.0,
		"price_list_group_extra_keys": []map[string]interface{}{
			{"code": "PG06", "value": "PG06_1", "seq": 1},
		},
	}}
	body, _ := json.Marshal(payload)

	ctx := requestcontext.WithUser(context.Background(), "somchai")

	if _, err := UpdateExtras(ctx, string(body)); err != nil {
		t.Fatalf("UpdateExtras ต้องผ่าน แต่ได้ error: %v", err)
	}

	if len(captured) != 1 {
		t.Fatalf("ต้องเขียน 1 แถว ได้ %d", len(captured))
	}
	if captured[0].UpdateBy != "somchai" {
		t.Fatalf("UpdateBy = %q, ต้องการ somchai", captured[0].UpdateBy)
	}
}

// ไม่มี user ใน context (เช่นถูกเรียกจากระบบที่ยังไม่ผ่าน middleware ใหม่) ต้องได้ค่าว่าง
// ไม่ใช่ "system" — requestcontext.GetUserOrDefault คืนค่าว่างโดยตั้งใจ ไม่ใช่ชื่อสมมติ
func TestUpdateExtras_NoUserInContextFallsBackToEmpty(t *testing.T) {
	var captured []models.PriceListGroupExtra
	original := updateExtraFunc
	updateExtraFunc = func(extras []models.PriceListGroupExtra) error {
		captured = extras
		return nil
	}
	defer func() { updateExtraFunc = original }()

	payload := []map[string]interface{}{{
		"price_list_group_id": uuid.New().String(),
		"extra_key":           "PG06_1",
		"condition_code":      "PG06",
		"operator":            "<=",
		"value_int":           1.0,
		"length_extra_key":    1,
		"cond_range_min":      0.0,
		"cond_range_max":      45.0,
		"price_list_group_extra_keys": []map[string]interface{}{
			{"code": "PG06", "value": "PG06_1", "seq": 1},
		},
	}}
	body, _ := json.Marshal(payload)

	if _, err := UpdateExtras(context.Background(), string(body)); err != nil {
		t.Fatalf("UpdateExtras ต้องผ่าน แต่ได้ error: %v", err)
	}

	if len(captured) != 1 {
		t.Fatalf("ต้องเขียน 1 แถว ได้ %d", len(captured))
	}
	if captured[0].UpdateBy != "" {
		t.Fatalf("UpdateBy = %q, ต้องการค่าว่าง (ไม่ใช่ system)", captured[0].UpdateBy)
	}
}
