package shared

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FormatUUID formats a UUID for SQL insertion
func FormatUUID(id uuid.UUID) string {
	return fmt.Sprintf("'%s'::uuid", id.String())
}

// FormatString escapes and formats a string for SQL insertion
func FormatString(val string) string {
	escaped := strings.ReplaceAll(val, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
}

// FormatFloat formats a float64 for SQL insertion with 2 decimal places
func FormatFloat(val float64) string {
	return fmt.Sprintf("%.2f", val)
}

// FormatTimestamp formats a time.Time for SQL insertion
func FormatTimestamp(t time.Time) string {
	return fmt.Sprintf("'%s'::timestamp", t.Format(time.RFC3339))
}

// EscapeSQLString escapes single quotes in a string for SQL
func EscapeSQLString(val string) string {
	return strings.ReplaceAll(val, "'", "''")
}

// FormatJSONB formats a JSON string for SQL JSONB insertion
// Escapes single quotes and wraps in quotes with ::jsonb cast
func FormatJSONB(val string) string {
	// Escape single quotes for SQL
	escaped := strings.ReplaceAll(val, "'", "''")
	return fmt.Sprintf("'%s'::jsonb", escaped)
}
