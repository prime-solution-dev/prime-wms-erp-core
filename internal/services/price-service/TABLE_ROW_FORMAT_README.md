# Table Row Format Implementation

## 🎯 Overview

This document explains the **Table Row Format** implementation that transforms grouped price data into a spreadsheet-like structure where:

- **Rows** = Unique combinations of `PRODUCT_GROUP6` + `PRODUCT_GROUP3`
- **Columns** = Different `PRODUCT_GROUP5` values
- **Each Row** has a unique ID and contains data for multiple columns

## 📊 Transformation Flow

```
Raw SubGroups
    ↓
Group by Pattern (GROUP6|GROUP3|GROUP5)
    ↓
Convert to Table Rows
    ↓
Table Structure with Rows and Columns
```

## 🏗️ Data Structures

### 1. TableStructure
Top-level structure representing one complete table (one per PRODUCT_GROUP2).

```go
type TableStructure struct {
    ProductGroup2 string         `json:"product_group_2"`
    UniqueGroup5  []string       `json:"unique_group_5_values"` // Column headers
    Rows          []TableRowData `json:"rows"`
}
```

**Example:**
```json
{
  "product_group_2": "GROUP_2_ITEM_2",
  "unique_group_5_values": ["GROUP_5_ITEM_1", "GROUP_5_ITEM_2", "GROUP_5_ITEM_3"],
  "rows": [...]
}
```

### 2. TableRowData
Represents a single row in the table.

```go
type TableRowData struct {
    ID            uuid.UUID              `json:"id"`
    ProductGroup2 string                 `json:"product_group_2"`
    ProductGroup6 string                 `json:"product_group_6"`
    ProductGroup3 string                 `json:"product_group_3"`
    Columns       map[string]ColumnValue `json:"columns"` // key = product_group_5
}
```

**Key Points:**
- Each row has a **unique ID** (UUID)
- **ProductGroup6 + ProductGroup3** identify the row
- **Columns** map contains data for each PRODUCT_GROUP5 value

**Example:**
```json
{
  "id": "a1b2c3d4-1234-5678-90ab-cdef12345678",
  "product_group_2": "GROUP_2_ITEM_2",
  "product_group_6": "GROUP_6_ITEM_1",
  "product_group_3": "GROUP_3_ITEM_1",
  "columns": {
    "GROUP_5_ITEM_1": {...},
    "GROUP_5_ITEM_3": {...}
  }
}
```

### 3. ColumnValue
Represents the data for a specific column (PRODUCT_GROUP5) within a row.

```go
type ColumnValue struct {
    ProductGroup5 string    `json:"product_group_5"`
    SubGroupID    uuid.UUID `json:"subgroup_id"`
    PriceUnit     float64   `json:"price_unit"`
    PriceWeight   float64   `json:"price_weight"`
    IsTrading     bool      `json:"is_trading"`
    SubGroupKey   string    `json:"subgroup_key"`
}
```

**Example:**
```json
{
  "product_group_5": "GROUP_5_ITEM_1",
  "subgroup_id": "5ed9008d-1e45-4664-bcd1-577fda93c547",
  "price_unit": 780,
  "price_weight": 26.9,
  "is_trading": false,
  "subgroup_key": "GROUP_1_ITEM_1|GROUP_2_ITEM_2|GROUP_5_ITEM_1|GROUP_6_ITEM_1|GROUP_3_ITEM_1"
}
```

## 🔄 Transformation Logic

### Function: `convertToTableRows()`

**Algorithm:**
```
For each PRODUCT_GROUP2 set:
  1. Create map: "GROUP6|GROUP3" → TableRowData
  2. For each pattern group:
     a. Create row key: "GROUP6|GROUP3"
     b. Initialize row if not exists
     c. Add column data for GROUP5
  3. Convert maps to arrays
  4. Return TableStructure
```

**Complexity:** O(n) where n = number of pattern groups

