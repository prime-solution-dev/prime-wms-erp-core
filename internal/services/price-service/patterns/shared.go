package patterns

import (
	"embed"
	"encoding/json"
	"fmt"
	"prime-erp-core/internal/models"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
)

//go:embed configs/*.json
var patternConfigs embed.FS

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
	SpanRows        bool                   `json:"spanRows,omitempty"`
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
	EnableCellSpan         bool `json:"enableCellSpan,omitempty"`
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
	EnableCellSpan         *bool `json:"enableCellSpan,omitempty"`
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
	SpanRows        *bool       `json:"spanRows,omitempty"`
}

type CellStyle struct {
	TextAlign       string `json:"textAlign,omitempty"`
	FontWeight      string `json:"fontWeight,omitempty"`
	FontSize        string `json:"fontSize,omitempty"`
	BackgroundColor string `json:"backgroundColor,omitempty"`
}

type AGGridRowData map[string]interface{}

func loadConfiguration(groupKey string) (*PriceTableConfiguration, error) {
	if groupKey == "" {
		return nil, fmt.Errorf("groupKey is required")
	}

	configPath := fmt.Sprintf("configs/%s_PATTERN.json", groupKey)

	data, err := patternConfigs.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pattern file %s: %w", configPath, err)
	}

	var config PriceTableConfiguration
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pattern file %s: %w", configPath, err)
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

	// Sort keys to ensure consistent column order
	sortedKeys := make([]string, 0, len(uniqueValues))
	for valueName := range uniqueValues {
		sortedKeys = append(sortedKeys, valueName)
	}
	sort.Strings(sortedKeys)

	for _, valueName := range sortedKeys {
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

// buildHierarchyMap builds a dynamic nested map structure from subGroups based on columnLevels
func buildHierarchyMap(subGroups []models.PriceListSubGroupResponse, columnLevels []ColumnLevel) map[string]interface{} {
	hierarchy := make(map[string]interface{})

	for _, sg := range subGroups {
		// Extract level values dynamically
		levelValues := make([]string, len(columnLevels))
		for i, level := range columnLevels {
			levelValues[i] = getValueNameByGroupCode(sg.SubGroupKeys, level.Field)
		}

		// Build nested map structure dynamically
		current := hierarchy
		for i, value := range levelValues {
			if i == len(levelValues)-1 {
				// Last level: mark as leaf
				if current[value] == nil {
					current[value] = true
				}
			} else {
				// Intermediate level: create nested map if needed
				if current[value] == nil {
					current[value] = make(map[string]interface{})
				}
				current = current[value].(map[string]interface{})
			}
		}
	}

	return hierarchy
}

// buildColumnGroupsRecursive recursively builds ColumnDef structures from hierarchy
func buildColumnGroupsRecursive(
	hierarchy map[string]interface{},
	pattern *PatternConfig,
	levelIndex int,
	levelPath []string,
) []ColumnDef {
	columns := []ColumnDef{}

	// Get sorted keys for current level
	keys := make([]string, 0, len(hierarchy))
	for key := range hierarchy {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := hierarchy[key]
		// Create a new slice to avoid modifying the shared levelPath
		currentPath := make([]string, len(levelPath)+1)
		copy(currentPath, levelPath)
		currentPath[len(levelPath)] = key

		// Build GroupID dynamically from level path
		sanitizedPathParts := make([]string, len(currentPath))
		for i, part := range currentPath {
			sanitizedPathParts[i] = sanitizeFieldName(part)
		}
		groupID := fmt.Sprintf("group_l%d_%s", levelIndex+1, strings.Join(sanitizedPathParts, "_"))

		// Check if this is a leaf node (final level)
		if levelIndex == len(pattern.ColumnLevels)-1 {
			// This is the final level - create column group with pattern.Columns as children
			columnGroup := ColumnDef{
				HeaderName:    key,
				GroupID:       groupID,
				OpenByDefault: boolPtr(true),
				Children:      []ColumnDef{},
			}

			// Build field prefix dynamically from all level values
			fieldPrefix := strings.Join(sanitizedPathParts, "_")

			// Add pattern columns as children
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

				columnGroup.Children = append(columnGroup.Children, childCol)
			}

			columns = append(columns, columnGroup)
		} else {
			// This is an intermediate level - recurse into nested map
			if nestedMap, ok := value.(map[string]interface{}); ok {
				columnGroup := ColumnDef{
					HeaderName:    key,
					GroupID:       groupID,
					OpenByDefault: boolPtr(true),
					Children:      buildColumnGroupsRecursive(nestedMap, pattern, levelIndex+1, currentPath),
				}
				columns = append(columns, columnGroup)
			}
		}
	}

	return columns
}

func buildMultiLevelColumns(pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse, columnGroupFields []string) []ColumnDef {
	if len(pattern.ColumnLevels) == 0 {
		return []ColumnDef{}
	}

	// Build hierarchy dynamically based on number of columnLevels
	hierarchy := buildHierarchyMap(subGroups, pattern.ColumnLevels)

	// Build columns recursively
	columns := buildColumnGroupsRecursive(hierarchy, pattern, 0, []string{})

	return columns
}

