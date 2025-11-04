# Visual Example: Table Row Format

## 🎯 Understanding the Transformation

### Step 1: Raw Data (from price.json)

```
Record 1:
  GROUP_2: GROUP_2_ITEM_2
  GROUP_6: GROUP_6_ITEM_1
  GROUP_3: GROUP_3_ITEM_1
  GROUP_5: GROUP_5_ITEM_1
  Price: 780 / 26.9

Record 2:
  GROUP_2: GROUP_2_ITEM_2
  GROUP_6: GROUP_6_ITEM_1
  GROUP_3: GROUP_3_ITEM_4
  GROUP_5: GROUP_5_ITEM_2
  Price: 780 / 26.9

Record 3:
  GROUP_2: GROUP_2_ITEM_2
  GROUP_6: GROUP_6_ITEM_1
  GROUP_3: GROUP_3_ITEM_1
  GROUP_5: GROUP_5_ITEM_3
  Price: 780 / 26.9
```

### Step 2: Identify Row Combinations

**Row Key = GROUP_6 + GROUP_3**

```
Row A: GROUP_6_ITEM_1 + GROUP_3_ITEM_1
  ├─ Has: GROUP_5_ITEM_1 (Record 1)
  └─ Has: GROUP_5_ITEM_3 (Record 3)

Row B: GROUP_6_ITEM_1 + GROUP_3_ITEM_4
  └─ Has: GROUP_5_ITEM_2 (Record 2)
```

### Step 3: Create Table Structure

```
Table for GROUP_2_ITEM_2
├─ Columns: [GROUP_5_ITEM_1, GROUP_5_ITEM_2, GROUP_5_ITEM_3]
├─ Row A (ID: a1b2c3d4)
│  ├─ GROUP_6: GROUP_6_ITEM_1
│  ├─ GROUP_3: GROUP_3_ITEM_1
│  └─ Data:
│     ├─ GROUP_5_ITEM_1: ✓ (780 / 26.9)
│     ├─ GROUP_5_ITEM_2: ✗ (no data)
│     └─ GROUP_5_ITEM_3: ✓ (780 / 26.9)
└─ Row B (ID: b2c3d4e5)
   ├─ GROUP_6: GROUP_6_ITEM_1
   ├─ GROUP_3: GROUP_3_ITEM_4
   └─ Data:
      ├─ GROUP_5_ITEM_1: ✗ (no data)
      ├─ GROUP_5_ITEM_2: ✓ (780 / 26.9)
      └─ GROUP_5_ITEM_3: ✗ (no data)
```

## 📊 As a Spreadsheet

### Table View

```
┌──────────────────────┬──────────────────────┬──────────────────────┬──────────────────────┬──────────────────────┐
│                      │                      │  PRODUCT_GROUP_5     │  PRODUCT_GROUP_5     │  PRODUCT_GROUP_5     │
│  PRODUCT_GROUP_6     │  PRODUCT_GROUP_3     │  GROUP_5_ITEM_1      │  GROUP_5_ITEM_2      │  GROUP_5_ITEM_3      │
├──────────────────────┼──────────────────────┼──────────────────────┼──────────────────────┼──────────────────────┤
│  GROUP_6_ITEM_1      │  GROUP_3_ITEM_1      │     780 / 26.9       │         -            │     780 / 26.9       │
├──────────────────────┼──────────────────────┼──────────────────────┼──────────────────────┼──────────────────────┤
│  GROUP_6_ITEM_1      │  GROUP_3_ITEM_4      │         -            │     780 / 26.9       │         -            │
└──────────────────────┴──────────────────────┴──────────────────────┴──────────────────────┴──────────────────────┘
```

### Excel-like View

| Row | ID       | Group 6       | Group 3       | ITEM_1 (price/weight) | ITEM_2 (price/weight) | ITEM_3 (price/weight) |
|-----|----------|---------------|---------------|-----------------------|-----------------------|-----------------------|
| 1   | a1b2c3d4 | GROUP_6_ITEM_1| GROUP_3_ITEM_1| 780 / 26.9            | -                     | 780 / 26.9            |
| 2   | b2c3d4e5 | GROUP_6_ITEM_1| GROUP_3_ITEM_4| -                     | 780 / 26.9            | -                     |

## 🔄 Comparing Formats

### Old Format (Pattern Groups) - Hierarchical

```json
{
  "product_group_2": "GROUP_2_ITEM_2",
  "pattern_groups": [
    {
      "pattern": "GROUP_6_ITEM_1|GROUP_3_ITEM_1|GROUP_5_ITEM_1",
      "product_group_6": "GROUP_6_ITEM_1",
      "product_group_3": "GROUP_3_ITEM_1",
      "product_group_5": "GROUP_5_ITEM_1",
      "sub_groups": [...]
    },
    {
      "pattern": "GROUP_6_ITEM_1|GROUP_3_ITEM_1|GROUP_5_ITEM_3",
      "product_group_6": "GROUP_6_ITEM_1",
      "product_group_3": "GROUP_3_ITEM_1",
      "product_group_5": "GROUP_5_ITEM_3",
      "sub_groups": [...]
    },
    {
      "pattern": "GROUP_6_ITEM_1|GROUP_3_ITEM_4|GROUP_5_ITEM_2",
      ...
    }
  ]
}
```

**Characteristics:**
- ✅ Each pattern is separate
- ✅ Easy to iterate through all patterns
- ❌ Harder to build table rows
- ❌ Need to manually group by GROUP_6 + GROUP_3

