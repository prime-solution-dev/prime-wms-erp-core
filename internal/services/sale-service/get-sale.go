package saleService

import (
	"encoding/json"
	"errors"

	"prime-erp-core/internal/db"
	models "prime-erp-core/internal/models"
	repositorySale "prime-erp-core/internal/repositories/sale"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetSaleRequest struct {
	ID                   []uuid.UUID `json:"id"`
	SaleCode             []string    `json:"sale_code"`
	CustomerCode         []string    `json:"customer_code"`
	Status               []string    `json:"status"`
	StatusApprove        []string    `json:"status_approve"`
	StatusPayment        []string    `json:"status_payment"`
	ProductCode          []string    `json:"product_code"`
	SiteCode             []string    `json:"site_code"`
	CompanyCode          []string    `json:"company_code"`
	IsApproved           []bool      `json:"is_approved"`
	IsAvailableQty       bool        `json:"is_available_qty"`
	SaleCodeLike         string      `json:"sale_code_like"`
	DocumentRefLike      string      `json:"document_ref_like"`
	CompletedDateStart   string      `json:"completed_date_start"`
	CompletedDateEnd     string      `json:"completed_date_end"`
	CustomerCodeLike     string      `json:"customer_code_like"`
	CustomerNameLike     string      `json:"customer_name_like"`
	CreateDateStart      string      `json:"create_date_start"`
	CreateDateEnd        string      `json:"create_date_end"`
	ProductCodeLike      string      `json:"product_code_like"`
	ExpirePriceDateStart string      `json:"expire_price_date_start"`
	ExpirePriceDateEnd   string      `json:"expire_price_date_end"`
	DeliveryDateStart    string      `json:"delivery_date_start"`
	DeliveryDateEnd      string      `json:"delivery_date_end"`
	StatusFilter         []string    `json:"status_filter"`
	Page                 int         `json:"page"`
	PageSize             int         `json:"page_size"`
}
type ResultSale struct {
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
	Sale       []models.Sale `json:"sale"`
}

func GetSale(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetSaleRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	if req.IsAvailableQty {
		// If filtering by available qty, get all data first (no pagination)
		// then filter and apply pagination manually
		return getSaleWithAvailableQtyFilter(req)
	}

	// Normal flow without qty filtering - use repository
	sale, totalPages, totalRecords, errApproval := repositorySale.GetSalePreload(
		req.CompanyCode,
		req.SiteCode,
		req.ID,
		req.SaleCode,
		req.CustomerCode,
		req.Status,
		req.StatusApprove,
		req.StatusPayment,
		req.ProductCode,
		req.IsApproved,
		req.SaleCodeLike,
		req.DocumentRefLike,
		req.CompletedDateStart,
		req.CompletedDateEnd,
		req.CustomerCodeLike,
		req.CustomerNameLike,
		req.CreateDateStart,
		req.CreateDateEnd,
		req.ProductCodeLike,
		req.ExpirePriceDateStart,
		req.ExpirePriceDateEnd,
		req.DeliveryDateStart,
		req.DeliveryDateEnd,
		req.StatusFilter,
		req.Page,
		req.PageSize)
	if errApproval != nil {
		return nil, errApproval
	}

	resultSale := ResultSale{
		Total:      totalRecords,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		Sale:       sale,
	}

	return resultSale, nil
}

func getSaleWithAvailableQtyFilter(req GetSaleRequest) (interface{}, error) {
	// Get all sales without pagination first
	sale, _, _, errApproval := repositorySale.GetSalePreload(
		req.CompanyCode,
		req.SiteCode,
		req.ID,
		req.SaleCode,
		req.CustomerCode,
		req.Status,
		req.StatusApprove,
		req.StatusPayment,
		req.ProductCode,
		req.IsApproved,
		req.SaleCodeLike,
		req.DocumentRefLike,
		req.CompletedDateStart,
		req.CompletedDateEnd,
		req.CustomerCodeLike,
		req.CustomerNameLike,
		req.CreateDateStart,
		req.CreateDateEnd,
		req.ProductCodeLike,
		req.ExpirePriceDateStart,
		req.ExpirePriceDateEnd,
		req.DeliveryDateStart,
		req.DeliveryDateEnd,
		req.StatusFilter,
		1, 0) // pageSize=0 means get all
	if errApproval != nil {
		return nil, errApproval
	}

	// Filter sales by available qty (remove items with 0 remaining qty)
	filteredSales, err := filterSalesByAvailableQty(sale)
	if err != nil {
		return nil, err
	}

	// Apply pagination to filtered results
	totalRecordsAfterFilter := len(filteredSales)
	totalPages := 1
	if req.PageSize > 0 {
		totalPages = int((float64(totalRecordsAfterFilter) + float64(req.PageSize) - 1) / float64(req.PageSize))
	}

	// Apply pagination
	startIndex := 0
	endIndex := totalRecordsAfterFilter

	if req.PageSize > 0 && req.Page > 0 {
		startIndex = (req.Page - 1) * req.PageSize
		endIndex = startIndex + req.PageSize

		if startIndex >= totalRecordsAfterFilter {
			// Page out of range, return empty
			filteredSales = []models.Sale{}
		} else {
			if endIndex > totalRecordsAfterFilter {
				endIndex = totalRecordsAfterFilter
			}
			filteredSales = filteredSales[startIndex:endIndex]
		}
	}

	resultSale := ResultSale{
		Total:      totalRecordsAfterFilter, // Use filtered total
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		Sale:       filteredSales,
	}

	return resultSale, nil
}

// filterSalesByAvailableQty filters sales that still have available qty
func filterSalesByAvailableQty(sales []models.Sale) ([]models.Sale, error) {
	if len(sales) == 0 {
		return sales, nil
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	// Extract sale codes for delivery query
	saleCodes := make([]string, len(sales))
	for i, sale := range sales {
		saleCodes[i] = sale.SaleCode
	}

	// Get all deliveries for these sales with PENDING or COMPLETED status
	var deliveries []models.Delivery
	err = gormx.Where("document_ref IN ? AND status IN ?", saleCodes, []string{"PENDING", "COMPLETED"}).
		Find(&deliveries).Error
	if err != nil {
		return nil, err
	}

	// Get delivery IDs for querying delivery items
	deliveryIDs := make([]uuid.UUID, len(deliveries))
	deliveryDocRefMap := make(map[uuid.UUID]string) // Map delivery ID to document_ref
	for i, delivery := range deliveries {
		deliveryIDs[i] = delivery.ID
		deliveryDocRefMap[delivery.ID] = delivery.DocumentRef
	}

	// Get all delivery items for these deliveries
	var allDeliveryItems []models.DeliveryItem
	if len(deliveryIDs) > 0 {
		err = gormx.Where("delivery_id IN ?", deliveryIDs).Find(&allDeliveryItems).Error
		if err != nil {
			return nil, err
		}
	}

	// Create a map of sale_code -> delivery items for quick lookup
	deliveryMap := make(map[string][]models.DeliveryItem)
	for _, deliveryItem := range allDeliveryItems {
		if docRef, exists := deliveryDocRefMap[deliveryItem.DeliveryID]; exists {
			deliveryMap[docRef] = append(deliveryMap[docRef], deliveryItem)
		}
	}

	// Filter sales that have available qty
	var filteredSales []models.Sale
	for _, sale := range sales {
		var availableSaleItems []models.SaleItem
		deliveryItems, hasDeliveries := deliveryMap[sale.SaleCode]

		for _, saleItem := range sale.SaleItem {
			usedQty := 0.0

			// Calculate used qty from deliveries
			if hasDeliveries {
				for _, deliveryItem := range deliveryItems {
					if deliveryItem.DocumentRefItem == saleItem.SaleItem {
						usedQty += deliveryItem.Qty
					}
				}
			}

			// Check if there's remaining qty
			remainingQty := saleItem.Qty - usedQty
			if remainingQty > 0 {
				availableSaleItems = append(availableSaleItems, saleItem)
			}
		}

		// Only include sales that have at least one item with available qty > 0
		if len(availableSaleItems) > 0 {
			sale.SaleItem = availableSaleItems // Update with filtered sale items
			filteredSales = append(filteredSales, sale)
		}
	}

	return filteredSales, nil
}
