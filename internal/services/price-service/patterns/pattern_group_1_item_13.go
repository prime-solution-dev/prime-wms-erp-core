package patterns

import (
	"fmt"
	"sort"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func BuildGroup1Item13Response(priceListData []models.GetPriceListResponse, groupCode string) (PriceListDetailApiResponse, error) {
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

	productGroup1Code := getGroupCodeFromConfig(config, pattern, "productGroup1", "PRODUCT_GROUP1")
	productGroup7Code := getGroupCodeFromConfig(config, pattern, "productGroup7", "PRODUCT_GROUP7")
	productGroup4Code := getGroupCodeFromConfig(config, pattern, "productGroup4", "PRODUCT_GROUP4")
	productGroup3Code := getGroupCodeFromConfig(config, pattern, "productGroup3", "PRODUCT_GROUP3")
	sort.SliceStable(allSubGroups, func(i, j int) bool {
		pg1I := getValueNameByGroupCode(allSubGroups[i].SubGroupKeys, productGroup1Code)
		pg1J := getValueNameByGroupCode(allSubGroups[j].SubGroupKeys, productGroup1Code)
		if pg1I == pg1J {
			lengthI := getValueNameByGroupCode(allSubGroups[i].SubGroupKeys, productGroup7Code)
			lengthJ := getValueNameByGroupCode(allSubGroups[j].SubGroupKeys, productGroup7Code)
			if lengthI == lengthJ {
				size4I := getValueNameByGroupCode(allSubGroups[i].SubGroupKeys, productGroup4Code)
				size4J := getValueNameByGroupCode(allSubGroups[j].SubGroupKeys, productGroup4Code)
				if size4I == size4J {
					size3I := getValueNameByGroupCode(allSubGroups[i].SubGroupKeys, productGroup3Code)
					size3J := getValueNameByGroupCode(allSubGroups[j].SubGroupKeys, productGroup3Code)
					return size3I < size3J
				}
				return size4I < size4J
			}
			return lengthI < lengthJ
		}
		return pg1I < pg1J
	})

	columns := buildFixedColumns(pattern)
	rowData := buildDirectRows(config, pattern, allSubGroups)
	tableData := make([]map[string]interface{}, len(rowData))
	for i, row := range rowData {
		tableData[i] = map[string]interface{}(row)
	}

	tabLabel := "I-Beams"
	if len(allSubGroups) > 0 {
		if pg1 := getValueNameByGroupCode(allSubGroups[0].SubGroupKeys, productGroup1Code); pg1 != "" {
			tabLabel = pg1
		}
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

	return PriceListDetailApiResponse{
		Id:   uuid.MustParse(priceListData[0].ID),
		Name: "Price List Detail",
		Tabs: []PriceListDetailTabConfig{tab},
	}, nil
}
