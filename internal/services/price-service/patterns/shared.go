package patterns

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"prime-erp-core/internal/models"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

type PatternConfig struct {
	ID                   string              `json:"id"`
	Name                 string              `json:"name"`
	Description          string              `json:"description"`
	Enabled              bool                `json:"enabled"`
	Grouping             GroupingConfig      `json:"grouping"`
	ColumnLevels         []ColumnLevel       `json:"columnLevels,omitempty"`
	Columns              []ColumnConfigItem  `json:"columns"`
	FixedColumns         []ColumnConfigItem  `json:"fixedColumns"`
	ColumnGroups         []ColumnGroupConfig `json:"columnGroups,omitempty"`
	ApplicableCategories []string            `json:"applicableCategories"`
	EditableSuffixes     []string            `json:"editable_suffixes,omitempty"`
	FetchableSuffixes    []string            `json:"fetchable_suffixes,omitempty"`
}

type GroupingConfig struct {
	Tabs         string `json:"tabs"`
	Rows         string `json:"rows"`
	ColumnGroups string `json:"columnGroups"`
}

type ColumnLevel struct {
	Level    int      `json:"level"`
	Field    string   `json:"field"`
	Examples []string `json:"examples"`
}

type ColumnConfigItem struct {
	Field           string                 `json:"field"`
	HeaderName      string                 `json:"headerName"`
	Width           int                    `json:"width"`
	Pinned          string                 `json:"pinned,omitempty"`
	LockPosition    bool                   `json:"lockPosition,omitempty"`
	SuppressMovable bool                   `json:"suppressMovable,omitempty"`
	ValueGetter     string                 `json:"valueGetter,omitempty"`
	CellRenderer    string                 `json:"cellRenderer,omitempty"`
	CellStyle       map[string]interface{} `json:"cellStyle,omitempty"`
	DataMapping     string                 `json:"dataMapping,omitempty"`
	EnableTooltip   bool                   `json:"enableTooltip,omitempty"`
}

type ColumnGroupConfig struct {
	HeaderName    string             `json:"headerName"`
	GroupID       string             `json:"groupId"`
	OpenByDefault bool               `json:"openByDefault,omitempty"`
	Children      []ColumnConfigItem `json:"children"`
}

type TableConfigSettings struct {
	GroupHeaderHeight int               `json:"groupHeaderHeight"`
	HeaderHeight      int               `json:"headerHeight"`
	Pagination        bool              `json:"pagination"`
	Toolbar           ToolbarConfig     `json:"toolbar"`
	GridOptions       GridOptionsConfig `json:"gridOptions"`
}

type ToolbarConfig struct {
	Show             bool `json:"show"`
	ShowSearch       bool `json:"showSearch"`
	ShowRefresh      bool `json:"showRefresh"`
	ShowColumnToggle bool `json:"showColumnToggle"`
}

type GridOptionsConfig struct {
	SuppressMovableColumns bool `json:"suppressMovableColumns"`
	SuppressMenuHide       bool `json:"suppressMenuHide"`
}

type PriceTableConfiguration struct {
	Patterns       []PatternConfig     `json:"patterns"`
	DefaultPattern string              `json:"defaultPattern"`
	TableConfig    TableConfigSettings `json:"tableConfig"`
}

type PriceListDetailApiResponse struct {
	Id   uuid.UUID                  `json:"id"`
	Name string                     `json:"name"`
	Tabs []PriceListDetailTabConfig `json:"tabs"`
}

type PriceListDetailTabConfig struct {
	ID                uuid.UUID                `json:"id"`
	Label             string                   `json:"label"`
	TableConfig       TableConfig              `json:"tableConfig"`
	TableData         []map[string]interface{} `json:"tableData"`
	EditableSuffixes  []string                 `json:"editable_suffixes,omitempty"`
	FetchableSuffixes []string                 `json:"fetchable_suffixes,omitempty"`
}

