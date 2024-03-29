package optional

import (
	"fmt"
	"reflect"

	"github.com/hypersequent/zen"
)

var (
	OptionalType = "4d63.com/optional.Optional"
	OptionalFunc = func(c *zen.Converter, t reflect.Type, s string, g string, validate string, i int) string {
		return fmt.Sprintf("%s.optional().nullish()", c.ConvertType(t.Elem(), s, validate, i))
	}
)
