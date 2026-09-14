package quotationService

import (
	"testing"
	"time"
)

// buildQuotationApproveDateMap ต้องนับเฉพาะแถวที่ status='COMPLETED' และแมพวันที่อนุมัติ
// ให้ตรงกับ quotation_code ของมัน — quotation ที่ไม่มีแถว COMPLETED ต้องไม่ติดอยู่ใน map
func TestBuildQuotationApproveDateMap(t *testing.T) {
	completedDate := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)
	olderCompletedDate := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)
	newerCompletedDate := time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		rows []quotationApprovalRow
		want map[string]time.Time
	}{
		{
			name: "match found",
			rows: []quotationApprovalRow{
				{DocumentCode: "QO-001", Status: "COMPLETED", UpdateDate: completedDate},
			},
			want: map[string]time.Time{"QO-001": completedDate},
		},
		{
			name: "no match stays out of map เมื่อไม่มีแถว COMPLETED เลย",
			rows: []quotationApprovalRow{
				{DocumentCode: "QO-002", Status: "PENDING", UpdateDate: completedDate},
			},
			want: map[string]time.Time{},
		},
		{
			name: "multiple quotations mapped correctly",
			rows: []quotationApprovalRow{
				{DocumentCode: "QO-001", Status: "COMPLETED", UpdateDate: completedDate},
				{DocumentCode: "QO-002", Status: "COMPLETED", UpdateDate: olderCompletedDate},
			},
			want: map[string]time.Time{
				"QO-001": completedDate,
				"QO-002": olderCompletedDate,
			},
		},
		{
			name: "only status COMPLETED counted แม้จะมีแถวอื่นของ quotation เดียวกัน",
			rows: []quotationApprovalRow{
				{DocumentCode: "QO-001", Status: "REJECT", UpdateDate: newerCompletedDate},
				{DocumentCode: "QO-001", Status: "COMPLETED", UpdateDate: completedDate},
			},
			want: map[string]time.Time{"QO-001": completedDate},
		},
		{
			name: "quotation เดียวกันมีหลายแถว COMPLETED ให้ใช้วันที่ล่าสุด",
			rows: []quotationApprovalRow{
				{DocumentCode: "QO-001", Status: "COMPLETED", UpdateDate: olderCompletedDate},
				{DocumentCode: "QO-001", Status: "COMPLETED", UpdateDate: newerCompletedDate},
			},
			want: map[string]time.Time{"QO-001": newerCompletedDate},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := buildQuotationApproveDateMap(c.rows)
			if len(got) != len(c.want) {
				t.Fatalf("len(got) = %d, want %d (got=%v)", len(got), len(c.want), got)
			}
			for code, wantDate := range c.want {
				gotDate, ok := got[code]
				if !ok {
					t.Fatalf("missing %q in result", code)
				}
				if !gotDate.Equal(wantDate) {
					t.Errorf("result[%q] = %v, want %v", code, gotDate, wantDate)
				}
			}
		})
	}
}

// fillQuotationApproveDates ต้องเติม ApproveDate เฉพาะใบที่พบใน map เท่านั้น
// ใบที่ไม่พบต้องคง ApproveDate เป็น nil ไม่ใช่ zero-value time
func TestFillQuotationApproveDates(t *testing.T) {
	approveDate := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)
	quotations := []GetQuotationResponse{
		{QuotationCode: "QO-001"},
		{QuotationCode: "QO-002"},
	}
	approveDateMap := map[string]time.Time{"QO-001": approveDate}

	fillQuotationApproveDates(quotations, approveDateMap)

	if quotations[0].ApproveDate == nil || !quotations[0].ApproveDate.Equal(approveDate) {
		t.Errorf("QO-001 ApproveDate = %v, want %v", quotations[0].ApproveDate, approveDate)
	}
	if quotations[1].ApproveDate != nil {
		t.Errorf("QO-002 ApproveDate = %v, want nil (ไม่มีใน map)", quotations[1].ApproveDate)
	}
}