type TableConfig struct {
	Title             string       `json:"title,omitempty"`
	Toolbar           *Toolbar     `json:"toolbar,omitempty"`
	Pagination        *bool        `json:"pagination,omitempty"`
	GroupHeaderHeight *int         `json:"groupHeaderHeight,omitempty"`
	HeaderHeight      *int         `json:"headerHeight,omitempty"`
	Columns           []ColumnDef  `json:"columns"`
	GridOptions       *GridOptions `json:"gridOptions,omitempty"`
}

type Toolbar struct {
	Show             *bool `json:"show,omitempty"`
	ShowSearch       *bool `json:"showSearch,omitempty"`
	ShowRefresh      *bool `json:"showRefresh,omitempty"`
	ShowColumnToggle *bool `json:"showColumnToggle,omitempty"`
}

type GridOptions struct {
	SuppressMovableColumns *bool `json:"suppressMovableColumns,omitempty"`
	SuppressMenuHide       *bool `json:"suppressMenuHide,omitempty"`
}

type ColumnDef struct {
	Field           string      `json:"field,omitempty"`
	HeaderName      string      `json:"headerName,omitempty"`
	Width           *int        `json:"width,omitempty"`
	Pinned          string      `json:"pinned,omitempty"`
	LockPosition    *bool       `json:"lockPosition,omitempty"`
	SuppressMovable *bool       `json:"suppressMovable,omitempty"`
	ValueGetter     string      `json:"valueGetter,omitempty"`
	Filter          string      `json:"filter,omitempty"`
	CellRenderer    string      `json:"cellRenderer,omitempty"`
	CellStyle       *CellStyle  `json:"cellStyle,omitempty"`
	HeaderClass     string      `json:"headerClass,omitempty"`
	EnableTooltip   *bool       `json:"enableTooltip,omitempty"`
	GroupID         string      `json:"groupId,omitempty"`
	OpenByDefault   *bool       `json:"openByDefault,omitempty"`
	Children        []ColumnDef `json:"children,omitempty"`
}

type CellStyle struct {
	TextAlign       string `json:"textAlign,omitempty"`
	FontWeight      string `json:"fontWeight,omitempty"`
	FontSize        string `json:"fontSize,omitempty"`
	BackgroundColor string `json:"backgroundColor,omitempty"`
}

type AGGridRowData map[string]interface{}

func loadConfiguration(groupKey string) (*PriceTableConfiguration, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	if groupKey == "" {
		groupKey = "GROUP_1_ITEM_1"
	}

	patternFileName := fmt.Sprintf("%s_PATTERN.json", groupKey)
	configDir := filepath.Join(currentDir, "internal", "services", "price-service", "patterns", "configs")
	configPath := filepath.Join(configDir, patternFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if groupKey != "GROUP_1_ITEM_1" {
			fallbackPath := filepath.Join(configDir, "GROUP_1_ITEM_1_PATTERN.json")
			data, err = os.ReadFile(fallbackPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read pattern file %s and fallback GROUP_1_ITEM_1_PATTERN.json: %w", patternFileName, err)
			}
		} else {
			return nil, fmt.Errorf("failed to read pattern file %s: %w", patternFileName, err)
		}
	}

	var config PriceTableConfiguration
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pattern file %s: %w", patternFileName, err)
	}

	return &config, nil
}

func getValueNameByGroupCode(subGroupKeys []models.PriceListSubGroupKeyResponse, groupCode string) string {
	for _, sgk := range subGroupKeys {
		if sgk.GroupCode == groupCode {
			return sgk.ValueName
		}
	}
	return ""
}

func buildCompositeKey(subGroupKeys []models.PriceListSubGroupKeyResponse, groupCodes []string) string {
	parts := []string{}
	for _, code := range groupCodes {
		valueName := getValueNameByGroupCode(subGroupKeys, code)
		if valueName != "" {
			parts = append(parts, valueName)
		}
	}
	return strings.Join(parts, "|")
}

