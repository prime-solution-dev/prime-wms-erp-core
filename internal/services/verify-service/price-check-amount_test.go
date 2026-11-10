package verifyService

import "testing"

func float64Ref(v float64) *float64 { return &v }

// ยอดที่ใช้เทียบ price list แยกจากยอดที่ใช้คิด credit
// เคส QU202610-0503-R2: total_amount รวม VAT 22.47 ทำให้ราคา 21 ผ่าน price list 21.42
func TestPriceCheckAmountPrefersAmountBeforeVat(t *testing.T) {
	item := VerifyApproveItem{TotalAmount: 22.47, PriceCheckAmount: float64Ref(21)}
	if got := priceCheckAmount(item); got != 21 {
		t.Errorf("priceCheckAmount = %v, want 21", got)
	}
}

// ผู้เรียกเดิม (QU create/update, SO update) ไม่ส่งค่านี้ ต้องได้พฤติกรรมเดิม
func TestPriceCheckAmountFallsBackToTotalAmount(t *testing.T) {
	item := VerifyApproveItem{TotalAmount: 21}
	if got := priceCheckAmount(item); got != 21 {
		t.Errorf("priceCheckAmount = %v, want 21", got)
	}
}
