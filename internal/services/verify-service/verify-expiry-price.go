package verifyService

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type VerifyExpiryPriceRequest struct {
	DocumentRef        string    `json:"document_ref"`
	EffectiveDatePrice time.Time `json:"effective_date_price"`
}

type VerifyExpiryPriceResponse struct {
	IsPassVerified     bool      `json:"is_pass_verified"`
	DocumentRef        string    `json:"document_ref"`
	EffectiveDatePrice time.Time `json:"effective_date_price"`
	ExpiryDay          int64     `json:"expiry_day"`
	ExpireDate         time.Time `json:"expire_date"`
}

func VerifyExpiryPrice(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := []VerifyExpiryPriceRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	return VerifyExpiryPriceLogic(gormx, req)
}

func VerifyExpiryPriceLogic(gormx *gorm.DB, req []VerifyExpiryPriceRequest) (*[]VerifyExpiryPriceResponse, error) {
	res := []VerifyExpiryPriceResponse{}

	topic := `PRICE`
	configCodes := []string{`EXPIRY_PRICE_DAYS`}
	configMap, err := GetConfigSystem(gormx, topic, configCodes)
	if err != nil {
		return nil, err
	}

	expiryDaysConfig, exists := configMap[fmt.Sprintf(`%s|%s`, topic, `EXPIRY_PRICE_DAYS`)]
	if !exists {
		return nil, errors.New("missing configuration for expiry price days")
	}

	expiryDays, err := strconv.ParseInt(expiryDaysConfig.Value, 10, 64)
	if err != nil {
		return nil, errors.New("failed to convert expiry days to int64: " + err.Error())
	}

	for _, itemReq := range req {
		expiryDate := time.Now().AddDate(0, 0, int(expiryDays))

		newRes := VerifyExpiryPriceResponse{
			IsPassVerified:     false,
			DocumentRef:        itemReq.DocumentRef,
			EffectiveDatePrice: itemReq.EffectiveDatePrice,
			ExpiryDay:          expiryDays,
			ExpireDate:         expiryDate,
		}

		if newRes.EffectiveDatePrice.Before(expiryDate) {
			newRes.IsPassVerified = true
		} else {
			newRes.IsPassVerified = false
		}

		res = append(res, newRes)
	}

	return &res, nil
}

func GetConfigSystem(gomx *gorm.DB, topics string, configCodes []string) (map[string]models.SystemConfig, error) {
	var configMap map[string]models.SystemConfig

	configMap = make(map[string]models.SystemConfig)
	var configs []models.SystemConfig
	if err := gomx.Where("topic_code = ? AND config_code IN ?", topics, configCodes).Find(&configs).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	for _, config := range configs {
		key := fmt.Sprintf(`%s|%s`, config.TopicCode, config.ConfigCode)
		configMap[key] = config
	}

	return configMap, nil
}
