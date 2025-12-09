package saleService

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	externalService "prime-erp-core/external/pack-service"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
)

type GetSalePackRequest struct {
	IsNotMatchIv bool     `json:"is_not_match_iv"`
	PackingCode  []string `json:"packing_code"`
	StatusPack   []string `json:"status_pack"`
	SaleCode     []string `json:"sale_code"`
	CompanyCode  []string `json:"company_code"`
	SiteCode     []string `json:"site_code"`
	Page         int      `json:"page"`
	PageSize     int      `json:"page_size"`
}

type GetSalePackResponse struct {
	ID               uuid.UUID                    `json:"id"`
	SaleCode         string                       `json:"sale_code"`
	CompanyCode      string                       `json:"company_code"`
	SiteCode         string                       `json:"site_code"`
	CustomerCode     string                       `json:"customer_code"`
	CustomerName     string                       `json:"customer_name"`
	DeliveryDate     *time.Time                   `json:"delivery_date"`
	TotalAmount      float64                      `json:"total_amount"`
	TotalWeight      float64                      `json:"total_weight"`
	Status           string                       `json:"status"`
	StatusPayment    string                       `json:"status_payment"`
	CreateDate       *time.Time                   `json:"create_date"`
	CreateBy         string                       `json:"create_by"`
	UpdateDate       *time.Time                   `json:"update_date"`
	UpdateBy         string                       `json:"update_by"`
	SaleItem         []models.SaleItem            `json:"sale_item"`
	Deliveries       []models.Delivery            `json:"deliveries"`
	DeliveryCodes    []string                     `json:"delivery_codes,omitempty"`
	ExcludedPackCode []string                     `json:"excluded_pack_code,omitempty"`
	PackingResponse  *PackingResponseWithDelivery `json:"packing_response,omitempty"`
}

type PackingServiceRequest struct {
	DeliveryCodes    []string `json:"delivery_codes"`
	ExcludedPackCode []string `json:"excluded_pack_code"`
}

// Enhanced response structures with delivery data
type PackingResponseWithDelivery struct {
	Total      int                       `json:"total"`
	Page       int                       `json:"page"`
	PageSize   int                       `json:"pageSize"`
	TotalPages int                       `json:"totalPages"`
	Packings   []PackingWithDeliveryData `json:"packings"`
}

type PackingWithDeliveryData struct {
	ID              string                        `json:"id"`
	PackingCode     string                        `json:"packingCode"`
	PackingType     string                        `json:"packingType"`
	TenantId        string                        `json:"tenantId"`
	CompanyCode     string                        `json:"companyCode"`
	SiteCode        string                        `json:"siteCode"`
	WarehouseCode   string                        `json:"warehouseCode"`
	DocumentRefType string                        `json:"documentRefType"`
	DocumentRef     string                        `json:"documentRef"`
	Status          string                        `json:"status"`
	SubmitDate      *time.Time                    `json:"submitDate"`
	CreateBy        string                        `json:"createBy"`
	CreateDtm       time.Time                     `json:"createDtm"`
	UpdateBy        string                        `json:"updateBy"`
	UpdateDtm       time.Time                     `json:"updateDtm"`
	PackingItem     []PackingItemWithDeliveryData `json:"packingItem"`
}

type PackingItemWithDeliveryData struct {
	ID                 string                                          `json:"id"`
	PackingId          string                                          `json:"packingId"`
	PackingItem        string                                          `json:"packingItem"`
	DocumentRef        string                                          `json:"documentRef"`
	DocumentRefItem    string                                          `json:"documentRefItem"`
	PackNo             string                                          `json:"packNo"`
	ZoneCode           string                                          `json:"zoneCode"`
	LocationCode       string                                          `json:"locationCode"`
	PalletCode         string                                          `json:"palletCode"`
	ContainerCode      string                                          `json:"containerCode"`
	ProductCode        string                                          `json:"productCode"`
	Qty                float64                                         `json:"qty"`
	UnitCode           string                                          `json:"unitCode"`
	BaseQty            float64                                         `json:"baseQty"`
	BaseUnitCode       string                                          `json:"baseUnitCode"`
	Remark             string                                          `json:"remark"`
	Status             string                                          `json:"status"`
	CreateBy           string                                          `json:"createBy"`
	CreateDtm          time.Time                                       `json:"createDtm"`
	UpdateBy           string                                          `json:"updateBy"`
	UpdateDtm          time.Time                                       `json:"updateDtm"`
	PackingItemConfirm []externalService.GetPackingItemConfirmResponse `json:"packingItemConfirm"`
	DeliveryData       *DeliveryData                                   `json:"delivery_data,omitempty"`
}

