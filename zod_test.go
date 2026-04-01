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
		golden.Assert(t, []byte(v4out))
	case 1:
		golden.Assert(t, []byte(StructToZodSchema(schema, optsFor(versions[0])...)))
	default:
		for _, ver := range versions {
			t.Run(ver, func(t *testing.T) {
				golden.Assert(t, []byte(StructToZodSchema(schema, optsFor(ver)...)))
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
	golden.Assert(t, []byte(v4out))
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
	t.Run("eq", func(t *testing.T) {
		type Eq struct {
			Name string `validate:"eq=hello"`
		}
		assertSchema(t, Eq{})
	})

	t.Run("ne", func(t *testing.T) {
		type Ne struct {
			Name string `validate:"ne=hello"`
		}
		assertSchema(t, Ne{})
	})

	t.Run("oneof", func(t *testing.T) {
		type OneOf struct {
			Name string `validate:"oneof=hello world"`
		}
		assertSchema(t, OneOf{})
	})

	t.Run("oneof_separated", func(t *testing.T) {
		type OneOfSeparated struct {
			Name string `validate:"oneof='a b c' 'd e f'"`
		}
		assertSchema(t, OneOfSeparated{})
	})

	t.Run("len", func(t *testing.T) {
		type Len struct {
			Name string `validate:"len=5"`
		}
		assertSchema(t, Len{})
	})

	t.Run("min", func(t *testing.T) {
		type Min struct {
			Name string `validate:"min=5"`
		}
		assertSchema(t, Min{})
	})

	t.Run("max", func(t *testing.T) {
		type Max struct {
			Name string `validate:"max=5"`
		}
		assertSchema(t, Max{})
	})

	t.Run("minmax", func(t *testing.T) {
		type MinMax struct {
			Name string `validate:"min=3,max=7"`
		}
		assertSchema(t, MinMax{})
	})

	t.Run("gt", func(t *testing.T) {
		type Gt struct {
			Name string `validate:"gt=5"`
		}
		assertSchema(t, Gt{})
	})

	t.Run("gte", func(t *testing.T) {
		type Gte struct {
			Name string `validate:"gte=5"`
		}
		assertSchema(t, Gte{})
	})

	t.Run("lt", func(t *testing.T) {
		type Lt struct {
			Name string `validate:"lt=5"`
		}
		assertSchema(t, Lt{})
	})

	t.Run("lte", func(t *testing.T) {
		type Lte struct {
			Name string `validate:"lte=5"`
		}
		assertSchema(t, Lte{})
	})

	t.Run("contains", func(t *testing.T) {
		type Contains struct {
			Name string `validate:"contains=hello"`
		}
		assertSchema(t, Contains{})
	})

	t.Run("endswith", func(t *testing.T) {
		type EndsWith struct {
			Name string `validate:"endswith=hello"`
		}
		assertSchema(t, EndsWith{})
	})

	t.Run("startswith", func(t *testing.T) {
		type StartsWith struct {
			Name string `validate:"startswith=hello"`
		}
		assertSchema(t, StartsWith{})
	})

	t.Run("bad", func(t *testing.T) {
		type Bad struct {
			Name string `validate:"bad=hello"`
		}
		assert.Panics(t, func() {
			StructToZodSchema(Bad{})
		})
	})

	t.Run("required", func(t *testing.T) {
		type Required struct {
			Name string `validate:"required"`
		}
		assertSchema(t, Required{})
	})

	t.Run("email", func(t *testing.T) {
		type Email struct {
			Name string `validate:"email"`
		}
		assertSchema(t, Email{}, "v3", "v4")
	})

	t.Run("url", func(t *testing.T) {
		type URL struct {
			Name string `validate:"url"`
		}
		assertSchema(t, URL{}, "v3", "v4")
	})

	t.Run("ipv4", func(t *testing.T) {
		type IPv4 struct {
			Name string `validate:"ipv4"`
		}
		assertSchema(t, IPv4{}, "v3", "v4")
	})

	t.Run("ipv6", func(t *testing.T) {
		type IPv6 struct {
			Name string `validate:"ipv6"`
		}
		assertSchema(t, IPv6{}, "v3", "v4")
	})

	t.Run("ip4_addr", func(t *testing.T) {
		type IP4Addr struct {
			Name string `validate:"ip4_addr"`
		}
		assertSchema(t, IP4Addr{}, "v3", "v4")
	})

	t.Run("ip6_addr", func(t *testing.T) {
		type IP6Addr struct {
			Name string `validate:"ip6_addr"`
		}
		assertSchema(t, IP6Addr{}, "v3", "v4")
	})

	t.Run("ip", func(t *testing.T) {
		type IP struct {
			Name string `validate:"ip"`
		}
		assertSchema(t, IP{}, "v3", "v4")
	})

	t.Run("ip_addr", func(t *testing.T) {
		type IPAddr struct {
			Name string `validate:"ip_addr"`
		}
		assertSchema(t, IPAddr{}, "v3", "v4")
	})

	t.Run("http_url", func(t *testing.T) {
		type HttpURL struct {
			Name string `validate:"http_url"`
		}
		assertSchema(t, HttpURL{}, "v3", "v4")
	})

	t.Run("url_encoded", func(t *testing.T) {
		type URLEncoded struct {
			Name string `validate:"url_encoded"`
		}
		assertSchema(t, URLEncoded{})
	})

	t.Run("alpha", func(t *testing.T) {
		type Alpha struct {
			Name string `validate:"alpha"`
		}
		assertSchema(t, Alpha{})
	})

	t.Run("alphanum", func(t *testing.T) {
		type AlphaNum struct {
			Name string `validate:"alphanum"`
		}
		assertSchema(t, AlphaNum{})
	})

	t.Run("alphanumunicode", func(t *testing.T) {
		type AlphaNumUnicode struct {
			Name string `validate:"alphanumunicode"`
		}
		assertSchema(t, AlphaNumUnicode{})
	})

	t.Run("alphaunicode", func(t *testing.T) {
		type AlphaUnicode struct {
			Name string `validate:"alphaunicode"`
		}
		assertSchema(t, AlphaUnicode{})
	})

	t.Run("ascii", func(t *testing.T) {
		type ASCII struct {
			Name string `validate:"ascii"`
		}
		assertSchema(t, ASCII{})
	})

	t.Run("boolean", func(t *testing.T) {
		type Boolean struct {
			Name string `validate:"boolean"`
		}
		assertSchema(t, Boolean{})
	})

	t.Run("lowercase", func(t *testing.T) {
		type Lowercase struct {
			Name string `validate:"lowercase"`
		}
		assertSchema(t, Lowercase{})
	})

	t.Run("number", func(t *testing.T) {
		type Number struct {
			Name string `validate:"number"`
		}
		assertSchema(t, Number{})
	})

	t.Run("numeric", func(t *testing.T) {
		type Numeric struct {
			Name string `validate:"numeric"`
		}
		assertSchema(t, Numeric{})
	})

	t.Run("uppercase", func(t *testing.T) {
		type Uppercase struct {
			Name string `validate:"uppercase"`
		}
		assertSchema(t, Uppercase{})
	})

	t.Run("base64", func(t *testing.T) {
		type Base64 struct {
			Name string `validate:"base64"`
		}
		assertSchema(t, Base64{}, "v3", "v4")
	})

	t.Run("mongodb", func(t *testing.T) {
		type mongodb struct {
			Name string `validate:"mongodb"`
		}
		assertSchema(t, mongodb{})
	})

	t.Run("datetime", func(t *testing.T) {
		type datetime struct {
			Name string `validate:"datetime"`
		}
		assertSchema(t, datetime{}, "v3", "v4")
	})

	t.Run("hexadecimal", func(t *testing.T) {
		type Hexadecimal struct {
			Name string `validate:"hexadecimal"`
		}
		assertSchema(t, Hexadecimal{}, "v3", "v4")
	})

	t.Run("json", func(t *testing.T) {
		type json struct {
			Name string `validate:"json"`
		}
		assertSchema(t, json{})
	})

	t.Run("latitude", func(t *testing.T) {
		type Latitude struct {
			Name string `validate:"latitude"`
		}
		assertSchema(t, Latitude{})
	})

	t.Run("longitude", func(t *testing.T) {
		type Longitude struct {
			Name string `validate:"longitude"`
		}
		assertSchema(t, Longitude{})
	})

	t.Run("uuid", func(t *testing.T) {
		type UUID struct {
			Name string `validate:"uuid"`
		}
		assertSchema(t, UUID{}, "v3", "v4")
	})

	t.Run("uuid3", func(t *testing.T) {
		type UUID3 struct {
			Name string `validate:"uuid3"`
		}
		assertSchema(t, UUID3{}, "v3", "v4")
	})

	t.Run("uuid3_rfc4122", func(t *testing.T) {
		type UUID3RFC4122 struct {
			Name string `validate:"uuid3_rfc4122"`
		}
		assertSchema(t, UUID3RFC4122{}, "v3", "v4")
	})

	t.Run("uuid4", func(t *testing.T) {
		type UUID4 struct {
			Name string `validate:"uuid4"`
		}
		assertSchema(t, UUID4{}, "v3", "v4")
	})

	t.Run("uuid4_rfc4122", func(t *testing.T) {
		type UUID4RFC4122 struct {
			Name string `validate:"uuid4_rfc4122"`
		}
		assertSchema(t, UUID4RFC4122{}, "v3", "v4")
	})

	t.Run("uuid5", func(t *testing.T) {
		type UUID5 struct {
			Name string `validate:"uuid5"`
		}
		assertSchema(t, UUID5{}, "v3", "v4")
	})

	t.Run("uuid5_rfc4122", func(t *testing.T) {
		type UUID5RFC4122 struct {
			Name string `validate:"uuid5_rfc4122"`
		}
		assertSchema(t, UUID5RFC4122{}, "v3", "v4")
	})

	t.Run("uuid_rfc4122", func(t *testing.T) {
		type UUIDRFC4122 struct {
			Name string `validate:"uuid_rfc4122"`
		}
		assertSchema(t, UUIDRFC4122{}, "v3", "v4")
	})

	t.Run("md4", func(t *testing.T) {
		type MD4 struct {
			Name string `validate:"md4"`
		}
		assertSchema(t, MD4{})
	})

	t.Run("md5", func(t *testing.T) {
		type MD5 struct {
			Name string `validate:"md5"`
		}
		assertSchema(t, MD5{}, "v3", "v4")
	})

	t.Run("sha256", func(t *testing.T) {
		type SHA256 struct {
			Name string `validate:"sha256"`
		}
		assertSchema(t, SHA256{}, "v3", "v4")
	})

	t.Run("sha384", func(t *testing.T) {
		type SHA384 struct {
			Name string `validate:"sha384"`
		}
		assertSchema(t, SHA384{}, "v3", "v4")
	})

	t.Run("sha512", func(t *testing.T) {
		type SHA512 struct {
			Name string `validate:"sha512"`
		}
		assertSchema(t, SHA512{}, "v3", "v4")
	})

	t.Run("bad2", func(t *testing.T) {
		type Bad2 struct {
			Name string `validate:"bad2"`
		}
		assert.Panics(t, func() {
			StructToZodSchema(Bad2{})
		})
	})
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

		golden.Assert(t, []byte(NewConverterWithOpts(WithCustomTags(customTagHandlers)).Convert(Payload{})))
	})

	t.Run("ip unions inherit generic string constraints", func(t *testing.T) {
		type Payload struct {
			Address string `validate:"ip,required,max=45"`
		}

		assertSchema(t, Payload{}, "v4")
	})

	t.Run("oneof takes precedence over ip specialization", func(t *testing.T) {
		type Payload struct {
			Address string `validate:"oneof='127.0.0.1' '::1',ip"`
		}

		assertSchema(t, Payload{}, "v4")
	})

	t.Run("ip mixed with another format falls back to legacy chain semantics", func(t *testing.T) {
		type Payload struct {
			Address string `validate:"email,ip"`
		}

		assertSchema(t, Payload{}, "v4")
	})

	t.Run("enum keyed maps become partial records", func(t *testing.T) {
		type Payload struct {
			Metadata map[string]string `validate:"dive,keys,oneof=draft published,endkeys"`
		}

		assertSchema(t, Payload{}, "v4")
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

		assertSchema(t, Node{}, "v4")
	})

	t.Run("recursive embedded shapes keep named fields before spreads", func(t *testing.T) {
		type TreeNode struct {
			Value     string
			CreatedAt time.Time
			Children  *[]TreeNode
		}

		type Tree struct {
			TreeNode
			UpdatedAt time.Time
		}

		assertSchema(t, Tree{}, "v4")
	})
}

