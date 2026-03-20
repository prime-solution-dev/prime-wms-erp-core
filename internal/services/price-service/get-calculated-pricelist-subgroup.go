package priceService

import (
	"fmt"

	externalService "prime-erp-core/external/warehouse-service"
	"prime-erp-core/internal/models"
	priceDomain "prime-erp-core/internal/services/price-service/domain"
	"prime-erp-core/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// GetCalculatedPriceListSubGroup calculates price list sub group values using formulas
// and returns them without persisting to the database.
func GetCalculatedPriceListSubGroup(ctx *gin.Context) (interface{}, error) {
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
		return nil, &utils.BindingError{Message: fmt.Sprintf("Invalid request: %v", err.Error())}
	}

	// Determine update type, defaulting to "subgroup" for backward compatibility
	updateType := req.UpdateType
	if updateType == "" {
		updateType = "subgroup"
	}

	var (
		subGroupUUIDs []uuid.UUID
		subGroups     []models.PriceListSubGroup
		err           error
	)

	switch updateType {
	case "subgroup":
		// Validate subgroup_ids presence
		if len(req.SubGroupIDs) == 0 {
			return nil, &utils.BindingError{
				Message: "subgroup_ids is required when update_type is 'subgroup'",
			}
		}

		// Convert all SubGroupIDs to UUIDs upfront with validation
		subGroupUUIDs = make([]uuid.UUID, 0, len(req.SubGroupIDs))
		for _, subGroupID := range req.SubGroupIDs {
			parsed, parseErr := uuid.Parse(subGroupID)
			if parseErr != nil {
				return nil, &utils.BindingError{
					Message: "Validation failed: subgroup_ids must be valid UUID",
				}
			}
			subGroupUUIDs = append(subGroupUUIDs, parsed)
		}

		// Fetch all sub groups in one batch query
		subGroups, err = getPriceListSubGroupsByIDsFunc(subGroupUUIDs)
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
	case "group":
		// Validate group_codes presence
		if len(req.GroupCodes) == 0 {
			return nil, &utils.BindingError{
				Message: "group_codes is required when update_type is 'group'",
			}
		}

		// Fetch all sub groups for the given group codes
		subGroups, err = getPriceListSubGroupsByGroupCodesFunc(req.GroupCodes)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch latest price list sub groups by group codes: %w", err)
		}

		if len(subGroups) == 0 {
			return nil, &utils.BindingError{
				Message: "No price list sub groups found for the provided group_codes",
			}
		}

		// Build subGroupUUIDs from fetched sub groups
		subGroupUUIDs = make([]uuid.UUID, 0, len(subGroups))
		for i := range subGroups {
			subGroupUUIDs = append(subGroupUUIDs, subGroups[i].ID)
		}
	}

	// Create a map of sub groups by ID for efficient lookup (for both modes)
	subGroupMap := make(map[uuid.UUID]*models.PriceListSubGroup, len(subGroups))
	for i := range subGroups {
		subGroupMap[subGroups[i].ID] = &subGroups[i]
	}

	// Collect all sub group codes for batch formula fetching
	subGroupCodes := make([]string, 0, len(subGroups))
	for _, subGroup := range subGroups {
		// Include SubGroupCode in the list, even if it's empty string
		subGroupCodes = append(subGroupCodes, subGroup.SubGroupCode)
	}
	// Fetch all formulas in one batch query
	formulasMap, err := getPriceListSubGroupFormulasMapBySubGroupCodesFunc(subGroupCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch default price list formulas: %w", err)
	}
	// Collect all key values from all subgroups for inventory service request
	keyValues := []externalService.InventoryByProductCodeKeyValue{}
	companyCodeSet := make(map[string]bool)
	siteCodeSet := make(map[string]bool)

	for _, subGroup := range subGroups {
		// Collect company code and site code
		if subGroup.PriceListGroup.CompanyCode != "" {
			companyCodeSet[subGroup.PriceListGroup.CompanyCode] = true
		}
		if subGroup.PriceListGroup.SiteCode != "" {
			siteCodeSet[subGroup.PriceListGroup.SiteCode] = true
		}

		// Build key values from subgroup keys
		for _, sgk := range subGroup.PriceListSubGroupKeys {
			keyValues = append(keyValues, externalService.InventoryByProductCodeKeyValue{
				ID:         subGroup.ID.String(),
				GroupCode:  sgk.Code,
				GroupValue: sgk.Value,
				Seq:        sgk.Seq,
			})
		}
	}

	// Create inventory maps for quick lookup
	inventoryMap := make(map[string][]models.InventoryWeightResponse)
	supplierCodeMap := make(map[string]string)

	// Call inventory service if we have key values
	if len(keyValues) > 0 {
		// Convert sets to slices
		companyCodes := []string{}
		for code := range companyCodeSet {
			companyCodes = append(companyCodes, code)
		}
		siteCodes := []string{}
		for code := range siteCodeSet {
			siteCodes = append(siteCodes, code)
		}

		// Use first company code for the request
		companyCode := ""
		if len(companyCodes) > 0 {
			companyCode = companyCodes[0]
		}

		// Call inventory service
		inventoryResponse, err := externalService.GetInventoryWeightByKey(companyCode, siteCodes, keyValues)
		if err != nil {
			// Log error but continue without inventory data
			fmt.Printf("Warning: failed to get inventory data: %v\n", err)
		} else {
			// Build maps of inventory data by ID for quick lookup
			for _, invItem := range inventoryResponse {
				inventoryMap[invItem.ID] = invItem.InventoryWeight
				supplierCodeMap[invItem.ID] = invItem.SupplierCode
			}
		}
	}

	// Prepare response data for each sub group
	responseData := make([]models.GetCalculatedPriceListSubGroupItem, 0, len(subGroupUUIDs))

	// Process each sub group using the maps
	for _, subGroupID := range subGroupUUIDs {
		subGroup := subGroupMap[subGroupID]

		// Capture the current database values as "before" values
		beforeTotalNetPriceUnit := subGroup.TotalNetPriceUnit
		beforeTotalNetPriceWeight := subGroup.TotalNetPriceWeight

		priceListFormulas := formulasMap[subGroup.SubGroupCode]
		// Initialize calculated values with current values
		totalNetPriceUnit := subGroup.TotalNetPriceUnit
		totalNetPriceWeight := subGroup.TotalNetPriceWeight

		// Calculate Extra from price_list_group_extras / group_item (for weight)
		extraPriceWeight, extraPriceUnit, err := calculateExtraForSubGroup(subGroup)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate extra for sub group %s: %w", subGroupID, err)
		}

		// Get inventory data for this subgroup
		avgKgStock := 1.0
		weightSpec := 1.0
		pcs := 0.0
		kg := 0.0
		if inventoryWeight, ok := inventoryMap[subGroupID.String()]; ok && len(inventoryWeight) > 0 {
			// Use AvgProduct from first inventory weight response
			if inventoryWeight[0].AvgWeight == 0 {
				avgKgStock = 1.0
			} else {
				avgKgStock = inventoryWeight[0].AvgWeight
			}
			if inventoryWeight[0].TotalWeight == 0 {
				weightSpec = 1.0
			} else {
				weightSpec = inventoryWeight[0].TotalWeight
			}
			pcs = inventoryWeight[0].TotalQty
			kg = inventoryWeight[0].TotalWeight
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
							AvgKgStock: avgKgStock,
							WeightSpec: weightSpec,
							Pcs:        pcs,
							Kg:         kg,
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
							Extra:      extraPriceUnit,
							AvgKgStock: avgKgStock,
							WeightSpec: weightSpec,
							Pcs:        pcs,
							Kg:         kg,
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

		// Find default UOM for this subgroup
		defaultUom := ""
		if formulas, ok := formulasMap[subGroup.SubGroupCode]; ok {
			for _, formula := range formulas {
				if formula.IsDefault {
					defaultUom = formula.PriceListFormulas.Uom
					break
				}
			}
		}

		// Add calculated data to response
		responseData = append(responseData, models.GetCalculatedPriceListSubGroupItem{
			SubGroupID:                subGroupID.String(),
			TotalNetPriceUnit:         totalNetPriceUnit,
			TotalNetPriceWeight:       totalNetPriceWeight,
			ExtraPriceUnit:            extraPriceUnit,
			ExtraPriceWeight:          extraPriceWeight,
			BeforeTotalNetPriceUnit:   beforeTotalNetPriceUnit,
			BeforeTotalNetPriceWeight: beforeTotalNetPriceWeight,
			DefaultUom:                defaultUom,
		})
	}

	return models.GetCalculatedPriceListSubGroupResponse{
		Success: true,
		Message: "Price calculations retrieved successfully",
		Data:    responseData,
	}, nil
}
