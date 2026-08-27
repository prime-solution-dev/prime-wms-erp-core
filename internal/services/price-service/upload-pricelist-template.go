package priceService

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"prime-erp-core/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// Default context for the TMI Pricelist template, which omits company/site columns.
const (
	defaultTemplateCompanyCode = "09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3"
	defaultTemplateSiteCode    = "TMI_WH"
	defaultTemplateSheet       = "Pricelist"
)

// UploadPricelistTemplateMultipart accepts the single-sheet "Pricelist" upload
// template (columns: Product Group, formula_code_default, formula_code_convert,
// Price_unit, Price_weight, Product code, Product name, Product Group 1..10) and
// loads it into price_list_sub_group (+ price_list_sub_group_key, and optionally
// price_list_subgroup_formulas_map). It reuses CreatePricelist for the upsert so
// existing price_list_group rows are matched by group_code.
//
// Optional multipart form fields:
//   - company_code    (default 09dcb573-...)
//   - site_code       (default TMI_WH)
//   - sheet           (default "Pricelist"; falls back to first sheet if absent)
//   - create_by       (default "system")
//   - include_formulas ("true"/"1"/"yes" to also write formulas_map; default off)
func UploadPricelistTemplateMultipart(ctx *gin.Context) (interface{}, error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	file, _, err := ctx.Request.FormFile("files")
	if err != nil {
		return &CreatePricelistResponse{ResponseCode: 1, Message: fmt.Sprintf("missing file (form-data key: files): %v", err)}, nil
	}
	defer file.Close()

	opts := templateParseOptions{
		CompanyCode:     firstNonEmpty(ctx.PostForm("company_code"), defaultTemplateCompanyCode),
		SiteCode:        firstNonEmpty(ctx.PostForm("site_code"), defaultTemplateSiteCode),
		Sheet:           strings.TrimSpace(ctx.PostForm("sheet")),
		CreateBy:        firstNonEmpty(ctx.PostForm("create_by"), "system"),
		IncludeFormulas: parseBoolLoose(ctx.PostForm("include_formulas")),
	}

	req, err := buildCreatePricelistRequestFromTemplate(file, opts)
	if err != nil {
		return &CreatePricelistResponse{ResponseCode: 1, Message: err.Error()}, nil
	}

	// Dedup by (price_list_group_id, subgroup_key): remove any existing rows that
	// share a subgroup_key with this upload (placeholder SGxx rows and prior loads)
	// before inserting fresh, so there is exactly one row per (group, subgroup_key).
	// Delete + CreatePricelist run in one transaction (nested tx = savepoint).
	txErr := gormx.Transaction(func(tx *gorm.DB) error {
		if err := deleteExistingSubgroupsByGroupKey(tx, opts.CompanyCode, opts.SiteCode, req.SubGroups); err != nil {
			return err
		}
		resp, err := CreatePricelist(tx, *req)
		if err != nil {
			return err
		}
		if resp.ResponseCode != 0 {
			return fmt.Errorf("%s", resp.Message)
		}
		return nil
	})
	if txErr != nil {
		return &CreatePricelistResponse{ResponseCode: 1, Message: txErr.Error()}, nil
	}
	return &CreatePricelistResponse{ResponseCode: 0, Message: "success"}, nil
}

