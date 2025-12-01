package priceService

import (
	"fmt"

	"prime-erp-core/internal/models"
	priceListRepository "prime-erp-core/internal/repositories/priceList"
	"prime-erp-core/internal/utils"

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

	subGroupID, err := uuid.Parse(req.SubGroupID)
	if err != nil {
		return nil, &utils.BindingError{
			Message: "subgroup_id must be a valid UUID",
		}
	}

	utils.PrintJSON(req)

	subGroup, err := getLatestSubGroupFunc(subGroupID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest price list sub group: %w", err)
	}

	if subGroup == nil {
		return nil, &utils.BindingError{
			Message: "Price list sub group not found",
		}
	}

	return subGroup, nil
}
