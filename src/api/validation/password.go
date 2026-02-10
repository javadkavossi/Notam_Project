package validation

import (
	"github.com/go-playground/validator/v10"
	"github.com/hossein-repo/BaseProject/common"
)

func PasswordValidator(fld validator.FieldLevel) bool {
	value, ok := fld.Field().Interface().(string)
	if !ok {
		fld.Param()
		return false
	}

	return common.IsValidPassword(value)

}
