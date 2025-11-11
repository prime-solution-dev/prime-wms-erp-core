package priceService

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	groupService "prime-erp-core/internal/services/group-service"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// Configuration Structures for config.json
// ============================================================================

// PatternConfig represents the configuration for a pricing table pattern
type PatternConfig struct {
	ID                   string             `json:"id"`
	Name                 string             `json:"name"`
	Description          string             `json:"description"`
	Enabled              bool               `json:"enabled"`
	Grouping             GroupingConfig     `json:"grouping"`
	ColumnLevels         []ColumnLevel      `json:"columnLevels,omitempty"`
	Columns              []ColumnConfigItem `json:"columns"`
	FixedColumns         []ColumnConfigItem `json:"fixedColumns"`
	ApplicableCategories []string           `json:"applicableCategories"`
	EditableSuffixes     []string           `json:"editable_suffixes,omitempty"`
	FetchableSuffixes    []string           `json:"fetchable_suffixes,omitempty"`
}

// GroupingConfig defines how data should be grouped
type GroupingConfig struct {
	Tabs         string `json:"tabs"`
	Rows         string `json:"rows"`
	ColumnGroups string `json:"columnGroups"`
}

// ColumnLevel for multi-level nested columns
type ColumnLevel struct {
	Level    int      `json:"level"`
	Field    string   `json:"field"`
	Examples []string `json:"examples"`
}

// ColumnConfigItem defines a column configuration from config.json
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

// TableConfigSettings from config.json
type TableConfigSettings struct {
	GroupHeaderHeight int               `json:"groupHeaderHeight"`
	HeaderHeight      int               `json:"headerHeight"`
	Pagination        bool              `json:"pagination"`
	Toolbar           ToolbarConfig     `json:"toolbar"`
	GridOptions       GridOptionsConfig `json:"gridOptions"`
}

// ToolbarConfig from config.json
type ToolbarConfig struct {
	Show             bool `json:"show"`
	ShowSearch       bool `json:"showSearch"`
	ShowRefresh      bool `json:"showRefresh"`
	ShowColumnToggle bool `json:"showColumnToggle"`
}

// GridOptionsConfig from config.json
type GridOptionsConfig struct {
	SuppressMovableColumns bool `json:"suppressMovableColumns"`
	SuppressMenuHide       bool `json:"suppressMenuHide"`
}

// PriceTableConfiguration represents the full config.json structure
type PriceTableConfiguration struct {
	Patterns       []PatternConfig     `json:"patterns"`
	DefaultPattern string              `json:"defaultPattern"`
	TableConfig    TableConfigSettings `json:"tableConfig"`
}

// ============================================================================
// API Response Structures
// ============================================================================

// PriceListDetailApiResponse represents the main API response structure
type PriceListDetailApiResponse struct {
	Id   uuid.UUID                  `json:"id"`
	Name string                     `json:"name"`
	Tabs []PriceListDetailTabConfig `json:"tabs"`
}

// GetPriceTableRequest represents the request structure
type GetPriceTableRequest struct {
	CompanyCode       string     `json:"company_code"`
	SiteCodes         []string   `json:"site_codes"`
	GroupCodes        []string   `json:"group_codes"`
	EffectiveDateFrom *time.Time `json:"effective_date_from"`
	EffectiveDateTo   *time.Time `json:"effective_date_to"`
}

// PriceListDetailTabConfig represents a tab configuration with table config and data
type PriceListDetailTabConfig struct {
	ID               uuid.UUID                `json:"id"`
	Label            string                   `json:"label"`
	TableConfig      TableConfig              `json:"tableConfig"`
	TableData        []map[string]interface{} `json:"tableData"`
	EditableSuffixes []string                 `json:"editable_suffixes,omitempty"`
	FetchableSuffixes []string                 `json:"fetchable_suffixes,omitempty"`
}

// TableConfig contains the table configuration including columns, toolbar, and options
type TableConfig struct {
	Title             string       `json:"title,omitempty"`
	Toolbar           *Toolbar     `json:"toolbar,omitempty"`
	Pagination        *bool        `json:"pagination,omitempty"`
	GroupHeaderHeight *int         `json:"groupHeaderHeight,omitempty"`
	HeaderHeight      *int         `json:"headerHeight,omitempty"`
	Columns           []ColumnDef  `json:"columns"`
	GridOptions       *GridOptions `json:"gridOptions,omitempty"`
}

