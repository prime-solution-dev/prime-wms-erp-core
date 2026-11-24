package invoiceService

import (
	"context"
	"errors"
	"strings"
	"testing"

	models "prime-erp-core/internal/models"
	interfaceService "prime-erp-core/internal/services/interface-service"

	"github.com/google/uuid"
)

type apFabResendStub struct {
	hookRows  int
	invoices  []models.Invoice
	hookErr   error
	hookResp  interface{}
	sent      []models.Invoice
	saved     map[uuid.UUID]string
	loadCalls int
}

func stubAPFabResend(t *testing.T, s *apFabResendStub) {
	t.Helper()
	prev := []interface{}{apFabGetHookConfig, apFabLoadInvoices, apFabGetSuppliers, apFabGetProducts, apFabHookInterface, apFabSaveExternalID}
	s.saved = map[uuid.UUID]string{}
	apFabGetHookConfig = func(_ context.Context, _ map[string]interface{}) ([]interfaceService.HookConfig, error) {
		rows := []interfaceService.HookConfig{}
		for i := 0; i < s.hookRows; i++ {
			rows = append(rows, interfaceService.HookConfig{HookUrl: "http://doc/interface/invoice-ap-create"})
		}
		return rows, nil
	}
	apFabLoadInvoices = func(ids []uuid.UUID) ([]models.Invoice, error) {
		s.loadCalls++
		return s.invoices, nil
	}
	apFabGetSuppliers = func(_ context.Context, _ models.GetSupplierListRequest) (map[string]models.Supplier, error) {
		return map[string]models.Supplier{"D-002": {SupplierCode: "D-002", ExternalID: "TR-SUP-9"}}, nil
	}
	apFabGetProducts = func(_ context.Context, _ models.GetProductRequest) (map[string]models.GetProductsDetailComponent, error) {
		return map[string]models.GetProductsDetailComponent{"SVC": {ProductCode: "SVC", ProductName: "ค่าบริการ"}}, nil
	}
	apFabHookInterface = func(_ context.Context, req interfaceService.HookInterfaceRequest) (interface{}, error) {
		s.sent = append(s.sent, req.RequestData.([]models.Invoice)...)
		if s.hookErr != nil {
			return nil, s.hookErr
		}
		return s.hookResp, nil
	}
	apFabSaveExternalID = func(id uuid.UUID, externalID string) error {
		s.saved[id] = externalID
		return nil
	}
	t.Cleanup(func() {
		apFabGetHookConfig = prev[0].(func(context.Context, map[string]interface{}) ([]interfaceService.HookConfig, error))
		apFabLoadInvoices = prev[1].(func([]uuid.UUID) ([]models.Invoice, error))
		apFabGetSuppliers = prev[2].(func(context.Context, models.GetSupplierListRequest) (map[string]models.Supplier, error))
		apFabGetProducts = prev[3].(func(context.Context, models.GetProductRequest) (map[string]models.GetProductsDetailComponent, error))
		apFabHookInterface = prev[4].(func(context.Context, interfaceService.HookInterfaceRequest) (interface{}, error))
		apFabSaveExternalID = prev[5].(func(uuid.UUID, string) error)
	})
}

func unsentGRA(id uuid.UUID, externalID string) models.Invoice {
	return models.Invoice{
		ID: id, InvoiceCode: "GRA2611-00010", InvoiceType: "AP-FAB", DocumentRefType: "FABRICATION",
		DocumentRef: "FAB2611-00005", PartyCode: "D-002", CompanyCode: "TMI", SiteCode: "TMI", ExternalID: externalID,
		InvoiceItem: []models.InvoiceItem{{ProductCode: "FG-1", DocumentRef: "FAB2611-00005", DocumentRefItem: "IOP-001", PriceUnit: 0.62, Weight: 30681.748, UnitUom: "KG"}},
	}
}

