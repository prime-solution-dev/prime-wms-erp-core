package utils

import (
	"encoding/json"
	"fmt"
)

func PrintJSON(v any) {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}
	fmt.Println(string(bytes))
}
