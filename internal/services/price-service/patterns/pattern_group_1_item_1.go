package patterns

import (
	"fmt"
	"sort"
	"strings"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func BuildGroup1Item1Response(priceListData []models.GetPriceListResponse) (PriceListDetailApiResponse, error) {
	// utils.PrintJson(priceListData)
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
		config, err := LoadConfiguration(groupKey)
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
			rowData := buildDynamicRows(config, pattern, subGroups)

			// Regroup rows to prevent data loss when the same column_group_key appears
			// multiple times for a given row_group_value (e.g., multiple subgroup_ids
			// under the same PRODUCT_GROUP5 and thickness).
			//
			// Strategy:
			//   1. Group raw rows by row_group_value.
			//   2. Inside each row group, group again by column_group_key.
			//   3. For each row group, compute the maximum number of entries across all
			//      column groups (maxCount). Then build maxCount logical rows by taking
			//      the i-th entry from each column group (if present) and merging their
			//      column-specific fields into a single AGGrid row.
			rowFields := strings.Split(pattern.Grouping.Rows, "|")

			// 1) Group by row_group_value using the same composite logic as before
			rowsByRowGroup := make(map[string][]AGGridRowData)
			skippedRowsEmptyGroup := 0
			for _, row := range rowData {
				mergeKeyParts := make([]string, 0, len(rowFields))
				for _, field := range rowFields {
					fieldName := convertGroupCodeToFieldName(field)
					if val, ok := row[fieldName]; ok {
						valStr := strings.TrimSpace(fmt.Sprintf("%v", val))
						if valStr != "" {
							mergeKeyParts = append(mergeKeyParts, valStr)
						}
					}
				}

				rowGroupValue := strings.Join(mergeKeyParts, "|")
				if rowGroupValue == "" {
					if v, ok := row["row_group_value"]; ok {
						rowGroupValue = strings.TrimSpace(fmt.Sprintf("%v", v))
					}
				}
				if rowGroupValue == "" {
					skippedRowsEmptyGroup++
					continue
				}

				rowsByRowGroup[rowGroupValue] = append(rowsByRowGroup[rowGroupValue], row)
			}

			// Sort row_group_value keys for deterministic row order
			rowGroupKeys := make([]string, 0, len(rowsByRowGroup))
			for k := range rowsByRowGroup {
				rowGroupKeys = append(rowGroupKeys, k)
			}
			sort.Strings(rowGroupKeys)

			mergedRows := make([]AGGridRowData, 0, len(rowData))

			for _, rowGroupValue := range rowGroupKeys {
				groupRows := rowsByRowGroup[rowGroupValue]
				if len(groupRows) == 0 {
					continue
				}

				// 2) Group rows in this thickness group by column_group_key
				columnsByKey := make(map[string][]AGGridRowData)
				skippedRowsEmptyColKey := 0
				for _, row := range groupRows {
					colKey := strings.TrimSpace(fmt.Sprintf("%v", row["column_group_key"]))
					if colKey == "" {
						skippedRowsEmptyColKey++
						continue
					}
					columnsByKey[colKey] = append(columnsByKey[colKey], row)
				}

				if len(columnsByKey) == 0 {
					continue
				}

				// Sort column keys for deterministic merge order
				columnKeys := make([]string, 0, len(columnsByKey))
				for k := range columnsByKey {
					columnKeys = append(columnKeys, k)
				}
				sort.Strings(columnKeys)

				// Optional: sort each column's rows by its own row_number to keep visual order stable
				for _, colKey := range columnKeys {
					rowsForCol := columnsByKey[colKey]
					rowNumberField := fmt.Sprintf("%s_row_number", colKey)
					sort.SliceStable(rowsForCol, func(i, j int) bool {
						vi, iOK := rowsForCol[i][rowNumberField]
						vj, jOK := rowsForCol[j][rowNumberField]
						if !iOK || !jOK {
							return false
						}
						return fmt.Sprintf("%v", vi) < fmt.Sprintf("%v", vj)
					})
					columnsByKey[colKey] = rowsForCol
				}

				// 3) Determine how many logical rows we need for this row_group_value
				maxCount := 0
				for _, rowsForCol := range columnsByKey {
					if len(rowsForCol) > maxCount {
						maxCount = len(rowsForCol)
					}
				}
				if maxCount == 0 {
					continue
				}

				// Helper: get representative values for row grouping fields from the first row
				baseRow := groupRows[0]

				for i := 0; i < maxCount; i++ {
					mergedRow := make(AGGridRowData)

					// New id for each logical row
					mergedRow["id"] = uuid.New().String()
					mergedRow["row_group_value"] = rowGroupValue

					// Copy row grouping fields (same for the whole thickness group)
					for _, field := range rowFields {
						fieldName := convertGroupCodeToFieldName(field)
						if val, ok := baseRow[fieldName]; ok {
							mergedRow[fieldName] = val
						}
					}

					// Track whether we've set top-level subgroup_id / is_trading
					mainSubgroupSet := false

					// Merge column-specific data for each PRODUCT_GROUP5 column
					for _, colKey := range columnKeys {
						rowsForCol := columnsByKey[colKey]
						if len(rowsForCol) <= i {
							continue
						}
						src := rowsForCol[i]

						// Preserve a representative column_group_key/value (first non-empty wins)
						if _, ok := mergedRow["column_group_key"]; !ok {
							if v, ok := src["column_group_key"]; ok {
								mergedRow["column_group_key"] = v
							}
						}
						if _, ok := mergedRow["column_group_value"]; !ok {
							if v, ok := src["column_group_value"]; ok {
								mergedRow["column_group_value"] = v
							}
						}

						for key, value := range src {
							// Skip fields that are common or handled separately
							if key == "id" || key == "row_group_value" ||
								key == "column_group_key" || key == "column_group_value" {
								continue
							}

							// Skip row grouping fields
							skipField := false
							for _, field := range rowFields {
								fieldName := convertGroupCodeToFieldName(field)
								if key == fieldName {
									skipField = true
									break
								}
							}
							if skipField {
								continue
							}

							// Set top-level subgroup_id / is_trading from the first column that has them
							if !mainSubgroupSet && (key == "subgroup_id" || key == "is_trading") {
								mergedRow[key] = value
								if key == "subgroup_id" {
									mainSubgroupSet = true
								}
								continue
							}
							if key == "subgroup_id" || key == "is_trading" {
								// Other occurrences of these keys are column-specific; keep them as-is
								mergedRow[key] = value
								continue
							}

							// Copy all remaining fields (column group–specific fields, tooltips, UDFs, etc.)
							mergedRow[key] = value
						}
					}

					mergedRows = append(mergedRows, mergedRow)
				}
			}

			// Sort final merged rows by row_group_value to keep the previous behavior
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
					TableData:         tableData,
					EditableSuffixes:  pattern.EditableSuffixes,
					FetchableSuffixes: pattern.FetchableSuffixes,
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
