package api

import (
	"simplebank/util"

	"github.com/go-playground/validator/v10"
)

var validCurrency validator.Func = func(fieldLevel validator.FieldLevel) bool {
	currency, ok := fieldLevel.Field().Interface().(string)
	if ok {
		util.IsSupportedCurrency(currency)
		return true
	}
	return false
}