func ExtractGroupKey(subgroupKey string) string {
	if subgroupKey == "" {
		return ""
	}
	parts := strings.Split(subgroupKey, "|")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func selectPatternForCategory(config *PriceTableConfiguration, productGroup2ValueName string) *PatternConfig {
	for _, pattern := range config.Patterns {
		if !pattern.Enabled {
			continue
		}
		for _, category := range pattern.ApplicableCategories {
			if category == productGroup2ValueName {
				return &pattern
			}
		}
	}

	for _, pattern := range config.Patterns {
		if pattern.ID == config.DefaultPattern {
			return &pattern
		}
	}

	for _, pattern := range config.Patterns {
		if pattern.Enabled {
			return &pattern
		}
	}

	return nil
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func sanitizeFieldName(name string) string {
	name = regexp.MustCompile(`[^\w]+`).ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	return strings.ToLower(name)
}

func convertGroupCodeToFieldName(groupCode string) string {
	fieldName := strings.ToLower(groupCode)
	re := regexp.MustCompile(`([a-z]+)(\d+)$`)
	fieldName = re.ReplaceAllString(fieldName, "${1}_${2}")
	return fieldName
}

func groupDataByGroupKeyAndProductGroup2(priceListData []models.GetPriceListResponse) map[string]map[string][]models.PriceListSubGroupResponse {
	groupedData := make(map[string]map[string][]models.PriceListSubGroupResponse)

	for _, priceList := range priceListData {
		groupKey := priceList.GroupKey
		if groupKey == "" {
			if len(priceList.SubGroups) > 0 {
				groupKey = ExtractGroupKey(priceList.SubGroups[0].SubgroupKey)
			}
			if groupKey == "" {
				continue
			}
		}

		for _, subGroup := range priceList.SubGroups {
			productGroup2 := getValueNameByGroupCode(subGroup.SubGroupKeys, "PRODUCT_GROUP2")
			if productGroup2 == "" {
				continue
			}

			if groupedData[groupKey] == nil {
				groupedData[groupKey] = make(map[string][]models.PriceListSubGroupResponse)
			}

			groupedData[groupKey][productGroup2] = append(groupedData[groupKey][productGroup2], subGroup)
		}
	}

	return groupedData
}

func buildDynamicColumns(pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse) []ColumnDef {
	columns := []ColumnDef{}

	for _, fixedCol := range pattern.FixedColumns {
		col := ColumnDef{
			Field:           fixedCol.Field,
			HeaderName:      fixedCol.HeaderName,
			Width:           intPtr(fixedCol.Width),
			Pinned:          fixedCol.Pinned,
			LockPosition:    boolPtr(fixedCol.LockPosition),
			SuppressMovable: boolPtr(fixedCol.SuppressMovable),
			ValueGetter:     fixedCol.ValueGetter,
		}

		if fixedCol.CellStyle != nil {
			col.CellStyle = convertCellStyle(fixedCol.CellStyle)
		}

		columns = append(columns, col)
	}

	columnGroupFields := strings.Split(pattern.Grouping.ColumnGroups, "|")

	if len(pattern.ColumnLevels) > 0 {
		columns = append(columns, buildMultiLevelColumns(pattern, subGroups, columnGroupFields)...)
	} else {
		columns = append(columns, buildSingleLevelColumns(pattern, subGroups, columnGroupFields)...)
	}

	return columns
}

func buildSingleLevelColumns(pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse, columnGroupFields []string) []ColumnDef {
	columns := []ColumnDef{}
	uniqueValues := make(map[string]bool)
	for _, sg := range subGroups {
		key := buildCompositeKey(sg.SubGroupKeys, columnGroupFields)
		if key != "" {
			uniqueValues[key] = true
		}
	}

	for valueName := range uniqueValues {
		columnGroup := ColumnDef{
			HeaderName:    valueName,
			GroupID:       fmt.Sprintf("group_%s", sanitizeFieldName(valueName)),
			OpenByDefault: boolPtr(true),
			Children:      []ColumnDef{},
		}

		for _, colConfig := range pattern.Columns {
			childCol := ColumnDef{
				Field:        fmt.Sprintf("%s_%s", sanitizeFieldName(valueName), colConfig.Field),
				HeaderName:   colConfig.HeaderName,
				Width:        intPtr(colConfig.Width),
				CellRenderer: colConfig.CellRenderer,
			}

			if colConfig.CellStyle != nil {
				childCol.CellStyle = convertCellStyle(colConfig.CellStyle)
			}

			if colConfig.EnableTooltip {
				childCol.EnableTooltip = boolPtr(true)
			}

			columnGroup.Children = append(columnGroup.Children, childCol)
		}

		columns = append(columns, columnGroup)
	}

	return columns
}

func buildMultiLevelColumns(pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse, columnGroupFields []string) []ColumnDef {
	hierarchy := make(map[string]map[string]map[string]bool)

	for _, sg := range subGroups {
		level1 := ""
		level2 := ""
		level3 := ""

		if len(pattern.ColumnLevels) > 0 {
			level1 = getValueNameByGroupCode(sg.SubGroupKeys, pattern.ColumnLevels[0].Field)
		}
		if len(pattern.ColumnLevels) > 1 {
			level2 = getValueNameByGroupCode(sg.SubGroupKeys, pattern.ColumnLevels[1].Field)
		}
		if len(pattern.ColumnLevels) > 2 {
			level3 = getValueNameByGroupCode(sg.SubGroupKeys, pattern.ColumnLevels[2].Field)
		}

		if hierarchy[level1] == nil {
			hierarchy[level1] = make(map[string]map[string]bool)
		}
		if hierarchy[level1][level2] == nil {
			hierarchy[level1][level2] = make(map[string]bool)
		}
		hierarchy[level1][level2][level3] = true
	}

	columns := []ColumnDef{}
	for l1, l2Map := range hierarchy {
		l1Group := ColumnDef{
			HeaderName:    l1,
			GroupID:       fmt.Sprintf("group_l1_%s", sanitizeFieldName(l1)),
			OpenByDefault: boolPtr(true),
			Children:      []ColumnDef{},
		}

		for l2, l3Map := range l2Map {
			l2Group := ColumnDef{
				HeaderName:    l2,
				GroupID:       fmt.Sprintf("group_l2_%s_%s", sanitizeFieldName(l1), sanitizeFieldName(l2)),
				OpenByDefault: boolPtr(true),
				Children:      []ColumnDef{},
			}

			for l3 := range l3Map {
				l3Group := ColumnDef{
					HeaderName:    l3,
					GroupID:       fmt.Sprintf("group_l3_%s_%s_%s", sanitizeFieldName(l1), sanitizeFieldName(l2), sanitizeFieldName(l3)),
					OpenByDefault: boolPtr(true),
					Children:      []ColumnDef{},
				}

				fieldPrefix := fmt.Sprintf("%s_%s_%s", sanitizeFieldName(l1), sanitizeFieldName(l2), sanitizeFieldName(l3))
				for _, colConfig := range pattern.Columns {
					childCol := ColumnDef{
						Field:        fmt.Sprintf("%s_%s", fieldPrefix, colConfig.Field),
						HeaderName:   colConfig.HeaderName,
						Width:        intPtr(colConfig.Width),
						CellRenderer: colConfig.CellRenderer,
					}

					if colConfig.CellStyle != nil {
						childCol.CellStyle = convertCellStyle(colConfig.CellStyle)
					}

					if colConfig.EnableTooltip {
						childCol.EnableTooltip = boolPtr(true)
					}

					l3Group.Children = append(l3Group.Children, childCol)
				}

				l2Group.Children = append(l2Group.Children, l3Group)
			}

			l1Group.Children = append(l1Group.Children, l2Group)
		}

		columns = append(columns, l1Group)
	}

	return columns
}

func buildDynamicRows(pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse) []AGGridRowData {
	rowMap := make(map[string]AGGridRowData)
	rowFields := strings.Split(pattern.Grouping.Rows, "|")
	columnGroupFields := strings.Split(pattern.Grouping.ColumnGroups, "|")

	for _, sg := range subGroups {
		rowKey := buildCompositeKey(sg.SubGroupKeys, rowFields)
		if rowKey == "" {
			continue
		}

		if _, exists := rowMap[rowKey]; !exists {
			rowMap[rowKey] = AGGridRowData{
				"id": uuid.New().String(),
			}

			for _, field := range rowFields {
				valueName := getValueNameByGroupCode(sg.SubGroupKeys, field)
				fieldName := convertGroupCodeToFieldName(field)
				rowMap[rowKey][fieldName] = valueName
			}
		}

		var columnKey string
		if len(pattern.ColumnLevels) > 0 {
			parts := []string{}
			for _, level := range pattern.ColumnLevels {
				valueName := getValueNameByGroupCode(sg.SubGroupKeys, level.Field)
				parts = append(parts, sanitizeFieldName(valueName))
			}
			columnKey = strings.Join(parts, "_")
		} else {
			columnKey = sanitizeFieldName(buildCompositeKey(sg.SubGroupKeys, columnGroupFields))
		}

		udfData := make(map[string]interface{})
		isHighlightValue := false
		inactiveValue := false
		hasInactiveValue := false
		if len(sg.UdfJson) > 0 {
			if err := json.Unmarshal(sg.UdfJson, &udfData); err == nil {
				if h, ok := udfData["is_highlight"].(bool); ok {
					isHighlightValue = h
				}
				if inactive, ok := udfData["inactive"].(bool); ok {
					inactiveValue = inactive
					hasInactiveValue = true
				}
				for key, value := range udfData {
					if key == "is_highlight" || key == "inactive" {
						continue
					}

					if strings.HasSuffix(key, "_tooltip") {
						baseField := strings.TrimSuffix(key, "_tooltip")
						tooltipData := make(map[string]interface{})
						if tooltipMap, ok := value.(map[string]interface{}); ok {
							if text, hasText := tooltipMap["text"]; hasText {
								tooltipData["text"] = text
							}
							if icon, hasIcon := tooltipMap["icon"]; hasIcon {
								tooltipData["icon"] = icon
							}
						} else {
							tooltipData["text"] = value
						}
						rowMap[rowKey][fmt.Sprintf("%s_%s_tooltip", columnKey, sanitizeFieldName(baseField))] = tooltipData
					} else {
						rowMap[rowKey][fmt.Sprintf("%s_%s", columnKey, sanitizeFieldName(key))] = value
					}
				}
			}
		}

		for _, colConfig := range pattern.Columns {
			fieldName := fmt.Sprintf("%s_%s", columnKey, colConfig.Field)

			switch colConfig.DataMapping {
			case "product_group_3":
				rowMap[rowKey][fieldName] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP3")
			case "product_group_6":
				rowMap[rowKey][fieldName] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP6")
			case "price_list_group_id":
				rowMap[rowKey][fieldName] = sg.PriceListGroupID
			case "subgroup_key":
				rowMap[rowKey][fieldName] = sg.SubgroupKey
			case "price_unit":
				rowMap[rowKey][fieldName] = sg.PriceUnit
			case "extra_price_unit":
				rowMap[rowKey][fieldName] = sg.ExtraPriceUnit
			case "total_net_price_unit":
				rowMap[rowKey][fieldName] = sg.TotalNetPriceUnit
			case "price_weight":
				rowMap[rowKey][fieldName] = sg.PriceWeight
			case "extra_price_weight":
				rowMap[rowKey][fieldName] = sg.ExtraPriceWeight
			case "term_price_weight":
				rowMap[rowKey][fieldName] = sg.TermPriceWeight
			case "total_net_price_weight":
				rowMap[rowKey][fieldName] = sg.TotalNetPriceWeight
			case "before_price_unit":
				rowMap[rowKey][fieldName] = sg.BeforePriceUnit
			case "before_extra_price_unit":
				rowMap[rowKey][fieldName] = sg.BeforeExtraPriceUnit
			case "before_total_net_price_unit":
				rowMap[rowKey][fieldName] = sg.BeforeTotalNetPriceUnit
			case "before_price_weight":
				rowMap[rowKey][fieldName] = sg.BeforePriceWeight
			case "before_extra_price_weight":
				rowMap[rowKey][fieldName] = sg.BeforeExtraPriceWeight
			case "before_term_price_weight":
				rowMap[rowKey][fieldName] = sg.BeforeTermPriceWeight
			case "before_total_net_price_weight":
				rowMap[rowKey][fieldName] = sg.BeforeTotalNetPriceWeight
			case "effective_date":
				rowMap[rowKey][fieldName] = sg.EffectiveDate
			case "remark":
				rowMap[rowKey][fieldName] = sg.Remark
			case "create_by":
				rowMap[rowKey][fieldName] = sg.CreateBy
			case "create_dtm":
				rowMap[rowKey][fieldName] = sg.CreateDtm
			case "update_by":
				rowMap[rowKey][fieldName] = sg.UpdateBy
			case "update_dtm":
				rowMap[rowKey][fieldName] = sg.UpdateDtm
			case "is_highlight":
				rowMap[rowKey][fieldName] = isHighlightValue
			case "inactive":
				if hasInactiveValue {
					rowMap[rowKey][fieldName] = inactiveValue
				} else {
					rowMap[rowKey][fieldName] = false
				}
			}
		}

		rowMap[rowKey][fmt.Sprintf("%s_subgroup_id", columnKey)] = sg.ID
		rowMap[rowKey][fmt.Sprintf("%s_is_trading", columnKey)] = sg.IsTrading
	}

	rows := []AGGridRowData{}
	for _, row := range rowMap {
		rows = append(rows, row)
	}

	return rows
}

func buildDirectRows(pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse) []AGGridRowData {
	rows := []AGGridRowData{}

	for _, sg := range subGroups {
		row := AGGridRowData{
			"id": uuid.New().String(),
		}

		udfData := make(map[string]interface{})
		isHighlightValue := false
		inactiveValue := false
		hasInactiveValue := false
		var lineBundleValue *float64
		var marketWeightValue *float64
		if len(sg.UdfJson) > 0 {
			if err := json.Unmarshal(sg.UdfJson, &udfData); err == nil {
				if h, ok := udfData["is_highlight"].(bool); ok {
					isHighlightValue = h
				}
				if inactive, ok := udfData["inactive"].(bool); ok {
					inactiveValue = inactive
					hasInactiveValue = true
				}
				if lb, ok := udfData["line_bundle"].(float64); ok {
					lineBundleValue = &lb
				} else if lb, ok := udfData["line_bundle"].(int); ok {
					lbFloat := float64(lb)
					lineBundleValue = &lbFloat
				}
				if mw, ok := udfData["market_weight"].(float64); ok {
					marketWeightValue = &mw
				} else if mw, ok := udfData["market_weight"].(int); ok {
					mwFloat := float64(mw)
					marketWeightValue = &mwFloat
				}
			}
		}

		itemParts := []string{}
		for _, code := range []string{"PRODUCT_GROUP4", "PRODUCT_GROUP6", "PRODUCT_GROUP7"} {
			valueName := getValueNameByGroupCode(sg.SubGroupKeys, code)
			if valueName != "" {
				itemParts = append(itemParts, valueName)
			}
		}
		row["item"] = strings.Join(itemParts, "x")

		for _, fixedCol := range pattern.FixedColumns {
			switch fixedCol.DataMapping {
			case "item":
			case "type":
				row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP9")
			case "price_weight":
				row[fixedCol.Field] = sg.PriceWeight
			case "market_weight":
				if marketWeightValue != nil {
					row[fixedCol.Field] = *marketWeightValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "total_net_price_weight":
				row[fixedCol.Field] = sg.TotalNetPriceWeight
			case "is_highlight":
				row[fixedCol.Field] = isHighlightValue
			case "inactive":
				if hasInactiveValue {
					row[fixedCol.Field] = inactiveValue
				} else {
					row[fixedCol.Field] = false
				}
			}
		}

		for _, colGroup := range pattern.ColumnGroups {
			for _, childCol := range colGroup.Children {
				switch childCol.DataMapping {
				case "before_price_weight":
					row[childCol.Field] = sg.BeforePriceWeight
				case "total_net_price_weight":
					row[childCol.Field] = sg.TotalNetPriceWeight
				case "before_price_unit":
					row[childCol.Field] = sg.BeforePriceUnit
				case "total_net_price_unit":
					row[childCol.Field] = sg.TotalNetPriceUnit
				}
			}
		}

		for _, colConfig := range pattern.Columns {
			switch colConfig.DataMapping {
			case "extra_price_unit":
				row[colConfig.Field] = sg.ExtraPriceUnit
			case "line_bundle":
				if lineBundleValue != nil {
					row[colConfig.Field] = *lineBundleValue
				} else {
					row[colConfig.Field] = nil
				}
			case "remark":
				row[colConfig.Field] = sg.Remark
			}
		}

		row["subgroup_id"] = sg.ID
		row["is_trading"] = sg.IsTrading

		rows = append(rows, row)
	}

	return rows
}

func buildFixedColumns(pattern *PatternConfig) []ColumnDef {
	columns := []ColumnDef{}

	for _, fixedCol := range pattern.FixedColumns {
		col := ColumnDef{
			Field:           fixedCol.Field,
			HeaderName:      fixedCol.HeaderName,
			Width:           intPtr(fixedCol.Width),
			Pinned:          fixedCol.Pinned,
			LockPosition:    boolPtr(fixedCol.LockPosition),
			SuppressMovable: boolPtr(fixedCol.SuppressMovable),
			ValueGetter:     fixedCol.ValueGetter,
		}

		if fixedCol.CellStyle != nil {
			col.CellStyle = convertCellStyle(fixedCol.CellStyle)
		}

		if fixedCol.CellRenderer != "" {
			col.CellRenderer = fixedCol.CellRenderer
		}

		columns = append(columns, col)
	}

	for _, colGroupConfig := range pattern.ColumnGroups {
		columnGroup := ColumnDef{
			HeaderName:    colGroupConfig.HeaderName,
			GroupID:       colGroupConfig.GroupID,
			OpenByDefault: boolPtr(colGroupConfig.OpenByDefault),
			Children:      []ColumnDef{},
		}

		for _, childColConfig := range colGroupConfig.Children {
			childCol := ColumnDef{
				Field:      childColConfig.Field,
				HeaderName: childColConfig.HeaderName,
				Width:      intPtr(childColConfig.Width),
			}

			if childColConfig.CellStyle != nil {
				childCol.CellStyle = convertCellStyle(childColConfig.CellStyle)
			}

			if childColConfig.CellRenderer != "" {
				childCol.CellRenderer = childColConfig.CellRenderer
			}

			if childColConfig.EnableTooltip {
				childCol.EnableTooltip = boolPtr(true)
			}

			columnGroup.Children = append(columnGroup.Children, childCol)
		}

		columns = append(columns, columnGroup)
	}

	for _, colConfig := range pattern.Columns {
		col := ColumnDef{
			Field:      colConfig.Field,
			HeaderName: colConfig.HeaderName,
			Width:      intPtr(colConfig.Width),
		}

		if colConfig.CellStyle != nil {
			col.CellStyle = convertCellStyle(colConfig.CellStyle)
		}

		if colConfig.CellRenderer != "" {
			col.CellRenderer = colConfig.CellRenderer
		}

		if colConfig.EnableTooltip {
			col.EnableTooltip = boolPtr(true)
		}

		columns = append(columns, col)
	}

	return columns
}

func convertCellStyle(styleMap map[string]interface{}) *CellStyle {
	style := &CellStyle{}

	if val, ok := styleMap["textAlign"].(string); ok {
		style.TextAlign = val
	}
	if val, ok := styleMap["fontWeight"].(string); ok {
		style.FontWeight = val
	}
	if val, ok := styleMap["fontSize"].(string); ok {
		style.FontSize = val
	}
	if val, ok := styleMap["backgroundColor"].(string); ok {
		style.BackgroundColor = val
	}

	return style
}
