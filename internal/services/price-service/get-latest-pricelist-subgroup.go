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

// seam for unit testing: allow stubbing repository function
var getLatestSubGroupFunc = priceListRepository.GetPriceListSubGroupByID

// GetLatestPriceListSubGroup returns the refreshed data for the provided sub group.
func GetLatestPriceListSubGroup(ctx *gin.Context) (interface{}, error) {
	var req models.GetLatestPriceListSubGroupRequest

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

	var prices []priceDomain.Price

	for _, subGroupID := range req.SubGroupIDs {
		subGroup, err := getLatestSubGroupFunc(uuid.MustParse(subGroupID))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch latest price list sub group: %w", err)
		}
		if subGroup == nil {
			return nil, &utils.BindingError{
				Message: "Price list sub group not found",
			}
		}

		priceListFormulas, err := priceListRepository.GetPriceListSubGroupFormulasMapBySubGroupID(subGroup.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch default price list formulas: %w", err)
		}

		price := priceDomain.Price{
			Id:                  subGroup.ID.String(),
			TotalNetPriceUnit:   subGroup.TotalNetPriceUnit,
			TotalNetPriceWeight: subGroup.TotalNetPriceWeight,
			DefaultUom:          "kg",
		}

		if len(priceListFormulas) == 0 {
			prices = append(prices, price)
			continue
		}

		// Check for default input formula
		foundDefaultInput := false
		for _, formula := range priceListFormulas {
			if formula.PriceListFormulas.FormulaType == "input" && formula.IsDefault {
				price.DefaultUom = formula.PriceListFormulas.Uom
				prices = append(prices, price)
				foundDefaultInput = true
				break
			}
		}

		if foundDefaultInput {
			continue
		}

		// Process non-input formulas
		for _, formula := range priceListFormulas {
			if formula.PriceListFormulas.FormulaType == "input" {
				continue
			}
			switch formula.PriceListFormulas.Uom {
			case "pcs":
				priceData := priceDomain.PriceData{
					BasePrice:  subGroup.PriceListGroup.PriceUnit,
					Extra:      0,
					AvgKgStock: 0,
				}

				priceFormula := priceDomain.PriceFormula{
					Expression: formula.PriceListFormulas.Expression,
					Params:     formula.PriceListFormulas.Params,
					Rounding:   formula.PriceListFormulas.Rounding,
				}
				totalNetPriceUnit, err := CalculatePrice(priceFormula, priceData)
				if err != nil {
					return nil, fmt.Errorf("failed to calculate total net price unit: %w", err)
				}
				price.TotalNetPriceUnit = totalNetPriceUnit
				if formula.IsDefault {
					price.DefaultUom = "pcs"
				}
			case "kg":
				priceData := priceDomain.PriceData{
					BasePrice:  subGroup.PriceListGroup.PriceWeight,
					Extra:      0,
					AvgKgStock: 0,
				}
				priceFormula := priceDomain.PriceFormula{
					Expression: formula.PriceListFormulas.Expression,
					Params:     formula.PriceListFormulas.Params,
					Rounding:   formula.PriceListFormulas.Rounding,
				}
				totalNetPriceWeight, err := CalculatePrice(priceFormula, priceData)
				if err != nil {
					return nil, fmt.Errorf("failed to calculate total net price weight: %w", err)
				}
				price.TotalNetPriceWeight = totalNetPriceWeight
				if formula.IsDefault {
					price.DefaultUom = "kg"
				}
			}
		}

		prices = append(prices, price)
	}

	result := priceDomain.PriceResult{
		Prices: prices,
	}

	return result, nil
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
