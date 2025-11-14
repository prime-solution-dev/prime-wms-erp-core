# Price List Detail Service - Pattern System

## Overview

The Price List Detail service (`get-price-detail`) is a sophisticated pricing system that transforms raw price list data from the database into structured, configurable table views for different product categories. The system uses a pattern-based approach where different product groups can have different table layouts, column structures, and data organization strategies.

## Architecture

### Core Components

1. **Service Layer** (`get-price-detail.go`)
   - Main entry point for the API
   - Handles request validation and orchestration
   - Routes to appropriate pattern builders based on `GroupKey`

2. **Pattern System** (`patterns/`)
   - Configuration-driven table generation
   - Multiple pattern implementations for different product categories
   - Shared utilities for common operations

3. **Repository Layer** (`priceList/repository.go`)
   - Database access and data retrieval
   - GORM-based ORM operations
   - History tracking for audit purposes

4. **Data Models** (`models/pricelist.go`)
   - Type definitions for price list entities
   - Request/Response DTOs
   - Database schema mappings

## Request Flow

### 1. API Entry Point

**Function**: `GetPriceDetail(ctx *gin.Context, jsonPayload string)`

**Request Structure**:
```go
type GetPriceDetailRequest struct {
    CompanyCode       string     `json:"company_code"`
    SiteCodes         []string   `json:"site_codes"`
    GroupCodes        []string   `json:"group_codes"`
    EffectiveDateFrom *time.Time `json:"effective_date_from"`
    EffectiveDateTo   *time.Time `json:"effective_date_to"`
}
```

**Validation**:
- `company_code` is required
- `site_codes` must have at least one entry
- `group_codes` must have at least one entry

### 2. Data Loading Pipeline

The service follows this data loading sequence:

```
GetPriceDetail
  └─> loadPriceData()
      └─> getGroupSubGroup()      // Load groups and subgroups from DB
      └─> getTerms()               // Load payment terms
      └─> getExtras()              // Load extra pricing rules
      └─> transformToGetPriceListResponse()  // Transform to API format
```

### 3. Pattern Selection

After loading data, the system:

1. **Extracts GroupKey**: From the first price list item's `GroupKey` field
   - The `GroupKey` is extracted from the first subgroup's `subgroup_key` (first part before "|")
   - Example: `"GROUP_1_ITEM_1|GROUP_2_ITEM_2|..."` → `"GROUP_1_ITEM_1"`

2. **Routes to Pattern Handler**: Based on `GroupKey`, selects appropriate builder:
   - `GROUP_1_ITEM_1` → `BuildGroup1Item1Response()`
   - `GROUP_1_ITEM_2` → `BuildGroup1Item2Response()`
   - `GROUP_1_ITEM_3` → `BuildGroup1Item3Response()`
   - `GROUP_1_ITEM_4` → Returns empty response (placeholder)

## Pattern System

### Pattern Configuration Files

Patterns are defined in JSON configuration files located in `patterns/configs/`:
- `GROUP_1_ITEM_1_PATTERN.json`
- `GROUP_1_ITEM_2_PATTERN.json`
- `GROUP_1_ITEM_3_PATTERN.json`

Each configuration file contains:

```json
{
  "patterns": [
    {
      "id": "pattern_id",
      "name": "Pattern Name",
      "description": "Description",
      "enabled": true,
      "grouping": {
        "tabs": "PRODUCT_GROUP2",        // Field for tab grouping
        "rows": "PRODUCT_GROUP6",         // Field for row grouping
        "columnGroups": "PRODUCT_GROUP5"   // Field for column grouping
      },
      "columnLevels": [...],              // Multi-level column hierarchy
      "columns": [...],                   // Dynamic columns
      "fixedColumns": [...],             // Fixed/pinned columns
      "columnGroups": [...],              // Static column groups
      "applicableCategories": [...],      // Which categories use this pattern
      "editable_suffixes": [...],         // Editable field suffixes
      "fetchable_suffixes": [...]         // Fetchable field suffixes
    }
  ],
  "defaultPattern": "pattern_id",
  "tableConfig": {
    "groupHeaderHeight": 50,
    "headerHeight": 40,
    "pagination": false,
    "toolbar": {...},
    "gridOptions": {...}
  }
}
```

### Pattern Types

#### 1. GROUP_1_ITEM_1 Pattern

**Builder**: `BuildGroup1Item1Response()`

