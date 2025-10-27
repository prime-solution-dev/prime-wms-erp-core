package priceService

import (
	"encoding/json"
	"errors"
	"prime-erp-core/internal/models"
	priceListRepository "prime-erp-core/internal/repositories/priceList"

	"github.com/gin-gonic/gin"
)

func DeletePriceListBase(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := models.DeletePriceListBaseRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, err
	}

	if len(req.ID) == 0 {
		return nil, errors.New("id is required")
	}

	if err := priceListRepository.DeletePriceListBase(req.ID); err != nil {
		return nil, err
	}

	return nil, nil
}
