package priceService

/*
DATA GROUPING IMPLEMENTATION
============================

This file implements a hierarchical data grouping system for price lists based on Product Groups.

GROUPING STRATEGY:
-----------------
1. First Level: Group by PRODUCT_GROUP2 (creates distinct data sets)
2. Second Level: Within each PRODUCT_GROUP2, group by pattern: PRODUCT_GROUP6|PRODUCT_GROUP3|PRODUCT_GROUP5
3. Focus: Unique PRODUCT_GROUP5 values represent column variations in the table

TABLE STRUCTURE VISUALIZATION:
------------------------------
Based on the image reference, the table structure looks like:

| Product Group 6 | Product Group 3 | Product Group 5 | Product Group 5 | Product Group 5 | ... |
|-----------------|-----------------|-----------------|-----------------|-----------------|-----|
|       G6        |       G3        |   Headerxx      |   Headerxx      |   Headerxx      | ... |
|       G6        |      G31        |       xx        |       xx        |       xx        | ... |
|       G6        |      G32        |       xx        |       xx        |       xx        | ... |

Each unique PRODUCT_GROUP5 value becomes a column, grouped under PRODUCT_GROUP6 and PRODUCT_GROUP3 headers.

EXAMPLE DATA FLOW:
-----------------
Input SubGroups:
- SubGroup 1: GROUP_2_ITEM_2 | GROUP_6_ITEM_1 | GROUP_3_ITEM_1 | GROUP_5_ITEM_1
- SubGroup 2: GROUP_2_ITEM_2 | GROUP_6_ITEM_1 | GROUP_3_ITEM_4 | GROUP_5_ITEM_2
- SubGroup 3: GROUP_2_ITEM_1 | GROUP_6_ITEM_1 | GROUP_3_ITEM_1 | GROUP_5_ITEM_1

Output Grouped Data:
Set 1 (PRODUCT_GROUP2: GROUP_2_ITEM_2):
  - Pattern 1: GROUP_6_ITEM_1|GROUP_3_ITEM_1|GROUP_5_ITEM_1 → [SubGroup 1]
  - Pattern 2: GROUP_6_ITEM_1|GROUP_3_ITEM_4|GROUP_5_ITEM_2 → [SubGroup 2]

Set 2 (PRODUCT_GROUP2: GROUP_2_ITEM_1):
  - Pattern 1: GROUP_6_ITEM_1|GROUP_3_ITEM_1|GROUP_5_ITEM_1 → [SubGroup 3]
*/

