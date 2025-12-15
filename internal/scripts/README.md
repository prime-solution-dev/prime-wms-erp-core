# Price List Seed Scripts

This directory contains scripts for seeding price list data into the database.

## Directory Structure

```
scripts/
├── shared/                          # Shared utilities used by all scripts
│   ├── format.go                   # SQL formatting functions
│   ├── database.go                 # Database connection and execution
│   └── utils.go                     # Utility functions
├── price_list_formulas/             # Price list formulas seeding
│   ├── seed-price-list-formulas.go # Main script
│   └── price-list-formulas.json    # Formula data
└── price_list_sub_group/            # Price list sub group seeding
    ├── seed-price-list.go          # Main script
    ├── seed-price-list_test.go     # Tests
    └── *.json                       # Group configuration files
```

## Scripts

1. **price_list_sub_group/seed-price-list.go** - Generates SQL INSERT statements for `price_list_sub_group` and `price_list_sub_group_key` tables
2. **price_list_formulas/seed-price-list-formulas.go** - Seeds `price_list_formulas` table with formula definitions

---

# Price List Sub Group Seed Script

This script generates SQL INSERT statements for seeding price list data into the database. It creates records for `price_list_sub_group` and `price_list_sub_group_key` tables.

## Overview

The `seed-price-list` script generates test data for price list functionality. It can create multiple price list sub-groups with associated keys, generating random price values within specified ranges.

## Prerequisites

- Go installed and configured
- Access to a valid `price_list_group` record (you need its UUID)

## Usage

### Using Make (Recommended)

The easiest way to use the script is through the Makefile:

```bash
make seed-price-list GROUP_ID=<uuid> [OPTIONS]
```

### Direct Go Execution

You can also run the script directly:

```bash
go run ./internal/scripts/price_list_sub_group/seed-price-list.go --group-id=<uuid> [OPTIONS]
```

## Required Parameters

- `GROUP_ID` (or `--group-id`): The UUID of an existing `price_list_group` record. This is **required**.

## Optional Parameters

- `COUNT` (or `--count`)**: Number of `price_list_sub_group` records to generate (default: `10`)
- `PRICE_MIN` (or `--price-min`)**: Minimum price/value to generate (default: `0`)
- `PRICE_MAX` (or `--price-max`)**: Maximum price/value to generate (default: `1000`)
- `PRODUCT_GROUPS` (or `--product-groups`)**: Comma-separated product group definitions
- `GROUP_ITEMS` (or `--group-items`)**: JSON map/array or `@path` to JSON file describing items per product group
- `SUBGROUP_KEYS` (or `--subgroup-keys`)**: Comma-separated list of explicit subgroup_key values
- `SUBGROUP_KEY` (or `--subgroup-key`)**: Explicit subgroup_key value (can be repeated multiple times)
- `OUTPUT` (or `--output`)**: Optional output file path (defaults to stdout)
- `SEED` (or `--seed`)**: Seed for random generator (defaults to current timestamp)
- `EXECUTE` (or `--execute`)**: When set, apply the generated statements to the configured database
- `DATABASE` (or `--database`)**: Database suffix for `database_gorm_url_<suffix>` (default: `prime_erp`)

## Subgroup Key Generation

The script supports two methods for generating subgroup keys:

### Method 1: Using Product Groups

Product groups follow the pattern `PRODUCT_GROUP<N>` where `N` is a number. You can specify:

1. **Simple product group** (auto-generates items):
   ```
   PRODUCT_GROUP1,PRODUCT_GROUP2,PRODUCT_GROUP4
   ```
   This will generate random items like `GROUP_1_ITEM_1`, `GROUP_2_ITEM_5`, etc.

2. **Product group with specific items**:
   ```
   PRODUCT_GROUP1:GROUP_1_ITEM_1|GROUP_1_ITEM_2,PRODUCT_GROUP2:GROUP_2_ITEM_1
   ```
   This allows you to specify exactly which items to use for each product group.

### Method 2: Using Explicit Subgroup Keys

You can provide explicit subgroup keys directly:

```bash
--subgroup-keys="GROUP_1_ITEM_3|GROUP_2_ITEM_1|GROUP_4_ITEM_2,GROUP_1_ITEM_1|GROUP_2_ITEM_2"
```

Or use the repeatable flag:

