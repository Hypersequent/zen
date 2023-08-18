# Zen

Zod + Generate = Zen

Converts Go structs with go-validator validations to Zod schemas.

Zen supports self-referential types.

## Usage:

```go
type Post struct {
	Title string `validate:"required"`
}
type User struct {
	Name       string `validate:"required"`
	Nickname   *string // pointers become optional
	Age        int 	 `validate:"min=18"`
	Height     float64 `validate:"min=0,max=3"`
	Tags       []string `validate:"min=1"`
	Favourites []struct { // nested structs are kept inline
		Name string `validate:"required"`
	}
	Posts []Post // external structs are emitted as separate exports
}
StructToZodSchema(User{})
```

Outputs:

```typescript
export const PostSchema = z.object({
	Title: z.string().min(1),
})
export type Post = z.infer<typeof PostSchema>

export const UserSchema = z.object({
	Name: z.string().min(1),
	Nickname: z.string().nullable(),
	Age: z.number().gte(18),
	Height: z.number().gte(0).lte(3),
	Tags: z.string().array().min(1),
	Favourites: z.object({
		Name: z.string().min(1),
	}).array().nullable(),
	Posts: PostSchema.array().nullable(),
})
export type User = z.infer<typeof UserSchema>
```

It also works without any validations.

### How we use it at Hypersequent

- We have all the types declared in a single module
- Use `filepath.Walk` to find all the go files.
- We extract all the struct names using string manipulation on the files contents.
- Then using go templates and passing these struct names as input, we generate go code that is later used to generate the zod schemas.

```go.tmpl
	converter := zen.NewConverter(make(map[string]zen.CustomFn))

	{{range .TypesToGenerate}}
  converter.AddType(types.{{.}}{})
	{{end}}

	schema := converter.Export()
```

## Custom Types

You can pass type name mappings to custom conversion functions:

```go
c := zen.NewConverter(map[string]zen.CustomFn{
	"github.com/shopspring/decimal.Decimal": func (c *zen.Converter, t reflect.Type, s, g string, i int) string {
		// Shopspring's decimal type serialises to a string.
		return "z.string()"
	},
})

c.Convert(User{
	Money decimal.Decimal
})
```

Outputs:

```typescript
export const UserSchema = z.object({
	Money: z.string(),
})
export type User = z.infer<typeof UserSchema>
```

There are some custom types with tests in the "custom" directory.

The function signature for custom type handlers is:

```go
func(c *zen.Converter, t reflect.Type, typeName, genericTypeName string, indentLevel int) string
```

You can use the Converter to process nested types. The `genericTypeName` is the name of the `T` in `Generic[T]` and the
indent level is for passing to other converter APIs.

## Supported validations

### Network

| Tag         | Description                    |
|-------------|--------------------------------|
| ip          | Internet Protocol Address IP   |
| ip4_addr    | Internet Protocol Address IPv4 |
| ip6_addr    | Internet Protocol Address IPv6 |
| ip_addr     | Internet Protocol Address IP   |
| ipv4        | Internet Protocol Address IPv4 |
| ipv6        | Internet Protocol Address IPv6 |
| url         | URL String                     |
| http_url    | HTTP URL String                |
| url_encoded | URL Encoded                    |

### Strings

| Tag             | Description          |
|-----------------|----------------------|
| alpha           | Alpha Only           |
| alphanum        | Alphanumeric         |
| alphanumunicode | Alphanumeric Unicode |
| alphaunicode    | Alpha Unicode        |
| ascii           | ASCII                |
| boolean         | Boolean              |
| contains        | Contains             |
| endswith        | Ends With            |
| lowercase       | Lowercase            |
| number          | Number               |
| numeric         | Numeric              |
| startswith      | Starts With          |
| uppercase       | Uppercase            |

### Format

| Tag           | Description                                   |
|---------------|-----------------------------------------------|
| base64        | Base64 String                                 |
| mongodb       | MongoDB ObjectID                              |
| datetime      | Datetime                                      |
| email         | E-mail String                                 |
| hexadecimal   | Hexadecimal String                            |
| html_encoded  | HTML Encoded                                  |
| json          | JSON                                          |
| jwt           | JSON Web Token (JWT)                          |
| latitude      | Latitude                                      |
| longitude     | Longitude                                     |
| uuid          | Universally Unique Identifier UUID            |
| uuid3         | Universally Unique Identifier UUID v3         |
| uuid3_rfc4122 | Universally Unique Identifier UUID v3 RFC4122 |
| uuid4         | Universally Unique Identifier UUID v4         |
| uuid4_rfc4122 | Universally Unique Identifier UUID v4 RFC4122 |
| uuid5         | Universally Unique Identifier UUID v5         |
| uuid5_rfc4122 | Universally Unique Identifier UUID v5 RFC4122 |
| uuid_rfc4122  | Universally Unique Identifier UUID RFC4122    |
| md4           | MD4 hash                                      |
| md5           | MD5 hash                                      |
| sha256        | SHA256 hash                                   |
| sha384        | SHA384 hash                                   |
| sha512        | SHA512 hash                                   |

### Comparisons

| Tag | Description           |
|-----|-----------------------|
| eq  | Equals                |
| gt  | Greater than          |
| gte | Greater than or equal |
| lt  | Less Than             |
| lte | Less Than or Equal    |
| ne  | Not Equal             |

- For strings & numbers, will ensure that the value is compared to the parameter given. For slices, arrays, and maps,
	validates the number of items.
- (time, duration and maps are not supported)

### Other

| Tag      | Description |
|----------|-------------|
| len      | Length      |
| max      | Maximum     |
| min      | Minimum     |
| oneof    | One Of      |
| required | Required    |

- required checks that the value is not default, but we are not implementing this check for numbers and booleans

## Caveats

- Does not support cyclic types - it's a limitation of zod, but self-referential types are supported.
- Sometimes outputs in the wrong order - it really needs an intermediate DAG to solve this.

## License

- Distributed under MIT License, please see license file within the code for more details.

## Credits

- Inspired by [supervillain](https://github.com/Southclaws/supervillain)
- Uses several regexes from [validator](https://github.com/go-playground/validator)