import (
	"encoding/json"
	"errors"
	"fmt"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PriceListDetailApiResponse represents the main API response structure
type PriceListDetailApiResponse struct {
	Id   uuid.UUID                  `json:"id"`
	Name string                     `json:"name"`
	Tabs []PriceListDetailTabConfig `json:"tabs"`
}

// PriceListDetailTabConfig represents a tab configuration with table config and data
type PriceListDetailTabConfig struct {
	ID          uuid.UUID                `json:"id"`
	Label       string                   `json:"label"`
	TableConfig TableConfig              `json:"tableConfig"`
	TableData   []map[string]interface{} `json:"tableData"`
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
	SubGroups     []SubGroup `json:"sub_groups"`
}

// TableRowData represents a single row in the table format
type TableRowData struct {
	ID            uuid.UUID              `json:"id"`
	ProductGroup2 string                 `json:"product_group_2"`
	ProductGroup6 string                 `json:"product_group_6"`
	ProductGroup3 string                 `json:"product_group_3"`
	Columns       map[string]ColumnValue `json:"columns"` // key = product_group_5 value
}

// ColumnValue represents the data for each PRODUCT_GROUP5 column
type ColumnValue struct {
	ProductGroup5 string    `json:"product_group_5"`
	SubGroupID    uuid.UUID `json:"subgroup_id"`
	PriceUnit     float64   `json:"price_unit"`
	PriceWeight   float64   `json:"price_weight"`
	IsTrading     bool      `json:"is_trading"`
	SubGroupKey   string    `json:"subgroup_key"`
}

// TableStructure represents the complete table structure with rows
type TableStructure struct {
	ProductGroup2 string         `json:"product_group_2"`
	UniqueGroup5  []string       `json:"unique_group_5_values"` // All unique PRODUCT_GROUP5 for columns
	Rows          []TableRowData `json:"rows"`
}

// Helper function to get group key value by code
func getGroupKeyValue(groupKeys []GroupKey, code string) string {
	for _, gk := range groupKeys {
		if gk.Code == code {
			return gk.Value
		}
	}
	return ""
}

// getUniqueProductGroup5Values returns all unique PRODUCT_GROUP5 values from grouped data
func getUniqueProductGroup5Values(groupedData GroupedData) []string {
	uniqueMap := make(map[string]bool)
	result := []string{}

	for _, pattern := range groupedData.PatternGroups {
		if !uniqueMap[pattern.ProductGroup5] {
			uniqueMap[pattern.ProductGroup5] = true
			result = append(result, pattern.ProductGroup5)
		}
	}

	return result
}

// getPatternsByGroup3 returns pattern groups filtered by PRODUCT_GROUP3 value
func getPatternsByGroup3(groupedData GroupedData, group3Value string) []PatternGroup {
	result := []PatternGroup{}

	for _, pattern := range groupedData.PatternGroups {
		if pattern.ProductGroup3 == group3Value {
			result = append(result, pattern)
		}
	}

	return result
}

// convertToTableRows converts grouped data into table row format
// Each row represents a unique combination of PRODUCT_GROUP6 + PRODUCT_GROUP3
// Columns represent different PRODUCT_GROUP5 values
func convertToTableRows(groupedData []GroupedData) []TableStructure {
	tableStructures := []TableStructure{}

	for _, group := range groupedData {
		// Map to hold rows: key = "GROUP6|GROUP3"
		rowMap := make(map[string]*TableRowData)
		uniqueGroup5Map := make(map[string]bool)

		// Process each pattern group
		for _, pattern := range group.PatternGroups {
			rowKey := fmt.Sprintf("%s|%s", pattern.ProductGroup6, pattern.ProductGroup3)

			// Track unique GROUP5 values for column headers
			uniqueGroup5Map[pattern.ProductGroup5] = true

			// Initialize row if not exists
			if _, exists := rowMap[rowKey]; !exists {
				rowMap[rowKey] = &TableRowData{
					ID:            uuid.New(),
					ProductGroup2: group.ProductGroup2,
					ProductGroup6: pattern.ProductGroup6,
					ProductGroup3: pattern.ProductGroup3,
					Columns:       make(map[string]ColumnValue),
				}
			}

			// Add column data for this GROUP5
			// If multiple subgroups have same pattern, take the first one
			if len(pattern.SubGroups) > 0 {
				subGroup := pattern.SubGroups[0]
				rowMap[rowKey].Columns[pattern.ProductGroup5] = ColumnValue{
					ProductGroup5: pattern.ProductGroup5,
					SubGroupID:    subGroup.ID,
					PriceUnit:     subGroup.PriceUnit,
					PriceWeight:   subGroup.PriceWeight,
					IsTrading:     subGroup.IsTrading,
					SubGroupKey:   subGroup.SubGroupKey,
				}
			}
		}

		// Convert maps to slices
		uniqueGroup5List := []string{}
		for g5 := range uniqueGroup5Map {
			uniqueGroup5List = append(uniqueGroup5List, g5)
		}

		rows := []TableRowData{}
		for _, row := range rowMap {
			rows = append(rows, *row)
		}

		tableStructures = append(tableStructures, TableStructure{
			ProductGroup2: group.ProductGroup2,
			UniqueGroup5:  uniqueGroup5List,
			Rows:          rows,
		})
	}

	return tableStructures
}

// printTableStructure prints the table structure in a readable format
func printTableStructure(tables []TableStructure) {
	fmt.Println("\n========== TABLE STRUCTURE (ROW FORMAT) ==========")

	for i, table := range tables {
		fmt.Printf("\nTable %d - PRODUCT_GROUP2: %s\n", i+1, table.ProductGroup2)
		fmt.Printf("  Unique PRODUCT_GROUP5 columns: %d → %v\n", len(table.UniqueGroup5), table.UniqueGroup5)
		fmt.Printf("  Total Rows: %d\n\n", len(table.Rows))

		// Print each row
		for j, row := range table.Rows {
			fmt.Printf("  Row %d [ID: %s]\n", j+1, row.ID.String()[:8])
			fmt.Printf("    GROUP_6: %s | GROUP_3: %s\n", row.ProductGroup6, row.ProductGroup3)
			fmt.Printf("    Columns (%d):\n", len(row.Columns))

			for group5, colValue := range row.Columns {
				fmt.Printf("      - %s: price_unit=%.2f, price_weight=%.2f\n",
					group5, colValue.PriceUnit, colValue.PriceWeight)
			}
			fmt.Println()
		}
	}
	fmt.Println("==================================================")
}

// printGroupingSummary prints a summary of the grouped data
func printGroupingSummary(groupedData []GroupedData) {
	fmt.Println("\n========== DATA GROUPING SUMMARY ==========")
	fmt.Printf("Total unique PRODUCT_GROUP2 sets: %d\n\n", len(groupedData))

	for i, group := range groupedData {
		fmt.Printf("Set %d - PRODUCT_GROUP2: %s\n", i+1, group.ProductGroup2)
		fmt.Printf("  Unique patterns (PRODUCT_GROUP6|PRODUCT_GROUP3|PRODUCT_GROUP5): %d\n", len(group.PatternGroups))

		// Count unique PRODUCT_GROUP5 values
		uniqueGroup5 := make(map[string]bool)
		for _, pattern := range group.PatternGroups {
			uniqueGroup5[pattern.ProductGroup5] = true
			fmt.Printf("    - Pattern: %s (SubGroups: %d)\n", pattern.Pattern, len(pattern.SubGroups))
		}
		fmt.Printf("  Unique PRODUCT_GROUP5 values: %d\n", len(uniqueGroup5))
		fmt.Println()
	}
	fmt.Println("==========================================")
}

// groupDataByPattern groups subgroups by PRODUCT_GROUP2, then by pattern PRODUCT_GROUP6|PRODUCT_GROUP3|PRODUCT_GROUP5
func groupDataByPattern(responses []GetPriceListGroupResponse) []GroupedData {
	// Map to hold data grouped by PRODUCT_GROUP2
	group2Map := make(map[string]map[string]*PatternGroup)

	// Loop through all responses and their subgroups
	for _, resp := range responses {
		for _, subGroup := range resp.SubGroups {
			// Extract group values
			group2 := getGroupKeyValue(subGroup.GroupKeys, "PRODUCT_GROUP2")
			group3 := getGroupKeyValue(subGroup.GroupKeys, "PRODUCT_GROUP3")
			group5 := getGroupKeyValue(subGroup.GroupKeys, "PRODUCT_GROUP5")
			group6 := getGroupKeyValue(subGroup.GroupKeys, "PRODUCT_GROUP6")

			// Create pattern: PRODUCT_GROUP6|PRODUCT_GROUP3|PRODUCT_GROUP5
			pattern := fmt.Sprintf("%s|%s|%s", group6, group3, group5)

			// Initialize map for PRODUCT_GROUP2 if not exists
			if _, exists := group2Map[group2]; !exists {
				group2Map[group2] = make(map[string]*PatternGroup)
			}

			// Initialize PatternGroup if not exists
			if _, exists := group2Map[group2][pattern]; !exists {
				group2Map[group2][pattern] = &PatternGroup{
					Pattern:       pattern,
					ProductGroup6: group6,
					ProductGroup3: group3,
					ProductGroup5: group5,
					SubGroups:     []SubGroup{},
				}
			}

			// Add subgroup to the pattern group
			group2Map[group2][pattern].SubGroups = append(group2Map[group2][pattern].SubGroups, subGroup)
		}
	}

	// Convert map to slice
	result := []GroupedData{}
	for group2Val, patternMap := range group2Map {
		groupedData := GroupedData{
			ProductGroup2: group2Val,
			PatternGroups: []PatternGroup{},
		}

		// Add all pattern groups
		for _, patternGroup := range patternMap {
			groupedData.PatternGroups = append(groupedData.PatternGroups, *patternGroup)
		}

		result = append(result, groupedData)
	}

	return result
}

func GetPriceTable(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := GetPriceListGroupRequest{
		GroupCodes:  []string{"GROUP_1_ITEM_1"},
		CompanyCode: "7eb85b75-e708-4e5d-9010-4b43427c15be",
		SiteCodes:   []string{"PRM-00A"},
	}
	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	sqlx, err := db.ConnectSqlx(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer sqlx.Close()

	res, err := getGroupSubGroup(sqlx, req)
	if err != nil {
		return nil, fmt.Errorf("GetGroupSubGroup error: %w", err)
	}

	// Group data according to pattern
	groupedData := groupDataByPattern(res)

	// Print summary statistics
	printGroupingSummary(groupedData)

	// Convert to table row format
	tableStructures := convertToTableRows(groupedData)

	// Print table structure
	printTableStructure(tableStructures)

	// Print JSON output
	fmt.Println("\n========== JSON OUTPUT ==========")
	utils.PrintJSON(tableStructures)

	return tableStructures, nil

	// return PriceListDetailApiResponse{
	// 	Id:   uuid.New(),
	// 	Name: "หมวดเหล็ก แผ่น",
	// 	Tabs: []PriceListDetailTabConfig{
	// 		{
	// 			ID:    uuid.New(),
	// 			Label: "หมวดเหล็กแผ่น",
	// 			TableConfig: TableConfig{
	// 				Title: "หมวดเหล็กแผ่น",
	// 				Toolbar: &Toolbar{
	// 					Show:             &[]bool{true}[0],
	// 					ShowSearch:       &[]bool{true}[0],
	// 					ShowRefresh:      &[]bool{true}[0],
	// 					ShowColumnToggle: &[]bool{true}[0],
	// 				},
	// 				Pagination: &[]bool{false}[0],
	// 				Columns: []ColumnDef{
	// 					{
	// 						Field:           "#",
	// 						HeaderName:      "#",
	// 						Width:           &[]int{60}[0],
	// 						Pinned:          "left",
	// 						LockPosition:    &[]bool{true}[0],
	// 						SuppressMovable: &[]bool{true}[0],
	// 						ValueGetter:     "rowIndex",
	// 						CellStyle: &CellStyle{
	// 							TextAlign:  "center",
	// 							FontWeight: "500",
	// 						},
	// 					},
	// 					{
	// 						Field:      "name",
	// 						HeaderName: "Name",
	// 					},
	// 				},
	// 			},
	// 			TableData: []map[string]interface{}{
	// 				{
	// 					"id":        uuid.New(),
	// 					"thickness": 15, "type": "T", "highlight": true, "sizeBefore": 675.7, "sizeAfter": 0, "extra": 0, "unit": "29", "status": "ป่วม",
	// 					"thickness2": 15, "type2": "T", "highlight2": false, "sizeBefore2": 675.7, "sizeAfter2": 0, "extra2": 0, "unit2": "29", "status2": "",
	// 					"thickness3": 15, "type3": "T", "highlight3": true, "sizeBefore3": 675.7, "sizeAfter3": 0, "extra3": 0, "unit3": "29", "status3": "",
	// 				},
	// 			},
	// 		},
	// 		{
	// 			ID:    uuid.New(),
	// 			Label: "หมวดเหล็กแผ่นลาย",
	// 		},
	// 		{
	// 			ID:    uuid.New(),
	// 			Label: "เหล็กแผ่น special",
	// 		},
	// 	},
	// }, nil
}
