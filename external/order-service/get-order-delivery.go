package externalService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"prime-erp-core/config"

	"github.com/google/uuid"
)

type GetOrderDeliveryRequest struct {
	DeliveryCode []string `json:"delivery_code"`
	DeliveryItem []string `json:"delivery_item"`
}

type GetOrderDeliveryResponse struct {
	ID                  uuid.UUID                      `gorm:"type:uuid;primary_key" json:"id"`
	OrderCode           string                         `gorm:"type:varchar(50)" json:"order_code"`
	OrderType           string                         `gorm:"type:varchar(50)" json:"order_type"`
	OrderDate           *time.Time                     `gorm:"type:timestamp" json:"order_date"`
	TenantID            uuid.UUID                      `gorm:"type:uuid" json:"tenant_id"`
	CustomerCode        string                         `gorm:"type:varchar(50)" json:"customer_code"`
	CustomerName        string                         `gorm:"type:varchar(50)" json:"customer_name"`
	SoldToCode          string                         `gorm:"type:varchar(50)" json:"sold_to_code"`
	ShipToCode          string                         `gorm:"type:varchar(50)" json:"ship_to_code"`
	BillToCode          string                         `gorm:"type:varchar(50)" json:"bill_to_code"`
	TransportZone       string                         `gorm:"type:varchar" json:"transport_zone"`
	InterfaceQty        float64                        `gorm:"type:decimal(10,4)" json:"interface_qty"`
	InterfaceUnitCode   string                         `gorm:"type:varchar(50)" json:"interface_unit_code"`
	Qty                 float64                        `gorm:"type:decimal(10,4)" json:"qty"`
	UnitCode            string                         `gorm:"type:varchar(50)" json:"unit_code"`
	EstimatePickingDate *time.Time                     `gorm:"type:timestamp" json:"estimate_picking_date"`
	DeliveryDate        *time.Time                     `gorm:"type:timestamp" json:"delivery_date"`
	SubmitDate          *time.Time                     `gorm:"type:timestamp" json:"submit_date"`
	Status              string                         `gorm:"type:varchar(20)" json:"status"`
	DocumentRefType     string                         `gorm:"type:varchar(10)" json:"document_ref_type"`
	DocumentRef         string                         `gorm:"type:varchar(50)" json:"document_ref"`
	Remark              string                         `gorm:"type:varchar" json:"remark"`
	CompanyCode         string                         `gorm:"type:varchar(50)" json:"company_code"`
	SiteCode            string                         `gorm:"type:varchar(50)" json:"site_code"`
	DocumentRef2        string                         `gorm:"type:varchar(50)" json:"document_ref_2"`
	DocumentRefType2    string                         `gorm:"type:varchar(10)" json:"document_ref_type_2"`
	PartyCode           string                         `gorm:"type:varchar(50)" json:"party_code"`
	PartyName           string                         `gorm:"type:varchar(100)" json:"party_name"`
	PartyType           string                         `gorm:"type:varchar(20)" json:"party_type"`
	Reason              string                         `gorm:"type:varchar(100)" json:"reason"`
	ShippingAddress     string                         `gorm:"type:varchar(200)" json:"shipping_address"`
	DeliveryMethod      string                         `gorm:"type:varchar(50)" json:"delivery_method"`
	BookingDate         *time.Time                     `gorm:"type:timestamp" json:"booking_date"`
	DeliveryTimeCode    string                         `gorm:"type:varchar(50)" json:"delivery_time_code"`
	Tel                 string                         `gorm:"type:varchar(20)" json:"tel"`
	LicensePlate        string                         `gorm:"type:varchar(20)" json:"license_plate"`
	ContactName         string                         `gorm:"type:varchar(100)" json:"contact_name"`
	CreateBy            string                         `gorm:"type:varchar(100)" json:"create_by"`
	CreateDtm           time.Time                      `gorm:"type:timestamp" json:"create_dtm"`
	UpdateBy            string                         `gorm:"type:varchar(100)" json:"update_by"`
	UpdateDtm           time.Time                      `gorm:"type:timestamp" json:"update_dtm"`
	OrderItem           []GetOrderItemDeliveryResponse `gorm:"foreignKey:OrderID;references:ID" json:"order_item"`
}

