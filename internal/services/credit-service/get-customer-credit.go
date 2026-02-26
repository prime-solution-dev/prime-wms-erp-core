package creditService

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"prime-erp-core/internal/db"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GetCustomerCreditRequest struct {
	IsActive *bool `json:"is_active"`
}

type GetCustomerCreditResponse struct {
	ResponseCode int                             `json:"response_code"`
	Message      string                          `json:"message"`
	Data         GetCustomerCreditResponseResult `json:"data"`
}

type GetCustomerCreditResponseResult struct {
	CustomerCodes []string `json:"customer_codes"`
}

func GetCustomerCreditRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := GetCustomerCreditRequest{}

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

	return GetCustomerCredit(gormx, req)
}

func GetCustomerCredit(gormx *gorm.DB, req GetCustomerCreditRequest) (*GetCustomerCreditResponse, error) {
	res := &GetCustomerCreditResponse{
		ResponseCode: 200,
		Message:      "success",
		Data: GetCustomerCreditResponseResult{
			CustomerCodes: []string{},
		},
	}

	q := gormx.Table("credit").
		Select("DISTINCT customer_code").
		Where("customer_code IS NOT NULL").
		Where("customer_code <> ''")

	if req.IsActive != nil {
		q = q.Where("is_active = ?", *req.IsActive)
	}

	var codes []string
	if err := q.Pluck("customer_code", &codes).Error; err != nil {
		return nil, errors.New("query customer credits failed: " + err.Error())
	}

	res.Data.CustomerCodes = codes
	return res, nil
}
