# Zen

Zod + Generate = Zen

Converts Go structs with go-validator validations to Zod schemas.

Usage:

```go
type Post struct {
Title string
}
type User struct {
Name       string
Nickname   *string // pointers become optional
Age        int
Height     float64
Tags       []string
Favourites []struct { // nested structs are kept inline
Name string
}
Posts []Post // external structs are emitted as separate exports
}
StructToZodSchema(User{})

```

Outputs:

```typescript
export const PostSchema = z.object({
  title: z.string(),
});
export type Post = z.infer<typeof PostSchema>;

export const UserSchema = z.object({
  name: z.string(),
  nickname: z.string().optional(),
  age: z.number(),
  height: z.number(),
  tags: z.string().array(),
  favourites: z
    .object({
      name: z.string(),
    })
    .array(),
  posts: PostSchema.array(),
});
export type User = z.infer<typeof UserSchema>;
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

### Fields

|      | Tag           | Description                                                 |
|------|---------------|-------------------------------------------------------------|
|      | eqcsfield     | Field Equals Another Field (relative)                       |
|      | eqfield       | Field Equals Another Field                                  |
|      | fieldcontains | Check the indicated characters are present in the Field     |
|      | fieldexcludes | Check the indicated characters are not present in the field |
|      | gtcsfield     | Field Greater Than Another Relative Field                   |
|      | gtecsfield    | Field Greater Than or Equal To Another Relative Field       |
|      | gtefield      | Field Greater Than or Equal To Another Field                |
|      | gtfield       | Field Greater Than Another Field                            |
|      | ltcsfield     | Less Than Another Relative Field                            |
|      | ltecsfield    | Less Than or Equal To Another Relative Field                |
|      | ltefield      | Less Than or Equal To Another Field                         |
|      | ltfield       | Less Than Another Field                                     |
|      | necsfield     | Field Does Not Equal Another Field (relative)               |
|      | nefield       | Field Does Not Equal Another Field                          |

### Network

|   | Tag              | Description                                 |
|---|------------------|---------------------------------------------|
|   | cidr             | Classless Inter-Domain Routing CIDR         |
|   | cidrv4           | Classless Inter-Domain Routing CIDRv4       |
|   | cidrv6           | Classless Inter-Domain Routing CIDRv6       |
|   | datauri          | Data URL                                    |
|   | fqdn             | Full Qualified Domain Name (FQDN)           |
|   | hostname         | Hostname RFC 952                            |
|   | hostname_port    | HostPort                                    |
|   | hostname_rfc1123 | Hostname RFC 1123                           |
| ✅ | ip               | Internet Protocol Address IP                |
| ✅ | ip4_addr         | Internet Protocol Address IPv4              |
| ✅ | ip6_addr         | Internet Protocol Address IPv6              |
| ✅ | ip_addr          | Internet Protocol Address IP                |
| ✅ | ipv4             | Internet Protocol Address IPv4              |
| ✅ | ipv6             | Internet Protocol Address IPv6              |
|   | mac              | Media Access Control Address MAC            |
|   | tcp4_addr        | Transmission Control Protocol Address TCPv4 |
|   | tcp6_addr        | Transmission Control Protocol Address TCPv6 |
|   | tcp_addr         | Transmission Control Protocol Address TCP   |
|   | udp4_addr        | User Datagram Protocol Address UDPv4        |
|   | udp6_addr        | User Datagram Protocol Address UDPv6        |
|   | udp_addr         | User Datagram Protocol Address UDP          |
|   | unix_addr        | Unix domain socket end point Address        |
|   | uri              | URI String                                  |
| ✅ | url              | URL String                                  |
| ✅ | http_url         | HTTP URL String                             |
| ✅ | url_encoded      | URL Encoded                                 |
|   | urn_rfc2141      | Urn RFC 2141 String                         |

### Strings

|   | Tag             | Description           |
|---|-----------------|-----------------------|
| ✅ | alpha           | Alpha Only            |
| ✅ | alphanum        | Alphanumeric          |
| ✅ | alphanumunicode | Alphanumeric Unicode  |
| ✅ | alphaunicode    | Alpha Unicode         |
| ✅ | ascii           | ASCII                 |
| ✅ | boolean         | Boolean               |
| ✅ | contains        | Contains              |
|   | containsany     | Contains Any          |
|   | containsrune    | Contains Rune         |
|   | endsnotwith     | Ends Not With         |
| ✅ | endswith        | Ends With             |
|   | excludes        | Excludes              |
|   | excludesall     | Excludes All          |
|   | excludesrune    | Excludes Rune         |
| ✅ | lowercase       | Lowercase             |
|   | multibyte       | Multi-Byte Characters |
| ✅ | number          | Number                |
| ✅ | numeric         | Numeric               |
|   | printascii      | Printable ASCII       |
|   | startsnotwith   | Starts Not With       |
| ✅ | startswith      | Starts With           |
| ✅ | uppercase       | Uppercase             |

### Format

|   | Tag                           | Description                                                   |
|---|-------------------------------|---------------------------------------------------------------|
| ✅ | base64                        | Base64 String                                                 |
|   | base64url                     | Base64URL String                                              |
|   | base64rawurl                  | Base64RawURL String                                           |
|   | bic                           | Business Identifier Code (ISO 9362)                           |
|   | bcp47_language_tag            | Language tag (BCP 47)                                         |
|   | btc_addr                      | Bitcoin Address                                               |
|   | btc_addr_bech32               | Bitcoin Bech32 Address (segwit)                               |
|   | credit_card                   | Credit Card Number                                            |
| ✅ | mongodb                       | MongoDB ObjectID                                              |
|   | cron                          | Cron                                                          |
| ✅ | datetime                      | Datetime                                                      |
|   | e164                          | e164 formatted phone number                                   |
| ✅ | email                         | E-mail String                                                 |
|   | eth_addr                      | Ethereum Address                                              |
| ✅ | hexadecimal                   | Hexadecimal String                                            |
|   | hexcolor                      | Hexcolor String                                               |
|   | hsl                           | HSL String                                                    |
|   | hsla                          | HSLA String                                                   |
|   | html                          | HTML Tags                                                     |
| ✅ | html_encoded                  | HTML Encoded                                                  |
|   | isbn                          | International Standard Book Number                            |
|   | isbn10                        | International Standard Book Number 10                         |
|   | isbn13                        | International Standard Book Number 13                         |
|   | iso3166_1_alpha2              | Two-letter country code (ISO 3166-1 alpha-2)                  |
|   | iso3166_1_alpha3              | Three-letter country code (ISO 3166-1 alpha-3)                |
|   | iso3166_1_alpha_numeric       | Numeric country code (ISO 3166-1 numeric)                     |
|   | iso3166_2                     | Country subdivision code (ISO 3166-2)                         |
|   | iso4217                       | Currency code (ISO 4217)                                      |
| ✅ | json                          | JSON                                                          |
| ✅ | jwt                           | JSON Web Token (JWT)                                          |
| ✅ | latitude                      | Latitude                                                      |
| ✅ | longitude                     | Longitude                                                     |
|   | luhn_checksum                 | Luhn Algorithm Checksum (for strings and (u)int)              |
|   | postcode_iso3166_alpha2       | Postcode                                                      |
|   | postcode_iso3166_alpha2_field | Postcode                                                      |
|   | rgb                           | RGB String                                                    |
|   | rgba                          | RGBA String                                                   |
|   | ssn                           | Social Security Number SSN                                    |
|   | timezone                      | Timezone                                                      |
| ✅ | uuid                          | Universally Unique Identifier UUID                            |
| ✅ | uuid3                         | Universally Unique Identifier UUID v3                         |
| ✅ | uuid3_rfc4122                 | Universally Unique Identifier UUID v3 RFC4122                 |
| ✅ | uuid4                         | Universally Unique Identifier UUID v4                         |
| ✅ | uuid4_rfc4122                 | Universally Unique Identifier UUID v4 RFC4122                 |
| ✅ | uuid5                         | Universally Unique Identifier UUID v5                         |
| ✅ | uuid5_rfc4122                 | Universally Unique Identifier UUID v5 RFC4122                 |
| ✅ | uuid_rfc4122                  | Universally Unique Identifier UUID RFC4122                    |
| ✅ | md4                           | MD4 hash                                                      |
| ✅ | md5                           | MD5 hash                                                      |
| ✅ | sha256                        | SHA256 hash                                                   |
| ✅ | sha384                        | SHA384 hash                                                   |
| ✅ | sha512                        | SHA512 hash                                                   |
|   | ripemd128                     | RIPEMD-128 hash                                               |
|   | ripemd128                     | RIPEMD-160 hash                                               |
|   | tiger128                      | TIGER128 hash                                                 |
|   | tiger160                      | TIGER160 hash                                                 |
|   | tiger192                      | TIGER192 hash                                                 |
|   | semver                        | Semantic Versioning 2.0.0                                     |
|   | ulid                          | Universally Unique Lexicographically Sortable Identifier ULID |
|   | cve                           | Common Vulnerabilities and Exposures Identifier (CVE id)      |

### Comparisons

|   | Tag            | Description             |
|---|----------------|-------------------------|
| ✅ | eq             | Equals                  |
|   | eq_ignore_case | Equals ignoring case    |
| ✅ | gt             | Greater than            |
| ✅ | gte            | Greater than or equal   |
| ✅ | lt             | Less Than               |
| ✅ | lte            | Less Than or Equal      |
| ✅ | ne             | Not Equal               |
|   | ne_ignore_case | Not Equal ignoring case |

- For strings & numbers, will ensure that the value is compared to the parameter given. For slices, arrays, and maps,
  validates the number of items.
- (time, duration and maps are not supported as of now)

### Other

|   | Tag                  | Description          |
|---|----------------------|----------------------|
|   | dir                  | Existing Directory   |
|   | dirpath              | Directory Path       |
|   | file                 | Existing File        |
|   | filepath             | File Path            |
|   | isdefault            | Is Default           |
| ✅ | len                  | Length               |
| ✅ | max                  | Maximum              |
| ✅ | min                  | Minimum              |
| ✅ | oneof                | One Of               |
| ✅ | required             | Required             |
|   | required_if          | Required If          |
|   | required_unless      | Required Unless      |
|   | required_with        | Required With        |
|   | required_with_all    | Required With All    |
|   | required_without     | Required Without     |
|   | required_without_all | Required Without All |
|   | excluded_if          | Excluded If          |
|   | excluded_unless      | Excluded Unless      |
|   | excluded_with        | Excluded With        |
|   | excluded_with_all    | Excluded With All    |
|   | excluded_without     | Excluded Without     |
|   | excluded_without_all | Excluded Without All |
|   | unique               | Unique               |

- required checks that the value is not default, but we are not implementing this check for numbers and booleans

### Aliases

|      | Tag          | Description                                                 |
|------|--------------|-------------------------------------------------------------|
|      | iscolor      | hexcolor\|rgb\|rgba\|hsl\|hsla                              |
|      | country_code | iso3166_1_alpha2\|iso3166_1_alpha3\|iso3166_1_alpha_numeric |

## Caveats

- Does not support self-referential types - should be a simple fix.
- Sometimes outputs in the wrong order - it really needs an intermediate DAG to solve this.

## Credits

- Inspired by [supervillain](https://github.com/Southclaws/supervillain).
