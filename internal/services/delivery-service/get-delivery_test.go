package deliveryService

import (
	"reflect"
	"testing"
)

func TestComputePageBounds(t *testing.T) {
	tests := []struct {
		name           string
		totalRecords   int
		page           int
		pageSize       int
		wantStart      int
		wantEnd        int
		wantTotalPages int
	}{
		{
			name:           "normal page",
			totalRecords:   25,
			page:           2,
			pageSize:       10,
			wantStart:      10,
			wantEnd:        20,
			wantTotalPages: 3,
		},
		{
			name:           "last partial page",
			totalRecords:   25,
			page:           3,
			pageSize:       10,
			wantStart:      20,
			wantEnd:        25,
			wantTotalPages: 3,
		},
		{
			name:           "page beyond the end",
			totalRecords:   25,
			page:           5,
			pageSize:       10,
			wantStart:      25,
			wantEnd:        25,
			wantTotalPages: 3,
		},
		{
			name:           "page size zero returns everything as one page",
			totalRecords:   25,
			page:           2,
			pageSize:       0,
			wantStart:      0,
			wantEnd:        25,
			wantTotalPages: 25,
		},
		{
			name:           "page size negative returns everything as one page",
			totalRecords:   25,
			page:           1,
			pageSize:       -1,
			wantStart:      0,
			wantEnd:        25,
			wantTotalPages: 25,
		},
		{
			name:           "page zero or negative returns everything as one page",
			totalRecords:   25,
			page:           0,
			pageSize:       10,
			wantStart:      0,
			wantEnd:        25,
			wantTotalPages: 25,
		},
		{
			name:           "empty result set",
			totalRecords:   0,
			page:           1,
			pageSize:       10,
			wantStart:      0,
			wantEnd:        0,
			wantTotalPages: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, totalPages := computePageBounds(tt.totalRecords, tt.page, tt.pageSize)
			if start != tt.wantStart || end != tt.wantEnd || totalPages != tt.wantTotalPages {
				t.Errorf("computePageBounds(%d, %d, %d) = (%d, %d, %d), want (%d, %d, %d)",
					tt.totalRecords, tt.page, tt.pageSize,
					start, end, totalPages,
					tt.wantStart, tt.wantEnd, tt.wantTotalPages)
			}
			// bounds ที่คืนมาต้อง slice ได้เสมอโดยไม่ panic บน slice ที่ยาวเท่า totalRecords
			s := make([]int, tt.totalRecords)
			_ = s[start:end]
		})
	}
}

func TestImpliedDeliveryStatuses(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "draft maps to TEMP",
			input: []string{PickPackStatusDraft},
			want:  []string{"TEMP"},
		},
		{
			name:  "canceled maps to CANCELED",
			input: []string{PickPackStatusCanceled},
			want:  []string{"CANCELED"},
		},
		{
			name:  "completed maps to COMPLETED",
			input: []string{PickPackStatusCompleted},
			want:  []string{"COMPLETED"},
		},
		{
			name:  "new maps to PENDING",
			input: []string{PickPackStatusNew},
			want:  []string{"PENDING"},
		},
		{
			name:  "pending-pick maps to PENDING",
			input: []string{PickPackStatusPendingPick},
			want:  []string{"PENDING"},
		},
		{
			name:  "pending-pack maps to PENDING",
			input: []string{PickPackStatusPendingPack},
			want:  []string{"PENDING"},
		},
		{
			name:  "unrecognized value contributes nothing",
			input: []string{"bogus"},
			want:  nil,
		},
		{
			name:  "duplicates do not produce duplicate conditions",
			input: []string{PickPackStatusNew, PickPackStatusPendingPick, PickPackStatusPendingPack},
			want:  []string{"PENDING"},
		},
		{
			name:  "distinct statuses each appear once even with repeats",
			input: []string{PickPackStatusDraft, PickPackStatusDraft, PickPackStatusCompleted},
			want:  []string{"TEMP", "COMPLETED"},
		},
		{
			name:  "empty input yields no statuses",
			input: []string{},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := impliedDeliveryStatuses(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("impliedDeliveryStatuses(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildPaymentFilterCondition(t *testing.T) {
	const existsCash = "EXISTS (SELECT 1 FROM sale WHERE sale.sale_code = delivery_booking.document_ref AND sale.payment_method = ?)"
	const notExistsCash = "NOT EXISTS (SELECT 1 FROM sale WHERE sale.sale_code = delivery_booking.document_ref AND sale.payment_method = ?)"

	tests := []struct {
		name          string
		input         []string
		wantCondition string
		wantArgs      []interface{}
	}{
		{
			name:          "cash alone",
			input:         []string{"cash"},
			wantCondition: existsCash,
			wantArgs:      []interface{}{"CASH"},
		},
		{
			name:          "non-cash alone",
			input:         []string{"non-cash"},
			wantCondition: notExistsCash,
			wantArgs:      []interface{}{"CASH"},
		},
		{
			name:          "both values selected means no filtering",
			input:         []string{"cash", "non-cash"},
			wantCondition: "",
			wantArgs:      nil,
		},
		{
			name:          "neither value (empty input) means no filtering",
			input:         []string{},
			wantCondition: "",
			wantArgs:      nil,
		},
		{
			name:          "nil input means no filtering",
			input:         nil,
			wantCondition: "",
			wantArgs:      nil,
		},
		{
			name:          "unrecognized value alone means no filtering",
			input:         []string{"credit"},
			wantCondition: "",
			wantArgs:      nil,
		},
		{
			name:          "unrecognized value mixed with a valid one still filters on the valid one",
			input:         []string{"credit", "cash"},
			wantCondition: existsCash,
			wantArgs:      []interface{}{"CASH"},
		},
		{
			name:          "duplicates of cash still produce a single cash condition",
			input:         []string{"cash", "CASH", "cash"},
			wantCondition: existsCash,
			wantArgs:      []interface{}{"CASH"},
		},
		{
			name:          "duplicates of non-cash still produce a single non-cash condition",
			input:         []string{"non-cash", "Non-Cash", "non-cash"},
			wantCondition: notExistsCash,
			wantArgs:      []interface{}{"CASH"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCondition, gotArgs := buildPaymentFilterCondition(tt.input)
			if gotCondition != tt.wantCondition {
				t.Errorf("buildPaymentFilterCondition(%v) condition = %q, want %q", tt.input, gotCondition, tt.wantCondition)
			}
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("buildPaymentFilterCondition(%v) args = %v, want %v", tt.input, gotArgs, tt.wantArgs)
			}
		})
	}
}

func TestNormalizePickPackFilter(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "known values pass through lowercased",
			input: []string{"Draft", "NEW"},
			want:  []string{"draft", "new"},
		},
		{
			name:  "unknown values are dropped",
			input: []string{"cancelled", "bogus"},
			want:  nil,
		},
		{
			name:  "mixed known and unknown keeps only known",
			input: []string{"draft", "cancelled", "completed"},
			want:  []string{"draft", "completed"},
		},
		{
			name:  "duplicates collapse to one",
			input: []string{"new", "new", "pending-pick"},
			want:  []string{"new", "pending-pick"},
		},
		{
			name:  "empty input yields empty result",
			input: []string{},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizePickPackFilter(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("normalizePickPackFilter(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
