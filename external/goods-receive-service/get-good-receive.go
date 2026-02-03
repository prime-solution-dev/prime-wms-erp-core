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

type GoodsReceiveFilter struct {
	ID              []uuid.UUID `json:"id"`
	NotReceiveCode  []string    `json:"not_receive_code"`
	ReceiveCode     []string    `json:"receive_code"`
	PlanMethod      []string    `json:"plan_method"`
	PlanType        []string    `json:"plan_type"`
	ReferenceNo     []string    `json:"reference_no"`
	ReferenceType   []string    `json:"reference_type"`
	SupplierCode    []string    `json:"supplier_code"`
	SupplierName    []string    `json:"supplier_name"`
	InvoiceNo       []string    `json:"invoice_no"`
	PlanQty         float64     `json:"plan_qty"`
	DocumentRefItem []string    `json:"document_ref_item"`
	PlanQtyOperator string      `json:"plan_quantity_operator"`
	Worker          []string    `json:"worker"`
	StarPlanDate    *time.Time  `json:"star_plan_date"`
	EndPlanDate     *time.Time  `json:"end_plan_date"`
	StarReceiveDate *time.Time  `json:"star_receive_date"`
	EndReceiveDate  *time.Time  `json:"end_receive_date"`
	Status          []string    `json:"status"`
	DateFrom        *time.Time  `json:"date_from"`
	DateTo          *time.Time  `json:"date_to"`
	Page            int         `json:"page"`
	PageSize        int         `json:"page_size"`
}

type GoddsReceiveResult struct {
	Total        int            `json:"total"`
	Page         int            `json:"page"`
	PageSize     int            `json:"page_size"`
	TotalPages   int            `json:"total_pages"`
	GoodsReceive []GoodsReceive `json:"goods_receive"`
}
type GoodsReceive struct {
	Action                string                `json:"action"`
	ReceiveID             uuid.UUID             `json:"receive_id"`
	ReceiveCode           string                `json:"receive_code"`
	ReceiveType           string                `json:"receive_type"`
	SiteCode              string                `json:"site_code"`
	WarehouseCode         string                `json:"warehouse_code"`
	IsApproved            bool                  `json:"is_approved"`
	DocumentRefType       string                `json:"document_ref_type"`
	DocumentRef           string                `json:"document_ref"`
	Status                string                `json:"status"`
	Remark                string                `json:"remark"`
	SubmitDate            *time.Time            `json:"submit_date"`
	DocumentToken         string                `json:"document_token"`
	CompanyCode           string                `json:"company_code"`
	Invoice               string                `json:"invoice"`
	PartnerCode           string                `json:"partner_code"`
	PartnerName           string                `json:"partner_name"`
	CreateDtm             string                `json:"create_dtm"`
	CreateBy              string                `json:"create_by"`
	UpdateDtm             string                `json:"update_dtm"`
	UpdateBy              string                `json:"update_by"`
	CustomDocNo           string                `json:"custom_doc_no"`
	SupllierCode          string                `json:"supllier_code"`
	SupplierName          string                `json:"supplier_name"`
	ReceiveFunction       string                `json:"receive_function"`
	Assignee              uuid.UUID             `json:"assignee"`
	PlanQTY               float64               `json:"plan_qty"`
	PlanDate              *time.Time            `json:"plan_date"`
	Unit                  Unit                  `json:"unit"`
	EstimateDate          *time.Time            `json:"estimate_date"`
	PartialPutaWay        bool                  `json:"partial_putaway"`
	ReceiveingDock        string                `json:"receiveing_dock"`
	DocEntry              int                   `json:"doc_entry"`
	FlgInterface          *bool                 `json:"flg_interface"`
	ErrorInterfaceMeassge string                `json:"error_interface_meassge"`
	IsKernal              bool                  `json:"is_kernal"`
	IsDefaultReceive      bool                  `json:"is_default_receive"`
	GoodsReceiveReserve   []GoodsReceiveReserve `json:"receive_reserve"`
	GoodsReceiveItem      []GoodsReceiveItem    `json:"goods_receive_item"`
}

