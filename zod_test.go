package zen

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xorcare/golden"
)

// goldenMeta holds metadata written as comments at the top of golden files.
type goldenMeta struct {
	zodVersion  string // "v3", "v4", or "" (works with all versions)
	noTypecheck bool   // opt out of docker type-check tests
}

type goldenOpt func(*goldenMeta)

func withGoldenZodVersion(v string) goldenOpt {
	return func(m *goldenMeta) { m.zodVersion = v }
}

// goldenAssert wraps golden.Assert, prepending metadata comments to the file.
// The metadata is used by the docker type-check script to determine which zod
// version to install and whether to include the file in type checking.
//
// All golden files are type-checked by default.
func goldenAssert(t *testing.T, data []byte, opts ...goldenOpt) {
	t.Helper()
	var meta goldenMeta
	for _, o := range opts {
		o(&meta)
	}
	var lines []string
	if meta.zodVersion != "" {
		lines = append(lines, "// @zod-version: "+meta.zodVersion)
	}
	if !meta.noTypecheck {
		lines = append(lines, "// @typecheck")
	}
	if len(lines) > 0 {
		header := strings.Join(lines, "\n") + "\n"
		data = append([]byte(header), data...)
	}
	golden.Assert(t, data)
}

// assertSchema is a golden file test helper for Zod schema output.
//
// When no versions are specified, it asserts that v3 and v4 produce identical
// output and golden-tests that output once.
//
// When one version is specified ("v3" or "v4"), it golden-tests that version's
// output directly without a subtest.
//
// When multiple versions are specified, it creates a subtest per version and
// golden-tests each independently.
func assertSchema(t *testing.T, schema any, versions ...string) {
	t.Helper()

	optsFor := func(ver string) []Opt {
		if ver == "v3" {
			return []Opt{WithZodV3()}
		}
		return nil
	}

	switch len(versions) {
	case 0:
		v3out := StructToZodSchema(schema, WithZodV3())
		v4out := StructToZodSchema(schema)
		assert.Equal(t, v3out, v4out)
		goldenAssert(t, []byte(v4out))
	case 1:
		goldenAssert(t, []byte(StructToZodSchema(schema, optsFor(versions[0])...)), withGoldenZodVersion(versions[0]))
	default:
		for _, ver := range versions {
			t.Run(ver, func(t *testing.T) {
				goldenAssert(t, []byte(StructToZodSchema(schema, optsFor(ver)...)), withGoldenZodVersion(ver))
			})
		}
	}
}

// buildValidatorConverter creates a converter with dynamically-built single-field structs.
// Each entry maps a name to a validate tag. The field type is determined by fieldType.
func buildValidatorConverter(fieldType reflect.Type, validators []struct{ name, tag string }, opts ...Opt) *Converter {
	c := NewConverterWithOpts(opts...)
	for _, v := range validators {
		field := reflect.StructField{
			Name: "Value",
			Type: fieldType,
			Tag:  reflect.StructTag(fmt.Sprintf(`validate:"%s" json:"value"`, v.tag)),
		}
		st := reflect.StructOf([]reflect.StructField{field})
		c.AddTypeWithName(reflect.New(st).Elem().Interface(), v.name)
	}
	return c
}

// assertValidators golden-tests a list of validators.
// With no versions: asserts v3==v4, writes one golden file.
// With versions specified: writes separate golden files per version.
func assertValidators(t *testing.T, fieldType reflect.Type, validators []struct{ name, tag string }, versions ...string) {
	t.Helper()
	switch len(versions) {
	case 0:
		v3 := buildValidatorConverter(fieldType, validators, WithZodV3())
		v4 := buildValidatorConverter(fieldType, validators)
		assert.Equal(t, v3.Export(), v4.Export())
		goldenAssert(t, []byte(v4.Export()))
	default:
		for _, ver := range versions {
			t.Run(ver, func(t *testing.T) {
				var opts []Opt
				if ver == "v3" {
					opts = append(opts, WithZodV3())
				}
				c := buildValidatorConverter(fieldType, validators, opts...)
				goldenAssert(t, []byte(c.Export()), withGoldenZodVersion(ver))
			})
		}
	}
}

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
	assertSchema(t, User{})
}