// Toolbar represents the toolbar configuration
type Toolbar struct {
	Show             *bool `json:"show,omitempty"`
	ShowSearch       *bool `json:"showSearch,omitempty"`
	ShowRefresh      *bool `json:"showRefresh,omitempty"`
	ShowColumnToggle *bool `json:"showColumnToggle,omitempty"`
}

// GridOptions represents additional grid options
type GridOptions struct {
	SuppressMovableColumns *bool `json:"suppressMovableColumns,omitempty"`
	SuppressMenuHide       *bool `json:"suppressMenuHide,omitempty"`
}

// ColumnDef represents a column definition (can be either a regular column or column group)
type ColumnDef struct {
	// Common fields
	Field           string     `json:"field,omitempty"`
	HeaderName      string     `json:"headerName,omitempty"`
	Width           *int       `json:"width,omitempty"`
	Pinned          string     `json:"pinned,omitempty"`
	LockPosition    *bool      `json:"lockPosition,omitempty"`
	SuppressMovable *bool      `json:"suppressMovable,omitempty"`
	ValueGetter     string     `json:"valueGetter,omitempty"`
	Filter          string     `json:"filter,omitempty"`
	CellRenderer    string     `json:"cellRenderer,omitempty"`
	CellStyle       *CellStyle `json:"cellStyle,omitempty"`
	HeaderClass     string     `json:"headerClass,omitempty"`
	EnableTooltip   *bool      `json:"enableTooltip,omitempty"`

	// Column group specific fields
	GroupID       string      `json:"groupId,omitempty"`
	OpenByDefault *bool       `json:"openByDefault,omitempty"`
	Children      []ColumnDef `json:"children,omitempty"`
}

// CellStyle represents the cell style configuration
type CellStyle struct {
	TextAlign       string `json:"textAlign,omitempty"`
	FontWeight      string `json:"fontWeight,omitempty"`
	FontSize        string `json:"fontSize,omitempty"`
	BackgroundColor string `json:"backgroundColor,omitempty"`
}

// GroupedData represents data grouped by PRODUCT_GROUP2
type GroupedData struct {
	ProductGroup2 string         `json:"product_group_2"`
	PatternGroups []PatternGroup `json:"pattern_groups"`
}

// PatternGroup represents data grouped by pattern: PRODUCT_GROUP6|PRODUCT_GROUP3|PRODUCT_GROUP5
type PatternGroup struct {
	Pattern       string     `json:"pattern"`
	ProductGroup6 string     `json:"product_group_6"`
	ProductGroup3 string     `json:"product_group_3"`
	ProductGroup5 string     `json:"product_group_5"`
	ProductGroup8 string     `json:"product_group_8"`
	SubGroups     []SubGroup `json:"sub_groups"`
}

// AGGridRowData represents a single row in AG Grid format
// Each row represents one PRODUCT_GROUP6 value with dynamic columns for each PRODUCT_GROUP5
type AGGridRowData map[string]interface{}

// CellData represents the data for a single cell (combination of G6, G3, G5)
type CellData struct {
	ProductGroup3  string    `json:"product_group_3"` // Grade
	ProductGroup5  string    `json:"product_group_5"` // Size (4x8, 5x10, etc.)
	ProductGroup6  string    `json:"product_group_6"` // Thickness
	SubGroupID     uuid.UUID `json:"subgroup_id"`
	PriceUnit      float64   `json:"price_unit"`       // ราคาขาย before
	PriceUnitAfter float64   `json:"price_unit_after"` // ราคาขาย After
	PriceWeight    float64   `json:"price_weight"`     // น.น. (weight)
	ExtraPrice     float64   `json:"extra_price"`      // Extra (THB)
	IsTrading      bool      `json:"is_trading"`
	IsHighlight    bool      `json:"is_highlight"` // Highlight สีฟ้า
	Inactive       bool      `json:"inactive"`
	Remark         string    `json:"remark"` // หมายเหตุ
	SubGroupKey    string    `json:"subgroup_key"`
}

