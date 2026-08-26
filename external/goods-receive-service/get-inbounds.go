package goodsReceiveService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"prime-erp-core/config"
	"time"

	"github.com/google/uuid"
)

type InboundFilter struct {
	ID                         []uuid.UUID `json:"id"`
	InboundCode                []string    `json:"inbound_code"`
	InterfaceRef               []string    `json:"interface_ref"`
	Status                     []string    `json:"status"`
	DocumentRef                []string    `json:"invoice_no"`
	SupplierCode               []string    `json:"supplier_code"`
	SupplierName               []string    `json:"supplier_name"`
	Page                       int         `json:"page"`
	PageSize                   int         `json:"page_size"`
	StarEstimateGrDate         *time.Time  `json:"star_estimate_gr_date"`
	EndEstimateGrDate          *time.Time  `json:"end_estimate_gr_date"`
	NotProductCode             []string    `json:"not_product_code"`
	InboundType                []string    `json:"inbound_type"`
	ReceiveFunction            []string    `json:"receive_function"`
	PlanQty                    float64     `json:"plan_quantity"`
	PlanQtyOperator            string      `json:"plan_quantity_operator"`
	IsRemainQTY                bool        `json:"is_remain_qty"`
	DocumentRefType            []string    `json:"document_ref_type"`
	InboundItemDocumentRef     []string    `json:"inbound_item_document_ref"`
	InboundItemDocumentRefItem []string    `json:"inbound_item_document_ref_item"`
	// LIKE search fields
	PlanCodeLike     string `json:"plan_code_like"`
	SupplierCodeLike string `json:"supplier_code_like"`
	SupplierNameLike string `json:"supplier_name_like"`
	InvoiceNoLike    string `json:"invoice_no_like"`
}

type ResultInbound struct {
	Total      int          `json:"total"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
	TotalPages int          `json:"total_pages"`
	InboundRes []InboundRes `json:"inbound"`
}

type InboundRes struct {
	ID              *uuid.UUID       `json:"id"`
	InboundCode     string           `json:"inbound_code"`
	TenantID        *uuid.UUID       `json:"tenant_id"`
	WarehouseCode   string           `json:"warehouse_code"`
	IsApproved      bool             `json:"is_approved"`
	Status          string           `json:"status"`
	Remark          string           `json:"remark"`
	DocumentRefType string           `json:"document_ref_type"`
	DocumentRef     string           `json:"document_ref"`
	Partner         int              `json:"partner"`
	PartnerCode     string           `json:"partner_code"`
	PartnerName     string           `json:"partner_name"`
	SubmitDate      time.Time        `json:"submit_date"`
	PartialGR       bool             `json:"partial_gr"`
	SiteCode        string           `json:"site_code"`
	CompanyCode     string           `json:"company_code"`
	InboundType     string           `json:"inbound_type"`
	ReceiveFunction string           `json:"inbound_function"`
	EstimateGrDate  *time.Time       `json:"estimate_gr_date"`
	PlanQty         float64          `json:"plan_qty"`
	ReturnReason    string           `json:"return_reason"`
	InterfaceRef    string           `json:"interface_ref"`
	CreateBy        *string          `json:"create_by"`
	CreateDTM       *time.Time       `json:"create_dtm"`
	UpdateBy        *string          `json:"update_by"`
	UpdateDTM       *time.Time       `json:"update_dtm"`
	Unit            Unit             `json:"unit"`
	ReceiveingDock  string           `json:"receiveing_dock"`
	DocumentRef2    string           `json:"document_ref2"`
	DocEntry        int              `json:"doc_entry"`
	InboundItemRes  []InboundItemRes `json:"inbound_items"`
}

type Unit struct {
	UnitCode string  `json:"unit_code"`
	UnitName string  `json:"unit_name"`
	Qty      float64 `json:"qty"`
}

type InboundItemRes struct {
	ID              *uuid.UUID          `json:"id"`
	InboundID       *uuid.UUID          `json:"inbound_id"`
	InboundItem     string              `json:"inbound_item"`
	DocumentRefType string              `json:"document_ref_type"`
	DocumentRef     string              `json:"document_ref"`
	DocumentRefItem string              `json:"document_ref_item"`
	WarehouseCode   string              `json:"warehouse_code"`
	ProductCode     string              `json:"product_code"`
	ProductName     string              `json:"product_name"`
	Qty             float64             `json:"qty"`
	RemainQTY       float64             `json:"remain_plan_qty"`
	UnitCode        string              `json:"unit_code"`
	Remark          string              `json:"remark"`
	MfgDate         *time.Time          `json:"mfg_date"`
	ExpDate         *time.Time          `json:"exp_date"`
	BatchNo         string              `json:"batch_no"`
	DocumentRef2    string              `json:"document_ref2"`
	PalletNo        string              `json:"pallet_no"`
	CartonNo        string              `json:"carton_no"`
	LineProduct     int                 `json:"line_product"`
	TotalWeight     float64             `json:"total_weight"`
	WeightUnit      string              `json:"weight_unit"`
	SerialNo        []InboundItemSerial `json:"serial_no"`
}

type InboundItemSerial struct {
	Action        string     `json:"action"`
	InboundItemID uuid.UUID  `json:"inbound_item_id"`
	SerialCode    string     `json:"serial_code"`
	Qty           float64    `json:"qty"`
	UnitCode      string     `json:"unit_code"`
	MfgDate       *time.Time `json:"mfg_date"`
	ExpiryDate    *time.Time `json:"expiry_date"`
}

func GetInbounds(jsonPayload InboundFilter) (ResultInbound, error) {

	jsonData, err := json.Marshal(jsonPayload)
	if err != nil {
		return ResultInbound{}, errors.New("Error marshaling struct to JSON: " + err.Error())
	}

	req, err := http.NewRequest("POST", config.GET_INBOUND_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		return ResultInbound{}, errors.New("Error creating request: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	// timeout กันปลายทางค้างแล้วลาก request ของเราค้างตาม (default ของ http.Client คือไม่มี timeout)
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ResultInbound{}, errors.New("Error sending request: " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ResultInbound{}, errors.New("Error reading response body: " + err.Error())
	}

	// เดิมข้ามการเช็ค status ทำให้ปลายทางตอบ 500 แล้วเราคืน struct เปล่ากับ error = nil
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			Error string `json:"error"`
		}
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Error != "" {
			return ResultInbound{}, errors.New(errResp.Error)
		}
		return ResultInbound{}, fmt.Errorf("API error status %d: %s", resp.StatusCode, string(body))
	}

	var dataRes ResultInbound
	if err = json.Unmarshal(body, &dataRes); err != nil {
		return ResultInbound{}, errors.New("Error parsing response: " + err.Error())
	}

	return dataRes, nil
}
