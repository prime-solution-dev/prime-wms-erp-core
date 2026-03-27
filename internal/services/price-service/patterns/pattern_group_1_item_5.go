package patterns

import (
	"fmt"
	"sort"
	"strings"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func BuildGroup1Item5Response(priceListData []models.GetPriceListResponse, groupCode string) (PriceListDetailApiResponse, error) {
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

	tabOrder := buildTabOrder(pattern.ApplicableCategories, groupedByProductGroup2)
	tabs := make([]PriceListDetailTabConfig, 0, len(tabOrder))

	for _, tabLabel := range tabOrder {
		subGroups := groupedByProductGroup2[tabLabel]
		columns := buildGroup1Item5Columns(pattern, subGroups)
		rowData := buildDynamicRows(config, pattern, subGroups)

		tableData := make([]map[string]interface{}, len(rowData))
		for i, row := range rowData {
			tableData[i] = map[string]interface{}(row)
		}

		tab := PriceListDetailTabConfig{
			ID:    uuid.NewSHA1(uuid.NameSpaceDNS, []byte(tabLabel)),
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

func buildGroup1Item5Columns(pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse) []ColumnDef {
	columnGroupFields := strings.Split(pattern.Grouping.ColumnGroups, "|")
	type columnGroupValue struct {
		Label string
		Code  string
	}
	uniqueValues := make(map[string]columnGroupValue)
	for _, sg := range subGroups {
		label := buildCompositeKey(sg.SubGroupKeys, columnGroupFields)
		if label == "" {
			continue
		}
		code := buildCompositeCodeKey(sg.SubGroupKeys, columnGroupFields)
		mapKey := fmt.Sprintf("%s|%s", label, code)
		uniqueValues[mapKey] = columnGroupValue{
			Label: label,
			Code:  code,
		}
	}

	// Sort keys to ensure consistent column order by label
	sortedKeys := make([]string, 0, len(uniqueValues))
	for key := range uniqueValues {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return uniqueValues[sortedKeys[i]].Label < uniqueValues[sortedKeys[j]].Label
	})

	columns := make([]ColumnDef, 0, len(sortedKeys))
	for _, key := range sortedKeys {
		value := uniqueValues[key]
		groupIdentifier := sanitizeIdentifier(value.Code, value.Label)

		colGroup := ColumnDef{
			HeaderName:    value.Label,
			GroupID:       groupIdentifier,
			OpenByDefault: boolPtr(true),
			Children:      []ColumnDef{},
		}

		for _, colConfig := range pattern.Columns {
			colGroup.Children = append(colGroup.Children, buildItem5ColumnDef(colConfig, groupIdentifier))
		}

		for _, nestedGroup := range pattern.ColumnGroups {
			colGroup.Children = append(colGroup.Children, buildItem5NestedGroup(nestedGroup, groupIdentifier))
		}

		columns = append(columns, colGroup)
	}

	return columns
}

func buildItem5ColumnDef(config ColumnConfigItem, prefix string) ColumnDef {
	field := fmt.Sprintf("%s_%s", prefix, config.Field)
	col := ColumnDef{
		Field:        field,
		HeaderName:   config.HeaderName,
		Width:        intPtr(config.Width),
		CellRenderer: config.CellRenderer,
	}

	if config.CellStyle != nil {
		col.CellStyle = convertCellStyle(config.CellStyle)
	}
	if config.EnableTooltip {
		col.EnableTooltip = boolPtr(true)
	}

	return col
}

func buildItem5NestedGroup(config ColumnGroupConfig, prefix string) ColumnDef {
	groupID := config.GroupID
	if groupID == "" {
		groupID = config.HeaderName
	}

	group := ColumnDef{
		HeaderName:    config.HeaderName,
		GroupID:       fmt.Sprintf("%s_%s", prefix, sanitizeFieldName(groupID)),
		OpenByDefault: boolPtr(config.OpenByDefault),
		Children:      []ColumnDef{},
	}

	for _, child := range config.Children {
		group.Children = append(group.Children, buildItem5ColumnDef(child, prefix))
	}

	return group
}

func buildTabOrder(preferred []string, groupedData map[string][]models.PriceListSubGroupResponse) []string {
	tabOrder := make([]string, 0, len(groupedData))
	seen := make(map[string]bool)

	for _, label := range preferred {
		if _, ok := groupedData[label]; ok {
			tabOrder = append(tabOrder, label)
			seen[label] = true
		}
	}

	remaining := make([]string, 0)
	for label := range groupedData {
		if !seen[label] {
			remaining = append(remaining, label)
		}
	}
	sort.Strings(remaining)

	return append(tabOrder, remaining...)
}
