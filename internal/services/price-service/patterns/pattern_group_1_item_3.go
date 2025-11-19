package patterns

import (
	"fmt"
	"sort"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func BuildGroup1Item3Response(priceListData []models.GetPriceListResponse, groupKey string) (PriceListDetailApiResponse, error) {
	config, err := loadConfiguration(groupKey)
	if err != nil {
		return PriceListDetailApiResponse{}, fmt.Errorf("load configuration for %s: %w", groupKey, err)
	}

	var pattern *PatternConfig
	for i := range config.Patterns {
		if config.Patterns[i].ID == config.DefaultPattern && config.Patterns[i].Enabled {
			pattern = &config.Patterns[i]
			break
		}
	}
	if pattern == nil {
		return PriceListDetailApiResponse{}, fmt.Errorf("no enabled pattern found for %s", groupKey)
	}

	// Collect all subgroups
	allSubGroups := make([]models.PriceListSubGroupResponse, 0)
	for _, priceList := range priceListData {
		allSubGroups = append(allSubGroups, priceList.SubGroups...)
	}

	// Group subgroups by PRODUCT_GROUP4
	groupedByProductGroup4 := make(map[string][]models.PriceListSubGroupResponse)
	for _, sg := range allSubGroups {
		productGroup4 := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP4")
		if productGroup4 == "" {
			// If no PRODUCT_GROUP4, use a default group
			productGroup4 = "Other"
		}
		groupedByProductGroup4[productGroup4] = append(groupedByProductGroup4[productGroup4], sg)
	}

	// Sort productGroup4 keys for consistent tab order
	productGroup4Keys := make([]string, 0, len(groupedByProductGroup4))
	for pg4 := range groupedByProductGroup4 {
		productGroup4Keys = append(productGroup4Keys, pg4)
	}
	sort.Strings(productGroup4Keys)

	// Build columns once (same for all tabs)
	columns := buildFixedColumns(pattern)

	// Create tabs for each PRODUCT_GROUP4
	tabs := make([]PriceListDetailTabConfig, 0)
	for _, productGroup4 := range productGroup4Keys {
		subGroups := groupedByProductGroup4[productGroup4]
		rowData := buildDirectRows(pattern, subGroups)

		tableData := make([]map[string]interface{}, len(rowData))
		for i, row := range rowData {
			tableData[i] = map[string]interface{}(row)
		}

		tab := PriceListDetailTabConfig{
			ID:    uuid.New(),
			Label: productGroup4,
			TableConfig: TableConfig{
				Title:             productGroup4,
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
