package priceService

import (
	"encoding/json"
	"fmt"
	"math"
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

	// Check each group for overlapping conditions
	for _, groupExtras := range groups {
		// Check for overlaps between all pairs
		for i := 0; i < len(groupExtras); i++ {
			for j := i + 1; j < len(groupExtras); j++ {
				e1 := groupExtras[i]
				e2 := groupExtras[j]

				r1 := effectiveRange(e1.Operator, e1.CondRangeMin, e1.CondRangeMax)
				r2 := effectiveRange(e2.Operator, e2.CondRangeMin, e2.CondRangeMax)

				if !rangesOverlap(r1, r2) {
					continue
				}

				// แถวที่ไม่มี condition ครอบทุกค่าอยู่แล้ว การชนกันจึงแปลว่า
				// extra_key ซ้ำ ไม่ใช่ช่วงตัวเลขทับกัน รายงานด้วยข้อความคนละแบบ
				// ไม่งั้นผู้ใช้จะเห็น "operator=\"\" [-Inf, +Inf]" ซึ่งอ่านไม่รู้เรื่อง
				if strings.TrimSpace(e1.ConditionCode) == "" {
					return fmt.Errorf(
						"duplicate extra detected: price_list_group_id=%s, extra_key=%s ซ้ำกัน",
						e1.PriceListGroupID, e1.ExtraKey,
					)
				}

				return fmt.Errorf(
					"overlapping condition detected: price_list_group_id=%s, condition_code=%s. "+
						"Conflicting ranges: operator=%s [%.2f, %.2f] and operator=%s [%.2f, %.2f]",
					groupExtras[0].PriceListGroupID,
					e1.ConditionCode,
					e1.Operator, r1.lo, r1.hi,
					e2.Operator, r2.lo, r2.hi,
				)
			}
		}
	}

	return nil
}

// condRange คือช่วงค่าที่ operator หนึ่งครอบคลุม พร้อมบอกว่าขอบแต่ละด้าน
// นับรวมตัวมันเองหรือไม่
//
// การละ inclusive ทำให้ช่วงที่ติดกันตามเจตนาอย่าง "<> 30..38" คู่กับ "> 38"
// ถูกมองว่าทับกันที่ 38 แล้ว reject ทั้งที่ตั้งค่าถูก
type condRange struct {
	lo, hi       float64
	loInc, hiInc bool
}

// effectiveRange แปลง operator เป็นช่วง
//
// ต้องใช้ cond_range_max เป็นค่าอ้างอิงของ operator ตัวเดียวทุกตัว ให้ตรงกับ
// extraConditionMatched ซึ่งเป็นตัวตัดสินราคาจริง
func effectiveRange(operator string, min, max float64) condRange {
	switch operator {
	case ">":
		return condRange{max, math.Inf(1), false, false}
	case ">=":
		return condRange{max, math.Inf(1), true, false}
	case "<":
		return condRange{math.Inf(-1), max, false, false}
	case "<=":
		return condRange{math.Inf(-1), max, false, true}
	case "=":
		return condRange{max, max, true, true}
	case "<>":
		return condRange{min, max, true, true}
	case "":
		// ไม่มี condition = บวกทุกค่า จึงทับกับทุกช่วงใน extra_key เดียวกัน
		return condRange{math.Inf(-1), math.Inf(1), true, true}
	default:
		// operator ที่ไม่รู้จักถูก validateExtras ปฏิเสธไปก่อนถึงตรงนี้แล้ว
		// คืนช่วงว่างไว้เพื่อไม่ให้ default กลืนมันเป็น "ทับทุกช่วง" เงียบ ๆ
		// ซึ่งจะสวนทางกับ extraConditionMatched ที่คืน false ให้ operator แบบนี้
		return condRange{math.NaN(), math.NaN(), false, false}
	}
}

func rangesOverlap(a, b condRange) bool {
	// ช่วงว่าง (NaN) ไม่ทับกับอะไรเลย · ถ้าไม่ดักไว้ การเปรียบเทียบกับ NaN
	// จะ false ทุกบรรทัดแล้วตกไปคืน true ซึ่งกลับด้านกับที่ต้องการ
	if math.IsNaN(a.lo) || math.IsNaN(a.hi) || math.IsNaN(b.lo) || math.IsNaN(b.hi) {
		return false
	}
	if a.hi < b.lo || b.hi < a.lo {
		return false
	}
	// แตะกันที่จุดเดียว: ทับกันต่อเมื่อทั้งสองฝั่งนับจุดนั้นเป็นของตัวเอง
	if a.hi == b.lo && !(a.hiInc && b.loInc) {
		return false
	}
	if b.hi == a.lo && !(b.hiInc && a.loInc) {
		return false
	}
	return true
}

// validExtraOperators คือ operator ที่ extraConditionMatched รองรับ
// (update-latest-pricelist-subgroup.go) ซึ่งเป็นตัวตัดสินราคาจริง
// ห้ามใช้ effectiveRange เป็นแหล่งความจริง มันเป็นแค่ helper ของการตรวจ overlap
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
		// condition_code กับ operator ต้องมีหรือไม่มีพร้อมกัน
		//
		// group ที่ config ไม่มีแกน condition (เช่น หมวดตัวซี: PG01, PG04) ไม่มีทาง
		// มี condition_code ได้เลย เพราะหน้าจอเซ็ตให้อัตโนมัติจาก extraConfig ที่
		// is_condition=true เท่านั้น และไม่มีช่องให้ผู้ใช้กรอกเอง
		// การบังคับว่าห้ามว่างทำให้หน้าเหล่านั้นกด Update ไม่ผ่านถาวร แก้จาก UI ไม่ได้
		hasCondition := strings.TrimSpace(e.ConditionCode) != ""
		operator := strings.TrimSpace(e.Operator)

		if hasCondition && !validExtraOperators[operator] {
			return &utils.BindingError{
				Message: fmt.Sprintf("รายการที่ %d: operator %q ไม่ถูกต้อง ต้องเป็น =, >=, <=, <, > หรือ <>", i+1, e.Operator),
			}
		}
		if !hasCondition && operator != "" {
			return &utils.BindingError{
				Message: fmt.Sprintf("รายการที่ %d: มี operator %q แต่ไม่ได้ระบุ condition_code", i+1, e.Operator),
			}
		}
		// เฉพาะ "<>" เท่านั้นที่ใช้ทั้ง min และ max พร้อมกัน (extraConditionMatched:
		// val >= min && val <= max) operator อื่นใช้แค่ max
		// ค่า min ไม่มีความหมายและอาจเป็นข้อมูลเก่าที่ถูกต้องอยู่แล้ว
		if operator == "<>" && e.CondRangeMin > e.CondRangeMax {
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