func TestResendUnsentAPFab(t *testing.T) {
	ctx := context.Background()

	t.Run("ไม่มีใบค้าง = ไม่ทำอะไร", func(t *testing.T) {
		s := &apFabResendStub{hookRows: 1}
		stubAPFabResend(t, s)
		if err := resendUnsentAPFab(ctx, nil); err != nil {
			t.Fatal(err)
		}
		if s.loadCalls != 0 || len(s.sent) != 0 {
			t.Fatalf("must not load/send, got %d/%d", s.loadCalls, len(s.sent))
		}
	})

	t.Run("ไม่มี hook_config (ไม่มี TRCloud) = external_id ว่างเป็นเรื่องปกติ ไม่ส่ง", func(t *testing.T) {
		id := uuid.New()
		s := &apFabResendStub{hookRows: 0, invoices: []models.Invoice{unsentGRA(id, "")}}
		stubAPFabResend(t, s)
		if err := resendUnsentAPFab(ctx, map[uuid.UUID]models.Invoice{id: unsentGRA(id, "")}); err != nil {
			t.Fatal(err)
		}
		if len(s.sent) != 0 {
			t.Fatalf("must not send, got %d", len(s.sent))
		}
	})

	t.Run("ส่ง GRA ใบเดิมซ้ำ เลขเดิม + supplier external id + สินค้าค่าบริการ แล้วเก็บ external_id", func(t *testing.T) {
		id := uuid.New()
		s := &apFabResendStub{hookRows: 1, invoices: []models.Invoice{unsentGRA(id, "")}, hookResp: map[string]interface{}{"id": "AP-777"}}
		stubAPFabResend(t, s)
		if err := resendUnsentAPFab(ctx, map[uuid.UUID]models.Invoice{id: unsentGRA(id, "")}); err != nil {
			t.Fatal(err)
		}
		if len(s.sent) != 1 {
			t.Fatalf("want 1 send, got %d", len(s.sent))
		}
		inv := s.sent[0]
		if inv.InvoiceCode != "GRA2611-00010" || inv.PartyExternalID != "TR-SUP-9" {
			t.Fatalf("bad header: %+v", inv)
		}
		it := inv.InvoiceItem[0]
		if it.ProductCode != "SVC" || it.ProductName != "ค่าบริการ" || it.PriceUnit != 0.62 || it.UnitUom != "KG" {
			t.Fatalf("bad item: %+v", it)
		}
		if s.saved[id] != "AP-777" {
			t.Fatalf("external_id not saved: %v", s.saved)
		}
	})

	t.Run("ใบที่มี external_id แล้ว (มีคนส่งก่อน) ไม่ส่งซ้ำ", func(t *testing.T) {
		id := uuid.New()
		s := &apFabResendStub{hookRows: 1, invoices: []models.Invoice{unsentGRA(id, "AP-1")}}
		stubAPFabResend(t, s)
		if err := resendUnsentAPFab(ctx, map[uuid.UUID]models.Invoice{id: unsentGRA(id, "")}); err != nil {
			t.Fatal(err)
		}
		if len(s.sent) != 0 {
			t.Fatalf("must not resend, got %d", len(s.sent))
		}
	})

	t.Run("TRCloud ล้มอีก = error (production-core ค้าง AP pending แล้ว retry ได้)", func(t *testing.T) {
		id := uuid.New()
		s := &apFabResendStub{hookRows: 1, invoices: []models.Invoice{unsentGRA(id, "")}, hookErr: errors.New("upstream down")}
		stubAPFabResend(t, s)
		err := resendUnsentAPFab(ctx, map[uuid.UUID]models.Invoice{id: unsentGRA(id, "")})
		if err == nil || !strings.Contains(err.Error(), "GRA2611-00010") {
			t.Fatalf("want error naming the GRA, got %v", err)
		}
		if len(s.saved) != 0 {
			t.Fatalf("must not save external_id, got %v", s.saved)
		}
	})

	t.Run("TRCloud ไม่คืน id = error ไม่เก็บ", func(t *testing.T) {
		id := uuid.New()
		s := &apFabResendStub{hookRows: 1, invoices: []models.Invoice{unsentGRA(id, "")}, hookResp: "200 Success"}
		stubAPFabResend(t, s)
		if err := resendUnsentAPFab(ctx, map[uuid.UUID]models.Invoice{id: unsentGRA(id, "")}); err == nil {
			t.Fatal("want error")
		}
		if len(s.saved) != 0 {
			t.Fatalf("must not save external_id, got %v", s.saved)
		}
	})
}

func TestRound3(t *testing.T) {
	if got := round3(306.81748); got != 306.817 {
		t.Fatalf("round3 = %v", got)
	}
	if got := round2(306.81748); got != 306.82 {
		t.Fatalf("round2 must stay 2 digits for non AP-FAB, got %v", got)
	}
}
