package priceService

import (
	"encoding/json"
	"prime-erp-core/internal/models"
	priceListRepository "prime-erp-core/internal/repositories/priceList"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreatePriceListBase(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := []models.CreatePriceListBaseRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, err
	}

	priceListGroups := []models.PriceListGroup{}
	for _, r := range req {
		now := time.Now().UTC()

		priceListGroup := models.PriceListGroup{
			ID:                uuid.New(),
			CompanyCode:       r.CompanyCode,
			SiteCode:          r.SiteCode,
			GroupCode:         r.GroupCode,
			PriceUnit:         r.PriceUnit,
			PriceWeight:       r.PriceWeight,
			BeforePriceUnit:   0,
			BeforePriceWeight: 0,
			Currency:          r.Currency,
			EffectiveDate:     r.EffectiveDate,
			Remark:            r.Remark,
			CreateBy:          "system", // TODO: get user from auth
			CreateDtm:         now,
			UpdateBy:          "system", // TODO: get user from auth
			UpdateDtm:         now,
		}

		terms := []models.PriceListGroupTerm{}
		for _, t := range r.Terms {
			term := models.PriceListGroupTerm{
				ID:               uuid.New(),
				PriceListGroupID: priceListGroup.ID,
				TermCode:         t.TermCode,
				Pdc:              t.Pdc,
				PdcPercent:       t.PdcPercent,
				Due:              t.Due,
				DuePercent:       t.DuePercent,
				CreateBy:         "system", // TODO: get user from auth
				CreateDtm:        &now,
				UpdateBy:         "system", // TODO: get user from auth
				UpdateDtm:        &now,
			}
			terms = append(terms, term)
		}
		priceListGroup.PriceListGroupTerms = terms
		priceListGroups = append(priceListGroups, priceListGroup)
	}

	if err := priceListRepository.CreatePriceListBase(priceListGroups); err != nil {
		return nil, err
	}

	return nil, nil
}