func buildDynamicRows(pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse) []AGGridRowData {
	rowMap := make(map[string]AGGridRowData)
	rowFields := strings.Split(pattern.Grouping.Rows, "|")
	columnGroupFields := strings.Split(pattern.Grouping.ColumnGroups, "|")
	rowCounters := make(map[string]int)
	rows := []AGGridRowData{}

	for _, sg := range subGroups {
		rowKey := buildCompositeKey(sg.SubGroupKeys, rowFields)
		if rowKey == "" {
			continue
		}

		var columnLabel string
		var columnKey string
		if len(pattern.ColumnLevels) > 0 {
			labelParts := []string{}
			keyParts := []string{}
			for _, level := range pattern.ColumnLevels {
				valueName := getValueNameByGroupCode(sg.SubGroupKeys, level.Field)
				labelParts = append(labelParts, valueName)
				keyParts = append(keyParts, sanitizeFieldName(valueName))
			}
			columnLabel = strings.Join(labelParts, " | ")
			columnKey = strings.Join(keyParts, "_")
		} else {
			columnLabel = buildCompositeKey(sg.SubGroupKeys, columnGroupFields)
			columnKey = sanitizeFieldName(columnLabel)
		}
		if columnLabel == "" {
			columnLabel = columnKey
		}
		if columnKey == "" {
			continue
		}

		compositeKey := fmt.Sprintf("%s|%s|%s", columnKey, rowKey, sg.ID)
		row, exists := rowMap[compositeKey]
		if !exists {
			row = AGGridRowData{
				"id":                 uuid.New().String(),
				"row_group_value":    rowKey,
				"column_group_value": columnLabel,
				"column_group_key":   columnKey,
			}

			for _, field := range rowFields {
				valueName := getValueNameByGroupCode(sg.SubGroupKeys, field)
				fieldName := convertGroupCodeToFieldName(field)
				row[fieldName] = valueName
			}

			rowMap[compositeKey] = row
			rows = append(rows, row)
		}

		rowCounters[columnKey]++
		rowNumberField := fmt.Sprintf("%s_row_number", columnKey)
		row[rowNumberField] = rowCounters[columnKey]

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

					if key == "awaiting_production" {
						if apMap, ok := value.(map[string]interface{}); ok {
							if importDate, ok := apMap["import_date"]; ok {
								row[fmt.Sprintf("%s_awaiting_production_%s", columnKey, sanitizeFieldName("import_date"))] = importDate
							}
							if ton, ok := apMap["ton"]; ok {
								row[fmt.Sprintf("%s_awaiting_production_%s", columnKey, sanitizeFieldName("ton"))] = ton
							}
							if producer, ok := apMap["producer"]; ok {
								row[fmt.Sprintf("%s_awaiting_production_%s", columnKey, sanitizeFieldName("producer"))] = producer
							}
						}
						continue
					}

					if key == "sale" {
						if saleMap, ok := value.(map[string]interface{}); ok {
							if fast, ok := saleMap["fast"]; ok {
								row[fmt.Sprintf("%s_sale_%s", columnKey, sanitizeFieldName("fast"))] = fast
							}
							if slow, ok := saleMap["slow"]; ok {
								row[fmt.Sprintf("%s_sale_%s", columnKey, sanitizeFieldName("slow"))] = slow
							}
						}
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
						row[fmt.Sprintf("%s_%s_tooltip", columnKey, sanitizeFieldName(baseField))] = tooltipData
					} else {
						row[fmt.Sprintf("%s_%s", columnKey, sanitizeFieldName(key))] = value
					}
				}
			}
		}

		for _, colConfig := range pattern.Columns {
			fieldName := fmt.Sprintf("%s_%s", columnKey, colConfig.Field)

			switch colConfig.DataMapping {
			case "product_group_3":
				row[fieldName] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP3")
			case "product_group_8":
				row[fieldName] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP8")
			case "product_group_6":
				row[fieldName] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP6")
			case "price_list_group_id":
				row[fieldName] = sg.PriceListGroupID
			case "subgroup_key":
				row[fieldName] = sg.SubgroupKey
			case "price_unit":
				row[fieldName] = sg.PriceUnit
			case "extra_price_unit":
				row[fieldName] = sg.ExtraPriceUnit
			case "total_net_price_unit":
				row[fieldName] = sg.TotalNetPriceUnit
			case "price_weight":
				row[fieldName] = sg.PriceWeight
			case "extra_price_weight":
				row[fieldName] = sg.ExtraPriceWeight
			case "term_price_weight":
				row[fieldName] = sg.TermPriceWeight
			case "total_net_price_weight":
				row[fieldName] = sg.TotalNetPriceWeight
			case "before_price_unit":
				row[fieldName] = sg.BeforePriceUnit
			case "before_extra_price_unit":
				row[fieldName] = sg.BeforeExtraPriceUnit
			case "before_total_net_price_unit":
				row[fieldName] = sg.BeforeTotalNetPriceUnit
			case "before_price_weight":
				row[fieldName] = sg.BeforePriceWeight
			case "before_extra_price_weight":
				row[fieldName] = sg.BeforeExtraPriceWeight
			case "before_term_price_weight":
				row[fieldName] = sg.BeforeTermPriceWeight
			case "before_total_net_price_weight":
				row[fieldName] = sg.BeforeTotalNetPriceWeight
			case "effective_date":
				row[fieldName] = sg.EffectiveDate
			case "remark":
				row[fieldName] = sg.Remark
			case "create_by":
				row[fieldName] = sg.CreateBy
			case "create_dtm":
				row[fieldName] = sg.CreateDtm
			case "update_by":
				row[fieldName] = sg.UpdateBy
			case "update_dtm":
				row[fieldName] = sg.UpdateDtm
			case "is_highlight":
				row[fieldName] = isHighlightValue
			case "inactive":
				if hasInactiveValue {
					row[fieldName] = inactiveValue
				} else {
					row[fieldName] = false
				}
			}
		}

		for _, colGroup := range pattern.ColumnGroups {
			for _, childCol := range colGroup.Children {
				fieldName := fmt.Sprintf("%s_%s", columnKey, childCol.Field)

				switch childCol.DataMapping {
				case "before_price_weight":
					row[fieldName] = sg.BeforePriceWeight
				case "total_net_price_weight":
					row[fieldName] = sg.TotalNetPriceWeight
				case "before_price_unit":
					row[fieldName] = sg.BeforePriceUnit
				case "total_net_price_unit":
					row[fieldName] = sg.TotalNetPriceUnit
				}
			}
		}

		row[fmt.Sprintf("%s_subgroup_id", columnKey)] = sg.ID
		row[fmt.Sprintf("%s_is_trading", columnKey)] = sg.IsTrading
		row["subgroup_id"] = sg.ID
		row["is_trading"] = sg.IsTrading
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
		var odValue interface{}
		var stockValue interface{}
		var importDateValue interface{}
		var tonValue interface{}
		var producerValue interface{}
		fastValue := false
		slowValue := false
		hasFastValue := false
		hasSlowValue := false
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
				if od, ok := udfData["od"]; ok {
					odValue = od
				}
				if stock, ok := udfData["stock"]; ok {
					stockValue = stock
				}
				if importDate, ok := udfData["import_date"]; ok {
					importDateValue = importDate
				}
				if ton, ok := udfData["ton"]; ok {
					tonValue = ton
				}
				if producer, ok := udfData["producer"]; ok {
					producerValue = producer
				}
				if apMap, ok := udfData["awaiting_production"].(map[string]interface{}); ok {
					if importDate, ok := apMap["import_date"]; ok {
						importDateValue = importDate
					}
					if ton, ok := apMap["ton"]; ok {
						tonValue = ton
					}
					if producer, ok := apMap["producer"]; ok {
						producerValue = producer
					}
				}
				if fast, ok := udfData["fast"].(bool); ok {
					fastValue = fast
					hasFastValue = true
				}
				if slow, ok := udfData["slow"].(bool); ok {
					slowValue = slow
					hasSlowValue = true
				}
				if saleMap, ok := udfData["sale"].(map[string]interface{}); ok {
					if fast, ok := saleMap["fast"].(bool); ok {
						fastValue = fast
						hasFastValue = true
					}
					if slow, ok := saleMap["slow"].(bool); ok {
						slowValue = slow
						hasSlowValue = true
					}
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
			case "product_group_2":
				row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP2")
			case "product_group_3":
				row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP3")
			case "product_group_4":
				row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP4")
			case "product_group_8":
				row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP8")
			case "product_group_6":
				row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP6")
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
			case "od":
				row[fixedCol.Field] = odValue
			case "stock":
				row[fixedCol.Field] = stockValue
			case "extra_price_unit":
				row[fixedCol.Field] = sg.ExtraPriceUnit
			case "remark":
				row[fixedCol.Field] = sg.Remark
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
				case "import_date":
					row[childCol.Field] = importDateValue
				case "ton":
					row[childCol.Field] = tonValue
				case "producer":
					row[childCol.Field] = producerValue
				case "fast":
					if hasFastValue {
						row[childCol.Field] = fastValue
					} else {
						row[childCol.Field] = false
					}
				case "slow":
					if hasSlowValue {
						row[childCol.Field] = slowValue
					} else {
						row[childCol.Field] = false
					}
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
			case "od":
				row[colConfig.Field] = odValue
			case "stock":
				row[colConfig.Field] = stockValue
			case "import_date":
				row[colConfig.Field] = importDateValue
			case "ton":
				row[colConfig.Field] = tonValue
			case "producer":
				row[colConfig.Field] = producerValue
			case "fast":
				if hasFastValue {
					row[colConfig.Field] = fastValue
				} else {
					row[colConfig.Field] = false
				}
			case "slow":
				if hasSlowValue {
					row[colConfig.Field] = slowValue
				} else {
					row[colConfig.Field] = false
				}
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

		if fixedCol.SpanRows {
			col.SpanRows = boolPtr(true)
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

			if childColConfig.SpanRows {
				childCol.SpanRows = boolPtr(true)
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
