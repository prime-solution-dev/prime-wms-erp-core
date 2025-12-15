package patterns

import (
	"fmt"
	"sort"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func BuildGroup1Item11Response(priceListData []models.GetPriceListResponse, groupCode string) (PriceListDetailApiResponse, error) {
	config, err := LoadConfiguration(groupCode)
	if err != nil {
		return PriceListDetailApiResponse{}, fmt.Errorf("load configuration for %s: %w", groupCode, err)
	}
	var pattern *PatternConfig
	for i := range config.Patterns {
		if config.Patterns[i].ID == config.DefaultPattern && config.Patterns[i].Enabled {
			pattern = &config.Patterns[i]
			break
		}
	}
	if pattern == nil {
		return PriceListDetailApiResponse{}, fmt.Errorf("no enabled pattern found for %s", groupCode)
	}

	allSubGroups := make([]models.PriceListSubGroupResponse, 0)
	for _, priceList := range priceListData {
		allSubGroups = append(allSubGroups, priceList.SubGroups...)
	}
	if len(allSubGroups) == 0 {
		return PriceListDetailApiResponse{
			Id:   uuid.MustParse(priceListData[0].ID),
			Name: "Price List Detail",
			Tabs: []PriceListDetailTabConfig{},
		}, nil
	}

	// Build fixed columns
	fixedColumns := buildFixedColumns(pattern)

	// Build dynamic column groups from PRODUCT_GROUP2 values
	dynamicColumnGroups := buildProductGroup2ColumnGroups(pattern, allSubGroups)

	// Combine fixed columns with dynamic column groups, but drop standalone pattern columns
	allColumns := append(fixedColumns, dynamicColumnGroups...)
	allColumns = removeStandalonePatternColumns(allColumns, pattern.Columns)

	// Sort subgroups by "หนา x ยาว" (PRODUCT_GROUP6 x PRODUCT_GROUP7) for row spanning
	sort.SliceStable(allSubGroups, func(i, j int) bool {
		pg6I := getValueNameByGroupCode(allSubGroups[i].SubGroupKeys, "PRODUCT_GROUP6")
		pg7I := getValueNameByGroupCode(allSubGroups[i].SubGroupKeys, "PRODUCT_GROUP7")
		pg6J := getValueNameByGroupCode(allSubGroups[j].SubGroupKeys, "PRODUCT_GROUP6")
		pg7J := getValueNameByGroupCode(allSubGroups[j].SubGroupKeys, "PRODUCT_GROUP7")

		compositeI := fmt.Sprintf("%s x %s", pg6I, pg7I)
		compositeJ := fmt.Sprintf("%s x %s", pg6J, pg7J)

		if compositeI == compositeJ {
			// If same "หนา x ยาว", sort by "ขนาด" (PRODUCT_GROUP5 + PRODUCT_GROUP3)
			pg5I := getValueNameByGroupCode(allSubGroups[i].SubGroupKeys, "PRODUCT_GROUP5")
			pg3I := getValueNameByGroupCode(allSubGroups[i].SubGroupKeys, "PRODUCT_GROUP3")
			pg5J := getValueNameByGroupCode(allSubGroups[j].SubGroupKeys, "PRODUCT_GROUP5")
			pg3J := getValueNameByGroupCode(allSubGroups[j].SubGroupKeys, "PRODUCT_GROUP3")

			sizeI := pg5I + pg3I
			sizeJ := pg5J + pg3J
			return sizeI < sizeJ
		}
		return compositeI < compositeJ
	})

	// Build rows with fixed columns and dynamic column group data
	rowData := buildDirectRowsWithProductGroup2(pattern, allSubGroups)
	tableData := make([]map[string]interface{}, len(rowData))
	for i, row := range rowData {
		tableData[i] = map[string]interface{}(row)
	}

	tab := PriceListDetailTabConfig{
		ID:    uuid.New(),
		Label: "หมวดเหล็กแบน",
		TableConfig: TableConfig{
			Title:             "หมวดเหล็กแบน",
			GroupHeaderHeight: intPtr(config.TableConfig.GroupHeaderHeight),
			HeaderHeight:      intPtr(config.TableConfig.HeaderHeight),
			Pagination:        boolPtr(config.TableConfig.Pagination),
			Toolbar: &Toolbar{
				Show:             boolPtr(config.TableConfig.Toolbar.Show),
				ShowSearch:       boolPtr(config.TableConfig.Toolbar.ShowSearch),
				ShowRefresh:      boolPtr(config.TableConfig.Toolbar.ShowRefresh),
				ShowColumnToggle: boolPtr(config.TableConfig.Toolbar.ShowColumnToggle),
			},
			GridOptions: &GridOptions{
				SuppressMovableColumns: boolPtr(config.TableConfig.GridOptions.SuppressMovableColumns),
				SuppressMenuHide:       boolPtr(config.TableConfig.GridOptions.SuppressMenuHide),
				EnableCellSpan:         boolPtr(config.TableConfig.GridOptions.EnableCellSpan),
			},
			Columns: allColumns,
		},
		TableData:         tableData,
		EditableSuffixes:  pattern.EditableSuffixes,
		FetchableSuffixes: pattern.FetchableSuffixes,
	}

	return PriceListDetailApiResponse{
		Id:   uuid.MustParse(priceListData[0].ID),
		Name: "Price List Detail",
		Tabs: []PriceListDetailTabConfig{tab},
	}, nil
}

func removeStandalonePatternColumns(columns []ColumnDef, standaloneConfigs []ColumnConfigItem) []ColumnDef {
	if len(columns) == 0 {
		return columns
	}

	// Build a lookup for fields defined in pattern.Columns (they should only appear inside column groups)
	standaloneFields := make(map[string]struct{}, len(standaloneConfigs))
	for _, col := range standaloneConfigs {
		if col.Field != "" {
			standaloneFields[col.Field] = struct{}{}
		}
	}

	if len(standaloneFields) == 0 {
		return columns
	}

	filtered := make([]ColumnDef, 0, len(columns))
	for _, col := range columns {
		if col.Field != "" {
			if _, shouldRemove := standaloneFields[col.Field]; shouldRemove {
				continue
			}
		}
		filtered = append(filtered, col)
	}

	return filtered
}
