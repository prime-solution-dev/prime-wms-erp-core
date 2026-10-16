package deliveryService

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// บรรทัดใบจองต้องเก็บน้ำหนัก/จำนวน/ราคา/หน่วยขาย ของบรรทัดตัวเอง
// ATP (wms-warehouse-service) อ่าน delivery_booking_item.total_weight เป็นน้ำหนักที่จองไว้
func TestNewDeliveryItemFillsBookedFields(t *testing.T) {
	deliveryID := uuid.New()
	now := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)

	item := newDeliveryItem(deliveryID, 0, CreateDeliveryItemsRequest{
		ProductCode:     "PRZ50251.3",
		Qty:             600,
		UnitCode:        "PCS",
		Weight:          5280,
		WeightUnit:      8.8,
		DocumentRefItem: "SO-ITEM-1",
		SaleUnitCode:    "PC",
		PriceListUnit:   22.7,
	}, "tester", now)

	if item.TotalWeight != 5280 {
		t.Fatalf("total_weight = %v, want booked weight 5280 (not SO line weight)", item.TotalWeight)
	}
	if item.SaleQty != 600 {
		t.Fatalf("sale_qty = %v, want booked qty 600", item.SaleQty)
	}
	if item.PriceListUnit != 22.7 {
		t.Fatalf("price_list_unit = %v, want 22.7", item.PriceListUnit)
	}
	if item.SaleUnitCode != "PC" {
		t.Fatalf("sale_unit_code = %q, want PC", item.SaleUnitCode)
	}
	if item.Weight != 5280 || item.Qty != 600 || item.DeliveryID != deliveryID || item.Status != "PENDING" {
		t.Fatalf("existing fields changed: %+v", item)
	}
}

// total_weight ต้องเท่ากับ weight ของบรรทัดเสมอ แม้หน้าบ้านจะส่งค่าอื่นมา (เช่นน้ำหนักทั้งบรรทัด SO)
func TestNewDeliveryItemTotalWeightFollowsLineWeight(t *testing.T) {
	item := newDeliveryItem(uuid.New(), 1, CreateDeliveryItemsRequest{
		Qty:         30,
		Weight:      1260.004,
		TotalWeight: 99999,
		SaleQty:     30,
	}, "tester", time.Now())

	if item.TotalWeight != 1260 {
		t.Fatalf("total_weight = %v, want 1260 (rounded line weight)", item.TotalWeight)
	}
}

// sale_qty = qty ที่จองเสมอ ไม่ใช้ค่าที่หน้าบ้านส่ง (เคยส่ง sale_qty ทั้งบรรทัด SO มา)
func TestNewDeliveryItemSaleQtyIsBookedQty(t *testing.T) {
	item := newDeliveryItem(uuid.New(), 0, CreateDeliveryItemsRequest{Qty: 4, SaleQty: 100, Weight: 10}, "tester", time.Now())

	if item.SaleQty != 4 {
		t.Fatalf("sale_qty = %v, want booked qty 4", item.SaleQty)
	}
}
