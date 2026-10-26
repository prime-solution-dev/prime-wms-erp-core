package creditService

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"prime-erp-core/internal/models"
)

func TestValidateExtraCreditRequestsAllowsActiveBase(t *testing.T) {
	calls := 0
	err := validateExtraCreditRequests(context.Background(), []models.CreditRequest{
		{CustomerCode: "C001", RequestType: "EXTRA"},
		{CustomerCode: "C001", RequestType: "EXTRA"},
	}, func(_ context.Context, payload string) (interface{}, error) {
		calls++
		var req GetHistoryReq
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			t.Fatal(err)
		}
		if len(req.CustomerCode) != 1 || req.CustomerCode[0] != "C001" {
			t.Fatalf("unexpected customer_code: %#v", req.CustomerCode)
		}
		return ResultHistory{HistoryRes: []GetHistoryRes{{RequestType: "BASE", Status: "ACTIVE"}}}, nil
	})

	if err != nil {
		t.Fatalf("expected request to pass, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected one GetHistory call for duplicate customer, got %d", calls)
	}
}

func TestValidateExtraCreditRequestsRejectsWithoutActiveBase(t *testing.T) {
	err := validateExtraCreditRequests(context.Background(), []models.CreditRequest{
		{CustomerCode: "C001", RequestType: "EXTRA"},
	}, func(context.Context, string) (interface{}, error) {
		return ResultHistory{HistoryRes: []GetHistoryRes{
			{RequestType: "BASE", Status: "INACTIVE"},
			{RequestType: "EXTRA", Status: "ACTIVE"},
		}}, nil
	})

	const expected = "customer C001 must have an ACTIVE BASE credit before creating an EXTRA credit request"
	if err == nil || err.Error() != expected {
		t.Fatalf("expected %q, got %v", expected, err)
	}
}

func TestValidateExtraCreditRequestsSkipsNonExtraRequests(t *testing.T) {
	calls := 0
	err := validateExtraCreditRequests(context.Background(), []models.CreditRequest{
		{CustomerCode: "C001", RequestType: "BASE"},
	}, func(context.Context, string) (interface{}, error) {
		calls++
		return nil, nil
	})

	if err != nil {
		t.Fatalf("expected BASE request to pass, got %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected GetHistory not to be called, got %d calls", calls)
	}
}

func TestValidateExtraCreditRequestsPropagatesGetHistoryError(t *testing.T) {
	expected := errors.New("history unavailable")
	err := validateExtraCreditRequests(context.Background(), []models.CreditRequest{
		{CustomerCode: "C001", RequestType: "EXTRA"},
	}, func(context.Context, string) (interface{}, error) {
		return nil, expected
	})

	if !errors.Is(err, expected) {
		t.Fatalf("expected GetHistory error, got %v", err)
	}
}
