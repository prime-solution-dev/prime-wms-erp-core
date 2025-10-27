package verifyService

import priceService "prime-erp-core/internal/services/price-service"

func VerifyPrice(compareReq priceService.GetComparePriceRequest) (*priceService.GetComparePriceResponse, error) {
	compareRes, err := priceService.ComparePrice(compareReq)
	if err != nil {
		return nil, err
	}

	return &compareRes, nil
}
