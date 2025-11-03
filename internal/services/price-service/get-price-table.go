package priceService

import (
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

func GetPriceTable(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	// var req GetPriceTableRequest
	// if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
	// 	return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	// }
	return PriceListDetailApiResponse{
		Id:   uuid.New(),
		Name: "หมวดเหล็ก แผ่น",
		Tabs: []PriceListDetailTabConfig{
			{
				ID:    uuid.New(),
				Label: "หมวดเหล็กแผ่น",
				TableConfig: TableConfig{
					Title: "หมวดเหล็กแผ่น",
					Toolbar: &Toolbar{
						Show:             &[]bool{true}[0],
						ShowSearch:       &[]bool{true}[0],
						ShowRefresh:      &[]bool{true}[0],
						ShowColumnToggle: &[]bool{true}[0],
					},
					Pagination: &[]bool{false}[0],
					Columns: []ColumnDef{
						{
							Field:           "#",
							HeaderName:      "#",
							Width:           &[]int{60}[0],
							Pinned:          "left",
							LockPosition:    &[]bool{true}[0],
							SuppressMovable: &[]bool{true}[0],
							ValueGetter:     "rowIndex",
							CellStyle: &CellStyle{
								TextAlign:  "center",
								FontWeight: "500",
							},
						},
						{
							Field:      "name",
							HeaderName: "Name",
						},
					},
				},
				TableData: []map[string]interface{}{
					{
						"id":        uuid.New(),
						"thickness": 15, "type": "T", "highlight": true, "sizeBefore": 675.7, "sizeAfter": 0, "extra": 0, "unit": "29", "status": "ป่วม",
						"thickness2": 15, "type2": "T", "highlight2": false, "sizeBefore2": 675.7, "sizeAfter2": 0, "extra2": 0, "unit2": "29", "status2": "",
						"thickness3": 15, "type3": "T", "highlight3": true, "sizeBefore3": 675.7, "sizeAfter3": 0, "extra3": 0, "unit3": "29", "status3": "",
					},
				},
			},
			{
				ID:    uuid.New(),
				Label: "หมวดเหล็กแผ่นลาย",
			},
			{
				ID:    uuid.New(),
				Label: "เหล็กแผ่น special",
			},
		},
	}, nil
}
