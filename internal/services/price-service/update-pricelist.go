package priceService

import (
	"encoding/json"
	"prime-erp-core/internal/models"
	priceListRepository "prime-erp-core/internal/repositories/priceList"
	"time"

	"github.com/gin-gonic/gin"
)

func UpdatePriceListBase(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := []models.UpdatePriceListBaseRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, err
	}

	priceListGroup := []models.PriceListGroup{}
	priceListGroupTerm := []models.PriceListGroupTerm{}
	for _, r := range req {
		effectiveDate := time.Now().UTC()
		if r.EffectiveDate != nil {
			effectiveDate = r.EffectiveDate.UTC()
		}

		now := time.Now().UTC()

		priceListGroup = append(priceListGroup, models.PriceListGroup{
			ID:            r.ID,
			PriceUnit:     r.PriceUnit,
			PriceWeight:   r.PriceWeight,
			Currency:      r.Currency,
			EffectiveDate: &effectiveDate,
			Remark:        r.Remark,
			UpdateBy:      "system", // TODO: get user from auth
			UpdateDtm:     now,
		})

		if len(r.Terms) > 0 {
			termNow := time.Now().UTC()
			for _, term := range r.Terms {
				priceListGroupTerm = append(priceListGroupTerm, models.PriceListGroupTerm{
					ID:         term.ID,
					Pdc:        term.Pdc,
					PdcPercent: term.PdcPercent,
					Due:        term.Due,
					DuePercent: term.DuePercent,
					UpdateBy:   "system", // TODO: get user from auth
					UpdateDtm:  termNow,
				})
			}
		}
	}

	if err := priceListRepository.UpdatePriceListBase(priceListGroup); err != nil {
		return nil, err
	}

	if err := priceListRepository.UpdatePriceListTerm(priceListGroupTerm); err != nil {
		return nil, err
	}

	return nil, nil
}
