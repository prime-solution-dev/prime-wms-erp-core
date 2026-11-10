package invoiceService

import (
	"fmt"
	"strings"
)

func calculateAPPriceUnit(poUnit, interfaceUnit string, poPrice, qty, weight float64) (float64, error) {
	poUnit = strings.ToUpper(strings.TrimSpace(poUnit))
	interfaceUnit = strings.ToUpper(strings.TrimSpace(interfaceUnit))
	if poUnit == interfaceUnit || (poUnit != "KG" && interfaceUnit != "KG") {
		return poPrice, nil
	}
	if interfaceUnit == "KG" {
		if weight == 0 {
			return 0, fmt.Errorf("GR weight must be non-zero to convert price to KG")
		}
		return poPrice * qty / weight, nil
	}
	if qty == 0 {
		return 0, fmt.Errorf("GR qty must be non-zero to convert price from KG")
	}
	return poPrice * weight / qty, nil
}