**Characteristics**:
- **Multi-tab layout**: Creates separate tabs for each `PRODUCT_GROUP2` value
- **Dynamic columns**: Columns are generated based on unique values in `columnGroups` field
- **Pattern selection**: Can select different patterns per tab based on `applicableCategories`
- **Row grouping**: Rows grouped by fields specified in `grouping.rows`
- **Column grouping**: Columns grouped by fields specified in `grouping.columnGroups`

**Data Flow**:
```
1. Group data by GroupKey and PRODUCT_GROUP2
2. For each PRODUCT_GROUP2:
   a. Load pattern configuration
   b. Select pattern based on applicableCategories
   c. Build dynamic columns from unique columnGroup values
   d. Build rows from subgroups
   e. Create tab with table config and data
3. Sort tabs by pattern order and category name
```

#### 2. GROUP_1_ITEM_2 Pattern

**Builder**: `BuildGroup1Item2Response()`

**Characteristics**:
- **Single tab layout**: One tab with fixed label "หมวดเหล็กตัวซี"
- **Fixed columns**: Uses `buildFixedColumns()` - columns defined in pattern config
- **Direct rows**: Uses `buildDirectRows()` - one row per subgroup
- **Default pattern**: Uses the default pattern from config

**Data Flow**:
```
1. Load pattern configuration
2. Select default enabled pattern
3. Collect all subgroups from all price lists
4. Build fixed columns from pattern config
5. Build direct rows (one per subgroup)
6. Create single tab with data
```

#### 3. GROUP_1_ITEM_3 Pattern

**Builder**: `BuildGroup1Item3Response()`

**Characteristics**:
- **Multi-tab layout**: Creates tabs grouped by `PRODUCT_GROUP4`
- **Fixed columns**: Same column structure for all tabs
- **Direct rows**: One row per subgroup
- **Default pattern**: Uses the default pattern from config

**Data Flow**:
```
1. Load pattern configuration
2. Select default enabled pattern
3. Collect all subgroups
4. Group subgroups by PRODUCT_GROUP4
5. Build fixed columns once (shared across tabs)
6. For each PRODUCT_GROUP4:
   a. Build direct rows for that group
   b. Create tab with same columns but different data
```

## Data Transformation

### Helper Functions

#### `getGroupAndItemMappings()`
- Fetches group and group item data from group service
- Fetches payment terms
- Creates lookup maps for resolving codes to names
- Returns: `groupMap`, `groupItemMap`, `paymentTermMap`

#### `transformToGetPriceListResponse()`
- Transforms internal `GetPriceListGroupResponse` to API `GetPriceListResponse`
- Resolves group codes to names using mappings
- Extracts `GroupKey` from subgroup keys
- Formats dates to RFC3339
- Maps subgroup keys with code-to-name resolution

### Column Building

#### `buildDynamicColumns()`
Builds columns based on pattern configuration:

1. **Fixed Columns**: Always included, typically pinned left
2. **Dynamic Columns**: Generated from unique values in data:
   - **Single Level**: `buildSingleLevelColumns()` - flat column groups
   - **Multi Level**: `buildMultiLevelColumns()` - nested column hierarchy

**Single Level Example**:
```
Column Groups: PRODUCT_GROUP5
Unique Values: ["Size A", "Size B", "Size C"]
Result: 3 column groups, each with pattern.columns as children
```

**Multi Level Example**:
```
Column Levels: [
  {level: 1, field: "PRODUCT_GROUP3"},
  {level: 2, field: "PRODUCT_GROUP5"}
]
Result: Nested column groups with hierarchy
```

#### `buildFixedColumns()`
Builds columns from pattern configuration:
- Fixed columns (pinned, locked)
- Column groups (static groups with children)
- Regular columns

### Row Building

#### `buildDynamicRows()`
Creates rows for dynamic column layouts:

1. **Row Key**: Composite key from `grouping.rows` fields
2. **Column Key**: Composite key from `grouping.columnGroups` or `columnLevels`
3. **Field Mapping**: Maps data using `dataMapping` from column config
4. **UDF Data**: Extracts custom fields from `udf_json`
5. **Tooltips**: Handles tooltip data with `_tooltip` suffix

**Field Naming Convention**:
- Row fields: `{sanitized_group_code}` (e.g., `product_group_6`)
- Column fields: `{sanitized_column_key}_{field}` (e.g., `size_a_price_unit`)
- Special fields: `{column_key}_subgroup_id`, `{column_key}_is_trading`

