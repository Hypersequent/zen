package zen

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFieldName(t *testing.T) {
	assert.Equal(t,
		fieldName(reflect.StructField{Name: "RCONPassword"}),
		"RCONPassword",
	)

	assert.Equal(t,
		fieldName(reflect.StructField{Name: "LANMode"}),
		"LANMode",
	)

	assert.Equal(t,
		fieldName(reflect.StructField{Name: "ABC"}),
		"ABC",
	)
}

func TestFieldNameJsonTag(t *testing.T) {
	type S struct {
		NotTheFieldName string `json:"fieldName"`
	}

	assert.Equal(t,
		fieldName(reflect.TypeOf(S{}).Field(0)),
		"fieldName",
	)
}

func TestFieldNameJsonTagOmitEmpty(t *testing.T) {
	type S struct {
		NotTheFieldName string `json:"fieldName,omitempty"`
	}

	assert.Equal(t,
		fieldName(reflect.TypeOf(S{}).Field(0)),
		"fieldName",
	)
}

func TestSchemaName(t *testing.T) {
	assert.Equal(t,
		schemaName("", "User"),
		"UserSchema",
	)
	assert.Equal(t,
		schemaName("Bot", "User"),
		"BotUserSchema",
	)
}

func TestStructSimple(t *testing.T) {
	type User struct {
		Name   string
		Age    int
		Height float64
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Name: z.string(),
  Age: z.number(),
  Height: z.number(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestStructSimpleWithOmittedField(t *testing.T) {
	type User struct {
		Name        string
		Age         int
		Height      float64
		NotExported string `json:"-"`
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Name: z.string(),
  Age: z.number(),
  Height: z.number(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestStructSimplePrefix(t *testing.T) {
	type User struct {
		Name   string
		Age    int
		Height float64
	}
	assert.Equal(t,
		`export const BotUserSchema = z.object({
  Name: z.string(),
  Age: z.number(),
  Height: z.number(),
})
export type BotUser = z.infer<typeof BotUserSchema>

`,
		StructToZodSchema(User{}, WithPrefix("Bot")))
}

func TestNestedStruct(t *testing.T) {
	type HasID struct {
		ID string
	}
	type HasName struct {
		Name string `json:"name"`
	}
	type User struct {
		HasID
		HasName
		Tags []string
	}
	assert.Equal(t,
		`export const HasIDSchema = z.object({
  ID: z.string(),
})
export type HasID = z.infer<typeof HasIDSchema>

export const HasNameSchema = z.object({
  name: z.string(),
})
export type HasName = z.infer<typeof HasNameSchema>

export const UserSchema = z.object({
  Tags: z.string().array().nullable(),
}).merge(HasIDSchema).merge(HasNameSchema)
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestStringArray(t *testing.T) {
	type User struct {
		Tags []string
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Tags: z.string().array().nullable(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestStringNestedArray(t *testing.T) {
	type TagPair [2]string
	type User struct {
		TagPairs []TagPair
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  TagPairs: z.string().array().length(2).array().nullable(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestStructSlice(t *testing.T) {
	type User struct {
		Favourites []struct {
			Name string
		}
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Favourites: z.object({
    Name: z.string(),
  }).array().nullable(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestStructSliceOptional(t *testing.T) {
	type User struct {
		Favourites []struct {
			Name string
		} `json:",omitempty"`
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Favourites: z.object({
    Name: z.string(),
  }).array().optional(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestStructSliceOptionalNullable(t *testing.T) {
	type User struct {
		Favourites *[]struct {
			Name string
		} `json:",omitempty"`
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Favourites: z.object({
    Name: z.string(),
  }).array().optional().nullable(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestStringOptional(t *testing.T) {
	type User struct {
		Name     string
		Nickname string `json:",omitempty"`
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Name: z.string(),
  Nickname: z.string().optional(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestStringNullable(t *testing.T) {
	type User struct {
		Name     string
		Nickname *string
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Name: z.string(),
  Nickname: z.string().nullable(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestStringOptionalNotNullable(t *testing.T) {
	type User struct {
		Name     string
		Nickname *string `json:",omitempty"` // nil values are omitted
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Name: z.string(),
  Nickname: z.string().optional(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestStringOptionalNullable(t *testing.T) {
	type User struct {
		Name     string
		Nickname **string `json:",omitempty"` // nil values are omitted
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Name: z.string(),
  Nickname: z.string().optional().nullable(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestStringArrayNullable(t *testing.T) {
	type User struct {
		Name string
		Tags []*string
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Name: z.string(),
  Tags: z.string().array().nullable(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestNullableWithValidations(t *testing.T) {
	type User struct {
		Name string `validate:"required"`

		PtrMapOptionalNullable1 *map[string]interface{} `json:",omitempty"`
		PtrMapOptionalNullable2 *map[string]interface{} `json:",omitempty" validate:"omitempty,min=2,max=5"`
		PtrMap1                 *map[string]interface{} `validate:"min=2,max=5"`
		PtrMap2                 *map[string]interface{} `json:",omitempty" validate:"min=2,max=5"`
		PtrMapNullable          *map[string]interface{} `validate:"omitempty,min=2,max=5"`

		MapOptional1 map[string]interface{} `json:",omitempty"`
		MapOptional2 map[string]interface{} `json:",omitempty" validate:"omitempty,min=2,max=5"`
		Map1         map[string]interface{} `validate:"min=2,max=5"`
		Map2         map[string]interface{} `json:",omitempty" validate:"min=2,max=5"`
		MapNullable  map[string]interface{} `validate:"omitempty,min=2,max=5"`

		PtrSliceOptionalNullable1 *[]string `json:",omitempty"`
		PtrSliceOptionalNullable2 *[]string `json:",omitempty" validate:"omitempty,min=2,max=5"`
		PtrSlice1                 *[]string `validate:"min=2,max=5"`
		PtrSlice2                 *[]string `json:",omitempty" validate:"min=2,max=5"`
		PtrSliceNullable          *[]string `validate:"omitempty,min=2,max=5"`

		SliceOptional1 []string `json:",omitempty"`
		SliceOptional2 []string `json:",omitempty" validate:"omitempty,min=2,max=5"`
		Slice1         []string `validate:"min=2,max=5"`
		Slice2         []string `json:",omitempty" validate:"min=2,max=5"`
		SliceNullable  []string `validate:"omitempty,min=2,max=5"`

		PtrIntOptional1 *int `json:",omitempty"`
		PtrIntOptional2 *int `json:",omitempty" validate:"omitempty,min=2,max=5"`
		PtrInt1         *int `validate:"min=2,max=5"`
		PtrInt2         *int `json:",omitempty" validate:"min=2,max=5"`
		PtrIntNullable  *int `validate:"omitempty,min=2,max=5"`

		// Not handled by zen for now
		// IntOptionalNullable int `json:",omitempty"`
		// Int1                int `validate:"min=2,max=5"`
		// Int2                int `json:",omitempty" validate:"min=2,max=5"`
		// IntNullable1        int `validate:"omitempty,min=2,max=5"`
		// IntNullable2        int `json:",omitempty" validate:"omitempty,min=2,max=5"`

		PtrStringOptional1 *string `json:",omitempty"`
		PtrStringOptional2 *string `json:",omitempty" validate:"omitempty,min=2,max=5"`
		PtrString1         *string `validate:"min=2,max=5"`
		PtrString2         *string `json:",omitempty" validate:"min=2,max=5"`
		PtrStringNullable  *string `validate:"omitempty,min=2,max=5"`

		// Not handled by zen for now
		// StringOptionalNullable string `json:",omitempty"`
		// String1                string `validate:"min=2,max=5"`
		// String2                string `json:",omitempty" validate:"min=2,max=5"`
		// StringNullable1        string `validate:"omitempty,min=2,max=5"`
		// StringNullable2        string `json:",omitempty" validate:"omitempty,min=2,max=5"`
	}

	assert.Equal(t,
		`export const UserSchema = z.object({
  Name: z.string().min(1),
  PtrMapOptionalNullable1: z.record(z.string(), z.any()).optional().nullable(),
  PtrMapOptionalNullable2: z.record(z.string(), z.any()).refine((val) => Object.keys(val).length >= 2, 'Map too small').refine((val) => Object.keys(val).length <= 5, 'Map too large').optional().nullable(),
  PtrMap1: z.record(z.string(), z.any()).refine((val) => Object.keys(val).length >= 2, 'Map too small').refine((val) => Object.keys(val).length <= 5, 'Map too large'),
  PtrMap2: z.record(z.string(), z.any()).refine((val) => Object.keys(val).length >= 2, 'Map too small').refine((val) => Object.keys(val).length <= 5, 'Map too large'),
  PtrMapNullable: z.record(z.string(), z.any()).refine((val) => Object.keys(val).length >= 2, 'Map too small').refine((val) => Object.keys(val).length <= 5, 'Map too large').nullable(),
  MapOptional1: z.record(z.string(), z.any()).optional(),
  MapOptional2: z.record(z.string(), z.any()).refine((val) => Object.keys(val).length >= 2, 'Map too small').refine((val) => Object.keys(val).length <= 5, 'Map too large').optional(),
  Map1: z.record(z.string(), z.any()).refine((val) => Object.keys(val).length >= 2, 'Map too small').refine((val) => Object.keys(val).length <= 5, 'Map too large'),
  Map2: z.record(z.string(), z.any()).refine((val) => Object.keys(val).length >= 2, 'Map too small').refine((val) => Object.keys(val).length <= 5, 'Map too large'),
  MapNullable: z.record(z.string(), z.any()).refine((val) => Object.keys(val).length >= 2, 'Map too small').refine((val) => Object.keys(val).length <= 5, 'Map too large').nullable(),
  PtrSliceOptionalNullable1: z.string().array().optional().nullable(),
  PtrSliceOptionalNullable2: z.string().array().min(2).max(5).optional().nullable(),
  PtrSlice1: z.string().array().min(2).max(5),
  PtrSlice2: z.string().array().min(2).max(5),
  PtrSliceNullable: z.string().array().min(2).max(5).nullable(),
  SliceOptional1: z.string().array().optional(),
  SliceOptional2: z.string().array().min(2).max(5).optional(),
  Slice1: z.string().array().min(2).max(5),
  Slice2: z.string().array().min(2).max(5),
  SliceNullable: z.string().array().min(2).max(5).nullable(),
  PtrIntOptional1: z.number().optional(),
  PtrIntOptional2: z.number().gte(2).lte(5).optional(),
  PtrInt1: z.number().gte(2).lte(5),
  PtrInt2: z.number().gte(2).lte(5),
  PtrIntNullable: z.number().gte(2).lte(5).nullable(),
  PtrStringOptional1: z.string().optional(),
  PtrStringOptional2: z.string().min(2).max(5).optional(),
  PtrString1: z.string().min(2).max(5),
  PtrString2: z.string().min(2).max(5),
  PtrStringNullable: z.string().min(2).max(5).nullable(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestStringValidations(t *testing.T) {
	type Eq struct {
		Name string `validate:"eq=hello"`
	}
	assert.Equal(t,
		`export const EqSchema = z.object({
  Name: z.string().refine((val) => val === "hello"),
})
export type Eq = z.infer<typeof EqSchema>

`,
		StructToZodSchema(Eq{}))

	type Ne struct {
		Name string `validate:"ne=hello"`
	}
	assert.Equal(t,
		`export const NeSchema = z.object({
  Name: z.string().refine((val) => val !== "hello"),
})
export type Ne = z.infer<typeof NeSchema>

`,
		StructToZodSchema(Ne{}))

	type OneOf struct {
		Name string `validate:"oneof=hello world"`
	}
	assert.Equal(t,
		`export const OneOfSchema = z.object({
  Name: z.enum(["hello", "world"] as const),
})
export type OneOf = z.infer<typeof OneOfSchema>

`,
		StructToZodSchema(OneOf{}))

	type OneOfSeparated struct {
		Name string `validate:"oneof='a b c' 'd e f'"`
	}
	assert.Equal(t,
		`export const OneOfSeparatedSchema = z.object({
  Name: z.enum(["a b c", "d e f"] as const),
})
export type OneOfSeparated = z.infer<typeof OneOfSeparatedSchema>

`,
		StructToZodSchema(OneOfSeparated{}))

	// TODO: This test case is not supported yet even for the go-validator package whose logic
	// I stole to parse the value after oneof=.
	//
	//	type OneOfEscaped struct {
	//		Name string `validate:"oneof='a b c' 'd e f' 'g\\' h'"`
	//	}
	//	assert.Equal(t,
	//		`export const OneOfEscapedSchema = z.object({
	//  Name: z.string().enum(["a b c", "d e f", "g' h"]),
	//})
	//export type OneOfEscaped = z.infer<typeof OneOfEscapedSchema>
	//
	//`,
	//		StructToZodSchema(OneOfEscaped{}))

	// Same story as above.
	//	type OneOfEscaped2 struct {
	//		Name string `validate:"oneof='a b c' 'd e f' 'g\x27 h'"`
	//	}
	//	assert.Equal(t,
	//		`export const OneOfEscapedSchema = z.object({
	//  Name: z.string().enum(["a b c", "d e f", "g' h"]),
	//})
	//export type OneOfEscaped = z.infer<typeof OneOfEscapedSchema>
	//
	//`,
	//		StructToZodSchema(OneOfEscaped2{}))

	type Len struct {
		Name string `validate:"len=5"`
	}
	assert.Equal(t,
		`export const LenSchema = z.object({
  Name: z.string().length(5),
})
export type Len = z.infer<typeof LenSchema>

`,
		StructToZodSchema(Len{}))

	type Min struct {
		Name string `validate:"min=5"`
	}
	assert.Equal(t,
		`export const MinSchema = z.object({
  Name: z.string().min(5),
})
export type Min = z.infer<typeof MinSchema>

`,
		StructToZodSchema(Min{}))

	type Max struct {
		Name string `validate:"max=5"`
	}
	assert.Equal(t,
		`export const MaxSchema = z.object({
  Name: z.string().max(5),
})
export type Max = z.infer<typeof MaxSchema>

`,
		StructToZodSchema(Max{}))

	type MinMax struct {
		Name string `validate:"min=3,max=7"`
	}
	assert.Equal(t,
		`export const MinMaxSchema = z.object({
  Name: z.string().min(3).max(7),
})
export type MinMax = z.infer<typeof MinMaxSchema>

`,
		StructToZodSchema(MinMax{}))

	type Gt struct {
		Name string `validate:"gt=5"`
	}
	assert.Equal(t,
		`export const GtSchema = z.object({
  Name: z.string().min(6),
})
export type Gt = z.infer<typeof GtSchema>

`,
		StructToZodSchema(Gt{}))

	type Gte struct {
		Name string `validate:"gte=5"`
	}
	assert.Equal(t,
		`export const GteSchema = z.object({
  Name: z.string().min(5),
})
export type Gte = z.infer<typeof GteSchema>

`,
		StructToZodSchema(Gte{}))

	type Lt struct {
		Name string `validate:"lt=5"`
	}
	assert.Equal(t,
		`export const LtSchema = z.object({
  Name: z.string().max(4),
})
export type Lt = z.infer<typeof LtSchema>

`,
		StructToZodSchema(Lt{}))

	type Lte struct {
		Name string `validate:"lte=5"`
	}
	assert.Equal(t,
		`export const LteSchema = z.object({
  Name: z.string().max(5),
})
export type Lte = z.infer<typeof LteSchema>

`,
		StructToZodSchema(Lte{}))

	type Contains struct {
		Name string `validate:"contains=hello"`
	}
	assert.Equal(t,
		`export const ContainsSchema = z.object({
  Name: z.string().includes("hello"),
})
export type Contains = z.infer<typeof ContainsSchema>

`,
		StructToZodSchema(Contains{}))

	type EndsWith struct {
		Name string `validate:"endswith=hello"`
	}
	assert.Equal(t,
		`export const EndsWithSchema = z.object({
  Name: z.string().endsWith("hello"),
})
export type EndsWith = z.infer<typeof EndsWithSchema>

`,
		StructToZodSchema(EndsWith{}))

	type StartsWith struct {
		Name string `validate:"startswith=hello"`
	}
	assert.Equal(t,
		`export const StartsWithSchema = z.object({
  Name: z.string().startsWith("hello"),
})
export type StartsWith = z.infer<typeof StartsWithSchema>

`,
		StructToZodSchema(StartsWith{}))

	type Bad struct {
		Name string `validate:"bad=hello"`
	}
	assert.Panics(t, func() {
		StructToZodSchema(Bad{})
	})

	type Required struct {
		Name string `validate:"required"`
	}
	assert.Equal(t,
		`export const RequiredSchema = z.object({
  Name: z.string().min(1),
})
export type Required = z.infer<typeof RequiredSchema>

`,
		StructToZodSchema(Required{}))

	type Email struct {
		Name string `validate:"email"`
	}
	assert.Equal(t,
		`export const EmailSchema = z.object({
  Name: z.string().email(),
})
export type Email = z.infer<typeof EmailSchema>

`,
		StructToZodSchema(Email{}))

	type URL struct {
		Name string `validate:"url"`
	}
	assert.Equal(t,
		`export const URLSchema = z.object({
  Name: z.string().url(),
})
export type URL = z.infer<typeof URLSchema>

`,
		StructToZodSchema(URL{}))

	type IPv4 struct {
		Name string `validate:"ipv4"`
	}
	assert.Equal(t,
		`export const IPv4Schema = z.object({
  Name: z.string().ip({ version: "v4" }),
})
export type IPv4 = z.infer<typeof IPv4Schema>

`,
		StructToZodSchema(IPv4{}))

	type IPv6 struct {
		Name string `validate:"ipv6"`
	}
	assert.Equal(t,
		`export const IPv6Schema = z.object({
  Name: z.string().ip({ version: "v6" }),
})
export type IPv6 = z.infer<typeof IPv6Schema>

`,
		StructToZodSchema(IPv6{}))

	type IP4Addr struct {
		Name string `validate:"ip4_addr"`
	}
	assert.Equal(t,
		`export const IP4AddrSchema = z.object({
  Name: z.string().ip({ version: "v4" }),
})
export type IP4Addr = z.infer<typeof IP4AddrSchema>

`,
		StructToZodSchema(IP4Addr{}))

	type IP6Addr struct {
		Name string `validate:"ip6_addr"`
	}
	assert.Equal(t,
		`export const IP6AddrSchema = z.object({
  Name: z.string().ip({ version: "v6" }),
})
export type IP6Addr = z.infer<typeof IP6AddrSchema>

`,
		StructToZodSchema(IP6Addr{}))

	type IP struct {
		Name string `validate:"ip"`
	}
	assert.Equal(t,
		`export const IPSchema = z.object({
  Name: z.string().ip(),
})
export type IP = z.infer<typeof IPSchema>

`,
		StructToZodSchema(IP{}))

	type IPAddr struct {
		Name string `validate:"ip_addr"`
	}
	assert.Equal(t,
		`export const IPAddrSchema = z.object({
  Name: z.string().ip(),
})
export type IPAddr = z.infer<typeof IPAddrSchema>

`,
		StructToZodSchema(IPAddr{}))

	type HttpURL struct {
		Name string `validate:"http_url"`
	}
	assert.Equal(t,
		`export const HttpURLSchema = z.object({
  Name: z.string().url(),
})
export type HttpURL = z.infer<typeof HttpURLSchema>

`,
		StructToZodSchema(HttpURL{}))

	type URLEncoded struct {
		Name string `validate:"url_encoded"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const URLEncodedSchema = z.object({
  Name: z.string().regex(/%s/),
})
export type URLEncoded = z.infer<typeof URLEncodedSchema>

`, uRLEncodedRegexString),
		StructToZodSchema(URLEncoded{}))

	type Alpha struct {
		Name string `validate:"alpha"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const AlphaSchema = z.object({
  Name: z.string().regex(/%s/),
})
export type Alpha = z.infer<typeof AlphaSchema>

`, alphaRegexString),
		StructToZodSchema(Alpha{}))

	type AlphaNum struct {
		Name string `validate:"alphanum"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const AlphaNumSchema = z.object({
  Name: z.string().regex(/%s/),
})
export type AlphaNum = z.infer<typeof AlphaNumSchema>

`, alphaNumericRegexString),
		StructToZodSchema(AlphaNum{}))

	type AlphaNumUnicode struct {
		Name string `validate:"alphanumunicode"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const AlphaNumUnicodeSchema = z.object({
  Name: z.string().regex(/%s/),
})
export type AlphaNumUnicode = z.infer<typeof AlphaNumUnicodeSchema>

`, alphaUnicodeNumericRegexString),
		StructToZodSchema(AlphaNumUnicode{}))

	type AlphaUnicode struct {
		Name string `validate:"alphaunicode"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const AlphaUnicodeSchema = z.object({
  Name: z.string().regex(/%s/),
})
export type AlphaUnicode = z.infer<typeof AlphaUnicodeSchema>

`, alphaUnicodeRegexString),
		StructToZodSchema(AlphaUnicode{}))

	type ASCII struct {
		Name string `validate:"ascii"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const ASCIISchema = z.object({
  Name: z.string().regex(/%s/),
})
export type ASCII = z.infer<typeof ASCIISchema>

`, aSCIIRegexString),
		StructToZodSchema(ASCII{}))

	type Boolean struct {
		Name string `validate:"boolean"`
	}
	assert.Equal(t,
		`export const BooleanSchema = z.object({
  Name: z.enum(['true', 'false']),
})
export type Boolean = z.infer<typeof BooleanSchema>

`,
		StructToZodSchema(Boolean{}))

	type Lowercase struct {
		Name string `validate:"lowercase"`
	}
	assert.Equal(t,
		`export const LowercaseSchema = z.object({
  Name: z.string().refine((val) => val === val.toLowerCase()),
})
export type Lowercase = z.infer<typeof LowercaseSchema>

`,
		StructToZodSchema(Lowercase{}))

	type Number struct {
		Name string `validate:"number"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const NumberSchema = z.object({
  Name: z.string().regex(/%s/),
})
export type Number = z.infer<typeof NumberSchema>

`, numberRegexString),
		StructToZodSchema(Number{}))

	type Numeric struct {
		Name string `validate:"numeric"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const NumericSchema = z.object({
  Name: z.string().regex(/%s/),
})
export type Numeric = z.infer<typeof NumericSchema>

`, numericRegexString),
		StructToZodSchema(Numeric{}))

	type Uppercase struct {
		Name string `validate:"uppercase"`
	}
	assert.Equal(t,
		`export const UppercaseSchema = z.object({
  Name: z.string().refine((val) => val === val.toUpperCase()),
})
export type Uppercase = z.infer<typeof UppercaseSchema>

`,
		StructToZodSchema(Uppercase{}))

	type Base64 struct {
		Name string `validate:"base64"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const Base64Schema = z.object({
  Name: z.string().regex(/%s/),
})
export type Base64 = z.infer<typeof Base64Schema>

`, base64RegexString),
		StructToZodSchema(Base64{}))

	type mongodb struct {
		Name string `validate:"mongodb"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const mongodbSchema = z.object({
  Name: z.string().regex(/%s/),
})
export type mongodb = z.infer<typeof mongodbSchema>

`, mongodbRegexString),
		StructToZodSchema(mongodb{}))

	type datetime struct {
		Name string `validate:"datetime"`
	}
	assert.Equal(t,
		`export const datetimeSchema = z.object({
  Name: z.string().datetime(),
})
export type datetime = z.infer<typeof datetimeSchema>

`,
		StructToZodSchema(datetime{}))

	type Hexadecimal struct {
		Name string `validate:"hexadecimal"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const HexadecimalSchema = z.object({
  Name: z.string().regex(/%s/),
})
export type Hexadecimal = z.infer<typeof HexadecimalSchema>

`, hexadecimalRegexString),
		StructToZodSchema(Hexadecimal{}))

	type json struct {
		Name string `validate:"json"`
	}
	assert.Equal(t,
		`export const jsonSchema = z.object({
  Name: z.string().refine((val) => { try { JSON.parse(val); return true } catch { return false } }),
})
export type json = z.infer<typeof jsonSchema>

`,
		StructToZodSchema(json{}))

	type Latitude struct {
		Name string `validate:"latitude"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const LatitudeSchema = z.object({
  Name: z.string().regex(/%s/),
})
export type Latitude = z.infer<typeof LatitudeSchema>

`, latitudeRegexString),
		StructToZodSchema(Latitude{}))

	type Longitude struct {
		Name string `validate:"longitude"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const LongitudeSchema = z.object({
  Name: z.string().regex(/%s/),
})
export type Longitude = z.infer<typeof LongitudeSchema>

`, longitudeRegexString),
		StructToZodSchema(Longitude{}))

	type UUID struct {
		Name string `validate:"uuid"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const UUIDSchema = z.object({
  Name: z.string().regex(/%s/),
})
export type UUID = z.infer<typeof UUIDSchema>

`, uUIDRegexString),
		StructToZodSchema(UUID{}))

	type UUID3 struct {
		Name string `validate:"uuid3"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const UUID3Schema = z.object({
  Name: z.string().regex(/%s/),
})
export type UUID3 = z.infer<typeof UUID3Schema>

`, uUID3RegexString),
		StructToZodSchema(UUID3{}))

	type UUID3RFC4122 struct {
		Name string `validate:"uuid3_rfc4122"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const UUID3RFC4122Schema = z.object({
  Name: z.string().regex(/%s/),
})
export type UUID3RFC4122 = z.infer<typeof UUID3RFC4122Schema>

`, uUID3RFC4122RegexString),
		StructToZodSchema(UUID3RFC4122{}))

	type UUID4 struct {
		Name string `validate:"uuid4"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const UUID4Schema = z.object({
  Name: z.string().regex(/%s/),
})
export type UUID4 = z.infer<typeof UUID4Schema>

`, uUID4RegexString),
		StructToZodSchema(UUID4{}))

	type UUID4RFC4122 struct {
		Name string `validate:"uuid4_rfc4122"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const UUID4RFC4122Schema = z.object({
  Name: z.string().regex(/%s/),
})
export type UUID4RFC4122 = z.infer<typeof UUID4RFC4122Schema>

`, uUID4RFC4122RegexString),
		StructToZodSchema(UUID4RFC4122{}))

	type UUID5 struct {
		Name string `validate:"uuid5"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const UUID5Schema = z.object({
  Name: z.string().regex(/%s/),
})
export type UUID5 = z.infer<typeof UUID5Schema>

`, uUID5RegexString),
		StructToZodSchema(UUID5{}))

	type UUID5RFC4122 struct {
		Name string `validate:"uuid5_rfc4122"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const UUID5RFC4122Schema = z.object({
  Name: z.string().regex(/%s/),
})
export type UUID5RFC4122 = z.infer<typeof UUID5RFC4122Schema>

`, uUID5RFC4122RegexString),
		StructToZodSchema(UUID5RFC4122{}))

	type UUIDRFC4122 struct {
		Name string `validate:"uuid_rfc4122"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const UUIDRFC4122Schema = z.object({
  Name: z.string().regex(/%s/),
})
export type UUIDRFC4122 = z.infer<typeof UUIDRFC4122Schema>

`, uUIDRFC4122RegexString),
		StructToZodSchema(UUIDRFC4122{}))

	type MD4 struct {
		Name string `validate:"md4"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const MD4Schema = z.object({
  Name: z.string().regex(/%s/),
})
export type MD4 = z.infer<typeof MD4Schema>

`, md4RegexString),
		StructToZodSchema(MD4{}))

	type MD5 struct {
		Name string `validate:"md5"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const MD5Schema = z.object({
  Name: z.string().regex(/%s/),
})
export type MD5 = z.infer<typeof MD5Schema>

`, md5RegexString),
		StructToZodSchema(MD5{}))

	type SHA256 struct {
		Name string `validate:"sha256"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const SHA256Schema = z.object({
  Name: z.string().regex(/%s/),
})
export type SHA256 = z.infer<typeof SHA256Schema>

`, sha256RegexString),
		StructToZodSchema(SHA256{}))

	type SHA384 struct {
		Name string `validate:"sha384"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const SHA384Schema = z.object({
  Name: z.string().regex(/%s/),
})
export type SHA384 = z.infer<typeof SHA384Schema>

`, sha384RegexString),
		StructToZodSchema(SHA384{}))

	type SHA512 struct {
		Name string `validate:"sha512"`
	}
	assert.Equal(t,
		fmt.Sprintf(`export const SHA512Schema = z.object({
  Name: z.string().regex(/%s/),
})
export type SHA512 = z.infer<typeof SHA512Schema>

`, sha512RegexString),
		StructToZodSchema(SHA512{}))

	type Bad2 struct {
		Name string `validate:"bad2"`
	}
	assert.Panics(t, func() {
		StructToZodSchema(Bad2{})
	})
}

