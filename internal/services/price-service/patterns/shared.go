package patterns

import (
	"embed"
	"encoding/json"
	"fmt"
	"prime-erp-core/internal/models"
	"regexp"
	"sort"
	"strconv"
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
	Summary              *SummaryConfig      `json:"summary,omitempty"`
	Grouping             GroupingConfig      `json:"grouping"`
	ColumnLevels         []ColumnLevel       `json:"columnLevels,omitempty"`
	Columns              []ColumnConfigItem  `json:"columns"`
	FixedColumns         []ColumnConfigItem  `json:"fixedColumns"`
	ColumnGroups         []ColumnGroupConfig `json:"columnGroups,omitempty"`
	ApplicableCategories []string            `json:"applicableCategories"`
	EditableSuffixes     []string            `json:"editable_suffixes,omitempty"`
	FetchableSuffixes    []string            `json:"fetchable_suffixes,omitempty"`
	ItemFormat           []ItemFormatPart    `json:"itemFormat,omitempty"`
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
	SummaryRows       []SummaryRow             `json:"summaryRows,omitempty"`
	SummaryField      map[string]interface{}   `json:"summaryField,omitempty"`
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

type SummaryConfig struct {
	RowGroupField string                 `json:"rowGroupField"`
	LabelField    string                 `json:"labelField,omitempty"`
	LabelValue    string                 `json:"labelValue,omitempty"`
	Columns       []SummaryColumnConfig  `json:"columns"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type SummaryColumnConfig struct {
	Field               string `json:"field"`
	Aggregation         string `json:"aggregation"`
	ApplyToColumnGroups *bool  `json:"applyToColumnGroups,omitempty"`
}

type SummaryRow struct {
	RowGroupValue string                 `json:"row_group_value"`
	Label         string                 `json:"label,omitempty"`
	Data          map[string]interface{} `json:"data"`
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

type ItemFormatPart struct {
	Type  string `json:"type"`  // "group" or "literal"
	Value string `json:"value"` // group code or literal text
}

var defaultItemFormat = []ItemFormatPart{
	{Type: "group", Value: "PRODUCT_GROUP4"},
	{Type: "literal", Value: "x"},
	{Type: "group", Value: "PRODUCT_GROUP6"},
	{Type: "literal", Value: "x"},
	{Type: "group", Value: "PRODUCT_GROUP7"},
}

func loadConfiguration(groupCode string) (*PriceTableConfiguration, error) {
	if groupCode == "" {
		return nil, fmt.Errorf("groupCode is required")
	}

	configPath := fmt.Sprintf("configs/%s_PATTERN.json", groupCode)

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

func getValueCodeByGroupCode(subGroupKeys []models.PriceListSubGroupKeyResponse, groupCode string) string {
	for _, sgk := range subGroupKeys {
		if sgk.GroupCode == groupCode {
			return sgk.ValueCode
		}
	}
	return ""
}

func buildCompositeKey(subGroupKeys []models.PriceListSubGroupKeyResponse, groupCodes []string) string {
	return buildCompositeKeyBy(subGroupKeys, groupCodes, getValueNameByGroupCode)
}

func buildCompositeCodeKey(subGroupKeys []models.PriceListSubGroupKeyResponse, groupCodes []string) string {
	return buildCompositeKeyBy(subGroupKeys, groupCodes, getValueCodeByGroupCode)
}

func buildCompositeKeyBy(subGroupKeys []models.PriceListSubGroupKeyResponse, groupCodes []string, extractor func([]models.PriceListSubGroupKeyResponse, string) string) string {
	parts := []string{}
	for _, code := range groupCodes {
		value := extractor(subGroupKeys, code)
		if value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, "|")
}

func sanitizeIdentifier(primary, fallback string) string {
	if sanitized := sanitizeFieldName(primary); sanitized != "" {
		return sanitized
	}
	if sanitized := sanitizeFieldName(fallback); sanitized != "" {
		return sanitized
	}
	return "value"
}

const hierarchyKeySeparator = "|:|"

func composeHierarchyKey(code, label string) string {
	return fmt.Sprintf("%s%s%s", code, hierarchyKeySeparator, label)
}

func splitHierarchyKey(key string) (string, string) {
	parts := strings.SplitN(key, hierarchyKeySeparator, 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", key
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

		if fixedCol.CellRenderer != "" {
			col.CellRenderer = fixedCol.CellRenderer
		}

		if fixedCol.SpanRows {
			col.SpanRows = boolPtr(true)
		}

		columns = append(columns, col)
	}

	columnGroupFields := strings.Split(pattern.Grouping.ColumnGroups, "|")

	if len(pattern.ColumnLevels) > 0 {
		columns = append(columns, buildMultiLevelColumns(pattern, subGroups)...)
	} else {
		columns = append(columns, buildSingleLevelColumns(pattern, subGroups, columnGroupFields)...)
	}

	return columns
}

func buildSingleLevelColumns(pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse, columnGroupFields []string) []ColumnDef {
	type columnGroupValue struct {
		Label string
		Code  string
	}

	columns := []ColumnDef{}
	uniqueValues := make(map[string]columnGroupValue)
	for _, sg := range subGroups {
		label := buildCompositeKey(sg.SubGroupKeys, columnGroupFields)
		if label == "" {
			continue
		}
		code := buildCompositeCodeKey(sg.SubGroupKeys, columnGroupFields)
		mapKey := fmt.Sprintf("%s|%s", label, code)
		uniqueValues[mapKey] = columnGroupValue{
			Label: label,
			Code:  code,
		}
	}

	// Sort keys to ensure consistent column order by label
	sortedKeys := make([]string, 0, len(uniqueValues))
	for key := range uniqueValues {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return uniqueValues[sortedKeys[i]].Label < uniqueValues[sortedKeys[j]].Label
	})

	for _, key := range sortedKeys {
		value := uniqueValues[key]
		groupIdentifier := sanitizeIdentifier(value.Code, value.Label)

		columnGroup := ColumnDef{
			HeaderName:    value.Label,
			GroupID:       fmt.Sprintf("group_%s", groupIdentifier),
			OpenByDefault: boolPtr(true),
			Children:      []ColumnDef{},
		}

		for _, colConfig := range pattern.Columns {
			childCol := ColumnDef{
				Field:        fmt.Sprintf("%s_%s", groupIdentifier, colConfig.Field),
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
		levelKeys := make([]string, len(columnLevels))
		for i, level := range columnLevels {
			label := getValueNameByGroupCode(sg.SubGroupKeys, level.Field)
			code := getValueCodeByGroupCode(sg.SubGroupKeys, level.Field)
			levelKeys[i] = composeHierarchyKey(code, label)
		}

		current := hierarchy
		for i, key := range levelKeys {
			if key == "" {
				continue
			}
			if i == len(levelKeys)-1 {
				if current[key] == nil {
					current[key] = true
				}
			} else {
				if current[key] == nil {
					current[key] = make(map[string]interface{})
				}
				current = current[key].(map[string]interface{})
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
	labelPath []string,
	codePath []string,
) []ColumnDef {
	columns := []ColumnDef{}

	// Get sorted keys for current level
	keys := make([]string, 0, len(hierarchy))
	for key := range hierarchy {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, encodedKey := range keys {
		value := hierarchy[encodedKey]
		code, label := splitHierarchyKey(encodedKey)

		currentLabelPath := append([]string{}, labelPath...)
		currentCodePath := append([]string{}, codePath...)
		currentLabelPath = append(currentLabelPath, label)
		currentCodePath = append(currentCodePath, code)

		sanitizedPathParts := make([]string, len(currentCodePath))
		for i := range currentCodePath {
			sanitizedPathParts[i] = sanitizeIdentifier(currentCodePath[i], currentLabelPath[i])
		}

		groupID := fmt.Sprintf("group_l%d_%s", levelIndex+1, strings.Join(sanitizedPathParts, "_"))

		if levelIndex == len(pattern.ColumnLevels)-1 {
			columnGroup := ColumnDef{
				HeaderName:    label,
				GroupID:       groupID,
				OpenByDefault: boolPtr(true),
				Children:      []ColumnDef{},
			}

			fieldPrefix := strings.Join(sanitizedPathParts, "_")

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
			if nestedMap, ok := value.(map[string]interface{}); ok {
				columnGroup := ColumnDef{
					HeaderName:    label,
					GroupID:       groupID,
					OpenByDefault: boolPtr(true),
					Children:      buildColumnGroupsRecursive(nestedMap, pattern, levelIndex+1, currentLabelPath, currentCodePath),
				}
				columns = append(columns, columnGroup)
			}
		}
	}

	return columns
}

func buildMultiLevelColumns(pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse) []ColumnDef {
	if len(pattern.ColumnLevels) == 0 {
		return []ColumnDef{}
	}

	// Build hierarchy dynamically based on number of columnLevels
	hierarchy := buildHierarchyMap(subGroups, pattern.ColumnLevels)

	// Build columns recursively
	columns := buildColumnGroupsRecursive(hierarchy, pattern, 0, []string{}, []string{})

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

		itemValue := buildItemValue(pattern, sg)
		var columnLabel string
		var columnKey string
		if len(pattern.ColumnLevels) > 0 {
			labelParts := []string{}
			keyParts := []string{}
			for _, level := range pattern.ColumnLevels {
				valueName := getValueNameByGroupCode(sg.SubGroupKeys, level.Field)
				valueCode := getValueCodeByGroupCode(sg.SubGroupKeys, level.Field)
				labelParts = append(labelParts, valueName)
				keyParts = append(keyParts, sanitizeIdentifier(valueCode, valueName))
			}
			columnLabel = strings.Join(labelParts, " | ")
			columnKey = strings.Join(keyParts, "_")
		} else {
			columnLabel = buildCompositeKey(sg.SubGroupKeys, columnGroupFields)
			columnCode := buildCompositeCodeKey(sg.SubGroupKeys, columnGroupFields)
			columnKey = sanitizeIdentifier(columnCode, columnLabel)
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

			if itemValue != "" {
				row["item"] = itemValue
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
		var lineBundleValue *float64
		var stockValue interface{}
		var stockQuantityValue interface{}
		var batchNoValue interface{}
		var warehouseValue interface{}
		var codeValue interface{}
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
				if sq, ok := udfData["stock_quantity"]; ok {
					stockQuantityValue = sq
				}
				if bn, ok := udfData["batch_no"]; ok {
					batchNoValue = bn
				}
				if wh, ok := udfData["warehouse"]; ok {
					warehouseValue = wh
				}
				if code, ok := udfData["code"]; ok {
					codeValue = code
				}
				for key, value := range udfData {
					if key == "is_highlight" || key == "inactive" || key == "stock_quantity" || key == "batch_no" || key == "warehouse" || key == "code" {
						continue
					}

					if key == "stock" {
						stockValue = value
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

		row["is_highlight"] = isHighlightValue
		row["weight_spec"] = sg.PriceWeight
		row["avg_kg_stock"] = sg.TotalNetPriceWeight

		for _, colConfig := range pattern.Columns {
			fieldName := fmt.Sprintf("%s_%s", columnKey, colConfig.Field)

			switch colConfig.DataMapping {
			case "product_group_3":
				row[fieldName] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP3")
			case "product_group_4":
				row[fieldName] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP4")
			case "product_group_8":
				row[fieldName] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP8")
			case "product_group_7":
				row[fieldName] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP7")
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
			case "stock":
				row[fieldName] = stockValue
			case "line_bundle":
				if lineBundleValue != nil {
					row[fieldName] = *lineBundleValue
				} else {
					row[fieldName] = nil
				}
			case "stock_quantity":
				row[fieldName] = stockQuantityValue
			case "batch_no":
				row[fieldName] = batchNoValue
			case "warehouse":
				row[fieldName] = warehouseValue
			case "code":
				row[fieldName] = codeValue
			case "":
				// Empty dataMapping - set default values for calculated/empty fields
				if colConfig.Field == "weight_spec" || colConfig.Field == "avg_kg_stock" {
					row[fieldName] = 0.0
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

func buildItemValue(pattern *PatternConfig, sg models.PriceListSubGroupResponse) string {
	format := pattern.ItemFormat
	if len(format) == 0 {
		format = defaultItemFormat
	}

	var builder strings.Builder
	for _, part := range format {
		switch strings.ToLower(part.Type) {
		case "group":
			if part.Value == "" {
				continue
			}
			if groupVal := getValueNameByGroupCode(sg.SubGroupKeys, part.Value); groupVal != "" {
				builder.WriteString(groupVal)
			}
		case "literal":
			builder.WriteString(part.Value)
		default:
			continue
		}
	}

	return strings.TrimSpace(builder.String())
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
		var deliveryDateValue interface{}
		var tonValue interface{}
		var producerValue interface{}
		var stockQuantityValue interface{}
		var batchNoValue interface{}
		var warehouseValue interface{}
		var codeValue interface{}
		var bkkValue interface{}
		var factoryValue interface{}
		var countryValue interface{}
		var shipNoValue interface{}
		var tsmValue interface{}
		var instituteValue interface{}
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
				if deliveryDate, ok := udfData["delivery_date"]; ok {
					deliveryDateValue = deliveryDate
				}
				if ton, ok := udfData["ton"]; ok {
					tonValue = ton
				}
				if producer, ok := udfData["producer"]; ok {
					producerValue = producer
				}
				if sq, ok := udfData["stock_quantity"]; ok {
					stockQuantityValue = sq
				}
				if bn, ok := udfData["batch_no"]; ok {
					batchNoValue = bn
				}
				if wh, ok := udfData["warehouse"]; ok {
					warehouseValue = wh
				}
				if code, ok := udfData["code"]; ok {
					codeValue = code
				}
				if bkk, ok := udfData["bkk"]; ok {
					bkkValue = bkk
				} else if bn, ok := udfData["batch_no"]; ok {
					bkkValue = bn
				}
				if factory, ok := udfData["factory"]; ok {
					factoryValue = factory
				} else if producer, ok := udfData["producer"]; ok {
					factoryValue = producer
				}
				if country, ok := udfData["country"]; ok {
					countryValue = country
				}
				if shipNo, ok := udfData["ship_no"]; ok {
					shipNoValue = shipNo
				}
				if tsm, ok := udfData["tsm"]; ok {
					tsmValue = tsm
				}
				if institute, ok := udfData["institute"]; ok {
					instituteValue = institute
				}
				if apMap, ok := udfData["awaiting_production"].(map[string]interface{}); ok {
					if importDate, ok := apMap["import_date"]; ok {
						importDateValue = importDate
					}
					if deliveryDate, ok := apMap["delivery_date"]; ok {
						deliveryDateValue = deliveryDate
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
		row["item"] = buildItemValue(pattern, sg)
		for _, fixedCol := range pattern.FixedColumns {
			switch fixedCol.DataMapping {
			case "item":
			case "type":
				row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP9")
			case "product_group_5":
				row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP5")
			case "product_group_2":
				row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP2")
			case "product_group_3":
				row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP3")
			case "product_group_4":
				row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP4")
			case "product_group_6":
				row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP6")
			case "product_group_7":
				row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP7")
			case "product_group_8":
				row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP8")
			case "product_group_9":
				row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP9")
			case "ship_no":
				row[fixedCol.Field] = shipNoValue
			case "price_weight":
				row[fixedCol.Field] = sg.PriceWeight
			case "before_price_weight":
				row[fixedCol.Field] = sg.BeforePriceWeight
			case "extra_price_weight":
				row[fixedCol.Field] = sg.ExtraPriceWeight
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
			case "product_group_6_x_product_group_7":
				// Composite field: PRODUCT_GROUP6 " x " PRODUCT_GROUP7
				pg6 := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP6")
				pg7 := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP7")
				row[fixedCol.Field] = fmt.Sprintf("%s x %s", pg6, pg7)
			case "product_group_4_x_product_group_3":
				// Composite field: PRODUCT_GROUP4 " x " PRODUCT_GROUP3
				pg4 := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP4")
				pg3 := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP3")
				row[fixedCol.Field] = fmt.Sprintf("%s x %s", pg4, pg3)
			case "product_group_5_product_group_3":
				// Composite field: PRODUCT_GROUP5 + PRODUCT_GROUP3 (no separator)
				pg5 := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP5")
				pg3 := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP3")
				row[fixedCol.Field] = pg5 + pg3
			}

			if fixedCol.DataMapping == "" {
				switch fixedCol.Field {
				case "weight_spec":
					row[fixedCol.Field] = 0.0
				case "avg_kg_stock":
					row[fixedCol.Field] = 0.0
				case "product_group_6":
					row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP6")
				default:
					// For other fields without dataMapping, try to infer from field name
					if strings.HasPrefix(fixedCol.Field, "product_group_") {
						// Convert "product_group_6" to "PRODUCT_GROUP6"
						parts := strings.Split(fixedCol.Field, "_")
						if len(parts) >= 3 {
							groupCode := fmt.Sprintf("PRODUCT_GROUP%s", parts[2])
							row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, groupCode)
						} else {
							row[fixedCol.Field] = ""
						}
					} else {
						// Set to empty string if no mapping found
						row[fixedCol.Field] = ""
					}
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
				case "import_date":
					row[childCol.Field] = importDateValue
				case "delivery_date":
					row[childCol.Field] = deliveryDateValue
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
			case "product_group_4_x_product_group_3":
				pg4 := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP4")
				pg3 := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP3")
				row[colConfig.Field] = fmt.Sprintf("%s x %s", pg4, pg3)
			case "extra_price_unit":
				row[colConfig.Field] = sg.ExtraPriceUnit
			case "product_group_3":
				row[colConfig.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP3")
			case "product_group_5":
				row[colConfig.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP5")
			case "product_group_6":
				row[colConfig.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP6")
			case "product_group_8":
				row[colConfig.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP8")
			case "product_group_9":
				row[colConfig.Field] = getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP9")
			case "before_price_weight":
				row[colConfig.Field] = sg.BeforePriceWeight
			case "total_net_price_weight":
				row[colConfig.Field] = sg.TotalNetPriceWeight
			case "price_weight":
				row[colConfig.Field] = sg.PriceWeight
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
			case "delivery_date":
				row[colConfig.Field] = deliveryDateValue
			case "ton":
				row[colConfig.Field] = tonValue
			case "producer":
				row[colConfig.Field] = producerValue
			case "tsm":
				row[colConfig.Field] = tsmValue
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
			case "stock_quantity":
				row[colConfig.Field] = stockQuantityValue
			case "batch_no":
				row[colConfig.Field] = batchNoValue
			case "warehouse":
				row[colConfig.Field] = warehouseValue
			case "code":
				row[colConfig.Field] = codeValue
			case "bkk":
				row[colConfig.Field] = bkkValue
			case "factory":
				row[colConfig.Field] = factoryValue
			case "country":
				row[colConfig.Field] = countryValue
			case "institute":
				row[colConfig.Field] = instituteValue
			case "is_highlight":
				row[colConfig.Field] = isHighlightValue
			case "inactive":
				if hasInactiveValue {
					row[colConfig.Field] = inactiveValue
				} else {
					row[colConfig.Field] = false
				}
			case "":
				// Empty dataMapping - set default values for calculated/empty fields
				if colConfig.Field == "weight_spec" || colConfig.Field == "avg_kg_stock" {
					row[colConfig.Field] = 0.0
				} else {
					// For other fields with empty dataMapping, set to empty string
					row[colConfig.Field] = ""
				}
			}
		}

		row["subgroup_id"] = sg.ID
		row["is_trading"] = sg.IsTrading

		rows = append(rows, row)
	}

	return rows
}

// buildProductGroup2ColumnGroups builds dynamic column groups from PRODUCT_GROUP2 values
// Each column group contains children columns defined in pattern.Columns
func buildProductGroup2ColumnGroups(pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse) []ColumnDef {
	type columnGroupValue struct {
		Label string
		Code  string
	}

	columns := []ColumnDef{}
	uniqueValues := make(map[string]columnGroupValue)

	// Extract unique PRODUCT_GROUP2 values
	for _, sg := range subGroups {
		label := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP2")
		if label == "" {
			continue
		}
		code := getValueCodeByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP2")
		mapKey := fmt.Sprintf("%s|%s", label, code)
		uniqueValues[mapKey] = columnGroupValue{
			Label: label,
			Code:  code,
		}
	}

	// Sort keys to ensure consistent column order by label
	sortedKeys := make([]string, 0, len(uniqueValues))
	for key := range uniqueValues {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return uniqueValues[sortedKeys[i]].Label < uniqueValues[sortedKeys[j]].Label
	})

	// Build column groups with children from pattern.Columns
	for _, key := range sortedKeys {
		value := uniqueValues[key]
		groupIdentifier := sanitizeIdentifier(value.Code, value.Label)

		columnGroup := ColumnDef{
			HeaderName:    value.Label,
			GroupID:       fmt.Sprintf("group_%s", groupIdentifier),
			OpenByDefault: boolPtr(true),
			Children:      []ColumnDef{},
		}

		// Add children columns from pattern.Columns
		for _, colConfig := range pattern.Columns {
			childCol := ColumnDef{
				Field:        fmt.Sprintf("%s_%s", groupIdentifier, colConfig.Field),
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

// buildDirectRowsWithProductGroup2 builds rows with fixed columns and dynamic PRODUCT_GROUP2 column group data
func buildDirectRowsWithProductGroup2(pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse) []AGGridRowData {
	if len(subGroups) == 0 {
		return nil
	}

	// Collect PRODUCT_GROUP2 metadata for consistent column ordering/defaults
	type pg2Entry struct {
		Code  string
		Label string
	}

	pg2Map := make(map[string]string) // code -> label
	for _, sg := range subGroups {
		code := getValueCodeByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP2")
		label := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP2")
		if code != "" {
			pg2Map[code] = label
		}
	}

	pg2Entries := make([]pg2Entry, 0, len(pg2Map))
	for code, label := range pg2Map {
		pg2Entries = append(pg2Entries, pg2Entry{Code: code, Label: label})
	}
	sort.Slice(pg2Entries, func(i, j int) bool {
		return pg2Entries[i].Label < pg2Entries[j].Label
	})

	rows := []AGGridRowData{}
	rowMap := make(map[string]AGGridRowData)
	rowOrder := []string{}

	for _, sg := range subGroups {
		thickness := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP6")
		length := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP7")
		thicknessLength := strings.TrimSpace(fmt.Sprintf("%s x %s", thickness, length))

		sizePart1 := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP5")
		sizePart2 := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP3")
		size := strings.TrimSpace(fmt.Sprintf("%s%s", sizePart1, sizePart2))

		rowKey := fmt.Sprintf("%s|%s", thicknessLength, size)
		if rowKey == "|" {
			rowKey = sg.ID
		}

		row, exists := rowMap[rowKey]
		if !exists {
			row = AGGridRowData{
				"id":                 uuid.New().String(),
				"thickness_x_length": thicknessLength,
				"size":               size,
			}
			if itemValue := buildItemValue(pattern, sg); itemValue != "" {
				row["item"] = itemValue
			}

			// Initialize default values for every PRODUCT_GROUP2 column group
			for _, entry := range pg2Entries {
				identifier := sanitizeIdentifier(entry.Code, entry.Label)
				row[fmt.Sprintf("%s_subgroup_id", identifier)] = ""
				for _, colConfig := range pattern.Columns {
					fieldName := fmt.Sprintf("%s_%s", identifier, colConfig.Field)
					row[fieldName] = defaultValueForProductGroup2Column(colConfig)
				}
			}

			rowMap[rowKey] = row
			rowOrder = append(rowOrder, rowKey)
		}

		// Update base descriptors if new data is available
		row["thickness_x_length"] = thicknessLength
		row["size"] = size

		// Parse UDF data for highlight flag
		isHighlightValue := false
		if len(sg.UdfJson) > 0 {
			udfData := make(map[string]interface{})
			if err := json.Unmarshal(sg.UdfJson, &udfData); err == nil {
				if h, ok := udfData["is_highlight"].(bool); ok {
					isHighlightValue = h
				}
			}
		}

		pg2Code := getValueCodeByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP2")
		pg2Label := getValueNameByGroupCode(sg.SubGroupKeys, "PRODUCT_GROUP2")
		if pg2Code == "" {
			continue
		}
		identifier := sanitizeIdentifier(pg2Code, pg2Label)

		// Set subgroup identifier for the specific PRODUCT_GROUP2 entry
		row[fmt.Sprintf("%s_subgroup_id", identifier)] = sg.ID

		// Populate dynamic column values for this PRODUCT_GROUP2
		for _, colConfig := range pattern.Columns {
			fieldName := fmt.Sprintf("%s_%s", identifier, colConfig.Field)
			switch colConfig.DataMapping {
			case "is_highlight":
				row[fieldName] = isHighlightValue
			case "price_weight":
				row[fieldName] = sg.PriceWeight
			case "total_net_price_weight":
				row[fieldName] = sg.TotalNetPriceWeight
			case "before_price_unit":
				row[fieldName] = sg.BeforePriceUnit
			case "total_net_price_unit":
				row[fieldName] = sg.TotalNetPriceUnit
			case "extra_price_unit":
				row[fieldName] = sg.ExtraPriceUnit
			default:
				row[fieldName] = nil
			}
		}
	}

	// Convert map to ordered slice
	for _, key := range rowOrder {
		rows = append(rows, rowMap[key])
	}

	return rows
}

func defaultValueForProductGroup2Column(colConfig ColumnConfigItem) interface{} {
	switch colConfig.DataMapping {
	case "is_highlight":
		return false
	case "price_weight", "total_net_price_weight", "before_price_unit", "total_net_price_unit", "extra_price_unit":
		return 0.0
	default:
		return nil
	}
}

func buildSummaryRows(pattern *PatternConfig, rows []AGGridRowData) []SummaryRow {
	if pattern == nil || pattern.Summary == nil || len(rows) == 0 {
		return nil
	}

	cfg := pattern.Summary
	if cfg.RowGroupField == "" || len(cfg.Columns) == 0 {
		return nil
	}

	summaryMap := make(map[string]*SummaryRow)
	order := make([]string, 0)

	for _, row := range rows {
		rowGroupValue := getRowGroupValue(row, cfg)
		if rowGroupValue == "" {
			continue
		}

		summaryRow, exists := summaryMap[rowGroupValue]
		if !exists {
			// Initialize data map with all aggregated fields set to 0
			data := make(map[string]interface{})

			// Initialize all summary column fields to 0
			for _, column := range cfg.Columns {
				if column.Field == "" {
					continue
				}

				applyToColumnGroups := true
				if column.ApplyToColumnGroups != nil {
					applyToColumnGroups = *column.ApplyToColumnGroups
				}

				fieldName := column.Field
				if applyToColumnGroups {
					// For column groups, we'll initialize when we encounter the first row with that column group
					// For now, we'll handle it in the aggregation loop
					continue
				} else {
					// Initialize non-column-group fields to 0
					data[fieldName] = float64(0)
				}
			}

			summaryRow = &SummaryRow{
				RowGroupValue: rowGroupValue,
				Data:          data,
			}
			summaryMap[rowGroupValue] = summaryRow
			order = append(order, rowGroupValue)
		}

		for _, column := range cfg.Columns {
			fieldName := column.Field
			if fieldName == "" {
				continue
			}

			applyToColumnGroups := true
			if column.ApplyToColumnGroups != nil {
				applyToColumnGroups = *column.ApplyToColumnGroups
			}

			if applyToColumnGroups {
				columnKey := fmt.Sprintf("%v", row["column_group_key"])
				if columnKey == "" {
					continue
				}
				fieldName = fmt.Sprintf("%s_%s", columnKey, column.Field)
			}

			aggregateSummaryValue(summaryRow.Data, fieldName, row[fieldName], column.Aggregation)
		}
	}

	if len(summaryMap) == 0 {
		return nil
	}

	// Ensure all summary fields are present in data, initializing to 0 if missing
	for _, summaryRow := range summaryMap {
		for _, column := range cfg.Columns {
			if column.Field == "" {
				continue
			}

			applyToColumnGroups := true
			if column.ApplyToColumnGroups != nil {
				applyToColumnGroups = *column.ApplyToColumnGroups
			}

			if !applyToColumnGroups {
				// For non-column-group fields, ensure they exist in data
				if _, exists := summaryRow.Data[column.Field]; !exists {
					summaryRow.Data[column.Field] = float64(0)
				}
			}
		}
	}

	result := make([]SummaryRow, 0, len(summaryMap))
	for _, key := range order {
		if sr, ok := summaryMap[key]; ok {
			result = append(result, *sr)
		}
	}

	return result
}

func buildSummaryField(summaryRows []SummaryRow) map[string]interface{} {
	if len(summaryRows) == 0 {
		return nil
	}

	summaryField := make(map[string]interface{})

	// Iterate through all summaryRows
	for _, summaryRow := range summaryRows {
		if summaryRow.Data == nil {
			continue
		}

		// For each field in the summaryRow's Data, sum the numeric values
		for fieldName, fieldValue := range summaryRow.Data {
			// Try to convert to float64
			value, ok := toFloat64(fieldValue)
			if !ok {
				// Skip non-numeric values
				continue
			}

			// Get current value in summaryField (default to 0 if not present)
			current, _ := toFloat64(summaryField[fieldName])
			summaryField[fieldName] = current + value
		}
	}

	// Return nil if no numeric fields were found
	if len(summaryField) == 0 {
		return nil
	}

	return summaryField
}

func getRowGroupValue(row AGGridRowData, cfg *SummaryConfig) string {
	if row == nil || cfg == nil {
		return ""
	}

	if value, ok := row["row_group_value"]; ok {
		if str := fmt.Sprintf("%v", value); str != "" {
			return str
		}
	}

	if cfg.RowGroupField != "" {
		if value, ok := row[cfg.RowGroupField]; ok {
			return fmt.Sprintf("%v", value)
		}
	}

	return ""
}

func aggregateSummaryValue(target map[string]interface{}, field string, raw interface{}, aggregation string) {
	if target == nil || field == "" {
		return
	}

	value, ok := toFloat64(raw)
	if !ok {
		return
	}

	current, _ := toFloat64(target[field])
	switch strings.ToLower(aggregation) {
	case "sum", "":
		target[field] = current + value
	}
}

func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case nil:
		return 0, false
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint64:
		return float64(v), true
	case uint32:
		return float64(v), true
	case string:
		if v == "" {
			return 0, false
		}
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed, true
		}
	}
	return 0, false
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
