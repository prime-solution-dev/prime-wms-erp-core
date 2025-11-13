package patterns

import (
	"fmt"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func BuildGroup1Item2Response(priceListData []models.GetPriceListResponse, groupKey string) (PriceListDetailApiResponse, error) {
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

	subGroups := make([]models.PriceListSubGroupResponse, 0)
	for _, priceList := range priceListData {
		subGroups = append(subGroups, priceList.SubGroups...)
	}

	columns := buildFixedColumns(pattern)
	rowData := buildDirectRows(pattern, subGroups)

	tableData := make([]map[string]interface{}, len(rowData))
	for i, row := range rowData {
		tableData[i] = map[string]interface{}(row)
	}

	tab := PriceListDetailTabConfig{
		ID:    uuid.New(),
		Label: "หมวดเหล็กตัวซี",
		TableConfig: TableConfig{
			Title:             "หมวดเหล็กตัวซี",
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

	return PriceListDetailApiResponse{
		Id:   uuid.MustParse(priceListData[0].ID),
		Name: "หมวดเหล็กตัวซี",
		Tabs: []PriceListDetailTabConfig{tab},
	}, nil
}