type GetOrderItemDeliveryResponse struct {
	ID                   uuid.UUID                    `gorm:"type:uuid;primary_key" json:"id"`
	OrderID              uuid.UUID                    `gorm:"type:uuid" json:"order_id"`
	OrderItem            string                       `gorm:"type:varchar(50)" json:"order_item"`
	DocumentRefItem      string                       `gorm:"type:varchar(10)" json:"document_ref_item"`
	ProductCode          string                       `gorm:"type:varchar(50)" json:"product_code"`
	ProductType          string                       `gorm:"type:varchar(10)" json:"product_type"`
	ProductName          string                       `gorm:"-" json:"product_name"`
	InterfaceOrderQty    float64                      `gorm:"type:decimal(10,4)" json:"interface_order_qty"`
	Qty                  float64                      `gorm:"type:decimal(10,4)" json:"qty"`
	UnitCode             string                       `gorm:"type:varchar(50)" json:"unit_code"`
	IsFocGwp             bool                         `gorm:"type:boolean" json:"is_foc_gwp"`
	WarehouseCode        string                       `gorm:"type:varchar(50)" json:"warehouse_code"`
	BatchNo              string                       `gorm:"type:varchar(50)" json:"batch_no"`
	SerialCode           string                       `gorm:"type:varchar(50)" json:"serial_code"`
	Remark               string                       `gorm:"type:varchar" json:"remark"`
	Status               string                       `gorm:"type:varchar(20)" json:"status"`
	SaleUnitCode         string                       `gorm:"type:varchar(50)" json:"sale_unit_code"`
	SaleMethod           string                       `gorm:"type:varchar(50)" json:"sale_method"`
	InterfaceOrderWeight float64                      `gorm:"type:decimal(10,4)" json:"interface_order_weight"`
	Weight               float64                      `gorm:"type:decimal(10,4)" json:"weight"`
	WeightUnit           float64                      `gorm:"type:decimal(10,4)" json:"weight_unit"`
	MfgDate              *time.Time                   `gorm:"type:timestamp" json:"mfg_date"`
	ExpDate              *time.Time                   `gorm:"type:timestamp" json:"exp_date"`
	LocationCode         string                       `gorm:"type:varchar(50)" json:"location_code"`
	StorageType          string                       `gorm:"type:varchar(50)" json:"storage_type"`
	CreateBy             string                       `gorm:"type:varchar(100)" json:"create_by"`
	CreateDtm            time.Time                    `gorm:"type:timestamp" json:"create_dtm"`
	UpdateBy             string                       `gorm:"type:varchar(100)" json:"update_by"`
	UpdateDtm            time.Time                    `gorm:"type:timestamp" json:"update_dtm"`
	OutboundItem         []OutboundItemWithGoodsIssue `gorm:"-" json:"outbound_item"`
}

type GetOutboundItemResponse struct {
	ID              uuid.UUID                    `gorm:"type:uuid;primary_key" json:"id"`
	OutboundId      uuid.UUID                    `gorm:"type:uuid" json:"outbound_id"`
	WarehouseCode   string                       `gorm:"type:varchar(50)" json:"warehouse_code"`
	OutboundItem    string                       `gorm:"type:varchar(50)" json:"outbound_item"`
	DocumentRef     string                       `gorm:"type:varchar(50)" json:"document_ref"`
	DocumentRefItem string                       `gorm:"type:varchar(50)" json:"document_ref_item"`
	ProductCode     string                       `gorm:"type:varchar(50)" json:"product_code"`
	Qty             float64                      `gorm:"type:numeric(10,4)" json:"qty"`
	UnitCode        string                       `gorm:"type:varchar(50)" json:"unit_code"`
	Status          string                       `gorm:"type:varchar(20)" json:"status"`
	CreateBy        string                       `gorm:"type:varchar(50)" json:"create_by"`
	CreateDtm       time.Time                    `gorm:"type:date" json:"create_dtm"`
	UpdateBy        string                       `gorm:"type:varchar(50)" json:"update_by"`
	UpdateDtm       time.Time                    `gorm:"type:date" json:"update_dtm"`
	ConfirmQty      float64                      `gorm:"type:numeric(10,4)" json:"confirm_qty"`
	ConfirmWeight   float64                      `gorm:"type:numeric(10,4)" json:"confirm_weight"`
	FlowTracking    []OutboundFlowTracking       `gorm:"foreignKey:OutboundItemId" json:"flow_tracking"`
	Processes       []GetOutboundProcessResponse `gorm:"foreignKey:OutboundItemId" json:"processes"`
}

type OutboundFlowTracking struct {
	ID             uuid.UUID `json:"id"`
	OutboundItemId uuid.UUID `json:"outbound_item_id"`
	ProcessCode    string    `json:"process_code"`
	Status         string    `json:"status"`
	CreateBy       string    `json:"create_by"`
	CreateDtm      time.Time `json:"create_dtm"`
}

