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
	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

type GetPackingRequest struct {
	DeliveryCodes    []string `json:"delivery_codes"`
	ExcludedPackCode []string `json:"excluded_pack_code"`
	PackingCode      []string `json:"packing_code"`
	StatusPack       []string `json:"status_pack"`
	Page             int      `json:"page"`
	PageSize         int      `json:"page_size"`
}

type GetPackingResponse struct {
	ID              uuid.UUID                `gorm:"type:uuid;primary_key" json:"id"`
	PackingCode     string                   `gorm:"type:varchar(50)" json:"packing_code"`
	PackingType     string                   `gorm:"type:varchar(50)" json:"packing_type"`
	TenantId        uuid.UUID                `gorm:"type:uuid" json:"tenant_id"`
	CompanyCode     string                   `gorm:"type:varchar(50)" json:"company_code"`
	SiteCode        string                   `gorm:"type:varchar(50)" json:"site_code"`
	WarehouseCode   string                   `gorm:"type:varchar(50)" json:"warehouse_code"`
	DocumentRefType string                   `gorm:"type:varchar(50)" json:"document_ref_type"`
	DocumentRef     string                   `gorm:"type:varchar(50)" json:"document_ref"`
	Status          string                   `gorm:"type:varchar(50)" json:"status"`
	SubmitDate      *time.Time               `gorm:"type:timestamp" json:"submit_date"`
	CreateBy        string                   `gorm:"type:varchar(50)" json:"create_by"`
	CreateDtm       time.Time                `gorm:"type:timestamp" json:"create_dtm"`
	UpdateBy        string                   `gorm:"type:varchar(50)" json:"update_by"`
	UpdateDtm       time.Time                `gorm:"type:timestamp" json:"update_dtm"`
	PackingItem     []GetPackingItemResponse `gorm:"foreignKey:packing_id" json:"packing_item"`
}

type GetPackingItemResponse struct {
	ID                 uuid.UUID                       `gorm:"type:uuid;primary_key" json:"id"`
	PackingId          uuid.UUID                       `gorm:"type:uuid" json:"packing_id"`
	PackingItem        string                          `gorm:"type:varchar(50)" json:"packing_item"`
	DocumentRef        string                          `gorm:"type:varchar(50)" json:"document_ref"`
	DocumentRefItem    string                          `gorm:"type:varchar(50)" json:"document_ref_item"`
	PackNo             string                          `gorm:"type:varchar(50)" json:"pack_no"`
	ZoneCode           string                          `gorm:"type:varchar(50)" json:"zone_code"`
	LocationCode       string                          `gorm:"type:varchar(50)" json:"location_code"`
	PalletCode         string                          `gorm:"type:varchar(50)" json:"pallet_code"`
	ContainerCode      string                          `gorm:"type:varchar(50)" json:"container_code"`
	ProductCode        string                          `gorm:"type:varchar(50)" json:"product_code"`
	Qty                float64                         `gorm:"type:numeric(10,4)" json:"qty"`
	UnitCode           string                          `gorm:"type:varchar(50)" json:"unit_code"`
	BaseQty            float64                         `gorm:"type:numeric(10,4)" json:"base_qty"`
	BaseUnitCode       string                          `gorm:"type:varchar(50)" json:"base_unit_code"`
	Remark             string                          `gorm:"type:varchar(50)" json:"remark"`
	Status             string                          `gorm:"type:varchar(50)" json:"status"`
	CreateBy           string                          `gorm:"type:varchar(50)" json:"create_by"`
	CreateDtm          time.Time                       `gorm:"type:timestamp" json:"create_dtm"`
	UpdateBy           string                          `gorm:"type:varchar(50)" json:"update_by"`
	UpdateDtm          time.Time                       `gorm:"type:timestamp" json:"update_dtm"`
	PackingItemConfirm []GetPackingItemConfirmResponse `gorm:"foreignKey:packing_item_id" json:"confirm"`
	Outbound           *OutboundWithOrderData          `gorm:"-" json:"outbound,omitempty"`
}

