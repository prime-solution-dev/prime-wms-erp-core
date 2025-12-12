package patterns

import (
	"fmt"
	"sort"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func BuildGroup1Item8Response(priceListData []models.GetPriceListResponse, groupCode string) (PriceListDetailApiResponse, error) {
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
	for _, pl := range priceListData {
		allSubGroups = append(allSubGroups, pl.SubGroups...)
	}
	if len(allSubGroups) == 0 {
		return PriceListDetailApiResponse{
			Id:   uuid.MustParse(priceListData[0].ID),
			Name: "Price List Detail",
			Tabs: []PriceListDetailTabConfig{},
		}, nil
	}

	groupedByProductGroup1 := make(map[string][]models.PriceListSubGroupResponse)
	for _, sg := range allSubGroups {
		tabKey := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP1")
		if tabKey == "" {
			tabKey = "อื่นๆ"
		}
		groupedByProductGroup1[tabKey] = append(groupedByProductGroup1[tabKey], sg)
	}

	tabOrder := make([]string, 0)
	seen := make(map[string]bool)
	for _, cat := range pattern.ApplicableCategories {
		if _, ok := groupedByProductGroup1[cat]; ok {
			tabOrder = append(tabOrder, cat)
			seen[cat] = true
		}
	}
	remaining := make([]string, 0)
	for key := range groupedByProductGroup1 {
		if !seen[key] {
			remaining = append(remaining, key)
		}
	}
	sort.Strings(remaining)
	tabOrder = append(tabOrder, remaining...)

	columns := buildFixedColumns(pattern)
	tabs := make([]PriceListDetailTabConfig, 0, len(tabOrder))

	for _, tabLabel := range tabOrder {
		subGroups := groupedByProductGroup1[tabLabel]
		if len(subGroups) == 0 {
			continue
		}

		rows := buildDirectRows(pattern, subGroups)

		sort.SliceStable(rows, func(i, j int) bool {
			thicknessI := fmt.Sprintf("%v", rows[i]["product_group_6"])
			thicknessJ := fmt.Sprintf("%v", rows[j]["product_group_6"])
			if thicknessI == thicknessJ {
				shipI := fmt.Sprintf("%v", rows[i]["ship_no"])
				shipJ := fmt.Sprintf("%v", rows[j]["ship_no"])
				return shipI < shipJ
			}
			return thicknessI < thicknessJ
		})

		tableData := make([]map[string]interface{}, len(rows))
		for i, row := range rows {
			tableData[i] = map[string]interface{}(row)
		}

		summaryRows := buildSummaryRows(pattern, rows)
		summaryField := buildSummaryField(summaryRows)

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
			SummaryRows:       summaryRows,
			SummaryField:      summaryField,
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
