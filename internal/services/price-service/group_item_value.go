package priceService

import (
	"strconv"
	"strings"

	"prime-erp-core/internal/models"
)

// parseGroupItemValue อ่าน group_item.value ของ item_code ที่ให้มาแล้วแปลงเป็นตัวเลข
//
// value เก็บเป็น varchar พร้อม thousand separator เช่น "10,000.00" จึงต้องตัด ","
// ก่อน ParseFloat ส่วน value_int มีคอลัมน์อยู่แต่เป็น 0 ทั้งตาราง (SyncGroupMaster
// ไม่ populate) จึงใช้แทนไม่ได้
//
// bool ที่คืนคือ "resolve ค่าได้ไหม" ไม่ใช่ "ไม่ใช่ศูนย์" — ต้องแยก value = "0"
// ที่เป็นค่าจริง (คืน (0, true)) ออกจากกรณีไม่มี record / value ว่าง / parse ไม่ได้
// (คืน (0, false)) เพราะ comparator ใช้ bool นี้ตัดสินว่าจะดันไปท้ายรายการหรือไม่
func parseGroupItemValue(m map[string]models.GetGroupItemResponse, code string) (float64, bool) {
	item, ok := m[code]
	if !ok {
		return 0, false
	}

	raw := strings.TrimSpace(strings.ReplaceAll(item.Value, ",", ""))
	if raw == "" {
		return 0, false
	}

	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}

	return v, true
}
