package priceService

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"

	"prime-erp-core/internal/models"
	priceListRepository "prime-erp-core/internal/repositories/priceList"
	priceDomain "prime-erp-core/internal/services/price-service/domain"
	"prime-erp-core/internal/utils"

	"github.com/expr-lang/expr"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// seam for unit testing: allow stubbing repository functions
var updateLatestSubGroupFunc = priceListRepository.UpdatePriceListSubGroups
var getPriceListSubGroupsByIDsFunc = priceListRepository.GetPriceListSubGroupsByIDs
var getPriceListSubGroupFormulasMapBySubGroupCodesFunc = priceListRepository.GetPriceListSubGroupFormulasMapBySubGroupCodes

// UpdateLatestPriceListSubGroup calculates and updates the price list sub group data in the database.
func UpdateLatestPriceListSubGroup(ctx *gin.Context) (interface{}, error) {
	var req models.UpdateLatestPriceListSubGroupRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var errorMessages []string
			for _, fieldError := range validationErrors {
				errorMessages = append(errorMessages, getValidationErrorMessage(fieldError))
			}
			return nil, &utils.BindingError{
				Message: fmt.Sprintf("Validation failed: %v", errorMessages),
			}
		}
		return nil, &utils.BindingError{
			Message: fmt.Sprintf("Invalid request: %v", err.Error()),
		}
	}

	// Convert all SubGroupIDs to UUIDs upfront
	subGroupUUIDs := make([]uuid.UUID, 0, len(req.SubGroupIDs))
	for _, subGroupID := range req.SubGroupIDs {
		subGroupUUIDs = append(subGroupUUIDs, uuid.MustParse(subGroupID))
	}

	// Fetch all sub groups in one batch query
	subGroups, err := getPriceListSubGroupsByIDsFunc(subGroupUUIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest price list sub groups: %w", err)
	}

	// Create a map of sub groups by ID for efficient lookup
	subGroupMap := make(map[uuid.UUID]*models.PriceListSubGroup, len(subGroups))
	for i := range subGroups {
		subGroupMap[subGroups[i].ID] = &subGroups[i]
	}

	// Verify all requested sub groups were found
	for _, subGroupID := range subGroupUUIDs {
		if _, found := subGroupMap[subGroupID]; !found {
			return nil, &utils.BindingError{
				Message: "Price list sub group not found",
			}
		}
	}

	// Collect all sub group codes for batch formula fetching
	subGroupCodes := make([]string, 0, len(subGroups))
	for _, subGroup := range subGroups {
		if subGroup.SubGroupCode != "" {
			subGroupCodes = append(subGroupCodes, subGroup.SubGroupCode)
		}
	}

	// Fetch all formulas in one batch query
	formulasMap, err := getPriceListSubGroupFormulasMapBySubGroupCodesFunc(subGroupCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch default price list formulas: %w", err)
	}

	// Prepare update requests for each sub group
	updateChanges := make([]models.UpdatePriceListSubGroupItem, 0, len(subGroupUUIDs))

	// Process each sub group using the maps
	for _, subGroupID := range subGroupUUIDs {
		subGroup := subGroupMap[subGroupID]

		priceListFormulas := formulasMap[subGroup.SubGroupCode]

		// Initialize calculated values with current values
		totalNetPriceUnit := subGroup.TotalNetPriceUnit
		totalNetPriceWeight := subGroup.TotalNetPriceWeight

		// Calculate Extra from price_list_group_extras / group_item (for weight)
		extraPriceWeight, err := calculateExtraForSubGroup(subGroup)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate extra for sub group %s: %w", subGroupID, err)
		}

		if len(priceListFormulas) > 0 {
			// Check for default input formula
			foundDefaultInput := false
			for _, formula := range priceListFormulas {
				if formula.PriceListFormulas.FormulaType == "input" && formula.IsDefault {
					// Keep existing values for input formulas
					foundDefaultInput = true
					break
				}
			}

			if !foundDefaultInput {
				// Process non-input formulas
				for _, formula := range priceListFormulas {
					if formula.PriceListFormulas.FormulaType == "input" {
						continue
					}
					switch formula.PriceListFormulas.Uom {
					case "pcs":
						priceData := priceDomain.PriceData{
							BasePrice:  subGroup.PriceListGroup.PriceUnit,
							Extra:      extraPriceWeight,
							AvgKgStock: 0, // TODO: get avg kg stock from stock
							WeightSpec: 0, // TODO: get weight spec from product
							Pcs:        0, // TODO: get pcs from product
							Kg:         0, // TODO: get kg from product
						}

						priceFormula := priceDomain.PriceFormula{
							Expression: formula.PriceListFormulas.Expression,
							Params:     formula.PriceListFormulas.Params,
							Rounding:   formula.PriceListFormulas.Rounding,
						}
						calculatedTotalNetPriceUnit, err := CalculatePrice(priceFormula, priceData)
						if err != nil {
							return nil, fmt.Errorf("failed to calculate total net price unit: %w", err)
						}
						totalNetPriceUnit = calculatedTotalNetPriceUnit
					case "kg":
						priceData := priceDomain.PriceData{
							BasePrice:  subGroup.PriceListGroup.PriceWeight,
							Extra:      extraPriceWeight,
							AvgKgStock: 0, // TODO: get avg kg stock from stock
							WeightSpec: 0, // TODO: get weight spec from product
							Pcs:        0, // TODO: get pcs from product
							Kg:         0, // TODO: get kg from product
						}
						priceFormula := priceDomain.PriceFormula{
							Expression: formula.PriceListFormulas.Expression,
							Params:     formula.PriceListFormulas.Params,
							Rounding:   formula.PriceListFormulas.Rounding,
						}
						calculatedTotalNetPriceWeight, err := CalculatePrice(priceFormula, priceData)
						if err != nil {
							return nil, fmt.Errorf("failed to calculate total net price weight: %w", err)
						}
						totalNetPriceWeight = calculatedTotalNetPriceWeight
					}
				}
			}
		}

		// Create update request for this sub group
		updateChanges = append(updateChanges, models.UpdatePriceListSubGroupItem{
			SubGroupID:          subGroupID,
			TotalNetPriceUnit:   &totalNetPriceUnit,
			TotalNetPriceWeight: &totalNetPriceWeight,
			ExtraPriceWeight:    &extraPriceWeight,
		})
	}

	// Update all sub groups in the database
	updateRequest := models.UpdatePriceListSubGroupRequest{
		Changes: updateChanges,
	}

	if err := updateLatestSubGroupFunc(updateRequest); err != nil {
		return nil, fmt.Errorf("failed to update price list sub groups: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": "Price list sub groups updated successfully",
	}, nil
}

