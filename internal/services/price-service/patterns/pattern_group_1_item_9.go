package patterns

import (
	"fmt"
	"sort"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func BuildGroup1Item9Response(priceListData []models.GetPriceListResponse, groupCode string) (PriceListDetailApiResponse, error) {
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

	columns := buildDynamicColumns(pattern, allSubGroups)
	rowData := buildDynamicRows(pattern, allSubGroups)
	mergedRows := mergeGroup1Item9Rows(rowData)

	sort.SliceStable(mergedRows, func(i, j int) bool {
		productGroupI := fmt.Sprintf("%v", mergedRows[i]["product_group_2"])
		productGroupJ := fmt.Sprintf("%v", mergedRows[j]["product_group_2"])
		if productGroupI == productGroupJ {
			lengthI := fmt.Sprintf("%v", mergedRows[i]["product_group_7"])
			lengthJ := fmt.Sprintf("%v", mergedRows[j]["product_group_7"])
			if lengthI == lengthJ {
				sizeI := fmt.Sprintf("%v", mergedRows[i]["product_group_6"])
				sizeJ := fmt.Sprintf("%v", mergedRows[j]["product_group_6"])
				return sizeI < sizeJ
			}
			return lengthI < lengthJ
		}
		return productGroupI < productGroupJ
	})

	tableData := make([]map[string]interface{}, len(mergedRows))
	for i, row := range mergedRows {
		tableData[i] = map[string]interface{}(row)
	}

	tabLabel := pattern.Name
	if tabLabel == "" {
		tabLabel = groupCode
	}

	tab := PriceListDetailTabConfig{
		ID:    uuid.New(),
		Label: tabLabel,
		TableConfig: TableConfig{
			Title:             tabLabel,
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
			Columns: columns,
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

func mergeGroup1Item9Rows(rows []AGGridRowData) []AGGridRowData {
	if len(rows) == 0 {
		return rows
	}

	mergedMap := make(map[string]AGGridRowData)
	order := make([]string, 0)

	for _, row := range rows {
		rowGroup := fmt.Sprintf("%v", row["row_group_value"])
		if rowGroup == "" {
			continue
		}

		if existing, ok := mergedMap[rowGroup]; ok {
			for key, value := range row {
				if key == "id" {
					continue
				}
				existing[key] = value
			}
			continue
		}

		cloned := make(AGGridRowData, len(row))
		for key, value := range row {
			cloned[key] = value
		}
		mergedMap[rowGroup] = cloned
		order = append(order, rowGroup)
	}

	mergedRows := make([]AGGridRowData, 0, len(order))
	for _, key := range order {
		if row, ok := mergedMap[key]; ok {
			mergedRows = append(mergedRows, row)
		}
	}

	return mergedRows
}
