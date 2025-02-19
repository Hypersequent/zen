package decimal_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/hypersequent/zen"
	customDecimal "github.com/hypersequent/zen/custom/decimal"
)

func TestCustom(t *testing.T) {
	opt := zen.WithCustomTypes(map[string]zen.CustomFn{
		customDecimal.DecimalType: customDecimal.DecimalFunc,
	})
	c := zen.NewConverterWithOpts(opt)

	type User struct {
		Money decimal.Decimal
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Money: z.string(),
})
export type User = z.infer<typeof UserSchema>

`,
		c.Convert(User{}))
}