**Code Flow:**
```go
// Process each pattern
rowKey := "GROUP_6_ITEM_1|GROUP_3_ITEM_1"

// Initialize row
row := TableRowData{
    ID:            uuid.New(),
    ProductGroup6: "GROUP_6_ITEM_1",
    ProductGroup3: "GROUP_3_ITEM_1",
    Columns:       map[string]ColumnValue{},
}

// Add column data
row.Columns["GROUP_5_ITEM_1"] = ColumnValue{
    PriceUnit: 780,
    PriceWeight: 26.9,
    ...
}
```

## 📋 Sample Data Transformation

### Input (from price.json)
```
SubGroup 1: GROUP_6_ITEM_1 | GROUP_3_ITEM_1 | GROUP_5_ITEM_1 → Price: 780
SubGroup 2: GROUP_6_ITEM_1 | GROUP_3_ITEM_4 | GROUP_5_ITEM_2 → Price: 780
SubGroup 3: GROUP_6_ITEM_1 | GROUP_3_ITEM_1 | GROUP_5_ITEM_3 → Price: 780
```

### Output (Table Rows)
```
Table for PRODUCT_GROUP2: GROUP_2_ITEM_2

Row 1: GROUP_6_ITEM_1 | GROUP_3_ITEM_1
  - Columns: {GROUP_5_ITEM_1: data, GROUP_5_ITEM_3: data}

Row 2: GROUP_6_ITEM_1 | GROUP_3_ITEM_4
  - Columns: {GROUP_5_ITEM_2: data}
```

## 🎨 Visual Representation

### As Table
```
┌──────────────┬──────────────┬──────────────┬──────────────┬──────────────┐
│  Group 6     │  Group 3     │ GROUP_5_1    │ GROUP_5_2    │ GROUP_5_3    │
├──────────────┼──────────────┼──────────────┼──────────────┼──────────────┤
│ G6_ITEM_1    │ G3_ITEM_1    │  780 / 26.9  │      -       │  780 / 26.9  │
├──────────────┼──────────────┼──────────────┼──────────────┼──────────────┤
│ G6_ITEM_1    │ G3_ITEM_4    │      -       │  780 / 26.9  │      -       │
└──────────────┴──────────────┴──────────────┴──────────────┴──────────────┘
```

### As Spreadsheet
| Row ID | Product Group 6 | Product Group 3 | GROUP_5_ITEM_1 | GROUP_5_ITEM_2 | GROUP_5_ITEM_3 |
|--------|-----------------|-----------------|----------------|----------------|----------------|
| a1b2c3 | GROUP_6_ITEM_1  | GROUP_3_ITEM_1  | 780 / 26.9     | -              | 780 / 26.9     |
| b2c3d4 | GROUP_6_ITEM_1  | GROUP_3_ITEM_4  | -              | 780 / 26.9     | -              |

## 💻 Frontend Integration

### Example 1: React Table
```javascript
const TableView = ({ tableStructure }) => {
  // Create column definitions
  const columns = [
    { field: 'product_group_6', header: 'Product Group 6' },
    { field: 'product_group_3', header: 'Product Group 3' },
    ...tableStructure.unique_group_5_values.map(g5 => ({
      field: g5,
      header: g5,
      render: (row) => {
        const colValue = row.columns[g5];
        return colValue 
          ? `${colValue.price_unit} / ${colValue.price_weight}`
          : '-';
      }
    }))
  ];
  
  return (
    <DataTable 
      columns={columns} 
      data={tableStructure.rows} 
    />
  );
};
```

### Example 2: AG Grid
```javascript
const gridOptions = {
  columnDefs: [
    { field: 'product_group_6', headerName: 'Group 6', pinned: 'left' },
    { field: 'product_group_3', headerName: 'Group 3', pinned: 'left' },
    ...tableStructure.unique_group_5_values.map(g5 => ({
      field: g5,
      headerName: g5,
      valueGetter: (params) => {
        const colValue = params.data.columns[g5];
        return colValue ? colValue.price_unit : null;
      }
    }))
  ],
  rowData: tableStructure.rows
};
```

