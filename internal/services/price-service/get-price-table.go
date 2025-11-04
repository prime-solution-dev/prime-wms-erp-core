package priceService

import (
	"fmt"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/utils"
	"regexp"
	"strings"

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
	Highlight      bool      `json:"highlight"` // Highlight สีฟ้า
	Remark         string    `json:"remark"`    // หมายเหตุ
	SubGroupKey    string    `json:"subgroup_key"`
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

// buildAGGridColumnsWithGrade creates AG Grid column definitions with column groups INCLUDING grade
// Structure: [#, PRODUCT_GROUP6, PRODUCT_GROUP5_1 (with children including grade), ...]
// Used for pattern_g6_g3_g5
func buildAGGridColumnsWithGrade(uniqueGroup5 []string) []ColumnDef {
	columns := []ColumnDef{}

	// Add row number column
	columns = append(columns, ColumnDef{
		Field:           "#",
		HeaderName:      "#",
		Width:           intPtr(60),
		Pinned:          "left",
		LockPosition:    boolPtr(true),
		SuppressMovable: boolPtr(true),
		ValueGetter:     "node.rowIndex + 1",
		CellStyle: &CellStyle{
			TextAlign:  "center",
			FontWeight: "500",
		},
	})

	// Add PRODUCT_GROUP6 column (sheet thickness)
	columns = append(columns, ColumnDef{
		Field:           "product_group_6",
		HeaderName:      "แผ่น mm.", // Sheet mm.
		Width:           intPtr(120),
		Pinned:          "left",
		LockPosition:    boolPtr(true),
		SuppressMovable: boolPtr(true),
		CellStyle: &CellStyle{
			FontWeight: "600",
		},
	})

	// Add column groups for each PRODUCT_GROUP5
	for _, g5 := range uniqueGroup5 {
		columnGroup := ColumnDef{
			HeaderName:    g5,
			GroupID:       fmt.Sprintf("group_%s", g5),
			OpenByDefault: boolPtr(true),
			Children:      []ColumnDef{},
		}

		// Add child columns for this group (7 columns including grade)
		// 1. เกรด (Grade) - PRODUCT_GROUP3
		columnGroup.Children = append(columnGroup.Children, ColumnDef{
			Field:      fmt.Sprintf("%s_grade", sanitizeFieldName(g5)),
			HeaderName: "เกรด",
			Width:      intPtr(80),
		})

		// 2. Highlight สีฟ้า
		columnGroup.Children = append(columnGroup.Children, ColumnDef{
			Field:        fmt.Sprintf("%s_highlight", sanitizeFieldName(g5)),
			HeaderName:   "Highlight สีฟ้า",
			Width:        intPtr(100),
			CellRenderer: "checkboxRenderer",
		})

		// 3. ราคาขาย (Pcs) before
		columnGroup.Children = append(columnGroup.Children, ColumnDef{
			Field:      fmt.Sprintf("%s_price_before", sanitizeFieldName(g5)),
			HeaderName: "ราคาขาย (Pcs) before",
			Width:      intPtr(130),
			CellStyle: &CellStyle{
				TextAlign: "right",
			},
		})

		// 4. ราคาขาย (Pcs) After
		columnGroup.Children = append(columnGroup.Children, ColumnDef{
			Field:      fmt.Sprintf("%s_price_after", sanitizeFieldName(g5)),
			HeaderName: "ราคาขาย (Pcs) After",
			Width:      intPtr(130),
			CellStyle: &CellStyle{
				TextAlign: "right",
			},
		})

		// 5. Extra (THB)
		columnGroup.Children = append(columnGroup.Children, ColumnDef{
			Field:      fmt.Sprintf("%s_extra", sanitizeFieldName(g5)),
			HeaderName: "Extra (THB)",
			Width:      intPtr(100),
			CellStyle: &CellStyle{
				TextAlign: "right",
			},
		})

		// 6. น.น. (Weight)
		columnGroup.Children = append(columnGroup.Children, ColumnDef{
			Field:      fmt.Sprintf("%s_weight", sanitizeFieldName(g5)),
			HeaderName: "น.น.",
			Width:      intPtr(80),
			CellStyle: &CellStyle{
				TextAlign: "right",
			},
		})

		// 7. หมายเหตุ (Remark)
		columnGroup.Children = append(columnGroup.Children, ColumnDef{
			Field:      fmt.Sprintf("%s_remark", sanitizeFieldName(g5)),
			HeaderName: "หมายเหตุ",
			Width:      intPtr(150),
		})

		columns = append(columns, columnGroup)
	}

	return columns
}

// buildAGGridColumns creates AG Grid column definitions with column groups WITHOUT grade
// Structure: [#, PRODUCT_GROUP6, PRODUCT_GROUP5_1 (with children), PRODUCT_GROUP5_2 (with children), ...]
// Used for pattern_g6_g5
func buildAGGridColumns(uniqueGroup5 []string) []ColumnDef {
	columns := []ColumnDef{}

	// Add row number column
	columns = append(columns, ColumnDef{
		Field:           "#",
		HeaderName:      "#",
		Width:           intPtr(60),
		Pinned:          "left",
		LockPosition:    boolPtr(true),
		SuppressMovable: boolPtr(true),
		ValueGetter:     "node.rowIndex + 1",
		CellStyle: &CellStyle{
			TextAlign:  "center",
			FontWeight: "500",
		},
	})

	// Add PRODUCT_GROUP6 column (sheet thickness)
	columns = append(columns, ColumnDef{
		Field:           "product_group_6",
		HeaderName:      "แผ่น mm.", // Sheet mm.
		Width:           intPtr(120),
		Pinned:          "left",
		LockPosition:    boolPtr(true),
		SuppressMovable: boolPtr(true),
		CellStyle: &CellStyle{
			FontWeight: "600",
		},
	})

	// Add column groups for each PRODUCT_GROUP5
	for _, g5 := range uniqueGroup5 {
		columnGroup := ColumnDef{
			HeaderName:    g5,
			GroupID:       fmt.Sprintf("group_%s", g5),
			OpenByDefault: boolPtr(true),
			Children:      []ColumnDef{},
		}

		// Add child columns for this group (6 columns based on image)
		// 1. Highlight สีฟ้า
		columnGroup.Children = append(columnGroup.Children, ColumnDef{
			Field:        fmt.Sprintf("%s_highlight", sanitizeFieldName(g5)),
			HeaderName:   "Highlight สีฟ้า",
			Width:        intPtr(100),
			CellRenderer: "checkboxRenderer",
		})

		// 2. ราคาขาย (Pcs) before
		columnGroup.Children = append(columnGroup.Children, ColumnDef{
			Field:      fmt.Sprintf("%s_price_before", sanitizeFieldName(g5)),
			HeaderName: "ราคาขาย (Pcs) before",
			Width:      intPtr(130),
			CellStyle: &CellStyle{
				TextAlign: "right",
			},
		})

		// 3. ราคาขาย (Pcs) After
		columnGroup.Children = append(columnGroup.Children, ColumnDef{
			Field:      fmt.Sprintf("%s_price_after", sanitizeFieldName(g5)),
			HeaderName: "ราคาขาย (Pcs) After",
			Width:      intPtr(130),
			CellStyle: &CellStyle{
				TextAlign: "right",
			},
		})

		// 4. Extra (THB)
		columnGroup.Children = append(columnGroup.Children, ColumnDef{
			Field:      fmt.Sprintf("%s_extra", sanitizeFieldName(g5)),
			HeaderName: "Extra (THB)",
			Width:      intPtr(100),
			CellStyle: &CellStyle{
				TextAlign: "right",
			},
		})

		// 5. น.น. (Weight)
		columnGroup.Children = append(columnGroup.Children, ColumnDef{
			Field:      fmt.Sprintf("%s_weight", sanitizeFieldName(g5)),
			HeaderName: "น.น.",
			Width:      intPtr(80),
			CellStyle: &CellStyle{
				TextAlign: "right",
			},
		})

		// 6. หมายเหตุ (Remark)
		columnGroup.Children = append(columnGroup.Children, ColumnDef{
			Field:      fmt.Sprintf("%s_remark", sanitizeFieldName(g5)),
			HeaderName: "หมายเหตุ",
			Width:      intPtr(150),
		})

		columns = append(columns, columnGroup)
	}

	return columns
}

// buildAGGridRowsWithGrade converts grouped data into AG Grid row format WITH grade
// Each row represents one PRODUCT_GROUP6 value
// Columns are flattened with naming: {g5}_grade, {g5}_highlight, {g5}_price_before, etc.
// Pattern: PRODUCT_GROUP6 | PRODUCT_GROUP3 | PRODUCT_GROUP5 (same G6, different G3+G5)
func buildAGGridRowsWithGrade(groupedData GroupedData) []AGGridRowData {
	// Map to hold rows: key = PRODUCT_GROUP6
	rowMap := make(map[string]AGGridRowData)

	// Process each pattern group
	for _, pattern := range groupedData.PatternGroups {
		g6 := pattern.ProductGroup6
		g3 := pattern.ProductGroup3
		g5 := pattern.ProductGroup5

		// Initialize row if not exists
		if _, exists := rowMap[g6]; !exists {
			rowMap[g6] = AGGridRowData{
				"id":              uuid.New().String(),
				"product_group_2": groupedData.ProductGroup2,
				"product_group_6": g6,
			}
		}

		// Add data for this G5 column group (including grade)
		if len(pattern.SubGroups) > 0 {
			subGroup := pattern.SubGroups[0]
			g5Safe := sanitizeFieldName(g5)

			// Set all child column values (7 columns including grade)
			rowMap[g6][fmt.Sprintf("%s_grade", g5Safe)] = g3        // Include grade
			rowMap[g6][fmt.Sprintf("%s_highlight", g5Safe)] = false // TODO: determine from data
			rowMap[g6][fmt.Sprintf("%s_price_before", g5Safe)] = subGroup.PriceUnit
			rowMap[g6][fmt.Sprintf("%s_price_after", g5Safe)] = subGroup.TotalNetPriceUnit
			rowMap[g6][fmt.Sprintf("%s_extra", g5Safe)] = subGroup.ExtraPriceUnit
			rowMap[g6][fmt.Sprintf("%s_weight", g5Safe)] = subGroup.PriceWeight
			rowMap[g6][fmt.Sprintf("%s_remark", g5Safe)] = subGroup.Remark

			// Store metadata
			rowMap[g6][fmt.Sprintf("%s_subgroup_id", g5Safe)] = subGroup.ID.String()
			rowMap[g6][fmt.Sprintf("%s_is_trading", g5Safe)] = subGroup.IsTrading
			rowMap[g6][fmt.Sprintf("%s_product_group_3", g5Safe)] = g3 // Store G3 for reference
		}
	}

	// Convert map to slice
	rows := []AGGridRowData{}
	for _, row := range rowMap {
		rows = append(rows, row)
	}

	return rows
}

// buildAGGridRows converts grouped data into AG Grid row format WITHOUT grade
// Each row represents one PRODUCT_GROUP6 value
// Columns are flattened with naming: {g5}_highlight, {g5}_price_before, etc.
// Pattern: PRODUCT_GROUP6 | PRODUCT_GROUP5 (same G6, different G5)
func buildAGGridRows(groupedData GroupedData) []AGGridRowData {
	// Map to hold rows: key = PRODUCT_GROUP6
	rowMap := make(map[string]AGGridRowData)

	// Process each pattern group
	for _, pattern := range groupedData.PatternGroups {
		g6 := pattern.ProductGroup6
		g5 := pattern.ProductGroup5

		// Initialize row if not exists
		if _, exists := rowMap[g6]; !exists {
			rowMap[g6] = AGGridRowData{
				"id":              uuid.New().String(),
				"product_group_2": groupedData.ProductGroup2,
				"product_group_6": g6,
			}
		}

		// Add data for this G5 column group
		if len(pattern.SubGroups) > 0 {
			subGroup := pattern.SubGroups[0]
			g5Safe := sanitizeFieldName(g5)

			// Set all child column values (6 columns)
			rowMap[g6][fmt.Sprintf("%s_highlight", g5Safe)] = false // TODO: determine from data
			rowMap[g6][fmt.Sprintf("%s_price_before", g5Safe)] = subGroup.PriceUnit
			rowMap[g6][fmt.Sprintf("%s_price_after", g5Safe)] = subGroup.TotalNetPriceUnit
			rowMap[g6][fmt.Sprintf("%s_extra", g5Safe)] = subGroup.ExtraPriceUnit
			rowMap[g6][fmt.Sprintf("%s_weight", g5Safe)] = subGroup.PriceWeight
			rowMap[g6][fmt.Sprintf("%s_remark", g5Safe)] = subGroup.Remark

			// Store metadata
			rowMap[g6][fmt.Sprintf("%s_subgroup_id", g5Safe)] = subGroup.ID.String()
			rowMap[g6][fmt.Sprintf("%s_is_trading", g5Safe)] = subGroup.IsTrading
		}
	}

	// Convert map to slice
	rows := []AGGridRowData{}
	for _, row := range rowMap {
		rows = append(rows, row)
	}

	return rows
}

// buildAGGridColumnsMultiLevel creates AG Grid column definitions with 3-level nested groups
// Structure: [#, PRODUCT_GROUP6, PRODUCT_GROUP3 > PRODUCT_GROUP8 > PRODUCT_GROUP5 > child columns]
// Pattern: G3 | G8 | G5 (e.g., LT > Go 7 = YK > 3090 x 6096 > [thickness, highlight, price_before, ...])
func buildAGGridColumnsMultiLevel(groupedData GroupedData) []ColumnDef {
	columns := []ColumnDef{}

	// Add row number column
	columns = append(columns, ColumnDef{
		Field:           "#",
		HeaderName:      "#",
		Width:           intPtr(60),
		Pinned:          "left",
		LockPosition:    boolPtr(true),
		SuppressMovable: boolPtr(true),
		ValueGetter:     "node.rowIndex + 1",
		CellStyle: &CellStyle{
			TextAlign:  "center",
			FontWeight: "500",
		},
	})

	// Add PRODUCT_GROUP6 column (thickness - row identifier)
	columns = append(columns, ColumnDef{
		Field:           "product_group_6",
		HeaderName:      "หนา",
		Width:           intPtr(100),
		Pinned:          "left",
		LockPosition:    boolPtr(true),
		SuppressMovable: boolPtr(true),
		CellStyle: &CellStyle{
			FontWeight: "600",
		},
	})

	// Build hierarchical structure: G3 > G8 > G5
	g3Map := make(map[string]map[string]map[string]bool) // G3 > G8 > G5

	for _, pattern := range groupedData.PatternGroups {
		g3 := pattern.ProductGroup3
		g8 := pattern.ProductGroup8
		g5 := pattern.ProductGroup5

		if g3Map[g3] == nil {
			g3Map[g3] = make(map[string]map[string]bool)
		}
		if g3Map[g3][g8] == nil {
			g3Map[g3][g8] = make(map[string]bool)
		}
		g3Map[g3][g8][g5] = true
	}

	// Build column groups
	for g3, g8Map := range g3Map {
		g3Group := ColumnDef{
			HeaderName:    g3,
			GroupID:       fmt.Sprintf("group_g3_%s", sanitizeFieldName(g3)),
			OpenByDefault: boolPtr(true),
			Children:      []ColumnDef{},
		}

		for g8, g5Map := range g8Map {
			g8Group := ColumnDef{
				HeaderName:    g8,
				GroupID:       fmt.Sprintf("group_g8_%s_%s", sanitizeFieldName(g3), sanitizeFieldName(g8)),
				OpenByDefault: boolPtr(true),
				Children:      []ColumnDef{},
			}

			for g5 := range g5Map {
				g5Group := ColumnDef{
					HeaderName:    g5,
					GroupID:       fmt.Sprintf("group_g5_%s_%s_%s", sanitizeFieldName(g3), sanitizeFieldName(g8), sanitizeFieldName(g5)),
					OpenByDefault: boolPtr(true),
					Children:      []ColumnDef{},
				}

				// Create unique field prefix
				fieldPrefix := fmt.Sprintf("%s_%s_%s", sanitizeFieldName(g3), sanitizeFieldName(g8), sanitizeFieldName(g5))

				// Add child columns (7 columns)
				// 1. หนา (Thickness) - This shows the PRODUCT_GROUP6 value in the cell
				g5Group.Children = append(g5Group.Children, ColumnDef{
					Field:      fmt.Sprintf("%s_thickness", fieldPrefix),
					HeaderName: "หนา",
					Width:      intPtr(80),
				})

				// 2. Highlight สีฟ้า
				g5Group.Children = append(g5Group.Children, ColumnDef{
					Field:        fmt.Sprintf("%s_highlight", fieldPrefix),
					HeaderName:   "Highlight สีฟ้า",
					Width:        intPtr(100),
					CellRenderer: "checkboxRenderer",
				})

				// 3. ราคาขาย before
				g5Group.Children = append(g5Group.Children, ColumnDef{
					Field:      fmt.Sprintf("%s_price_before", fieldPrefix),
					HeaderName: "ราคาขาย before",
					Width:      intPtr(130),
					CellStyle: &CellStyle{
						TextAlign: "right",
					},
				})

				// 4. ราคาขาย After
				g5Group.Children = append(g5Group.Children, ColumnDef{
					Field:      fmt.Sprintf("%s_price_after", fieldPrefix),
					HeaderName: "ราคาขาย After",
					Width:      intPtr(130),
					CellStyle: &CellStyle{
						TextAlign: "right",
					},
				})

				// 5. Extra (THB)
				g5Group.Children = append(g5Group.Children, ColumnDef{
					Field:      fmt.Sprintf("%s_extra", fieldPrefix),
					HeaderName: "Extra (THB)",
					Width:      intPtr(100),
					CellStyle: &CellStyle{
						TextAlign: "right",
					},
				})

				// 6. น.น. (Weight)
				g5Group.Children = append(g5Group.Children, ColumnDef{
					Field:      fmt.Sprintf("%s_weight", fieldPrefix),
					HeaderName: "น.น.",
					Width:      intPtr(80),
					CellStyle: &CellStyle{
						TextAlign: "right",
					},
				})

				// 7. หมายเหตุ (Remark)
				g5Group.Children = append(g5Group.Children, ColumnDef{
					Field:      fmt.Sprintf("%s_remark", fieldPrefix),
					HeaderName: "หมายเหตุ",
					Width:      intPtr(150),
				})

				g8Group.Children = append(g8Group.Children, g5Group)
			}

			g3Group.Children = append(g3Group.Children, g8Group)
		}

		columns = append(columns, g3Group)
	}

	return columns
}

// buildAGGridRowsMultiLevel converts grouped data into AG Grid row format for multi-level pattern
// Pattern: G3 | G8 | G5 (rows by G6)
func buildAGGridRowsMultiLevel(groupedData GroupedData) []AGGridRowData {
	// Map to hold rows: key = PRODUCT_GROUP6
	rowMap := make(map[string]AGGridRowData)

	// Process each pattern group
	for _, pattern := range groupedData.PatternGroups {
		g6 := pattern.ProductGroup6
		g3 := pattern.ProductGroup3
		g8 := pattern.ProductGroup8
		g5 := pattern.ProductGroup5

		// Initialize row if not exists
		if _, exists := rowMap[g6]; !exists {
			rowMap[g6] = AGGridRowData{
				"id":              uuid.New().String(),
				"product_group_2": groupedData.ProductGroup2,
				"product_group_6": g6,
			}
		}

		// Add data for this G3|G8|G5 combination
		if len(pattern.SubGroups) > 0 {
			subGroup := pattern.SubGroups[0]
			fieldPrefix := fmt.Sprintf("%s_%s_%s", sanitizeFieldName(g3), sanitizeFieldName(g8), sanitizeFieldName(g5))

			// Set all child column values (7 columns)
			rowMap[g6][fmt.Sprintf("%s_thickness", fieldPrefix)] = g6
			rowMap[g6][fmt.Sprintf("%s_highlight", fieldPrefix)] = false // TODO: determine from data
			rowMap[g6][fmt.Sprintf("%s_price_before", fieldPrefix)] = subGroup.PriceUnit
			rowMap[g6][fmt.Sprintf("%s_price_after", fieldPrefix)] = subGroup.TotalNetPriceUnit
			rowMap[g6][fmt.Sprintf("%s_extra", fieldPrefix)] = subGroup.ExtraPriceUnit
			rowMap[g6][fmt.Sprintf("%s_weight", fieldPrefix)] = subGroup.PriceWeight
			rowMap[g6][fmt.Sprintf("%s_remark", fieldPrefix)] = subGroup.Remark

			// Store metadata
			rowMap[g6][fmt.Sprintf("%s_subgroup_id", fieldPrefix)] = subGroup.ID.String()
			rowMap[g6][fmt.Sprintf("%s_is_trading", fieldPrefix)] = subGroup.IsTrading
		}
	}

	// Convert map to slice
	rows := []AGGridRowData{}
	for _, row := range rowMap {
		rows = append(rows, row)
	}

	return rows
}

// detectPattern determines which pattern to use based on PRODUCT_GROUP2
func detectPattern(productGroup2 string) string {
	// Pattern 3: Multi-level nested (G3 | G8 | G5)
	if productGroup2 == "เหล็กแผ่น special" {
		return "pattern_g3_g8_g5"
	}

	// Pattern 2: Simple (G6 | G5)
	if productGroup2 == "หมวดเหล็กแผ่นลาย" {
		return "pattern_g6_g5"
	}

	// Pattern 1: Standard (G6 | G3 | G5)
	return "pattern_g6_g3_g5"
}

// printAGGridStructure prints the AG Grid structure in a readable format
func printAGGridStructure(tabs []PriceListDetailTabConfig) {
	fmt.Println("\n========== AG GRID STRUCTURE ==========")

	for i, tab := range tabs {
		fmt.Printf("\nTab %d - %s\n", i+1, tab.Label)
		fmt.Printf("  Total Columns: %d\n", len(tab.TableConfig.Columns))
		fmt.Printf("  Total Rows: %d\n", len(tab.TableData))

		// Print column structure
		fmt.Println("\n  Column Structure:")
		for j, col := range tab.TableConfig.Columns {
			if len(col.Children) > 0 {
				fmt.Printf("    %d. Column Group: %s (GroupID: %s)\n", j+1, col.HeaderName, col.GroupID)
				fmt.Printf("       Children: %d columns\n", len(col.Children))
				for k, child := range col.Children {
					fmt.Printf("         %d.%d. %s (field: %s)\n", j+1, k+1, child.HeaderName, child.Field)
				}
			} else {
				fmt.Printf("    %d. Column: %s (field: %s)\n", j+1, col.HeaderName, col.Field)
			}
		}

		// Print sample row data
		if len(tab.TableData) > 0 {
			fmt.Println("\n  Sample Row Data:")
			fmt.Printf("    %v\n", tab.TableData[0])
		}
	}
	fmt.Println("\n=======================================")
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

// groupDataByPattern groups subgroups by PRODUCT_GROUP2, then by pattern PRODUCT_GROUP6|PRODUCT_GROUP3|PRODUCT_GROUP5|PRODUCT_GROUP8
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
			group8 := getGroupKeyValue(subGroup.GroupKeys, "PRODUCT_GROUP8")

			// Create pattern: PRODUCT_GROUP6|PRODUCT_GROUP3|PRODUCT_GROUP5|PRODUCT_GROUP8
			pattern := fmt.Sprintf("%s|%s|%s|%s", group6, group3, group5, group8)

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
					ProductGroup8: group8,
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

// loadPriceData loads and parses the price.json file
func loadPriceData() ([]GetPriceListGroupResponse, error) {
	// Get the current working directory
	// currentDir, err := os.Getwd()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get current directory: %w", err)
	// }

	// // Construct path to price.json
	// jsonPath := filepath.Join(currentDir, "internal", "services", "price-service", "price.json")

	// // Read the JSON file
	// data, err := os.ReadFile(jsonPath)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to read price.json: %w", err)
	// }

	// // Parse JSON data into SubGroup array
	// var subGroups []SubGroup
	// if err := json.Unmarshal(data, &subGroups); err != nil {
	// 	return nil, fmt.Errorf("failed to unmarshal price.json: %w", err)
	// }

	// // Wrap subgroups in GetPriceListGroupResponse structure
	// // Since the JSON contains a flat array of subgroups, we wrap them in one response
	// response := GetPriceListGroupResponse{
	// 	PriceListGroup: PriceListGroup{
	// 		SubGroups: subGroups,
	// 	},
	// }
	// return []GetPriceListGroupResponse{response}, nil
	sqlx, err := db.ConnectSqlx(`prime_erp`)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer sqlx.Close()

	groupSubGroup, err := getGroupSubGroup(sqlx, GetPriceListGroupRequest{
		CompanyCode: "7eb85b75-e708-4e5d-9010-4b43427c15be",
		SiteCodes:   []string{"PRM-00A"},
		GroupCodes:  []string{"GROUP_1_ITEM_1"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get group sub group: %w", err)
	}

	// groupSubGroup is already []GetPriceListGroupResponse, so return it directly
	return groupSubGroup, nil
}

func GetPriceTable(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	// Load data from price.json
	res, err := loadPriceData()
	if err != nil {
		return nil, fmt.Errorf("failed to load price data: %w", err)
	}

	fmt.Println("\n========== LOADED DATA ==========")
	fmt.Printf("Total responses: %d\n", len(res))
	for i, resp := range res {
		fmt.Printf("Response %d: %d subgroups\n", i+1, len(resp.SubGroups))
	}

	// Group data according to pattern
	groupedData := groupDataByPattern(res)

	// Print summary statistics
	printGroupingSummary(groupedData)

	// Build AG Grid response with tabs
	tabs := []PriceListDetailTabConfig{}

	for _, group := range groupedData {
		// Detect which pattern to use
		patternType := detectPattern(group.ProductGroup2)

		var columns []ColumnDef
		var rowData []AGGridRowData

		switch patternType {
		case "pattern_g3_g8_g5":
			// Multi-level nested pattern (G3 > G8 > G5)
			columns = buildAGGridColumnsMultiLevel(group)
			rowData = buildAGGridRowsMultiLevel(group)
		case "pattern_g6_g3_g5":
			// Pattern with grade (G6 | G3 | G5)
			// Collect unique PRODUCT_GROUP5 values for columns
			uniqueGroup5Map := make(map[string]bool)
			for _, pattern := range group.PatternGroups {
				uniqueGroup5Map[pattern.ProductGroup5] = true
			}

			// Convert to sorted slice
			uniqueGroup5 := []string{}
			for g5 := range uniqueGroup5Map {
				uniqueGroup5 = append(uniqueGroup5, g5)
			}

			// Build column definitions with grade
			columns = buildAGGridColumnsWithGrade(uniqueGroup5)

			// Build row data with grade
			rowData = buildAGGridRowsWithGrade(group)
		default:
			// Simple pattern (G6 | G5) - without grade
			// Collect unique PRODUCT_GROUP5 values for columns
			uniqueGroup5Map := make(map[string]bool)
			for _, pattern := range group.PatternGroups {
				uniqueGroup5Map[pattern.ProductGroup5] = true
			}

			// Convert to sorted slice
			uniqueGroup5 := []string{}
			for g5 := range uniqueGroup5Map {
				uniqueGroup5 = append(uniqueGroup5, g5)
			}

			// Build column definitions without grade
			columns = buildAGGridColumns(uniqueGroup5)

			// Build row data without grade
			rowData = buildAGGridRows(group)
		}

		// Convert AGGridRowData to []map[string]interface{}
		tableData := make([]map[string]interface{}, len(rowData))
		for i, row := range rowData {
			tableData[i] = map[string]interface{}(row)
		}

		// Create tab configuration
		tab := PriceListDetailTabConfig{
			ID:    uuid.New(),
			Label: group.ProductGroup2,
			TableConfig: TableConfig{
				Title:             group.ProductGroup2,
				GroupHeaderHeight: intPtr(40),
				HeaderHeight:      intPtr(50),
				Pagination:        boolPtr(false),
				Toolbar: &Toolbar{
					Show:             boolPtr(true),
					ShowSearch:       boolPtr(true),
					ShowRefresh:      boolPtr(true),
					ShowColumnToggle: boolPtr(true),
				},
				GridOptions: &GridOptions{
					SuppressMovableColumns: boolPtr(false),
					SuppressMenuHide:       boolPtr(false),
				},
				Columns: columns,
			},
			TableData: tableData,
		}

		tabs = append(tabs, tab)
	}

	// Create final response
	response := PriceListDetailApiResponse{
		Id:   res[0].ID,
		Name: "Price List Detail",
		Tabs: tabs,
	}

	// Print AG Grid structure
	printAGGridStructure(tabs)

	// Print JSON output
	fmt.Println("\n========== JSON OUTPUT ==========")
	utils.PrintJSON(response)

	return response, nil
}