type GetPackingItemConfirmResponse struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	PackageNo          string     `gorm:"type:varchar(50)" json:"package_no"`
	PackingItemId      uuid.UUID  `gorm:"type:uuid" json:"pack_item_id"`
	SrcZoneCode        string     `gorm:"type:varchar(50)" json:"src_zone_code"`
	SrcLocationCode    string     `gorm:"type:varchar(50)" json:"src_location_code"`
	SrcPalletCode      string     `gorm:"type:varchar(50)" json:"src_pallet_code"`
	SrcContainerCode   string     `gorm:"type:varchar(50)" json:"src_container_code"`
	DestWarehouse_Code string     `gorm:"type:varchar(50)" json:"dest_warehouse_code"`
	DestZoneCode       string     `gorm:"type:varchar(50)" json:"dest_zone_code"`
	DestLocationCode   string     `gorm:"type:varchar(50)" json:"dest_location_code"`
	DestPalletCode     string     `gorm:"type:varchar(50)" json:"dest_pallet_code"`
	DestContainerCode  string     `gorm:"type:varchar(50)" json:"dest_container_code"`
	Qty                float64    `gorm:"type:numeric(10,4)" json:"qty"`
	UnitCode           string     `gorm:"type:varchar(50)" json:"unit_code"`
	BaseQty            float64    `gorm:"type:numeric(10,4)" json:"base_qty"`
	BaseUnitCode       string     `gorm:"type:varchar(50)" json:"base_unit_code"`
	BatchCode          string     `gorm:"type:varchar(50)" json:"batch_code"`
	SerialCode         string     `gorm:"type:varchar(50)" json:"serial_code"`
	MfgDate            *time.Time `gorm:"type:timestamp" json:"mfg_date"`
	ExpDate            *time.Time `gorm:"type:timestamp" json:"exp_date"`
	ReceiveDate        *time.Time `gorm:"type:timestamp" json:"receive_date"`
	Weight             float64    `gorm:"type:numeric(10,4)" json:"weight"`
	CreateBy           string     `gorm:"type:varchar(50)" json:"create_by"`
	CreateDtm          time.Time  `gorm:"type:timestamp" json:"create_dtm"`
}

type OutboundWithOrderData struct {
	ID              string                      `json:"id"`
	OutboundCode    string                      `json:"outbound_code"`
	TenantId        string                      `json:"tenant_id"`
	OutboundType    string                      `json:"outbound_type"`
	CompanyCode     string                      `json:"company_code"`
	SiteCode        string                      `json:"site_code"`
	DocumentRefType string                      `json:"document_ref_type"`
	DocumentRef     string                      `json:"document_ref"`
	IsApproved      bool                        `json:"is_approved"`
	Status          string                      `json:"Status"`
	EstimateGiDate  *time.Time                  `json:"estimate_gi_date"`
	EstimateGiTime  string                      `json:"estimate_gi_time"`
	SubmitDate      *time.Time                  `json:"submit_date"`
	DocumentToken   string                      `json:"document_token"`
	Remark          string                      `json:"remark"`
	CreateBy        string                      `json:"create_by"`
	CreateDtm       string                      `json:"create_dtm"`
	UpdateBy        string                      `json:"update_by"`
	UpdateDtm       string                      `json:"update_dtm"`
	Flows           []interface{}               `json:"flows"`
	Items           []OutboundItemWithOrderData `json:"items"`
}
type OutboundItemWithOrderData struct {
	ID              string             `json:"id"`
	OutboundId      string             `json:"outbound_id"`
	WarehouseCode   string             `json:"warehouse_code"`
	OutboundItem    string             `json:"outbound_item"`
	DocumentRef     string             `json:"document_ref"`
	DocumentRefItem string             `json:"document_ref_item"`
	ProductCode     string             `json:"product_code"`
	Qty             float64            `json:"qty"`
	UnitCode        string             `json:"unit_code"`
	Status          string             `json:"status"`
	CreateBy        string             `json:"create_by"`
	CreateDtm       string             `json:"create_dtm"`
	UpdateBy        string             `json:"update_by"`
	UpdateDtm       string             `json:"update_dtm"`
	ConfirmQty      float64            `json:"confirm_qty"`
	Processes       []interface{}      `json:"processes"`
	OrderData       GetOrderSOResponse `json:"order_data,omitempty"`
}