func TestNumberValidations(t *testing.T) {
	t.Run("gte_lte", func(t *testing.T) {
		type User1 struct {
			Age int `validate:"gte=18,lte=60"`
		}
		assertSchema(t, User1{})
	})

	t.Run("gt_lt", func(t *testing.T) {
		type User2 struct {
			Age int `validate:"gt=18,lt=60"`
		}
		assertSchema(t, User2{})
	})

	t.Run("eq", func(t *testing.T) {
		type User3 struct {
			Age int `validate:"eq=18"`
		}
		assertSchema(t, User3{})
	})

	t.Run("ne", func(t *testing.T) {
		type User4 struct {
			Age int `validate:"ne=18"`
		}
		assertSchema(t, User4{})
	})

	t.Run("oneof", func(t *testing.T) {
		type User5 struct {
			Age int `validate:"oneof=18 19 20"`
		}
		assertSchema(t, User5{})
	})

	t.Run("min_max", func(t *testing.T) {
		type User6 struct {
			Age int `validate:"min=18,max=60"`
		}
		assertSchema(t, User6{})
	})

	t.Run("len", func(t *testing.T) {
		type User7 struct {
			Age int `validate:"len=18"`
		}
		assertSchema(t, User7{})
	})

	t.Run("bad", func(t *testing.T) {
		type User8 struct {
			Age int `validate:"bad=18"`
		}
		assert.Panics(t, func() {
			StructToZodSchema(User8{})
		})
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
	t.Run("required", func(t *testing.T) {
		type Required struct {
			Map map[string]string `validate:"required"`
		}
		assertSchema(t, Required{})
	})

	t.Run("min", func(t *testing.T) {
		type Min struct {
			Map map[string]string `validate:"min=1"`
		}
		assertSchema(t, Min{})
	})

	t.Run("max", func(t *testing.T) {
		type Max struct {
			Map map[string]string `validate:"max=1"`
		}
		assertSchema(t, Max{})
	})

	t.Run("len", func(t *testing.T) {
		type Len struct {
			Map map[string]string `validate:"len=1"`
		}
		assertSchema(t, Len{})
	})

	t.Run("minmax", func(t *testing.T) {
		type MinMax struct {
			Map map[string]string `validate:"min=1,max=2"`
		}
		assertSchema(t, MinMax{})
	})

	t.Run("eq", func(t *testing.T) {
		type Eq struct {
			Map map[string]string `validate:"eq=1"`
		}
		assertSchema(t, Eq{})
	})

	t.Run("ne", func(t *testing.T) {
		type Ne struct {
			Map map[string]string `validate:"ne=1"`
		}
		assertSchema(t, Ne{})
	})

	t.Run("gt", func(t *testing.T) {
		type Gt struct {
			Map map[string]string `validate:"gt=1"`
		}
		assertSchema(t, Gt{})
	})

	t.Run("gte", func(t *testing.T) {
		type Gte struct {
			Map map[string]string `validate:"gte=1"`
		}
		assertSchema(t, Gte{})
	})

	t.Run("lt", func(t *testing.T) {
		type Lt struct {
			Map map[string]string `validate:"lt=1"`
		}
		assertSchema(t, Lt{})
	})

	t.Run("lte", func(t *testing.T) {
		type Lte struct {
			Map map[string]string `validate:"lte=1"`
		}
		assertSchema(t, Lte{})
	})

	t.Run("bad", func(t *testing.T) {
		type Bad struct {
			Map map[string]string `validate:"bad=1"`
		}
		assert.Panics(t, func() { StructToZodSchema(Bad{}) })
	})

	t.Run("dive1", func(t *testing.T) {
		type Dive1 struct {
			Map map[string]string `validate:"dive,min=2"`
		}
		assertSchema(t, Dive1{})
	})

	t.Run("dive2", func(t *testing.T) {
		type Dive2 struct {
			Map []map[string]string `validate:"required,dive,min=2,dive,min=3"`
		}
		assertSchema(t, Dive2{})
	})

	t.Run("dive3", func(t *testing.T) {
		type Dive3 struct {
			Map []map[string]string `validate:"required,dive,min=2,dive,keys,min=3,endkeys,max=4"`
		}
		assertSchema(t, Dive3{})
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

func TestCustom(t *testing.T) {
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
	golden.Assert(t, []byte(v4out))
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
	golden.Assert(t, []byte(v4out))
}

func TestConvertSliceWithValidations(t *testing.T) {
	t.Run("required", func(t *testing.T) {
		type Required struct {
			Slice []string `validate:"required"`
		}
		assertSchema(t, Required{})
	})

	t.Run("min", func(t *testing.T) {
		type Min struct {
			Slice []string `validate:"min=1"`
		}
		assertSchema(t, Min{})
	})

	t.Run("max", func(t *testing.T) {
		type Max struct {
			Slice []string `validate:"max=1"`
		}
		assertSchema(t, Max{})
	})

	t.Run("len", func(t *testing.T) {
		type Len struct {
			Slice []string `validate:"len=1"`
		}
		assertSchema(t, Len{})
	})

	t.Run("eq", func(t *testing.T) {
		type Eq struct {
			Slice []string `validate:"eq=1"`
		}
		assertSchema(t, Eq{})
	})

	t.Run("gt", func(t *testing.T) {
		type Gt struct {
			Slice []string `validate:"gt=1"`
		}
		assertSchema(t, Gt{})
	})

	t.Run("gte", func(t *testing.T) {
		type Gte struct {
			Slice []string `validate:"gte=1"`
		}
		assertSchema(t, Gte{})
	})

	t.Run("lt", func(t *testing.T) {
		type Lt struct {
			Slice []string `validate:"lt=1"`
		}
		assertSchema(t, Lt{})
	})

	t.Run("lte", func(t *testing.T) {
		type Lte struct {
			Slice []string `validate:"lte=1"`
		}
		assertSchema(t, Lte{})
	})

	t.Run("ne", func(t *testing.T) {
		type Ne struct {
			Slice []string `validate:"ne=0"`
		}
		assertSchema(t, Ne{})
	})

	t.Run("bad_oneof", func(t *testing.T) {
		assert.Panics(t, func() {
			type Bad struct {
				Slice []string `validate:"oneof=a b c"`
			}
			StructToZodSchema(Bad{})
		})
	})

	t.Run("dive1", func(t *testing.T) {
		type Dive1 struct {
			Slice [][]string `validate:"dive,required"`
		}
		assertSchema(t, Dive1{})
	})

	t.Run("dive2", func(t *testing.T) {
		type Dive2 struct {
			Slice [][]string `validate:"required,dive,min=1"`
		}
		assertSchema(t, Dive2{})
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
	golden.Assert(t, []byte(v4out))
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
		golden.Assert(t, []byte(NewConverterWithOpts(WithCustomTags(customTagHandlers), WithZodV3()).Convert(Request{})))
	})
	t.Run("v4", func(t *testing.T) {
		golden.Assert(t, []byte(NewConverterWithOpts(WithCustomTags(customTagHandlers)).Convert(Request{})))
	})
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
		golden.Assert(t, []byte(c.Export()))
	})
	t.Run("v4", func(t *testing.T) {
		c := NewConverterWithOpts()
		c.AddType(ItemA{})
		c.AddType(ItemB{})
		c.AddType(ItemC{})
		c.AddType(ItemD{})
		c.AddType(ItemE{})
		c.AddType(ItemF{})
		golden.Assert(t, []byte(c.Export()))
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