func TestStructSimpleWithOmittedField(t *testing.T) {
	type User struct {
		Name        string
		Age         int
		Height      float64
		NotExported string `json:"-"`
	}
	assertSchema(t, User{})
}

func TestStructSimplePrefix(t *testing.T) {
	type User struct {
		Name   string
		Age    int
		Height float64
	}
	v3out := StructToZodSchema(User{}, WithPrefix("Bot"), WithZodV3())
	v4out := StructToZodSchema(User{}, WithPrefix("Bot"))
	assert.Equal(t, v3out, v4out)
	goldenAssert(t, []byte(v4out))
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
	assertSchema(t, User{}, "v3", "v4")
}

func TestStringArray(t *testing.T) {
	type User struct {
		Tags []string
	}
	assertSchema(t, User{})
}

func TestStringNestedArray(t *testing.T) {
	type TagPair [2]string
	type User struct {
		TagPairs []TagPair
	}
	assertSchema(t, User{})
}

func TestStructSlice(t *testing.T) {
	type User struct {
		Favourites []struct {
			Name string
		}
	}
	assertSchema(t, User{})
}

func TestStructSliceOptional(t *testing.T) {
	type User struct {
		Favourites []struct {
			Name string
		} `json:",omitempty"`
	}
	assertSchema(t, User{})
}

func TestStructSliceOptionalNullable(t *testing.T) {
	type User struct {
		Favourites *[]struct {
			Name string
		} `json:",omitempty"`
	}
	assertSchema(t, User{})
}

func TestStringOptional(t *testing.T) {
	type User struct {
		Name     string
		Nickname string `json:",omitempty"`
	}
	assertSchema(t, User{})
}

func TestStringNullable(t *testing.T) {
	type User struct {
		Name     string
		Nickname *string
	}
	assertSchema(t, User{})
}

func TestStringOptionalNotNullable(t *testing.T) {
	type User struct {
		Name     string
		Nickname *string `json:",omitempty"` // nil values are omitted
	}
	assertSchema(t, User{})
}

func TestStringOptionalNullable(t *testing.T) {
	type User struct {
		Name     string
		Nickname **string `json:",omitempty"` // nil values are omitted
	}
	assertSchema(t, User{})
}

func TestStringArrayNullable(t *testing.T) {
	type User struct {
		Name string
		Tags []*string
	}
	assertSchema(t, User{})
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

	assertSchema(t, User{})
}

