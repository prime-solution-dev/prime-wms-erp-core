package deliveryService

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	externalService "prime-erp-core/external/order-service"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetDeliveryCORequest struct {
	DeliveryCode []string `json:"delivery_code"`
	DeliveryItem []string `json:"delivery_item"`
}

func (GetDeliveryCOResponse) TableName() string { return "delivery_booking" }

func (GetDeliveryItemCOResponse) TableName() string { return "delivery_booking_item" }

type GetDeliveryCOResponse struct {
	ID               uuid.UUID                   `gorm:"type:uuid;primary_key" json:"id"`
	DeliveryCode     string                      `gorm:"type:varchar(50)" json:"delivery_code"`
	CompanyCode      string                      `gorm:"type:varchar(50)" json:"company_code"`
	SiteCode         string                      `gorm:"type:varchar(50)" json:"site_code"`
	DeliveryMethod   string                      `gorm:"type:varchar(50)" json:"delivery_method"`
	DocumentRef      string                      `gorm:"type:varchar(50)" json:"document_ref"`
	CustomerCode     string                      `gorm:"type:varchar(50)" json:"customer_code"`
	ShipToAddress    string                      `gorm:"type:varchar(255)" json:"ship_to_address"`
	DeliveryDate     *time.Time                  `gorm:"type:date" json:"delivery_date"`
	DeliveryTimeCode string                      `gorm:"type:varchar(50)" json:"delivery_time_code"`
	LicensePlate     string                      `gorm:"type:varchar(50)" json:"license_plate"`
	ContactName      string                      `gorm:"type:varchar(100)" json:"contact_name"`
	Tel              string                      `gorm:"type:varchar(20)" json:"tel"`
	TotalWeight      float64                     `gorm:"type:numeric" json:"total_weight"`
	Status           string                      `gorm:"type:varchar(50)" json:"status"`
	Remark           string                      `gorm:"type:varchar(255)" json:"remark"`
	BookingSlotType  string                      `gorm:"type:varchar(50)" json:"booking_slot_type"`
	CreateDate       *time.Time                  `gorm:"type:date" json:"create_date"`
	CreateBy         string                      `gorm:"type:varchar(50)" json:"create_by"`
	UpdateDate       *time.Time                  `gorm:"type:date" json:"update_date"`
	UpdateBy         string                      `gorm:"type:varchar(50)" json:"update_by"`
	Items            []GetDeliveryItemCOResponse `gorm:"foreignKey:DeliveryID" json:"items"`
}

type GetDeliveryItemCOResponse struct {
	ID              uuid.UUID                                `gorm:"type:uuid;primary_key" json:"id"`
	DeliveryItem    string                                   `gorm:"type:varchar(50)" json:"delivery_item"`
	DeliveryID      uuid.UUID                                `gorm:"type:uuid" json:"delivery_id"`
	ProductCode     string                                   `gorm:"type:varchar(50)" json:"product_code"`
	Qty             float64                                  `gorm:"type:numeric" json:"qty"`
	UnitCode        string                                   `gorm:"type:varchar(20)" json:"unit_code"`
	PriceListUnit   float64                                  `gorm:"type:numeric" json:"price_list_unit"`
	SaleQty         float64                                  `gorm:"type:numeric" json:"sale_qty"`
	SaleUnitCode    string                                   `gorm:"type:varchar(20)" json:"sale_unit_code"`
	TotalWeight     float64                                  `gorm:"type:numeric" json:"total_weight"`
	DocumentRefItem string                                   `gorm:"type:varchar(50)" json:"document_ref_item"`
	Status          string                                   `gorm:"type:varchar(50)" json:"status"`
	Weight          float64                                  `gorm:"type:numeric" json:"weight"`
	WeightUnit      float64                                  `gorm:"type:numeric" json:"weight_unit"`
	Remark          string                                   `gorm:"type:varchar(255)" json:"remark"`
	CreateDate      *time.Time                               `gorm:"type:date" json:"create_date"`
	CreateBy        string                                   `gorm:"type:varchar(50)" json:"create_by"`
	UpdateDate      *time.Time                               `gorm:"type:date" json:"update_date"`
	UpdateBy        string                                   `gorm:"type:varchar(50)" json:"update_by"`
	Sale            models.Sale                              `gorm:"-" json:"sale"`
	Order           externalService.GetOrderDeliveryResponse `gorm:"-" json:"order"`
}

