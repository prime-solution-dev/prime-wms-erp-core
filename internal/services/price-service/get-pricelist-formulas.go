package priceService

import (
	"context"

	"prime-erp-core/internal/models"
	priceListRepository "prime-erp-core/internal/repositories/priceList"
)

// GetPriceListFormulas returns all price list formulas, ordered from newest to oldest.
func GetPriceListFormulas(_ context.Context, _ string) (interface{}, error) {
	formulas, err := priceListRepository.GetPriceListFormulas()
	if err != nil {
		return nil, err
	}

	if formulas == nil {
		formulas = []models.PriceListFormulas{}
	}

	return formulas, nil
}