// ============================================================================
// Helper Functions
// ============================================================================

// loadConfiguration loads and parses the pattern configuration file based on GroupKey
// If GroupKey is empty or pattern file doesn't exist, falls back to GROUP_1_ITEM_1_PATTERN.json
func loadConfiguration(groupKey string) (*PriceTableConfiguration, error) {
	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Default to GROUP_1_ITEM_1 if groupKey is empty
	if groupKey == "" {
		groupKey = "GROUP_1_ITEM_1"
	}

	// Construct path to pattern file: {GroupKey}_PATTERN.json
	patternFileName := fmt.Sprintf("%s_PATTERN.json", groupKey)
	configPath := filepath.Join(currentDir, "internal", "services", "price-service", "patterns", patternFileName)

	// Try to read the pattern file
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Fallback to GROUP_1_ITEM_1_PATTERN.json if pattern file doesn't exist
		if groupKey != "GROUP_1_ITEM_1" {
			fallbackPath := filepath.Join(currentDir, "internal", "services", "price-service", "GROUP_1_ITEM_1_PATTERN.json")
			data, err = os.ReadFile(fallbackPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read pattern file %s and fallback GROUP_1_ITEM_1_PATTERN.json: %w", patternFileName, err)
			}
		} else {
			return nil, fmt.Errorf("failed to read pattern file %s: %w", patternFileName, err)
		}
	}

	// Parse JSON data
	var config PriceTableConfiguration
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pattern file %s: %w", patternFileName, err)
	}

	return &config, nil
}

// getValueNameByGroupCode extracts value_name from sub_group_keys by group_code
func getValueNameByGroupCode(subGroupKeys []models.PriceListSubGroupKeyResponse, groupCode string) string {
	for _, sgk := range subGroupKeys {
		if sgk.GroupCode == groupCode {
			return sgk.ValueName
		}
	}
	return ""
}

// getValueCodeByGroupCode extracts value_code from sub_group_keys by group_code
func getValueCodeByGroupCode(subGroupKeys []models.PriceListSubGroupKeyResponse, groupCode string) string {
	for _, sgk := range subGroupKeys {
		if sgk.GroupCode == groupCode {
			return sgk.ValueCode
		}
	}
	return ""
}

// buildCompositeKey builds a composite key from multiple group codes using value_name
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

