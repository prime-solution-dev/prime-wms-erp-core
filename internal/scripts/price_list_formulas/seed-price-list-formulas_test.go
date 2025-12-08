package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"

	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var postgresContainer tc.Container

func TestMain(m *testing.M) {
	ctx := context.Background()
	req := tc.ContainerRequest{
		Image:        "postgres:16",
		Env:          map[string]string{"POSTGRES_PASSWORD": "test", "POSTGRES_USER": "test", "POSTGRES_DB": "testdb"},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		fmt.Printf("failed to start postgres container: %v\n", err)
		os.Exit(1)
	}
	postgresContainer = container

	host, err := container.Host(ctx)
	if err != nil {
		fmt.Printf("failed to get host: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	mappedPort, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		fmt.Printf("failed to get mapped port: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	dsn := fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, mappedPort.Port())
	os.Setenv("database_gorm_url_prime_erp", dsn)

	if err := createFormulasTestSchema(); err != nil {
		fmt.Printf("failed to create schema: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	code := m.Run()
	_ = container.Terminate(ctx)
	os.Exit(code)
}

func createFormulasTestSchema() error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	statements := []string{
		`CREATE TABLE IF NOT EXISTS price_list_formulas (
			id uuid PRIMARY KEY,
			formula_code text NOT NULL,
			name text NOT NULL,
			uom text NOT NULL,
			formula_type text NOT NULL,
			expression text NOT NULL,
			params jsonb NOT NULL,
			rounding integer NOT NULL,
			create_dtm timestamp NOT NULL
		);`,
	}

	for _, stmt := range statements {
		if err := gormx.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func TestGenerateSQLStatements(t *testing.T) {
	testCases := []struct {
		name     string
		formulas []PriceListFormulaInput
		wantLen  int
	}{
		{
			name: "single formula with all fields",
			formulas: []PriceListFormulaInput{
				{
					Name:        "Test Formula",
					Uom:         "kg",
					FormulaType: "price_calc",
					Expression:  stringPtr("base_price + extra"),
					Params:      stringPtr(`{"required": ["base_price", "extra"]}`),
					Rounding:    2,
				},
			},
			wantLen: 1,
		},
		{
			name: "formula with null expression and params (input type)",
			formulas: []PriceListFormulaInput{
				{
					Name:        "pcs = input",
					Uom:         "pcs",
					FormulaType: "input",
					Expression:  nil,
					Params:      nil,
					Rounding:    2,
				},
			},
			wantLen: 1,
		},
		{
			name: "multiple formulas",
			formulas: []PriceListFormulaInput{
				{
					Name:        "Formula 1",
					Uom:         "kg",
					FormulaType: "price_calc",
					Expression:  stringPtr("base_price + extra"),
					Params:      stringPtr(`{"required": ["base_price"]}`),
					Rounding:    2,
				},
				{
					Name:        "Formula 2",
					Uom:         "pcs",
					FormulaType: "price_calc",
					Expression:  stringPtr("kg * weight_spec"),
					Params:      stringPtr(`{"required": ["kg", "weight_spec"]}`),
					Rounding:    0,
				},
			},
			wantLen: 2,
		},
		{
			name: "formula with special characters in name",
			formulas: []PriceListFormulaInput{
				{
					Name:        "Formula with 'quotes' and \"double quotes\"",
					Uom:         "kg",
					FormulaType: "price_calc",
					Expression:  stringPtr("base_price + extra"),
					Params:      stringPtr(`{"required": ["base_price"]}`),
					Rounding:    2,
				},
			},
			wantLen: 1,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			statements := generateSQLStatements(tc.formulas)
			require.Len(t, statements, tc.wantLen)

			for _, stmt := range statements {
				require.NotEmpty(t, stmt)
				require.Contains(t, stmt, "INSERT INTO public.price_list_formulas")
				require.Contains(t, stmt, "ON CONFLICT (id) DO NOTHING")
			}
		})
	}
}

func TestGenerateSQLStatementsWithNullValues(t *testing.T) {
	formula := PriceListFormulaInput{
		Name:        "Input formula",
		Uom:         "pcs",
		FormulaType: "input",
		Expression:  nil,
		Params:      nil,
		Rounding:    0,
	}

	statements := generateSQLStatements([]PriceListFormulaInput{formula})
	require.Len(t, statements, 1)

	stmt := statements[0]
	// Should contain empty string for expression and {} for params
	require.Contains(t, stmt, "''")
	require.Contains(t, stmt, "'{}'::jsonb")
}

func TestGenerateSQLStatementsIncludesFormulaCode(t *testing.T) {
	formula := PriceListFormulaInput{
		Name:        "Test Formula",
		Uom:         "kg",
		FormulaType: "price_calc",
		Expression:  stringPtr("base_price"),
		Params:      stringPtr("{}"),
		Rounding:    2,
	}

	statements := generateSQLStatements([]PriceListFormulaInput{formula})
	require.Len(t, statements, 1)
	require.Contains(t, statements[0], "INSERT INTO public.price_list_formulas")
	require.Contains(t, statements[0], "formula_code")
	require.Contains(t, statements[0], "FM-test_formula")
	require.True(t, strings.Contains(statements[0], "FM-"))
}

func TestGeneratedSQLExecutesAgainstPostgres(t *testing.T) {
	gormx, err := db.ConnectGORM("prime_erp")
	require.NoError(t, err)
	defer db.CloseGORM(gormx)

	formulas := []PriceListFormulaInput{
		{
			Name:        "kg = Base price + Extra",
			Uom:         "kg",
			FormulaType: "price_calc",
			Expression:  stringPtr("base_price + extra"),
			Params:      stringPtr(`{"required": ["base_price", "extra"], "description": "kg = Base price + Extra"}`),
			Rounding:    2,
		},
		{
			Name:        "pcs = input",
			Uom:         "pcs",
			FormulaType: "input",
			Expression:  nil,
			Params:      nil,
			Rounding:    2,
		},
		{
			Name:        "Pcs = [kg] x [Weight Spec]",
			Uom:         "pcs",
			FormulaType: "price_calc",
			Expression:  stringPtr("kg * weight_spec"),
			Params:      stringPtr(`{"required": ["kg", "weight_spec"], "description": "Pcs = [kg] x [Weight Spec]"}`),
			Rounding:    0,
		},
	}

	statements := generateSQLStatements(formulas)
	require.Len(t, statements, 3)

	// Execute statements
	for _, stmt := range statements {
		require.NoError(t, gormx.Exec(stmt).Error)
	}

	// Verify data was inserted
	var insertedFormulas []models.PriceListFormulas
	require.NoError(t, gormx.Find(&insertedFormulas).Error)
	require.Len(t, insertedFormulas, 3)

	// Verify first formula
	require.Equal(t, formulas[0].Name, insertedFormulas[0].Name)
	require.Equal(t, formulas[0].Uom, insertedFormulas[0].Uom)
	require.Equal(t, formulas[0].FormulaType, insertedFormulas[0].FormulaType)
	require.Equal(t, *formulas[0].Expression, insertedFormulas[0].Expression)
	require.Equal(t, formulas[0].Rounding, insertedFormulas[0].Rounding)
	require.NotEmpty(t, insertedFormulas[0].FormulaCode)
	require.NotEmpty(t, insertedFormulas[0].ID)

	// Verify second formula (input type with null values)
	require.Equal(t, formulas[1].Name, insertedFormulas[1].Name)
	require.Equal(t, "", insertedFormulas[1].Expression) // Should be empty string
	var paramsJSON map[string]interface{}
	require.NoError(t, json.Unmarshal(insertedFormulas[1].Params, &paramsJSON))
	require.Empty(t, paramsJSON) // Should be empty JSON object

	// Verify third formula
	require.Equal(t, formulas[2].Name, insertedFormulas[2].Name)
	require.Equal(t, formulas[2].Rounding, insertedFormulas[2].Rounding)
}

func TestGeneratedSQLHandlesOnConflict(t *testing.T) {
	gormx, err := db.ConnectGORM("prime_erp")
	require.NoError(t, err)
	defer db.CloseGORM(gormx)

	formula := PriceListFormulaInput{
		Name:        "Duplicate Test",
		Uom:         "kg",
		FormulaType: "price_calc",
		Expression:  stringPtr("base_price"),
		Params:      stringPtr("{}"),
		Rounding:    2,
	}

	statements := generateSQLStatements([]PriceListFormulaInput{formula})
	require.Len(t, statements, 1)

	// Insert first time
	require.NoError(t, gormx.Exec(statements[0]).Error)

	// Insert again - should not error due to ON CONFLICT DO NOTHING
	require.NoError(t, gormx.Exec(statements[0]).Error)

	// Verify only one record exists (by formula_code since we can't predict the UUID)
	var count int64
	require.NoError(t, gormx.Model(&models.PriceListFormulas{}).Where("formula_code = ?", "duplicate_test").Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestGenerateSQLStatementsWithComplexExpression(t *testing.T) {
	formula := PriceListFormulaInput{
		Name:        "Complex Formula",
		Uom:         "pcs",
		FormulaType: "price_calc",
		Expression:  stringPtr("(base_price + 1.4) * avg_kg_stock * 1.02"),
		Params:      stringPtr(`{"required": ["base_price", "avg_kg_stock"], "description": "Pcs = ( [Base price] + 1.4 ) x [Avg kg. stock] x (1+2%)"}`),
		Rounding:    0,
	}

	statements := generateSQLStatements([]PriceListFormulaInput{formula})
	require.Len(t, statements, 1)

	stmt := statements[0]
	require.Contains(t, stmt, "(base_price + 1.4) * avg_kg_stock * 1.02")
	require.Contains(t, stmt, "base_price")
	require.Contains(t, stmt, "avg_kg_stock")
}

func TestGenerateSQLStatementsWithJSONParams(t *testing.T) {
	formula := PriceListFormulaInput{
		Name:        "JSON Params Test",
		Uom:         "kg",
		FormulaType: "price_calc",
		Expression:  stringPtr("pcs / avg_kg_stock"),
		Params:      stringPtr(`{"required": ["pcs", "avg_kg_stock"], "description": "kg = [Pcs] / [Avg. kg stock]"}`),
		Rounding:    2,
	}

	statements := generateSQLStatements([]PriceListFormulaInput{formula})
	require.Len(t, statements, 1)

	stmt := statements[0]
	// Should contain the JSON params as a JSONB cast
	require.Contains(t, stmt, "::jsonb")
	require.Contains(t, stmt, "required")
	require.Contains(t, stmt, "pcs")
	require.Contains(t, stmt, "avg_kg_stock")
}

func TestGenerateFormulaCode(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "kg = Base price + Extra",
			expected: "FM-kg_base_price_extra",
		},
		{
			name:     "name with brackets",
			input:    "kg = [Pcs] / [Avg. kg stock]",
			expected: "FM-kg",
		},
		{
			name:     "name with special characters",
			input:    "Pcs = ( [Base price] + 1.4 ) x [Avg kg. stock] x (1+2%)",
			expected: "FM-pcs",
		},
		{
			name:     "lowercase input",
			input:    "pcs = input",
			expected: "FM-pcs_input",
		},
		{
			name:     "name with multiple spaces",
			input:    "Pcs = [kg]  x [Weight Spec]",
			expected: "FM-pcs",
		},
		{
			name:     "empty after processing",
			input:    "[][][]",
			expected: "FM-formula",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result := generateFormulaCode(tc.input)
			require.Equal(t, tc.expected, result)
			require.True(t, strings.HasPrefix(result, "FM-"))
			require.LessOrEqual(t, len(result), 50)
		})
	}
}

func TestGenerateSQLStatementsWithCustomFormulaCode(t *testing.T) {
	customCode := "custom_formula_code"
	formula := PriceListFormulaInput{
		FormulaCode: &customCode,
		Name:        "Test Formula",
		Uom:         "kg",
		FormulaType: "price_calc",
		Expression:  stringPtr("base_price"),
		Params:      stringPtr("{}"),
		Rounding:    2,
	}

	statements := generateSQLStatements([]PriceListFormulaInput{formula})
	require.Len(t, statements, 1)
	require.Contains(t, statements[0], customCode)
}

func stringPtr(s string) *string {
	return &s
}