func TestNumberValidations(t *testing.T) {
	type User1 struct {
		Age int `validate:"gte=18,lte=60"`
	}
	assert.Equal(t,
		`export const User1Schema = z.object({
  Age: z.number().gte(18).lte(60),
})
export type User1 = z.infer<typeof User1Schema>

`, StructToZodSchema(User1{}))

	type User2 struct {
		Age int `validate:"gt=18,lt=60"`
	}
	assert.Equal(t,
		`export const User2Schema = z.object({
  Age: z.number().gt(18).lt(60),
})
export type User2 = z.infer<typeof User2Schema>

`, StructToZodSchema(User2{}))

	type User3 struct {
		Age int `validate:"eq=18"`
	}
	assert.Equal(t,
		`export const User3Schema = z.object({
  Age: z.number().refine((val) => val === 18),
})
export type User3 = z.infer<typeof User3Schema>

`, StructToZodSchema(User3{}))

	type User4 struct {
		Age int `validate:"ne=18"`
	}
	assert.Equal(t,
		`export const User4Schema = z.object({
  Age: z.number().refine((val) => val !== 18),
})
export type User4 = z.infer<typeof User4Schema>

`, StructToZodSchema(User4{}))

	type User5 struct {
		Age int `validate:"oneof=18 19 20"`
	}
	assert.Equal(t,
		`export const User5Schema = z.object({
  Age: z.number().refine((val) => [18, 19, 20].includes(val)),
})
export type User5 = z.infer<typeof User5Schema>

`, StructToZodSchema(User5{}))

	type User6 struct {
		Age int `validate:"min=18,max=60"`
	}
	assert.Equal(t,
		`export const User6Schema = z.object({
  Age: z.number().gte(18).lte(60),
})
export type User6 = z.infer<typeof User6Schema>

`, StructToZodSchema(User6{}))

	type User7 struct {
		Age int `validate:"len=18"`
	}
	assert.Equal(t,
		`export const User7Schema = z.object({
  Age: z.number().refine((val) => val === 18),
})
export type User7 = z.infer<typeof User7Schema>

`, StructToZodSchema(User7{}))

	type User8 struct {
		Age int `validate:"bad=18"`
	}
	assert.Panics(t, func() {
		StructToZodSchema(User8{})
	})
}

