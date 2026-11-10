package systemConfigService

import (
	"encoding/json"
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

// TestReserveRunningCodesFromConfigUsesPrefixWithoutPersistingIt pins the exact bug that
// produced the CO numbering incident: a caller passing a per-call prefix (invoice's
// RUNNING_AR alternates "IV"/"CS", RUNNING_CN alternates "CN"/"CC") must get codes built
// with that prefix, but the config that gets written back to the row must keep the
// ORIGINAL stored prefix untouched — never the one-off prefix from this call — because a
// later caller that omits prefix (sale/delivery/quotation always do) relies on the stored
// value as its default.
func TestReserveRunningCodesFromConfigUsesPrefixWithoutPersistingIt(t *testing.T) {
	original := models.RunningConfigJSON{
		Year:           "2026",
		Month:          "09",
		Prefix:         "IV",
		RunningDigit:   4,
		CurrentRunning: 5,
	}
	originalJSON, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal original: %v", err)
	}

	codes, updated := reserveRunningCodesFromConfig(original, 2, "CS", RunningPeriod{Year: "2026", Month: "09"})

	// Half 1: the codes for THIS call use the one-off prefix.
	wantCodes := []string{"CS202609-0006", "CS202609-0007"}
	if len(codes) != len(wantCodes) {
		t.Fatalf("got %d codes, want %d: %v", len(codes), len(wantCodes), codes)
	}
	for i := range wantCodes {
		if codes[i] != wantCodes[i] {
			t.Errorf("codes[%d] = %q, want %q", i, codes[i], wantCodes[i])
		}
	}

	// Half 2: the config that would be written back keeps the ORIGINAL stored prefix,
	// byte-identical to what it was before the call — only current_running moved.
	if updated.Prefix != "IV" {
		t.Fatalf("stored prefix drifted to %q, want unchanged %q", updated.Prefix, "IV")
	}
	updatedJSON, err := json.Marshal(updated)
	if err != nil {
		t.Fatalf("marshal updated: %v", err)
	}
	var originalRoundTrip, updatedRoundTrip map[string]interface{}
	if err := json.Unmarshal(originalJSON, &originalRoundTrip); err != nil {
		t.Fatalf("unmarshal original: %v", err)
	}
	if err := json.Unmarshal(updatedJSON, &updatedRoundTrip); err != nil {
		t.Fatalf("unmarshal updated: %v", err)
	}
	if originalRoundTrip["prefix"] != updatedRoundTrip["prefix"] {
		t.Fatalf("stored JSON prefix field changed: before %v after %v", originalRoundTrip["prefix"], updatedRoundTrip["prefix"])
	}
	if updated.CurrentRunning != original.CurrentRunning+2 {
		t.Errorf("CurrentRunning = %d, want %d", updated.CurrentRunning, original.CurrentRunning+2)
	}
}

// TestReserveRunningCodesFromConfigEmptyPrefixUsesStoredPrefix pins the default path that
// sale/delivery/quotation rely on: an empty prefix argument means "use whatever prefix is
// already stored", both for building codes and for what gets persisted.
func TestReserveRunningCodesFromConfigEmptyPrefixUsesStoredPrefix(t *testing.T) {
	original := models.RunningConfigJSON{
		Year:           "2026",
		Month:          "09",
		Prefix:         "DBS",
		RunningDigit:   4,
		CurrentRunning: 0,
	}

	codes, updated := reserveRunningCodesFromConfig(original, 1, "", RunningPeriod{Year: "2026", Month: "09"})

	if codes[0] != "DBS202609-0001" {
		t.Errorf("codes[0] = %q, want DBS202609-0001", codes[0])
	}
	if updated.Prefix != "DBS" {
		t.Errorf("Prefix = %q, want unchanged %q", updated.Prefix, "DBS")
	}
}

// TestReserveRunningCodesFromConfigResetsOnNewPeriod pins that a period rollover still
// resets Year/Month/CurrentRunning exactly as before, independent of the prefix fix.
func TestReserveRunningCodesFromConfigResetsOnNewPeriod(t *testing.T) {
	original := models.RunningConfigJSON{
		Year:           "2025",
		Month:          "12",
		Prefix:         "QU",
		RunningDigit:   4,
		CurrentRunning: 999,
	}

	codes, updated := reserveRunningCodesFromConfig(original, 1, "", RunningPeriod{Year: "2026", Month: "01"})

	if codes[0] != "QU202601-0001" {
		t.Errorf("codes[0] = %q, want QU202601-0001", codes[0])
	}
	if updated.Year != "2026" || updated.Month != "01" || updated.CurrentRunning != 1 {
		t.Errorf("updated = %+v, want Year=2026 Month=01 CurrentRunning=1", updated)
	}
}
