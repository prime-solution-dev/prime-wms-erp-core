package externalProductService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"prime-erp-core/config"
	"prime-erp-core/internal/models"
	"time"

	"github.com/google/uuid"
)

func (GetProductsComponent) TableName() string {
	return "product"
}

func (GetUnitsComponent) TableName() string {
	return "product_unit"
}

func (GetUnitsBarcodeComponent) TableName() string {
	return "unit_barcode"
}

type GetProductRequest struct {
	ID              []uuid.UUID `json:"id"`
	ProductCode     []string    `json:"product_code"`
	ProductCodeLike string      `json:"product_code_like"`
	ProductNameLike string      `json:"product_name_like"`
	VersionNameLike string      `json:"version_name_like"`
	SupplierCode    string      `json:"supplier_code"`
	SiteCodeLike    string      `json:"site_code_like"`
	Barcode         []string    `json:"barcode"`
	NotProductCode  []string    `json:"not_product_code"`
	ActiveFlg       []bool      `json:"active_flg"`
	SiteCode        []string    `json:"site_code"`
	CompanyCode     []string    `json:"company_code"`
	GroupCodes      []string    `json:"group_codes"`
	ItemCodes       []string    `json:"item_codes"`
	IsMasterData    bool        `json:"is_master_data"`
	Page            int         `json:"page"`
	PageSize        int         `json:"page_size"`
}

type GetProductsResponse struct {
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	TotalPages int                    `json:"total_pages"`
	Products   []GetProductsComponent `json:"products"`
}

type GetProductsComponent struct {
	ID                            uuid.UUID             `json:"product_id"`
	ProductCode                   string                `json:"product_code"`
	TenantId                      uuid.UUID             `json:"tenant_id"`
	ProductName                   string                `json:"product_name"`
	Description                   string                `json:"description"`
	ActiveFlg                     bool                  `json:"active_flg"`
	CategoryCode                  string                `json:"category_code"`
	ImgUrl                        string                `json:"img_url"`
	ProductType                   string                `json:"product_type"`
	ShelfLifeDay                  int                   `json:"shelf_life_day"`
	FlagProductExpire             bool                  `json:"flag_product_expire"`
	Brand                         string                `json:"brand"`
	ABCIndicator                  string                `json:"abc_indicator"`
	FlgExcludeAccounting          bool                  `json:"flg_exclude_accounting"`
	FlgIgnoreCustomerMinShelfLife bool                  `json:"flg_ignore_customer_min_shelf_life"`
	NormalGoodsMinShelfLifeDay    int                   `json:"normal_goods_min_shelf_life_day"`
	FreeGoodsMinShelfLifeDay      int                   `json:"free_goods_min_shelf_life_day"`
	StorageZone                   string                `json:"storage_zone"`
	PutawayStrategyCode           string                `json:"putaway_strategy_code"`
	SupplierCode                  string                `json:"supplier_code"`
	SubstitutionMaterialCode      string                `json:"substitution_material_code"`
	MaxQty                        float64               `json:"max_qty"`
	ReorderQty                    float64               `json:"reorder_qty"`
	RoundingUnit                  string                `json:"rounding_unit"`
	QtyBomKitting                 float64               `json:"qty_bom_kitting"`
	QtyBomProduction              float64               `json:"qty_bom_production"`
	SiteCode                      string                `json:"site_code"`
	CompanyCode                   string                `json:"company_code"`
	CreateBy                      string                `json:"create_by"`
	CreateDtm                     time.Time             `json:"create_dtm"`
	UpdateBy                      string                `json:"update_by"`
	UpdateDtm                     time.Time             `json:"update_dtm"`
	ExternalID                    string                `json:"external_id"`
	GRTolerance                   float64               `json:"gr_tolerance"`
	GRToleranceActive             bool                  `json:"gr_tolerance_active"`
	UnitInterface                 string                `json:"unit_interface"`
	AdjustmentUnit                string                `json:"adjustment_unit"`
	Weight                        float64               `json:"weight"`
	IsBatch                       bool                  `gorm:"column:is_batch;type:bool" json:"is_batch"`
	IsSerial                      bool                  `gorm:"column:is_serial;type:bool" json:"is_serial"`
	IsMfgDateControl              bool                  `gorm:"column:is_mfg_date_control;type:bool" json:"is_mfg_date_control"`
	IsExpiryDateControl           bool                  `gorm:"column:is_expiry_date_control;type:bool" json:"is_expiry_date_control"`
	Units                         []GetUnitsComponent   `json:"units"`
	ProductGroup                  []models.ProductGroup `json:"product_groups"`
	//Groups                        []GetGroupResponse    `gorm:"-" json:"groups"`
}

type GetUnitsComponent struct {
	ID            uuid.UUID                  `json:"id"`
	ProductId     uuid.UUID                  `json:"product_id"`
	UnitId        uuid.UUID                  `json:"unit_id"`
	Dimension     float64                    `json:"dimension"`
	Ratio         float64                    `json:"ratio"`
	RatioStandard float64                    `json:"ratio_standard"`
	Width         float64                    `json:"width"`
	Length        float64                    `json:"length"`
	Height        float64                    `json:"height"`
	Weight        float64                    `json:"weight"`
	Cubic         float64                    `json:"cubic"`
	ImgUrl        string                     `json:"img_url"`
	FlagBase      bool                       `json:"flag_base"`
	FlagGr        bool                       `json:"flag_gr"`
	FlagSale      bool                       `json:"flag_sale"`
	FlagGi        bool                       `json:"flag_gi"`
	FlagPick      bool                       `json:"flag_pick"`
	FlagTransfer  bool                       `json:"flag_transfer"`
	FlagCount     bool                       `json:"flag_count"`
	FlagPallet    bool                       `json:"flag_pallet"`
	FlagContainer bool                       `json:"flag_container"`
	ActiveFlg     bool                       `json:"active_flg"`
	IsWeight      bool                       `json:"is_weight"`
	UnitCode      string                     `json:"unit_code"`
	UnitName      string                     `json:"unit_name"`
	IsBatch       bool                       `gorm:"type:bool" json:"is_batch"`
	IsSerial      bool                       `gorm:"type:bool" json:"is_serial"`
	Barcodes      []GetUnitsBarcodeComponent `json:"barcodes"`
}

type GetUnitsBarcodeComponent struct {
	ID            uuid.UUID `gorm:"type:uuid" json:"id"`
	ProductUnitId uuid.UUID `gorm:"type:uuid" json:"product_unit_id"`
	Barcode       string    `gorm:"type:varchar(100)" json:"barcode"`
}

func GetProduct(jsonPayload GetProductRequest) (GetProductsResponse, error) {

	jsonData, err := json.Marshal(jsonPayload)
	if err != nil {
		return GetProductsResponse{}, errors.New("Error marshaling struct to JSON:")
	}
	req, err := http.NewRequest("POST", config.GET_CUSTOMER_MASTER_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		return GetProductsResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}

	req.Header.Set("Content-Type", "application/json")

	// Create a client and execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return GetProductsResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Response Status:", err)
	}
	var res GetProductsResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		fmt.Println("Response Status:", err)
	}
	return res, nil
}
