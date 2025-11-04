# Price List Data Grouping Implementation

## 📋 Overview

This implementation provides a hierarchical data grouping system for price lists based on Product Group patterns. The system organizes price data into structured sets that can be easily consumed by frontend applications to build dynamic tables.

## 🎯 Problem Statement

Given price data with multiple product group dimensions (GROUP1 through GROUP6), we need to:
1. Organize data into logical sets based on `PRODUCT_GROUP2`
2. Within each set, group by the pattern: `PRODUCT_GROUP6 | PRODUCT_GROUP3 | PRODUCT_GROUP5`
3. Focus on unique `PRODUCT_GROUP5` values as column variations

## 🔍 Visual Representation

### Input Data Structure
```
SubGroup Records with GroupKeys:
├── PRODUCT_GROUP1
├── PRODUCT_GROUP2  ← Primary grouping level
├── PRODUCT_GROUP3  ← Part of pattern
├── PRODUCT_GROUP5  ← Part of pattern (focus: unique values)
└── PRODUCT_GROUP6  ← Part of pattern
```

### Output Data Structure
```
GroupedData[]
├── Set 1 (PRODUCT_GROUP2: GROUP_2_ITEM_2)
│   ├── Pattern 1: GROUP_6_ITEM_1|GROUP_3_ITEM_1|GROUP_5_ITEM_1
│   │   └── SubGroups[]
│   ├── Pattern 2: GROUP_6_ITEM_1|GROUP_3_ITEM_4|GROUP_5_ITEM_2
│   │   └── SubGroups[]
│   └── Pattern 3: GROUP_6_ITEM_1|GROUP_3_ITEM_1|GROUP_5_ITEM_3
│       └── SubGroups[]
└── Set 2 (PRODUCT_GROUP2: GROUP_2_ITEM_1)
    ├── Pattern 1: GROUP_6_ITEM_1|GROUP_3_ITEM_1|GROUP_5_ITEM_1
    │   └── SubGroups[]
    ├── Pattern 2: GROUP_6_ITEM_1|GROUP_3_ITEM_4|GROUP_5_ITEM_2
    │   └── SubGroups[]
    └── Pattern 3: GROUP_6_ITEM_1|GROUP_3_ITEM_1|GROUP_5_ITEM_3
        └── SubGroups[]
```

### Table Visualization

Based on the provided image, the grouped data maps to a table structure like this:

```
┌─────────────────┬─────────────────┬─────────────────┬─────────────────┬─────────────────┐
│ Product Group 6 │ Product Group 3 │ Product Group 5 │ Product Group 5 │ Product Group 5 │
├─────────────────┼─────────────────┼─────────────────┼─────────────────┼─────────────────┤
│       G6        │       G3        │   Headerxx      │   Headerxx      │   Headerxx      │
├─────────────────┼─────────────────┼─────────────────┼─────────────────┼─────────────────┤
│       G6        │      G31        │       xx        │       xx        │       xx        │
│       G6        │      G32        │       xx        │       xx        │       xx        │
│       G6        │       G3        │       xx        │       xx        │       xx        │
└─────────────────┴─────────────────┴─────────────────┴─────────────────┴─────────────────┘
```

Each unique `PRODUCT_GROUP5` value becomes a column, grouped under `PRODUCT_GROUP6` and `PRODUCT_GROUP3` row headers.

## 📊 Sample Data Analysis

### Input (from price.json)
```json
{
  "subgroup_key": "GROUP_1_ITEM_1|GROUP_2_ITEM_2|GROUP_5_ITEM_1|GROUP_6_ITEM_1|GROUP_3_ITEM_1",
  "group_keys": [
    {"code": "PRODUCT_GROUP1", "value": "GROUP_1_ITEM_1"},
    {"code": "PRODUCT_GROUP2", "value": "GROUP_2_ITEM_2"},
    {"code": "PRODUCT_GROUP5", "value": "GROUP_5_ITEM_1"},
    {"code": "PRODUCT_GROUP6", "value": "GROUP_6_ITEM_1"},
    {"code": "PRODUCT_GROUP3", "value": "GROUP_3_ITEM_1"}
  ]
}
```

### Output (grouped)
```json
[
  {
    "product_group_2": "GROUP_2_ITEM_2",
    "pattern_groups": [
      {
        "pattern": "GROUP_6_ITEM_1|GROUP_3_ITEM_1|GROUP_5_ITEM_1",
        "product_group_6": "GROUP_6_ITEM_1",
        "product_group_3": "GROUP_3_ITEM_1",
        "product_group_5": "GROUP_5_ITEM_1",
        "sub_groups": [...]
      }
    ]
  }
]
```

## 🔧 Implementation Details

### Core Functions

#### 1. `groupDataByPattern()`
**Purpose**: Main grouping logic that organizes subgroups into the hierarchical structure.

**Algorithm**:
```
1. Initialize map[PRODUCT_GROUP2][Pattern] → PatternGroup
2. For each SubGroup:
   a. Extract GROUP2, GROUP3, GROUP5, GROUP6 values
   b. Create pattern key: "GROUP6|GROUP3|GROUP5"
   c. Add SubGroup to map[GROUP2][Pattern]
3. Convert map to array structure
```

**Complexity**: O(n) where n = number of subgroups

#### 2. `getGroupKeyValue()`
**Purpose**: Extract specific group key value from GroupKeys array.

#### 3. `printGroupingSummary()`
**Purpose**: Output human-readable summary of grouped data for debugging.