```bash
--subgroup-key="GROUP_1_ITEM_3|GROUP_2_ITEM_1|GROUP_4_ITEM_2"
--subgroup-key="GROUP_1_ITEM_1|GROUP_2_ITEM_2"
```

**Note**: You must provide either `--product-groups` or `--subgroup-key`/`--subgroup-keys` (at least one is required).

## Examples

### Basic Usage

Generate 10 records with default settings:

```bash
make seed-price-list GROUP_ID=550e8400-e29b-41d4-a716-446655440000
```

### Custom Count and Price Range

Generate 5 records with prices between 100 and 500:

```bash
make seed-price-list GROUP_ID=550e8400-e29b-41d4-a716-446655440000 \
  COUNT=5 \
  PRICE_MIN=100 \
  PRICE_MAX=500
```

### Using Product Groups

Generate records using product groups with specific items:

```bash
make seed-price-list GROUP_ID=550e8400-e29b-41d4-a716-446655440000 \
  PRODUCT_GROUPS="PRODUCT_GROUP1:GROUP_1_ITEM_1|GROUP_1_ITEM_2,PRODUCT_GROUP2:GROUP_2_ITEM_1"
```

### Using Explicit Subgroup Keys

Generate records with explicit subgroup keys:

```bash
make seed-price-list GROUP_ID=550e8400-e29b-41d4-a716-446655440000 \
  SUBGROUP_KEYS="GROUP_1_ITEM_3|GROUP_2_ITEM_1|GROUP_4_ITEM_2,GROUP_1_ITEM_1|GROUP_2_ITEM_2"
```

### Using Group Items JSON File

Create a JSON file `groups.json` using either map or array syntax.

**Object map syntax (existing behavior):**
```json
{
  "PRODUCT_GROUP1": ["GROUP_1_ITEM_1", "GROUP_1_ITEM_2"],
  "PRODUCT_GROUP2": ["GROUP_2_ITEM_1", "GROUP_2_ITEM_2"]
}
```

**Array syntax (new):**
```json
[
  {
    "PRODUCT_GROUP1": ["GROUP_1_ITEM_4"],
    "PRODUCT_GROUP2": ["GROUP_2_ITEM_6"],
    "PRODUCT_GROUP3": ["GROUP_3_ITEM_9"],
    "PRODUCT_GROUP4": ["GROUP_4_ITEM_9"],
    "PRODUCT_GROUP5": ["GROUP_5_ITEM_10"],
    "PRODUCT_GROUP6": ["GROUP_6_ITEM_2"]
  },
  {
    "PRODUCT_GROUP1": ["GROUP_1_ITEM_4"],
    "PRODUCT_GROUP2": ["GROUP_2_ITEM_6"],
    "PRODUCT_GROUP3": ["GROUP_3_ITEM_9"],
    "PRODUCT_GROUP4": ["GROUP_4_ITEM_9"],
    "PRODUCT_GROUP5": ["GROUP_5_ITEM_11"],
    "PRODUCT_GROUP6": ["GROUP_6_ITEM_2"]
  }
]
```

Then use it:

```bash
make seed-price-list GROUP_ID=550e8400-e29b-41d4-a716-446655440000 \
  GROUP_ITEMS="@groups.json"
```

Or inline JSON:

```bash
make seed-price-list GROUP_ID=550e8400-e29b-41d4-a716-446655440000 \
  GROUP_ITEMS='{"PRODUCT_GROUP1":["GROUP_1_ITEM_1"],"PRODUCT_GROUP2":["GROUP_2_ITEM_1"]}'
```

or with array syntax:

```bash
make seed-price-list GROUP_ID=550e8400-e29b-41d4-a716-446655440000 \
  GROUP_ITEMS='[{"code":"PRODUCT_GROUP1","items":["GROUP_1_ITEM_1"]}]'
```

### Execute Against Database

By default the script prints SQL, but it can also apply the data directly to your database:

1. Export a connection string using the naming pattern `database_gorm_url_<NAME>`. Example:
   ```bash
   export database_gorm_url_prime_erp="postgres://user:pass@localhost:5432/prime?sslmode=disable"
   ```
