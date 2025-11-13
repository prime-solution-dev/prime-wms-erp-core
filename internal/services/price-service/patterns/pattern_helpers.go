package patterns

// buildTableConfig constructs a `TableConfig` from the provided settings and column definitions.
func buildTableConfig(settings TableConfigSettings, columns []ColumnDef, title string) TableConfig {
	tableConfig := TableConfig{
		Title:             title,
		GroupHeaderHeight: intPtr(settings.GroupHeaderHeight),
		HeaderHeight:      intPtr(settings.HeaderHeight),
		Pagination:        boolPtr(settings.Pagination),
		Columns:           columns,
	}

	if settings.Toolbar.Show || settings.Toolbar.ShowSearch || settings.Toolbar.ShowRefresh || settings.Toolbar.ShowColumnToggle {
		tableConfig.Toolbar = &Toolbar{
			Show:             boolPtr(settings.Toolbar.Show),
			ShowSearch:       boolPtr(settings.Toolbar.ShowSearch),
			ShowRefresh:      boolPtr(settings.Toolbar.ShowRefresh),
			ShowColumnToggle: boolPtr(settings.Toolbar.ShowColumnToggle),
		}
	} else {
		tableConfig.Toolbar = &Toolbar{
			Show:             boolPtr(settings.Toolbar.Show),
			ShowSearch:       boolPtr(settings.Toolbar.ShowSearch),
			ShowRefresh:      boolPtr(settings.Toolbar.ShowRefresh),
			ShowColumnToggle: boolPtr(settings.Toolbar.ShowColumnToggle),
		}
	}

	tableConfig.GridOptions = &GridOptions{
		SuppressMovableColumns: boolPtr(settings.GridOptions.SuppressMovableColumns),
		SuppressMenuHide:       boolPtr(settings.GridOptions.SuppressMenuHide),
	}

	return tableConfig
}