type DeliveryData struct {
	DeliveryCode string                 `json:"delivery_code"`
	SaleCode     string                 `json:"sale_code"`
	CustomerCode string                 `json:"customer_code"`
	CustomerName string                 `json:"customer_name"`
	DeliveryDate string                 `json:"delivery_date"`
	Status       string                 `json:"status"`
	Items        []DeliveryItemResponse `json:"items"`
}

type DeliveryItemResponse struct {
	DeliveryItemCode string  `json:"delivery_item_code"`
	ProductCode      string  `json:"product_code"`
	ProductName      string  `json:"product_name"`
	Quantity         float64 `json:"quantity"`
	UnitCode         string  `json:"unit_code"`
}

func GetSalePack(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var res []GetSalePackResponse
	var req GetSalePackRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	// Query sales with status_payment == "PENDING"
	query := gormx.Where("status_payment = ?", "PENDING").
		Preload("SaleItem").
		Order("update_date DESC")

	// Apply filters
	if len(req.SaleCode) > 0 {
		query = query.Where("sale_code IN ?", req.SaleCode)
	}

	if len(req.CompanyCode) > 0 {
		query = query.Where("company_code IN ?", req.CompanyCode)
	}

	if len(req.SiteCode) > 0 {
		query = query.Where("site_code IN ?", req.SiteCode)
	}

	// Execute query
	var sales []models.Sale
	if err := query.Find(&sales).Error; err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve sales"})
		return nil, err
	}

	// Process each sale
	for _, sale := range sales {
		saleResponse := GetSalePackResponse{
			ID:            sale.ID,
			SaleCode:      sale.SaleCode,
			CompanyCode:   sale.CompanyCode,
			SiteCode:      sale.SiteCode,
			CustomerCode:  sale.CustomerCode,
			CustomerName:  sale.CustomerName,
			DeliveryDate:  sale.DeliveryDate,
			TotalAmount:   sale.TotalAmount,
			TotalWeight:   sale.TotalWeight,
			Status:        sale.Status,
			StatusPayment: sale.StatusPayment,
			CreateDate:    sale.CreateDate,
			CreateBy:      sale.CreateBy,
			UpdateDate:    sale.UpdateDate,
			UpdateBy:      sale.UpdateBy,
			SaleItem:      sale.SaleItem,
		}

		// Join delivery from saleCode with delivery.documentRef
		var deliveries []models.Delivery
		if err := gormx.Where("document_ref = ?", sale.SaleCode).Find(&deliveries).Error; err == nil {
			saleResponse.Deliveries = deliveries

			// Extract delivery codes for external service call
			deliveryCodesMap := make(map[string]bool)
			for _, delivery := range deliveries {
				if delivery.DeliveryCode != "" {
					deliveryCodesMap[delivery.DeliveryCode] = true
				}
			}

			// Convert delivery codes map to slice
			for deliveryCode := range deliveryCodesMap {
				saleResponse.DeliveryCodes = append(saleResponse.DeliveryCodes, deliveryCode)
			}
		}

		// If is_not_match_iv == true, join with invoice items to get excluded pack codes
		if req.IsNotMatchIv {
			var invoiceItems []models.InvoiceItem
			if err := gormx.Joins("JOIN invoice ON invoice_item.invoice_id = invoice.id").
				Where("invoice.status IN ? AND invoice_item.document_ref = ? AND invoice.invoice_type = ?",
					[]string{"PENDING", "COMPLETED"}, sale.SaleCode, "AR").
				Find(&invoiceItems).Error; err == nil {

				// Extract unique excluded pack codes from sourceCode (pack codes to avoid)
				excludedPackCodesMap := make(map[string]bool)
				for _, item := range invoiceItems {
					if item.SourceCode != "" {
						excludedPackCodesMap[item.SourceCode] = true
					}
				}

				// Convert excluded pack codes map to slice
				for packCode := range excludedPackCodesMap {
					saleResponse.ExcludedPackCode = append(saleResponse.ExcludedPackCode, packCode)
				}
			}
		}

		res = append(res, saleResponse)
	}

	// Call external packing service
	externalPackingResponse, err := callPackingService(res, req)
	if err != nil {
		fmt.Printf("Error calling external packing service: %v\n", err)
		// Return empty result if external service fails
		return externalService.ResultPackingResponse{
			Total:      0,
			Page:       req.Page,
			PageSize:   req.PageSize,
			TotalPages: 0,
			Packings:   []externalService.GetPackingResponse{},
		}, nil
	}

	// Map sale and delivery data directly into the external response
	err = mapDeliveryDataToOrderItems(gormx, &externalPackingResponse.Packings, res)
	if err != nil {
		fmt.Printf("Error mapping delivery data: %v\n", err)
	}

	return externalPackingResponse, nil
}

