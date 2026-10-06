package patterns

import (
	"fmt"
	"sort"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func BuildGroup1Item4Response(priceListData []models.GetPriceListResponse, groupCode string) (PriceListDetailApiResponse, error) {
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

	productGroup2Code := getGroupCodeFromConfig(config, pattern, "productGroup2", "PRODUCT_GROUP2")
	groupedByProductGroup2 := make(map[string][]models.PriceListSubGroupResponse)
	for _, sg := range allSubGroups {
		productGroup2 := getValueNameByGroupCode(sg.SubGroupKeys, productGroup2Code)
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
	allSubGroupsForTabs := make([]models.PriceListSubGroupResponse, 0)
	for key, sgs := range groupedByProductGroup2 {
		allSubGroupsForTabs = append(allSubGroupsForTabs, sgs...)
		if !seen[key] {
			remaining = append(remaining, key)
		}
	}
	// sort.Strings ก่อนเพื่อให้ลำดับตั้งต้นนิ่ง แล้วจึงเรียงด้วย group_item.value
	sort.Strings(remaining)
	sortLabelsByValue(remaining, allSubGroupsForTabs,
		getGroupCodeFromConfig(config, pattern, "productGroup2", "PRODUCT_GROUP2"))
	tabOrder = append(tabOrder, remaining...)

	columns := buildFixedColumns(pattern)

	tabs := make([]PriceListDetailTabConfig, 0, len(tabOrder))
	for _, tabLabel := range tabOrder {
		subGroups := groupedByProductGroup2[tabLabel]

		productGroup4Code := getGroupCodeFromConfig(config, pattern, "productGroup4", "PRODUCT_GROUP4")
		productGroup6Code := getGroupCodeFromConfig(config, pattern, "productGroup6", "PRODUCT_GROUP6")
		// เรียงด้วยค่าตัวเลขจาก group_item.value — เทียบ ValueName แบบ string
		// ให้ 1.2, 1.4, 1.9, 10, 100, 12 ซึ่งผิด
		SortSubGroupsByValue(subGroups, productGroup4Code, productGroup6Code)

		rowData := buildDirectRows(config, pattern, subGroups)
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
