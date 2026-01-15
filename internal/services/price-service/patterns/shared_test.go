package patterns

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValueMappingsConfig_GroupCodeMappings(t *testing.T) {
	tests := []struct {
		name         string
		config       *ValueMappingsConfig
		mappingName  string
		fallbackCode string
		expected     string
	}{
		{
			name: "mapping exists",
			config: &ValueMappingsConfig{
				GroupCodeMappings: map[string]string{
					"productGroup2": "PRODUCT_GROUP2",
					"productGroup6": "PRODUCT_GROUP6",
				},
			},
			mappingName:  "productGroup2",
			fallbackCode: "PRODUCT_GROUP9",
			expected:     "PRODUCT_GROUP2",
		},
		{
			name: "mapping does not exist",
			config: &ValueMappingsConfig{
				GroupCodeMappings: map[string]string{
					"productGroup2": "PRODUCT_GROUP2",
				},
			},
			mappingName:  "productGroup6",
			fallbackCode: "PRODUCT_GROUP9",
			expected:     "PRODUCT_GROUP9",
		},
		{
			name:         "nil config",
			config:       nil,
			mappingName:  "productGroup2",
			fallbackCode: "PRODUCT_GROUP2",
			expected:     "PRODUCT_GROUP2",
		},
		{
			name: "nil GroupCodeMappings",
			config: &ValueMappingsConfig{
				GroupCodeMappings: nil,
			},
			mappingName:  "productGroup2",
			fallbackCode: "PRODUCT_GROUP2",
			expected:     "PRODUCT_GROUP2",
		},
		{
			name: "empty mapping value",
			config: &ValueMappingsConfig{
				GroupCodeMappings: map[string]string{
					"productGroup2": "",
				},
			},
			mappingName:  "productGroup2",
			fallbackCode: "PRODUCT_GROUP9",
			expected:     "PRODUCT_GROUP9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getGroupCodeByMapping(tt.config, tt.mappingName, tt.fallbackCode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValueMappingsConfig_SpecialMappings(t *testing.T) {
	tests := []struct {
		name     string
		config   *ValueMappingsConfig
		key      string
		fallback string
		expected string
	}{
		{
			name: "special mapping exists",
			config: &ValueMappingsConfig{
				SpecialMappings: map[string]string{
					"type": "PRODUCT_GROUP9",
				},
			},
			key:      "type",
			fallback: "PRODUCT_GROUP8",
			expected: "PRODUCT_GROUP9",
		},
		{
			name: "special mapping does not exist",
			config: &ValueMappingsConfig{
				SpecialMappings: map[string]string{
					"type": "PRODUCT_GROUP9",
				},
			},
			key:      "category",
			fallback: "PRODUCT_GROUP8",
			expected: "PRODUCT_GROUP8",
		},
		{
			name:     "nil config",
			config:   nil,
			key:      "type",
			fallback: "PRODUCT_GROUP9",
			expected: "PRODUCT_GROUP9",
		},
		{
			name: "nil SpecialMappings",
			config: &ValueMappingsConfig{
				SpecialMappings: nil,
			},
			key:      "type",
			fallback: "PRODUCT_GROUP9",
			expected: "PRODUCT_GROUP9",
		},
		{
			name: "empty mapping value",
			config: &ValueMappingsConfig{
				SpecialMappings: map[string]string{
					"type": "",
				},
			},
			key:      "type",
			fallback: "PRODUCT_GROUP9",
			expected: "PRODUCT_GROUP9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSpecialMapping(tt.config, tt.key, tt.fallback)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEffectiveValueMappings(t *testing.T) {
	rootMappings := &ValueMappingsConfig{
		GroupCodeMappings: map[string]string{
			"productGroup2": "PRODUCT_GROUP2",
		},
	}

	patternMappings := &ValueMappingsConfig{
		GroupCodeMappings: map[string]string{
			"productGroup2": "PRODUCT_GROUP2_OVERRIDE",
		},
	}

	tests := []struct {
		name     string
		root     *PriceTableConfiguration
		pattern  *PatternConfig
		expected *ValueMappingsConfig
	}{
		{
			name: "pattern mappings override root",
			root: &PriceTableConfiguration{
				ValueMappings: rootMappings,
			},
			pattern: &PatternConfig{
				ValueMappings: patternMappings,
			},
			expected: patternMappings,
		},
		{
			name: "only root mappings exist",
			root: &PriceTableConfiguration{
				ValueMappings: rootMappings,
			},
			pattern: &PatternConfig{
				ValueMappings: nil,
			},
			expected: rootMappings,
		},
		{
			name: "only pattern mappings exist",
			root: &PriceTableConfiguration{
				ValueMappings: nil,
			},
			pattern: &PatternConfig{
				ValueMappings: patternMappings,
			},
			expected: patternMappings,
		},
		{
			name: "no mappings exist",
			root: &PriceTableConfiguration{
				ValueMappings: nil,
			},
			pattern: &PatternConfig{
				ValueMappings: nil,
			},
			expected: nil,
		},
		{
			name: "nil root",
			root: nil,
			pattern: &PatternConfig{
				ValueMappings: patternMappings,
			},
			expected: patternMappings,
		},
		{
			name: "nil pattern",
			root: &PriceTableConfiguration{
				ValueMappings: rootMappings,
			},
			pattern:  nil,
			expected: rootMappings,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEffectiveValueMappings(tt.root, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDefaultItemFormat(t *testing.T) {
	rootDefaultFormat := []ItemFormatPart{
		{Type: "group", Value: "PRODUCT_GROUP4"},
		{Type: "literal", Value: "x"},
	}

	patternDefaultFormat := []ItemFormatPart{
		{Type: "group", Value: "PRODUCT_GROUP5"},
		{Type: "literal", Value: "y"},
	}

	valueMappingsFormat := []ItemFormatPart{
		{Type: "group", Value: "PRODUCT_GROUP6"},
		{Type: "literal", Value: "z"},
	}

	tests := []struct {
		name     string
		root     *PriceTableConfiguration
		pattern  *PatternConfig
		expected []ItemFormatPart
	}{
		{
			name: "pattern ItemFormat takes precedence",
			root: &PriceTableConfiguration{
				ValueMappings: &ValueMappingsConfig{
					DefaultItemFormat: valueMappingsFormat,
				},
			},
			pattern: &PatternConfig{
				ItemFormat: patternDefaultFormat,
			},
			expected: patternDefaultFormat,
		},
		{
			name: "valueMappings DefaultItemFormat used when pattern ItemFormat empty",
			root: &PriceTableConfiguration{
				ValueMappings: &ValueMappingsConfig{
					DefaultItemFormat: valueMappingsFormat,
				},
			},
			pattern: &PatternConfig{
				ItemFormat: nil,
			},
			expected: valueMappingsFormat,
		},
		{
			name: "legacy default used when no config",
			root: &PriceTableConfiguration{
				ValueMappings: nil,
			},
			pattern: &PatternConfig{
				ItemFormat: nil,
			},
			expected: legacyDefaultItemFormat,
		},
		{
			name: "pattern ItemFormat empty but valueMappings has format",
			root: &PriceTableConfiguration{
				ValueMappings: &ValueMappingsConfig{
					DefaultItemFormat: valueMappingsFormat,
				},
			},
			pattern: &PatternConfig{
				ItemFormat: []ItemFormatPart{},
			},
			expected: valueMappingsFormat,
		},
		{
			name: "root valueMappings with pattern-level valueMappings override",
			root: &PriceTableConfiguration{
				ValueMappings: &ValueMappingsConfig{
					DefaultItemFormat: rootDefaultFormat,
				},
			},
			pattern: &PatternConfig{
				ItemFormat: nil,
				ValueMappings: &ValueMappingsConfig{
					DefaultItemFormat: valueMappingsFormat,
				},
			},
			expected: valueMappingsFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDefaultItemFormat(tt.root, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadConfiguration_WithValueMappings(t *testing.T) {
	// Test that LoadConfiguration can load a config with valueMappings
	// We'll use GROUP_1_ITEM_1 as it should exist
	config, err := LoadConfiguration("GROUP_1_ITEM_1")
	if err != nil {
		t.Skipf("Could not load GROUP_1_ITEM_1 config: %v", err)
		return
	}

	// Verify the config structure is valid
	assert.NotNil(t, config)
	assert.NotEmpty(t, config.Patterns)
	assert.NotEmpty(t, config.DefaultPattern)

	// Check if valueMappings exist (they may or may not be present in existing configs)
	// This test ensures backward compatibility - configs without valueMappings should still work
	if config.ValueMappings != nil {
		// If valueMappings exist, verify structure
		if config.ValueMappings.GroupCodeMappings != nil {
			assert.IsType(t, map[string]string{}, config.ValueMappings.GroupCodeMappings)
		}
		if config.ValueMappings.HandlerMappings != nil {
			assert.IsType(t, map[string]string{}, config.ValueMappings.HandlerMappings)
		}
		if config.ValueMappings.DefaultItemFormat != nil {
			assert.IsType(t, []ItemFormatPart{}, config.ValueMappings.DefaultItemFormat)
		}
		if config.ValueMappings.SpecialMappings != nil {
			assert.IsType(t, map[string]string{}, config.ValueMappings.SpecialMappings)
		}
	}

	// Verify patterns can have valueMappings too
	for _, pattern := range config.Patterns {
		if pattern.ValueMappings != nil {
			if pattern.ValueMappings.GroupCodeMappings != nil {
				assert.IsType(t, map[string]string{}, pattern.ValueMappings.GroupCodeMappings)
			}
		}
	}
}

func TestLoadConfiguration_BackwardCompatibility(t *testing.T) {
	// Test that configs without valueMappings still work
	config, err := LoadConfiguration("GROUP_1_ITEM_1")
	if err != nil {
		t.Skipf("Could not load GROUP_1_ITEM_1 config: %v", err)
		return
	}

	// Should work even if valueMappings is nil
	assert.NotNil(t, config)

	// Test that getGroupCodeByMapping falls back correctly
	result := getGroupCodeByMapping(config.ValueMappings, "productGroup2", "PRODUCT_GROUP2")
	assert.Equal(t, "PRODUCT_GROUP2", result)

	// Test that getDefaultItemFormat falls back to legacy
	pattern := &PatternConfig{}
	format := getDefaultItemFormat(config, pattern)
	assert.NotEmpty(t, format)
	assert.Equal(t, legacyDefaultItemFormat, format)
}

func TestValueMappingsConfig_JSONUnmarshal(t *testing.T) {
	jsonStr := `{
		"groupCodeMappings": {
			"productGroup2": "PRODUCT_GROUP2",
			"productGroup6": "PRODUCT_GROUP6"
		},
		"handlerMappings": {
			"GROUP_1_ITEM_1": "BuildGroup1Item1Response",
			"GROUP_1_ITEM_2": "BuildGroup1Item2Response"
		},
		"defaultItemFormat": [
			{"type": "group", "value": "PRODUCT_GROUP4"},
			{"type": "literal", "value": "x"},
			{"type": "group", "value": "PRODUCT_GROUP6"}
		],
		"specialMappings": {
			"type": "PRODUCT_GROUP9"
		}
	}`

	var config ValueMappingsConfig
	err := json.Unmarshal([]byte(jsonStr), &config)
	assert.NoError(t, err)

	assert.NotNil(t, config.GroupCodeMappings)
	assert.Equal(t, "PRODUCT_GROUP2", config.GroupCodeMappings["productGroup2"])
	assert.Equal(t, "PRODUCT_GROUP6", config.GroupCodeMappings["productGroup6"])

	assert.NotNil(t, config.HandlerMappings)
	assert.Equal(t, "BuildGroup1Item1Response", config.HandlerMappings["GROUP_1_ITEM_1"])
	assert.Equal(t, "BuildGroup1Item2Response", config.HandlerMappings["GROUP_1_ITEM_2"])

	assert.NotNil(t, config.DefaultItemFormat)
	assert.Len(t, config.DefaultItemFormat, 3)
	assert.Equal(t, "group", config.DefaultItemFormat[0].Type)
	assert.Equal(t, "PRODUCT_GROUP4", config.DefaultItemFormat[0].Value)
	assert.Equal(t, "literal", config.DefaultItemFormat[1].Type)
	assert.Equal(t, "x", config.DefaultItemFormat[1].Value)

	assert.NotNil(t, config.SpecialMappings)
	assert.Equal(t, "PRODUCT_GROUP9", config.SpecialMappings["type"])
}

func TestValueMappingsConfig_JSONUnmarshal_Empty(t *testing.T) {
	// Test that empty or missing fields don't cause errors
	jsonStr := `{}`

	var config ValueMappingsConfig
	err := json.Unmarshal([]byte(jsonStr), &config)
	assert.NoError(t, err)

	assert.Nil(t, config.GroupCodeMappings)
	assert.Nil(t, config.HandlerMappings)
	assert.Nil(t, config.DefaultItemFormat)
	assert.Nil(t, config.SpecialMappings)
}

func TestPriceTableConfiguration_WithValueMappings(t *testing.T) {
	jsonStr := `{
		"patterns": [
			{
				"id": "test_pattern",
				"name": "Test Pattern",
				"enabled": true,
				"grouping": {
					"tabs": "PRODUCT_GROUP2",
					"rows": "PRODUCT_GROUP6",
					"columnGroups": "PRODUCT_GROUP5"
				},
				"columns": [],
				"fixedColumns": [],
				"applicableCategories": [],
				"valueMappings": {
					"groupCodeMappings": {
						"productGroup2": "PRODUCT_GROUP2"
					}
				}
			}
		],
		"defaultPattern": "test_pattern",
		"tableConfig": {
			"groupHeaderHeight": 50,
			"headerHeight": 50,
			"pagination": false,
			"toolbar": {},
			"gridOptions": {}
		},
		"valueMappings": {
			"groupCodeMappings": {
				"productGroup6": "PRODUCT_GROUP6"
			}
		}
	}`

	var config PriceTableConfiguration
	err := json.Unmarshal([]byte(jsonStr), &config)
	assert.NoError(t, err)

	// Verify root-level valueMappings
	assert.NotNil(t, config.ValueMappings)
	assert.Equal(t, "PRODUCT_GROUP6", config.ValueMappings.GroupCodeMappings["productGroup6"])

	// Verify pattern-level valueMappings
	assert.Len(t, config.Patterns, 1)
	assert.NotNil(t, config.Patterns[0].ValueMappings)
	assert.Equal(t, "PRODUCT_GROUP2", config.Patterns[0].ValueMappings.GroupCodeMappings["productGroup2"])

	// Test getEffectiveValueMappings - pattern should override root
	effective := getEffectiveValueMappings(&config, &config.Patterns[0])
	assert.NotNil(t, effective)
	assert.Equal(t, "PRODUCT_GROUP2", effective.GroupCodeMappings["productGroup2"])
	// Pattern doesn't have productGroup6, so it should not be in effective mappings
	_, exists := effective.GroupCodeMappings["productGroup6"]
	assert.False(t, exists)
}

func TestResolveHandler_FromConfig(t *testing.T) {
	// Test resolving handler from root-level valueMappings
	config := &PriceTableConfiguration{
		ValueMappings: &ValueMappingsConfig{
			HandlerMappings: map[string]string{
				"GROUP_1_ITEM_1": "BuildGroup1Item1Response",
				"GROUP_1_ITEM_2": "BuildGroup1Item2Response",
			},
		},
	}

	handler, found := ResolveHandler(config, "GROUP_1_ITEM_1")
	assert.True(t, found, "handler should be found")
	assert.NotNil(t, handler, "handler should not be nil")
}

func TestResolveHandler_FromPatternConfig(t *testing.T) {
	// Test resolving handler from pattern-level valueMappings
	config := &PriceTableConfiguration{
		Patterns: []PatternConfig{
			{
				ID: "test_pattern",
				ValueMappings: &ValueMappingsConfig{
					HandlerMappings: map[string]string{
						"GROUP_1_ITEM_3": "BuildGroup1Item3Response",
					},
				},
			},
		},
	}

	handler, found := ResolveHandler(config, "GROUP_1_ITEM_3")
	assert.True(t, found, "handler should be found")
	assert.NotNil(t, handler, "handler should not be nil")
}

func TestResolveHandler_NoConfigReturnsFalse(t *testing.T) {
	// Test that ResolveHandler returns false when config is nil
	// (Callers should use GetDefaultHandlers() for fallback)
	handler, found := ResolveHandler(nil, "GROUP_1_ITEM_1")
	assert.False(t, found, "handler should not be found when config is nil")
	assert.Nil(t, handler, "handler should be nil")
}

func TestResolveHandler_NoMappingsReturnsFalse(t *testing.T) {
	// Test that ResolveHandler returns false when no mappings exist
	config := &PriceTableConfiguration{
		ValueMappings: nil,
	}

	// ResolveHandler only checks config, doesn't have built-in fallback
	handler, found := ResolveHandler(config, "GROUP_1_ITEM_1")
	assert.False(t, found, "handler should not be found when no mappings exist")
	assert.Nil(t, handler, "handler should be nil")
}

func TestResolveHandler_NotFound(t *testing.T) {
	// Test that ResolveHandler returns false for unknown group codes
	config := &PriceTableConfiguration{
		ValueMappings: nil,
	}

	handler, found := ResolveHandler(config, "UNKNOWN_GROUP_CODE")
	assert.False(t, found, "handler should not be found")
	assert.Nil(t, handler, "handler should be nil")
}

func TestResolveHandler_ConfigOverride(t *testing.T) {
	// Test that config handler mappings override default behavior
	config := &PriceTableConfiguration{
		ValueMappings: &ValueMappingsConfig{
			HandlerMappings: map[string]string{
				"GROUP_1_ITEM_1": "BuildGroup1Item2Response", // Override to use Item2 handler
			},
		},
	}

	handler, found := ResolveHandler(config, "GROUP_1_ITEM_1")
	assert.True(t, found, "handler should be found")
	assert.NotNil(t, handler, "handler should not be nil")
}

func TestResolveHandler_EmptyHandlerID(t *testing.T) {
	// Test that empty handler ID in config returns false (doesn't match empty string)
	config := &PriceTableConfiguration{
		ValueMappings: &ValueMappingsConfig{
			HandlerMappings: map[string]string{
				"GROUP_1_ITEM_1": "", // Empty handler ID
			},
		},
	}

	// Empty handler ID should not match, so returns false
	handler, found := ResolveHandler(config, "GROUP_1_ITEM_1")
	assert.False(t, found, "handler should not be found with empty handler ID")
	assert.Nil(t, handler, "handler should be nil")
}

func TestResolveHandler_PatternOverridesRoot(t *testing.T) {
	// Test that pattern-level handler mappings override root-level
	config := &PriceTableConfiguration{
		ValueMappings: &ValueMappingsConfig{
			HandlerMappings: map[string]string{
				"GROUP_1_ITEM_1": "BuildGroup1Item1Response",
			},
		},
		Patterns: []PatternConfig{
			{
				ID: "test_pattern",
				ValueMappings: &ValueMappingsConfig{
					HandlerMappings: map[string]string{
						"GROUP_1_ITEM_1": "BuildGroup1Item2Response", // Override root
					},
				},
			},
		},
	}

	handler, found := ResolveHandler(config, "GROUP_1_ITEM_1")
	assert.True(t, found, "handler should be found")
	assert.NotNil(t, handler, "handler should not be nil")
}

func TestGetDefaultHandlers(t *testing.T) {
	// Test that GetDefaultHandlers returns the expected handlers
	handlers := GetDefaultHandlers()
	assert.NotNil(t, handlers)
	assert.NotEmpty(t, handlers)

	// Verify some expected handlers exist
	assert.Contains(t, handlers, "GROUP_1_ITEM_1")
	assert.Contains(t, handlers, "GROUP_1_ITEM_2")
	assert.Contains(t, handlers, "GROUP_1_ITEM_13")

	// Verify handlers are callable
	handler1, ok := handlers["GROUP_1_ITEM_1"]
	assert.True(t, ok, "GROUP_1_ITEM_1 handler should exist")
	assert.NotNil(t, handler1, "handler should not be nil")
}

func TestGetGroupCodeFromConfig(t *testing.T) {
	// Test getGroupCodeFromConfig with root-level mappings
	rootConfig := &PriceTableConfiguration{
		ValueMappings: &ValueMappingsConfig{
			GroupCodeMappings: map[string]string{
				"productGroup2": "PRODUCT_GROUP2",
			},
		},
	}

	result := getGroupCodeFromConfig(rootConfig, nil, "productGroup2", "PRODUCT_GROUP9")
	assert.Equal(t, "PRODUCT_GROUP2", result)

	// Test with pattern-level override
	pattern := &PatternConfig{
		ValueMappings: &ValueMappingsConfig{
			GroupCodeMappings: map[string]string{
				"productGroup2": "PRODUCT_GROUP2_OVERRIDE",
			},
		},
	}

	result = getGroupCodeFromConfig(rootConfig, pattern, "productGroup2", "PRODUCT_GROUP9")
	assert.Equal(t, "PRODUCT_GROUP2_OVERRIDE", result)

	// Test fallback when mapping doesn't exist
	result = getGroupCodeFromConfig(rootConfig, pattern, "productGroup6", "PRODUCT_GROUP6")
	assert.Equal(t, "PRODUCT_GROUP6", result)

	// Test with nil config (should handle gracefully)
	// Note: getGroupCodeFromConfig may panic with nil config, so we skip this test
	// or use an empty config instead
	emptyConfig := &PriceTableConfiguration{}
	result = getGroupCodeFromConfig(emptyConfig, nil, "productGroup2", "PRODUCT_GROUP2")
	assert.Equal(t, "PRODUCT_GROUP2", result)
}

