package shared

import (
	"fmt"
	"os"
)

// ExitWithError prints an error message and exits with code 1
func ExitWithError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

