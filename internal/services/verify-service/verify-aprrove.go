package verifyService

import (
	"encoding/json"
	"errors"
	"time"

	"prime-erp-core/internal/db"
	priceService "prime-erp-core/internal/services/price-service"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"gorm.io/gorm"
)

type VerifyApproveRequest struct {
	IsVerifyPrice          bool                    `json:"is_verify_price"`
	IsVerifyExpiryPrice    bool                    `json:"is_verify_expiry_price"`
	IsVerifyInventory      bool                    `json:"is_verify_inventory"`
	IsVerifyCredit         bool                    `json:"is_verify_credit"`
	CompanyCode            string                  `json:"company_code"`
	SiteCode               string                  `json:"site_code"`
	SaleDate               time.Time               `json:"sale_date"`
	InventoryWarehouseCode []string                `json:"inventory_warehouse_code"` //For check Inventory
	StorageType            []string                `json:"storage_type"`
	Documents              []VerifyApproveDocument `json:"documents"`
}

type VerifyApproveDocument struct {
	DocRef                       string              `json:"doc_ref"`
	CustomerCode                 string              `json:"customer_code"`
	EffectiveDatePrice           time.Time           `json:"effective_date_price"`
	TransportCost                float64             `json:"transport_cost"`
	TransportType                string              `json:"transport_type"`
	IsVerifyWithOldTransportCost bool                `json:"is_verify_with_old_transport_cost"`
	Items                        []VerifyApproveItem `json:"items"`

	//Result
	IsPassPrice       bool `json:"is_pass_price"`
	IsPassCredit      bool `json:"is_pass_credit"`
	IsPassExpiryPrice bool `json:"is_pass_expiry_price"`
	IsPassInventory   bool `json:"is_pass_inventory"`
}

type VerifyApproveItem struct {
	ItemRef       string  `json:"item_ref"`
	ProductCode   string  `json:"product_code"`
	Qty           float64 `json:"qty"`
	Unit          string  `json:"unit_code"`
	TotalWeight   float64 `json:"total_weight"`
	PriceUnit     float64 `json:"price"`
	PriceListUnit float64 `json:"price_list_unit"`
	TotalAmount   float64 `json:"total_amount"`
	SaleUnit      string  `json:"sale_unit"`
	SaleUnitType  string  `json:"sale_unit_type"`

	//Option
	TransportCostUnit       *float64 `json:"transport_cost_unit"`
	TransportCostUnitWeight *float64 `json:"transport_cost_unit_weight"`
}

type VerifyApproveResponse struct {
	IsPassPrice       bool                    `json:"is_pass_price"`
	IsPassCredit      bool                    `json:"is_pass_credit"`
	IsPassExpiryPrice bool                    `json:"is_pass_expiry_price"`
	IsPassInventory   bool                    `json:"is_pass_inventory"`
	Documents         []VerifyApproveDocument `json:"documents"`
}