type GoodsReceiveReserve struct {
	ID           uuid.UUID `json:"id"`
	ReceiveID    uuid.UUID `json:"receive_id"`
	ReserveType  string    `json:"reserve_type"`
	ReserveValue string    `json:"reserve_value"`
	ProductCode  string    `json:"product_code"`
	Qty          float64   `json:"qty"`
	UnitCode     string    `json:"unit_code"`
}
type GoodsReceiveItem struct {
	Action                    string                      `json:"action"`
	ReceiveItemID             uuid.UUID                   `json:"receive_item_id"`
	ReceiveCode               string                      `json:"receive_code"`
	ReceiveID                 uuid.UUID                   `json:"receive_id"`
	ReceiveItem               string                      `json:"receive_item"`
	DocumentRefItem           string                      `json:"document_ref_item"`
	ZoneCode                  string                      `json:"zone_code"`
	LocationCode              string                      `json:"location_code"`
	ProductCode               string                      `json:"product_code"`
	Qty                       float64                     `json:"qty"`
	TotalWeight               float64                     `json:"total_weight"`
	WeightUnit                string                      `json:"weight_unit"`
	ActualQty                 float64                     `json:"actual_qty"`
	UnitCode                  string                      `json:"unit_code"`
	BaseQty                   float64                     `json:"base_qty"`
	BaseUnitCode              string                      `json:"base_unit_code"`
	BaseConfirmQty            float64                     `json:"base_confirm_qty"`
	BaseConfirmQtyUnitCode    string                      `json:"base_confirm_unit_code"`
	ConfirmQty                float64                     `json:"confirm_qty"`
	ConfirmUnitCode           string                      `json:"confirm_unit_code"`
	Remark                    string                      `json:"remark"`
	Status                    string                      `json:"status"`
	SubmitDate                *time.Time                  `json:"submit_date"`
	InboundItemDocRef         string                      `json:"inbound_item_doc_ref"`
	InboundItemDocRefItem     string                      `json:"inbound_item_doc_ref_item"`
	ProductName               string                      `json:"product_name"`
	InboundQty                float64                     `json:"inbound_qty"`
	InboundUnitCode           string                      `json:"inbound_unit_code"`
	TotalConfirmQTY           string                      `json:"total_confirm_qty"`
	Label                     string                      `json:"label"`
	ConfirmItem               string                      `json:"confirm_item"`
	MfgDate                   *time.Time                  `json:"mfg_date"`
	ExpiryDate                *time.Time                  `json:"expiry_date"`
	Pallet                    string                      `json:"pallet"`
	Container                 string                      `json:"container"`
	IsForceBatch              bool                        `json:"is_force_batch"`
	IsForceSerial             bool                        `json:"is_force_serial"`
	BatchNO                   string                      `json:"batch_no"`
	RemainQTY                 float64                     `json:"remain_plan_qty"`
	UpdateDTM                 time.Time                   `json:"update_dtm"`
	LineProduct               int                         `json:"line_product"`
	WarehouseCode             string                      `gorm:"type:varchar(50)" json:"warehouse_code"`
	ReceiveDate               *time.Time                  `json:"receive_date"`
	GoodsReceiveConfirm       []GoodsReceiveConfirm       `json:"goods_receive_confirm"`
	GoodsReceiveSuggestBatch  []GoodsReceiveSuggestBatch  `json:"goods_receive_suggest_batch"`
	GoodsReceiveSuggestSerial []GoodsReceiveSuggestSerial `json:"goods_receive_suggest_serial"`
	InboundBatchRes           string                      `json:"inbound_batchs"`
}
type GoodsReceiveSuggestBatch struct {
	BatchNo      string     `json:"batch_no"`
	Qty          float64    `json:"qty"`
	UnitCode     string     `json:"unit_code"`
	BaseQty      float64    `json:"base_qty"`
	BaseUnitCode string     `json:"base_unit_code"`
	MfgDate      *time.Time `json:"mfgDate"`
	ExpiryDate   *time.Time `json:"expiryDate"`
}

type GoodsReceiveSuggestSerial struct {
	SerialNo     string     `json:"serial_code"`
	Qty          float64    `json:"qty"`
	UnitCode     string     `json:"unit_code"`
	BaseQty      float64    `json:"base_qty"`
	BaseUnitCode string     `json:"base_unit_code"`
	BatchNo      string     `json:"batch_no"`
	MfgDate      *time.Time `json:"mfgDate"`
	ExpiryDate   *time.Time `json:"expiryDate"`
}

type GoodsReceiveConfirm struct {
	Action                    string                      `json:"action"`
	ReceiveConfirmID          uuid.UUID                   `json:"receive_confirm_id"`
	ReceiveItemID             uuid.UUID                   `json:"receive_item_id"`
	ZoneCode                  string                      `json:"zone_code"`
	LocationCode              string                      `json:"location_code"`
	PalletCode                string                      `json:"pallet_code"`
	ContainerCode             string                      `json:"container_code"`
	Qty                       float64                     `json:"qty"`
	TotalWeight               float64                     `json:"total_weight"`
	WeightUnit                string                      `json:"weight_unit"`
	UnitCode                  string                      `json:"unit_code"`
	BaseQty                   float64                     `json:"base_qty"`
	BaseUnitCode              string                      `json:"base_unit_code"`
	Remark                    string                      `json:"remark"`
	Status                    string                      `json:"status"`
	MfgDate                   *time.Time                  `json:"mfg_date"`
	ExpiryDate                *time.Time                  `json:"expiry_date"`
	DamageReasonCode          string                      `json:"damage_reason_code"`
	BatchNo                   string                      `json:"batch_no"`
	IsSplitDupUquit           bool                        `json:"is_split_dup_uquit"`
	IsComplete                bool                        `json:"is_complete"`
	GoodsReceiveConfirmSerial []GoodsReceiveConfirmSerial `json:"goods_receive_confirm_serial"`
}

type GoodsReceiveConfirmSerial struct {
	Action           string     `json:"action"`
	ReceiveConfirmID uuid.UUID  `json:"receive_confirm_id"`
	SerialCode       string     `json:"serial_code"`
	Qty              float64    `json:"qty"`
	UnitCode         string     `json:"unit_code"`
	MfgDate          *time.Time `json:"mfg_date"`
	ExpiryDate       *time.Time `json:"expiry_date"`
}

func GetGoodsReceives(jsonPayload GoodsReceiveFilter) (GoddsReceiveResult, error) {
	jsonData, err := json.Marshal(jsonPayload)
	if err != nil {
		return GoddsReceiveResult{}, errors.New("Error marshaling struct to JSON:" + err.Error())
	}

	req, err := http.NewRequest("POST", config.GET_INBOUND_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		return GoddsReceiveResult{}, errors.New("Error parsing DateTo: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return GoddsReceiveResult{}, errors.New("Error parsing DateTo: " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	var dataRes GoddsReceiveResult
	err = json.Unmarshal(body, &dataRes)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	return dataRes, nil
}
