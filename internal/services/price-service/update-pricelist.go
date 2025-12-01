package priceService

import (
	"encoding/json"
	"prime-erp-core/internal/models"
	priceListRepository "prime-erp-core/internal/repositories/priceList"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
					ID:               term.ID,
					PriceListGroupID: r.ID,
					TermCode:         term.TermCode,
					Pdc:              term.Pdc,
					PdcPercent:       term.PdcPercent,
					Due:              term.Due,
					DuePercent:       term.DuePercent,
					CreateBy:         term.CreateBy,
					CreateDtm:        term.CreateDtm,
					UpdateBy:         "system", // TODO: get user from auth
					UpdateDtm:        &termNow,
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

func UpdateExtras(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := []models.UpdatePriceListExtraRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, err
	}

	extras := []models.PriceListGroupExtra{}
	for _, r := range req {
		now := time.Now().UTC()

		var id uuid.UUID
		if r.ID == nil {
			id = uuid.New()
		} else {
			id = *r.ID
		}

		extraKeys := []models.PriceListGroupExtraKey{}
		for _, extraKey := range r.PriceListGroupExtraKeys {
			var keyId uuid.UUID
			if extraKey.ID == nil {
				keyId = uuid.New()
			} else {
				keyId = *extraKey.ID
			}

			extraKeys = append(extraKeys, models.PriceListGroupExtraKey{
				ID:           keyId,
				GroupExtraID: id,
				Code:         extraKey.Code,
				Value:        extraKey.Value,
				Seq:          extraKey.Seq,
			})
		}

		extras = append(extras, models.PriceListGroupExtra{
			ID:                      id,
			PriceListGroupID:        r.PriceListGroupID,
			ExtraKey:                r.ExtraKey,
			ConditionCode:           r.ConditionCode,
			ValueInt:                r.ValueInt,
			LengthExtraKey:          r.LengthExtraKey,
			Operator:                r.Operator,
			CondRangeMin:            r.CondRangeMin,
			CondRangeMax:            r.CondRangeMax,
			CreateBy:                r.CreateBy,
			CreateDtm:               &r.CreateDtm,
			UpdateBy:                "system", // TODO: get user from auth
			UpdateDtm:               &now,
			PriceListGroupExtraKeys: extraKeys,
		})
	}

	if err := priceListRepository.UpdateExtra(extras); err != nil {
		return nil, err
	}

	return nil, nil
}
