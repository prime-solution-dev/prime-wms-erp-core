package saleService

import (
	"encoding/json"
	"errors"
	"fmt"
	"prime-erp-core/internal/db"
	verifyService "prime-erp-core/internal/services/verify-service"
	"time"

	"github.com/gin-gonic/gin"
)

// ValidateSaleRequest - เฉพาะข้อมูลที่จำเป็นสำหรับการ validate
type ValidateSaleRequest struct {
	IsVerifyPrice      bool                     `json:"is_verify_price"`
	IsVerifyCredit     bool                     `json:"is_verify_credit"`
	IsVerifyExpiryDate bool                     `json:"is_verify_expiry_date"`
	IsVerifyInventory  bool                     `json:"is_verify_inventory"`
	Sales              []SaleValidationDocument `json:"sales"`
}

type SaleValidationDocument struct {
	CompanyCode  string               `json:"company_code"`
	SiteCode     string               `json:"site_code"`
	CustomerCode string               `json:"customer_code"`
	DeliveryDate *time.Time           `json:"delivery_date"`
	Items        []SaleValidationItem `json:"items"`
}

type SaleValidationItem struct {
	ProductCode   string  `json:"product_code"`
	Qty           float64 `json:"qty"`
	Unit          string  `json:"unit"`
	TotalWeight   float64 `json:"total_weight"`
	PriceUnit     float64 `json:"price_unit"`
	PriceListUnit float64 `json:"price_list_unit"`
	TotalAmount   float64 `json:"total_amount"`
	SaleUnit      string  `json:"sale_unit"`
	SaleUnitType  string  `json:"sale_unit_type"`
}

type ValidateSaleResponse struct {
	CanCreateSO      bool   `json:"can_create_so"`    // true = สามารถสร้าง SO ได้
	RecommendStatus  string `json:"recommend_status"` // APPROVED หรือ WAIT_FOR_APPROVED
	IsPassPrice      bool   `json:"is_pass_price"`
	IsPassCredit     bool   `json:"is_pass_credit"`
	IsPassInventory  bool   `json:"is_pass_inventory"`
	IsPassExpiryDate bool   `json:"is_pass_expiry_date"`
	Message          string `json:"message"` // ข้อความอธิบายผลลัพธ์

	// ยอดเครดิตคงเหลือของลูกค้า สำหรับโชว์ "( Balance : x )" ข้างผล Credit limit
	// nil เมื่อ is_verify_credit = false (เช่น จ่ายเงินสด) เพราะ VerifyApproveLogic ไม่คำนวณให้
	CreditCalculation *verifyService.VerifyCreditCalculation `json:"credit_calculation,omitempty"`
	// ผล ATP รายสินค้า สำหรับโชว์ ATP คงเหลือใต้ qty ของ item ที่ is_pass = false
	// รวม qty ตาม product_code แล้ว 1 product = 1 แถว (VerifyApproveLogic:200-213)
	InventoryCalculations []verifyService.VerifyInventoryCalculation `json:"inventory_calculations"`
	// ผลดิบทั้งก้อนจาก VerifyApproveLogic เผื่อหน้าจอต้องใช้ field อื่นเพิ่ม
	VerifyResult *verifyService.VerifyApproveResponse `json:"verify_result,omitempty"`
}