func VerifyApprove(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := VerifyApproveRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	sqlx, err := db.ConnectSqlx(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer sqlx.Close()

	gormx, err := db.ConnectGORM(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	return VerifyApproveLogic(gormx, sqlx, req)
}

func VerifyApproveLogic(gormx *gorm.DB, sqlx *sqlx.DB, req VerifyApproveRequest) (*VerifyApproveResponse, error) {
	res := VerifyApproveResponse{
		Documents: []VerifyApproveDocument{},
	}

	if len(req.Documents) == 0 {
		return nil, errors.New("document reference is required")
	}

	for _, document := range req.Documents {
		if len(document.Items) == 0 {
			return nil, errors.New("require at least one item")
		}
	}

	if req.SaleDate.IsZero() {
		return nil, errors.New("sale date is required")
	}

	if req.CompanyCode == `` {
		return nil, errors.New("company code is required")
	}

	if req.SiteCode == `` {
		return nil, errors.New("site code is required")
	}

	expPriceReq := []VerifyExpiryPriceRequest{}
	priceReqMap := map[string]priceService.GetComparePriceRequest{}
	creditCustomerMap := map[string]VerifyCreditCustomer{}
	inventoryReq := VerifyInventoryRequest{}
	productInvMap := map[string]VerifyInventoryProduct{}

	inventoryReq.CompanyCode = req.CompanyCode
	inventoryReq.SiteCode = req.SiteCode
	inventoryReq.StorageTypes = req.StorageType
	inventoryReq.ToDate = &req.SaleDate

	for _, document := range req.Documents {
		//Build Res
		resDoc := VerifyApproveDocument{
			DocRef:       document.DocRef,
			CustomerCode: document.CustomerCode,
			Items:        document.Items,
		}

		isVerifyWithOldTransportCost := document.IsVerifyWithOldTransportCost

		//Price
		priceKey := document.DocRef
		newPriceReq := priceService.GetComparePriceRequest{}
		newPriceReq.UnitCode = `PC`       //TODO: get from config
		newPriceReq.UnitCodeWeight = `KG` //TODO: get from config

		//Expiry
		expPriceReq = append(expPriceReq, VerifyExpiryPriceRequest{
			DocumentRef:        document.DocRef,
			EffectiveDatePrice: document.EffectiveDatePrice,
		})

		//Credit
		creditCustKey := document.CustomerCode
		creditCust, exstCredit := creditCustomerMap[creditCustKey]
		if !exstCredit {

			creditCust = VerifyCreditCustomer{
				CustomerCode: document.CustomerCode,
				NeedAmount:   0,
			}
		}

		if document.TransportType == `EXCL` {
			creditCust.NeedAmount += document.TransportCost
		}

		for _, docItem := range document.Items {
			//Price
			itemPrice := priceService.ItemComparePrice{
				RefItem:       docItem.ItemRef,
				ProductCode:   docItem.ProductCode,
				Qty:           docItem.Qty,
				Unit:          docItem.Unit,
				PriceUnit:     docItem.PriceUnit,
				PriceListUnit: docItem.PriceListUnit,
				TotalAmount:   docItem.TotalAmount,
				TotalWeight:   docItem.TotalWeight,
				SaleUnit:      docItem.SaleUnit,
				SaleUnitType:  docItem.SaleUnitType,
			}

			//Optional for transport cost verification
			if isVerifyWithOldTransportCost {
				itemPrice.TransportCostUnit = docItem.TransportCostUnit
				itemPrice.TransportCostUnitWeight = docItem.TransportCostUnitWeight
			}

			newPriceReq.Items = append(newPriceReq.Items, itemPrice)

			//Credit
			creditCust.NeedAmount += docItem.TotalAmount

			//Inventory
			productInvKey := docItem.ProductCode
			productInv, existProductInv := productInvMap[productInvKey]
			if !existProductInv {
				newProductInv := VerifyInventoryProduct{
					ProductCode: docItem.ProductCode,
					Qty:         0,
				}

				productInv = newProductInv
			}

			productInv.Qty += docItem.Qty
			productInvMap[productInvKey] = productInv
		}

		priceReqMap[priceKey] = newPriceReq
		creditCustomerMap[creditCustKey] = creditCust
		res.Documents = append(res.Documents, resDoc)
	}

	for _, product := range productInvMap {
		inventoryReq.Products = append(inventoryReq.Products, product)
	}

	//Expiry Date Validation
	if req.IsVerifyExpiryPrice {
		res.IsPassExpiryPrice = true

		expPrice, err := VerifyExpiryPriceLogic(gormx, expPriceReq)
		if err != nil {
			return nil, err
		}

		for _, vExp := range *expPrice {
			for cRes, resDocument := range res.Documents {
				if resDocument.DocRef == vExp.DocumentRef {
					res.Documents[cRes].IsPassExpiryPrice = vExp.IsPassVerified
					break
				}
			}

			if res.IsPassExpiryPrice && !vExp.IsPassVerified {
				res.IsPassExpiryPrice = false
			}
		}
	}

	//Price Validation
	if req.IsVerifyPrice {
		res.IsPassPrice = true

		for priceKey, priceReq := range priceReqMap {
			compareRes, err := VerifyPrice(priceReq)
			if err != nil {
				return nil, err
			}

			for cResDoc, resDoc := range res.Documents {
				if resDoc.DocRef == priceKey {
					res.Documents[cResDoc].IsPassPrice = compareRes.IsPassPriceUnitAll && compareRes.IsPassPriceWeightAll

					if res.IsPassPrice && !res.Documents[cResDoc].IsPassPrice {
						res.IsPassPrice = false
					}
				}
			}
		}
	}

	// Credit Validation
	if req.IsVerifyCredit {
		res.IsPassCredit = true

		creditReq := VerifyCreditRequest{
			Customers: []VerifyCreditCustomer{},
		}

		for _, cust := range creditCustomerMap {
			creditReq.Customers = append(creditReq.Customers, cust)
		}

		creditRes, err := VerifyCreditLogic(sqlx, creditReq)
		if err != nil {
			return nil, err
		}

		for _, creditCust := range creditRes.Customers {
			for i, doc := range res.Documents {
				if doc.CustomerCode == creditCust.CustomerCode {
					res.Documents[i].IsPassCredit = creditCust.IsPass
				}
			}

			if res.IsPassCredit && !creditCust.IsPass {
				res.IsPassCredit = false
			}
		}
	}

	//Inventory Validation
	if req.IsVerifyInventory {
		res.IsPassInventory = true

		// invenRes, err := VerifyInventoryLogic(inventoryReq)
		// if err != nil {
		// 	return nil, err
		// }

		// if res.IsPassInventory && !invenRes.IsPassInventory {
		// 	res.IsPassInventory = false
		// }
	}

	return &res, nil
}
