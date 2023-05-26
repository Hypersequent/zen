package optional_test

import (
	"testing"

	"4d63.com/optional"
	"github.com/hypersequent/zen"
	customoptional "github.com/hypersequent/zen/custom/optional"
	"github.com/stretchr/testify/assert"
)

func TestCustom(t *testing.T) {
	c := zen.NewConverter(map[string]zen.CustomFn{
		customoptional.OptionalType: customoptional.OptionalFunc,
	})

	type Profile struct {
		Bio     string
		Twitter optional.Optional[string]
	}

	type User struct {
		MaybeName    optional.Optional[string]
		MaybeAge     optional.Optional[int]
		MaybeHeight  optional.Optional[float64]
		MaybeProfile optional.Optional[Profile]
	}
	assert.Equal(t,
		`export const ProfileSchema = z.object({
  Bio: z.string(),
  Twitter: z.string().optional().nullish(),
})
export type Profile = z.infer<typeof ProfileSchema>

export const UserSchema = z.object({
  MaybeName: z.string().optional().nullish(),
  MaybeAge: z.number().optional().nullish(),
  MaybeHeight: z.number().optional().nullish(),
  MaybeProfile: ProfileSchema.optional().nullish(),
})
export type User = z.infer<typeof UserSchema>

`,
		c.Convert(User{}))
}
