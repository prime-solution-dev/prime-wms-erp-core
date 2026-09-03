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
	"gorm.io/gorm/clause"
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
	Formulas         []PriceListFormulaCreateDTO
	// ReplaceAll wipes every price list row of the (company_code, site_code)
	// pairs present in Groups before inserting, inside the same transaction.
	ReplaceAll bool
}

// PriceListFormulaCreateDTO is the master formula row from the
// "price_list_formulars" sheet, loaded into table price_list_formulas.
type PriceListFormulaCreateDTO struct {
	ID          string
	FormulaCode string
	Name        string
	Uom         string
	FormulaType string
	Expression  string
	Params      string
	Rounding    int
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
	PdcPercent  float64
	Due         float64
	DuePercent  float64
	CreateBy    string
}

type PriceListGroupExtraCreateDTO struct {
	CompanyCode    string
	SiteCode       string
	GroupCode      string
	ExtraKey       string // GEN from price_list_group_extra.PGxx
	ConditionCode  string
	Operator       string
	ValueInt       float64
	RowNo          int // 1-based row in the extra sheet; each row is its own record
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
	// replace_all=true wipes the existing price list of every (company_code,
	// site_code) in the file before inserting, in the same transaction.
	req.ReplaceAll = parseBoolLoose(ctx.PostForm("replace_all"))

	return CreatePricelist(gormx, *req)
}

