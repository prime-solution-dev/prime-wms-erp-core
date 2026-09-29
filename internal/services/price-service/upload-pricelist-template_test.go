package priceService

import (
	"context"
	"testing"

	"prime-erp-core/internal/requestcontext"
)

// templateSheet builds the single "Pricelist" sheet buildCreatePricelistRequestFromTemplate
// expects: Product Group, Product code, Price_unit, Price_weight, and at least one
// "Product Group N" column.
func templateSheet(rows ...[]string) [][]string {
	out := [][]string{{"Product Group", "Product code", "Price_unit", "Price_weight", "Product Group 1"}}
	return append(out, rows...)
}

// create_by ต้อง fallback ไปที่ user ใน context (requestcontext.GetUserOrDefault) ไม่ใช่
// ค่า "system" ตายตัวแบบเดิม — form field ยังเป็นแหล่งหลักเหมือนเดิม (request shape เดิม)
// เทสนี้ปักว่าแถวที่ build ออกมาต้องพก user จาก context จริงๆ เมื่อ form ไม่ส่ง create_by มา
// (CreatePricelist เขียน DTO.CreateBy ลง create_by/update_by ตรงๆ ไม่มีการแปลงเพิ่ม จึงเท่ากับ
// ทดสอบแถวที่ถูกเขียนจริง)
func TestUploadPricelistTemplate_CreateByFallsBackToContextUser(t *testing.T) {
	ctx := requestcontext.WithUser(context.Background(), "somchai")

	xlsx := buildXlsx(t, sheets{
		"Pricelist": templateSheet([]string{"G1", "SG01", "10", "20", "V1"}),
	})

	opts := templateParseOptions{
		CompanyCode: defaultTemplateCompanyCode,
		SiteCode:    defaultTemplateSiteCode,
		Sheet:       defaultTemplateSheet,
		// form ไม่ส่ง create_by มาเลย เหมือนกรณีจริงที่ทดสอบ
		CreateBy: resolveTemplateCreateBy(ctx, map[string][]string{}),
	}

	req, err := buildCreatePricelistRequestFromTemplate(xlsx, opts)
	if err != nil {
		t.Fatalf("buildCreatePricelistRequestFromTemplate: %v", err)
	}
	if len(req.SubGroups) != 1 {
		t.Fatalf("ต้องได้ 1 subgroup ได้ %d", len(req.SubGroups))
	}
	if req.SubGroups[0].CreateBy != "somchai" {
		t.Fatalf("CreateBy = %q, ต้องการ somchai (จาก context) ไม่ใช่ system", req.SubGroups[0].CreateBy)
	}
}

// form ที่ส่ง create_by มาเองยังต้องชนะ (form field เป็นแหล่งหลัก, context เป็นแค่ fallback)
func TestUploadPricelistTemplate_CreateByFormFieldStillWins(t *testing.T) {
	ctx := requestcontext.WithUser(context.Background(), "somchai")

	got := resolveTemplateCreateBy(ctx, map[string][]string{"create_by": {"explicit-user"}})
	if got != "explicit-user" {
		t.Fatalf("CreateBy = %q, ต้องการ explicit-user (form ต้องชนะ context)", got)
	}
}

// ไม่มีทั้ง form field และ user ใน context ต้องได้ค่าว่าง ไม่ใช่ "system"
func TestUploadPricelistTemplate_CreateByNoUserFallsBackToEmpty(t *testing.T) {
	got := resolveTemplateCreateBy(context.Background(), map[string][]string{})
	if got != "" {
		t.Fatalf("CreateBy = %q, ต้องการค่าว่าง (ไม่ใช่ system)", got)
	}
}
