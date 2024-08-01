package decimal

import (
	"reflect"

	"github.com/hypersequent/zen"
)

var (
	DecimalType = "github.com/shopspring/decimal.Decimal"
	DecimalFunc = func(c *zen.Converter, t reflect.Type, validate string, i int) string {
		// Shopspring's decimal type serialises to a string.
		return "z.string()"
	}
)