#### `buildDirectRows()`
Creates rows for fixed column layouts:

1. **One row per subgroup**: Direct mapping
2. **Fixed field mapping**: Uses `fixedColumns` and `columnGroups` dataMapping
3. **UDF extraction**: Extracts custom fields (line_bundle, market_weight, od, stock, etc.)
4. **Item construction**: Builds item string from PRODUCT_GROUP4, GROUP6, GROUP7

**Data Mapping Types**:
- `product_group_3`, `product_group_6`: Group value names
- `price_unit`, `price_weight`, etc.: Direct price fields
- `before_*`: Previous price values
- `is_highlight`, `inactive`: Boolean flags from UDF
- `od`, `stock`, `import_date`, `ton`, `producer`: Custom UDF fields
- `fast`, `slow`: Boolean flags from UDF

## Database Schema

### Core Tables

1. **price_list_group**
   - Main price list header
   - Fields: company_code, site_code, group_code, price_unit, price_weight, currency, effective_date

2. **price_list_sub_group**
   - Price list line items
   - Fields: subgroup_key, price_unit, price_weight, extra_price_unit, term_price_weight, udf_json
   - Links to price_list_group via price_list_group_id

3. **price_list_sub_group_key**
   - Key-value pairs for subgroup classification
   - Fields: code (e.g., PRODUCT_GROUP1), value (e.g., GROUP_1_ITEM_1), seq
   - Links to price_list_sub_group via sub_group_id

4. **price_list_group_term**
   - Payment terms for price groups
   - Fields: term_code, pdc, due, pdc_percent, due_percent

5. **price_list_group_extra**
   - Extra pricing rules
   - Fields: extra_key, operator, cond_range_min, cond_range_max

### History Tables

- `price_list_group_history`: Tracks changes to price groups
- `price_list_sub_group_history`: Tracks changes to subgroups

## Key Concepts

### Subgroup Key

The `subgroup_key` is a pipe-delimited string that identifies the product classification:
```
"GROUP_1_ITEM_1|GROUP_2_ITEM_2|GROUP_5_ITEM_2|GROUP_6_ITEM_2"
```

- First part (`GROUP_1_ITEM_1`) is the **GroupKey** - determines which pattern to use
- Remaining parts represent different product group classifications
- Used to build row and column keys in dynamic layouts

### Group Keys

Each subgroup has multiple `SubGroupKeys` that map to product classifications:
- `code`: Group code (e.g., `PRODUCT_GROUP1`, `PRODUCT_GROUP2`)
- `value`: Item code (e.g., `GROUP_1_ITEM_1`, `GROUP_2_ITEM_3`)
- `seq`: Display order

These are resolved to names using the group service:
- `GroupCode` → `GroupName` (e.g., `PRODUCT_GROUP1` → "Product Category 1")
- `ItemCode` → `ItemName` (e.g., `GROUP_1_ITEM_1` → "Steel Sheet")

### UDF JSON

The `udf_json` field in subgroups stores custom, flexible data:
```json
{
  "is_highlight": true,
  "inactive": false,
  "line_bundle": 10.5,
  "market_weight": 25.0,
  "od": "50mm",
  "stock": "Available",
  "import_date": "2024-01-15",
  "ton": 1000,
  "producer": "Producer A",
  "fast": true,
  "slow": false,
  "price_unit_tooltip": {
    "text": "Special pricing",
    "icon": "info"
  }
}
```

This allows patterns to display custom fields without schema changes.

## Response Structure

### PriceListDetailApiResponse

```go
type PriceListDetailApiResponse struct {
    Id   uuid.UUID                  `json:"id"`
    Name string                     `json:"name"`
    Tabs []PriceListDetailTabConfig `json:"tabs"`
}
```

### PriceListDetailTabConfig

```go
type PriceListDetailTabConfig struct {
    ID                uuid.UUID                `json:"id"`
    Label             string                   `json:"label"`
    TableConfig       TableConfig              `json:"tableConfig"`
    TableData         []map[string]interface{} `json:"tableData"`
    EditableSuffixes  []string                 `json:"editable_suffixes,omitempty"`
    FetchableSuffixes []string                 `json:"fetchable_suffixes,omitempty"`
}
```

### TableConfig