// callPackingService รวบรวม delivery codes และ excluded pack codes จาก sales ทั้งหมด แล้วเรียก external packing service
func callPackingService(sales []GetSalePackResponse, req GetSalePackRequest) (externalService.ResultPackingResponse, error) {
	allDeliveryCodes := make(map[string]bool)
	allExcludedPackCodes := make(map[string]bool)

	for _, sale := range sales {
		// Collect delivery codes
		for _, deliveryCode := range sale.DeliveryCodes {
			if deliveryCode != "" {
				allDeliveryCodes[deliveryCode] = true
			}
		}

		// Collect excluded pack codes
		for _, packCode := range sale.ExcludedPackCode {
			if packCode != "" {
				allExcludedPackCodes[packCode] = true
			}
		}
	}

	// Convert maps to slices
	deliveryCodes := make([]string, 0, len(allDeliveryCodes))
	excludedPackCodes := make([]string, 0, len(allExcludedPackCodes))

	for deliveryCode := range allDeliveryCodes {
		deliveryCodes = append(deliveryCodes, deliveryCode)
	}

	for packCode := range allExcludedPackCodes {
		excludedPackCodes = append(excludedPackCodes, packCode)
	}

	// Build external service request
	packingRequest := externalService.GetPackingRequest{
		DeliveryCodes:    deliveryCodes,
		ExcludedPackCode: excludedPackCodes,
		PackingCode:      req.PackingCode,
		StatusPack:       req.StatusPack,
		Page:             req.Page,
		PageSize:         req.PageSize,
	}

	fmt.Printf("packingRequest: %+v\n", packingRequest)
	packingResponse, err := externalService.GetPackSo(packingRequest)
	if err != nil {
		return externalService.ResultPackingResponse{}, errors.New("Error calling packing service: " + err.Error())
	}
	fmt.Printf("packingResponse: %+v\n", packingResponse)

	return packingResponse, nil
}

