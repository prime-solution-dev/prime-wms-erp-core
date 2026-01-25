package priceService

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"prime-erp-core/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

var (
	pgCols = []string{"PG01", "PG02", "PG03", "PG04", "PG05", "PG06", "PG07", "PG08", "PG09", "PG10"}
)

type CreatePricelistRequest struct {
	Groups           []PriceListGroupCreateDTO
	Terms            []PriceListGroupTermCreateDTO
	Extras           []PriceListGroupExtraCreateDTO
	SubGroups        []PriceListSubGroupCreateDTO
	GroupKeys        []PriceListGroupKeyDTO
	SubGroupKeys     []PriceListSubGroupKeyDTO
	ExtraKeys        []PriceListGroupExtraKeyDTO
	SubGroupFormulas []PriceListSubGroupFormulasCreateDTO
}

type PriceListSubGroupFormulasCreateDTO struct {
	SubGroupCode string `json:"subgroup_code"`
	FormulaCode  string `json:"formula_code"`
	IsDefault    bool   `json:"is_default"`
}

type PriceListGroupCreateDTO struct {
	CompanyCode       string
	SiteCode          string
	GroupCode         string
	GroupName         string
	Currency          string
	EffectiveDate     *time.Time
	PriceUnit         float64
	PriceWeight       float64
	BeforePriceUnit   float64
	BeforePriceWeight float64
	Remark            string
	CreateBy          string
	UpdateBy          string
}

type PriceListGroupTermCreateDTO struct {
	CompanyCode string
	SiteCode    string
	GroupCode   string
	TermCode    string
	Pdc         float64
	PdcPercent  int
	Due         float64
	DuePercent  int
	CreateBy    string
}

type PriceListGroupExtraCreateDTO struct {
	CompanyCode    string
	SiteCode       string
	GroupCode      string
	ExtraKey       string // GEN from price_list_group_extra.PGxx
	ConditionCode  string
	Operator       string
	ValueInt       int
	LengthExtraKey int
	CondRangeMin   float64
	CondRangeMax   float64
	CreateBy       string
}

type PriceListSubGroupCreateDTO struct {
	CompanyCode               string
	SiteCode                  string
	GroupCode                 string
	SubGroupKey               string // GEN from price_list_sub_group.PGxx
	IsTrading                 bool
	PriceUnit                 float64
	ExtraPriceUnit            float64
	TotalNetPriceUnit         float64
	PriceWeight               float64
	ExtraPriceWeight          float64
	TermPriceWeight           float64
	TotalNetPriceWeight       float64
	BeforePriceUnit           float64
	BeforeExtraPriceUnit      float64
	BeforeTermPriceUnit       float64
	BeforeTotalNetPriceUnit   float64
	BeforePriceWeight         float64
	BeforeExtraPriceWeight    float64
	BeforeTermPriceWeight     float64
	BeforeTotalNetPriceWeight float64
	EffectiveDate             *time.Time
	Remark                    string
	UdfJson                   json.RawMessage
	CreateBy                  string
	SubGroupCode              string
}

type PriceListGroupKeyDTO struct {
	CompanyCode string
	SiteCode    string
	GroupCode   string
	Seq         int
	Code        string // PG01..PG10
	Value       string // GroupItem.ItemCode
}

type PriceListSubGroupKeyDTO struct {
	CompanyCode string
	SiteCode    string
	GroupCode   string
	SubGroupKey string
	Seq         int
	Code        string // PG01..PG10
	Value       string // GroupItem.ItemCode
}

type PriceListGroupExtraKeyDTO struct {
	CompanyCode string
	SiteCode    string
	GroupCode   string
	ExtraKey    string // GEN
	Seq         int
	Code        string // PG01..PG10
	Value       string // GroupItem.ItemCode
}

type CreatePricelistResponse struct {
	ResponseCode int    `json:"response_code"`
	Message      string `json:"message"`
}