**Output Example**:
```
========== DATA GROUPING SUMMARY ==========
Total unique PRODUCT_GROUP2 sets: 2

Set 1 - PRODUCT_GROUP2: GROUP_2_ITEM_2
  Unique patterns (PRODUCT_GROUP6|PRODUCT_GROUP3|PRODUCT_GROUP5): 3
    - Pattern: GROUP_6_ITEM_1|GROUP_3_ITEM_1|GROUP_5_ITEM_1 (SubGroups: 1)
    - Pattern: GROUP_6_ITEM_1|GROUP_3_ITEM_4|GROUP_5_ITEM_2 (SubGroups: 1)
    - Pattern: GROUP_6_ITEM_1|GROUP_3_ITEM_1|GROUP_5_ITEM_3 (SubGroups: 1)
  Unique PRODUCT_GROUP5 values: 3

Set 2 - PRODUCT_GROUP2: GROUP_2_ITEM_1
  Unique patterns (PRODUCT_GROUP6|PRODUCT_GROUP3|PRODUCT_GROUP5): 3
    - Pattern: GROUP_6_ITEM_1|GROUP_3_ITEM_1|GROUP_5_ITEM_1 (SubGroups: 1)
    - Pattern: GROUP_6_ITEM_1|GROUP_3_ITEM_4|GROUP_5_ITEM_2 (SubGroups: 1)
    - Pattern: GROUP_6_ITEM_1|GROUP_3_ITEM_1|GROUP_5_ITEM_3 (SubGroups: 1)
  Unique PRODUCT_GROUP5 values: 3
==========================================
```

### Utility Functions

#### `getUniqueProductGroup5Values()`
Returns all unique PRODUCT_GROUP5 values from a GroupedData set. Useful for building table columns.

```go
uniqueColumns := getUniqueProductGroup5Values(groupedData)
// Result: ["GROUP_5_ITEM_1", "GROUP_5_ITEM_2", "GROUP_5_ITEM_3"]
```

#### `getPatternsByGroup3()`
Filters pattern groups by specific PRODUCT_GROUP3 value. Useful for building table rows.

```go
rowPatterns := getPatternsByGroup3(groupedData, "GROUP_3_ITEM_1")
// Result: Patterns where PRODUCT_GROUP3 = "GROUP_3_ITEM_1"
```

## 📦 Data Types

### `GroupedData`
Top-level structure representing one PRODUCT_GROUP2 set.

```go
type GroupedData struct {
    ProductGroup2 string         `json:"product_group_2"`
    PatternGroups []PatternGroup `json:"pattern_groups"`
}
```

### `PatternGroup`
Represents one unique pattern combination within a PRODUCT_GROUP2 set.

```go
type PatternGroup struct {
    Pattern       string     `json:"pattern"`
    ProductGroup6 string     `json:"product_group_6"`
    ProductGroup3 string     `json:"product_group_3"`
    ProductGroup5 string     `json:"product_group_5"`
    SubGroups     []SubGroup `json:"sub_groups"`
}
```

## 🚀 Usage

### API Endpoint
```
POST /api/price-service/price-table
```

### Request Body
```json
{
  "company_code": "7eb85b75-e708-4e5d-9010-4b43427c15be",
  "site_codes": ["PRM-00A"],
  "group_codes": ["GROUP_1_ITEM_1"]
}
```

### Response Structure
See `SAMPLE_OUTPUT_EXAMPLE.json` for complete example.

## 🎨 Frontend Integration

### Building Dynamic Tables

```javascript
// Example: Build table columns from grouped data
groupedData.forEach(set => {
  const columns = getUniqueProductGroup5Values(set);
  
  columns.forEach(group5Value => {
    // Create column for each unique PRODUCT_GROUP5
    createTableColumn({
      field: group5Value,
      headerName: group5Value
    });
  });
  
  set.pattern_groups.forEach(pattern => {
    // Create row with values from pattern.sub_groups
    createTableRow({
      group6: pattern.product_group_6,
      group3: pattern.product_group_3,
      data: pattern.sub_groups
    });
  });
});
```

### Creating Tabs

```javascript
// Example: Create tabs for each PRODUCT_GROUP2
groupedData.forEach((set, index) => {
  createTab({
    id: index,
    label: `Set: ${set.product_group_2}`,
    content: buildTableFromPatterns(set.pattern_groups)
  });
});
```

## 📝 Key Benefits

1. **Hierarchical Organization**: Clear two-level grouping structure
2. **Easy Navigation**: Filter and query by PRODUCT_GROUP2 or pattern
3. **Dynamic UI**: Easy to build tables, tabs, and other UI components
4. **Performance**: O(n) grouping algorithm with map-based lookups
5. **Scalability**: Works with any number of unique values
6. **Type Safety**: Strongly typed Go structures
7. **Debugging**: Built-in summary printing for development

## 📚 Related Files

- `get-price-table.go` - Main implementation
- `get-pricelist.go` - Data fetching from database
- `price.json` - Sample test data
- `GROUPING_PATTERN_EXPLANATION.md` - Detailed pattern explanation
- `SAMPLE_OUTPUT_EXAMPLE.json` - Output structure example

## 🔍 Testing

To test the implementation:

1. Ensure database connection is configured
2. Call the endpoint with valid company_code and site_codes
3. Check console output for grouping summary
4. Verify JSON response matches expected structure

## 💡 Future Enhancements

- Add sorting options for patterns and groups
- Implement filtering by multiple PRODUCT_GROUP values
- Add pagination for large datasets
- Cache grouped results for repeated queries
- Add aggregation functions (sum, average) per pattern group

---

**Author**: Price Service Team  
**Last Updated**: November 3, 2025  
**Version**: 1.0