func TestStringValidations(t *testing.T) {
	assertValidators(t, reflect.TypeOf(""), []struct{ name, tag string }{
		{"eq", "eq=hello"},
		{"ne", "ne=hello"},
		{"oneof", "oneof=hello world"},
		{"oneof_separated", "oneof='a b c' 'd e f'"},
		{"len", "len=5"},
		{"min", "min=5"},
		{"max", "max=5"},
		{"minmax", "min=3,max=7"},
		{"gt", "gt=5"},
		{"gte", "gte=5"},
		{"lt", "lt=5"},
		{"lte", "lte=5"},
		{"contains", "contains=hello"},
		{"endswith", "endswith=hello"},
		{"startswith", "startswith=hello"},
		{"required", "required"},
		{"url_encoded", "url_encoded"},
		{"alpha", "alpha"},
		{"alphanum", "alphanum"},
		{"alphanumunicode", "alphanumunicode"},
		{"alphaunicode", "alphaunicode"},
		{"ascii", "ascii"},
		{"boolean_validator", "boolean"},
		{"lowercase", "lowercase"},
		{"number_validator", "number"},
		{"numeric", "numeric"},
		{"uppercase", "uppercase"},
		{"mongodb", "mongodb"},
		{"json_validator", "json"},
		{"latitude", "latitude"},
		{"longitude", "longitude"},
		{"md4", "md4"},
	})

	t.Run("bad tag panics", func(t *testing.T) {
		type Bad struct {
			Name string `validate:"bad=hello"`
		}
		assert.Panics(t, func() { StructToZodSchema(Bad{}) })
	})

	t.Run("unknown tag panics", func(t *testing.T) {
		type Bad2 struct {
			Name string `validate:"bad2"`
		}
		assert.Panics(t, func() { StructToZodSchema(Bad2{}) })
	})

	t.Run("gt with non-integer panics", func(t *testing.T) {
		type Bad struct {
			Name string `validate:"gt=abc"`
		}
		assert.Panics(t, func() { StructToZodSchema(Bad{}) })
	})

	t.Run("lt with non-integer panics", func(t *testing.T) {
		type Bad struct {
			Name string `validate:"lt=abc"`
		}
		assert.Panics(t, func() { StructToZodSchema(Bad{}) })
	})

	t.Run("escapeJSString escapes quotes and backslashes", func(t *testing.T) {
		assert.Equal(t, `foo\"bar`, escapeJSString(`foo"bar`))
		assert.Equal(t, `foo\\bar`, escapeJSString(`foo\bar`))
		assert.Equal(t, `a\"b\\c`, escapeJSString(`a"b\c`))
		assert.Equal(t, `no change`, escapeJSString(`no change`))
	})

	t.Run("special chars in tag values are escaped in output", func(t *testing.T) {
		// Go struct tag syntax can't contain raw quotes, but reflect.StructOf can.
		// This tests that the generated JS output correctly escapes them.
		c := NewConverterWithOpts()

		contains := reflect.StructOf([]reflect.StructField{{
			Name: "Value", Type: reflect.TypeOf(""),
			Tag: reflect.StructTag(`validate:"contains=foo\"bar" json:"value"`),
		}})
		c.AddTypeWithName(reflect.New(contains).Elem().Interface(), "ContainsQuote")

		eq := reflect.StructOf([]reflect.StructField{{
			Name: "Value", Type: reflect.TypeOf(""),
			Tag: reflect.StructTag(`validate:"eq=a\\b" json:"value"`),
		}})
		c.AddTypeWithName(reflect.New(eq).Elem().Interface(), "EqBackslash")

		goldenAssert(t, []byte(c.Export()))
	})

	t.Run("enum ignores other validators", func(t *testing.T) {
		c := NewConverterWithOpts()
		c.AddTypeWithName(struct {
			V string `validate:"required,oneof=a b" json:"v"`
		}{}, "RequiredOneof")
		c.AddTypeWithName(struct {
			V string `validate:"oneof=a b,contains=x" json:"v"`
		}{}, "OneofContains")
		c.AddTypeWithName(struct {
			V string `validate:"oneof=a b,startswith=a" json:"v"`
		}{}, "OneofStartswith")
		c.AddTypeWithName(struct {
			V string `validate:"oneof=a b,endswith=z" json:"v"`
		}{}, "OneofEndswith")
		c.AddTypeWithName(struct {
			V string `validate:"oneof='127.0.0.1' '::1',ip" json:"v"`
		}{}, "OneofIp")
		goldenAssert(t, []byte(c.Export()))
	})
}

func TestOneofRequired(t *testing.T) {
	type Payload struct {
		Status string `json:"status" validate:"required,oneof=active inactive"`
		// Would generate the same schema as the above. This doesn't mirror go validator exactly as it allows empty values.
		// For now let's assume that empty strings are not valid enum values, but we can revisit if there's demand for that.
		StatusImplicitRequired string  `json:"statusImplicitRequired" validate:"oneof=active inactive"`
		Channel                *string `json:"channel,omitempty" validate:"omitempty,oneof=email sms"`
	}

	assertSchema(t, Payload{})
}

