package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"prime-erp-core/internal/scripts/shared"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type PriceListFormulaInput struct {
	FormulaCode *string `json:"formula_code,omitempty"` // Optional, will be auto-generated if not provided
	Name        string  `json:"name"`
	Uom         string  `json:"uom"`
	FormulaType string  `json:"formula_type"`
	Expression  *string `json:"expression"`
	Params      *string `json:"params"`
	Rounding    int     `json:"rounding"`
}

type PriceListFormulasData struct {
	PriceListFormulas []PriceListFormulaInput `json:"price_list_formulas"`
}

func main() {
	var (
		inputFile        = flag.String("input", "price-list-formulas.json", "Path to JSON file containing formulas")
		outputPath       = flag.String("output", "", "Optional output file path for SQL statements (defaults to stdout)")
		executeSeed      = flag.Bool("execute", false, "Execute generated statements against the configured database")
		databaseName     = flag.String("database", "prime_erp", "Database name suffix (reads database_gorm_url_<name> from env)")
		connectionString = flag.String("connection-string", "", "Direct database connection string (overrides database name/env variable)")
	)

	flag.Parse()

	// Read JSON file
	data, err := os.ReadFile(*inputFile)
	if err != nil {
		shared.ExitWithError(fmt.Errorf("failed to read input file: %w", err))
	}

	var formulasData PriceListFormulasData
	if err := json.Unmarshal(data, &formulasData); err != nil {
		shared.ExitWithError(fmt.Errorf("failed to parse JSON: %w", err))
	}

	// Generate SQL statements
	statements := generateSQLStatements(formulasData.PriceListFormulas)

	// Output SQL statements
	writer := os.Stdout
	if *outputPath != "" {
		file, err := os.Create(*outputPath)
		if err != nil {
			shared.ExitWithError(fmt.Errorf("failed to create output file: %w", err))
		}
		defer file.Close()
		writer = file
	}

	fmt.Fprintf(writer, "-- Auto-generated seed for price_list_formulas (%s)\n", time.Now().UTC().Format(time.RFC3339))
	for _, stmt := range statements {
		fmt.Fprintln(writer, stmt)
	}

	// Execute if requested
	if *executeSeed {
		if err := shared.ExecuteSeedStatements(*databaseName, *connectionString, statements); err != nil {
			shared.ExitWithError(err)
		}
		fmt.Fprintf(os.Stderr, "Successfully inserted %d formulas into price_list_formulas table\n", len(formulasData.PriceListFormulas))
	}
}

func generateSQLStatements(formulas []PriceListFormulaInput) []string {
	statements := make([]string, 0, len(formulas))

	for _, formula := range formulas {
		// Auto-generate UUID
		id := uuid.New()

		// Auto-generate create_dtm
		createDtm := time.Now().UTC()

		// Auto-generate formula_code if not provided
		formulaCode := ""
		if formula.FormulaCode != nil && *formula.FormulaCode != "" {
			formulaCode = *formula.FormulaCode
		} else {
			formulaCode = generateFormulaCode(formula.Name)
		}

		// Handle null expression and params
		expression := ""
		if formula.Expression != nil {
			expression = *formula.Expression
		}

		params := "{}"
		if formula.Params != nil {
			params = *formula.Params
		}

		// Escape single quotes in strings
		formulaCodeEscaped := shared.EscapeSQLString(formulaCode)
		name := shared.EscapeSQLString(formula.Name)
		uom := shared.EscapeSQLString(formula.Uom)
		formulaType := shared.EscapeSQLString(formula.FormulaType)
		expressionEscaped := shared.EscapeSQLString(expression)
		paramsEscaped := shared.EscapeSQLString(params)

		stmt := fmt.Sprintf(
			"INSERT INTO public.price_list_formulas (id, formula_code, name, uom, formula_type, expression, params, rounding, create_dtm) VALUES (%s, %s, %s, %s, %s, %s, %s::jsonb, %d, %s) ON CONFLICT (id) DO NOTHING;",
			shared.FormatUUID(id),
			shared.FormatString(formulaCodeEscaped),
			shared.FormatString(name),
			shared.FormatString(uom),
			shared.FormatString(formulaType),
			shared.FormatString(expressionEscaped),
			shared.FormatString(paramsEscaped),
			formula.Rounding,
			shared.FormatTimestamp(createDtm),
		)

		statements = append(statements, stmt)
	}

	return statements
}

// generateFormulaCode creates a code from the formula name
// Converts to lowercase, replaces spaces and special chars with underscores, removes brackets
// Prefixes with "FM-"
func generateFormulaCode(name string) string {
	// Convert to lowercase
	code := strings.ToLower(name)

	// Remove brackets and their contents
	bracketRegex := regexp.MustCompile(`\[.*?\]`)
	code = bracketRegex.ReplaceAllString(code, "")

	// Replace special characters with underscores
	specialCharRegex := regexp.MustCompile(`[^a-z0-9]+`)
	code = specialCharRegex.ReplaceAllString(code, "_")

	// Remove leading/trailing underscores and collapse multiple underscores
	code = strings.Trim(code, "_")
	underscoreRegex := regexp.MustCompile(`_+`)
	code = underscoreRegex.ReplaceAllString(code, "_")

	// Limit length to 47 characters (50 - 3 for "FM-" prefix)
	if len(code) > 47 {
		code = code[:47]
		code = strings.Trim(code, "_")
	}

	// If empty after processing, use a default
	if code == "" {
		code = "formula"
	}

	// Prefix with "FM-"
	return "FM-" + code
}