### Example 3: HTML Table
```javascript
function renderTable(tableStructure) {
  let html = '<table><thead><tr>';
  html += '<th>Group 6</th><th>Group 3</th>';
  
  tableStructure.unique_group_5_values.forEach(g5 => {
    html += `<th>${g5}</th>`;
  });
  
  html += '</tr></thead><tbody>';
  
  tableStructure.rows.forEach(row => {
    html += `<tr data-row-id="${row.id}">`;
    html += `<td>${row.product_group_6}</td>`;
    html += `<td>${row.product_group_3}</td>`;
    
    tableStructure.unique_group_5_values.forEach(g5 => {
      const colValue = row.columns[g5];
      html += colValue 
        ? `<td>${colValue.price_unit}</td>`
        : '<td>-</td>';
    });
    
    html += '</tr>';
  });
  
  html += '</tbody></table>';
  return html;
}
```

## 🔍 Console Output

When the API runs, you'll see:

```
========== TABLE STRUCTURE (ROW FORMAT) ==========

Table 1 - PRODUCT_GROUP2: GROUP_2_ITEM_2
  Unique PRODUCT_GROUP5 columns: 3 → [GROUP_5_ITEM_1 GROUP_5_ITEM_2 GROUP_5_ITEM_3]
  Total Rows: 2

  Row 1 [ID: a1b2c3d4]
    GROUP_6: GROUP_6_ITEM_1 | GROUP_3: GROUP_3_ITEM_1
    Columns (2):
      - GROUP_5_ITEM_1: price_unit=780.00, price_weight=26.90
      - GROUP_5_ITEM_3: price_unit=780.00, price_weight=26.90

  Row 2 [ID: b2c3d4e5]
    GROUP_6: GROUP_6_ITEM_1 | GROUP_3: GROUP_3_ITEM_4
    Columns (1):
      - GROUP_5_ITEM_2: price_unit=780.00, price_weight=26.90

==================================================
```

## ✨ Key Benefits

1. **✅ Unique Row IDs**: Each row has a UUID for tracking and updates
2. **✅ Flexible Columns**: Dynamically adapts to any number of GROUP_5 values
3. **✅ Sparse Data**: Missing values (no data for a GROUP_5) are simply not in the map
4. **✅ Easy Rendering**: Direct mapping to HTML tables, data grids, or spreadsheets
5. **✅ Efficient Lookup**: O(1) access to column data via map
6. **✅ Type Safe**: Strongly typed Go structures
7. **✅ Frontend Ready**: Structure designed for immediate use in UI frameworks

## 📝 Handling Missing Data

If a row doesn't have data for a specific PRODUCT_GROUP5:

**Backend (Go):**
```go
// Check if column exists
if colValue, exists := row.Columns["GROUP_5_ITEM_2"]; exists {
    fmt.Printf("Price: %.2f\n", colValue.PriceUnit)
} else {
    fmt.Println("No data")
}
```

**Frontend (JavaScript):**
```javascript
// Safely access column data
const colValue = row.columns["GROUP_5_ITEM_2"];
const displayValue = colValue ? colValue.price_unit : '-';
```

## 🎯 Use Cases

1. **Price List Tables**: Display prices in grid format
2. **Comparison Views**: Compare prices across different GROUP_5 values
3. **Data Entry Forms**: Allow editing of specific cells
4. **Excel Export**: Easy to convert to CSV/Excel format
5. **Pivot Tables**: Base structure for creating pivot views
6. **Reports**: Generate formatted price reports

## 🔄 Migration from Pattern Groups

If you prefer the old pattern group format, both formats are available:

```go
// Pattern Group Format (hierarchical)
groupedData := groupDataByPattern(res)

// Table Row Format (flat, spreadsheet-like)
tableStructures := convertToTableRows(groupedData)
```

The API currently returns the **Table Row Format**, but you can modify `GetPriceTable()` to return either or both formats based on your needs.

## 📚 Related Files

- `get-price-table.go` - Implementation
- `SAMPLE_TABLE_ROWS_OUTPUT.json` - Sample output
- `README_GROUPING.md` - Original pattern grouping documentation

---

**Last Updated**: November 3, 2025  
**Version**: 2.0 (Table Row Format)