func TestZodV4Defaults(t *testing.T) {
	t.Run("embedded structs use shape spreads", func(t *testing.T) {
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

		assertSchema(t, User{}, "v4")
	})

	t.Run("string formats use zod v4 builders", func(t *testing.T) {
		type Payload struct {
			Email    string `validate:"email"`
			Link     string `validate:"http_url"`
			Base64   string `validate:"base64"`
			ID       string `validate:"uuid4"`
			Checksum string `validate:"md5"`
		}

		assertSchema(t, Payload{}, "v4")
	})

	t.Run("string tag order is preserved around v4 format helpers", func(t *testing.T) {
		type Payload struct {
			TrimmedThenEmail string `validate:"trim,email"`
			EmailThenTrimmed string `validate:"email,trim"`
		}

		customTagHandlers := map[string]CustomFn{
			"trim": func(c *Converter, t reflect.Type, validate string, i int) string {
				return ".trim()"
			},
		}

		goldenAssert(t, []byte(NewConverterWithOpts(WithCustomTags(customTagHandlers)).Convert(Payload{})), withGoldenZodVersion("v4"))
	})

	t.Run("ip unions inherit generic string constraints", func(t *testing.T) {
		type Payload struct {
			Address string `validate:"ip,required,max=45"`
		}

		assertSchema(t, Payload{}, "v4")
	})

	t.Run("format combined with union panics", func(t *testing.T) {
		type Payload struct {
			Address string `validate:"email,ip"`
		}

		assert.Panics(t, func() { StructToZodSchema(Payload{}) })
	})

	t.Run("multiple formats panics", func(t *testing.T) {
		type Payload struct {
			Value string `validate:"email,url"`
		}

		assert.Panics(t, func() { StructToZodSchema(Payload{}) })
	})

	t.Run("optional format with nullable pointer", func(t *testing.T) {
		type Payload struct {
			Email *string `validate:"omitempty,email" json:"email"`
		}

		assertSchema(t, Payload{}, "v3", "v4")
	})

	t.Run("named field shadows embedded field", func(t *testing.T) {
		type Base struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		type Child struct {
			Base
			ID int `json:"id"` // shadows Base.ID, keeps Base.Name
		}

		assertSchema(t, Child{}, "v3", "v4")
	})

	t.Run("recursive embedded shapes preserve encounter order for duplicate keys", func(t *testing.T) {
		type Base struct {
			ID string `json:"id"`
		}

		type Node struct {
			Base
			ID   int   `json:"id"`
			Next *Node `json:"next"`
		}

		goldenAssert(t, []byte(StructToZodSchema(Node{})), withGoldenZodVersion("v4"))
	})

	t.Run("recursive embedded shapes keep named fields after spreads to override embedded fields", func(t *testing.T) {
		type TreeNode struct {
			Value     string
			CreatedAt time.Time
			Children  *[]TreeNode
			UpdatedAt string
		}

		type Tree struct {
			TreeNode
			UpdatedAt time.Time
		}

		assertSchema(t, Tree{}, "v4")
	})
}

func TestNumberValidations(t *testing.T) {
	assertValidators(t, reflect.TypeOf(0), []struct{ name, tag string }{
		{"gte_lte", "gte=18,lte=60"},
		{"gt_lt", "gt=18,lt=60"},
		{"eq", "eq=18"},
		{"ne", "ne=18"},
		{"oneof", "oneof=18 19 20"},
		{"min_max", "min=18,max=60"},
		{"len", "len=18"},
	})

	t.Run("bad tag panics", func(t *testing.T) {
		type Bad struct {
			Age int `validate:"bad=18"`
		}
		assert.Panics(t, func() { StructToZodSchema(Bad{}) })
	})

	t.Run("non-numeric arg panics", func(t *testing.T) {
		tags := []string{"gt", "gte", "lt", "lte", "min", "max", "eq", "ne", "len"}
		for _, tag := range tags {
			t.Run(tag, func(t *testing.T) {
				assert.Panics(t, func() {
					st := reflect.StructOf([]reflect.StructField{{
						Name: "V",
						Type: reflect.TypeOf(0),
						Tag:  reflect.StructTag(fmt.Sprintf(`validate:"%s=abc" json:"v"`, tag)),
					}})
					StructToZodSchema(reflect.New(st).Elem().Interface())
				})
			})
		}
		t.Run("oneof", func(t *testing.T) {
			assert.Panics(t, func() {
				st := reflect.StructOf([]reflect.StructField{{
					Name: "V",
					Type: reflect.TypeOf(0),
					Tag:  reflect.StructTag(`validate:"oneof=1 abc 3" json:"v"`),
				}})
				StructToZodSchema(reflect.New(st).Elem().Interface())
			})
		})
	})

	t.Run("float args are accepted", func(t *testing.T) {
		type S struct {
			V float64 `validate:"gt=1.5,lt=9.9"`
		}
		assert.NotPanics(t, func() { StructToZodSchema(S{}) })
	})
}