func UploadPricelistMultipart(ctx *gin.Context) (interface{}, error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	file, _, err := ctx.Request.FormFile("files")
	if err != nil {
		return &CreatePricelistResponse{ResponseCode: 1, Message: fmt.Sprintf("missing file (form-data key: file): %v", err)}, nil
	}
	defer file.Close()

	req, err := buildCreatePricelistRequestFromExcel(file)
	if err != nil {
		return &CreatePricelistResponse{ResponseCode: 1, Message: err.Error()}, nil
	}

	return CreatePricelist(gormx, *req)
}

func CreatePricelist(gormx *gorm.DB, req CreatePricelistRequest) (*CreatePricelistResponse, error) {
	res := &CreatePricelistResponse{ResponseCode: 0, Message: "success"}
	const batchSize = 500

	err := gormx.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		groupKey := func(c, s, g string) string { return c + "|" + s + "|" + g }
		subKey := func(c, s, g, sg string) string { return groupKey(c, s, g) + "|SUB|" + sg }
		extraKey := func(c, s, g, ek string) string { return groupKey(c, s, g) + "|EXTRA|" + ek }

		// ---------- map IDs ----------
		groupIDs := map[string]uuid.UUID{}
		for _, g := range req.Groups {
			groupIDs[groupKey(g.CompanyCode, g.SiteCode, g.GroupCode)] = uuid.New()
		}

		subGroupIDs := map[string]uuid.UUID{}
		for _, s := range req.SubGroups {
			subGroupIDs[subKey(s.CompanyCode, s.SiteCode, s.GroupCode, s.SubGroupKey)] = uuid.New()
		}

		extraIDs := map[string]uuid.UUID{}
		for _, e := range req.Extras {
			extraIDs[extraKey(e.CompanyCode, e.SiteCode, e.GroupCode, e.ExtraKey)] = uuid.New()
		}

		// ---------- validate refs ----------
		for _, t := range req.Terms {
			gk := groupKey(t.CompanyCode, t.SiteCode, t.GroupCode)
			if _, ok := groupIDs[gk]; !ok {
				return fmt.Errorf("term references missing group in file: %s (term_code=%s)", gk, t.TermCode)
			}
		}
		for _, e := range req.Extras {
			gk := groupKey(e.CompanyCode, e.SiteCode, e.GroupCode)
			if _, ok := groupIDs[gk]; !ok {
				return fmt.Errorf("extra references missing group in file: %s (extra_key=%s)", gk, e.ExtraKey)
			}
		}
		for _, s := range req.SubGroups {
			gk := groupKey(s.CompanyCode, s.SiteCode, s.GroupCode)
			if _, ok := groupIDs[gk]; !ok {
				return fmt.Errorf("subgroup references missing group in file: %s (subgroup_key=%s)", gk, s.SubGroupKey)
			}
		}

		// ---------- create-only duplicate check in DB (group) ----------
		if err := checkDuplicateGroupsCreateOnly(tx, req.Groups); err != nil {
			return err
		}

		// ---------- BUILD batch records ----------
		groupRecs := make([]map[string]any, 0, len(req.Groups))
		for _, g := range req.Groups {
			gk := groupKey(g.CompanyCode, g.SiteCode, g.GroupCode)
			groupRecs = append(groupRecs, map[string]any{
				"id":                  groupIDs[gk],
				"company_code":        g.CompanyCode,
				"site_code":           g.SiteCode,
				"group_code":          g.GroupCode,
				"group_name":          g.GroupName,
				"currency":            g.Currency,
				"effective_date":      g.EffectiveDate,
				"price_unit":          g.PriceUnit,
				"price_weight":        g.PriceWeight,
				"before_price_unit":   g.BeforePriceUnit,
				"before_price_weight": g.BeforePriceWeight,
				"remark":              g.Remark,
				"create_by":           g.CreateBy,
				"create_dtm":          now,
				"update_by":           g.UpdateBy,
				"update_dtm":          now,
			})
		}

		termRecs := make([]map[string]any, 0, len(req.Terms))
		for _, t := range req.Terms {
			gk := groupKey(t.CompanyCode, t.SiteCode, t.GroupCode)
			termRecs = append(termRecs, map[string]any{
				"id":                  uuid.New(),
				"price_list_group_id": groupIDs[gk],
				"term_code":           t.TermCode,
				"pdc":                 t.Pdc,
				"pdc_percent":         t.PdcPercent,
				"due":                 t.Due,
				"due_percent":         t.DuePercent,
				"create_by":           t.CreateBy,
				"create_dtm":          now,
				"update_by":           t.CreateBy,
				"update_dtm":          now,
			})
		}

		extraRecs := make([]map[string]any, 0, len(req.Extras))
		for _, e := range req.Extras {
			gk := groupKey(e.CompanyCode, e.SiteCode, e.GroupCode)
			ek := extraKey(e.CompanyCode, e.SiteCode, e.GroupCode, e.ExtraKey)
			extraRecs = append(extraRecs, map[string]any{
				"id":                  extraIDs[ek],
				"price_list_group_id": groupIDs[gk],
				"extra_key":           e.ExtraKey,
				"condition_code":      e.ConditionCode,
				"operator":            e.Operator,
				"value_int":           e.ValueInt,
				"length_extra_key":    e.LengthExtraKey,
				"cond_range_min":      e.CondRangeMin,
				"cond_range_max":      e.CondRangeMax,
				"create_by":           e.CreateBy,
				"create_dtm":          now,
				"update_by":           e.CreateBy,
				"update_dtm":          now,
			})
		}

		subRecs := make([]map[string]any, 0, len(req.SubGroups))
		for _, s := range req.SubGroups {
			gk := groupKey(s.CompanyCode, s.SiteCode, s.GroupCode)
			sk := subKey(s.CompanyCode, s.SiteCode, s.GroupCode, s.SubGroupKey)
			subRecs = append(subRecs, map[string]any{
				"id":                            subGroupIDs[sk],
				"price_list_group_id":           groupIDs[gk],
				"subgroup_key":                  s.SubGroupKey,
				"is_trading":                    s.IsTrading,
				"price_unit":                    s.PriceUnit,
				"extra_price_unit":              s.ExtraPriceUnit,
				"total_net_price_unit":          s.TotalNetPriceUnit,
				"price_weight":                  s.PriceWeight,
				"extra_price_weight":            s.ExtraPriceWeight,
				"term_price_weight":             s.TermPriceWeight,
				"total_net_price_weight":        s.TotalNetPriceWeight,
				"before_price_unit":             s.BeforePriceUnit,
				"before_extra_price_unit":       s.BeforeExtraPriceUnit,
				"before_term_price_unit":        s.BeforeTermPriceUnit,
				"before_total_net_price_unit":   s.BeforeTotalNetPriceUnit,
				"before_price_weight":           s.BeforePriceWeight,
				"before_extra_price_weight":     s.BeforeExtraPriceWeight,
				"before_term_price_weight":      s.BeforeTermPriceWeight,
				"before_total_net_price_weight": s.BeforeTotalNetPriceWeight,
				"effective_date":                s.EffectiveDate,
				"remark":                        s.Remark,
				"udf_json":                      s.UdfJson,
				"create_by":                     s.CreateBy,
				"create_dtm":                    now,
				"update_by":                     s.CreateBy,
				"subgroup_code":                 s.SubGroupCode,
				"update_dtm":                    now,
			})
		}

		groupKeyRecs := make([]map[string]any, 0, len(req.GroupKeys))
		for _, k := range req.GroupKeys {
			gk := groupKey(k.CompanyCode, k.SiteCode, k.GroupCode)
			groupKeyRecs = append(groupKeyRecs, map[string]any{
				"id":                  uuid.New(),
				"price_list_group_id": groupIDs[gk],
				"seq":                 k.Seq,
				"code":                k.Code,
				"value":               k.Value,
			})
		}

		extraKeyRecs := make([]map[string]any, 0, len(req.ExtraKeys))
		for _, k := range req.ExtraKeys {
			ek := extraKey(k.CompanyCode, k.SiteCode, k.GroupCode, k.ExtraKey)
			extraKeyRecs = append(extraKeyRecs, map[string]any{
				"id":             uuid.New(),
				"group_extra_id": extraIDs[ek],
				"seq":            k.Seq,
				"code":           k.Code,
				"value":          k.Value,
			})
		}

		subKeyRecs := make([]map[string]any, 0, len(req.SubGroupKeys))
		for _, k := range req.SubGroupKeys {
			sk := subKey(k.CompanyCode, k.SiteCode, k.GroupCode, k.SubGroupKey)
			subKeyRecs = append(subKeyRecs, map[string]any{
				"id":           uuid.New(),
				"sub_group_id": subGroupIDs[sk],
				"seq":          k.Seq,
				"code":         k.Code,
				"value":        k.Value,
			})
		}

		subGroupFormulasRecs := make([]map[string]any, 0, len(req.SubGroupFormulas))
		for _, f := range req.SubGroupFormulas {
			subGroupFormulasRecs = append(subGroupFormulasRecs, map[string]any{
				"id":                       uuid.New(),
				"price_list_subgroup_code": f.SubGroupCode,
				"price_list_formulas_code": f.FormulaCode,
				"is_default":               f.IsDefault,
			})
		}

		// ---------- BATCH INSERT ----------
		if len(groupRecs) > 0 {
			if err := tx.Table("price_list_group").CreateInBatches(groupRecs, batchSize).Error; err != nil {
				return err
			}
		}
		if len(termRecs) > 0 {
			if err := tx.Table("price_list_group_term").CreateInBatches(termRecs, batchSize).Error; err != nil {
				return err
			}
		}
		if len(extraRecs) > 0 {
			if err := tx.Table("price_list_group_extra").CreateInBatches(extraRecs, batchSize).Error; err != nil {
				return err
			}
		}
		if len(subRecs) > 0 {
			if err := tx.Table("price_list_sub_group").CreateInBatches(subRecs, batchSize).Error; err != nil {
				return err
			}
		}
		if len(groupKeyRecs) > 0 {
			if err := tx.Table("price_list_group_key").CreateInBatches(groupKeyRecs, batchSize).Error; err != nil {
				return err
			}
		}
		if len(extraKeyRecs) > 0 {
			if err := tx.Table("price_list_group_extra_key").CreateInBatches(extraKeyRecs, batchSize).Error; err != nil {
				return err
			}
		}
		if len(subKeyRecs) > 0 {
			if err := tx.Table("price_list_sub_group_key").CreateInBatches(subKeyRecs, batchSize).Error; err != nil {
				return err
			}
		}
		if len(subGroupFormulasRecs) > 0 {
			if err := tx.Table("price_list_subgroup_formulas_map").CreateInBatches(subGroupFormulasRecs, batchSize).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return &CreatePricelistResponse{ResponseCode: 1, Message: err.Error()}, nil
	}
	return res, nil
}