func TestInterfaceAny(t *testing.T) {
	type User struct {
		Name     string
		Metadata interface{}
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Name: z.string(),
  Metadata: z.any(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestInterfacePointerAny(t *testing.T) {
	type User struct {
		Name     string
		Metadata *interface{}
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Name: z.string(),
  Metadata: z.any(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestInterfaceEmptyAny(t *testing.T) {
	type User struct {
		Name     string
		Metadata interface{} `json:",omitempty"`
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Name: z.string(),
  Metadata: z.any(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestInterfacePointerEmptyAny(t *testing.T) {
	type User struct {
		Name     string
		Metadata *interface{} `json:",omitempty"`
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Name: z.string(),
  Metadata: z.any(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestMapStringToString(t *testing.T) {
	type User struct {
		Name     string
		Metadata map[string]string
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Name: z.string(),
  Metadata: z.record(z.string(), z.string()).nullable(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestMapStringToInterface(t *testing.T) {
	type User struct {
		Name     string
		Metadata map[string]interface{}
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Name: z.string(),
  Metadata: z.record(z.string(), z.any()).nullable(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestMapWithStruct(t *testing.T) {
	type PostWithMetaData struct {
		Title string
	}
	type User struct {
		MapWithStruct map[string]PostWithMetaData
	}
	assert.Equal(t,
		`export const PostWithMetaDataSchema = z.object({
  Title: z.string(),
})
export type PostWithMetaData = z.infer<typeof PostWithMetaDataSchema>

export const UserSchema = z.object({
  MapWithStruct: z.record(z.string(), PostWithMetaDataSchema).nullable(),
})
export type User = z.infer<typeof UserSchema>

`, StructToZodSchema(User{}))
}

func TestMapWithValidations(t *testing.T) {
	type Required struct {
		Map map[string]string `validate:"required"`
	}
	assert.Equal(t,
		`export const RequiredSchema = z.object({
  Map: z.record(z.string(), z.string()).refine((val) => Object.keys(val).length > 0, 'Empty map'),
})
export type Required = z.infer<typeof RequiredSchema>

`, StructToZodSchema(Required{}))

	type Min struct {
		Map map[string]string `validate:"min=1"`
	}
	assert.Equal(t,
		`export const MinSchema = z.object({
  Map: z.record(z.string(), z.string()).refine((val) => Object.keys(val).length >= 1, 'Map too small'),
})
export type Min = z.infer<typeof MinSchema>

`, StructToZodSchema(Min{}))

	type Max struct {
		Map map[string]string `validate:"max=1"`
	}
	assert.Equal(t,
		`export const MaxSchema = z.object({
  Map: z.record(z.string(), z.string()).refine((val) => Object.keys(val).length <= 1, 'Map too large'),
})
export type Max = z.infer<typeof MaxSchema>

`, StructToZodSchema(Max{}))

	type Len struct {
		Map map[string]string `validate:"len=1"`
	}
	assert.Equal(t,
		`export const LenSchema = z.object({
  Map: z.record(z.string(), z.string()).refine((val) => Object.keys(val).length === 1, 'Map wrong size'),
})
export type Len = z.infer<typeof LenSchema>

`, StructToZodSchema(Len{}))

	type MinMax struct {
		Map map[string]string `validate:"min=1,max=2"`
	}
	assert.Equal(t,
		`export const MinMaxSchema = z.object({
  Map: z.record(z.string(), z.string()).refine((val) => Object.keys(val).length >= 1, 'Map too small').refine((val) => Object.keys(val).length <= 2, 'Map too large'),
})
export type MinMax = z.infer<typeof MinMaxSchema>

`, StructToZodSchema(MinMax{}))

	type Eq struct {
		Map map[string]string `validate:"eq=1"`
	}
	assert.Equal(t,
		`export const EqSchema = z.object({
  Map: z.record(z.string(), z.string()).refine((val) => Object.keys(val).length === 1, 'Map wrong size'),
})
export type Eq = z.infer<typeof EqSchema>

`, StructToZodSchema(Eq{}))

	type Ne struct {
		Map map[string]string `validate:"ne=1"`
	}
	assert.Equal(t,
		`export const NeSchema = z.object({
  Map: z.record(z.string(), z.string()).refine((val) => Object.keys(val).length !== 1, 'Map wrong size'),
})
export type Ne = z.infer<typeof NeSchema>

`, StructToZodSchema(Ne{}))

	type Gt struct {
		Map map[string]string `validate:"gt=1"`
	}
	assert.Equal(t,
		`export const GtSchema = z.object({
  Map: z.record(z.string(), z.string()).refine((val) => Object.keys(val).length > 1, 'Map too small'),
})
export type Gt = z.infer<typeof GtSchema>

`, StructToZodSchema(Gt{}))

	type Gte struct {
		Map map[string]string `validate:"gte=1"`
	}
	assert.Equal(t,
		`export const GteSchema = z.object({
  Map: z.record(z.string(), z.string()).refine((val) => Object.keys(val).length >= 1, 'Map too small'),
})
export type Gte = z.infer<typeof GteSchema>

`, StructToZodSchema(Gte{}))

	type Lt struct {
		Map map[string]string `validate:"lt=1"`
	}
	assert.Equal(t,
		`export const LtSchema = z.object({
  Map: z.record(z.string(), z.string()).refine((val) => Object.keys(val).length < 1, 'Map too large'),
})
export type Lt = z.infer<typeof LtSchema>

`, StructToZodSchema(Lt{}))

	type Lte struct {
		Map map[string]string `validate:"lte=1"`
	}
	assert.Equal(t,
		`export const LteSchema = z.object({
  Map: z.record(z.string(), z.string()).refine((val) => Object.keys(val).length <= 1, 'Map too large'),
})
export type Lte = z.infer<typeof LteSchema>

`, StructToZodSchema(Lte{}))

	type Bad struct {
		Map map[string]string `validate:"bad=1"`
	}
	assert.Panics(t, func() { StructToZodSchema(Bad{}) })

	type Dive1 struct {
		Map map[string]string `validate:"dive,min=2"`
	}
	assert.Equal(t,
		`export const Dive1Schema = z.object({
  Map: z.record(z.string(), z.string().min(2)).nullable(),
})
export type Dive1 = z.infer<typeof Dive1Schema>

`, StructToZodSchema(Dive1{}))

	type Dive2 struct {
		Map []map[string]string `validate:"required,dive,min=2,dive,min=3"`
	}
	assert.Equal(t,
		`export const Dive2Schema = z.object({
  Map: z.record(z.string(), z.string().min(3)).refine((val) => Object.keys(val).length >= 2, 'Map too small').array(),
})
export type Dive2 = z.infer<typeof Dive2Schema>

`, StructToZodSchema(Dive2{}))

	type Dive3 struct {
		Map []map[string]string `validate:"required,dive,min=2,dive,keys,min=3,endkeys,max=4"`
	}
	assert.Equal(t,
		`export const Dive3Schema = z.object({
  Map: z.record(z.string().min(3), z.string().max(4)).refine((val) => Object.keys(val).length >= 2, 'Map too small').array(),
})
export type Dive3 = z.infer<typeof Dive3Schema>

`, StructToZodSchema(Dive3{}))
}

func TestMapWithNonStringKey(t *testing.T) {
	type Map1 struct {
		Name     string
		Metadata map[int]string
	}

	assert.Equal(t,
		`export const Map1Schema = z.object({
  Name: z.string(),
  Metadata: z.record(z.coerce.number(), z.string()).nullable(),
})
export type Map1 = z.infer<typeof Map1Schema>

`, StructToZodSchema(Map1{}))

	type Map2 struct {
		Name     string
		Metadata map[time.Time]string
	}

	assert.Equal(t,
		`export const Map2Schema = z.object({
  Name: z.string(),
  Metadata: z.record(z.coerce.date(), z.string()).nullable(),
})
export type Map2 = z.infer<typeof Map2Schema>

`, StructToZodSchema(Map2{}))

	type Map3 struct {
		Name     string
		Metadata map[float64]string
	}

	assert.Equal(t,
		`export const Map3Schema = z.object({
  Name: z.string(),
  Metadata: z.record(z.coerce.number(), z.string()).nullable(),
})
export type Map3 = z.infer<typeof Map3Schema>

`, StructToZodSchema(Map3{}))
}

func TestGetValidateKeys(t *testing.T) {
	assert.Equal(t, "min=3", getValidateKeys("dive,keys,min=3,endkeys,max=4"))
	assert.Equal(t, "min=3,max=5", getValidateKeys("dive,keys,min=3,max=5,endkeys,max=4"))
	assert.Equal(t, "min=3", getValidateKeys("dive,keys,min=3,endkeys"))
	assert.Equal(t, "min=3,max=5", getValidateKeys("dive,keys,min=3,max=5,endkeys"))
	assert.Equal(t, "", getValidateKeys("dive,keys,endkeys,max=4"))
	assert.Equal(t, "", getValidateKeys("dive,max=4"))
	assert.Equal(t, "min=3", getValidateKeys("dive,keys,min=3,endkeys,max=4,dive,keys,min=3,endkeys,max=4"))
	assert.Equal(t, "min=3,max=5", getValidateKeys("dive,keys,min=3,max=5,endkeys,max=4,dive,keys,min=3,max=5,endkeys,max=4"))
	assert.Equal(t, "min=3", getValidateKeys("dive,keys,min=3,endkeys,dive,keys,min=3,endkeys"))
	assert.Equal(t, "min=3,max=5", getValidateKeys("dive,keys,min=3,max=5,endkeys,dive,keys,min=3,max=5,endkeys"))
	assert.Equal(t, "", getValidateKeys("dive,keys,endkeys,max=4,dive,keys,endkeys,max=4"))
	assert.Equal(t, "min=3", getValidateKeys("min=2,dive,keys,min=3,endkeys,max=4"))
}

func TestGetValidateValues(t *testing.T) {
	assert.Equal(t, "max=4", getValidateValues("dive,keys,min=3,endkeys,max=4"))
	assert.Equal(t, "max=4", getValidateValues("dive,keys,min=3,max=5,endkeys,max=4"))
	assert.Equal(t, "", getValidateValues("dive,keys,min=3,endkeys"))
	assert.Equal(t, "", getValidateValues("dive,keys,min=3,max=5,endkeys"))
	assert.Equal(t, "max=4", getValidateValues("dive,keys,endkeys,max=4"))

	assert.Equal(t, "max=4", getValidateValues("dive,keys,min=3,endkeys,max=4,dive,keys,min=3,endkeys,max=4"))
	assert.Equal(t, "min=3,max=4", getValidateValues("dive,keys,min=3,max=5,endkeys,min=3,max=4,dive,keys,min=3,max=5,endkeys,max=4"))
	assert.Equal(t, "", getValidateValues("dive,keys,min=3,endkeys,dive,keys,min=3,endkeys"))
	assert.Equal(t, "", getValidateValues("dive,keys,min=3,max=5,endkeys,dive,keys,min=3,max=5,endkeys"))
	assert.Equal(t, "max=4", getValidateValues("dive,keys,endkeys,max=4,dive,keys,endkeys,max=4"))

	assert.Equal(t, "min=3", getValidateValues("min=2,dive,min=3"))
	assert.Equal(t, "min=3,max=4", getValidateValues("dive,min=3,max=4,dive,min=4,max=5"))
	assert.Equal(t, "max=4", getValidateValues("min=2,dive,keys,min=3,endkeys,max=4"))
}

func TestGetValidateCurrent(t *testing.T) {
	assert.Equal(t, "required", getValidateCurrent("required,dive,min=2,dive,min=3"))
	assert.Equal(t, "", getValidateCurrent("dive,min=2,dive,min=3,max=4"))
	assert.Equal(t, "min=2,max=3", getValidateCurrent("min=2,max=3,dive,min=2,dive,min=3,max=4"))
}

func TestEverything(t *testing.T) {
	// The order matters PostWithMetaData needs to be declared after post otherwise it will raise a
	// `Block-scoped variable 'Post' used before its declaration.` typescript error.
	type Post struct {
		Title string
	}
	type PostWithMetaData struct {
		Title string
		Post  Post
	}
	type User struct {
		Name                 string
		Nickname             *string // pointers become optional
		Age                  int
		Height               float64
		OldPostWithMetaData  PostWithMetaData
		Tags                 []string
		TagsOptional         []string   `json:",omitempty"` // slices with omitempty cannot be null
		TagsOptionalNullable *[]string  `json:",omitempty"` // pointers to slices with omitempty can be null or undefined
		Favourites           []struct { // nested structs are kept inline
			Name string
		}
		Posts                         []Post             // external structs are emitted as separate exports
		Post                          Post               `json:",omitempty"` // this tag is ignored because structs don't have an empty value
		PostOptional                  *Post              `json:",omitempty"` // single struct pointers with omitempty cannot be null
		PostOptionalNullable          **Post             `json:",omitempty"` // double struct pointers with omitempty can be null
		Metadata                      map[string]string  // maps can be null
		MetadataOptional              map[string]string  `json:",omitempty"` // maps with omitempty cannot be null
		MetadataOptionalNullable      *map[string]string `json:",omitempty"` // pointers to maps with omitempty can be null or undefined
		ExtendedProps                 interface{}        // interfaces are just "any" even though they can be null
		ExtendedPropsOptional         interface{}        `json:",omitempty"` // interfaces with omitempty are still just "any"
		ExtendedPropsNullable         *interface{}       // pointers to interfaces are just "any"
		ExtendedPropsOptionalNullable *interface{}       `json:",omitempty"` // pointers to interfaces with omitempty are also just "any"
		ExtendedPropsVeryIndirect     ****interface{}    // interfaces are always "any" no matter the levels of indirection
		NewPostWithMetaData           PostWithMetaData
		VeryNewPost                   Post
		MapWithStruct                 map[string]PostWithMetaData
	}

	assert.Equal(t,
		`export const PostSchema = z.object({
  Title: z.string(),
})
export type Post = z.infer<typeof PostSchema>

export const PostWithMetaDataSchema = z.object({
  Title: z.string(),
  Post: PostSchema,
})
export type PostWithMetaData = z.infer<typeof PostWithMetaDataSchema>

export const UserSchema = z.object({
  Name: z.string(),
  Nickname: z.string().nullable(),
  Age: z.number(),
  Height: z.number(),
  OldPostWithMetaData: PostWithMetaDataSchema,
  Tags: z.string().array().nullable(),
  TagsOptional: z.string().array().optional(),
  TagsOptionalNullable: z.string().array().optional().nullable(),
  Favourites: z.object({
    Name: z.string(),
  }).array().nullable(),
  Posts: PostSchema.array().nullable(),
  Post: PostSchema,
  PostOptional: PostSchema.optional(),
  PostOptionalNullable: PostSchema.optional().nullable(),
  Metadata: z.record(z.string(), z.string()).nullable(),
  MetadataOptional: z.record(z.string(), z.string()).optional(),
  MetadataOptionalNullable: z.record(z.string(), z.string()).optional().nullable(),
  ExtendedProps: z.any(),
  ExtendedPropsOptional: z.any(),
  ExtendedPropsNullable: z.any(),
  ExtendedPropsOptionalNullable: z.any(),
  ExtendedPropsVeryIndirect: z.any(),
  NewPostWithMetaData: PostWithMetaDataSchema,
  VeryNewPost: PostSchema,
  MapWithStruct: z.record(z.string(), PostWithMetaDataSchema).nullable(),
})
export type User = z.infer<typeof UserSchema>

`, StructToZodSchema(User{}))
}

func TestEverythingWithValidations(t *testing.T) {
	// The order matters PostWithMetaData needs to be declared after post otherwise it will raise a
	// `Block-scoped variable 'Post' used before its declaration.` typescript error.
	type Post struct {
		Title string `validate:"required"`
	}
	type PostWithMetaData struct {
		Title string `validate:"required"`
		Post  Post
	}
	type User struct {
		Name                 string           `validate:"required"`
		Nickname             *string          // pointers become optional
		Age                  int              `validate:"required,min=18"`
		Height               float64          `validate:"required,min=1.5"`
		OldPostWithMetaData  PostWithMetaData `validate:"required"`
		Tags                 []string         `validate:"required,min=1"`
		TagsOptional         []string         `json:",omitempty"` // slices with omitempty cannot be null
		TagsOptionalNullable *[]string        `json:",omitempty"` // pointers to slices with omitempty can be null or undefined
		Favourites           []struct {       // nested structs are kept inline
			Name string `validate:"required"`
		}
		Posts                         []Post             `validate:"required"` // external structs are emitted as separate exports
		Post                          Post               `json:",omitempty"`   // this tag is ignored because structs don't have an empty value
		PostOptional                  *Post              `json:",omitempty"`   // single struct pointers with omitempty cannot be null
		PostOptionalNullable          **Post             `json:",omitempty"`   // double struct pointers with omitempty can be null
		Metadata                      map[string]string  // maps can be null
		MetadataLength                map[string]string  `validate:"required,min=1,max=10"` // maps with key length 1 to 10
		MetadataOptional              map[string]string  `json:",omitempty"`                // maps with omitempty cannot be null
		MetadataOptionalNullable      *map[string]string `json:",omitempty"`                // pointers to maps with omitempty can be null or undefined
		ExtendedProps                 interface{}        // interfaces are just "any" even though they can be null
		ExtendedPropsOptional         interface{}        `json:",omitempty"` // interfaces with omitempty are still just "any"
		ExtendedPropsNullable         *interface{}       // pointers to interfaces are just "any"
		ExtendedPropsOptionalNullable *interface{}       `json:",omitempty"` // pointers to interfaces with omitempty are also just "any"
		ExtendedPropsVeryIndirect     ****interface{}    // interfaces are always "any" no matter the levels of indirection
		NewPostWithMetaData           PostWithMetaData
		VeryNewPost                   Post
		MapWithStruct                 map[string]PostWithMetaData
	}
	assert.Equal(t,
		`export const PostSchema = z.object({
  Title: z.string().min(1),
})
export type Post = z.infer<typeof PostSchema>

export const PostWithMetaDataSchema = z.object({
  Title: z.string().min(1),
  Post: PostSchema,
})
export type PostWithMetaData = z.infer<typeof PostWithMetaDataSchema>

export const UserSchema = z.object({
  Name: z.string().min(1),
  Nickname: z.string().nullable(),
  Age: z.number().gte(18).refine((val) => val !== 0),
  Height: z.number().gte(1.5).refine((val) => val !== 0),
  OldPostWithMetaData: PostWithMetaDataSchema,
  Tags: z.string().array().min(1),
  TagsOptional: z.string().array().optional(),
  TagsOptionalNullable: z.string().array().optional().nullable(),
  Favourites: z.object({
    Name: z.string().min(1),
  }).array().nullable(),
  Posts: PostSchema.array(),
  Post: PostSchema,
  PostOptional: PostSchema.optional(),
  PostOptionalNullable: PostSchema.optional().nullable(),
  Metadata: z.record(z.string(), z.string()).nullable(),
  MetadataLength: z.record(z.string(), z.string()).refine((val) => Object.keys(val).length > 0, 'Empty map').refine((val) => Object.keys(val).length >= 1, 'Map too small').refine((val) => Object.keys(val).length <= 10, 'Map too large'),
  MetadataOptional: z.record(z.string(), z.string()).optional(),
  MetadataOptionalNullable: z.record(z.string(), z.string()).optional().nullable(),
  ExtendedProps: z.any(),
  ExtendedPropsOptional: z.any(),
  ExtendedPropsNullable: z.any(),
  ExtendedPropsOptionalNullable: z.any(),
  ExtendedPropsVeryIndirect: z.any(),
  NewPostWithMetaData: PostWithMetaDataSchema,
  VeryNewPost: PostSchema,
  MapWithStruct: z.record(z.string(), PostWithMetaDataSchema).nullable(),
})
export type User = z.infer<typeof UserSchema>

`, StructToZodSchema(User{}))
}

func TestConvertArray(t *testing.T) {
	type Array struct {
		Arr [10]string
	}
	assert.Equal(t,
		`export const ArraySchema = z.object({
  Arr: z.string().array().length(10),
})
export type Array = z.infer<typeof ArraySchema>

`, StructToZodSchema(Array{}))

	type MultiArray struct {
		Arr [10][20][30]string
	}
	assert.Equal(t,
		`export const MultiArraySchema = z.object({
  Arr: z.string().array().length(30).array().length(20).array().length(10),
})
export type MultiArray = z.infer<typeof MultiArraySchema>

`, StructToZodSchema(MultiArray{}))
}

func TestConvertSlice(t *testing.T) {
	type Foo struct {
		Bar string
		Baz string
		Quz string
	}

	type Zip struct {
		Zap *Foo
	}

	type Whim struct {
		Wham *Foo
	}
	c := NewConverterWithOpts()
	types := []interface{}{
		Zip{},
		Whim{},
	}
	assert.Equal(t,
		`export const FooSchema = z.object({
  Bar: z.string(),
  Baz: z.string(),
  Quz: z.string(),
})
export type Foo = z.infer<typeof FooSchema>

export const ZipSchema = z.object({
  Zap: FooSchema.nullable(),
})
export type Zip = z.infer<typeof ZipSchema>

export const WhimSchema = z.object({
  Wham: FooSchema.nullable(),
})
export type Whim = z.infer<typeof WhimSchema>

`, c.ConvertSlice(types))
}

func TestConvertSliceWithValidations(t *testing.T) {
	type Required struct {
		Slice []string `validate:"required"`
	}
	assert.Equal(t,
		`export const RequiredSchema = z.object({
  Slice: z.string().array(),
})
export type Required = z.infer<typeof RequiredSchema>

`, StructToZodSchema(Required{}))

	type Min struct {
		Slice []string `validate:"min=1"`
	}
	assert.Equal(t, `export const MinSchema = z.object({
  Slice: z.string().array().min(1),
})
export type Min = z.infer<typeof MinSchema>

`, StructToZodSchema(Min{}))

	type Max struct {
		Slice []string `validate:"max=1"`
	}
	assert.Equal(t, `export const MaxSchema = z.object({
  Slice: z.string().array().max(1),
})
export type Max = z.infer<typeof MaxSchema>

`, StructToZodSchema(Max{}))

	type Len struct {
		Slice []string `validate:"len=1"`
	}
	assert.Equal(t, `export const LenSchema = z.object({
  Slice: z.string().array().length(1),
})
export type Len = z.infer<typeof LenSchema>

`, StructToZodSchema(Len{}))

	type Eq struct {
		Slice []string `validate:"eq=1"`
	}
	assert.Equal(t, `export const EqSchema = z.object({
  Slice: z.string().array().length(1),
})
export type Eq = z.infer<typeof EqSchema>

`, StructToZodSchema(Eq{}))

	type Gt struct {
		Slice []string `validate:"gt=1"`
	}
	assert.Equal(t, `export const GtSchema = z.object({
  Slice: z.string().array().min(2),
})
export type Gt = z.infer<typeof GtSchema>

`, StructToZodSchema(Gt{}))

	type Gte struct {
		Slice []string `validate:"gte=1"`
	}
	assert.Equal(t, `export const GteSchema = z.object({
  Slice: z.string().array().min(1),
})
export type Gte = z.infer<typeof GteSchema>

`, StructToZodSchema(Gte{}))

	type Lt struct {
		Slice []string `validate:"lt=1"`
	}
	assert.Equal(t, `export const LtSchema = z.object({
  Slice: z.string().array().max(0),
})
export type Lt = z.infer<typeof LtSchema>

`, StructToZodSchema(Lt{}))

	type Lte struct {
		Slice []string `validate:"lte=1"`
	}
	assert.Equal(t, `export const LteSchema = z.object({
  Slice: z.string().array().max(1),
})
export type Lte = z.infer<typeof LteSchema>

`, StructToZodSchema(Lte{}))

	type Ne struct {
		Slice []string `validate:"ne=0"`
	}
	assert.Equal(t, `export const NeSchema = z.object({
  Slice: z.string().array().refine((val) => val.length !== 0),
})
export type Ne = z.infer<typeof NeSchema>

`, StructToZodSchema(Ne{}))

	assert.Panics(t, func() {
		type Bad struct {
			Slice []string `validate:"oneof=a b c"`
		}
		StructToZodSchema(Bad{})
	})

	type Dive1 struct {
		Slice [][]string `validate:"dive,required"`
	}
	assert.Equal(t, `export const Dive1Schema = z.object({
  Slice: z.string().array().array().nullable(),
})
export type Dive1 = z.infer<typeof Dive1Schema>

`, StructToZodSchema(Dive1{}))

	type Dive2 struct {
		Slice [][]string `validate:"required,dive,min=1"`
	}
	assert.Equal(t, `export const Dive2Schema = z.object({
  Slice: z.string().array().min(1).array(),
})
export type Dive2 = z.infer<typeof Dive2Schema>

`, StructToZodSchema(Dive2{}))
}

func TestStructTime(t *testing.T) {
	type User struct {
		Name string
		When time.Time
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Name: z.string(),
  When: z.coerce.date(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestTimeWithRequired(t *testing.T) {
	type User struct {
		When time.Time `validate:"required"`
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  When: z.coerce.date().refine((val) => val.getTime() !== new Date('0001-01-01T00:00:00Z').getTime() && val.getTime() !== new Date(0).getTime(), 'Invalid date'),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestDuration(t *testing.T) {
	type User struct {
		HowLong time.Duration
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  HowLong: z.number(),
})
export type User = z.infer<typeof UserSchema>

`,
		StructToZodSchema(User{}))
}

func TestCustom(t *testing.T) {
	c := NewConverterWithOpts(WithCustomTypes(map[string]CustomFn{
		"github.com/hypersequent/zen.Decimal": func(c *Converter, t reflect.Type, validate string, i int) string {
			return "z.string()"
		},
	}))

	type Decimal struct {
		Value    int
		Exponent int
	}

	type User struct {
		Name  string
		Money Decimal
	}
	assert.Equal(t,
		`export const UserSchema = z.object({
  Name: z.string(),
  Money: z.string(),
})
export type User = z.infer<typeof UserSchema>

`,
		c.Convert(User{}))
}

func TestRecursive1(t *testing.T) {
	type NestedItem struct {
		ID        int           `json:"id"`
		Title     string        `json:"title"`
		Pos       int           `json:"pos"`
		ParentID  int           `json:"parent_id"`
		ProjectID int           `json:"project_id"`
		Children  []*NestedItem `json:"children"`
	}

	assert.Equal(t, `export type NestedItem = {
  id: number,
  title: string,
  pos: number,
  parent_id: number,
  project_id: number,
  children: NestedItem[] | null,
}
export const NestedItemSchema: z.ZodType<NestedItem> = z.object({
  id: z.number(),
  title: z.string(),
  pos: z.number(),
  parent_id: z.number(),
  project_id: z.number(),
  children: z.lazy(() => NestedItemSchema).array().nullable(),
})

`, StructToZodSchema(NestedItem{}))
}

func TestRecursive2(t *testing.T) {
	type Node struct {
		Value int   `json:"value"`
		Next  *Node `json:"next"`
	}

	type Parent struct {
		Child *Node `json:"child"`
	}

	assert.Equal(t, `export type Node = {
  value: number,
  next: Node | null,
}
export const NodeSchema: z.ZodType<Node> = z.object({
  value: z.number(),
  next: z.lazy(() => NodeSchema).nullable(),
})

export const ParentSchema = z.object({
  child: NodeSchema.nullable(),
})
export type Parent = z.infer<typeof ParentSchema>

`, StructToZodSchema(Parent{}))
}

type TestCyclicA struct {
	B *TestCyclicB
}

type TestCyclicB struct {
	A *TestCyclicA
}

func TestCyclic(t *testing.T) {
	assert.Panics(t, func() {
		StructToZodSchema(TestCyclicA{})
	})
}

type GenericPair[T any, U any] struct {
	First  T
	Second U
}

type StringIntPair GenericPair[string, int]

type PairMap[K comparable, T any, U any] struct {
	Items map[K]GenericPair[T, U] `json:"items"`
}

func TestGenerics(t *testing.T) {
	c := NewConverterWithOpts()
	c.AddType(StringIntPair{})
	c.AddType(GenericPair[int, bool]{})
	c.AddType(PairMap[string, int, bool]{})
	assert.Equal(t, `export const StringIntPairSchema = z.object({
  First: z.string(),
  Second: z.number(),
})
export type StringIntPair = z.infer<typeof StringIntPairSchema>

export const GenericPairIntBoolSchema = z.object({
  First: z.number(),
  Second: z.boolean(),
})
export type GenericPairIntBool = z.infer<typeof GenericPairIntBoolSchema>

export const PairMapStringIntBoolSchema = z.object({
  items: z.record(z.string(), GenericPairIntBoolSchema).nullable(),
})
export type PairMapStringIntBool = z.infer<typeof PairMapStringIntBoolSchema>

`, c.Export())
}

func TestSliceFields(t *testing.T) {
	type TestSliceFieldsStruct struct {
		NoValidate       []int
		Required         []int `validate:"required"`
		Min              []int `validate:"min=1"`
		OmitEmpty        []int `validate:"omitempty"`
		JSONOmitEmpty    []int `json:",omitempty"`
		MinOmitEmpty     []int `validate:"min=1,omitempty"`
		JSONMinOmitEmpty []int `json:",omitempty" validate:"min=1,omitempty"`
	}

	assert.Equal(t, `export const TestSliceFieldsStructSchema = z.object({
  NoValidate: z.number().array().nullable(),
  Required: z.number().array(),
  Min: z.number().array().min(1),
  OmitEmpty: z.number().array().nullable(),
  JSONOmitEmpty: z.number().array().optional(),
  MinOmitEmpty: z.number().array().min(1).nullable(),
  JSONMinOmitEmpty: z.number().array().min(1).optional(),
})
export type TestSliceFieldsStruct = z.infer<typeof TestSliceFieldsStructSchema>

`, StructToZodSchema(TestSliceFieldsStruct{}))
}

func TestCustomTag(t *testing.T) {
	type SortParams struct {
		Order *string `json:"order,omitempty" validate:"omitempty,oneof=asc desc"`
		Field *string `json:"field,omitempty"`
	}

	type Request struct {
		SortParams       `validate:"sortFields=title address age dob"`
		PaginationParams struct {
			Start *int `json:"start,omitempty" validate:"omitempty,gt=0"`
			End   *int `json:"end,omitempty" validate:"omitempty,gt=0"`
		} `validate:"pageParams"`
		Search *string `json:"search,omitempty" validate:"identifier"`
	}

	customTagHandlers := map[string]CustomFn{
		"identifier": func(c *Converter, t reflect.Type, validate string, i int) string {
			return ".refine((val) => !val || /^[a-z0-9_]*$/.test(val), 'Invalid search identifier')"
		},
		"pageParams": func(c *Converter, t reflect.Type, validate string, i int) string {
			return ".refine((val) => !val.start || !val.end || val.start < val.end, 'Start should be less than end')"
		},
		"sortFields": func(c *Converter, t reflect.Type, validate string, i int) string {
			sortFields := strings.Split(validate, " ")
			for i := range sortFields {
				sortFields[i] = fmt.Sprintf("'%s'", sortFields[i])
			}
			return fmt.Sprintf(".extend({field: z.enum([%s])})", strings.Join(sortFields, ", "))
		},
	}

	assert.Equal(t, `export const SortParamsSchema = z.object({
  order: z.enum(["asc", "desc"] as const).optional(),
  field: z.string().optional(),
})
export type SortParams = z.infer<typeof SortParamsSchema>

export const RequestSchema = z.object({
  PaginationParams: z.object({
    start: z.number().gt(0).optional(),
    end: z.number().gt(0).optional(),
  }).refine((val) => !val.start || !val.end || val.start < val.end, 'Start should be less than end'),
  search: z.string().refine((val) => !val || /^[a-z0-9_]*$/.test(val), 'Invalid search identifier').optional(),
}).merge(SortParamsSchema.extend({field: z.enum(['title', 'address', 'age', 'dob'])}))
export type Request = z.infer<typeof RequestSchema>

`, NewConverterWithOpts(WithCustomTags(customTagHandlers)).Convert(Request{}))
}
