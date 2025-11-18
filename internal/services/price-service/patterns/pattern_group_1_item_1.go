package patterns

import (
	"fmt"
	"sort"

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

			tableData := make([]map[string]interface{}, len(rowData))
			for i, row := range rowData {
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