2. Run the script with `--execute` (and optionally `--database <NAME>` if you used a different suffix):
   ```bash
   make seed-price-list GROUP_ID=550e8400-e29b-41d4-a716-446655440000 \
     PRODUCT_GROUPS="PRODUCT_GROUP1,PRODUCT_GROUP2" \
     EXECUTE=true \
     DATABASE=prime_erp
   ```

The script opens a GORM connection, runs all inserts inside a single transaction, and rolls back if any statement fails. A short confirmation message is printed to stderr after a successful execution.

### Save Output to File

Save the generated SQL to a file:

```bash
make seed-price-list GROUP_ID=550e8400-e29b-41d4-a716-446655440000 \
  OUTPUT=seed.sql
```

### Reproducible Results with Seed

Use a fixed seed for reproducible results:

```bash
make seed-price-list GROUP_ID=550e8400-e29b-41d4-a716-446655440000 \
  SEED=12345
```

## Output Format

The script generates SQL INSERT statements in the following format:

```sql
-- Auto-generated seed (2024-01-15T10:30:00Z)
INSERT INTO public.price_list_sub_group (id, price_list_group_id, subgroup_key, ...) VALUES (...);
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq) VALUES (...);
INSERT INTO public.price_list_sub_group_key (id, sub_group_id, code, value, seq) VALUES (...);
...
```

Each `price_list_sub_group` record will have multiple `price_list_sub_group_key` records associated with it, one for each segment in the subgroup_key (separated by `|`).

## Generated Fields

The script generates the following fields with random values (within the specified price range):

- `price_unit`, `extra_price_unit`, `total_net_price_unit`
- `price_weight`, `extra_price_weight`, `term_price_weight`, `total_net_price_weight`
- `before_price_unit`, `before_extra_price_unit`, `before_total_net_price_unit`
- `before_price_weight`, `before_extra_price_weight`, `before_term_price_weight`, `before_total_net_price_weight`
- `is_trading` (randomly true or false)

## Testing

Run the test suite:

```bash
make seed-price-list-test
```

Or directly:

```bash
go test ./internal/scripts -run Test
```

The tests verify:
- Correct SQL statement generation
- Proper subgroup key parsing
- SQL execution against a test database
- Data validation

## Error Handling

The script will exit with an error if:

- `GROUP_ID` is not provided or is invalid
- `COUNT` is zero or negative
- `PRICE_MIN` or `PRICE_MAX` is negative
- `PRICE_MIN` is greater than `PRICE_MAX`
- Neither product groups nor subgroup keys are provided
- Invalid subgroup key format
- Invalid JSON in group-items
- File read errors (for `@path` group-items)

## Notes

- Subgroup keys are generated by joining product group items with `|` separator
- When using explicit keys, they are cycled if `COUNT` exceeds the number of provided keys
- Product groups are automatically sorted by code
- All prices are formatted to 2 decimal places
- UUIDs are automatically generated for each record
- The script escapes single quotes in string values

---

# Price List Formulas Seed Script

This script seeds the `price_list_formulas` table with formula definitions used for price calculations.

## Overview

The `seed-price-list-formulas` script reads formula definitions from a JSON file and generates SQL INSERT statements to populate the `price_list_formulas` table. It can also execute the statements directly against the database.

## Prerequisites

- Go installed and configured
- JSON file containing formula definitions (default: `price-list-formulas.json`)

## Usage

### Direct Go Execution

Run the script directly:

```bash
cd internal/scripts/price_list_formulas
go run seed-price-list-formulas.go [OPTIONS]
```

Or from the project root:

```bash
go run ./internal/scripts/price_list_formulas/seed-price-list-formulas.go [OPTIONS]
```

### Generate SQL Only (Default)

By default, the script outputs SQL statements to stdout:

```bash
go run ./internal/scripts/price_list_formulas/seed-price-list-formulas.go
```

### Save SQL to File

Save the generated SQL to a file:

```bash
go run ./internal/scripts/price_list_formulas/seed-price-list-formulas.go --output=formulas.sql
```

### Execute Against Database

To apply the data directly to your database:

1. Export a connection string using the naming pattern `database_gorm_url_<NAME>`. Example:
   ```bash
   export database_gorm_url_prime_erp="postgres://user:pass@localhost:5432/prime?sslmode=disable"
   ```

2. Run the script with `--execute`:
   ```bash
   go run ./internal/scripts/price_list_formulas/seed-price-list-formulas.go --execute --database=prime_erp
   ```

