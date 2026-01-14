package quotationService

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CancelQuotationRequest struct {
	ID uuid.UUID `json:"id"`
}

func CancelQuotation(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := CancelQuotationRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Update quotation status to CANCELED
	updateFields := map[string]interface{}{
		"status":      "CANCELED",
		"update_date": nowDateOnly,
	}

	if err := gormx.Model(&models.Quotation{}).
		Where("id = ?", req.ID).
		Updates(updateFields).Error; err != nil {
		return nil, fmt.Errorf("failed to update quotation status: %v", err)
	}

	// Also update quotation_item status to CANCELED
	if err := gormx.Model(&models.QuotationItem{}).
		Where("quotation_id = ?", req.ID).
		Updates(map[string]interface{}{
			"status":      "CANCELED",
			"update_date": nowDateOnly,
		}).Error; err != nil {
		return nil, fmt.Errorf("failed to update quotation items status: %v", err)
	}

	return map[string]interface{}{
		"status":  "success",
		"message": "Quotation canceled successfully",
	}, nil
}
