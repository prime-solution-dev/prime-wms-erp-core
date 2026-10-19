package saleService

import "testing"

func float64Ptr(v float64) *float64 { return &v }

// ด่านราคาใช้ยอดก่อน VAT ส่วนด่าน credit ยังใช้ยอดรวม VAT
func TestBuildValidateApproveItemSplitsPriceAndCreditAmount(t *testing.T) {
	item := SaleValidationItem{ProductCode: "PCB6.07575", Qty: 1, PriceUnit: 21, PriceListUnit: 21.42, TotalAmount: 22.47, SubtotalExclVat: float64Ptr(21)}

	got := buildValidateApproveItem(item)

	if got.TotalAmount != 22.47 {
		t.Errorf("TotalAmount (credit) = %v, want 22.47", got.TotalAmount)
	}
	if got.PriceCheckAmount == nil || *got.PriceCheckAmount != 21 {
		t.Errorf("PriceCheckAmount = %v, want 21", got.PriceCheckAmount)
	}
}

// หน้าเว็บตัวเก่าที่ยังไม่ส่ง subtotal_excl_vat ต้องได้พฤติกรรมเดิม
func TestBuildValidateApproveItemWithoutSubtotalKeepsOldBehavior(t *testing.T) {
	got := buildValidateApproveItem(SaleValidationItem{TotalAmount: 22.47})
	if got.PriceCheckAmount != nil {
		t.Errorf("PriceCheckAmount = %v, want nil", *got.PriceCheckAmount)
	}
}