// deleteExistingSubgroupsByGroupKey removes price_list_sub_group rows (and their
// keys) that match one of the uploaded (price_list_group_id, subgroup_key) pairs.
// Groups are resolved by (company_code, site_code, group_code).
func deleteExistingSubgroupsByGroupKey(tx *gorm.DB, companyCode, siteCode string, subs []PriceListSubGroupCreateDTO) error {
	if len(subs) == 0 {
		return nil
	}

	// Resolve group_code -> id for the groups referenced by this upload.
	groupCodeSet := map[string]struct{}{}
	for _, s := range subs {
		groupCodeSet[s.GroupCode] = struct{}{}
	}
	groupCodes := make([]string, 0, len(groupCodeSet))
	for gc := range groupCodeSet {
		groupCodes = append(groupCodes, gc)
	}

	var groups []struct {
		ID        uuid.UUID `gorm:"column:id"`
		GroupCode string    `gorm:"column:group_code"`
	}
	if err := tx.Table("price_list_group").
		Select("id, group_code").
		Where("company_code = ? AND site_code = ? AND group_code IN ?", companyCode, siteCode, groupCodes).
		Scan(&groups).Error; err != nil {
		return err
	}
	groupID := make(map[string]uuid.UUID, len(groups))
	for _, g := range groups {
		groupID[g.GroupCode] = g.ID
	}

	// Build (group_id, subgroup_key) pairs; skip groups not found (nothing to delete).
	placeholders := make([]string, 0, len(subs))
	args := make([]interface{}, 0, len(subs)*2)
	for _, s := range subs {
		gid, ok := groupID[s.GroupCode]
		if !ok || s.SubGroupKey == "" {
			continue
		}
		placeholders = append(placeholders, "(?,?)")
		args = append(args, gid, s.SubGroupKey)
	}
	if len(placeholders) == 0 {
		return nil
	}
	tuples := strings.Join(placeholders, ",")

	// Delete dependents first (FKs), then the subgroups:
	//   - formulas_map FK references price_list_sub_group.subgroup_code
	//   - sub_group_key   FK references price_list_sub_group.id
	if err := tx.Exec(
		"DELETE FROM price_list_subgroup_formulas_map m USING price_list_sub_group sg "+
			"WHERE m.price_list_subgroup_code = sg.subgroup_code AND (sg.price_list_group_id, sg.subgroup_key) IN ("+tuples+")",
		args...,
	).Error; err != nil {
		return err
	}
	if err := tx.Exec(
		"DELETE FROM price_list_sub_group_key k USING price_list_sub_group sg "+
			"WHERE k.sub_group_id = sg.id AND (sg.price_list_group_id, sg.subgroup_key) IN ("+tuples+")",
		args...,
	).Error; err != nil {
		return err
	}
	if err := tx.Exec(
		"DELETE FROM price_list_sub_group sg WHERE (sg.price_list_group_id, sg.subgroup_key) IN ("+tuples+")",
		args...,
	).Error; err != nil {
		return err
	}
	return nil
}

type templateParseOptions struct {
	CompanyCode     string
	SiteCode        string
	Sheet           string
	CreateBy        string
	IncludeFormulas bool
}

