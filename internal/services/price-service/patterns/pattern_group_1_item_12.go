package patterns

import (
	"fmt"
	"sort"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func BuildGroup1Item12Response(priceListData []models.GetPriceListResponse, groupCode string) (PriceListDetailApiResponse, error) {
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
	fmt.Println("rowData", len(rowData))
	mergedRows := mergeGroup1Item9Rows(rowData)
	fmt.Println("mergedRows", len(mergedRows))

	sort.SliceStable(mergedRows, func(i, j int) bool {
		rowGroupI := fmt.Sprintf("%v", mergedRows[i]["row_group_value"])
		rowGroupJ := fmt.Sprintf("%v", mergedRows[j]["row_group_value"])
		return rowGroupI < rowGroupJ
	})

	tableData := make([]map[string]interface{}, len(mergedRows))
	for i, row := range mergedRows {
		tableData[i] = map[string]interface{}(row)
	}

	tabLabel := "Price List Detail"
	if len(mergedRows) > 0 {
		if pg2 := fmt.Sprintf("%v", mergedRows[0]["product_group_2"]); pg2 != "" {
			tabLabel = pg2
		}
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