// ValidateSale - ตรวจสอบเงื่อนไขการสร้าง Sale Order โดยไม่สร้างข้อมูลจริง
func ValidateSale(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := ValidateSaleRequest{}
	var responses []ValidateSaleResponse

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

	verifyReqMap := map[string]verifyService.VerifyApproveRequest{}

	// สร้าง verification requests
	for _, saleReq := range req.Sales {
		verifyReqKey := fmt.Sprintf(`%s|%s`, saleReq.CompanyCode, saleReq.SiteCode)
		verifyReq, existVerifyReq := verifyReqMap[verifyReqKey]
		if !existVerifyReq {
			newVerifyReq := verifyService.VerifyApproveRequest{
				IsVerifyPrice:       req.IsVerifyPrice,
				IsVerifyCredit:      req.IsVerifyCredit,
				IsVerifyExpiryPrice: req.IsVerifyExpiryDate,
				IsVerifyInventory:   req.IsVerifyInventory,
				CompanyCode:         saleReq.CompanyCode,
				SiteCode:            saleReq.SiteCode,
				StorageType:         []string{`NORMAL`},
				SaleDate:            *saleReq.DeliveryDate,
			}

			verifyReq = newVerifyReq
		}

		newApprDoc := verifyService.VerifyApproveDocument{
			DocRef:       fmt.Sprintf("TEMP_%s_%s", saleReq.CompanyCode, saleReq.SiteCode),
			CustomerCode: saleReq.CustomerCode,
			Items:        []verifyService.VerifyApproveItem{},
		}

		for _, item := range saleReq.Items {
			newApprItem := verifyService.VerifyApproveItem{
				ItemRef:       fmt.Sprintf("temp_item_%s", item.ProductCode),
				ProductCode:   item.ProductCode,
				Qty:           item.Qty,
				Unit:          item.Unit,
				TotalWeight:   item.TotalWeight,
				PriceUnit:     item.PriceUnit,
				PriceListUnit: item.PriceListUnit,
				TotalAmount:   item.TotalAmount,
				SaleUnit:      item.SaleUnit,
				SaleUnitType:  item.SaleUnitType,
			}
			newApprDoc.Items = append(newApprDoc.Items, newApprItem)
		}

		verifyReq.Documents = append(verifyReq.Documents, newApprDoc)
		verifyReqMap[verifyReqKey] = verifyReq
	}

	// ตรวจสอบเงื่อนไขต่างๆ
	for _, verifyReq := range verifyReqMap {
		verifyRes, err := verifyService.VerifyApproveLogic(gormx, sqlx, verifyReq)
		if err != nil {
			return nil, err
		}

		// กำหนดผลลัพธ์ตามเงื่อนไข
		allPassed := verifyRes.IsPassPrice && verifyRes.IsPassCredit && verifyRes.IsPassInventory && verifyRes.IsPassExpiryPrice
		criticalPassed := verifyRes.IsPassCredit && verifyRes.IsPassInventory && verifyRes.IsPassExpiryPrice

		var canCreateSO bool
		var recommendStatus string
		var message string

		if !criticalPassed {
			canCreateSO = false
			recommendStatus = ""
			message = "ไม่สามารถสร้าง SO ได้ เนื่องจาก ATP stock, Credit limit หรือ Price valid date ไม่ผ่าน"
		} else if allPassed {
			canCreateSO = true
			recommendStatus = "APPROVED"
			message = "ผ่านเงื่อนไขทั้งหมด สามารถสร้าง SO ได้ทันที"
		} else {
			// criticalPassed = true แต่ price ไม่ผ่าน
			canCreateSO = true
			recommendStatus = "WAIT_FOR_APPROVED"
			message = "สามารถสร้าง SO ได้ แต่ต้องรออนุมัติเนื่องจากราคาไม่ผ่าน หรือสามารถกลับไปแก้ไข Quotation"
		}

		// เอกสารถูกสร้างแบบ 1 doc ต่อ 1 company|site (DocRef = TEMP_<company>_<site>)
		// จึงอ่านเครดิตจาก doc ตัวแรกได้
		var creditCalculation *verifyService.VerifyCreditCalculation
		if len(verifyRes.Documents) > 0 {
			creditCalculation = verifyRes.Documents[0].CreditCalculation
		}

		responses = append(responses, ValidateSaleResponse{
			CanCreateSO:           canCreateSO,
			RecommendStatus:       recommendStatus,
			IsPassPrice:           verifyRes.IsPassPrice,
			IsPassCredit:          verifyRes.IsPassCredit,
			IsPassInventory:       verifyRes.IsPassInventory,
			IsPassExpiryDate:      verifyRes.IsPassExpiryPrice,
			Message:               message,
			CreditCalculation:     creditCalculation,
			InventoryCalculations: verifyRes.InventoryCalculations,
			VerifyResult:          verifyRes,
		})
	}

	return responses, nil
}
