package priceService

import (
	"encoding/json"
	"fmt"
	"prime-erp-core/internal/models"
	priceListRepository "prime-erp-core/internal/repositories/priceList"
	"prime-erp-core/internal/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// seams for unit testing: allow stubbing the cascade dependencies
var getPriceListGroupCodesByIDsFunc = priceListRepository.GetPriceListGroupCodesByIDs
var runUpdateLatestSubGroupFunc = RunUpdateLatestPriceListSubGroup

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

	// The detail pages read their prices from price_list_sub_group, which is not
	// touched above. Recalculate them here so the new base price is stamped as the
	// "after" price (and the previous "after" moves to "before") right away,
	// instead of only when someone happens to hit /price/SubGroup/UpdateLatest.
	if err := cascadeBasePriceToSubGroups(priceListGroup); err != nil {
		return nil, err
	}

	return nil, nil
}

func cascadeBasePriceToSubGroups(priceListGroup []models.PriceListGroup) error {
	ids := make([]uuid.UUID, 0, len(priceListGroup))
	for _, group := range priceListGroup {
		ids = append(ids, group.ID)
	}

	groupCodes, err := getPriceListGroupCodesByIDsFunc(ids)
	if err != nil {
		return fmt.Errorf("failed to resolve group codes for base price cascade: %w", err)
	}
	if len(groupCodes) == 0 {
		return nil
	}

	if _, err := runUpdateLatestSubGroupFunc(models.UpdateLatestPriceListSubGroupRequest{
		UpdateType: "group",
		GroupCodes: groupCodes,
	}); err != nil {
		return fmt.Errorf("failed to cascade base price to sub groups: %w", err)
	}

	return nil
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
		case ">":
			// Range from min to infinity, same as >= (only min is meaningful)
			return e.CondRangeMin, 1e18
		case "<":
			// Range from min to max, same as <= (only max is meaningful)
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

// validExtraOperators คือ operator ที่ extraConditionMatched รองรับ
// (update-latest-pricelist-subgroup.go) ซึ่งเป็นตัวตัดสินราคาจริง
// ห้ามใช้ getEffectiveRange เป็นแหล่งความจริง มันเป็นแค่ helper ของการตรวจ overlap
// และ default ของมันกลืน operator ที่ไม่รู้จักไป
var validExtraOperators = map[string]bool{
	"=":  true,
	">=": true,
	"<=": true,
	"<":  true,
	">":  true,
	"<>": true,
}

// validateExtras ตรวจว่าข้อมูล extra ที่ส่งมาไม่มี field ว่างที่จำเป็น
//
// route /UpdatePriceListExtra ใช้ utils.ProcessRequest ซึ่งไม่ผ่าน ShouldBindJSON
// ฉะนั้น binding:"required" tag ใช้ไม่ได้ ต้องตรวจด้วยโค้ดตรง ๆ
// คืน *utils.BindingError เพื่อให้ ProcessRequest แปลงเป็น HTTP 400
func validateExtras(extras []models.UpdatePriceListExtraRequest) error {
	if len(extras) == 0 {
		return &utils.BindingError{Message: "ต้องส่งข้อมูล extra มาอย่างน้อย 1 รายการ"}
	}

	for i, e := range extras {
		if e.PriceListGroupID == uuid.Nil {
			return &utils.BindingError{
				Message: fmt.Sprintf("รายการที่ %d: price_list_group_id ห้ามว่าง", i+1),
			}
		}
		if strings.TrimSpace(e.ExtraKey) == "" {
			return &utils.BindingError{
				Message: fmt.Sprintf("รายการที่ %d: extra_key ห้ามว่าง", i+1),
			}
		}
		if strings.TrimSpace(e.ConditionCode) == "" {
			return &utils.BindingError{
				Message: fmt.Sprintf("รายการที่ %d: condition_code ห้ามว่าง", i+1),
			}
		}
		if !validExtraOperators[strings.TrimSpace(e.Operator)] {
			return &utils.BindingError{
				Message: fmt.Sprintf("รายการที่ %d: operator %q ไม่ถูกต้อง ต้องเป็น =, >=, <=, <, > หรือ <>", i+1, e.Operator),
			}
		}
		// เฉพาะ "<>" เท่านั้นที่ใช้ทั้ง min และ max พร้อมกัน (extraConditionMatched:
		// val >= min && val <= max) operator อื่นใช้ขอบเดียว (เช่น ">=" ใช้แค่ min)
		// ค่าอีกขอบไม่มีความหมายและอาจเป็นข้อมูลเก่าที่ถูกต้องอยู่แล้ว (เช่น min=100, max=0)
		if strings.TrimSpace(e.Operator) == "<>" && e.CondRangeMin > e.CondRangeMax {
			return &utils.BindingError{
				Message: fmt.Sprintf("รายการที่ %d: cond_range_min (%v) ต้องไม่มากกว่า cond_range_max (%v)", i+1, e.CondRangeMin, e.CondRangeMax),
			}
		}
		if len(e.PriceListGroupExtraKeys) == 0 {
			return &utils.BindingError{
				Message: fmt.Sprintf("รายการที่ %d: price_list_group_extra_keys ห้ามว่าง", i+1),
			}
		}
		for j, k := range e.PriceListGroupExtraKeys {
			if strings.TrimSpace(k.Code) == "" {
				return &utils.BindingError{
					Message: fmt.Sprintf("รายการที่ %d key ที่ %d: code ห้ามว่าง", i+1, j+1),
				}
			}
			if strings.TrimSpace(k.Value) == "" {
				return &utils.BindingError{
					Message: fmt.Sprintf("รายการที่ %d key ที่ %d: value ห้ามว่าง", i+1, j+1),
				}
			}
		}
	}

	return nil
}

func UpdateExtras(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := []models.UpdatePriceListExtraRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, &utils.BindingError{Message: "payload ไม่ใช่ JSON ที่ถูกต้อง: " + err.Error()}
	}

	if err := validateExtras(req); err != nil {
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
