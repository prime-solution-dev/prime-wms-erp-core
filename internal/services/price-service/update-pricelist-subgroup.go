package priceService

import (
	"fmt"
	"prime-erp-core/internal/models"
	priceListRepository "prime-erp-core/internal/repositories/priceList"
	"prime-erp-core/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// seam for unit testing: allow stubbing repository function
var updateSubGroupFunc = priceListRepository.UpdatePriceListSubGroups

// UpdatePriceListSubGroup updates a price_list_sub_group record using Gin binding and validation
func UpdatePriceListSubGroup(ctx *gin.Context) (interface{}, error) {
	var req models.UpdatePriceListSubGroupRequest

	// Use ShouldBindJSON for binding and validation
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// Handle validation errors
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

	utils.PrintJSON(req)

	// Call repository function (batch)
	if err := updateSubGroupFunc(req); err != nil {
		return nil, fmt.Errorf("failed to update price list sub group: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": "Price list sub group updated successfully",
	}, nil
}

// getValidationErrorMessage converts validator.ValidationError to user-friendly message
func getValidationErrorMessage(fieldError validator.FieldError) string {
	field := fieldError.Field()
	tag := fieldError.Tag()

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, fieldError.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", field, fieldError.Param())
	case "omitempty":
		return fmt.Sprintf("%s is invalid", field)
	case "uuid4":
		return fmt.Sprintf("%s must be a valid UUID", field)
	default:
		return fmt.Sprintf("%s failed validation for tag '%s'", field, tag)
	}
}
