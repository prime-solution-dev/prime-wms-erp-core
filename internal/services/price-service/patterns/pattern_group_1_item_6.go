package patterns

import (
	"fmt"
	"sort"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func BuildGroup1Item6Response(priceListData []models.GetPriceListResponse, groupCode string) (PriceListDetailApiResponse, error) {
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
			Name: "หมวดเหล็กฉาก",
			Tabs: []PriceListDetailTabConfig{},
		}, nil
	}

	groupedByProductGroup2 := make(map[string][]models.PriceListSubGroupResponse)
	for _, sg := range allSubGroups {
		productGroup2 := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP2")
		if productGroup2 == "" {
			productGroup2 = "อื่นๆ"
		}
		groupedByProductGroup2[productGroup2] = append(groupedByProductGroup2[productGroup2], sg)
	}

	tabOrder := buildTabOrder(pattern.ApplicableCategories, groupedByProductGroup2)
	columns := buildFixedColumns(pattern)

	tabs := make([]PriceListDetailTabConfig, 0, len(tabOrder))
	for _, tabLabel := range tabOrder {
		subGroups := groupedByProductGroup2[tabLabel]

		sort.SliceStable(subGroups, func(i, j int) bool {
			group4I := getValueNameByGroupCode(subGroups[i].SubGroupKeys, "PRODUCT_GROUP4")
			group4J := getValueNameByGroupCode(subGroups[j].SubGroupKeys, "PRODUCT_GROUP4")
			if group4I == group4J {
				group7I := getValueNameByGroupCode(subGroups[i].SubGroupKeys, "PRODUCT_GROUP7")
				group7J := getValueNameByGroupCode(subGroups[j].SubGroupKeys, "PRODUCT_GROUP7")
				return group7I < group7J
			}
			return group4I < group4J
		})

		rowData := buildDirectRows(pattern, subGroups)
		tableData := make([]map[string]interface{}, len(rowData))
		for i, row := range rowData {
			tableData[i] = map[string]interface{}(row)
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

		tabs = append(tabs, tab)
	}

	return PriceListDetailApiResponse{
		Id:   uuid.MustParse(priceListData[0].ID),
		Name: "Price List Detail",
		Tabs: tabs,
	}, nil
}