func TestInterfaceAny(t *testing.T) {
	type User struct {
		Name     string
		Metadata interface{}
	}
	assertSchema(t, User{})
}

func TestInterfacePointerAny(t *testing.T) {
	type User struct {
		Name     string
		Metadata *interface{}
	}
	assertSchema(t, User{})
}

func TestInterfaceEmptyAny(t *testing.T) {
	type User struct {
		Name     string
		Metadata interface{} `json:",omitempty"`
	}
	assertSchema(t, User{})
}

func TestInterfacePointerEmptyAny(t *testing.T) {
	type User struct {
		Name     string
		Metadata *interface{} `json:",omitempty"`
	}
	assertSchema(t, User{})
}

func TestMapStringToString(t *testing.T) {
	type User struct {
		Name     string
		Metadata map[string]string
	}
	assertSchema(t, User{})
}

func TestMapStringToInterface(t *testing.T) {
	type User struct {
		Name     string
		Metadata map[string]interface{}
	}
	assertSchema(t, User{})
}

func TestMapWithStruct(t *testing.T) {
	type PostWithMetaData struct {
		Title string
	}
	type User struct {
		MapWithStruct map[string]PostWithMetaData
	}
	assertSchema(t, User{})
}

func TestMapWithValidations(t *testing.T) {
	assertValidators(t, reflect.TypeOf(map[string]string{}), []struct{ name, tag string }{
		{"required", "required"},
		{"min", "min=1"},
		{"max", "max=1"},
		{"len", "len=1"},
		{"minmax", "min=1,max=2"},
		{"eq", "eq=1"},
		{"ne", "ne=1"},
		{"gt", "gt=1"},
		{"gte", "gte=1"},
		{"lt", "lt=1"},
		{"lte", "lte=1"},
		{"dive1", "dive,min=2"},
	})

	t.Run("dive_nested", func(t *testing.T) {
		assertValidators(t, reflect.TypeOf([]map[string]string{}), []struct{ name, tag string }{
			{"dive2", "required,dive,min=2,dive,min=3"},
			{"dive3", "required,dive,min=2,dive,keys,min=3,endkeys,max=4"},
		})
	})

	t.Run("bad tag panics", func(t *testing.T) {
		type Bad struct {
			Map map[string]string `validate:"bad=1"`
		}
		assert.Panics(t, func() { StructToZodSchema(Bad{}) })
	})

	t.Run("non-integer args panic", func(t *testing.T) {
		tags := []string{"min", "max", "len", "eq", "ne", "gt", "gte", "lt", "lte"}
		for _, tag := range tags {
			t.Run(tag, func(t *testing.T) {
				assert.Panics(t, func() {
					st := reflect.StructOf([]reflect.StructField{{
						Name: "M",
						Type: reflect.TypeOf(map[string]string{}),
						Tag:  reflect.StructTag(fmt.Sprintf(`validate:"%s=abc" json:"m"`, tag)),
					}})
					StructToZodSchema(reflect.New(st).Elem().Interface())
				})
			})
		}
	})
}

func TestMapWithNonStringKey(t *testing.T) {
	type Map1 struct {
		Name     string
		Metadata map[int]string
	}

	type Map2 struct {
		Name     string
		Metadata map[time.Time]string
	}

	type Map3 struct {
		Name     string
		Metadata map[float64]string
	}

	t.Run("int_key", func(t *testing.T) {
		assertSchema(t, Map1{})
	})

	t.Run("time_key", func(t *testing.T) {
		assertSchema(t, Map2{})
	})

	t.Run("float_key", func(t *testing.T) {
		assertSchema(t, Map3{})
	})
}