// mapDeliveryDataToOrderItems แมพ delivery_data เข้าไปใน order_item ของ outbound
func mapDeliveryDataToOrderItems(gormx *gorm.DB, packings *[]externalService.GetPackingResponse, salesData []GetSalePackResponse) error {
	// Create delivery data map for quick lookup
	deliveryDataMap := make(map[string]externalService.OrderedDeliveryData)

	// Build delivery data from sales
	for _, saleData := range salesData {
		for _, delivery := range saleData.Deliveries {
			// Get delivery items
			var deliveryItems []models.DeliveryItem
			gormx.Where("delivery_id = ?", delivery.ID).Find(&deliveryItems)

			// Build delivery item responses with ordered structure
			var items []externalService.OrderedDeliveryItem
			for _, item := range deliveryItems {
				// Filter sale items to only include the one that matches delivery_item.document_ref_item
				var filteredSaleItems []models.SaleItem
				for _, saleItem := range saleData.SaleItem {
					if saleItem.SaleItem == item.DocumentRefItem {
						filteredSaleItems = append(filteredSaleItems, saleItem)
					}
				}

				orderedItem := externalService.OrderedDeliveryItem{
					ID:              item.ID.String(),
					DeliveryItem:    item.DeliveryItem,
					DeliveryID:      item.DeliveryID.String(),
					ProductCode:     item.ProductCode,
					Qty:             item.Qty,
					UnitCode:        item.UnitCode,
					PriceListUnit:   item.PriceListUnit,
					SaleQty:         item.SaleQty,
					SaleUnitCode:    item.SaleUnitCode,
					TotalWeight:     item.TotalWeight,
					DocumentRefItem: item.DocumentRefItem,
					Status:          item.Status,
					Weight:          item.Weight,
					WeightUnit:      item.WeightUnit,
					CreateDate:      item.CreateDate.Format("2006-01-02T15:04:05Z"),
					CreateBy:        item.CreateBy,
					UpdateDate:      item.UpdateDate.Format("2006-01-02T15:04:05Z"),
					UpdateBy:        item.UpdateBy,
					Sale: externalService.SalePack{
						ID:            saleData.ID.String(),
						SaleCode:      saleData.SaleCode,
						CompanyCode:   saleData.CompanyCode,
						SiteCode:      saleData.SiteCode,
						CustomerCode:  saleData.CustomerCode,
						CustomerName:  saleData.CustomerName,
						DeliveryDate:  saleData.DeliveryDate,
						TotalAmount:   saleData.TotalAmount,
						TotalWeight:   saleData.TotalWeight,
						Status:        saleData.Status,
						StatusPayment: saleData.StatusPayment,
						CreateDate:    saleData.CreateDate,
						CreateBy:      saleData.CreateBy,
						UpdateDate:    saleData.UpdateDate,
						UpdateBy:      saleData.UpdateBy,
						SaleItem:      filteredSaleItems, // ← ใช้ filtered sale items แทน
					},
				}
				items = append(items, orderedItem)
			}

			var deliveryDate string
			if delivery.DeliveryDate != nil {
				deliveryDate = delivery.DeliveryDate.Format("2006-01-02T15:04:05Z")
			}

			// Create ordered delivery data structure
			deliveryData := externalService.OrderedDeliveryData{
				ID:               delivery.ID.String(),
				DeliveryCode:     delivery.DeliveryCode,
				CompanyCode:      delivery.CompanyCode,
				SiteCode:         delivery.SiteCode,
				DeliveryMethod:   delivery.DeliveryMethod,
				DocumentRef:      delivery.DocumentRef,
				CustomerCode:     delivery.CustomerCode,
				ShipToAddress:    delivery.ShipToAddress,
				DeliveryDate:     deliveryDate,
				DeliveryTimeCode: delivery.DeliveryTimeCode,
				DeliveryTimeName: "06:00 - 18:00",
				LicensePlate:     delivery.LicensePlate,
				ContactName:      delivery.ContactName,
				Tel:              delivery.Tel,
				TotalWeight:      delivery.TotalWeight,
				Status:           delivery.Status,
				Remark:           delivery.Remark,
				CreateDate:       delivery.CreateDate.Format("2006-01-02T15:04:05Z"),
				CreateBy:         delivery.CreateBy,
				UpdateDate:       delivery.UpdateDate.Format("2006-01-02T15:04:05Z"),
				UpdateBy:         delivery.UpdateBy,
				Items:            items,
			}

			deliveryDataMap[delivery.DeliveryCode] = deliveryData
		}
	}

	// Process packings and inject delivery_data into order_items
	for i := range *packings {
		packing := &(*packings)[i]

		for j := range packing.PackingItem {
			packingItem := &packing.PackingItem[j]

			if packingItem.Outbound != nil && len(packingItem.Outbound.Items) > 0 {
				for k := range packingItem.Outbound.Items {
					outboundItem := &packingItem.Outbound.Items[k]
					orderDocRef := outboundItem.OrderData.DocumentRef

					if len(outboundItem.OrderData.OrderItem) > 0 {
						for l := range outboundItem.OrderData.OrderItem {
							// Check if delivery data exists for this document ref
							if deliveryData, found := deliveryDataMap[orderDocRef]; found {
								// Use JSON manipulation เพื่อเพิ่ม delivery_data
								orderItemBytes, err := json.Marshal(outboundItem.OrderData.OrderItem[l])
								if err != nil {
									fmt.Printf("Error marshaling order item: %v\n", err)
									continue
								}

								var orderItemMap map[string]interface{}
								if err := json.Unmarshal(orderItemBytes, &orderItemMap); err != nil {
									fmt.Printf("Error unmarshaling order item: %v\n", err)
									continue
								}

								orderItemMap["delivery_data"] = deliveryData

								// Convert back และ update struct
								updatedBytes, err := json.Marshal(orderItemMap)
								if err != nil {
									fmt.Printf("Error marshaling updated order item: %v\n", err)
									continue
								}

								if err := json.Unmarshal(updatedBytes, &outboundItem.OrderData.OrderItem[l]); err != nil {
									fmt.Printf("Error unmarshaling updated order item: %v\n", err)
									continue
								}

								fmt.Printf("Added delivery_data for %s to order item %s\n", orderDocRef, outboundItem.OrderData.OrderItem[l].OrderItem)
							}
						}
					}
				}
			}
		}
	}

	fmt.Printf("Successfully processed delivery data mapping\n")
	return nil
}