func buildCreatePricelistRequestFromTemplate(r io.Reader, opts templateParseOptions) (*CreatePricelistRequest, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("open excel failed: %w", err)
	}
	defer func() { _ = f.Close() }()

	sheet := opts.Sheet
	if sheet == "" {
		sheet = defaultTemplateSheet
	}
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		// Fall back to the first sheet if the named one is missing/empty.
		if list := f.GetSheetList(); len(list) > 0 {
			sheet = list[0]
			rows, err = f.GetRows(sheet)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("read sheet %q: %w", sheet, err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("sheet %q has no data rows", sheet)
	}

	// Map header name -> column index.
	col := map[string]int{}
	for i, h := range rows[0] {
		col[strings.TrimSpace(h)] = i
	}
	need := func(name string) (int, error) {
		if idx, ok := col[name]; ok {
			return idx, nil
		}
		return -1, fmt.Errorf("template missing required column %q", name)
	}

	cGroup, err := need("Product Group")
	if err != nil {
		return nil, err
	}
	cCode, err := need("Product code")
	if err != nil {
		return nil, err
	}
	cUnit, err := need("Price_unit")
	if err != nil {
		return nil, err
	}
	cWeight, err := need("Price_weight")
	if err != nil {
		return nil, err
	}
	cName := col["Product name"]           // optional -> remark
	cFDefault := col["formula_code_default"]
	cFConvert := col["formula_code_convert"]

	// PG01..PG10 come from the "Product Group N" columns; seq is the absolute
	// PG position (matches genKeyFromCols / seed-price-list.go).
	type pgCol struct {
		idx  int
		code string // PG01..PG10
		seq  int
	}
	var pgColumns []pgCol
	for n := 1; n <= 10; n++ {
		if idx, ok := col[fmt.Sprintf("Product Group %d", n)]; ok {
			pgColumns = append(pgColumns, pgCol{idx: idx, code: fmt.Sprintf("PG%02d", n), seq: n})
		}
	}
	if len(pgColumns) == 0 {
		return nil, fmt.Errorf("template missing any 'Product Group N' columns")
	}

	cellAt := func(row []string, idx int) string {
		if idx >= 0 && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	req := &CreatePricelistRequest{}
	seen := map[string]bool{}

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		groupCode := cellAt(row, cGroup)
		subCode := cellAt(row, cCode)
		if groupCode == "" && subCode == "" {
			continue // blank row
		}
		if groupCode == "" {
			return nil, fmt.Errorf("row %d: 'Product Group' (group_code) is required", i+1)
		}
		if subCode == "" {
			return nil, fmt.Errorf("row %d: 'Product code' (subgroup_code) is required", i+1)
		}
		if seen[subCode] {
			return nil, fmt.Errorf("row %d: duplicate Product code %q", i+1, subCode)
		}
		seen[subCode] = true

		// Build subgroup_key + key rows from the PG columns.
		var parts []string
		for _, pc := range pgColumns {
			v := cellAt(row, pc.idx)
			if v == "" {
				continue
			}
			parts = append(parts, v)
			req.SubGroupKeys = append(req.SubGroupKeys, PriceListSubGroupKeyDTO{
				CompanyCode: opts.CompanyCode,
				SiteCode:    opts.SiteCode,
				GroupCode:   groupCode,
				SubGroupKey: "", // set below once fully built
				Seq:         pc.seq,
				Code:        pc.code,
				Value:       v,
			})
		}
		subKey := strings.Join(parts, "|")
		if subKey == "" {
			return nil, fmt.Errorf("row %d (Product code=%s): no 'Product Group N' value to build subgroup_key", i+1, subCode)
		}
		// Backfill SubGroupKey on the key DTOs we just appended for this row.
		for j := len(req.SubGroupKeys) - len(parts); j < len(req.SubGroupKeys); j++ {
			req.SubGroupKeys[j].SubGroupKey = subKey
		}

		priceUnit := parseFloatLoose(cellAt(row, cUnit))
		priceWeight := parseFloatLoose(cellAt(row, cWeight))

		// Excel Price_unit/Price_weight map to total_net_price_* (and before_total_net_*).
		req.SubGroups = append(req.SubGroups, PriceListSubGroupCreateDTO{
			CompanyCode:               opts.CompanyCode,
			SiteCode:                  opts.SiteCode,
			GroupCode:                 groupCode,
			SubGroupKey:               subKey,
			SubGroupCode:              subCode,
			IsTrading:                 false,
			TotalNetPriceUnit:         priceUnit,
			TotalNetPriceWeight:       priceWeight,
			BeforeTotalNetPriceUnit:   priceUnit,
			BeforeTotalNetPriceWeight: priceWeight,
			Remark:                    cellAt(row, cName),
			CreateBy:                  opts.CreateBy,
		})

		if opts.IncludeFormulas {
			if fd := cellAt(row, cFDefault); fd != "" {
				req.SubGroupFormulas = append(req.SubGroupFormulas, PriceListSubGroupFormulasCreateDTO{
					SubGroupCode: subCode, FormulaCode: fd, IsDefault: true,
				})
			}
			if fc := cellAt(row, cFConvert); fc != "" {
				req.SubGroupFormulas = append(req.SubGroupFormulas, PriceListSubGroupFormulasCreateDTO{
					SubGroupCode: subCode, FormulaCode: fc, IsDefault: false,
				})
			}
		}
	}

	if len(req.SubGroups) == 0 {
		return nil, fmt.Errorf("no valid rows found in sheet %q", sheet)
	}
	return req, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func parseFloatLoose(s string) float64 {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func parseBoolLoose(s string) bool {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "true", "1", "yes", "y":
		return true
	}
	return false
}