// extractGroupKey extracts the GroupKey (first part) from subgroup_key
// Example: "GROUP_1_ITEM_1|GROUP_2_ITEM_2|..." returns "GROUP_1_ITEM_1"
func extractGroupKey(subgroupKey string) string {
	if subgroupKey == "" {
		return ""
	}
	parts := strings.Split(subgroupKey, "|")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// selectPatternForCategory finds the appropriate pattern configuration for a PRODUCT_GROUP2 value
func selectPatternForCategory(config *PriceTableConfiguration, productGroup2ValueName string) *PatternConfig {
	// Find pattern by matching applicableCategories
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

	// Return default pattern if no match found
	for _, pattern := range config.Patterns {
		if pattern.ID == config.DefaultPattern {
			return &pattern
		}
	}

	// Return first enabled pattern as fallback
	for _, pattern := range config.Patterns {
		if pattern.Enabled {
			return &pattern
		}
	}

	return nil
}

// Helper function to get group key value by code (legacy support)
func getGroupKeyValue(groupKeys []GroupKey, code string) string {
	for _, gk := range groupKeys {
		if gk.Code == code {
			return gk.Value
		}
	}
	return ""
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}

// sanitizeFieldName converts field names to safe format for AG Grid
// Replaces special characters with underscores
func sanitizeFieldName(name string) string {
	// Replace common special characters
	name = regexp.MustCompile(`[^\w]+`).ReplaceAllString(name, "_")
	// Remove leading/trailing underscores
	name = strings.Trim(name, "_")
	// Convert to lowercase for consistency
	return strings.ToLower(name)
}

// convertGroupCodeToFieldName converts PRODUCT_GROUP codes to field names
// e.g., "PRODUCT_GROUP6" -> "product_group_6"
func convertGroupCodeToFieldName(groupCode string) string {
	// Convert to lowercase
	fieldName := strings.ToLower(groupCode)
	// Insert underscore before the last digit(s)
	// PRODUCT_GROUP6 -> product_group_6
	// PRODUCT_GROUP10 -> product_group_10
	re := regexp.MustCompile(`([a-z]+)(\d+)$`)
	fieldName = re.ReplaceAllString(fieldName, "${1}_${2}")
	return fieldName
}

// ============================================================================
// Dynamic Grouping and Building Functions
// ============================================================================

// groupDataByProductGroup2 groups subgroups by PRODUCT_GROUP2 value_name
func groupDataByProductGroup2(priceListData []models.GetPriceListResponse) map[string][]models.PriceListSubGroupResponse {
	groupedData := make(map[string][]models.PriceListSubGroupResponse)

	for _, priceList := range priceListData {
		for _, subGroup := range priceList.SubGroups {
			// Get PRODUCT_GROUP2 value_name
			productGroup2 := getValueNameByGroupCode(subGroup.SubGroupKeys, "PRODUCT_GROUP2")
			if productGroup2 == "" {
				continue
			}

			// Add subgroup to the appropriate group
			groupedData[productGroup2] = append(groupedData[productGroup2], subGroup)
		}
	}

	return groupedData
}

// groupDataByGroupKeyAndProductGroup2 groups subgroups by GroupKey first, then by PRODUCT_GROUP2
// Uses GroupKey from GetPriceListResponse instead of extracting from subgroup_key
// Returns: map[groupKey]map[productGroup2][]subGroups
func groupDataByGroupKeyAndProductGroup2(priceListData []models.GetPriceListResponse) map[string]map[string][]models.PriceListSubGroupResponse {
	groupedData := make(map[string]map[string][]models.PriceListSubGroupResponse)

	for _, priceList := range priceListData {
		// Use GroupKey from GetPriceListResponse
		groupKey := priceList.GroupKey
		if groupKey == "" {
			// Fallback: extract from first subgroup if GroupKey is not set
			if len(priceList.SubGroups) > 0 {
				groupKey = extractGroupKey(priceList.SubGroups[0].SubgroupKey)
			}
			if groupKey == "" {
				continue
			}
		}

		for _, subGroup := range priceList.SubGroups {
			// Get PRODUCT_GROUP2 value_name
			productGroup2 := getValueNameByGroupCode(subGroup.SubGroupKeys, "PRODUCT_GROUP2")
			if productGroup2 == "" {
				continue
			}

			// Initialize nested map if needed
			if groupedData[groupKey] == nil {
				groupedData[groupKey] = make(map[string][]models.PriceListSubGroupResponse)
			}

			// Add subgroup to the appropriate group
			groupedData[groupKey][productGroup2] = append(groupedData[groupKey][productGroup2], subGroup)
		}
	}

	return groupedData
}

// buildDynamicColumns builds AG Grid columns based on pattern configuration
func buildDynamicColumns(pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse) []ColumnDef {
	columns := []ColumnDef{}

	// Add fixed columns from configuration
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

	// Build dynamic column groups based on pattern
	columnGroupFields := strings.Split(pattern.Grouping.ColumnGroups, "|")

	if len(pattern.ColumnLevels) > 0 {
		// Multi-level nested columns (e.g., G3 > G8 > G5)
		columns = append(columns, buildMultiLevelColumns(pattern, subGroups, columnGroupFields)...)
	} else {
		// Single-level column groups (e.g., G5)
		columns = append(columns, buildSingleLevelColumns(pattern, subGroups, columnGroupFields)...)
	}

	return columns
}

// buildSingleLevelColumns builds single-level column groups
func buildSingleLevelColumns(pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse, columnGroupFields []string) []ColumnDef {
	columns := []ColumnDef{}

	// Collect unique values for column groups
	uniqueValues := make(map[string]bool)
	for _, sg := range subGroups {
		key := buildCompositeKey(sg.SubGroupKeys, columnGroupFields)
		if key != "" {
			uniqueValues[key] = true
		}
	}

	// Build column group for each unique value
	for valueName := range uniqueValues {
		columnGroup := ColumnDef{
			HeaderName:    valueName,
			GroupID:       fmt.Sprintf("group_%s", sanitizeFieldName(valueName)),
			OpenByDefault: boolPtr(true),
			Children:      []ColumnDef{},
		}

		// Add child columns from configuration
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

// buildMultiLevelColumns builds multi-level nested column groups
func buildMultiLevelColumns(pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse, columnGroupFields []string) []ColumnDef {
	// Build hierarchical structure based on columnLevels
	hierarchy := make(map[string]map[string]map[string]bool) // L1 > L2 > L3

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

	// Build nested column groups
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

				// Add child columns from configuration
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

// buildDynamicRows builds AG Grid rows based on pattern configuration
func buildDynamicRows(pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse) []AGGridRowData {
	rowMap := make(map[string]AGGridRowData)

	// Get row identifier fields
	rowFields := strings.Split(pattern.Grouping.Rows, "|")
	columnGroupFields := strings.Split(pattern.Grouping.ColumnGroups, "|")

	for _, sg := range subGroups {
		// Build row key
		rowKey := buildCompositeKey(sg.SubGroupKeys, rowFields)
		if rowKey == "" {
			continue
		}

		// Initialize row if not exists
		if _, exists := rowMap[rowKey]; !exists {
			rowMap[rowKey] = AGGridRowData{
				"id": uuid.New().String(),
			}

			// Set row identifier fields
			for _, field := range rowFields {
				valueName := getValueNameByGroupCode(sg.SubGroupKeys, field)
				fieldName := convertGroupCodeToFieldName(field)
				rowMap[rowKey][fieldName] = valueName
			}
		}

		// Build column group key
		var columnKey string
		if len(pattern.ColumnLevels) > 0 {
			// Multi-level: use sanitized composite key
			parts := []string{}
			for _, level := range pattern.ColumnLevels {
				valueName := getValueNameByGroupCode(sg.SubGroupKeys, level.Field)
				parts = append(parts, sanitizeFieldName(valueName))
			}
			columnKey = strings.Join(parts, "_")
		} else {
			// Single level
			columnKey = sanitizeFieldName(buildCompositeKey(sg.SubGroupKeys, columnGroupFields))
		}

		// Parse udf_json if present
		udfData := make(map[string]interface{})
		isHighlightValue := false
		inactiveValue := false
		hasInactiveValue := false
		if len(sg.UdfJson) > 0 {
			if err := json.Unmarshal(sg.UdfJson, &udfData); err == nil {
				// Extract is_highlight value if present
				if h, ok := udfData["is_highlight"].(bool); ok {
					isHighlightValue = h
				}
				// Extract inactive value if present
				if inactive, ok := udfData["inactive"].(bool); ok {
					inactiveValue = inactive
					hasInactiveValue = true
				}
				// Extract tooltips and merge additional udf_json fields into row data
				for key, value := range udfData {
					if key == "is_highlight" || key == "inactive" {
						continue
					}

					// Check if this is a tooltip field (ends with _tooltip)
					if strings.HasSuffix(key, "_tooltip") {
						// Extract the base field name (e.g., "total_net_price_unit_tooltip" -> "total_net_price_unit")
						baseField := strings.TrimSuffix(key, "_tooltip")
						// Create tooltip object with text and optional icon
						tooltipData := make(map[string]interface{})
						if tooltipMap, ok := value.(map[string]interface{}); ok {
							// If value is already an object, use it directly
							if text, hasText := tooltipMap["text"]; hasText {
								tooltipData["text"] = text
							}
							if icon, hasIcon := tooltipMap["icon"]; hasIcon {
								tooltipData["icon"] = icon
							}
						} else {
							// If value is a string, use it as text
							tooltipData["text"] = value
						}
						// Add tooltip to row data with field name pattern: {columnKey}_{baseField}_tooltip
						rowMap[rowKey][fmt.Sprintf("%s_%s_tooltip", columnKey, sanitizeFieldName(baseField))] = tooltipData
					} else {
						// Merge other udf_json fields into row data (with prefix to avoid conflicts)
						rowMap[rowKey][fmt.Sprintf("%s_%s", columnKey, sanitizeFieldName(key))] = value
					}
				}
			}
		}

		// Populate cell data based on dataMapping configuration
		for _, colConfig := range pattern.Columns {
			fieldName := fmt.Sprintf("%s_%s", columnKey, colConfig.Field)

			// Map data based on dataMapping
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
			case "term_price_unit":
				rowMap[rowKey][fieldName] = sg.TermPriceUnit
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
			case "before_term_price_unit":
				rowMap[rowKey][fieldName] = sg.BeforeTermPriceUnit
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

		// Store metadata
		rowMap[rowKey][fmt.Sprintf("%s_subgroup_id", columnKey)] = sg.ID
		rowMap[rowKey][fmt.Sprintf("%s_is_trading", columnKey)] = sg.IsTrading
	}

	// Convert map to slice
	rows := []AGGridRowData{}
	for _, row := range rowMap {
		rows = append(rows, row)
	}

	return rows
}

// convertCellStyle converts map[string]interface{} to CellStyle struct
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

// getGroupAndItemMappings gets group and group item mappings for value name resolution
func getGroupAndItemMappings(sqlx *sqlx.DB) (map[string]models.GetGroupResponse, map[string]models.GetGroupItemResponse, map[string]GetPaymentTermResponse, error) {
	// Get groups using group service
	groupReq := models.GetGroupRequest{
		GroupCodes: []string{},
		ItemCodes:  []string{},
	}

	groupReqJson, err := json.Marshal(groupReq)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to marshal group request: %w", err)
	}

	groupReqString := string(groupReqJson)

	// Note: We need a gin.Context for this call, but we're in a helper function
	// Let's create a minimal context or use nil if the function supports it
	resp, err := groupService.GetGroup(nil, groupReqString)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get groups: %w", err)
	}

	groupResp, ok := resp.([]models.GetGroupResponse)
	if !ok {
		return nil, nil, nil, fmt.Errorf("failed to cast group response")
	}

	groupMap := map[string]models.GetGroupResponse{}
	groupItemMap := map[string]models.GetGroupItemResponse{}
	for _, g := range groupResp {
		groupMap[g.GroupCode] = g
		for _, item := range g.Items {
			groupItemMap[item.ItemCode] = item
		}
	}

	// Get payment terms
	termReq := GetPaymentTermRequest{
		TermCode: []string{},
		TermType: []string{},
	}

	termReqJson, err := json.Marshal(termReq)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to marshal payment term request: %w", err)
	}

	termReqString := string(termReqJson)

	termResp, err := GetPaymentTerm(nil, termReqString)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get payment terms: %w", err)
	}

	paymentTerms, ok := termResp.([]GetPaymentTermResponse)
	if !ok {
		return nil, nil, nil, fmt.Errorf("failed to cast payment term response")
	}

	paymentTermMap := map[string]GetPaymentTermResponse{}
	for _, pt := range paymentTerms {
		paymentTermMap[pt.TermCode] = pt
	}

	return groupMap, groupItemMap, paymentTermMap, nil
}

// loadPriceData loads price list data from database using GetPriceList
func loadPriceData(sqlx *sqlx.DB, req GetPriceTableRequest) ([]models.GetPriceListResponse, error) {
	// Build GetPriceListGroupRequest from GetPriceTableRequest
	priceListReq := GetPriceListGroupRequest{
		CompanyCode:       req.CompanyCode,
		SiteCodes:         req.SiteCodes,
		GroupCodes:        req.GroupCodes,
		EffectiveDateFrom: req.EffectiveDateFrom,
		EffectiveDateTo:   req.EffectiveDateTo,
	}

	// Get price list groups with subgroups
	groupSubGroup, err := getGroupSubGroup(sqlx, priceListReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get group sub group: %w", err)
	}

	// Get terms
	groupSubGroup, err = getTerms(sqlx, groupSubGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to get terms: %w", err)
	}

	// Get extras
	groupSubGroup, err = getExtras(sqlx, groupSubGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to get extras: %w", err)
	}

	// Transform to GetPriceListResponse format (same as GetPriceList API)
	result, err := transformToGetPriceListResponse(sqlx, groupSubGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to transform response: %w", err)
	}

	return result, nil
}

// transformToGetPriceListResponse transforms internal response to API response format
func transformToGetPriceListResponse(sqlx *sqlx.DB, responses []GetPriceListGroupResponse) ([]models.GetPriceListResponse, error) {
	// Get group and group item mappings
	groupMap, groupItemMap, _, err := getGroupAndItemMappings(sqlx)
	if err != nil {
		return nil, fmt.Errorf("failed to get mappings: %w", err)
	}

	result := []models.GetPriceListResponse{}
	for _, resp := range responses {
		// Extract GroupKey from first subgroup's subgroup_key (first part before "|")
		groupKey := ""
		groupKeyName := ""
		if len(resp.SubGroups) > 0 && resp.SubGroups[0].SubGroupKey != "" {
			groupKey = extractGroupKey(resp.SubGroups[0].SubGroupKey)
			// Get GroupKeyName from group item map using the last part of group_key
			if groupKey != "" {
				// Extract the last part (e.g., "GROUP_1_ITEM_1" -> "GROUP_1_ITEM_1")
				// Actually, groupKey is already the first part, so we can use it directly
				if item, ok := groupItemMap[groupKey]; ok {
					groupKeyName = item.ItemName
				}
			}
		}

		priceListResp := models.GetPriceListResponse{
			ID:                resp.ID.String(),
			CompanyCode:       resp.CompanyCode,
			SiteCode:          resp.SiteCode,
			GroupCode:         resp.GroupCode,
			PriceUnit:         resp.PriceUnit,
			PriceWeight:       resp.PriceWeight,
			BeforePriceUnit:   resp.BeforePriceUnit,
			BeforePriceWeight: resp.BeforePriceWeight,
			Currency:          resp.Currency,
			Remark:            resp.Remark,
			GroupKey:          groupKey,
			GroupKeyName:      groupKeyName,
		}

		// Format effective date
		if !resp.EffectiveDate.IsZero() {
			priceListResp.EffectiveDate = resp.EffectiveDate.Format(time.RFC3339)
		}

		// Transform subgroups
		subGroups := []models.PriceListSubGroupResponse{}
		for _, sg := range resp.SubGroups {
			subGroupKeys := []models.PriceListSubGroupKeyResponse{}
			for _, sgk := range sg.GroupKeys {
				subGroupKeys = append(subGroupKeys, models.PriceListSubGroupKeyResponse{
					ID:         uuid.New().String(),
					SubGroupID: sg.ID.String(),
					GroupCode:  sgk.Code,
					GroupName:  groupMap[sgk.Code].GroupName,
					ValueCode:  sgk.Value,
					ValueName:  groupItemMap[sgk.Value].ItemName,
					Seq:        sgk.Seq,
				})
			}

			sgEffectiveDate := ""
			if !sg.EffectiveDate.IsZero() {
				sgEffectiveDate = sg.EffectiveDate.Format(time.RFC3339)
			}

			subGroups = append(subGroups, models.PriceListSubGroupResponse{
				ID:                        sg.ID.String(),
				PriceListGroupID:          resp.ID.String(),
				SubgroupKey:               sg.SubGroupKey,
				IsTrading:                 sg.IsTrading,
				PriceUnit:                 sg.PriceUnit,
				ExtraPriceUnit:            sg.ExtraPriceUnit,
				TermPriceUnit:             sg.TermPriceUnit,
				TotalNetPriceUnit:         sg.TotalNetPriceUnit,
				PriceWeight:               sg.PriceWeight,
				ExtraPriceWeight:          sg.ExtraPriceWeight,
				TermPriceWeight:           sg.TermPriceWeight,
				TotalNetPriceWeight:       sg.TotalNetPriceWeight,
				BeforePriceUnit:           sg.BeforePriceUnit,
				BeforeExtraPriceUnit:      sg.BeforeExtraPriceUnit,
				BeforeTermPriceUnit:       sg.BeforeTermPriceUnit,
				BeforeTotalNetPriceUnit:   sg.BeforeTotalNetPriceUnit,
				BeforePriceWeight:         sg.BeforePriceWeight,
				BeforeExtraPriceWeight:    sg.BeforeExtraPriceWeight,
				BeforeTermPriceWeight:     sg.BeforeTermPriceWeight,
				BeforeTotalNetPriceWeight: sg.BeforeTotalNetPriceWeight,
				EffectiveDate:             sgEffectiveDate,
				Remark:                    sg.Remark,
				UdfJson:                   sg.UdfJson,
				SubGroupKeys:              subGroupKeys,
			})
		}

		priceListResp.SubGroups = subGroups
		result = append(result, priceListResp)
	}

	return result, nil
}

func GetPriceTable(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	// Parse request
	var req GetPriceTableRequest
	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request JSON: %w", err)
	}

	// Validate required fields
	if req.CompanyCode == "" {
		return nil, fmt.Errorf("company_code is required")
	}
	if len(req.SiteCodes) == 0 {
		return nil, fmt.Errorf("site_codes is required")
	}
	if len(req.GroupCodes) == 0 {
		return nil, fmt.Errorf("group_codes is required")
	}

	// Connect to database
	sqlx, err := db.ConnectSqlx(`prime_erp`)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer sqlx.Close()

	// Load price data
	priceListData, err := loadPriceData(sqlx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to load price data: %w", err)
	}

	if len(priceListData) == 0 {
		return PriceListDetailApiResponse{
			Id:   uuid.New(),
			Name: "Price List Detail",
			Tabs: []PriceListDetailTabConfig{},
		}, nil
	}

	// fmt.Println("\n========== LOADED DATA ==========")
	// fmt.Printf("Total price lists: %d\n", len(priceListData))
	// for i, pl := range priceListData {
	// 	fmt.Printf("Price List %d: %d subgroups\n", i+1, len(pl.SubGroups))
	// }

	// Group data by GroupKey first, then by PRODUCT_GROUP2
	groupedData := groupDataByGroupKeyAndProductGroup2(priceListData)

	// fmt.Println("\n========== GROUPED DATA ==========")
	// fmt.Printf("Total GroupKeys: %d\n", len(groupedData))

	// Build AG Grid response with tabs
	tabs := []PriceListDetailTabConfig{}

	// Iterate over each GroupKey
	for groupKey, productGroup2Map := range groupedData {
		// Load configuration for this GroupKey
		config, err := loadConfiguration(groupKey)
		if err != nil {
			fmt.Printf("Warning: Failed to load configuration for GroupKey '%s': %v, skipping\n", groupKey, err)
			continue
		}

		// fmt.Printf("\n========== Processing GroupKey: %s ==========\n", groupKey)

		// Iterate over PRODUCT_GROUP2 values for this GroupKey
		for productGroup2, subGroups := range productGroup2Map {
			// Select pattern for this category
			pattern := selectPatternForCategory(config, productGroup2)
			if pattern == nil {
				fmt.Printf("Warning: No pattern found for GroupKey '%s' category '%s', skipping\n", groupKey, productGroup2)
				continue
			}

			// fmt.Printf("\nProcessing category: %s with pattern: %s\n", productGroup2, pattern.ID)

			// Build columns and rows using configuration
			columns := buildDynamicColumns(pattern, subGroups)
			rowData := buildDynamicRows(pattern, subGroups)

			// Convert AGGridRowData to []map[string]interface{}
			tableData := make([]map[string]interface{}, len(rowData))
			for i, row := range rowData {
				tableData[i] = map[string]interface{}(row)
			}

			// Create tab configuration
			tab := PriceListDetailTabConfig{
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
				
			}

			tabs = append(tabs, tab)
		}
	}

	// Create final response
	response := PriceListDetailApiResponse{
		Id:   uuid.MustParse(priceListData[0].ID),
		Name: "Price List Detail",
		Tabs: tabs,
	}

	// Print JSON output
	// fmt.Println("\n========== JSON OUTPUT ==========")
	// utils.PrintJSON(response)

	return response, nil
}
