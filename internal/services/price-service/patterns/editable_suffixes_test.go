package patterns

import (
	"encoding/json"
	"path"
	"testing"
)

// Columns the UI renders as a checkbox. They are only clickable when the pattern
// also lists them in editable_suffixes — GROUP_1_ITEM_8 shipped the Highlight and
// Inactive columns without doing so, which made them impossible to tick.
var checkboxFields = []string{"is_highlight", "inactive"}

// walkJSON visits every object in a decoded JSON document.
func walkJSON(node interface{}, visit func(obj map[string]interface{})) {
	switch value := node.(type) {
	case map[string]interface{}:
		visit(value)
		for _, child := range value {
			walkJSON(child, visit)
		}
	case []interface{}:
		for _, child := range value {
			walkJSON(child, visit)
		}
	}
}

// collectFieldsAndSuffixes returns every "field" declared anywhere in the config
// and every entry across all of its editable_suffixes lists.
func collectFieldsAndSuffixes(doc interface{}) (fields map[string]bool, suffixes map[string]bool) {
	fields = map[string]bool{}
	suffixes = map[string]bool{}

	walkJSON(doc, func(obj map[string]interface{}) {
		if field, ok := obj["field"].(string); ok {
			fields[field] = true
		}
		list, ok := obj["editable_suffixes"].([]interface{})
		if !ok {
			return
		}
		for _, entry := range list {
			if suffix, ok := entry.(string); ok {
				suffixes[suffix] = true
			}
		}
	})

	return fields, suffixes
}

// declaresField reports whether the config has a column for the field, either
// bare (`is_highlight`) or prefixed by a product group (`pg05_3_is_highlight`).
func declaresField(fields map[string]bool, name string) bool {
	for field := range fields {
		if field == name || len(field) > len(name) && field[len(field)-len(name)-1:] == "_"+name {
			return true
		}
	}
	return false
}

// Every pattern that shows a checkbox column must also make it editable, otherwise
// the column renders but no click can ever change it.
func TestCheckboxColumnsAreEditable(t *testing.T) {
	entries, err := patternConfigs.ReadDir("configs")
	if err != nil {
		t.Fatalf("read embedded configs: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no pattern configs are embedded")
	}

	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".json" {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			raw, err := patternConfigs.ReadFile(path.Join("configs", entry.Name()))
			if err != nil {
				t.Fatalf("read config: %v", err)
			}

			var doc interface{}
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatalf("parse config: %v", err)
			}

			fields, suffixes := collectFieldsAndSuffixes(doc)
			if len(suffixes) == 0 {
				t.Skip("config declares no editable_suffixes")
			}

			for _, name := range checkboxFields {
				if declaresField(fields, name) && !suffixes[name] {
					t.Errorf("column %q is rendered but missing from editable_suffixes, so it cannot be ticked", name)
				}
			}
		})
	}
}
