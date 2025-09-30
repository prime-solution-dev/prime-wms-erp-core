package priceService

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"prime-erp-core/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type GetPriceListGroupRequest struct {
	CompanyCode       string     `json:"company_code"`
	SiteCodes         []string   `json:"site_codes"`
	GroupCodes        []string   `json:"group_codes"`
	EffectiveDateFrom *time.Time `json:"effective_date_from"`
	EffectiveDateTo   *time.Time `json:"effective_date_to"`
	SubGroupCodes     []string   `json:"sub_group_codes"`
}

type GetPriceListGroupResponse struct {
	PriceListGroup
}

type PriceListGroup struct {
	ID                uuid.UUID             `json:"id"`
	CompanyCode       string                `json:"company_code"`
	SiteCode          string                `json:"site_code"`
	GroupCode         string                `json:"group_code"`
	PriceUnit         float64               `json:"price_unit"`
	PriceWeight       float64               `json:"price_weight"`
	BeforePriceUnit   float64               `json:"before_price_unit"`
	BeforePriceWeight float64               `json:"before_price_weight"`
	Currency          string                `json:"currency"`
	EffectiveDate     time.Time             `json:"effective_date"`
	ExtraPattern      string                `json:"extra_pattern"`
	Remark            string                `json:"remark"`
	Terms             []PriceListGroupTerm  `json:"terms"`
	Extras            []PriceListGroupExtra `json:"extras"`
	SubGroups         []SubGroup            `json:"sub_groups"`
}

type PriceListGroupTerm struct {
	ID         uuid.UUID `json:"id"`
	TermCode   string    `json:"term_code"`
	Pdc        float64   `json:"pdc"`
	PdcPercent float64   `json:"pdc_percent"`
	Due        float64   `json:"due"`
	DuePercent float64   `json:"due_percent"`
}

type PriceListGroupExtra struct {
	ID             uuid.UUID  `json:"id"`
	ExtraKey       string     `json:"extra_key"`
	LengthExtraKey int        `json:"length_extra_key"`
	Operator       string     `json:"operator"`
	CondRangeMin   float64    `json:"cond_range_min"`
	CondRangeMax   float64    `json:"cond_range_max"`
	GroupKeys      []GroupKey `json:"group_keys"`
}

type GroupKey struct {
	Code  string `json:"code"`
	Value string `json:"value"`
	Seq   int    `json:"seq"`
}

type SubGroup struct {
	ID                        uuid.UUID  `json:"id"`
	SubGroupKey               string     `json:"subgroup_key"`
	IsTrading                 bool       `json:"is_trading"`
	PriceUnit                 float64    `json:"price_unit"`
	ExtraPriceUnit            float64    `json:"extra_price_unit"`
	TermPriceUnit             float64    `json:"term_price_unit"`
	TotalNetPriceUnit         float64    `json:"total_net_price_unit"`
	PriceWeight               float64    `json:"price_weight"`
	ExtraPriceWeight          float64    `json:"extra_price_weight"`
	TermPriceWeight           float64    `json:"term_price_weight"`
	TotalNetPriceWeight       float64    `json:"total_net_price_weight"`
	BeforePriceUnit           float64    `json:"before_price_unit"`
	BeforeExtraPriceUnit      float64    `json:"before_extra_price_unit"`
	BeforeTermPriceUnit       float64    `json:"before_term_price_unit"`
	BeforeTotalNetPriceUnit   float64    `json:"before_total_net_price_unit"`
	BeforePriceWeight         float64    `json:"before_price_weight"`
	BeforeExtraPriceWeight    float64    `json:"before_extra_price_weight"`
	BeforeTermPriceWeight     float64    `json:"before_term_price_weight"`
	BeforeTotalNetPriceWeight float64    `json:"before_total_net_price_weight"`
	EffectiveDate             time.Time  `json:"effective_date"`
	Remark                    string     `json:"remark"`
	GroupKeys                 []GroupKey `json:"group_keys"`
}

