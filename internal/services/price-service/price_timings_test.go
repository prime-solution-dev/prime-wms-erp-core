package priceService

import (
	"context"
	"testing"
	"time"
)

func TestPriceTimings(t *testing.T) {
	var nilT priceTimings
	nilT.lap("x", time.Now()) // ต้องไม่ panic

	if timingsFrom(context.Background()) != nil {
		t.Fatal("expected nil timings for bare context")
	}

	tm := priceTimings{}
	ctx := context.WithValue(context.Background(), priceTimingsKey{}, tm)
	if timingsFrom(ctx) == nil {
		t.Fatal("expected timings from context")
	}
	tm["b"] = 1500 * time.Millisecond
	tm["a"] = 20 * time.Millisecond
	tm.lap("a", time.Now().Add(-time.Second))
	if got, want := tm.String(), " a=1.02s b=1.5s"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