Or use a direct connection string:

```bash
go run ./internal/scripts/price_list_formulas/seed-price-list-formulas.go --execute \
  --connection-string="postgres://user:pass@localhost:5432/prime?sslmode=disable"
```

## Optional Parameters

- `--input`: Path to JSON file containing formulas (default: `price-list-formulas.json`)
- `--output`: Optional output file path for SQL statements (defaults to stdout)
- `--execute`: Execute generated statements against the configured database (default: `false`)
- `--database`: Database suffix for `database_gorm_url_<suffix>` (default: `prime_erp`)
- `--connection-string`: Direct database connection string (overrides database name/env variable)

## JSON File Format

The JSON file should follow this structure:

```json
{
  "price_list_formulas": [
    {
      "id": "57eda1d0-49c6-4e19-b48d-80c83f33256b",
      "name": "kg = [Pcs] / [Avg. kg stock]",
      "uom": "kg",
      "formula_type": "price_calc",
      "expression": " pcs / avg_kg_stock ",
      "params": "{\"required\": [\"pcs\", \"avg_kg_stock\"], \"description\": \"kg = [Pcs] / [Avg. kg stock]\"}",
      "rounding": 2,
      "create_dtm": "2025-12-04T06:10:58.461Z"
    },
    {
      "id": "c2748d23-06ca-4231-bae2-d7cc29a76fd3",
      "name": "pcs = input",
      "uom": "pcs",
      "formula_type": "input",
      "expression": null,
      "params": null,
      "rounding": 2,
      "create_dtm": "2025-12-04T06:10:58.461Z"
    }
  ]
}
```

### Field Descriptions

- `id`: UUID string (required)
- `name`: Formula name/description (required)
- `uom`: Unit of measure (e.g., "kg", "pcs") (required)
- `formula_type`: Type of formula - "price_calc" or "input" (required)
- `expression`: Formula expression (can be `null` for "input" type formulas)
- `params`: JSON string containing formula parameters (can be `null` for "input" type formulas)
- `rounding`: Number of decimal places for rounding (required)
- `create_dtm`: ISO 8601 timestamp string (required)

## Examples

### Basic Usage (Output SQL)

```bash
go run ./internal/scripts/price_list_formulas/seed-price-list-formulas.go
```

### Custom Input File

```bash
go run ./internal/scripts/price_list_formulas/seed-price-list-formulas.go --input=my-formulas.json
```

### Execute and Save SQL

```bash
go run ./internal/scripts/price_list_formulas/seed-price-list-formulas.go --execute --output=formulas.sql
```

### Using Custom Database Connection

```bash
go run ./internal/scripts/price_list_formulas/seed-price-list-formulas.go --execute \
  --connection-string="postgres://user:pass@localhost:5432/mydb?sslmode=disable"
```

## Output Format

The script generates SQL INSERT statements in the following format:

```sql
-- Auto-generated seed for price_list_formulas (2024-01-15T10:30:00Z)
INSERT INTO public.price_list_formulas (id, name, uom, formula_type, expression, params, rounding, create_dtm) 
VALUES ('57eda1d0-49c6-4e19-b48d-80c83f33256b'::uuid, 'kg = [Pcs] / [Avg. kg stock]', 'kg', 'price_calc', ' pcs / avg_kg_stock ', '{"required": ["pcs", "avg_kg_stock"], "description": "kg = [Pcs] / [Avg. kg stock]"}'::jsonb, 2, '2025-12-04T06:10:58.461Z'::timestamp) 
ON CONFLICT (id) DO NOTHING;
```

The script uses `ON CONFLICT (id) DO NOTHING` to prevent duplicate insertions if the script is run multiple times.

## Error Handling

The script will exit with an error if:

- Input file cannot be read
- JSON parsing fails
- Invalid UUID format in data
- Invalid date format in `create_dtm`
- Database connection fails (when using `--execute`)
- SQL execution fails (when using `--execute`)

## Notes

- Null `expression` and `params` values (for "input" type formulas) are converted to empty string and empty JSON object `{}` respectively
- Single quotes in string values are automatically escaped
- All inserts are executed within a single transaction when using `--execute`
- The script will skip invalid UUIDs and continue processing other records
- Invalid dates default to the current UTC time