func CreatePricelist(gormx *gorm.DB, req CreatePricelistRequest) (*CreatePricelistResponse, error) {
	res := &CreatePricelistResponse{ResponseCode: 0, Message: "success"}
	const batchSize = 500

	err := gormx.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		groupKey := func(c, s, g string) string { return c + "|" + s + "|" + g }
		extraKey := func(c, s, g, ek string) string { return groupKey(c, s, g) + "|EXTRA|" + ek }

		if req.ReplaceAll {
			if err := deleteAllPriceListByScope(tx, req.Groups); err != nil {
				return err
			}
		}

		// ---------- map IDs ----------
		groupIDs := map[string]uuid.UUID{}
		for _, g := range req.Groups {
			groupIDs[groupKey(g.CompanyCode, g.SiteCode, g.GroupCode)] = uuid.New()
		}

		// Collect all group references from child entities
		referencedGroupKeys := make(map[string]bool)
		for _, t := range req.Terms {
			referencedGroupKeys[groupKey(t.CompanyCode, t.SiteCode, t.GroupCode)] = true
		}
		for _, e := range req.Extras {
			referencedGroupKeys[groupKey(e.CompanyCode, e.SiteCode, e.GroupCode)] = true
		}
		for _, s := range req.SubGroups {
			referencedGroupKeys[groupKey(s.CompanyCode, s.SiteCode, s.GroupCode)] = true
		}
		for _, k := range req.GroupKeys {
			referencedGroupKeys[groupKey(k.CompanyCode, k.SiteCode, k.GroupCode)] = true
		}

		// Query existing group IDs for groups that are referenced but not being upserted
		var missingGroupKeys []string
		for gk := range referencedGroupKeys {
			if _, exists := groupIDs[gk]; !exists {
				missingGroupKeys = append(missingGroupKeys, gk)
			}
		}

		if len(missingGroupKeys) > 0 {
			type GroupIDResult struct {
				ID          uuid.UUID `gorm:"column:id"`
				CompanyCode string    `gorm:"column:company_code"`
				SiteCode    string    `gorm:"column:site_code"`
				GroupCode   string    `gorm:"column:group_code"`
			}
			var existingGroups []GroupIDResult
			var conditions []string
			var args []interface{}

			for _, gk := range missingGroupKeys {
				parts := strings.Split(gk, "|")
				if len(parts) == 3 {
					conditions = append(conditions, "(company_code = ? AND site_code = ? AND group_code = ?)")
					args = append(args, parts[0], parts[1], parts[2])
				}
			}

			if len(conditions) > 0 {
				query := strings.Join(conditions, " OR ")
				if err := tx.Table("price_list_group").
					Select("id, company_code, site_code, group_code").
					Where(query, args...).
					Scan(&existingGroups).Error; err != nil {
					return err
				}

				for _, g := range existingGroups {
					gk := groupKey(g.CompanyCode, g.SiteCode, g.GroupCode)
					groupIDs[gk] = g.ID
				}
			}
		}

		// One id per source row, like extraIDs below: two rows may share the same
		// subgroup_key (same PG combination) while carrying different subgroup_code,
		// and dedup/ON CONFLICT run on subgroup_code. Keying ids by subgroup_key handed
		// both rows the same uuid and they collided on price_list_sub_group_pkey.
		subGroupIDs := make([]uuid.UUID, len(req.SubGroups))
		for i := range req.SubGroups {
			subGroupIDs[i] = uuid.New()
		}

		// One id per source row: the same (extra_key, condition_code) may legitimately
		// repeat with different operator / cond_range, and each is its own record.
		extraIDs := map[int]uuid.UUID{}
		extraKeyToGroupExtraID := map[string]uuid.UUID{} // first extra id per extra_key, for ExtraKeys FK
		for _, e := range req.Extras {
			ek := extraKey(e.CompanyCode, e.SiteCode, e.GroupCode, e.ExtraKey)
			id := uuid.New()
			extraIDs[e.RowNo] = id
			if _, ok := extraKeyToGroupExtraID[ek]; !ok {
				extraKeyToGroupExtraID[ek] = id
			}
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
			extraRecs = append(extraRecs, map[string]any{
				"id":                  extraIDs[e.RowNo],
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
		for i, s := range req.SubGroups {
			gk := groupKey(s.CompanyCode, s.SiteCode, s.GroupCode)
			subRecs = append(subRecs, map[string]any{
				"id":                            subGroupIDs[i],
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
			groupExtraID, ok := extraKeyToGroupExtraID[ek]
			if !ok || groupExtraID == uuid.Nil {
				continue // skip: no price_list_group_extra row for this extra_key (would violate FK)
			}
			extraKeyRecs = append(extraKeyRecs, map[string]any{
				"id":             uuid.New(),
				"group_extra_id": groupExtraID,
				"seq":            k.Seq,
				"code":           k.Code,
				"value":          k.Value,
			})
		}

		// subKeyRecs will be built after subgroup upsert to ensure correct IDs
		subKeyRecs := make([]map[string]any, 0)

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
			// Deduplicate groupRecs by compound key (company_code, site_code, group_code)
			groupRecsMap := make(map[string]map[string]any)
			for _, rec := range groupRecs {
				companyCode := rec["company_code"].(string)
				siteCode := rec["site_code"].(string)
				groupCode := rec["group_code"].(string)
				compoundKey := companyCode + "|" + siteCode + "|" + groupCode
				groupRecsMap[compoundKey] = rec
			}
			deduplicatedGroupRecs := make([]map[string]any, 0, len(groupRecsMap))
			for _, rec := range groupRecsMap {
				deduplicatedGroupRecs = append(deduplicatedGroupRecs, rec)
			}

			if err := tx.Table("price_list_group").
				Clauses(clause.OnConflict{
					Columns: []clause.Column{
						{Name: "group_code"},
					},
					DoUpdates: clause.Assignments(map[string]interface{}{
						"group_name":          gorm.Expr("COALESCE(NULLIF(excluded.group_name, ''), price_list_group.group_name)"),
						"currency":            gorm.Expr("COALESCE(NULLIF(excluded.currency, ''), price_list_group.currency)"),
						"effective_date":      gorm.Expr("COALESCE(excluded.effective_date, price_list_group.effective_date)"),
						"price_unit":          gorm.Expr("COALESCE(NULLIF(excluded.price_unit, 0), price_list_group.price_unit)"),
						"price_weight":        gorm.Expr("COALESCE(NULLIF(excluded.price_weight, 0), price_list_group.price_weight)"),
						"before_price_unit":   gorm.Expr("COALESCE(NULLIF(excluded.before_price_unit, 0), price_list_group.before_price_unit)"),
						"before_price_weight": gorm.Expr("COALESCE(NULLIF(excluded.before_price_weight, 0), price_list_group.before_price_weight)"),
						"remark":              gorm.Expr("COALESCE(NULLIF(excluded.remark, ''), price_list_group.remark)"),
						"update_by":           gorm.Expr("COALESCE(NULLIF(excluded.update_by, ''), price_list_group.update_by)"),
						"update_dtm":          gorm.Expr("excluded.update_dtm"),
					}),
				}).
				CreateInBatches(deduplicatedGroupRecs, batchSize).Error; err != nil {
				return err
			}

			// Query back actual IDs after upsert to ensure we have correct IDs for foreign key references
			type GroupIDResult struct {
				ID          uuid.UUID `gorm:"column:id"`
				CompanyCode string    `gorm:"column:company_code"`
				SiteCode    string    `gorm:"column:site_code"`
				GroupCode   string    `gorm:"column:group_code"`
			}
			var actualGroups []GroupIDResult

			// Build conditions for querying the groups we just upserted
			var conditions []string
			var args []interface{}
			for _, rec := range deduplicatedGroupRecs {
				conditions = append(conditions, "(company_code = ? AND site_code = ? AND group_code = ?)")
				args = append(args, rec["company_code"], rec["site_code"], rec["group_code"])
			}

			if len(conditions) > 0 {
				query := strings.Join(conditions, " OR ")
				if err := tx.Table("price_list_group").
					Select("id, company_code, site_code, group_code").
					Where(query, args...).
					Scan(&actualGroups).Error; err != nil {
					return err
				}

				// Update groupIDs map with actual IDs from database
				for _, g := range actualGroups {
					gk := groupKey(g.CompanyCode, g.SiteCode, g.GroupCode)
					groupIDs[gk] = g.ID
				}
			}

			// Rebuild child records with correct group IDs
			termRecs = make([]map[string]any, 0, len(req.Terms))
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

			extraRecs = make([]map[string]any, 0, len(req.Extras))
			for _, e := range req.Extras {
				gk := groupKey(e.CompanyCode, e.SiteCode, e.GroupCode)
				extraRecs = append(extraRecs, map[string]any{
					"id":                  extraIDs[e.RowNo],
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

			subRecs = make([]map[string]any, 0, len(req.SubGroups))
			for i, s := range req.SubGroups {
				gk := groupKey(s.CompanyCode, s.SiteCode, s.GroupCode)
				subRecs = append(subRecs, map[string]any{
					"id":                            subGroupIDs[i],
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

			groupKeyRecs = make([]map[string]any, 0, len(req.GroupKeys))
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
		}
		// Terms and extras are a full replacement for the groups in this upload:
		// delete first so re-uploading the same file does not accumulate rows
		// (same contract price_list_group_key / price_list_sub_group_key already use).
		touchedGroupIDs := make([]uuid.UUID, 0, len(req.Terms)+len(req.Extras))
		seenTouchedGroup := map[uuid.UUID]bool{}
		for _, t := range req.Terms {
			if id, ok := groupIDs[groupKey(t.CompanyCode, t.SiteCode, t.GroupCode)]; ok && !seenTouchedGroup[id] {
				seenTouchedGroup[id] = true
				touchedGroupIDs = append(touchedGroupIDs, id)
			}
		}
		for _, e := range req.Extras {
			if id, ok := groupIDs[groupKey(e.CompanyCode, e.SiteCode, e.GroupCode)]; ok && !seenTouchedGroup[id] {
				seenTouchedGroup[id] = true
				touchedGroupIDs = append(touchedGroupIDs, id)
			}
		}
		if len(touchedGroupIDs) > 0 {
			if err := tx.Exec(
				"DELETE FROM price_list_group_extra_key k USING price_list_group_extra e "+
					"WHERE k.group_extra_id = e.id AND e.price_list_group_id IN ?", touchedGroupIDs,
			).Error; err != nil {
				return err
			}
			if err := tx.Table("price_list_group_extra").
				Where("price_list_group_id IN ?", touchedGroupIDs).Delete(nil).Error; err != nil {
				return err
			}
			if err := tx.Table("price_list_group_term").
				Where("price_list_group_id IN ?", touchedGroupIDs).Delete(nil).Error; err != nil {
				return err
			}
		}

		if len(termRecs) > 0 {
			if err := tx.Table("price_list_group_term").CreateInBatches(termRecs, batchSize).Error; err != nil {
				return err
			}
		}
		if len(extraRecs) > 0 {
			deduplicatedExtraRecs := extraRecs

			if err := tx.Table("price_list_group_extra").
				Clauses(clause.OnConflict{
					Columns: []clause.Column{{Name: "id"}},
					DoUpdates: clause.Assignments(map[string]interface{}{
						"price_list_group_id": gorm.Expr("excluded.price_list_group_id"),
						"extra_key":           gorm.Expr("excluded.extra_key"),
						"condition_code":      gorm.Expr("excluded.condition_code"),
						"operator":            gorm.Expr("excluded.operator"),
						"value_int":           gorm.Expr("excluded.value_int"),
						"length_extra_key":    gorm.Expr("excluded.length_extra_key"),
						"cond_range_min":      gorm.Expr("excluded.cond_range_min"),
						"cond_range_max":      gorm.Expr("excluded.cond_range_max"),
						"update_by":           gorm.Expr("excluded.update_by"),
						"update_dtm":          gorm.Expr("excluded.update_dtm"),
					}),
				}).
				CreateInBatches(deduplicatedExtraRecs, batchSize).Error; err != nil {
				return err
			}

			// Query back actual extra IDs after upsert to ensure we have correct IDs for foreign key references
			type ExtraIDResult struct {
				ID               uuid.UUID `gorm:"column:id"`
				PriceListGroupID uuid.UUID `gorm:"column:price_list_group_id"`
				ExtraKey         string    `gorm:"column:extra_key"`
				ConditionCode    string    `gorm:"column:condition_code"`
			}
			var actualExtras []ExtraIDResult

			// Collect IDs we inserted to query back
			insertedIDs := make([]uuid.UUID, 0, len(deduplicatedExtraRecs))
			for _, rec := range deduplicatedExtraRecs {
				var id uuid.UUID
				switch v := rec["id"].(type) {
				case uuid.UUID:
					id = v
				case string:
					if parsed, err := uuid.Parse(v); err == nil {
						id = parsed
					} else {
						continue
					}
				default:
					continue
				}
				insertedIDs = append(insertedIDs, id)
			}

			// Query back extras by their IDs
			if len(insertedIDs) > 0 {
				if err := tx.Table("price_list_group_extra").
					Select("id, price_list_group_id, extra_key, condition_code").
					Where("id IN ?", insertedIDs).
					Scan(&actualExtras).Error; err != nil {
					return err
				}
			}

			// Map extra_key to first ID found (for ExtraKeys FK)
			// Key format: company_code|site_code|group_code|EXTRA|extra_key
			extraKeyToGroupExtraIDFromDB := make(map[string]uuid.UUID)
			for _, e := range actualExtras {
				// Find matching request extra to get company_code, site_code, group_code
				for _, reqExtra := range req.Extras {
					if reqExtra.ExtraKey == e.ExtraKey && reqExtra.ConditionCode == e.ConditionCode {
						ek := extraKey(reqExtra.CompanyCode, reqExtra.SiteCode, reqExtra.GroupCode, reqExtra.ExtraKey)
						if _, ok := extraKeyToGroupExtraIDFromDB[ek]; !ok {
							extraKeyToGroupExtraIDFromDB[ek] = e.ID
						}
						break
					}
				}
			}

			// Rebuild extraKeyRecs with actual IDs from database
			extraKeyRecs = make([]map[string]any, 0, len(req.ExtraKeys))
			for _, k := range req.ExtraKeys {
				ek := extraKey(k.CompanyCode, k.SiteCode, k.GroupCode, k.ExtraKey)
				groupExtraID, ok := extraKeyToGroupExtraIDFromDB[ek]
				if !ok || groupExtraID == uuid.Nil {
					// Fallback: find any extra with matching extra_key (use first one)
					for _, e := range actualExtras {
						if e.ExtraKey == k.ExtraKey {
							groupExtraID = e.ID
							extraKeyToGroupExtraIDFromDB[ek] = e.ID
							break
						}
					}
					if groupExtraID == uuid.Nil {
						continue // skip: no price_list_group_extra row for this extra_key
					}
				}
				extraKeyRecs = append(extraKeyRecs, map[string]any{
					"id":             uuid.New(),
					"group_extra_id": groupExtraID,
					"seq":            k.Seq,
					"code":           k.Code,
					"value":          k.Value,
				})
			}
		}
		if len(subRecs) > 0 {
			// Deduplicate subRecs by subgroup_code (keep last occurrence)
			subRecsMap := make(map[string]map[string]any)
			for _, rec := range subRecs {
				subGroupCode := rec["subgroup_code"].(string)
				subRecsMap[subGroupCode] = rec
			}
			deduplicatedSubRecs := make([]map[string]any, 0, len(subRecsMap))
			for _, rec := range subRecsMap {
				deduplicatedSubRecs = append(deduplicatedSubRecs, rec)
			}

			if err := tx.Table("price_list_sub_group").
				Clauses(clause.OnConflict{
					Columns: []clause.Column{
						{Name: "subgroup_code"},
					},
					DoUpdates: clause.Assignments(map[string]interface{}{
						"subgroup_key":                  gorm.Expr("COALESCE(NULLIF(excluded.subgroup_key, ''), price_list_sub_group.subgroup_key)"),
						"is_trading":                    gorm.Expr("excluded.is_trading"),
						"price_unit":                    gorm.Expr("COALESCE(NULLIF(excluded.price_unit, 0), price_list_sub_group.price_unit)"),
						"extra_price_unit":              gorm.Expr("COALESCE(NULLIF(excluded.extra_price_unit, 0), price_list_sub_group.extra_price_unit)"),
						"total_net_price_unit":          gorm.Expr("COALESCE(NULLIF(excluded.total_net_price_unit, 0), price_list_sub_group.total_net_price_unit)"),
						"price_weight":                  gorm.Expr("COALESCE(NULLIF(excluded.price_weight, 0), price_list_sub_group.price_weight)"),
						"extra_price_weight":            gorm.Expr("COALESCE(NULLIF(excluded.extra_price_weight, 0), price_list_sub_group.extra_price_weight)"),
						"term_price_weight":             gorm.Expr("COALESCE(NULLIF(excluded.term_price_weight, 0), price_list_sub_group.term_price_weight)"),
						"total_net_price_weight":        gorm.Expr("COALESCE(NULLIF(excluded.total_net_price_weight, 0), price_list_sub_group.total_net_price_weight)"),
						"before_price_unit":             gorm.Expr("COALESCE(NULLIF(excluded.before_price_unit, 0), price_list_sub_group.before_price_unit)"),
						"before_extra_price_unit":       gorm.Expr("COALESCE(NULLIF(excluded.before_extra_price_unit, 0), price_list_sub_group.before_extra_price_unit)"),
						"before_term_price_unit":        gorm.Expr("COALESCE(NULLIF(excluded.before_term_price_unit, 0), price_list_sub_group.before_term_price_unit)"),
						"before_total_net_price_unit":   gorm.Expr("COALESCE(NULLIF(excluded.before_total_net_price_unit, 0), price_list_sub_group.before_total_net_price_unit)"),
						"before_price_weight":           gorm.Expr("COALESCE(NULLIF(excluded.before_price_weight, 0), price_list_sub_group.before_price_weight)"),
						"before_extra_price_weight":     gorm.Expr("COALESCE(NULLIF(excluded.before_extra_price_weight, 0), price_list_sub_group.before_extra_price_weight)"),
						"before_term_price_weight":      gorm.Expr("COALESCE(NULLIF(excluded.before_term_price_weight, 0), price_list_sub_group.before_term_price_weight)"),
						"before_total_net_price_weight": gorm.Expr("COALESCE(NULLIF(excluded.before_total_net_price_weight, 0), price_list_sub_group.before_total_net_price_weight)"),
						"effective_date":                gorm.Expr("COALESCE(excluded.effective_date, price_list_sub_group.effective_date)"),
						"remark":                        gorm.Expr("COALESCE(NULLIF(excluded.remark, ''), price_list_sub_group.remark)"),
						"udf_json":                      gorm.Expr("COALESCE(excluded.udf_json, price_list_sub_group.udf_json)"),
						"update_by":                     gorm.Expr("COALESCE(NULLIF(excluded.update_by, ''), price_list_sub_group.update_by)"),
						"update_dtm":                    gorm.Expr("excluded.update_dtm"),
					}),
				}).
				CreateInBatches(deduplicatedSubRecs, batchSize).Error; err != nil {
				return err
			}

			// Query back actual subgroup IDs after upsert to ensure we have correct IDs for foreign key references
			type SubGroupIDResult struct {
				ID           uuid.UUID `gorm:"column:id"`
				SubGroupCode string    `gorm:"column:subgroup_code"`
			}
			var actualSubGroups []SubGroupIDResult

			// Build conditions for querying the subgroups we just upserted
			var subConditions []string
			var subArgs []interface{}
			for _, rec := range deduplicatedSubRecs {
				subConditions = append(subConditions, "subgroup_code = ?")
				subArgs = append(subArgs, rec["subgroup_code"])
			}

			// Map subgroup_code to actual ID from database
			subGroupCodeToID := make(map[string]uuid.UUID)
			if len(subConditions) > 0 {
				subQuery := strings.Join(subConditions, " OR ")
				if err := tx.Table("price_list_sub_group").
					Select("id, subgroup_code").
					Where(subQuery, subArgs...).
					Scan(&actualSubGroups).Error; err != nil {
					return err
				}

				for _, sg := range actualSubGroups {
					subGroupCodeToID[sg.SubGroupCode] = sg.ID
				}
			}

			// Rebuild subKeyRecs with correct subgroup IDs from database
			subKeyRecs = make([]map[string]any, 0, len(req.SubGroupKeys))
			for _, k := range req.SubGroupKeys {
				// Find the subgroup_code for this key by matching SubGroupKey
				var subGroupCode string
				for _, s := range req.SubGroups {
					if s.CompanyCode == k.CompanyCode && s.SiteCode == k.SiteCode &&
						s.GroupCode == k.GroupCode && s.SubGroupKey == k.SubGroupKey {
						subGroupCode = s.SubGroupCode
						break
					}
				}

				// Only add if we found a valid subgroup_code and it exists in the database
				if subGroupCode != "" {
					if actualID, ok := subGroupCodeToID[subGroupCode]; ok {
						subKeyRecs = append(subKeyRecs, map[string]any{
							"id":           uuid.New(),
							"sub_group_id": actualID,
							"seq":          k.Seq,
							"code":         k.Code,
							"value":        k.Value,
						})
					}
				}
			}
		} else {
			// If no subgroups to upsert, but we have SubGroupKeys, we still need to build subKeyRecs
			// Query existing subgroups by subgroup_code from the request
			if len(req.SubGroupKeys) > 0 {
				type SubGroupIDResult struct {
					ID           uuid.UUID `gorm:"column:id"`
					SubGroupCode string    `gorm:"column:subgroup_code"`
				}
				var actualSubGroups []SubGroupIDResult

				// Collect unique subgroup_codes from SubGroups
				subGroupCodes := make(map[string]bool)
				for _, s := range req.SubGroups {
					if s.SubGroupCode != "" {
						subGroupCodes[s.SubGroupCode] = true
					}
				}

				if len(subGroupCodes) > 0 {
					var subConditions []string
					var subArgs []interface{}
					for code := range subGroupCodes {
						subConditions = append(subConditions, "subgroup_code = ?")
						subArgs = append(subArgs, code)
					}

					subQuery := strings.Join(subConditions, " OR ")
					if err := tx.Table("price_list_sub_group").
						Select("id, subgroup_code").
						Where(subQuery, subArgs...).
						Scan(&actualSubGroups).Error; err != nil {
						return err
					}

					subGroupCodeToID := make(map[string]uuid.UUID)
					for _, sg := range actualSubGroups {
						subGroupCodeToID[sg.SubGroupCode] = sg.ID
					}

					// Build subKeyRecs with correct subgroup IDs from database
					subKeyRecs = make([]map[string]any, 0, len(req.SubGroupKeys))
					for _, k := range req.SubGroupKeys {
						// Find the subgroup_code for this key by matching SubGroupKey
						var subGroupCode string
						for _, s := range req.SubGroups {
							if s.CompanyCode == k.CompanyCode && s.SiteCode == k.SiteCode &&
								s.GroupCode == k.GroupCode && s.SubGroupKey == k.SubGroupKey {
								subGroupCode = s.SubGroupCode
								break
							}
						}

						// Only add if we found a valid subgroup_code and it exists in the database
						if subGroupCode != "" {
							if actualID, ok := subGroupCodeToID[subGroupCode]; ok {
								subKeyRecs = append(subKeyRecs, map[string]any{
									"id":           uuid.New(),
									"sub_group_id": actualID,
									"seq":          k.Seq,
									"code":         k.Code,
									"value":        k.Value,
								})
							}
						}
					}
				}
			}
		}
		if len(groupKeyRecs) > 0 {
			// Delete existing group keys for the groups being processed
			groupIDsToDelete := make([]uuid.UUID, 0, len(groupKeyRecs))
			seenGroupIDs := make(map[uuid.UUID]bool)
			for _, rec := range groupKeyRecs {
				var groupID uuid.UUID
				switch v := rec["price_list_group_id"].(type) {
				case uuid.UUID:
					groupID = v
				case string:
					var err error
					groupID, err = uuid.Parse(v)
					if err != nil {
						continue
					}
				default:
					continue
				}
				if !seenGroupIDs[groupID] {
					groupIDsToDelete = append(groupIDsToDelete, groupID)
					seenGroupIDs[groupID] = true
				}
			}
			if len(groupIDsToDelete) > 0 {
				if err := tx.Table("price_list_group_key").
					Where("price_list_group_id IN ?", groupIDsToDelete).
					Delete(nil).Error; err != nil {
					return err
				}
			}

			// Insert new group keys
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
			// Delete existing subgroup keys for the subgroups being processed
			subGroupIDsToDelete := make([]uuid.UUID, 0, len(subKeyRecs))
			seenSubGroupIDs := make(map[uuid.UUID]bool)
			for _, rec := range subKeyRecs {
				var subGroupID uuid.UUID
				switch v := rec["sub_group_id"].(type) {
				case uuid.UUID:
					subGroupID = v
				case string:
					var err error
					subGroupID, err = uuid.Parse(v)
					if err != nil {
						continue
					}
				default:
					continue
				}
				if !seenSubGroupIDs[subGroupID] {
					subGroupIDsToDelete = append(subGroupIDsToDelete, subGroupID)
					seenSubGroupIDs[subGroupID] = true
				}
			}
			if len(subGroupIDsToDelete) > 0 {
				if err := tx.Table("price_list_sub_group_key").
					Where("sub_group_id IN ?", subGroupIDsToDelete).
					Delete(nil).Error; err != nil {
					return err
				}
			}

			// Insert new subgroup keys
			if err := tx.Table("price_list_sub_group_key").CreateInBatches(subKeyRecs, batchSize).Error; err != nil {
				return err
			}
		}
		// Master formulas must exist before formulas_map rows reference them.
		if len(req.Formulas) > 0 {
			formulaRecs := make([]map[string]any, 0, len(req.Formulas))
			for _, f := range req.Formulas {
				id, err := uuid.Parse(f.ID)
				if err != nil {
					id = uuid.New()
				}
				params := f.Params
				if params == "" {
					params = "{}"
				}
				formulaRecs = append(formulaRecs, map[string]any{
					"id":           id,
					"formula_code": f.FormulaCode,
					"name":         f.Name,
					"uom":          f.Uom,
					"formula_type": f.FormulaType,
					"expression":   f.Expression,
					"params":       params,
					"rounding":     f.Rounding,
					"create_dtm":   now,
				})
			}
			if err := tx.Table("price_list_formulas").
				Clauses(clause.OnConflict{
					Columns: []clause.Column{{Name: "formula_code"}},
					DoUpdates: clause.AssignmentColumns([]string{
						"name", "uom", "formula_type", "expression", "params", "rounding",
					}),
				}).
				CreateInBatches(formulaRecs, batchSize).Error; err != nil {
				return err
			}
		}

		if len(subGroupFormulasRecs) > 0 {
			// Delete existing subgroup formulas for the subgroups being processed
			subGroupCodesToDelete := make([]string, 0, len(subGroupFormulasRecs))
			seenSubGroupCodes := make(map[string]bool)
			for _, rec := range subGroupFormulasRecs {
				if subGroupCode, ok := rec["price_list_subgroup_code"].(string); ok && subGroupCode != "" {
					if !seenSubGroupCodes[subGroupCode] {
						subGroupCodesToDelete = append(subGroupCodesToDelete, subGroupCode)
						seenSubGroupCodes[subGroupCode] = true
					}
				}
			}
			if len(subGroupCodesToDelete) > 0 {
				if err := tx.Table("price_list_subgroup_formulas_map").
					Where("price_list_subgroup_code IN ?", subGroupCodesToDelete).
					Delete(nil).Error; err != nil {
					return err
				}
			}

			// Insert new subgroup formulas
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

// deleteAllPriceListByScope removes every price list row belonging to the
// (company_code, site_code) pairs of the uploaded groups, children first so the
// FKs hold. History is wiped too: price_list_group_history has no ON DELETE
// CASCADE, so leaving it would make the final group delete fail. Only the
// price_list_formulas master is left alone.
func deleteAllPriceListByScope(tx *gorm.DB, groups []PriceListGroupCreateDTO) error {
	type scope struct{ company, site string }
	seen := map[scope]bool{}
	var where []string
	var args []interface{}
	for _, g := range groups {
		sc := scope{g.CompanyCode, g.SiteCode}
		if seen[sc] {
			continue
		}
		seen[sc] = true
		where = append(where, "(company_code = ? AND site_code = ?)")
		args = append(args, g.CompanyCode, g.SiteCode)
	}
	if len(where) == 0 {
		return nil
	}

	var groupIDs []uuid.UUID
	if err := tx.Table("price_list_group").
		Where(strings.Join(where, " OR "), args...).
		Pluck("id", &groupIDs).Error; err != nil {
		return err
	}
	if len(groupIDs) == 0 {
		return nil
	}

	stmts := []string{
		"DELETE FROM price_list_subgroup_formulas_map m USING price_list_sub_group sg " +
			"WHERE m.price_list_subgroup_code = sg.subgroup_code AND sg.price_list_group_id IN ?",
		"DELETE FROM price_list_sub_group_key k USING price_list_sub_group sg " +
			"WHERE k.sub_group_id = sg.id AND sg.price_list_group_id IN ?",
		"DELETE FROM price_list_sub_group WHERE price_list_group_id IN ?",
		"DELETE FROM price_list_group_extra_key k USING price_list_group_extra e " +
			"WHERE k.group_extra_id = e.id AND e.price_list_group_id IN ?",
		"DELETE FROM price_list_group_extra WHERE price_list_group_id IN ?",
		"DELETE FROM price_list_group_term WHERE price_list_group_id IN ?",
		"DELETE FROM price_list_group_key WHERE price_list_group_id IN ?",
		"DELETE FROM price_list_sub_group_history WHERE price_list_group_id IN ?",
		"DELETE FROM price_list_group_history WHERE price_list_group_id IN ?",
		"DELETE FROM price_list_group WHERE id IN ?",
	}
	for _, q := range stmts {
		if err := tx.Exec(q, groupIDs).Error; err != nil {
			return err
		}
	}
	return nil
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
	// Excel percent cells come back already formatted ("1.0%"), not as "0.01".
	// ponytail: precision follows the sheet's own display format (0.0% here);
	// read the raw cell value if a template ever needs more decimals than it shows.
	parsePercent := func(s string) float64 {
		s = strings.TrimSpace(s)
		if strings.HasSuffix(s, "%") {
			v, _ := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(s, "%")), 64)
			return v / 100
		}
		return parseFloat(s)
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
	seenGroupCode := map[string]int{}
	for i, r := range groupRows {
		if r["company_code"] == "" || r["site_code"] == "" || r["group_code"] == "" {
			return nil, fmt.Errorf("price_list_group: company_code, site_code, group_code are required")
		}
		if first, dup := seenGroupCode[r["group_code"]]; dup {
			return nil, fmt.Errorf("price_list_group: group_code %q ซ้ำ (แถว %d และ %d) — แต่ละกลุ่มต้องมี group_code ไม่ซ้ำกัน", r["group_code"], first+2, i+2)
		}
		seenGroupCode[r["group_code"]] = i

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
			PdcPercent:  parsePercent(r["pdc_percent"]),
			Due:         parseFloat(r["due"]),
			DuePercent:  parsePercent(r["due_percent"]),
			CreateBy:    r["create_by"],
		})
	}

	// ---- extra : gen extra_key + create extra_key rows from same PG01..PG10 ----
	extraRows, _ := readSheet("price_list_group_extra")
	for i, r := range extraRows {
		if r["company_code"] == "" || r["site_code"] == "" || r["group_code"] == "" {
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
			RowNo:          i + 2,
			Operator:       r["operator"],
			ValueInt:       parseFloat(r["value_int"]),
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
	seenSubGroupCode := map[string]int{}
	for i, r := range subRows {
		if r["company_code"] == "" || r["site_code"] == "" || r["group_code"] == "" {
			return nil, fmt.Errorf("price_list_sub_group: company_code, site_code, group_code are required")
		}
		if code := r["subgroup_code"]; code != "" {
			if first, dup := seenSubGroupCode[code]; dup {
				return nil, fmt.Errorf("price_list_sub_group: subgroup_code %q ซ้ำ (แถว %d และ %d)", code, first+2, i+2)
			}
			seenSubGroupCode[code] = i
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

	// ---- price_list_formulars (master formulas; optional sheet) ----
	knownFormulaCodes := map[string]bool{}
	formulaRows, _ := readSheet("price_list_formulars")
	for i, r := range formulaRows {
		if r["formula_code"] == "" {
			return nil, fmt.Errorf("price_list_formulars (แถว %d): formula_code is required", i+2)
		}
		if r["params"] != "" && !json.Valid([]byte(r["params"])) {
			return nil, fmt.Errorf("price_list_formulars (formula_code=%s): params invalid json", r["formula_code"])
		}
		knownFormulaCodes[r["formula_code"]] = true
		req.Formulas = append(req.Formulas, PriceListFormulaCreateDTO{
			ID:          r["id"],
			FormulaCode: r["formula_code"],
			Name:        r["name"],
			Uom:         r["uom"],
			FormulaType: r["formula_type"],
			Expression:  r["expression"],
			Params:      r["params"],
			Rounding:    parseInt(r["rounding"]),
		})
	}

	// ---- formulas_map ----
	formulasRows, err := readSheet("formulas_map")
	if err != nil {
		return nil, err
	}
	for _, r := range formulasRows {
		if r["subgroup_code"] == "" || r["formula_code_default"] == "" || r["formula_code_convert"] == "" {
			return nil, fmt.Errorf("formulas_map: subgroup_code, formula_code are required")
		}
		// Only cross-check when the master sheet is present; otherwise the codes
		// must already exist in price_list_formulas and the FK will say so.
		if len(knownFormulaCodes) > 0 {
			for _, code := range []string{r["formula_code_default"], r["formula_code_convert"]} {
				if !knownFormulaCodes[code] {
					return nil, fmt.Errorf("formulas_map (subgroup_code=%s): formula_code %q ไม่มีใน sheet price_list_formulars", r["subgroup_code"], code)
				}
			}
		}
		req.SubGroupFormulas = append(req.SubGroupFormulas, PriceListSubGroupFormulasCreateDTO{
			SubGroupCode: r["subgroup_code"],
			FormulaCode:  r["formula_code_default"],
			IsDefault:    true,
		})

		req.SubGroupFormulas = append(req.SubGroupFormulas, PriceListSubGroupFormulasCreateDTO{
			SubGroupCode: r["subgroup_code"],
			FormulaCode:  r["formula_code_convert"],
			IsDefault:    false,
		})
	}

	return req, nil
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