func GetDeliveryCO(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var res []GetDeliveryCOResponse
	var req GetDeliveryCORequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {

		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to database"})
		return nil, err
	}
	defer db.CloseGORM(gormx)

	query := gormx.Preload("Items")

	if len(req.DeliveryCode) > 0 {
		query = query.Where("delivery_code IN ?", req.DeliveryCode)
	}

	if len(req.DeliveryItem) > 0 {
		query = query.Joins("JOIN delivery_booking_item ON delivery_booking.id = delivery_booking_item.delivery_id").
			Where("delivery_booking_item.delivery_item IN ?", req.DeliveryItem)
	}

	if err := query.Find(&res).Error; err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve data"})
		return nil, err
	}

	// Collect unique documentRef values
	uniqueDocumentRefs := make(map[string]bool)
	var documentRefList []string

	for _, delivery := range res {
		if delivery.DocumentRef != "" && !uniqueDocumentRefs[delivery.DocumentRef] {
			uniqueDocumentRefs[delivery.DocumentRef] = true
			documentRefList = append(documentRefList, delivery.DocumentRef)
		}
	}

	// Query sales for all unique documentRef values at once
	var sales []models.Sale
	if len(documentRefList) > 0 {
		if err := gormx.Where("sale_code IN ?", documentRefList).
			Preload("SaleItem").
			Find(&sales).Error; err != nil {
			fmt.Println("Error fetching sales:", err)
		}
	}

	// Now find all delivery bookings for these sales
	var allDeliveries []GetDeliverySOResponse
	if len(sales) > 0 {
		// Collect all sale codes
		var saleCodes []string
		for _, sale := range sales {
			saleCodes = append(saleCodes, sale.SaleCode)
		}

		// Query all delivery bookings where documentRef matches sale codes
		if err := gormx.Preload("Items").
			Where("document_ref IN ?", saleCodes).
			Find(&allDeliveries).Error; err != nil {
			fmt.Println("Error fetching delivery bookings:", err)
			return res, nil
		}

		// Create maps for efficient lookup
		salesMap := make(map[string]models.Sale)
		saleItemsMap := make(map[string][]models.SaleItem)

		for _, sale := range sales {
			salesMap[sale.SaleCode] = sale
			saleItemsMap[sale.SaleCode] = sale.SaleItem
		}

		// Map sales to delivery items based on the criteria:
		// delivery_booking.documentRef = sale_code
		// delivery_booking_item.documentRefItem = sale_item
		for i := range allDeliveries {
			delivery := &allDeliveries[i]

			// Check if this delivery's documentRef matches any sale code
			if sale, exists := salesMap[delivery.DocumentRef]; exists {
				// Process each delivery item
				for j := range delivery.Items {
					deliveryItem := &delivery.Items[j]

					// Create a copy of sale for this delivery item
					itemSale := sale
					itemSale.SaleItem = []models.SaleItem{}

					// Filter saleItems: deliveryItem.documentRefItem == saleItem.SaleItem
					for _, saleItem := range saleItemsMap[delivery.DocumentRef] {
						if deliveryItem.DocumentRefItem == saleItem.SaleItem {
							itemSale.SaleItem = append(itemSale.SaleItem, saleItem)
						}
					}

					deliveryItem.Sale = itemSale
				}
			}
		}

		// GetOrderDelivery
		orderDeliveryResponse, err := GetOrderDelivery(allDeliveries)
		if err != nil {
			fmt.Println("Error in GetOrderDelivery:", err)
			return res, nil
		}

		// Map orders from orderDeliveryResponse to res items
		// Create map for efficient lookup of orders by delivery details
		orderMap := make(map[string]map[string]externalService.GetOrderDeliveryResponse)

		for _, order := range orderDeliveryResponse.Orders {
			// Group orders by delivery_code
			if orderMap[order.DocumentRef] == nil {
				orderMap[order.DocumentRef] = make(map[string]externalService.GetOrderDeliveryResponse)
			}

			// For each order item, create mapping by delivery_item (order_item)
			for _, orderItem := range order.OrderItem {
				orderMap[order.DocumentRef][orderItem.DocumentRefItem] = order
			}
		}

		// Map orders to delivery items in res
		for i := range res {
			delivery := &res[i]

			for j := range delivery.Items {
				deliveryItem := &delivery.Items[j]

				// Try to find matching order
				// The mapping logic might need adjustment based on your business rules
				// This assumes delivery_code maps to order_code and delivery_item maps to order_item
				if ordersByCode, exists := orderMap[delivery.DeliveryCode]; exists {
					if matchingOrder, itemExists := ordersByCode[deliveryItem.DeliveryItem]; itemExists {
						deliveryItem.Order = matchingOrder
					}
				}
			}
		}

		// Map sales to delivery items in res
		for i := range res {
			delivery := &res[i]

			for j := range delivery.Items {
				deliveryItem := &delivery.Items[j]

				// Find matching sale from allDeliveries
				for _, allDelivery := range allDeliveries {
					if allDelivery.DeliveryCode == delivery.DeliveryCode {
						for _, allDeliveryItem := range allDelivery.Items {
							if allDeliveryItem.DeliveryItem == deliveryItem.DeliveryItem {
								deliveryItem.Sale = allDeliveryItem.Sale
								break
							}
						}
						break
					}
				}
			}
		}

	}

	return res, nil
}

func GetOrderDelivery(allDeliveries []GetDeliverySOResponse) (externalService.ResultOrderDeliveryResponse, error) {
	getOrderRequest := externalService.GetOrderDeliveryRequest{}
	for _, row := range allDeliveries {
		getOrderRequest.DeliveryCode = append(getOrderRequest.DeliveryCode, row.DeliveryCode)

		for _, item := range row.Items {
			getOrderRequest.DeliveryItem = append(getOrderRequest.DeliveryItem, item.DeliveryItem)
		}
	}

	fmt.Println("getOrderRequest : ", getOrderRequest)
	getOrderResponse, err := externalService.GetOrdersDelivery(getOrderRequest)
	if err != nil {
		return externalService.ResultOrderDeliveryResponse{}, errors.New("Error get outbound : " + err.Error())
	}
	fmt.Println("getOrderResponse : ", getOrderResponse)

	return getOrderResponse, nil
}
