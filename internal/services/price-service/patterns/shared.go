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
	ID                   string               `json:"id"`
	Name                 string               `json:"name"`
	Description          string               `json:"description"`
	Enabled              bool                 `json:"enabled"`
	Summary              *SummaryConfig       `json:"summary,omitempty"`
	Grouping             GroupingConfig       `json:"grouping"`
	ColumnLevels         []ColumnLevel        `json:"columnLevels,omitempty"`
	Columns              []ColumnConfigItem   `json:"columns"`
	FixedColumns         []ColumnConfigItem   `json:"fixedColumns"`
	ColumnGroups         []ColumnGroupConfig  `json:"columnGroups,omitempty"`
	ApplicableCategories []string             `json:"applicableCategories"`
	EditableSuffixes     []string             `json:"editable_suffixes,omitempty"`
	FetchableSuffixes    []string             `json:"fetchable_suffixes,omitempty"`
	ItemFormat           []ItemFormatPart     `json:"itemFormat,omitempty"`
	RequiredGroupCodes   []string             `json:"requiredGroupCodes,omitempty"`
	ValueMappings        *ValueMappingsConfig `json:"valueMappings,omitempty"`
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
	Hide            bool                   `json:"hide,omitempty"`
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
	Patterns       []PatternConfig      `json:"patterns"`
	DefaultPattern string               `json:"defaultPattern"`
	TableConfig    TableConfigSettings  `json:"tableConfig"`
	ValueMappings  *ValueMappingsConfig `json:"valueMappings,omitempty"`
}