func GetPriceListGroup(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req GetPriceListGroupRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	sqlx, err := db.ConnectSqlx(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer sqlx.Close()

	res, err := getGroupSubGroup(sqlx, req)
	if err != nil {
		return nil, fmt.Errorf("GetGroupSubGroup error: %w", err)
	}

	res, err = getTerms(sqlx, res)
	if err != nil {
		return nil, fmt.Errorf("GetTerms error: %w", err)
	}

	res, err = getExtras(sqlx, res)
	if err != nil {
		return nil, fmt.Errorf("GetExtras error: %w", err)
	}

	return res, nil
}

func getExtras(sqlx *sqlx.DB, res []GetPriceListGroupResponse) ([]GetPriceListGroupResponse, error) {
	groupIDs := []string{}
	for _, r := range res {
		groupIDs = append(groupIDs, r.ID.String())
	}

	if len(groupIDs) == 0 {
		return res, nil
	}

	query := fmt.Sprintf(`
		select
			ple.price_list_group_id,
			ple.extra_key,
			ple.length_extra_key,
			ple.operator,
			ple.cond_range_min,
			ple.cond_range_max,
			gk.code,
			gk.value,
			gk.seq
		from price_list_group_extra ple
		left join price_list_group_extra_key gk on ple.id = gk.group_extra_id 
		where 1=1
		and ple.price_list_group_id in ('%s')
	`, strings.Join(groupIDs, `','`))
	println(query)
	rows, err := db.ExecuteQuery(sqlx, query)
	if err != nil {
		return nil, fmt.Errorf("ExecuteQuery error: %w", err)
	}

	for _, row := range rows {
		groupID := toString(row["price_list_group_id"])
		extraKey := toString(row["extra_key"])

		extra := PriceListGroupExtra{
			ExtraKey:       extraKey,
			LengthExtraKey: int(toFloat64(row["length_extra_key"])),
			Operator:       toString(row["operator"]),
			CondRangeMin:   toFloat64(row["cond_range_min"]),
			CondRangeMax:   toFloat64(row["cond_range_max"]),
		}

		groupKey := GroupKey{
			Code:  toString(row["code"]),
			Seq:   int(toFloat64(row["seq"])),
			Value: toString(row["value"]),
		}

		for i := range res {
			if res[i].ID.String() == groupID {
				found := false
				for j := range res[i].Extras {
					if res[i].Extras[j].ExtraKey == extraKey {
						res[i].Extras[j].GroupKeys = append(res[i].Extras[j].GroupKeys, groupKey)
						found = true
						break
					}
				}

				if !found {
					extra.GroupKeys = append(extra.GroupKeys, groupKey)
					res[i].Extras = append(res[i].Extras, extra)
				}
				break
			}
		}
	}

	return res, nil
}

func getTerms(sqlx *sqlx.DB, res []GetPriceListGroupResponse) ([]GetPriceListGroupResponse, error) {

	groupIDs := []string{}
	for _, r := range res {
		groupIDs = append(groupIDs, r.ID.String())
	}

	if len(groupIDs) == 0 {
		return res, nil
	}

	query := fmt.Sprintf(`
		select
			plt.price_list_group_id,
			plt.term_code,
			plt.pdc,
			plt.pdc_percent,
			plt.due,
			plt.due_percent
		from price_list_group_term plt
		where 1=1
		and plt.price_list_group_id in ('%s')
	`, strings.Join(groupIDs, `','`))
	rows, err := db.ExecuteQuery(sqlx, query)
	if err != nil {
		return nil, fmt.Errorf("ExecuteQuery error: %w", err)
	}

	for _, row := range rows {
		groupID := toString(row["price_list_group_id"])
		term := PriceListGroupTerm{
			TermCode:   toString(row["term_code"]),
			Pdc:        toFloat64(row["pdc"]),
			PdcPercent: toFloat64(row["pdc_percent"]),
			Due:        toFloat64(row["due"]),
			DuePercent: toFloat64(row["due_percent"]),
		}

		for i := range res {
			if res[i].ID.String() == groupID {
				res[i].Terms = append(res[i].Terms, term)
				break
			}
		}
	}

	return res, nil
}

func getGroupSubGroup(sqlx *sqlx.DB, req GetPriceListGroupRequest) ([]GetPriceListGroupResponse, error) {
	res := []GetPriceListGroupResponse{}
	cond := ``

	if req.CompanyCode != "" {
		cond += fmt.Sprintf(` and plg.company_code = '%s' `, req.CompanyCode)
	}

	if len(req.SiteCodes) > 0 {
		cond += fmt.Sprintf(` and plg.site_code in ('%s') `, strings.Join(req.SiteCodes, `','`))
	}

	if req.EffectiveDateFrom != nil {
		cond += fmt.Sprintf(` and plg.effective_date >= '%s' `, req.EffectiveDateFrom.Format(`2006-01-02`))
	}

	if req.EffectiveDateTo != nil {
		cond += fmt.Sprintf(` and plg.effective_date <= '%s' `, req.EffectiveDateTo.Format(`2006-01-02`))
	}

	if len(req.GroupCodes) > 0 {
		cond += fmt.Sprintf(` and plg.group_code in ('%s') `, strings.Join(req.GroupCodes, `','`))
	}

	if len(req.SubGroupCodes) > 0 {
		cond += fmt.Sprintf(` and plsg.subgroup_key in ('%s') `, strings.Join(req.SubGroupCodes, `','`))
		// cond += fmt.Sprintf(`
		// 	and exists (
		// 		select 0
		// 		from price_list_group plgx
		// 		left join price_list_sub_group plsgx on plgx.id = plsgx.price_list_group_id
		// 		where 1=1
		// 			and plgx.id = plg.id
		// 			and plsgx.subgroup_key in ('%s')
		// 	)
		// `, strings.Join(req.SubGroupCodes, `','`))
	}

	// Query Group + SubGroup
	query := fmt.Sprintf(`
		SELECT 
			plg.id as group_id,
			plg.company_code,
			plg.site_code,
			plg.group_code,
			COALESCE(plg.price_unit, 0) AS group_price_unit,
			COALESCE(plg.price_weight, 0) AS group_price_weight,
			COALESCE(plg.before_price_unit, 0) AS group_before_price_unit,
			COALESCE(plg.before_price_weight, 0) AS group_before_price_weight,
			COALESCE(plg.currency, '') AS currency,
			plg.effective_date AS group_effective_date,
			COALESCE(plg.extra_pattern, '') AS extra_pattern,
			COALESCE(plg.remark, '') AS group_remark,

			plsg.id as sub_id,
			COALESCE(plsg.subgroup_key, '') AS subgroup_key,
			COALESCE(plsg.is_trading, FALSE) AS is_trading,
			COALESCE(plsg.price_unit, 0) AS price_unit,
			COALESCE(plsg.extra_price_unit, 0) AS extra_price_unit,
			COALESCE(plsg.term_price_unit, 0) AS term_price_unit,
			COALESCE(plsg.total_net_price_unit, 0) AS total_net_price_unit,
			COALESCE(plsg.price_weight, 0) AS price_weight,
			COALESCE(plsg.extra_price_weight, 0) AS extra_price_weight,
			COALESCE(plsg.term_price_weight, 0) AS term_price_weight,
			COALESCE(plsg.total_net_price_weight, 0) AS total_net_price_weight,
			COALESCE(plsg.before_price_unit, 0) AS before_price_unit,
			COALESCE(plsg.before_extra_price_unit, 0) AS before_extra_price_unit,
			COALESCE(plsg.before_term_price_unit, 0) AS before_term_price_unit,
			COALESCE(plsg.before_total_net_price_unit, 0) AS before_total_net_price_unit,
			COALESCE(plsg.before_price_weight, 0) AS before_price_weight,
			COALESCE(plsg.before_extra_price_weight, 0) AS before_extra_price_weight,
			COALESCE(plsg.before_term_price_weight, 0) AS before_term_price_weight,
			COALESCE(plsg.before_total_net_price_weight, 0) AS before_total_net_price_weight,
			plsg.effective_date AS sub_effective_date,
			COALESCE(plsg.remark, '') AS sub_remark
		FROM price_list_group plg
		LEFT JOIN price_list_sub_group plsg ON plg.id = plsg.price_list_group_id
		WHERE 1=1 %s
	`, cond)
	//println(query)
	rows, err := db.ExecuteQuery(sqlx, query)
	if err != nil {
		return nil, fmt.Errorf("ExecuteQuery error: %w", err)
	}

	groupMap := map[string]*PriceListGroup{}

	for _, row := range rows {
		groupID := toString(row["group_id"])
		group, exists := groupMap[groupID]
		if !exists {
			group = &PriceListGroup{
				ID:                parseUUID(groupID),
				CompanyCode:       toString(row["company_code"]),
				SiteCode:          toString(row["site_code"]),
				GroupCode:         toString(row["group_code"]),
				PriceUnit:         toFloat64(row["group_price_unit"]),
				PriceWeight:       toFloat64(row["group_price_weight"]),
				BeforePriceUnit:   toFloat64(row["group_before_price_unit"]),
				BeforePriceWeight: toFloat64(row["group_before_price_weight"]),
				Currency:          toString(row["currency"]),
				ExtraPattern:      toString(row["extra_pattern"]),
				Remark:            toString(row["group_remark"]),
			}
			if t := toTime(row["group_effective_date"]); t != nil {
				group.EffectiveDate = *t
			}
			groupMap[groupID] = group
		}

		// Append SubGroup
		if toString(row["subgroup_key"]) != "" {
			subGroup := SubGroup{
				ID:                        parseUUID(toString(row["sub_id"])),
				SubGroupKey:               toString(row["subgroup_key"]),
				IsTrading:                 toBool(row["is_trading"]),
				PriceUnit:                 toFloat64(row["price_unit"]),
				ExtraPriceUnit:            toFloat64(row["extra_price_unit"]),
				TermPriceUnit:             toFloat64(row["term_price_unit"]),
				TotalNetPriceUnit:         toFloat64(row["total_net_price_unit"]),
				PriceWeight:               toFloat64(row["price_weight"]),
				ExtraPriceWeight:          toFloat64(row["extra_price_weight"]),
				TermPriceWeight:           toFloat64(row["term_price_weight"]),
				TotalNetPriceWeight:       toFloat64(row["total_net_price_weight"]),
				BeforePriceUnit:           toFloat64(row["before_price_unit"]),
				BeforeExtraPriceUnit:      toFloat64(row["before_extra_price_unit"]),
				BeforeTermPriceUnit:       toFloat64(row["before_term_price_unit"]),
				BeforeTotalNetPriceUnit:   toFloat64(row["before_total_net_price_unit"]),
				BeforePriceWeight:         toFloat64(row["before_price_weight"]),
				BeforeExtraPriceWeight:    toFloat64(row["before_extra_price_weight"]),
				BeforeTermPriceWeight:     toFloat64(row["before_term_price_weight"]),
				BeforeTotalNetPriceWeight: toFloat64(row["before_total_net_price_weight"]),
				Remark:                    toString(row["sub_remark"]),
			}
			if t := toTime(row["sub_effective_date"]); t != nil {
				subGroup.EffectiveDate = *t
			}

			group.SubGroups = append(group.SubGroups, subGroup)
		}
	}

	//Keep for fetching GroupKeys
	groupIDs := []string{}
	for _, g := range groupMap {
		for _, sg := range g.SubGroups {
			groupIDs = append(groupIDs, sg.ID.String())
		}
	}

	if len(groupIDs) > 0 {
		queryKeys := fmt.Sprintf(`
			SELECT sub_group_id, code, value, seq
			FROM price_list_sub_group_key
			WHERE sub_group_id IN ('%s')
		`, strings.Join(groupIDs, "','"))

		keyRows, err := db.ExecuteQuery(sqlx, queryKeys)
		if err != nil {
			return nil, err
		}

		for _, kr := range keyRows {
			subID := toString(kr["sub_group_id"])
			groupKey := GroupKey{
				Code:  toString(kr["code"]),
				Value: toString(kr["value"]),
				Seq:   int(toFloat64(kr["seq"])),
			}

			for _, g := range groupMap {
				for i := range g.SubGroups {
					if g.SubGroups[i].ID.String() == subID {
						g.SubGroups[i].GroupKeys = append(g.SubGroups[i].GroupKeys, groupKey)
					}
				}
			}
		}
	}

	// convert map to slice
	res = []GetPriceListGroupResponse{}
	for _, g := range groupMap {
		res = append(res, GetPriceListGroupResponse{PriceListGroup: *g})
	}

	return res, nil
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func toFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int64:
		return float64(val)
	case int:
		return float64(val)
	case []uint8: // numeric as bytes
		f, _ := strconv.ParseFloat(string(val), 64)
		return f
	default:
		return 0
	}
}

func toBool(v interface{}) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func toTime(v interface{}) *time.Time {
	if v == nil {
		return nil
	}
	if t, ok := v.(*time.Time); ok {
		return t
	}
	return nil
}

func parseUUID(s string) uuid.UUID {
	if s == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}
