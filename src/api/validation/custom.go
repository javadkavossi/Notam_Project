// validation/custom.go
package validation

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

type ValidationError struct {
	Property string `json:"property"`
	Tag      string `json:"tag"`
	Value    string `json:"value"`
	Message  string `json:"message"`
}

// validation/custom.go
func GetValidationErrors(err error) *[]ValidationError {
	var validationErrors []ValidationError
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, err := range ve {
			var el ValidationError
			el.Property = err.Field()
			el.Tag = err.Tag()
			el.Value = err.Param()
			el.Message = getValidationMessage(err) // اضافه کردن پیام کاربرپسند
			validationErrors = append(validationErrors, el)
		}
		return &validationErrors
	}
	return nil
}

func getValidationMessage(fieldError validator.FieldError) string {
	fieldName := strings.ToLower(fieldError.Field())

	switch fieldError.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fieldName)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", fieldName, fieldError.Param())
	case "email":
		return "Invalid email format"
	case "password":
		return "Password does not meet security requirements"
	case "mobile":
		return "Invalid mobile number format"
	default:
		return fmt.Sprintf("Invalid %s", fieldName)
	}
}
