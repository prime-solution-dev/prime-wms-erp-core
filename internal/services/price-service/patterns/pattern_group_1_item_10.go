package patterns

import (
	"fmt"
	"sort"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func BuildGroup1Item10Response(priceListData []models.GetPriceListResponse, groupCode string) (PriceListDetailApiResponse, error) {
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

	productGroup4Code := getGroupCodeFromConfig(config, pattern, "productGroup4", "PRODUCT_GROUP4")
	productGroup7Code := getGroupCodeFromConfig(config, pattern, "productGroup7", "PRODUCT_GROUP7")

	SortSubGroupsByValue(allSubGroups, productGroup4Code, productGroup7Code)
	columns := buildDynamicColumns(pattern, allSubGroups)
	rows := buildDynamicRows(config, pattern, allSubGroups)
	mergedRows := mergeGroup1Item9Rows(rows)

	// total_weight ไม่ได้มาจาก product group จึงยัง tie-break ด้วย float ตามเดิม
	// ใช้ SliceStable และคืน false เมื่อแกน product group ต่างกัน เพื่อไม่ทำลาย
	// ลำดับที่เรียงมาแล้วจากต้นทาง
	sort.SliceStable(mergedRows, func(i, j int) bool {
		itemI := fmt.Sprintf("%v", mergedRows[i]["product_group_4"])
		itemJ := fmt.Sprintf("%v", mergedRows[j]["product_group_4"])
		lengthI := fmt.Sprintf("%v", mergedRows[i]["product_group_7"])
		lengthJ := fmt.Sprintf("%v", mergedRows[j]["product_group_7"])
		if itemI != itemJ || lengthI != lengthJ {
			return false
		}
		weightI, _ := toFloat64(mergedRows[i]["total_weight"])
		weightJ, _ := toFloat64(mergedRows[j]["total_weight"])
		return weightI < weightJ
	})

	tableData := make([]map[string]interface{}, len(mergedRows))
	for i, row := range mergedRows {
		tableData[i] = map[string]interface{}(row)
	}

	tabLabel := pattern.Name
	if tabLabel == "" {
		tabLabel = groupCode
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
