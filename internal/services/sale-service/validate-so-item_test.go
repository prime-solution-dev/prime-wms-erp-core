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

// ค่าขนส่งต้องไปถึง verify ให้หักแบบเดียวกับตอน Submit QU (INCL) และบวก credit (EXCL)
// ยอดรวมของใบใช้เกลี่ยค่าขนส่ง INCL ลงแต่ละแถว ต้องเป็นยอดเดียวกับที่ใช้เทียบราคา (ก่อน VAT)
func TestBuildValidateApproveDocumentCarriesTransport(t *testing.T) {
	doc := buildValidateApproveDocument(SaleValidationDocument{
		CompanyCode:   "C1",
		SiteCode:      "S1",
		CustomerCode:  "TV0139",
		TransportCost: 12,
		TransportType: "INCL",
		Items: []SaleValidationItem{
			{ProductCode: "A", TotalAmount: 107, SubtotalExclVat: float64Ptr(100), TotalWeight: 2},
			{ProductCode: "B", TotalAmount: 53.5, SubtotalExclVat: float64Ptr(50), TotalWeight: 1},
		},
	})

	if doc.TransportCost != 12 || doc.TransportType != "INCL" {
		t.Errorf("transport = %v %q, want 12 INCL", doc.TransportCost, doc.TransportType)
	}
	if doc.TotalAmount != 150 {
		t.Errorf("TotalAmount = %v, want 150 (sum before VAT)", doc.TotalAmount)
	}
	if doc.TotalWeight != 3 {
		t.Errorf("TotalWeight = %v, want 3", doc.TotalWeight)
	}
	if len(doc.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(doc.Items))
	}
}

// หน้าเว็บตัวเก่าไม่ส่ง subtotal_excl_vat: ยอดรวมใช้ total_amount แทน
func TestBuildValidateApproveDocumentFallsBackToTotalAmount(t *testing.T) {
	doc := buildValidateApproveDocument(SaleValidationDocument{
		Items: []SaleValidationItem{{ProductCode: "A", TotalAmount: 107}},
	})
	if doc.TotalAmount != 107 {
		t.Errorf("TotalAmount = %v, want 107", doc.TotalAmount)
	}
}