type GetOrderSOResponse struct {
	ID                  uuid.UUID                `gorm:"type:uuid;primary_key" json:"id"`
	OrderCode           string                   `gorm:"type:varchar(50)" json:"order_code"`
	OrderType           string                   `gorm:"type:varchar(50)" json:"order_type"`
	OrderDate           *time.Time               `gorm:"type:timestamp" json:"order_date"`
	TenantID            uuid.UUID                `gorm:"type:uuid" json:"tenant_id"`
	CustomerCode        string                   `gorm:"type:varchar(50)" json:"customer_code"`
	CustomerName        string                   `gorm:"type:varchar(50)" json:"customer_name"`
	SoldToCode          string                   `gorm:"type:varchar(50)" json:"sold_to_code"`
	ShipToCode          string                   `gorm:"type:varchar(50)" json:"ship_to_code"`
	BillToCode          string                   `gorm:"type:varchar(50)" json:"bill_to_code"`
	TransportZone       string                   `gorm:"type:varchar" json:"transport_zone"`
	InterfaceQty        float64                  `gorm:"type:decimal(10,4)" json:"interface_qty"`
	InterfaceUnitCode   string                   `gorm:"type:varchar(50)" json:"interface_unit_code"`
	Qty                 float64                  `gorm:"type:decimal(10,4)" json:"qty"`
	UnitCode            string                   `gorm:"type:varchar(50)" json:"unit_code"`
	EstimatePickingDate *time.Time               `gorm:"type:timestamp" json:"estimate_picking_date"`
	DeliveryDate        *time.Time               `gorm:"type:timestamp" json:"delivery_date"`
	SubmitDate          *time.Time               `gorm:"type:timestamp" json:"submit_date"`
	Status              string                   `gorm:"type:varchar(20)" json:"status"`
	DocumentRefType     string                   `gorm:"type:varchar(10)" json:"document_ref_type"`
	DocumentRef         string                   `gorm:"type:varchar(50)" json:"document_ref"`
	Remark              string                   `gorm:"type:varchar" json:"remark"`
	CreateBy            string                   `gorm:"type:varchar(100)" json:"create_by"`
	CreateDtm           time.Time                `gorm:"type:timestamp" json:"create_dtm"`
	UpdateBy            string                   `gorm:"type:varchar(100)" json:"update_by"`
	UpdateDtm           time.Time                `gorm:"type:timestamp" json:"update_dtm"`
	OrderItem           []GetOrderItemSOResponse `gorm:"foreignKey:OrderID;references:ID" json:"order_item"`
}

type GetOrderItemSOResponse struct {
	ID                   uuid.UUID           `gorm:"type:uuid;primary_key" json:"id"`
	OrderID              uuid.UUID           `gorm:"type:uuid" json:"order_id"`
	OrderItem            string              `gorm:"type:varchar(50)" json:"order_item"`
	DocumentRefItem      string              `gorm:"type:varchar(10)" json:"document_ref_item"`
	ProductCode          string              `gorm:"type:varchar(50)" json:"product_code"`
	ProductType          string              `gorm:"type:varchar(10)" json:"product_type"`
	ProductName          string              `gorm:"-" json:"product_name"`
	InterfaceOrderQty    float64             `gorm:"type:decimal(10,4)" json:"interface_order_qty"`
	Qty                  float64             `gorm:"type:decimal(10,4)" json:"qty"`
	UnitCode             string              `gorm:"type:varchar(50)" json:"unit_code"`
	IsFocGwp             bool                `gorm:"type:boolean" json:"is_foc_gwp"`
	WarehouseCode        string              `gorm:"type:varchar(50)" json:"warehouse_code"`
	BatchNo              string              `gorm:"type:varchar(50)" json:"batch_no"`
	SerialCode           string              `gorm:"type:varchar(50)" json:"serial_code"`
	Remark               string              `gorm:"type:varchar" json:"remark"`
	Status               string              `gorm:"type:varchar(20)" json:"status"`
	SaleUnitCode         string              `gorm:"type:varchar(50)" json:"sale_unit_code"`
	SaleMethod           string              `gorm:"type:varchar(50)" json:"sale_method"`
	InterfaceOrderWeight float64             `gorm:"type:decimal(10,4)" json:"interface_order_weight"`
	Weight               float64             `gorm:"type:decimal(10,4)" json:"weight"`
	WeightUnit           float64             `gorm:"type:decimal(10,4)" json:"weight_unit"`
	CreateBy             string              `gorm:"type:varchar(100)" json:"create_by"`
	CreateDtm            time.Time           `gorm:"type:timestamp" json:"create_dtm"`
	UpdateBy             string              `gorm:"type:varchar(100)" json:"update_by"`
	UpdateDtm            time.Time           `gorm:"type:timestamp" json:"update_dtm"`
	DeliveryData         OrderedDeliveryData `json:"delivery_data,omitempty"`
}

