package patterns

import (
	"fmt"
	"sort"
	"strings"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func BuildGroup1Item1Response(priceListData []models.GetPriceListResponse) (PriceListDetailApiResponse, error) {
	groupedData := groupDataByGroupKeyAndProductGroup2(priceListData)
	tabs := make([]PriceListDetailTabConfig, 0)
	var loadErr error

	// Create a map to store tabs with their pattern order for sorting
	type tabWithOrder struct {
		tab           PriceListDetailTabConfig
		patternID     string
		patternIdx    int
		productGroup2 string
	}
	tabsWithOrder := []tabWithOrder{}

	for groupKey, productGroup2Map := range groupedData {
		config, err := loadConfiguration(groupKey)
		if err != nil {
			loadErr = fmt.Errorf("load configuration for %s: %w", groupKey, err)
			continue
		}

		// Sort productGroup2 keys to ensure consistent iteration order
		productGroup2Keys := make([]string, 0, len(productGroup2Map))
		for pg2 := range productGroup2Map {
			productGroup2Keys = append(productGroup2Keys, pg2)
		}
		sort.Strings(productGroup2Keys)

		for _, productGroup2 := range productGroup2Keys {
			subGroups := productGroup2Map[productGroup2]
			pattern := selectPatternForCategory(config, productGroup2)
			if pattern == nil {
				continue
			}

			columns := buildDynamicColumns(pattern, subGroups)
			rowData := buildDynamicRows(pattern, subGroups)

			// Merge rows with the same row_group_value into a single row
			// This groups all column group columns into one row per row key
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
					// Copy row grouping fields (these are the same for all rows being merged)
					for _, field := range strings.Split(pattern.Grouping.Rows, "|") {
						fieldName := convertGroupCodeToFieldName(field)
						if val, ok := row[fieldName]; ok {
							mergedRow[fieldName] = val
						}
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
					if key == "id" || key == "row_group_value" || key == "is_trading" || key == "subgroup_id" {
						continue
					}
					// Skip row grouping fields
					skipField := false
					for _, field := range strings.Split(pattern.Grouping.Rows, "|") {
						fieldName := convertGroupCodeToFieldName(field)
						if key == fieldName {
							skipField = true
							break
						}
					}
					if skipField {
						continue
					}
					// For column_group_key and column_group_value, keep the last value (they differ per column group)
					// This maintains some metadata while avoiding true duplicates
					if key == "column_group_key" || key == "column_group_value" {
						mergedRow[key] = value
						continue
					}
					// Copy all other fields (column group specific fields)
					mergedRow[key] = value
				}

				mergedRowMap[rowGroupValue] = mergedRow
			}

			// Convert merged rows to slice and sort
			mergedRows := make([]AGGridRowData, 0, len(mergedRowMap))
			for _, row := range mergedRowMap {
				mergedRows = append(mergedRows, row)
			}

			// Sort merged rows by row_group_value
			sort.SliceStable(mergedRows, func(i, j int) bool {
				rowGroupI := fmt.Sprintf("%v", mergedRows[i]["row_group_value"])
				rowGroupJ := fmt.Sprintf("%v", mergedRows[j]["row_group_value"])
				return rowGroupI < rowGroupJ
			})

			tableData := make([]map[string]interface{}, len(mergedRows))
			for i, row := range mergedRows {
				tableData[i] = map[string]interface{}(row)
			}

			// Find pattern index in config for sorting
			patternIdx := -1
			for i, p := range config.Patterns {
				if p.ID == pattern.ID {
					patternIdx = i
					break
				}
			}

			tabsWithOrder = append(tabsWithOrder, tabWithOrder{
				tab: PriceListDetailTabConfig{
					ID:    uuid.New(),
					Label: productGroup2,
					TableConfig: TableConfig{
						Title:             productGroup2,
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
					TableData:        tableData,
					EditableSuffixes: pattern.EditableSuffixes,
				},
				patternID:     pattern.ID,
				patternIdx:    patternIdx,
				productGroup2: productGroup2,
			})
		}
	}

	// Sort tabs by pattern order (patternIdx), then by productGroup2 name for same pattern
	sort.Slice(tabsWithOrder, func(i, j int) bool {
		if tabsWithOrder[i].patternIdx != tabsWithOrder[j].patternIdx {
			return tabsWithOrder[i].patternIdx < tabsWithOrder[j].patternIdx
		}
		return tabsWithOrder[i].productGroup2 < tabsWithOrder[j].productGroup2
	})

	// Extract sorted tabs
	for _, tw := range tabsWithOrder {
		tabs = append(tabs, tw.tab)
	}

	response := PriceListDetailApiResponse{
		Id:   uuid.MustParse(priceListData[0].ID),
		Name: "Price List Detail",
		Tabs: tabs,
	}

	if loadErr != nil && len(tabs) == 0 {
		return response, loadErr
	}

	return response, nil
}