// calculateExtraForSubGroup determines the Extra value (for weight) for a given sub group
// using price_list_group_extras, price_list_group_extra_keys and group_item.value_int.
func calculateExtraForSubGroup(subGroup *models.PriceListSubGroup) (float64, error) {
	// Build a quick lookup map from subgroup keys: code -> value
	subGroupKeyMap := make(map[string]string, len(subGroup.PriceListSubGroupKeys))
	for _, k := range subGroup.PriceListSubGroupKeys {
		subGroupKeyMap[k.Code] = k.Value
	}

	// Start from existing ExtraPriceWeight so that, in absence of matching config,
	// we preserve the current extra behavior.
	extra := subGroup.ExtraPriceWeight

	extras := subGroup.PriceListGroup.PriceListGroupExtras
	for _, e := range extras {
		// First check that all extra keys match this subgroup's keys
		matchedAllKeys := true
		for _, ek := range e.PriceListGroupExtraKeys {
			v, ok := subGroupKeyMap[ek.Code]
			if !ok || v != ek.Value {
				matchedAllKeys = false
				break
			}
		}
		if !matchedAllKeys {
			continue
		}

		// Now handle condition_code logic against group_item.value_int
		if e.ConditionCode == "" {
			continue
		}

		// Find the subgroup key corresponding to condition_code
		condValue, ok := subGroupKeyMap[e.ConditionCode]
		if !ok {
			continue
		}

		valInt, found, err := priceListRepository.GetGroupItemValueInt(e.ConditionCode, condValue)
		if err != nil {
			return 0, err
		}
		if !found {
			continue
		}

		if extraConditionMatched(valInt, e.Operator, e.CondRangeMin, e.CondRangeMax) {
			// Use value_int from extra row as the contribution for Extra
			extra += float64(e.ValueInt)
		}
	}

	return extra, nil
}

// extraConditionMatched evaluates the operator and cond_range_min/max against the
// group_item.value_int.
func extraConditionMatched(val float64, operator string, min, max float64) bool {
	switch operator {
	case "=":
		// Example from requirement: value_int must match cond_range_max
		return val == max
	case ">=":
		return val >= min
	case "<=":
		return val <= max
	case "<":
		return val < max
	case ">":
		return val > min
	case "<>":
		return val < min || val > max
	default:
		return false
	}
}

func CalculatePrice(formula priceDomain.PriceFormula, priceData priceDomain.PriceData) (float64, error) {
	// 1) parse params JSON
	paramMap := map[string]interface{}{}
	if len(formula.Params) > 0 {
		if err := json.Unmarshal(formula.Params, &paramMap); err != nil {
			return 0, err
		}
	}

	// 2) variables สำหรับ expression
	env := map[string]interface{}{
		"base_price":   priceData.BasePrice,
		"extra":        priceData.Extra,
		"avg_kg_stock": priceData.AvgKgStock,
		"weight_spec":  priceData.WeightSpec,
		"pcs":          priceData.Pcs,
		"kg":           priceData.Kg,
	}

	// รวม params → env
	maps.Copy(env, paramMap)

	// 3) compile expr
	program, err := expr.Compile(formula.Expression, expr.Env(env))
	if err != nil {
		return 0, fmt.Errorf("expression compile failed: %w", err)
	}

	// 4) run
	result, err := expr.Run(program, env)
	if err != nil {
		return 0, fmt.Errorf("expression run failed: %w", err)
	}

	// 5) convert to float64
	num, ok := result.(float64)
	if !ok {
		return 0, fmt.Errorf("expression did not return float64")
	}

	// 6) rounding
	factor := math.Pow(10, float64(formula.Rounding))
	rounded := math.Round(num*factor) / factor

	return rounded, nil
}