// OrderedDeliveryData ensures proper JSON field ordering
type OrderedDeliveryData struct {
	ID               string                `json:"id"`
	DeliveryCode     string                `json:"delivery_code"`
	CompanyCode      string                `json:"company_code"`
	SiteCode         string                `json:"site_code"`
	DeliveryMethod   string                `json:"delivery_method"`
	DocumentRef      string                `json:"document_ref"`
	CustomerCode     string                `json:"customer_code"`
	ShipToAddress    string                `json:"ship_to_address"`
	DeliveryDate     string                `json:"delivery_date"`
	DeliveryTimeCode string                `json:"delivery_time_code"`
	DeliveryTimeName string                `json:"delivery_time_name"`
	LicensePlate     string                `json:"license_plate"`
	ContactName      string                `json:"contact_name"`
	Tel              string                `json:"tel"`
	TotalWeight      float64               `json:"total_weight"`
	Status           string                `json:"status"`
	Remark           string                `json:"remark"`
	CreateDate       string                `json:"create_date"`
	CreateBy         string                `json:"create_by"`
	UpdateDate       string                `json:"update_date"`
	UpdateBy         string                `json:"update_by"`
	Items            []OrderedDeliveryItem `json:"items"`
}

// OrderedDeliveryItem ensures proper JSON field ordering for delivery items
type OrderedDeliveryItem struct {
	ID              string   `json:"id"`
	DeliveryItem    string   `json:"delivery_item"`
	DeliveryID      string   `json:"delivery_id"`
	ProductCode     string   `json:"product_code"`
	Qty             float64  `json:"qty"`
	UnitCode        string   `json:"unit_code"`
	PriceListUnit   float64  `json:"price_list_unit"`
	SaleQty         float64  `json:"sale_qty"`
	SaleUnitCode    string   `json:"sale_unit_code"`
	TotalWeight     float64  `json:"total_weight"`
	DocumentRefItem string   `json:"document_ref_item"`
	Status          string   `json:"status"`
	Weight          float64  `json:"weight"`
	WeightUnit      float64  `json:"weight_unit"`
	CreateDate      string   `json:"create_date"`
	CreateBy        string   `json:"create_by"`
	UpdateDate      string   `json:"update_date"`
	UpdateBy        string   `json:"update_by"`
	Sale            SalePack `json:"sale"`
}

type SalePack struct {
	ID            string               `json:"id"`
	SaleCode      string               `json:"sale_code"`
	CompanyCode   string               `json:"company_code"`
	SiteCode      string               `json:"site_code"`
	CustomerCode  string               `json:"customer_code"`
	CustomerName  string               `json:"customer_name"`
	DeliveryDate  *time.Time           `json:"delivery_date"`
	TotalAmount   float64              `json:"total_amount"`
	TotalWeight   float64              `json:"total_weight"`
	Status        string               `json:"status"`
	StatusPayment string               `json:"status_payment"`
	CreditTerm    string               `json:"credit_term"`
	RefPoDoc      string               `json:"ref_po_doc"`
	CreateDate    *time.Time           `json:"create_date"`
	CreateBy      string               `json:"create_by"`
	UpdateDate    *time.Time           `json:"update_date"`
	UpdateBy      string               `json:"update_by"`
	SaleDeposit   []models.SaleDeposit `json:"sale_deposit"`
	SaleItem      []models.SaleItem    `json:"sale_item"`
}

type ResultPackingResponse struct {
	Total      int                  `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	TotalPages int                  `json:"total_pages"`
	Packings   []GetPackingResponse `json:"packings"`
}

func GetPackSo(jsonPayload GetPackingRequest) (ResultPackingResponse, error) {

	jsonData, err := json.Marshal(jsonPayload)
	if err != nil {
		return ResultPackingResponse{}, errors.New("Error marshaling struct to JSON:" + err.Error())
	}
	req, err := http.NewRequest("POST", config.GET_PACK_SO_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		return ResultPackingResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ResultPackingResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	var dataRes ResultPackingResponse
	err = json.Unmarshal(body, &dataRes)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	return dataRes, nil
}
