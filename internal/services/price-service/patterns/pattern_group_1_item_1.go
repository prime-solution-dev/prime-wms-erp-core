package patterns

import (
	"fmt"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func BuildGroup1Item1Response(priceListData []models.GetPriceListResponse) (PriceListDetailApiResponse, error) {
	groupedData := groupDataByGroupKeyAndProductGroup2(priceListData)
	tabs := make([]PriceListDetailTabConfig, 0)
	var loadErr error

	for groupKey, productGroup2Map := range groupedData {
		config, err := loadConfiguration(groupKey)
		if err != nil {
			loadErr = fmt.Errorf("load configuration for %s: %w", groupKey, err)
			continue
		}

		for productGroup2, subGroups := range productGroup2Map {
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

			tabs = append(tabs, PriceListDetailTabConfig{
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
					},
					Columns: columns,
				},
				TableData:        tableData,
				EditableSuffixes: pattern.EditableSuffixes,
			})
		}
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