func buildCreatePricelistRequestFromExcel(r io.Reader) (*CreatePricelistRequest, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("open excel failed: %w", err)
	}
	defer func() { _ = f.Close() }()

	req := &CreatePricelistRequest{}

	readSheet := func(sheet string) ([]map[string]string, error) {
		rows, err := f.GetRows(sheet)
		if err != nil {
			return nil, fmt.Errorf("missing sheet %s: %w", sheet, err)
		}
		if len(rows) == 0 {
			return []map[string]string{}, nil
		}
		header := rows[0]
		out := []map[string]string{}

		for i := 1; i < len(rows); i++ {
			row := rows[i]
			empty := true
			for _, v := range row {
				if strings.TrimSpace(v) != "" {
					empty = false
					break
				}
			}
			if empty {
				continue
			}
			m := map[string]string{}
			for c, h := range header {
				h = strings.TrimSpace(h)
				if h == "" {
					continue
				}
				val := ""
				if c < len(row) {
					val = strings.TrimSpace(row[c])
				}
				m[h] = val
			}
			out = append(out, m)
		}
		return out, nil
	}

	parseTime := func(s string) (*time.Time, error) {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, nil
		}
		layouts := []string{"2006-01-02", "2006-01-02 15:04:05", time.RFC3339}
		for _, ly := range layouts {
			if t, err := time.Parse(ly, s); err == nil {
				return &t, nil
			}
		}
		return nil, fmt.Errorf("invalid date format: %s", s)
	}
	parseBool := func(s string) bool {
		s = strings.TrimSpace(strings.ToLower(s))
		return s == "true" || s == "1" || s == "yes" || s == "y"
	}
	parseFloat := func(s string) float64 {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0
		}
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}
	parseInt := func(s string) int {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0
		}
		v, _ := strconv.Atoi(s)
		return v
	}

	// ---- price_list_group : keys from PG01..PG10 ----
	groupRows, err := readSheet("price_list_group")
	if err != nil {
		return nil, err
	}
	for _, r := range groupRows {
		if r["company_code"] == "" || r["site_code"] == "" || r["group_code"] == "" {
			return nil, fmt.Errorf("price_list_group: company_code, site_code, group_code are required")
		}

		ed, err := parseTime(r["effective_date"])
		if err != nil {
			return nil, fmt.Errorf("price_list_group (group_code=%s): %w", r["group_code"], err)
		}

		req.Groups = append(req.Groups, PriceListGroupCreateDTO{
			CompanyCode:       r["company_code"],
			SiteCode:          r["site_code"],
			GroupCode:         r["group_code"],
			GroupName:         r["group_name"],
			Currency:          r["currency"],
			EffectiveDate:     ed,
			PriceUnit:         parseFloat(r["price_unit"]),
			PriceWeight:       parseFloat(r["price_weight"]),
			BeforePriceUnit:   parseFloat(r["before_price_unit"]),
			BeforePriceWeight: parseFloat(r["before_price_weight"]),
			Remark:            r["remark"],
			CreateBy:          r["create_by"],
			UpdateBy:          r["update_by"],
		})

		_, gKeys := genKeyFromCols(r, pgCols)
		for _, k := range gKeys {
			req.GroupKeys = append(req.GroupKeys, PriceListGroupKeyDTO{
				CompanyCode: r["company_code"],
				SiteCode:    r["site_code"],
				GroupCode:   r["group_code"],
				Seq:         k.Seq,
				Code:        k.Code,
				Value:       k.Value,
			})
		}
	}

	// ---- term ----
	termRows, _ := readSheet("price_list_group_term")
	for _, r := range termRows {
		if r["company_code"] == "" || r["site_code"] == "" || r["group_code"] == "" || r["term_code"] == "" {
			return nil, fmt.Errorf("price_list_group_term: company_code, site_code, group_code, term_code are required")
		}
		req.Terms = append(req.Terms, PriceListGroupTermCreateDTO{
			CompanyCode: r["company_code"],
			SiteCode:    r["site_code"],
			GroupCode:   r["group_code"],
			TermCode:    r["term_code"],
			Pdc:         parseFloat(r["pdc"]),
			PdcPercent:  parseInt(r["pdc_percent"]),
			Due:         parseFloat(r["due"]),
			DuePercent:  parseInt(r["due_percent"]),
			CreateBy:    r["create_by"],
		})
	}

	// ---- extra : gen extra_key + create extra_key rows from same PG01..PG10 ----
	extraRows, _ := readSheet("price_list_group_extra")
	for _, r := range extraRows {
		if r["company_code"] == "" || r["site_code"] == "" || r["group_code"] == "" || r["condition_code"] == "" {
			return nil, fmt.Errorf("price_list_group_extra: company_code, site_code, group_code, condition_code are required")
		}

		exKey, eKeys := genKeyFromCols(r, pgCols)
		if exKey == "" {
			return nil, fmt.Errorf("price_list_group_extra (group_code=%s): ต้องมีอย่างน้อย 1 ค่าใน PG01..PG10 เพื่อ gen extra_key", r["group_code"])
		}

		req.Extras = append(req.Extras, PriceListGroupExtraCreateDTO{
			CompanyCode:    r["company_code"],
			SiteCode:       r["site_code"],
			GroupCode:      r["group_code"],
			ExtraKey:       exKey,
			ConditionCode:  r["condition_code"],
			Operator:       r["operator"],
			ValueInt:       parseInt(r["value_int"]),
			LengthExtraKey: parseInt(r["length_extra_key"]),
			CondRangeMin:   parseFloat(r["cond_range_min"]),
			CondRangeMax:   parseFloat(r["cond_range_max"]),
			CreateBy:       r["create_by"],
		})

		for _, k := range eKeys {
			req.ExtraKeys = append(req.ExtraKeys, PriceListGroupExtraKeyDTO{
				CompanyCode: r["company_code"],
				SiteCode:    r["site_code"],
				GroupCode:   r["group_code"],
				ExtraKey:    exKey,
				Seq:         k.Seq,
				Code:        k.Code,
				Value:       k.Value,
			})
		}
	}

	// ---- sub_group : gen subgroup_key + create subgroup_key rows from same PG01..PG10 ----
	subRows, err := readSheet("price_list_sub_group")
	if err != nil {
		return nil, err
	}
	for _, r := range subRows {
		if r["company_code"] == "" || r["site_code"] == "" || r["group_code"] == "" {
			return nil, fmt.Errorf("price_list_sub_group: company_code, site_code, group_code are required")
		}

		subKeyVal, sKeys := genKeyFromCols(r, pgCols)
		if subKeyVal == "" {
			return nil, fmt.Errorf("price_list_sub_group (group_code=%s): ต้องมีอย่างน้อย 1 ค่าใน PG01..PG10 เพื่อ gen subgroup_key", r["group_code"])
		}

		ed, err := parseTime(r["effective_date"])
		if err != nil {
			return nil, fmt.Errorf("price_list_sub_group (subgroup_key=%s): %w", subKeyVal, err)
		}

		var udf json.RawMessage
		if strings.TrimSpace(r["udf_json"]) != "" {
			if !json.Valid([]byte(r["udf_json"])) {
				return nil, fmt.Errorf("price_list_sub_group (subgroup_key=%s): udf_json invalid json", subKeyVal)
			}
			udf = json.RawMessage([]byte(r["udf_json"]))
		}

		req.SubGroups = append(req.SubGroups, PriceListSubGroupCreateDTO{
			CompanyCode:               r["company_code"],
			SiteCode:                  r["site_code"],
			GroupCode:                 r["group_code"],
			SubGroupKey:               subKeyVal,
			IsTrading:                 parseBool(r["is_trading"]),
			PriceUnit:                 parseFloat(r["price_unit"]),
			ExtraPriceUnit:            parseFloat(r["extra_price_unit"]),
			TotalNetPriceUnit:         parseFloat(r["total_net_price_unit"]),
			PriceWeight:               parseFloat(r["price_weight"]),
			ExtraPriceWeight:          parseFloat(r["extra_price_weight"]),
			TermPriceWeight:           parseFloat(r["term_price_weight"]),
			TotalNetPriceWeight:       parseFloat(r["total_net_price_weight"]),
			BeforePriceUnit:           parseFloat(r["before_price_unit"]),
			BeforeExtraPriceUnit:      parseFloat(r["before_extra_price_unit"]),
			BeforeTermPriceUnit:       parseFloat(r["before_term_price_unit"]),
			BeforeTotalNetPriceUnit:   parseFloat(r["before_total_net_price_unit"]),
			BeforePriceWeight:         parseFloat(r["before_price_weight"]),
			BeforeExtraPriceWeight:    parseFloat(r["before_extra_price_weight"]),
			BeforeTermPriceWeight:     parseFloat(r["before_term_price_weight"]),
			BeforeTotalNetPriceWeight: parseFloat(r["before_total_net_price_weight"]),
			EffectiveDate:             ed,
			Remark:                    r["remark"],
			UdfJson:                   udf,
			CreateBy:                  r["create_by"],
			SubGroupCode:              r["subgroup_code"],
		})

		for _, k := range sKeys {
			req.SubGroupKeys = append(req.SubGroupKeys, PriceListSubGroupKeyDTO{
				CompanyCode: r["company_code"],
				SiteCode:    r["site_code"],
				GroupCode:   r["group_code"],
				SubGroupKey: subKeyVal,
				Seq:         k.Seq,
				Code:        k.Code,
				Value:       k.Value,
			})
		}
	}

	// ---- formulas_map ----
	formulasRows, err := readSheet("formulas_map")
	if err != nil {
		return nil, err
	}
	for _, r := range formulasRows {
		if r["subgroup_code"] == "" || r["formula_code"] == "" {
			return nil, fmt.Errorf("formulas_map: subgroup_code, formula_code are required")
		}
		req.SubGroupFormulas = append(req.SubGroupFormulas, PriceListSubGroupFormulasCreateDTO{
			SubGroupCode: r["subgroup_code"],
			FormulaCode:  r["formula_code"],
			IsDefault:    true,
		})
	}

	return req, nil
}

