package priceService

import (
	"context"
	"encoding/json"
	"fmt"
	"prime-erp-core/internal/models"
	priceListRepository "prime-erp-core/internal/repositories/priceList"
	"prime-erp-core/internal/utils"
	"strings"

	"github.com/go-playground/validator/v10"
)

// seam for unit testing: allow stubbing repository function
var updateSubGroupFunc = priceListRepository.UpdatePriceListSubGroups

// jsonValidator ใช้แทน gin's default binding validator ที่ ctx.ShouldBindJSON เคยเรียกให้
// gin ตั้งชื่อ tag เป็น "binding" (ไม่ใช่ "validate" ที่เป็น default ของ go-playground/validator)
// ดู github.com/gin-gonic/gin/binding/default_validator.go: lazyinit() -> SetTagName("binding")
// ต้องตั้งเหมือนกันทุกประการ ไม่งั้น struct tag `binding:"required"` ในโมเดลจะไม่ถูกอ่าน
var jsonValidator = func() *validator.Validate {
	v := validator.New()
	v.SetTagName("binding")
	return v
}()

// bindJSONRequest แทน ctx.ShouldBindJSON เดิม: decode payload แล้ว validate ด้วย struct
// tag "binding" ตัวเดียวกับที่ gin ใช้ ผล error (message/status ปลายทางที่ writeError คืน)
// ต้องเหมือนเดิมทุกประการ เพื่อไม่ให้ response shape ของ 3 endpoint นี้เปลี่ยน
//
// ต้องใช้ json.NewDecoder(...).Decode เหมือน gin/binding/json.go:decodeJSON ทุกประการ
// ไม่ใช่ json.Unmarshal — สอง decoder นี้ให้ error message ต่างกันตอน body ว่าง (EOF vs
// "unexpected end of JSON input") และ Decoder ยอมรับ body ที่มีขยะต่อท้าย JSON object
// ตัวแรกส่วน Unmarshal จะ reject ถ้าสลับไปใช้ Unmarshal validation behaviour จะเปลี่ยนจริง
func bindJSONRequest(jsonPayload string, req interface{}) error {
	if err := json.NewDecoder(strings.NewReader(jsonPayload)).Decode(req); err != nil {
		return &utils.BindingError{
			Message: fmt.Sprintf("Invalid request: %v", err.Error()),
		}
	}

	if err := jsonValidator.Struct(req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var errorMessages []string
			for _, fieldError := range validationErrors {
				errorMessages = append(errorMessages, getValidationErrorMessage(fieldError))
			}
			return &utils.BindingError{
				Message: fmt.Sprintf("Validation failed: %v", errorMessages),
			}
		}
		return &utils.BindingError{
			Message: fmt.Sprintf("Invalid request: %v", err.Error()),
		}
	}

	return nil
}

// UpdatePriceListSubGroup updates a price_list_sub_group record using JSON binding and validation
func UpdatePriceListSubGroup(ctx context.Context, jsonPayload string) (interface{}, error) {
	var req models.UpdatePriceListSubGroupRequest

	if err := bindJSONRequest(jsonPayload, &req); err != nil {
		return nil, err
	}

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
	case "uuid", "uuid4":
		return fmt.Sprintf("%s must be a valid UUID", field)
	default:
		return fmt.Sprintf("%s failed validation for tag '%s'", field, tag)
	}
}
