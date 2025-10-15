package deliveryService

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CreateDeliveryRequest struct {
	CompanyCode      string                       `json:"company_code"`
	SiteCode         string                       `json:"site_code"`
	DeliveryMethod   string                       `json:"delivery_method"`
	DocumentRef      string                       `json:"document_ref"`
	CustomerCode     string                       `json:"customer_code"`
	ShipToAddress    string                       `json:"ship_to_address"`
	DeliveryDate     *time.Time                   `json:"delivery_date"`
	DeliveryTimeCode string                       `json:"delivery_time_code"`
	LicensePlate     string                       `json:"license_plate"`
	ContactName      string                       `json:"contact_name"`
	Tel              string                       `json:"tel"`
	TotalWeight      float64                      `json:"total_weight"`
	Remark           string                       `json:"remark"`
	DeliveryItems    []CreateDeliveryItemsRequest `json:"delivery_items"`
}

type CreateDeliveryItemsRequest struct {
	ProductCode string  `json:"product_code"`
	Qty         float64 `json:"qty"`
	UnitCode    string  `json:"unit_code"`
	Weight      float64 `json:"weight"`
	WeightUnit  float64 `json:"weight_unit"`
}

func CreateDelivery(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req []CreateDeliveryRequest

	// Bind JSON payload
	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	// Connect to the database
	gormx, err := db.ConnectGORM("prime_erp")
	defer db.CloseGORM(gormx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to database"})
		return nil, err
	}

	tx := gormx.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	user := "SYSTEM" // TODO: get from ctx
	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	deliveryToAdd := []models.Delivery{}
	deliveryItemToAdd := []models.DeliveryItem{}

	for num, deliveryReq := range req {
		deliveryId := uuid.New()

		// แปลง DeliveryDate เป็น date-only format
		var deliveryDateOnly *time.Time
		if deliveryReq.DeliveryDate != nil {
			dateOnly := time.Date(deliveryReq.DeliveryDate.Year(), deliveryReq.DeliveryDate.Month(), deliveryReq.DeliveryDate.Day(), 0, 0, 0, 0, deliveryReq.DeliveryDate.Location())
			deliveryDateOnly = &dateOnly
		}

		newDelivery := models.Delivery{
			ID:               deliveryId,
			DeliveryCode:     "DELIVERY-" + time.Now().Format("20060102150405") + fmt.Sprintf("%d", num),
			CompanyCode:      deliveryReq.CompanyCode,
			SiteCode:         deliveryReq.SiteCode,
			DeliveryMethod:   deliveryReq.DeliveryMethod,
			DocumentRef:      deliveryReq.DocumentRef,
			CustomerCode:     deliveryReq.CustomerCode,
			ShipToAddress:    deliveryReq.ShipToAddress,
			DeliveryDate:     deliveryDateOnly, // ใช้ date-only
			DeliveryTimeCode: deliveryReq.DeliveryTimeCode,
			LicensePlate:     deliveryReq.LicensePlate,
			ContactName:      deliveryReq.ContactName,
			Tel:              deliveryReq.Tel,
			TotalWeight:      deliveryReq.TotalWeight,
			Remark:           deliveryReq.Remark,
			Status:           "PENDING",
			CreateBy:         user,
			CreateDate:       nowDateOnly, // date-only format
			UpdateBy:         user,
			UpdateDate:       nowDateOnly, // date-only format
		}

		deliveryToAdd = append(deliveryToAdd, newDelivery)

		for numItem, deliveryItem := range deliveryReq.DeliveryItems {
			deliveryItemId := uuid.New()

			newDeliveryItem := models.DeliveryItem{
				ID:           deliveryItemId,
				DeliveryItem: fmt.Sprintf("ITEM-%s-%d", deliveryId.String(), numItem),
				DeliveryID:   deliveryId,
				ProductCode:  deliveryItem.ProductCode,
				Qty:          deliveryItem.Qty,
				UnitCode:     deliveryItem.UnitCode,
				Weight:       deliveryItem.Weight,
				WeightUnit:   deliveryItem.WeightUnit,
				Status:       "PENDING",
				CreateDate:   nowDateOnly, // date-only format
				CreateBy:     user,
				UpdateDate:   nowDateOnly, // date-only format
				UpdateBy:     user,
			}

			deliveryItemToAdd = append(deliveryItemToAdd, newDeliveryItem)
		}
	}

	if len(deliveryToAdd) > 0 {
		if err := tx.Create(&deliveryToAdd).Error; err != nil {
			return nil, err
		}
	}

	if len(deliveryItemToAdd) > 0 {
		if err := tx.Create(&deliveryItemToAdd).Error; err != nil {
			return nil, err
		}
	}

	return gin.H{"status": "success", "message": "Create delivery successfully"}, nil
}