func TestMapWithEnumKey(t *testing.T) {
	type Payload struct {
		Metadata map[string]string `validate:"dive,keys,oneof=draft published,endkeys"`
	}

	assertSchema(t, Payload{}, "v3", "v4")
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

func TestStructTime(t *testing.T) {
	type User struct {
		Name string
		When time.Time
	}
	assertSchema(t, User{})
}

func TestTimeWithRequired(t *testing.T) {
	type User struct {
		When time.Time `validate:"required"`
	}
	assertSchema(t, User{})
}

func TestDuration(t *testing.T) {
	type User struct {
		HowLong time.Duration
	}
	assertSchema(t, User{})
}

// Wrapper mimics a generic optional type like 4d63.com/optional.Optional[T].
// The custom handler resolves the inner type via ConvertType(t.Elem(), ...).
type Wrapper[T any] struct{ Value T }

func TestCustomTypes(t *testing.T) {
	t.Run("custom type mapped to string", func(t *testing.T) {
		type Decimal struct {
			Value    int
			Exponent int
		}
		type User struct {
			Name  string
			Money Decimal
		}

		customTypes := map[string]CustomFn{
			"github.com/hypersequent/zen.Decimal": func(c *Converter, t reflect.Type, validate string, i int) string {
				return "z.string()"
			},
		}

		v3c := NewConverterWithOpts(WithCustomTypes(customTypes), WithZodV3())
		v4c := NewConverterWithOpts(WithCustomTypes(customTypes))
		v3out := v3c.Convert(User{})
		v4out := v4c.Convert(User{})
		assert.Equal(t, v3out, v4out)
		goldenAssert(t, []byte(v4out))
	})

	t.Run("custom type resolves inner generic type", func(t *testing.T) {
		type Profile struct {
			Bio string
		}
		type User struct {
			MaybeName    Wrapper[string]
			MaybeAge     Wrapper[int]
			MaybeHeight  Wrapper[float64]
			MaybeProfile Wrapper[Profile]
		}

		customTypes := map[string]CustomFn{
			"github.com/hypersequent/zen.Wrapper": func(c *Converter, t reflect.Type, validate string, i int) string {
				return fmt.Sprintf("%s.optional().nullish()", c.ConvertType(t.Field(0).Type, validate, i))
			},
		}

		v3c := NewConverterWithOpts(WithCustomTypes(customTypes), WithZodV3())
		v4c := NewConverterWithOpts(WithCustomTypes(customTypes))
		v3out := v3c.Convert(User{})
		v4out := v4c.Convert(User{})
		assert.Equal(t, v3out, v4out)
		goldenAssert(t, []byte(v4out))
	})

	t.Run("custom type with nullable control", func(t *testing.T) {
		type User struct {
			Name  string
			Email *Wrapper[string]
		}

		customTypes := map[string]CustomFn{
			"github.com/hypersequent/zen.Wrapper": func(c *Converter, t reflect.Type, validate string, i int) string {
				return fmt.Sprintf("%s.optional()", c.ConvertType(t.Field(0).Type, validate, i))
			},
		}

		v3c := NewConverterWithOpts(WithCustomTypes(customTypes), WithZodV3())
		v4c := NewConverterWithOpts(WithCustomTypes(customTypes))
		v3out := v3c.Convert(User{})
		v4out := v4c.Convert(User{})
		assert.Equal(t, v3out, v4out)
		goldenAssert(t, []byte(v4out))
	})
}

func TestWithIgnoreTags(t *testing.T) {
	type User struct {
		Name string `validate:"required,customtag=value"`
	}

	t.Run("panics on unknown tag", func(t *testing.T) {
		assert.Panics(t, func() { StructToZodSchema(User{}) })
	})

	t.Run("ignores specified tag", func(t *testing.T) {
		assert.NotPanics(t, func() {
			StructToZodSchema(User{}, WithIgnoreTags("customtag"))
		})
		goldenAssert(t, []byte(StructToZodSchema(User{}, WithIgnoreTags("customtag"))))
	})
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

	assertSchema(t, User{})
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
	assertSchema(t, User{})
}

func TestConvertArray(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		type Array struct {
			Arr [10]string
		}
		assertSchema(t, Array{})
	})

	t.Run("multi", func(t *testing.T) {
		type MultiArray struct {
			Arr [10][20][30]string
		}
		assertSchema(t, MultiArray{})
	})
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

	types := []interface{}{
		Zip{},
		Whim{},
	}

	v3c := NewConverterWithOpts(WithZodV3())
	v4c := NewConverterWithOpts()
	v3out := v3c.ConvertSlice(types)
	v4out := v4c.ConvertSlice(types)
	assert.Equal(t, v3out, v4out)
	goldenAssert(t, []byte(v4out))
}