// ValueMappingsConfig defines all configurable mappings that were previously hardcoded
// in Go code. All fields are optional; when missing, code falls back to existing defaults
// to preserve backward compatibility.
type ValueMappingsConfig struct {
	// GroupCodeMappings maps semantic keys (e.g. "productGroup2") to actual group codes
	// (e.g. "PRODUCT_GROUP2").
	GroupCodeMappings map[string]string `json:"groupCodeMappings,omitempty"`

	// HandlerMappings maps group codes (e.g. "GROUP_1_ITEM_1") to handler identifiers.
	// The identifiers are resolved via an internal registry; they do not use reflection.
	HandlerMappings map[string]string `json:"handlerMappings,omitempty"`

	// DefaultItemFormat allows overriding the global/default item format.
	DefaultItemFormat []ItemFormatPart `json:"defaultItemFormat,omitempty"`

	// SpecialMappings is a generic bag for one-off mappings, such as mapping "type"
	// to a particular PRODUCT_GROUP code. Prefer GroupCodeMappings when possible.
	SpecialMappings map[string]string `json:"specialMappings,omitempty"`
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
	Hide            *bool       `json:"hide,omitempty"`
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

// legacyDefaultItemFormat is kept for backward compatibility when no configuration
// is provided. New behavior should prefer configuration from ValueMappingsConfig
// or PatternConfig.ItemFormat.
var legacyDefaultItemFormat = []ItemFormatPart{
	{Type: "group", Value: "PRODUCT_GROUP4"},
	{Type: "literal", Value: "x"},
	{Type: "group", Value: "PRODUCT_GROUP6"},
	{Type: "literal", Value: "x"},
	{Type: "group", Value: "PRODUCT_GROUP7"},
}

// LoadConfiguration loads a pattern configuration file for the given groupCode
// Validates the configuration and logs warnings for missing optional configurations.
// Returns the configuration even if validation warnings occur (backward compatibility).
func LoadConfiguration(groupCode string) (*PriceTableConfiguration, error) {
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

	// Validate configuration (non-blocking, for backward compatibility)
	validateConfiguration(&config, groupCode)

	return &config, nil
}

// validateConfiguration validates the configuration and logs warnings for missing
// optional configurations. This is non-blocking to ensure backward compatibility.
func validateConfiguration(config *PriceTableConfiguration, groupCode string) {
	if config == nil {
		return
	}

	// Validate root-level ValueMappings if present
	if config.ValueMappings != nil {
		validateValueMappings(config.ValueMappings, fmt.Sprintf("root-level (groupCode: %s)", groupCode))
	}

	// Validate pattern-level ValueMappings
	for i, pattern := range config.Patterns {
		if pattern.ValueMappings != nil {
			validateValueMappings(pattern.ValueMappings, fmt.Sprintf("pattern '%s' (index: %d)", pattern.ID, i))
		}
	}
}

// validateValueMappings validates ValueMappingsConfig and logs warnings for
// common issues. This is non-blocking to ensure backward compatibility.
func validateValueMappings(vm *ValueMappingsConfig, context string) {
	if vm == nil {
		return
	}

	// Check for empty mappings (not an error, but could indicate misconfiguration)
	if vm.GroupCodeMappings != nil && len(vm.GroupCodeMappings) == 0 {
		fmt.Printf("[WARN] ValueMappingsConfig (%s): GroupCodeMappings is empty\n", context)
	}

	if vm.HandlerMappings != nil && len(vm.HandlerMappings) == 0 {
		fmt.Printf("[WARN] ValueMappingsConfig (%s): HandlerMappings is empty\n", context)
	}

	if vm.SpecialMappings != nil && len(vm.SpecialMappings) == 0 {
		fmt.Printf("[WARN] ValueMappingsConfig (%s): SpecialMappings is empty\n", context)
	}

	// Validate DefaultItemFormat structure if present
	if len(vm.DefaultItemFormat) > 0 {
		for i, part := range vm.DefaultItemFormat {
			if part.Type != "group" && part.Type != "literal" {
				fmt.Printf("[WARN] ValueMappingsConfig (%s): DefaultItemFormat[%d] has invalid type '%s' (expected 'group' or 'literal')\n", context, i, part.Type)
			}
			if part.Value == "" {
				fmt.Printf("[WARN] ValueMappingsConfig (%s): DefaultItemFormat[%d] has empty value\n", context, i)
			}
		}
	}
}

// GetHandlerFunctionName returns the handler function name for a given group code
// from the configuration, falling back to the group code itself if not configured.
// This supports backward compatibility by allowing the caller to use the group code
// as the handler identifier when no mapping is provided.
func GetHandlerFunctionName(config *PriceTableConfiguration, groupCode string) string {
	if config == nil {
		// Backward compatibility: return group code as-is when config is missing
		return groupCode
	}

	vm := config.ValueMappings
	if vm == nil || vm.HandlerMappings == nil {
		// Backward compatibility: return group code as-is when mappings are missing
		return groupCode
	}

	if handlerName, ok := vm.HandlerMappings[groupCode]; ok && handlerName != "" {
		return handlerName
	}

	// Backward compatibility: fall back to group code when mapping is not found
	fmt.Printf("[WARN] ValueMappingsConfig: handler mapping for '%s' not found, using group code as handler identifier\n", groupCode)
	return groupCode
}

// getEffectiveValueMappings returns the ValueMappingsConfig that should be used
// for the given pattern. Pattern-level mappings override root-level mappings.
// If neither is present, nil is returned and callers must fall back to defaults.
func getEffectiveValueMappings(root *PriceTableConfiguration, pattern *PatternConfig) *ValueMappingsConfig {
	if pattern != nil && pattern.ValueMappings != nil {
		return pattern.ValueMappings
	}
	return root.ValueMappings
}

// getGroupCodeByMapping resolves a semantic key (e.g. "productGroup2") to an
// actual group code (e.g. "PRODUCT_GROUP2") using ValueMappingsConfig.
// If there is no configuration for the key, fallbackCode is returned.
// Logs a warning when falling back to default for backward compatibility.
func getGroupCodeByMapping(vm *ValueMappingsConfig, mappingName, fallbackCode string) string {
	if vm == nil || vm.GroupCodeMappings == nil {
		// Backward compatibility: silently fall back to default when config is missing
		return fallbackCode
	}
	if code, ok := vm.GroupCodeMappings[mappingName]; ok && code != "" {
		return code
	}
	// Backward compatibility: fall back to default when mapping is not found
	// Only log if we have a config but the mapping is missing (not when config is nil)
	fmt.Printf("[WARN] ValueMappingsConfig: mapping '%s' not found, falling back to default '%s'\n", mappingName, fallbackCode)
	return fallbackCode
}

// getGroupCodeFromConfig is a convenience function for pattern files to resolve
// a semantic mapping name (e.g. "productGroup2") to an actual group code
// (e.g. "PRODUCT_GROUP2") using the configuration. Falls back to the
// provided fallbackCode if no mapping is configured.
func getGroupCodeFromConfig(config *PriceTableConfiguration, pattern *PatternConfig, mappingName, fallbackCode string) string {
	vm := getEffectiveValueMappings(config, pattern)
	return getGroupCodeByMapping(vm, mappingName, fallbackCode)
}

// getSpecialMapping returns a special mapping value by key, falling back to
// the provided default if not configured.
// Logs a warning when falling back to default for backward compatibility.
func getSpecialMapping(vm *ValueMappingsConfig, key, fallback string) string {
	if vm == nil || vm.SpecialMappings == nil {
		// Backward compatibility: silently fall back to default when config is missing
		return fallback
	}
	if v, ok := vm.SpecialMappings[key]; ok && v != "" {
		return v
	}
	// Backward compatibility: fall back to default when mapping is not found
	// Only log if we have a config but the mapping is missing (not when config is nil)
	fmt.Printf("[WARN] ValueMappingsConfig: special mapping '%s' not found, falling back to default '%s'\n", key, fallback)
	return fallback
}

// getDefaultItemFormat returns the item format to use, preferring:
//  1. PatternConfig.ItemFormat (pattern specific)
//  2. ValueMappings.DefaultItemFormat (root or pattern mappings)
//  3. legacyDefaultItemFormat (hardcoded legacy default)
//
// Logs a warning when falling back to legacy default for backward compatibility.
func getDefaultItemFormat(root *PriceTableConfiguration, pattern *PatternConfig) []ItemFormatPart {
	if pattern != nil && len(pattern.ItemFormat) > 0 {
		return pattern.ItemFormat
	}

	vm := getEffectiveValueMappings(root, pattern)
	if vm != nil && len(vm.DefaultItemFormat) > 0 {
		return vm.DefaultItemFormat
	}

	// Backward compatibility: fall back to legacy default
	// Only log if we have a config but no default item format is configured
	if root != nil {
		fmt.Printf("[WARN] ValueMappingsConfig: DefaultItemFormat not configured, falling back to legacy default\n")
	}
	return legacyDefaultItemFormat
}

// ExtractRequiredGroupCodes extracts all requiredGroupCodes from enabled patterns in the configuration
func ExtractRequiredGroupCodes(config *PriceTableConfiguration) []string {
	if config == nil {
		return []string{}
	}

	groupCodeMap := make(map[string]bool)

	for _, pattern := range config.Patterns {
		if pattern.Enabled && len(pattern.RequiredGroupCodes) > 0 {
			for _, groupCode := range pattern.RequiredGroupCodes {
				groupCodeMap[groupCode] = true
			}
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(groupCodeMap))
	for groupCode := range groupCodeMap {
		result = append(result, groupCode)
	}

	return result
}

// pgCodeRegex matches short group codes like PG02, PG03, PG04, PG07.
var pgCodeRegex = regexp.MustCompile(`^PG\d+$`)

// convertDataMappingToGroupCode converts a dataMapping value to a group code format.
// Handles: "PG02"/"PG04"/"PG07" (return as-is), "product_group_2" -> "PG2", etc.
// Returns empty string if the dataMapping is not a product group reference.
func convertDataMappingToGroupCode(dataMapping string) string {
	if dataMapping == "" {
		return ""
	}

	// If already in PG<number> format (e.g. PG02, PG03, PG04, PG07), return as is
	if pgCodeRegex.MatchString(dataMapping) {
		return dataMapping
	}

	// Convert lowercase format (product_group_2) to PG format (PG2)
	if strings.HasPrefix(dataMapping, "product_group_") {
		parts := strings.Split(dataMapping, "_")
		if len(parts) >= 3 {
			// Extract the number part (e.g., "2" from "product_group_2")
			groupNum := parts[2]
			return fmt.Sprintf("PG%s", groupNum)
		}
	}

	// For composite fields or other mappings, return empty string
	// The caller should handle these separately
	return ""
}

// extractGroupCodesFromCompositeMapping extracts group codes from composite dataMapping values.
// For example, "product_group_6_x_product_group_7" returns ["PRODUCT_GROUP6", "PRODUCT_GROUP7"]
func extractGroupCodesFromCompositeMapping(dataMapping string) []string {
	if dataMapping == "" {
		return nil
	}

	// Handle composite fields like "product_group_6_x_product_group_7" or "product_group_4_x_product_group_3"
	if strings.Contains(dataMapping, "_x_") {
		parts := strings.Split(dataMapping, "_x_")
		if len(parts) == 2 {
			code1 := convertDataMappingToGroupCode(parts[0])
			code2 := convertDataMappingToGroupCode(parts[1])
			if code1 != "" && code2 != "" {
				return []string{code1, code2}
			}
		}
	}

	// Handle composite fields without separator like "product_group_5_product_group_3"
	// This is trickier - we need to find where one group code ends and another begins
	// Pattern: product_group_<num>product_group_<num>
	re := regexp.MustCompile(`product_group_(\d+)`)
	matches := re.FindAllStringSubmatch(dataMapping, -1)
	if len(matches) >= 2 {
		result := make([]string, len(matches))
		for i, match := range matches {
			if len(match) >= 2 {
				result[i] = fmt.Sprintf("PRODUCT_GROUP%s", match[1])
			}
		}
		return result
	}

	return nil
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

// getAvgProductFromInventory extracts AvgProduct from the first InventoryWeight entry
// Returns 0.0 if inventory data is not available
func getAvgProductFromInventory(sg models.PriceListSubGroupResponse) string {
	if len(sg.InventoryWeight) > 0 {
		return fmt.Sprintf("%.2f", sg.InventoryWeight[0].AvgWeight)
	}
	return ""
}

// getWeightSpecFromInventory extracts WeightSpec from the first InventoryWeight entry
// Returns 0.0 if inventory data is not available
func getWeightSpecFromInventory(sg models.PriceListSubGroupResponse) float64 {
	if len(sg.InventoryWeight) > 0 {
		return sg.InventoryWeight[0].TotalWeight
	}
	return 0.0
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
	// Normalize the input value name for flexible matching
	normalizedValueName := strings.ToLower(strings.TrimSpace(productGroup2ValueName))

	for _, pattern := range config.Patterns {
		if !pattern.Enabled {
			continue
		}
		for _, category := range pattern.ApplicableCategories {
			// Normalize category name for comparison
			normalizedCategory := strings.ToLower(strings.TrimSpace(category))

			// Try exact match first (after normalization)
			if normalizedCategory == normalizedValueName {
				return &pattern
			}

			// Try partial match (contains check for flexibility)
			if strings.Contains(normalizedValueName, normalizedCategory) || strings.Contains(normalizedCategory, normalizedValueName) {
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

	// Cache resolved PRODUCT_GROUP2 codes per groupKey to avoid re-loading configuration
	productGroup2CodeCache := make(map[string]string)

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

		// Resolve the PRODUCT_GROUP2 code for this groupKey using configuration when available.
		productGroup2Code, ok := productGroup2CodeCache[groupKey]
		if !ok {
			productGroup2Code = "PG02"
			if config, err := LoadConfiguration(groupKey); err == nil {
				// Use root-level mappings; there is no specific pattern context at this stage.
				productGroup2Code = getGroupCodeFromConfig(config, nil, "productGroup2", "PRODUCT_GROUP2")
			}
			productGroup2CodeCache[groupKey] = productGroup2Code
		}

		for _, subGroup := range priceList.SubGroups {
			productGroup2 := getValueNameByGroupCode(subGroup.SubGroupKeys, productGroup2Code)
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
			Hide:            boolPtr(fixedCol.Hide),
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
				Hide:         boolPtr(colConfig.Hide),
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
		hasAnyValue := false

		for i, level := range columnLevels {
			label := getValueNameByGroupCode(sg.SubGroupKeys, level.Field)
			code := getValueCodeByGroupCode(sg.SubGroupKeys, level.Field)

			// Use placeholder if both are empty
			if label == "" && code == "" {
				label = "N/A"
				code = "missing"
			}

			if label != "" || code != "" {
				hasAnyValue = true
			}

			levelKeys[i] = composeHierarchyKey(code, label)
		}

		// Only skip if ALL levels are empty (shouldn't happen with our placeholder logic)
		if !hasAnyValue {
			continue
		}

		current := hierarchy
		for i, key := range levelKeys {
			// No longer skip empty keys - we use placeholders now
			if key == "" {
				key = composeHierarchyKey("missing", "N/A")
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
					Hide:         boolPtr(colConfig.Hide),
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

func buildDynamicRows(root *PriceTableConfiguration, pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse) []AGGridRowData {
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

		itemValue := buildItemValue(root, pattern, sg)
		var columnLabel string
		var columnKey string
		if len(pattern.ColumnLevels) > 0 {
			labelParts := []string{}
			keyParts := []string{}

			for _, level := range pattern.ColumnLevels {
				valueName := getValueNameByGroupCode(sg.SubGroupKeys, level.Field)
				valueCode := getValueCodeByGroupCode(sg.SubGroupKeys, level.Field)

				// Use empty string for missing values but continue building
				if valueName == "" {
					valueName = "" // Keep empty for label
				}
				if valueCode == "" {
					valueCode = "missing" // Use placeholder for missing codes
				}

				labelParts = append(labelParts, valueName)
				keyParts = append(keyParts, sanitizeIdentifier(valueCode, valueName))
			}

			// Build column label and key, filtering out empty parts for label
			nonEmptyLabels := []string{}
			for _, label := range labelParts {
				if label != "" {
					nonEmptyLabels = append(nonEmptyLabels, label)
				}
			}
			if len(nonEmptyLabels) > 0 {
				columnLabel = strings.Join(nonEmptyLabels, " | ")
			}

			// Filter out "missing" placeholders from key parts if we have real values
			filteredKeyParts := []string{}
			for _, key := range keyParts {
				if key != "missing" && key != "" {
					filteredKeyParts = append(filteredKeyParts, key)
				}
			}
			if len(filteredKeyParts) > 0 {
				columnKey = strings.Join(filteredKeyParts, "_")
			}

			// Fallback: if all group codes are missing, use subgroup ID
			if columnKey == "" {
				columnKey = fmt.Sprintf("col_%s", sg.ID)
				if columnLabel == "" {
					columnLabel = columnKey
				}
			}
		} else {
			columnLabel = buildCompositeKey(sg.SubGroupKeys, columnGroupFields)
			columnCode := buildCompositeCodeKey(sg.SubGroupKeys, columnGroupFields)
			columnKey = sanitizeIdentifier(columnCode, columnLabel)
		}
		if columnLabel == "" {
			columnLabel = columnKey
		}
		// No longer skip if columnKey is empty - we now always have a fallback
		if columnKey == "" {
			columnKey = fmt.Sprintf("col_%s", sg.ID)
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

		// Add flattened SubGroupKey fields for direct access
		for _, sgk := range sg.SubGroupKeys {
			row[sgk.GroupCode] = sgk.ValueName
		}

		udfData := make(map[string]interface{})
		isHighlightValue := false
		inactiveValue := false
		hasInactiveValue := false
		var lineBundleValue *float64
		var stockValue interface{}
		var stockQuantityValue interface{}
		var batchNoValue interface{}
		var supplierNameValue interface{}
		var warehouseValue interface{}
		var codeValue interface{}
		var deliveryDateValue interface{}
		var tonValue interface{}
		var nextProductionValue interface{}
		// Track which awaiting_production and selling fields were found
		hasAwaitingProductionImportDate := false
		hasAwaitingProductionTon := false
		hasAwaitingProductionProducer := false
		hasSellingFast := false
		hasSellingSlow := false
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
				if sn, ok := udfData["supplier_name"]; ok {
					supplierNameValue = sn
				}
				if wh, ok := udfData["warehouse"]; ok {
					warehouseValue = wh
				}
				if code, ok := udfData["code"]; ok {
					codeValue = code
				}
				if deliveryDate, ok := udfData["delivery_date"]; ok {
					deliveryDateValue = deliveryDate
				}
				if ton, ok := udfData["ton"]; ok {
					tonValue = ton
				}
				if nextProduction, ok := udfData["next_production"]; ok {
					nextProductionValue = nextProduction
				}

				for key, value := range udfData {
					if key == "is_highlight" || key == "inactive" || key == "stock_quantity" || key == "batch_no" || key == "warehouse" || key == "code" {
						continue
					}

					if key == "stock" {
						stockValue = value
					}

					// Handle awaiting_production fields directly from udf_json
					if key == "awaiting_production_import_date" {
						row[fmt.Sprintf("%s_awaiting_production_%s", columnKey, sanitizeFieldName("awaiting_production_import_date"))] = value
						hasAwaitingProductionImportDate = true
						continue
					}
					if key == "awaiting_production_ton" {
						row[fmt.Sprintf("%s_awaiting_production_%s", columnKey, sanitizeFieldName("awaiting_production_ton"))] = value
						hasAwaitingProductionTon = true
						continue
					}
					if key == "awaiting_production_producer" {
						row[fmt.Sprintf("%s_awaiting_production_%s", columnKey, sanitizeFieldName("awaiting_production_producer"))] = value
						hasAwaitingProductionProducer = true
						continue
					}

					// Handle selling fields directly from udf_json
					if key == "selling_fast" {
						if sellingFast, ok := value.(bool); ok {
							row[fmt.Sprintf("%s_selling_%s", columnKey, sanitizeFieldName("selling_fast"))] = sellingFast
						} else {
							row[fmt.Sprintf("%s_selling_%s", columnKey, sanitizeFieldName("selling_fast"))] = false
						}
						hasSellingFast = true
						continue
					}
					if key == "selling_slow" {
						if sellingSlow, ok := value.(bool); ok {
							row[fmt.Sprintf("%s_selling_%s", columnKey, sanitizeFieldName("selling_slow"))] = sellingSlow
						} else {
							row[fmt.Sprintf("%s_selling_%s", columnKey, sanitizeFieldName("selling_slow"))] = false
						}
						hasSellingSlow = true
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

		// Set default values for awaiting_production fields if they weren't found in udf_json
		if !hasAwaitingProductionImportDate {
			row[fmt.Sprintf("%s_awaiting_production_%s", columnKey, sanitizeFieldName("awaiting_production_import_date"))] = nil
		}
		if !hasAwaitingProductionTon {
			row[fmt.Sprintf("%s_awaiting_production_%s", columnKey, sanitizeFieldName("awaiting_production_ton"))] = nil
		}
		if !hasAwaitingProductionProducer {
			row[fmt.Sprintf("%s_awaiting_production_%s", columnKey, sanitizeFieldName("awaiting_production_producer"))] = nil
		}

		// Set default values for selling fields if they weren't found in udf_json
		if !hasSellingFast {
			row[fmt.Sprintf("%s_selling_%s", columnKey, sanitizeFieldName("selling_fast"))] = false
		}
		if !hasSellingSlow {
			row[fmt.Sprintf("%s_selling_%s", columnKey, sanitizeFieldName("selling_slow"))] = false
		}

		row["is_highlight"] = isHighlightValue
		row["total_weight"] = getWeightSpecFromInventory(sg)
		row["avg_kg_stock"] = getAvgProductFromInventory(sg)

		for _, colConfig := range pattern.Columns {
			fieldName := fmt.Sprintf("%s_%s", columnKey, colConfig.Field)

			// Check if dataMapping is a direct SubGroupKey field (e.g., "PG03")
			// Skip line_bundle so it is only ever taken from udf_json
			if colConfig.DataMapping != "" && colConfig.DataMapping != "line_bundle" {
				if value, exists := row[colConfig.DataMapping]; exists {
					row[fieldName] = value
					continue
				}
			}

			// Check if dataMapping is a product group reference
			groupCode := convertDataMappingToGroupCode(colConfig.DataMapping)
			if groupCode != "" {
				row[fieldName] = getValueNameByGroupCode(sg.SubGroupKeys, groupCode)
				continue
			}

			// Check if dataMapping is a composite product group reference
			compositeGroupCodes := extractGroupCodesFromCompositeMapping(colConfig.DataMapping)
			if len(compositeGroupCodes) == 2 {
				val1 := getValueNameByGroupCode(sg.SubGroupKeys, compositeGroupCodes[0])
				val2 := getValueNameByGroupCode(sg.SubGroupKeys, compositeGroupCodes[1])
				if strings.Contains(colConfig.DataMapping, "_x_") {
					row[fieldName] = fmt.Sprintf("%s x %s", val1, val2)
				} else {
					row[fieldName] = val1 + val2
				}
				continue
			}

			switch colConfig.DataMapping {
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
			case "delivery_date":
				row[fieldName] = deliveryDateValue
			case "ton":
				row[fieldName] = tonValue
			case "next_production":
				row[fieldName] = nextProductionValue
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
				if batchNoValue != nil && fmt.Sprintf("%v", batchNoValue) != "" {
					row[fieldName] = batchNoValue
				} else if sg.BatchNo != "" {
					row[fieldName] = sg.BatchNo
				} else {
					row[fieldName] = nil
				}
			case "supplier_name":
				if sg.SupplierName != "" {
					row[fieldName] = sg.SupplierName
				} else if supplierNameValue != nil {
					row[fieldName] = supplierNameValue
				} else {
					row[fieldName] = nil
				}
			case "warehouse":
				row[fieldName] = warehouseValue
			case "code":
				row[fieldName] = codeValue
			case "total_weight":
				row[fieldName] = getWeightSpecFromInventory(sg)
			case "avg_weight":
				row[fieldName] = getAvgProductFromInventory(sg)
			case "avg_kg_stock":
				row[fieldName] = getAvgProductFromInventory(sg)
			case "":
				// Empty dataMapping - set default values for calculated/empty fields
				if colConfig.Field == "total_weight" {
					row[fieldName] = getWeightSpecFromInventory(sg)
				} else if colConfig.Field == "avg_kg_stock" {
					row[fieldName] = getAvgProductFromInventory(sg)
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

func buildItemValue(root *PriceTableConfiguration, pattern *PatternConfig, sg models.PriceListSubGroupResponse) string {
	// Use getDefaultItemFormat to get the format with proper priority:
	// 1. PatternConfig.ItemFormat (pattern specific)
	// 2. ValueMappings.DefaultItemFormat (root or pattern mappings)
	// 3. legacyDefaultItemFormat (hardcoded legacy default)
	format := getDefaultItemFormat(root, pattern)

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

func buildDirectRows(root *PriceTableConfiguration, pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse) []AGGridRowData {
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
		var nextProductionValue interface{}
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
		var sellingFastValue bool
		var sellingSlowValue bool
		var awaitingProductionImportDateValue interface{}
		var awaitingProductionDeliveryDateValue interface{}
		var awaitingProductionTonValue interface{}
		var awaitingProductionProducerValue interface{}
		var supplierNameValue interface{}
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
				// Extract awaiting_production fields directly from udf_json
				if awaiting_production_import_date, ok := udfData["awaiting_production_import_date"]; ok {
					awaitingProductionImportDateValue = awaiting_production_import_date
				}
				if awaiting_production_delivery_date, ok := udfData["awaiting_production_delivery_date"]; ok {
					awaitingProductionDeliveryDateValue = awaiting_production_delivery_date
				}
				if awaiting_production_ton, ok := udfData["awaiting_production_ton"]; ok {
					awaitingProductionTonValue = awaiting_production_ton
				}
				if awaiting_production_producer, ok := udfData["awaiting_production_producer"]; ok {
					awaitingProductionProducerValue = awaiting_production_producer
				}
				if supplier_name, ok := udfData["supplier_name"]; ok {
					supplierNameValue = supplier_name
				}
				// Extract selling fields directly from udf_json
				if selling_fast, ok := udfData["selling_fast"].(bool); ok {
					sellingFastValue = selling_fast
				}
				if selling_slow, ok := udfData["selling_slow"].(bool); ok {
					sellingSlowValue = selling_slow
				}
				if next_production, ok := udfData["next_production"]; ok {
					nextProductionValue = next_production
				}
			}
		}
		row["item"] = buildItemValue(root, pattern, sg)

		// Add flattened SubGroupKey fields for direct access
		for _, sgk := range sg.SubGroupKeys {
			row[sgk.GroupCode] = sgk.ValueName
		}

		for _, fixedCol := range pattern.FixedColumns {
			// Skip columns that use valueGetter (client-side computed values like "#")
			// to avoid adding empty keys to tableData
			if fixedCol.ValueGetter != "" {
				continue
			}

			// Handle "type" as a configurable special case that historically mapped to PRODUCT_GROUP9.
			// Handle "type" as a configurable special case that historically mapped to PRODUCT_GROUP9.
			if fixedCol.DataMapping == "type" {
				vm := getEffectiveValueMappings(root, pattern)
				groupCode := getSpecialMapping(vm, "type", "PG09")
				row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, groupCode)
				continue
			}

			// Check if dataMapping is a direct SubGroupKey field (e.g., "PG03")
			// Skip line_bundle so it is only ever taken from udf_json
			if fixedCol.DataMapping != "" && fixedCol.DataMapping != "line_bundle" {
				if value, exists := row[fixedCol.DataMapping]; exists {
					row[fixedCol.Field] = value
					continue
				}
			}

			// Check if dataMapping is a product group reference
			groupCode := convertDataMappingToGroupCode(fixedCol.DataMapping)
			if groupCode != "" {
				row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, groupCode)
				continue
			}

			// Check if dataMapping is a composite product group reference
			compositeGroupCodes := extractGroupCodesFromCompositeMapping(fixedCol.DataMapping)
			if len(compositeGroupCodes) == 2 {
				val1 := getValueNameByGroupCode(sg.SubGroupKeys, compositeGroupCodes[0])
				val2 := getValueNameByGroupCode(sg.SubGroupKeys, compositeGroupCodes[1])
				if strings.Contains(fixedCol.DataMapping, "_x_") {
					row[fixedCol.Field] = fmt.Sprintf("%s x %s", val1, val2)
				} else {
					row[fixedCol.Field] = val1 + val2
				}
				continue
			}

			switch fixedCol.DataMapping {
			case "item":
			case "ship_no":
				row[fixedCol.Field] = shipNoValue
			case "price_weight":
				row[fixedCol.Field] = sg.PriceWeight
			case "before_price_weight":
				row[fixedCol.Field] = sg.BeforePriceWeight
			case "before_total_net_price_weight":
				row[fixedCol.Field] = sg.BeforeTotalNetPriceWeight
			case "extra_price_weight":
				row[fixedCol.Field] = sg.ExtraPriceWeight
			case "market_weight":
				if marketWeightValue != nil {
					row[fixedCol.Field] = *marketWeightValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "line_bundle":
				if lineBundleValue != nil {
					row[fixedCol.Field] = *lineBundleValue
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
			case "total_weight":
				row[fixedCol.Field] = getWeightSpecFromInventory(sg)
			case "avg_weight":
				row[fixedCol.Field] = getAvgProductFromInventory(sg)
			case "import_date":
				if importDateValue != nil {
					row[fixedCol.Field] = importDateValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "delivery_date":
				if deliveryDateValue != nil {
					row[fixedCol.Field] = deliveryDateValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "ton":
				if tonValue != nil {
					row[fixedCol.Field] = tonValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "producer":
				if producerValue != nil {
					row[fixedCol.Field] = producerValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "next_production":
				if nextProductionValue != nil {
					row[fixedCol.Field] = nextProductionValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "tsm":
				if tsmValue != nil {
					row[fixedCol.Field] = tsmValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "selling_fast":
				row[fixedCol.Field] = sellingFastValue
			case "selling_slow":
				row[fixedCol.Field] = sellingSlowValue
			case "stock_quantity":
				if stockQuantityValue != nil {
					row[fixedCol.Field] = stockQuantityValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "batch_no":
				if sg.BatchNo != "" {
					row[fixedCol.Field] = sg.BatchNo
				} else if batchNoValue != nil {
					row[fixedCol.Field] = batchNoValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "warehouse":
				if warehouseValue != nil {
					row[fixedCol.Field] = warehouseValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "code":
				if codeValue != nil {
					row[fixedCol.Field] = codeValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "bkk":
				if bkkValue != nil {
					row[fixedCol.Field] = bkkValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "factory":
				if factoryValue != nil {
					row[fixedCol.Field] = factoryValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "country":
				if countryValue != nil {
					row[fixedCol.Field] = countryValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "institute":
				if instituteValue != nil {
					row[fixedCol.Field] = instituteValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "awaiting_production_ton":
				if awaitingProductionTonValue != nil {
					row[fixedCol.Field] = awaitingProductionTonValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "awaiting_production_producer":
				if awaitingProductionProducerValue != nil {
					row[fixedCol.Field] = awaitingProductionProducerValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "awaiting_production_import_date":
				if awaitingProductionImportDateValue != nil {
					row[fixedCol.Field] = awaitingProductionImportDateValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "awaiting_production_delivery_date":
				if awaitingProductionDeliveryDateValue != nil {
					row[fixedCol.Field] = awaitingProductionDeliveryDateValue
				} else {
					row[fixedCol.Field] = nil
				}
			case "supplier_name":
				if sg.SupplierName != "" {
					row[fixedCol.Field] = sg.SupplierName
				} else if supplierNameValue != nil {
					row[fixedCol.Field] = supplierNameValue
				} else {
					row[fixedCol.Field] = nil
				}
			}

			if fixedCol.DataMapping == "" {
				switch fixedCol.Field {
				case "total_weight":
					row[fixedCol.Field] = getWeightSpecFromInventory(sg)
				case "avg_kg_stock":
					row[fixedCol.Field] = getAvgProductFromInventory(sg)
				default:
					// For other fields without dataMapping, try to infer from field name
					if strings.HasPrefix(fixedCol.Field, "product_group_") {
						// Convert "product_group_6" to "PRODUCT_GROUP6"
						groupCode := convertDataMappingToGroupCode(fixedCol.Field)
						if groupCode != "" {
							row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, groupCode)
						} else {
							row[fixedCol.Field] = ""
						}
					} else if pgCodeRegex.MatchString(fixedCol.Field) {
						// Preserve PGxx codes (e.g. PG06) by reading directly from SubGroupKeys
						row[fixedCol.Field] = getValueNameByGroupCode(sg.SubGroupKeys, fixedCol.Field)
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
				case "before_total_net_price_weight":
					row[childCol.Field] = sg.BeforeTotalNetPriceWeight
				case "total_net_price_weight":
					row[childCol.Field] = sg.TotalNetPriceWeight
				case "before_price_unit":
					row[childCol.Field] = sg.BeforePriceUnit
				case "before_total_net_price_unit":
					row[childCol.Field] = sg.BeforeTotalNetPriceUnit
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
				case "next_production":
					row[childCol.Field] = nextProductionValue
				case "fast", "selling_fast":
					row[childCol.Field] = sellingFastValue
				case "slow", "selling_slow":
					row[childCol.Field] = sellingSlowValue
				case "awaiting_production_import_date":
					row[childCol.Field] = awaitingProductionImportDateValue
				case "awaiting_production_delivery_date":
					row[childCol.Field] = awaitingProductionDeliveryDateValue
				case "awaiting_production_ton":
					row[childCol.Field] = awaitingProductionTonValue
				case "awaiting_production_producer":
					row[childCol.Field] = awaitingProductionProducerValue
				}
			}
		}

		for _, colConfig := range pattern.Columns {
			// Check if dataMapping is a direct SubGroupKey field (e.g., "PG03")
			// Skip line_bundle so it is only ever taken from udf_json
			if colConfig.DataMapping != "" && colConfig.DataMapping != "line_bundle" {
				if value, exists := row[colConfig.DataMapping]; exists {
					row[colConfig.Field] = value
					continue
				}
			}

			// Check if dataMapping is a product group reference
			groupCode := convertDataMappingToGroupCode(colConfig.DataMapping)
			if groupCode != "" {
				row[colConfig.Field] = getValueNameByGroupCode(sg.SubGroupKeys, groupCode)
				continue
			}

			// Check if dataMapping is a composite product group reference
			compositeGroupCodes := extractGroupCodesFromCompositeMapping(colConfig.DataMapping)
			if len(compositeGroupCodes) == 2 {
				val1 := getValueNameByGroupCode(sg.SubGroupKeys, compositeGroupCodes[0])
				val2 := getValueNameByGroupCode(sg.SubGroupKeys, compositeGroupCodes[1])
				if strings.Contains(colConfig.DataMapping, "_x_") {
					row[colConfig.Field] = fmt.Sprintf("%s x %s", val1, val2)
				} else {
					row[colConfig.Field] = val1 + val2
				}
				continue
			}

			switch colConfig.DataMapping {
			case "extra_price_unit":
				row[colConfig.Field] = sg.ExtraPriceUnit
			case "before_price_weight":
				row[colConfig.Field] = sg.BeforePriceWeight
			case "before_total_net_price_weight":
				row[colConfig.Field] = sg.BeforeTotalNetPriceWeight
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
				if odValue != nil {
					row[colConfig.Field] = odValue
				} else {
					row[colConfig.Field] = nil
				}
			case "stock":
				if stockValue != nil {
					row[colConfig.Field] = stockValue
				} else {
					row[colConfig.Field] = nil
				}
			case "import_date":
				if importDateValue != nil {
					row[colConfig.Field] = importDateValue
				} else {
					row[colConfig.Field] = nil
				}
			case "delivery_date":
				if deliveryDateValue != nil {
					row[colConfig.Field] = deliveryDateValue
				} else {
					row[colConfig.Field] = nil
				}
			case "ton":
				if tonValue != nil {
					row[colConfig.Field] = tonValue
				} else {
					row[colConfig.Field] = nil
				}
			case "producer":
				if producerValue != nil {
					row[colConfig.Field] = producerValue
				} else {
					row[colConfig.Field] = nil
				}
			case "next_production":
				if nextProductionValue != nil {
					row[colConfig.Field] = nextProductionValue
				} else {
					row[colConfig.Field] = nil
				}
			case "tsm":
				if tsmValue != nil {
					row[colConfig.Field] = tsmValue
				} else {
					row[colConfig.Field] = nil
				}
			case "selling_fast":
				row[colConfig.Field] = sellingFastValue
			case "selling_slow":
				row[colConfig.Field] = sellingSlowValue
			case "stock_quantity":
				if stockQuantityValue != nil {
					row[colConfig.Field] = stockQuantityValue
				} else {
					row[colConfig.Field] = nil
				}
			case "batch_no":
				if batchNoValue != nil && fmt.Sprintf("%v", batchNoValue) != "" {
					row[colConfig.Field] = batchNoValue
				} else if sg.BatchNo != "" {
					row[colConfig.Field] = sg.BatchNo
				} else {
					row[colConfig.Field] = nil
				}
			case "supplier_name":
				if sg.SupplierName != "" {
					row[colConfig.Field] = sg.SupplierName
				} else if supplierNameValue != nil {
					row[colConfig.Field] = supplierNameValue
				} else {
					row[colConfig.Field] = nil
				}
			case "warehouse":
				if warehouseValue != nil {
					row[colConfig.Field] = warehouseValue
				} else {
					row[colConfig.Field] = nil
				}
			case "code":
				if codeValue != nil {
					row[colConfig.Field] = codeValue
				} else {
					row[colConfig.Field] = nil
				}
			case "bkk":
				if bkkValue != nil {
					row[colConfig.Field] = bkkValue
				} else {
					row[colConfig.Field] = nil
				}
			case "factory":
				if factoryValue != nil {
					row[colConfig.Field] = factoryValue
				} else {
					row[colConfig.Field] = nil
				}
			case "country":
				if countryValue != nil {
					row[colConfig.Field] = countryValue
				} else {
					row[colConfig.Field] = nil
				}
			case "institute":
				if instituteValue != nil {
					row[colConfig.Field] = instituteValue
				} else {
					row[colConfig.Field] = nil
				}
			case "is_highlight":
				row[colConfig.Field] = isHighlightValue
			case "inactive":
				if hasInactiveValue {
					row[colConfig.Field] = inactiveValue
				} else {
					row[colConfig.Field] = false
				}
			case "awaiting_production_ton":
				if awaitingProductionTonValue != nil {
					row[colConfig.Field] = awaitingProductionTonValue
				} else {
					row[colConfig.Field] = nil
				}
			case "awaiting_production_producer":
				if awaitingProductionProducerValue != nil {
					row[colConfig.Field] = awaitingProductionProducerValue
				} else {
					row[colConfig.Field] = nil
				}
			case "awaiting_production_import_date":
				if awaitingProductionImportDateValue != nil {
					row[colConfig.Field] = awaitingProductionImportDateValue
				} else {
					row[colConfig.Field] = nil
				}
			case "awaiting_production_delivery_date":
				if awaitingProductionDeliveryDateValue != nil {
					row[colConfig.Field] = awaitingProductionDeliveryDateValue
				} else {
					row[colConfig.Field] = nil
				}
			case "total_weight":
				row[colConfig.Field] = getWeightSpecFromInventory(sg)
			case "avg_weight":
				row[colConfig.Field] = getAvgProductFromInventory(sg)
			case "avg_kg_stock":
				row[colConfig.Field] = getAvgProductFromInventory(sg)
			case "":
				// Empty dataMapping - set default values for calculated/empty fields
				if colConfig.Field == "total_weight" {
					row[colConfig.Field] = getWeightSpecFromInventory(sg)
				} else if colConfig.Field == "avg_kg_stock" {
					row[colConfig.Field] = getAvgProductFromInventory(sg)
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

// buildProductGroup2ColumnGroupsWithCode builds dynamic column groups using a configurable group code
func buildProductGroup2ColumnGroupsWithCode(pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse, productGroup2Code string) []ColumnDef {
	type columnGroupValue struct {
		Label string
		Code  string
	}

	columns := []ColumnDef{}
	uniqueValues := make(map[string]columnGroupValue)

	// Extract unique PRODUCT_GROUP2 values
	for _, sg := range subGroups {
		label := getValueNameByGroupCode(sg.SubGroupKeys, productGroup2Code)
		if label == "" {
			continue
		}
		code := getValueCodeByGroupCode(sg.SubGroupKeys, productGroup2Code)
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
				Hide:         boolPtr(colConfig.Hide),
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

// buildDirectRowsWithProductGroup2WithCode builds rows with fixed columns and dynamic column group data using configurable group codes
func buildDirectRowsWithProductGroup2WithCode(root *PriceTableConfiguration, pattern *PatternConfig, subGroups []models.PriceListSubGroupResponse, productGroup2Code, productGroup6Code, productGroup7Code, productGroup5Code, productGroup3Code string) []AGGridRowData {
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
		code := getValueCodeByGroupCode(sg.SubGroupKeys, productGroup2Code)
		label := getValueNameByGroupCode(sg.SubGroupKeys, productGroup2Code)
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
		thickness := getValueNameByGroupCode(sg.SubGroupKeys, productGroup6Code)
		length := getValueNameByGroupCode(sg.SubGroupKeys, productGroup7Code)
		thicknessLength := strings.TrimSpace(fmt.Sprintf("%s x %s", thickness, length))

		sizePart1 := getValueNameByGroupCode(sg.SubGroupKeys, productGroup5Code)
		sizePart2 := getValueNameByGroupCode(sg.SubGroupKeys, productGroup3Code)
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
			if itemValue := buildItemValue(root, pattern, sg); itemValue != "" {
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

		// Parse UDF data for highlight flag and others
		isHighlightValue := false
		hasInactiveValue := false
		inactiveValue := false
		var remarkValue interface{}
		var batchNoValue interface{}
		var supplierNameValue interface{}

		if len(sg.UdfJson) > 0 {
			udfData := make(map[string]interface{})
			if err := json.Unmarshal(sg.UdfJson, &udfData); err == nil {
				if h, ok := udfData["is_highlight"].(bool); ok {
					isHighlightValue = h
				}
				if i, ok := udfData["inactive"].(bool); ok {
					hasInactiveValue = true
					inactiveValue = i
				}
				if r, ok := udfData["remark"]; ok {
					remarkValue = r
				}
				if bn, ok := udfData["batch_no"]; ok {
					batchNoValue = bn
				}
				if sn, ok := udfData["supplier_name"]; ok {
					supplierNameValue = sn
				}
			}
		}

		pg2Code := getValueCodeByGroupCode(sg.SubGroupKeys, productGroup2Code)
		pg2Label := getValueNameByGroupCode(sg.SubGroupKeys, productGroup2Code)
		if pg2Code == "" {
			continue
		}
		identifier := sanitizeIdentifier(pg2Code, pg2Label)

		// Set subgroup identifier for the specific PRODUCT_GROUP2 entry
		row[fmt.Sprintf("%s_subgroup_id", identifier)] = sg.ID

		// Add flattened SubGroupKey fields for direct access
		for _, sgk := range sg.SubGroupKeys {
			row[sgk.GroupCode] = sgk.ValueCode
		}

		// Populate dynamic column values for this PRODUCT_GROUP2
		for _, colConfig := range pattern.Columns {
			fieldName := fmt.Sprintf("%s_%s", identifier, colConfig.Field)
			switch colConfig.DataMapping {
			case "is_highlight":
				row[fieldName] = isHighlightValue
			case "inactive":
				if hasInactiveValue {
					row[fieldName] = inactiveValue
				} else {
					row[fieldName] = false
				}
			case "remark":
				row[fieldName] = remarkValue
			case "price_weight":
				row[fieldName] = sg.PriceWeight
			case "before_total_net_price_weight":
				row[fieldName] = sg.BeforeTotalNetPriceWeight
			case "total_net_price_weight":
				row[fieldName] = sg.TotalNetPriceWeight
			case "extra_price_weight":
				row[fieldName] = sg.ExtraPriceWeight
			case "before_price_unit":
				row[fieldName] = sg.BeforePriceUnit
			case "total_net_price_unit":
				row[fieldName] = sg.TotalNetPriceUnit
			case "extra_price_unit":
				row[fieldName] = sg.ExtraPriceUnit
			case "total_weight":
				row[fieldName] = getWeightSpecFromInventory(sg)
			case "avg_weight":
				row[fieldName] = getAvgProductFromInventory(sg)
			case "avg_kg_stock":
				row[fieldName] = getAvgProductFromInventory(sg)
			case "batch_no":
				if batchNoValue != nil && fmt.Sprintf("%v", batchNoValue) != "" {
					row[fieldName] = batchNoValue
				} else if sg.BatchNo != "" {
					row[fieldName] = sg.BatchNo
				} else {
					row[fieldName] = nil
				}
			case "supplier_name":
				if supplierNameValue != nil && fmt.Sprintf("%v", supplierNameValue) != "" {
					row[fieldName] = supplierNameValue
				} else if sg.SupplierName != "" {
					row[fieldName] = sg.SupplierName
				} else {
					row[fieldName] = nil
				}
			default:
				// Fallback to empty string for string-types like remark if not matched
				if colConfig.DataMapping == "remark" {
					row[fieldName] = ""
				} else {
					row[fieldName] = nil
				}
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
			Hide:            boolPtr(fixedCol.Hide),
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
				Hide:       boolPtr(childColConfig.Hide),
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
			Hide:       boolPtr(colConfig.Hide),
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

// PriceTableHandler is the standard signature for all price table handlers
// All handlers must accept (data, groupCode) even if they don't use groupCode
type PriceTableHandler func([]models.GetPriceListResponse, string) (PriceListDetailApiResponse, error)

// handlerRegistry maps handler identifiers (e.g., "BuildGroup1Item1Response") to handler functions
var handlerRegistry = map[string]PriceTableHandler{
	"BuildGroup1Item1Response": func(data []models.GetPriceListResponse, _ string) (PriceListDetailApiResponse, error) {
		return BuildGroup1Item1Response(data)
	},
	"BuildGroup1Item2Response":  BuildGroup1Item2Response,
	"BuildGroup1Item3Response":  BuildGroup1Item3Response,
	"BuildGroup1Item4Response":  BuildGroup1Item4Response,
	"BuildGroup1Item5Response":  BuildGroup1Item5Response,
	"BuildGroup1Item6Response":  BuildGroup1Item6Response,
	"BuildGroup1Item7Response":  BuildGroup1Item7Response,
	"BuildGroup1Item8Response":  BuildGroup1Item8Response,
	"BuildGroup1Item9Response":  BuildGroup1Item9Response,
	"BuildGroup1Item10Response": BuildGroup1Item10Response,
	"BuildGroup1Item11Response": BuildGroup1Item11Response,
	"BuildGroup1Item12Response": BuildGroup1Item12Response,
	"BuildGroup1Item13Response": BuildGroup1Item13Response,
}

// ResolveHandler resolves a handler function from configuration.
// It checks both root-level and pattern-level HandlerMappings in the configuration.
// Returns the handler function and true if found, or nil and false if not found.
// Callers should fall back to GetDefaultHandlers() if this returns false.
func ResolveHandler(config *PriceTableConfiguration, groupCode string) (PriceTableHandler, bool) {
	if config == nil {
		return nil, false
	}

	// Check root-level ValueMappings first
	if config.ValueMappings != nil && config.ValueMappings.HandlerMappings != nil {
		if handlerID, ok := config.ValueMappings.HandlerMappings[groupCode]; ok && handlerID != "" {
			if handler, found := handlerRegistry[handlerID]; found {
				return handler, true
			}
		}
	}

	// Check pattern-level ValueMappings
	for _, pattern := range config.Patterns {
		if pattern.ValueMappings != nil && pattern.ValueMappings.HandlerMappings != nil {
			if handlerID, ok := pattern.ValueMappings.HandlerMappings[groupCode]; ok && handlerID != "" {
				if handler, found := handlerRegistry[handlerID]; found {
					return handler, true
				}
			}
		}
	}

	return nil, false
}

// GetDefaultHandlers returns the default hardcoded handlers map for backward compatibility.
// This is used when configuration is not available or doesn't specify handler mappings.
func GetDefaultHandlers() map[string]PriceTableHandler {
	return map[string]PriceTableHandler{
		"GROUP_1_ITEM_1": func(data []models.GetPriceListResponse, _ string) (PriceListDetailApiResponse, error) {
			return BuildGroup1Item1Response(data)
		},
		"GROUP_1_ITEM_2":  BuildGroup1Item2Response,
		"GROUP_1_ITEM_3":  BuildGroup1Item3Response,
		"GROUP_1_ITEM_4":  BuildGroup1Item4Response,
		"GROUP_1_ITEM_5":  BuildGroup1Item5Response,
		"GROUP_1_ITEM_6":  BuildGroup1Item6Response,
		"GROUP_1_ITEM_7":  BuildGroup1Item7Response,
		"GROUP_1_ITEM_8":  BuildGroup1Item8Response,
		"GROUP_1_ITEM_9":  BuildGroup1Item9Response,
		"GROUP_1_ITEM_10": BuildGroup1Item10Response,
		"GROUP_1_ITEM_11": BuildGroup1Item11Response,
		"GROUP_1_ITEM_12": BuildGroup1Item12Response,
		"GROUP_1_ITEM_13": BuildGroup1Item13Response,
		"GROUP_1_ITEM_14": BuildGroup1Item12Response,
		"GROUP_1_ITEM_15": BuildGroup1Item12Response,
		"GROUP_1_ITEM_16": BuildGroup1Item12Response,
		"GROUP_1_ITEM_17": BuildGroup1Item12Response,
		"GROUP_1_ITEM_18": BuildGroup1Item12Response,
		"GROUP_1_ITEM_19": BuildGroup1Item12Response,
		"GROUP_1_ITEM_20": BuildGroup1Item12Response,
		"GROUP_1_ITEM_21": func(data []models.GetPriceListResponse, _ string) (PriceListDetailApiResponse, error) {
			return BuildGroup1Item1Response(data)
		},
	}
}