### New Format (Table Rows) - Flat/Tabular

```json
{
  "product_group_2": "GROUP_2_ITEM_2",
  "unique_group_5_values": ["GROUP_5_ITEM_1", "GROUP_5_ITEM_2", "GROUP_5_ITEM_3"],
  "rows": [
    {
      "id": "a1b2c3d4-1234-5678-90ab-cdef12345678",
      "product_group_6": "GROUP_6_ITEM_1",
      "product_group_3": "GROUP_3_ITEM_1",
      "columns": {
        "GROUP_5_ITEM_1": { "price_unit": 780, "price_weight": 26.9, ... },
        "GROUP_5_ITEM_3": { "price_unit": 780, "price_weight": 26.9, ... }
      }
    },
    {
      "id": "b2c3d4e5-2345-6789-01bc-def123456789",
      "product_group_6": "GROUP_6_ITEM_1",
      "product_group_3": "GROUP_3_ITEM_4",
      "columns": {
        "GROUP_5_ITEM_2": { "price_unit": 780, "price_weight": 26.9, ... }
      }
    }
  ]
}
```

**Characteristics:**
- ✅ Direct mapping to table rows
- ✅ Each row has unique ID
- ✅ Easy to render as table/grid
- ✅ Grouped by row identifier (GROUP_6 + GROUP_3)
- ✅ Dynamic columns based on GROUP_5 values

## 💡 Real World Example

Imagine a price list for steel products:

### Raw Data
```
Product: Steel Plate, Type: Standard, Thickness: 10mm → Price: $780
Product: Steel Plate, Type: Standard, Thickness: 15mm → Price: $890
Product: Steel Plate, Type: Premium, Thickness: 10mm → Price: $820
```

### Transformed to Table Rows
```
Columns: [10mm, 15mm]

Row 1: Steel Plate | Standard
  - 10mm: $780
  - 15mm: $890

Row 2: Steel Plate | Premium
  - 10mm: $820
  - 15mm: (no data)
```

### As Table
```
┌─────────────┬───────────┬─────────┬─────────┐
│  Product    │  Type     │  10mm   │  15mm   │
├─────────────┼───────────┼─────────┼─────────┤
│ Steel Plate │ Standard  │  $780   │  $890   │
├─────────────┼───────────┼─────────┼─────────┤
│ Steel Plate │ Premium   │  $820   │    -    │
└─────────────┴───────────┴─────────┴─────────┘
```

## 🎨 Frontend Rendering Example

### JavaScript Code
```javascript
// Given tableStructure from API
const table = tableStructure.rows;

// Render table
console.table(
  table.map(row => {
    const rowData = {
      'Group 6': row.product_group_6,
      'Group 3': row.product_group_3
    };
    
    // Add columns dynamically
    tableStructure.unique_group_5_values.forEach(g5 => {
      const colValue = row.columns[g5];
      rowData[g5] = colValue 
        ? `${colValue.price_unit} / ${colValue.price_weight}`
        : '-';
    });
    
    return rowData;
  })
);
```

### Output in Browser Console
```
┌─────────┬────────────────┬────────────────┬──────────────┬──────────────┬──────────────┐
│ (index) │    Group 6     │    Group 3     │ GROUP_5_1    │ GROUP_5_2    │ GROUP_5_3    │
├─────────┼────────────────┼────────────────┼──────────────┼──────────────┼──────────────┤
│    0    │'GROUP_6_ITEM_1'│'GROUP_3_ITEM_1'│'780 / 26.9'  │     '-'      │'780 / 26.9'  │
│    1    │'GROUP_6_ITEM_1'│'GROUP_3_ITEM_4'│     '-'      │'780 / 26.9'  │     '-'      │
└─────────┴────────────────┴────────────────┴──────────────┴──────────────┴──────────────┘
```

## 🔍 Key Insights

### 1. Row Identification
```
Row Key = GROUP_6 + GROUP_3
Example: "GROUP_6_ITEM_1|GROUP_3_ITEM_1"
```

### 2. Column Identification
```
Columns = All unique GROUP_5 values
Example: ["GROUP_5_ITEM_1", "GROUP_5_ITEM_2", "GROUP_5_ITEM_3"]
```

### 3. Cell Value
```
Cell = Row[Column]
Example: Row["GROUP_6_ITEM_1|GROUP_3_ITEM_1"]["GROUP_5_ITEM_1"] = {price: 780, weight: 26.9}
```

## ✅ Benefits Summary

| Feature | Pattern Groups | Table Rows |
|---------|---------------|------------|
| Hierarchical structure | ✅ Yes | ❌ No (flat) |
| Direct table rendering | ❌ Complex | ✅ Simple |
| Unique row IDs | ❌ No | ✅ Yes |
| Dynamic columns | ❌ Manual | ✅ Automatic |
| Sparse data support | ❌ Hard | ✅ Easy |
| Excel/CSV export | ❌ Complex | ✅ Simple |
| Frontend integration | ⚠️ Moderate | ✅ Easy |

## 🎯 When to Use Table Row Format

✅ **Use Table Rows when:**
- Building data grids or tables
- Need direct Excel/CSV export
- Want unique row identifiers
- Dynamic column structure
- Spreadsheet-like UI

⚠️ **Use Pattern Groups when:**
- Need hierarchical data
- Complex filtering on patterns
- Tree-like data structure
- Nested grouping logic

---

**Tip:** The implementation provides both formats. Choose based on your UI requirements!