func TestConvertSliceWithValidations(t *testing.T) {
	assertValidators(t, reflect.TypeOf([]string{}), []struct{ name, tag string }{
		{"required", "required"},
		{"min", "min=1"},
		{"max", "max=1"},
		{"len", "len=1"},
		{"eq", "eq=1"},
		{"gt", "gt=1"},
		{"gte", "gte=1"},
		{"lt", "lt=1"},
		{"lte", "lte=1"},
		{"ne", "ne=0"},
	})

	t.Run("dive_nested", func(t *testing.T) {
		assertValidators(t, reflect.TypeOf([][]string{}), []struct{ name, tag string }{
			{"dive1", "dive,required"},
			{"dive2", "required,dive,min=1"},
		})
	})

	t.Run("dive_oneof", func(t *testing.T) {
		assertValidators(t, reflect.TypeOf([]string{}), []struct{ name, tag string }{
			{"dive_oneof", "dive,oneof=a b c"},
		})
	})

	t.Run("oneof without dive panics", func(t *testing.T) {
		assert.Panics(t, func() {
			type Bad struct {
				Slice []string `validate:"oneof=a b c"`
			}
			StructToZodSchema(Bad{})
		})
	})

	t.Run("non-integer args panic", func(t *testing.T) {
		tags := []string{"min", "max", "len", "eq", "ne", "gt", "gte", "lt", "lte"}
		for _, tag := range tags {
			t.Run(tag, func(t *testing.T) {
				assert.Panics(t, func() {
					st := reflect.StructOf([]reflect.StructField{{
						Name: "V",
						Type: reflect.TypeOf([]string{}),
						Tag:  reflect.StructTag(fmt.Sprintf(`validate:"%s=abc" json:"v"`, tag)),
					}})
					StructToZodSchema(reflect.New(st).Elem().Interface())
				})
			})
		}
	})
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

	assertSchema(t, NestedItem{}, "v3", "v4")
}

func TestRecursive2(t *testing.T) {
	type Node struct {
		Value int   `json:"value"`
		Next  *Node `json:"next"`
	}

	type Parent struct {
		Child *Node `json:"child"`
	}

	assertSchema(t, Parent{}, "v3", "v4")
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

	v3c := NewConverterWithOpts(WithZodV3())
	v3c.AddType(StringIntPair{})
	v3c.AddType(GenericPair[int, bool]{})
	v3c.AddType(PairMap[string, int, bool]{})

	v3out := v3c.Export()
	v4out := c.Export()
	assert.Equal(t, v3out, v4out)
	goldenAssert(t, []byte(v4out))
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

	assertSchema(t, TestSliceFieldsStruct{})
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

	t.Run("v3", func(t *testing.T) {
		goldenAssert(t, []byte(NewConverterWithOpts(WithCustomTags(customTagHandlers), WithZodV3()).Convert(Request{})), withGoldenZodVersion("v3"))
	})
	t.Run("v4", func(t *testing.T) {
		goldenAssert(t, []byte(NewConverterWithOpts(WithCustomTags(customTagHandlers)).Convert(Request{})), withGoldenZodVersion("v4"))
	})
}

func TestCustomTagReceivesCorrectType(t *testing.T) {
	// A "nonzero" custom tag that emits different validation depending on the
	// field type: strings check for non-empty, numbers check for non-zero,
	// time.Time checks for non-zero date.
	handler := map[string]CustomFn{
		"nonzero": func(c *Converter, t reflect.Type, validate string, i int) string {
			switch t.Kind() {
			case reflect.String:
				return `.refine((val) => val !== "", "must not be empty")`
			case reflect.Int, reflect.Float64:
				return ".refine((val) => val !== 0, \"must not be zero\")"
			case reflect.Struct:
				if t.Name() == "Time" {
					return ".refine((val) => val.getTime() !== 0, \"must not be zero time\")"
				}
				return ".refine((val) => true)"
			default:
				return ".refine((val) => true)"
			}
		},
	}

	type Payload struct {
		Name string    `json:"name" validate:"nonzero"`
		Age  int       `json:"age" validate:"nonzero"`
		When time.Time `json:"when" validate:"nonzero"`
	}

	output := NewConverterWithOpts(WithCustomTags(handler)).Convert(Payload{})

	assert.Contains(t, output, `val !== ""`, "string field should get string-specific check")
	assert.Contains(t, output, `val !== 0, "must not be zero"`, "number field should get number-specific check")
	assert.Contains(t, output, `val.getTime() !== 0`, "time field should get time-specific check")
}