Contains AG Grid configuration:
- Column definitions (with grouping, styling, renderers)
- Toolbar settings
- Pagination settings
- Grid options (movable columns, menu hide, etc.)

## Utility Functions

### Field Name Sanitization

- `sanitizeFieldName()`: Converts display names to valid field names
  - Removes special characters
  - Converts to lowercase
  - Replaces spaces with underscores

- `convertGroupCodeToFieldName()`: Converts group codes to field names
  - `PRODUCT_GROUP1` → `product_group_1`

### Key Building

- `buildCompositeKey()`: Builds composite keys from multiple group codes
  - Input: `[PRODUCT_GROUP1, PRODUCT_GROUP2]`
  - Output: `"Value1|Value2"`

- `ExtractGroupKey()`: Extracts first part of subgroup_key
  - Input: `"GROUP_1_ITEM_1|GROUP_2_ITEM_2"`
  - Output: `"GROUP_1_ITEM_1"`

### Pattern Selection

- `selectPatternForCategory()`: Selects pattern based on category
  1. Find pattern with matching `applicableCategories`
  2. Fallback to `defaultPattern`
  3. Fallback to first enabled pattern

## Error Handling

The service handles errors at multiple levels:

1. **Request Validation**: Returns error if required fields missing
2. **Database Connection**: Returns error if DB connection fails
3. **Data Loading**: Returns error if data loading fails
4. **Pattern Loading**: Falls back to GROUP_1_ITEM_1 if pattern file not found
5. **Pattern Selection**: Returns error if no enabled pattern found
6. **Empty Data**: Returns empty response with empty tabs array

## Usage Examples

### Example Request

```json
{
  "company_code": "COMP001",
  "site_codes": ["SITE001", "SITE002"],
  "group_codes": ["GROUP001"],
  "effective_date_from": "2024-01-01T00:00:00Z",
  "effective_date_to": "2024-12-31T23:59:59Z"
}
```

### Example Response (GROUP_1_ITEM_1)

```json
{
  "id": "uuid-here",
  "name": "Price List Detail",
  "tabs": [
    {
      "id": "tab-uuid",
      "label": "Steel Sheet Category A",
      "tableConfig": {
        "title": "Steel Sheet Category A",
        "columns": [
          {
            "field": "#",
            "headerName": "#",
            "pinned": "left",
            "width": 60
          },
          {
            "headerName": "Size A",
            "groupId": "group_size_a",
            "children": [
              {
                "field": "size_a_grade",
                "headerName": "เกรด",
                "width": 80
              },
              {
                "field": "size_a_price_unit",
                "headerName": "ราคาขาย Before",
                "width": 130
              }
            ]
          }
        ],
        "tableData": [
          {
            "id": "row-uuid",
            "product_group_6": "2mm",
            "size_a_grade": "Grade A",
            "size_a_price_unit": 800.0
          }
        ]
      }
    }
  ]
}
```

## Extension Points

### Adding New Patterns

1. Create pattern builder function in `patterns/`:
   ```go
   func BuildGroup1Item4Response(...) (PriceListDetailApiResponse, error)
   ```

2. Add handler in `get-price-detail.go`:
   ```go
   "GROUP_1_ITEM_4": pricePatterns.BuildGroup1Item4Response,
   ```

3. Create pattern config file: `GROUP_1_ITEM_4_PATTERN.json`

### Adding New Data Mappings

1. Add mapping case in `buildDynamicRows()` or `buildDirectRows()`
2. Update pattern config with new `dataMapping` value
3. Ensure UDF JSON structure supports the new field

## Related Services

- **GetPriceList**: Returns raw price list data (used internally by GetPriceDetail)
- **GetPaymentTerm**: Fetches payment term definitions
- **GetGroup**: Fetches group and item master data for name resolution
- **UpdatePriceListSubGroup**: Updates subgroup data (uses same pattern system for validation)

## Testing

Test files are located in:
- `get-latest-pricelist-subgroup_service_test.go`
- `update-pricelist-subgroup_service_test.go`
- `repository_test.go`

## Notes

- The system is designed to be flexible and configuration-driven
- Patterns can be enabled/disabled via JSON config
- Multiple patterns can exist in one config file for different categories
- The first subgroup's GroupKey determines the entire response pattern
- All dates are formatted as RFC3339 strings in responses
- UDF JSON allows extensibility without schema changes

