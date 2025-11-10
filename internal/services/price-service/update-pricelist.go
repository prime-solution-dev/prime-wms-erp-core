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
	for _, r := range req {

		now := time.Now().UTC()

		priceListGroupTerm := []models.PriceListGroupTerm{}
		if len(r.Terms) > 0 {
			termNow := time.Now().UTC()
			for _, term := range r.Terms {
				priceListGroupTerm = append(priceListGroupTerm, models.PriceListGroupTerm{
					ID:         term.ID,
					Pdc:        term.Pdc,
					PdcPercent: term.PdcPercent,
					Due:        term.Due,
					DuePercent: term.DuePercent,
					CreateBy:   term.CreateBy,
					CreateDtm:  term.CreateDtm,
					UpdateBy:   "system", // TODO: get user from auth
					UpdateDtm:  &termNow,
				})
			}
		}

		priceListGroup = append(priceListGroup, models.PriceListGroup{
			ID:                  r.ID,
			PriceUnit:           r.PriceUnit,
			PriceWeight:         r.PriceWeight,
			Currency:            r.Currency,
			EffectiveDate:       r.EffectiveDate,
			Remark:              r.Remark,
			UpdateBy:            "system", // TODO: get user from auth
			UpdateDtm:           now,
			PriceListGroupTerms: priceListGroupTerm,
		})

	}

	if err := priceListRepository.UpdatePriceListBase(priceListGroup); err != nil {
		return nil, err
	}

	return nil, nil
}
