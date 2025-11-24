package patterns

import (
	"fmt"
	"sort"

	"prime-erp-core/internal/models"
	"prime-erp-core/internal/utils"

	"github.com/google/uuid"
)

func BuildGroup1Item7Response(priceListData []models.GetPriceListResponse, groupCode string) (PriceListDetailApiResponse, error) {
	utils.PrintJSON(priceListData)
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

	groupedByProductGroup2 := make(map[string][]models.PriceListSubGroupResponse)
	for _, sg := range allSubGroups {
		productGroup2 := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP2")
		if productGroup2 == "" {
			productGroup2 = "อื่นๆ"
		}
		groupedByProductGroup2[productGroup2] = append(groupedByProductGroup2[productGroup2], sg)
	}

	tabOrder := make([]string, 0)
	seen := make(map[string]bool)
	for _, cat := range pattern.ApplicableCategories {
		if _, ok := groupedByProductGroup2[cat]; ok {
			tabOrder = append(tabOrder, cat)
			seen[cat] = true
		}
	}
	remaining := make([]string, 0)
	for key := range groupedByProductGroup2 {
		if !seen[key] {
			remaining = append(remaining, key)
		}
	}
	sort.Strings(remaining)
	tabOrder = append(tabOrder, remaining...)

	tabs := make([]PriceListDetailTabConfig, 0, len(tabOrder))
	for _, tabLabel := range tabOrder {
		subGroups := groupedByProductGroup2[tabLabel]
		if len(subGroups) == 0 {
			continue
		}

		columns := buildDynamicColumns(pattern, subGroups)
		rowData := buildDynamicRows(pattern, subGroups)

		sort.SliceStable(rowData, func(i, j int) bool {
			thicknessI := fmt.Sprintf("%v", rowData[i]["product_group_6"])
			thicknessJ := fmt.Sprintf("%v", rowData[j]["product_group_6"])
			if thicknessI == thicknessJ {
				return fmt.Sprintf("%v", rowData[i]["row_group_value"]) < fmt.Sprintf("%v", rowData[j]["row_group_value"])
			}
			return thicknessI < thicknessJ
		})

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
		Name: "ม้วน",
		Tabs: tabs,
	}, nil
}
