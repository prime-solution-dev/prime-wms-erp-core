# Price List Seed Script

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
go run ./internal/scripts/seed-price-list.go --group-id=<uuid> [OPTIONS]
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

- `price_unit`, `extra_price_unit`, `term_price_unit`, `total_net_price_unit`
- `price_weight`, `extra_price_weight`, `term_price_weight`, `total_net_price_weight`
- `before_price_unit`, `before_extra_price_unit`, `before_term_price_unit`, `before_total_net_price_unit`
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

