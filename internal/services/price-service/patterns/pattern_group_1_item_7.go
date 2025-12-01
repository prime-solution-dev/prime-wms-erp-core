package patterns

import (
	"fmt"
	"sort"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func BuildGroup1Item7Response(priceListData []models.GetPriceListResponse, groupCode string) (PriceListDetailApiResponse, error) {
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

		// Merge rows with the same row_group_value into a single row
		// This groups all PRODUCT_GROUP5 columns into one row per PRODUCT_GROUP6
		mergedRowMap := make(map[string]AGGridRowData)
		for _, row := range rowData {
			rowGroupValue := fmt.Sprintf("%v", row["row_group_value"])
			if rowGroupValue == "" {
				continue
			}

			mergedRow, exists := mergedRowMap[rowGroupValue]
			if !exists {
				// Create new merged row with common fields
				mergedRow = make(AGGridRowData)
				// Copy common fields (only once)
				if val, ok := row["id"]; ok {
					mergedRow["id"] = val
				}
				if val, ok := row["product_group_6"]; ok {
					mergedRow["product_group_6"] = val
				}
				if val, ok := row["row_group_value"]; ok {
					mergedRow["row_group_value"] = val
				}
				if val, ok := row["is_trading"]; ok {
					mergedRow["is_trading"] = val
				}
				if val, ok := row["subgroup_id"]; ok {
					mergedRow["subgroup_id"] = val
				}
			}

			// Merge all column-specific fields from this row
			for key, value := range row {
				// Skip common fields that we already set (these have the same value across all rows being merged)
				if key == "id" || key == "product_group_6" || key == "row_group_value" || key == "is_trading" || key == "subgroup_id" {
					continue
				}
				// For column_group_key and column_group_value, keep the last value (they differ per column group)
				// This maintains some metadata while avoiding true duplicates
				if key == "column_group_key" || key == "column_group_value" {
					mergedRow[key] = value
					continue
				}
				// Copy all other fields (column group specific fields like 4_x8_*, 5_x20_*)
				mergedRow[key] = value
			}

			mergedRowMap[rowGroupValue] = mergedRow
		}

		// Convert merged rows to slice and sort
		mergedRows := make([]AGGridRowData, 0, len(mergedRowMap))
		for _, row := range mergedRowMap {
			mergedRows = append(mergedRows, row)
		}

		sort.SliceStable(mergedRows, func(i, j int) bool {
			thicknessI := fmt.Sprintf("%v", mergedRows[i]["product_group_6"])
			thicknessJ := fmt.Sprintf("%v", mergedRows[j]["product_group_6"])
			return thicknessI < thicknessJ
		})

		tableData := make([]map[string]interface{}, len(mergedRows))
		for i, row := range mergedRows {
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
