package priceService

import (
	"encoding/json"
	"fmt"
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

// checkForOverlappingConditions validates that no conditions overlap
// Returns an error if overlapping conditions are found
func checkForOverlappingConditions(extras []models.UpdatePriceListExtraRequest) error {
	// Group extras by price_list_group_id, extra_key, and condition_code
	groups := make(map[string][]models.UpdatePriceListExtraRequest)

	for _, extra := range extras {
		key := extra.PriceListGroupID.String() + "|" + extra.ExtraKey + "|" + extra.ConditionCode
		groups[key] = append(groups[key], extra)
	}

	// Helper function to get effective range based on operator
	getEffectiveRange := func(e models.UpdatePriceListExtraRequest) (min, max float64) {
		switch e.Operator {
		case "<=":
			// Range from min to max (e.g., <= 45 means [0, 45])
			return e.CondRangeMin, e.CondRangeMax
		case ">=":
			// Range from min to infinity (treated as a very large number)
			return e.CondRangeMin, 1e18
		case "=":
			// Single value, min == max
			return e.CondRangeMin, e.CondRangeMax
		case "<>":
			// Range between min and max
			return e.CondRangeMin, e.CondRangeMax
		default:
			// Default: use the full range
			return e.CondRangeMin, e.CondRangeMax
		}
	}

	// Check each group for overlapping conditions
	for _, groupExtras := range groups {
		// Check for overlaps between all pairs
		for i := 0; i < len(groupExtras); i++ {
			for j := i + 1; j < len(groupExtras); j++ {
				e1 := groupExtras[i]
				e2 := groupExtras[j]

				min1, max1 := getEffectiveRange(e1)
				min2, max2 := getEffectiveRange(e2)

				// Overlap condition: min1 <= max2 && min2 <= max1
				if min1 <= max2 && min2 <= max1 {
					return fmt.Errorf(
						"overlapping condition detected: price_list_group_id=%s, condition_code=%s. "+
							"Conflicting ranges: operator=%s [%.2f, %.2f] and operator=%s [%.2f, %.2f]",
						groupExtras[0].PriceListGroupID,
						e1.ConditionCode,
						e1.Operator, min1, max1,
						e2.Operator, min2, max2,
					)
				}
			}
		}
	}

	return nil
}

func UpdateExtras(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := []models.UpdatePriceListExtraRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, err
	}

	// Validate for overlapping conditions
	if err := checkForOverlappingConditions(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
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