type existingGroupKeyRow struct {
	CompanyCode string `gorm:"column:company_code"`
	SiteCode    string `gorm:"column:site_code"`
	GroupCode   string `gorm:"column:group_code"`
}

func checkDuplicateGroupsCreateOnly(tx *gorm.DB, groups []PriceListGroupCreateDTO) error {
	if len(groups) == 0 {
		return nil
	}

	keys := make([]existingGroupKeyRow, 0, len(groups))
	seen := map[string]bool{}
	for _, g := range groups {
		k := g.CompanyCode + "|" + g.SiteCode + "|" + g.GroupCode
		if seen[k] {
			return fmt.Errorf("duplicate group in file: %s", k)
		}
		seen[k] = true
		keys = append(keys, existingGroupKeyRow{CompanyCode: g.CompanyCode, SiteCode: g.SiteCode, GroupCode: g.GroupCode})
	}

	const batchSize = 400
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]

		q := tx.Table("price_list_group").Select("company_code, site_code, group_code")
		for idx, k := range batch {
			if idx == 0 {
				q = q.Where("company_code=? AND site_code=? AND group_code=?", k.CompanyCode, k.SiteCode, k.GroupCode)
			} else {
				q = q.Or("company_code=? AND site_code=? AND group_code=?", k.CompanyCode, k.SiteCode, k.GroupCode)
			}
		}

		var existed []existingGroupKeyRow
		if err := q.Find(&existed).Error; err != nil {
			return err
		}
		if len(existed) > 0 {
			e := existed[0]
			return fmt.Errorf("duplicate group in DB: company=%s site=%s group_code=%s", e.CompanyCode, e.SiteCode, e.GroupCode)
		}
	}
	return nil
}

type genKeyPart struct {
	Seq   int
	Code  string
	Value string
}

func genKeyFromCols(row map[string]string, cols []string) (string, []genKeyPart) {
	parts := []string{}
	keys := []genKeyPart{}
	for i, c := range cols {
		v := strings.TrimSpace(row[c])
		if v == "" {
			continue
		}
		parts = append(parts, v)
		keys = append(keys, genKeyPart{
			Seq:   i + 1,
			Code:  c, // PG01..PG10
			Value: v,
		})
	}
	return strings.Join(parts, "|"), keys
}
