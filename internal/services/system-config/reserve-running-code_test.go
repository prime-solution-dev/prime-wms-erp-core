package systemConfigService

import (
	"fmt"
	"testing"
	"time"

	"prime-erp-core/internal/models"
)

func TestBuildRunningCodesPadsAndIncrements(t *testing.T) {
	codes := buildRunningCodes(models.RunningConfigJSON{
		Year:           "2026",
		Month:          "08",
		Prefix:         "DBS",
		RunningDigit:   4,
		CurrentRunning: 41,
	}, 3)

	want := []string{"DBS202608-0042", "DBS202608-0043", "DBS202608-0044"}
	if len(codes) != len(want) {
		t.Fatalf("got %d codes, want %d", len(codes), len(want))
	}

	for i := range want {
		if codes[i] != want[i] {
			t.Errorf("codes[%d] = %q, want %q", i, codes[i], want[i])
		}
	}
}

func TestBuildRunningCodesStartsAtOneOnAFreshPeriod(t *testing.T) {
	codes := buildRunningCodes(models.RunningConfigJSON{
		Year:           "2026",
		Month:          "09",
		Prefix:         "QU",
		RunningDigit:   4,
		CurrentRunning: 0,
	}, 1)

	if codes[0] != "QU202609-0001" {
		t.Errorf("codes[0] = %q, want QU202609-0001", codes[0])
	}
}

func TestStandardRunningPeriodUsesGregorianYear(t *testing.T) {
	now := time.Now()
	period := StandardRunningPeriod()

	if period.Year != now.Format("2006") {
		t.Errorf("Year = %q, want %q", period.Year, now.Format("2006"))
	}

	if period.Month != now.Format("01") {
		t.Errorf("Month = %q, want %q", period.Month, now.Format("01"))
	}
}

// ฝั่ง invoice ใช้ พ.ศ. 2 หลัก ยกเว้น RUNNING_AP ที่ยังใช้ ค.ศ.
func TestInvoiceRunningPeriodUsesBuddhistYearExceptForAP(t *testing.T) {
	now := time.Now()

	if got, want := InvoiceRunningPeriod("RUNNING_AR").Year, fmt.Sprintf("%02d", (now.Year()+543)%100); got != want {
		t.Errorf("RUNNING_AR year = %q, want %q", got, want)
	}

	if got, want := InvoiceRunningPeriod("RUNNING_AP").Year, fmt.Sprintf("%02d", now.Year()%100); got != want {
		t.Errorf("RUNNING_AP year = %q, want %q", got, want)
	}
}
