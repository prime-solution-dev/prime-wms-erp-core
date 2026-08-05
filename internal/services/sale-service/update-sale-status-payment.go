package saleService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositorySale "prime-erp-core/internal/repositories/sale"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UpdateSaleStatusPaymentReq struct {
	ID            uuid.UUID `json:"id"`
	StatusPayment string    `json:"status_payment"`
}

func UpdateSaleStatusPayment(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []UpdateSaleStatusPaymentReq

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	saleValue := []models.Sale{}

	for i := range req {

		saleValue = append(saleValue, models.Sale{
			ID:            req[i].ID,
			StatusPayment: req[i].StatusPayment,
		})

	}

	rowsAffected, errCreateApproval := repositorySale.UpdateStatusPayment(saleValue)
	if errCreateApproval != nil {
		return nil, errCreateApproval
	}
	if rowsAffected > 0 {
		return map[string]interface{}{
			"status":  "success",
			"message": "Update StatusPayment successfully",
		}, nil
	} else {
		return map[string]interface{}{
			"status":  "success",
			"message": "Update StatusPayment Not Have Rows Affected",
		}, nil
	}
}