type GetOutboundProcessResponse struct {
	ID                  uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	OutboundItemId      uuid.UUID `gorm:"type:uuid" json:"outbound_item_id"`
	OutboundProcessItem string    `gorm:"type:varchar(10)" json:"outbound_process_item"`
	DocumentRef         string    `gorm:"type:varchar(50)" json:"document_ref"`
	DocumentRefItem     string    `gorm:"type:varchar(50)" json:"document_ref_item"`
	Qty                 float64   `gorm:"type:numeric(10,4)" json:"qty"`
	UnitCode            string    `gorm:"type:varchar(50)" json:"unit_code"`
	BaseQty             float64   `gorm:"type:numeric(10,4)" json:"base_qty"`
	BaseUnitCode        string    `gorm:"type:varchar(50)" json:"base_unit_code"`
	Remark              string    `gorm:"type:varchar(50)" json:"remark"`
	IsSplitItem         bool      `gorm:"type:varchar(50)" json:"is_split_item"`
	SpliteItemRef       uuid.UUID `gorm:"type:uuid" json:"splite_item_ref"`
	Status              string    `gorm:"type:varchar(20)" json:"status"`
	CreateBy            string    `gorm:"type:varchar(50)" json:"create_by"`
	CreateDtm           time.Time `gorm:"type:date" json:"create_dtm"`
	UpdateBy            string    `gorm:"type:varchar(50)" json:"update_by"`
	UpdateDtm           time.Time `gorm:"type:date" json:"update_dtm"`
}

// Extended struct ที่รวม outbound item กับ goods issue item
type OutboundItemWithGoodsIssue struct {
	GetOutboundItemResponse
	GoodsIssueItem []GetIssueItemResponse `json:"goods_issue_item"`
}

type GetIssueItemResponse struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;not null" json:"id"`
	IssueID         uuid.UUID  `gorm:"type:uuid;not null" json:"issue_id"`
	IssueItem       string     `gorm:"type:varchar(50)" json:"issue_item"`
	DocumentRefItem string     `gorm:"type:varchar(50)" json:"document_ref_item"`
	WarehouseCode   string     `gorm:"type:varchar(50)" json:"warehouse_code"`
	LocationCode    string     `gorm:"type:varchar(50)" json:"location_code"`
	PalletCode      string     `gorm:"type:varchar(50)" json:"palet_code"`
	ContainerCode   string     `gorm:"type:varchar(50)" json:"container_code"`
	BatchCode       string     `gorm:"type:varchar(50)" json:"batch_code"`
	ExpDate         *time.Time `gorm:"type:timestamp with time zone" json:"exp_date"`
	MfgDate         *time.Time `gorm:"type:timestamp with time zone" json:"mfg_date"`
	ReceiveDate     *time.Time `gorm:"type:timestamp with time zone" json:"receive_date"`
	ProductCode     string     `gorm:"type:varchar(50)" json:"product_code"`
	Qty             float64    `gorm:"type:numeric(10,4)" json:"qty"`
	Weight          float64    `gorm:"type:numeric(10,4)" json:"weight"`
	UnitCode        string     `gorm:"type:varchar(50)" json:"unit_code"`
	Status          string     `gorm:"type:varchar(20)" json:"status"`
	CreateBy        string     `gorm:"type:varchar(100)" json:"create_by"`
	CreateDTM       time.Time  `gorm:"type:timestamp with time zone" json:"create_dtm"`
	UpdateBy        string     `gorm:"type:varchar(100)" json:"update_by"`
	UpdateDTM       time.Time  `gorm:"type:timestamp with time zone" json:"update_dtm" gorm-extension:""`
}

type ResultOrderDeliveryResponse struct {
	Status  string                     `json:"status"`
	Message string                     `json:"message"`
	Orders  []GetOrderDeliveryResponse `json:"orders"`
}

func GetOrdersDelivery(jsonPayload GetOrderDeliveryRequest) (ResultOrderDeliveryResponse, error) {

	jsonData, err := json.Marshal(jsonPayload)
	if err != nil {
		return ResultOrderDeliveryResponse{}, errors.New("Error marshaling struct to JSON:" + err.Error())
	}
	req, err := http.NewRequest("POST", config.GET_ORDER_DELIVERY_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		return ResultOrderDeliveryResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ResultOrderDeliveryResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	var dataRes ResultOrderDeliveryResponse
	err = json.Unmarshal(body, &dataRes)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	return dataRes, nil
}