func TestRecursiveEmbeddedStruct(t *testing.T) {
	type ItemA struct {
		Name     string
		Children []ItemA
	}

	type ItemB struct {
		ItemA
	}

	type ItemC struct {
		ItemB
	}

	type ItemD struct {
		ItemA ItemA
	}

	type ItemE struct {
		ItemA
		ItemD
		Children []ItemE
	}

	type ItemF struct {
		ItemE
	}

	t.Run("v3", func(t *testing.T) {
		c := NewConverterWithOpts(WithZodV3())
		c.AddType(ItemA{})
		c.AddType(ItemB{})
		c.AddType(ItemC{})
		c.AddType(ItemD{})
		c.AddType(ItemE{})
		c.AddType(ItemF{})
		goldenAssert(t, []byte(c.Export()), withGoldenZodVersion("v3"))
	})
	t.Run("v4", func(t *testing.T) {
		c := NewConverterWithOpts()
		c.AddType(ItemA{})
		c.AddType(ItemB{})
		c.AddType(ItemC{})
		c.AddType(ItemD{})
		c.AddType(ItemE{})
		c.AddType(ItemF{})
		goldenAssert(t, []byte(c.Export()), withGoldenZodVersion("v4"))
	})
}

func TestRecursiveEmbeddedWithPointersAndDates(t *testing.T) {
	t.Run("recursive struct with pointer field and date", func(t *testing.T) {
		type TreeNode struct {
			Value     string
			CreatedAt time.Time
			Children  *[]TreeNode
		}

		type Tree struct {
			TreeNode
			UpdatedAt time.Time
		}

		assertSchema(t, Tree{}, "v3", "v4")
	})

	t.Run("embedded struct with pointer to self and date", func(t *testing.T) {
		type Comment struct {
			Text      string
			Timestamp time.Time
			Reply     *Comment
		}

		type Article struct {
			Comment
			Title string
		}

		assertSchema(t, Article{}, "v3", "v4")
	})
}

func TestFormatValidators(t *testing.T) {
	allFormats := []string{
		"email", "url", "http_url",
		"ipv4", "ip4_addr", "ipv6", "ip6_addr",
		"base64", "datetime", "hexadecimal", "jwt",
		"uuid", "uuid3", "uuid3_rfc4122",
		"uuid4", "uuid4_rfc4122",
		"uuid5", "uuid5_rfc4122",
		"uuid_rfc4122",
		"md5", "sha256", "sha384", "sha512",
	}

	unionFormats := []string{"ip", "ip_addr"}

	toValidators := func(tags []string, prefix string) []struct{ name, tag string } {
		out := make([]struct{ name, tag string }, len(tags))
		for i, tag := range tags {
			out[i] = struct{ name, tag string }{tag, prefix + tag}
		}
		return out
	}

	t.Run("format only", func(t *testing.T) {
		assertValidators(t, reflect.TypeOf(""), toValidators(allFormats, ""), "v3", "v4")
	})

	t.Run("format with required", func(t *testing.T) {
		assertValidators(t, reflect.TypeOf(""), toValidators(allFormats, "required,"), "v3", "v4")
	})

	t.Run("union only", func(t *testing.T) {
		assertValidators(t, reflect.TypeOf(""), toValidators(unionFormats, ""), "v3", "v4")
	})

	t.Run("union with required", func(t *testing.T) {
		assertValidators(t, reflect.TypeOf(""), toValidators(unionFormats, "required,"), "v3", "v4")
	})
}
