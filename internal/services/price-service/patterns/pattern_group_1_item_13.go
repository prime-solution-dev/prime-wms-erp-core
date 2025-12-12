package patterns

import (
	"fmt"
	"sort"

	"prime-erp-core/internal/models"
	"prime-erp-core/internal/utils"

	"github.com/google/uuid"
)

func BuildGroup1Item13Response(priceListData []models.GetPriceListResponse, groupCode string) (PriceListDetailApiResponse, error) {
	config, err := loadConfiguration(groupCode)
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
	utils.PrintJSON(rowData)
	mergedRows := mergeGroup1Item9Rows(rowData)

	sort.SliceStable(mergedRows, func(i, j int) bool {
		pg2I := fmt.Sprintf("%v", mergedRows[i]["product_group_2"])
		pg2J := fmt.Sprintf("%v", mergedRows[j]["product_group_2"])
		if pg2I == pg2J {
			pg1I := fmt.Sprintf("%v", mergedRows[i]["product_group_1"])
			pg1J := fmt.Sprintf("%v", mergedRows[j]["product_group_1"])
			if pg1I == pg1J {
				size4I := fmt.Sprintf("%v", mergedRows[i]["product_group_4"])
				size4J := fmt.Sprintf("%v", mergedRows[j]["product_group_4"])
				if size4I == size4J {
					size3I := fmt.Sprintf("%v", mergedRows[i]["product_group_3"])
					size3J := fmt.Sprintf("%v", mergedRows[j]["product_group_3"])
					return size3I < size3J
				}
				return size4I < size4J
			}
			return pg1I < pg1J
		}
		return pg2I < pg2J
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
