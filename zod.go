package zen

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func NewConverter(custom map[string]CustomFn) Converter {
	c := Converter{
		prefix:  "",
		outputs: make(map[string]entry),
		custom:  custom,
	}

	return c
}

func (c *Converter) AddType(input interface{}) {
	t := reflect.TypeOf(input)

	c.addSchema(t.Name(), c.convertStructTopLevel(t))
}

func (c *Converter) Convert(input interface{}) string {
	t := reflect.TypeOf(input)

	c.addSchema(t.Name(), c.convertStructTopLevel(t))

	return c.Export()
}

func (c *Converter) ConvertSlice(inputs []interface{}) string {
	for _, input := range inputs {
		t := reflect.TypeOf(input)
		c.addSchema(t.Name(), c.convertStructTopLevel(t))
	}

	return c.Export()
}

func StructToZodSchema(input interface{}) string {
	c := Converter{
		prefix:  "",
		outputs: make(map[string]entry),
	}

	t := reflect.TypeOf(input)

	c.addSchema(t.Name(), c.convertStructTopLevel(t))

	return c.Export()
}

func StructToZodSchemaWithPrefix(prefix string, input interface{}) string {
	c := Converter{
		prefix:  prefix,
		outputs: make(map[string]entry),
	}

	t := reflect.TypeOf(input)

	c.addSchema(t.Name(), c.convertStructTopLevel(t))

	return c.Export()
}

var typeMapping = map[reflect.Kind]string{
	reflect.Bool:       "boolean",
	reflect.Int:        "number",
	reflect.Int8:       "number",
	reflect.Int16:      "number",
	reflect.Int32:      "number",
	reflect.Int64:      "number",
	reflect.Uint:       "number",
	reflect.Uint8:      "number",
	reflect.Uint16:     "number",
	reflect.Uint32:     "number",
	reflect.Uint64:     "number",
	reflect.Uintptr:    "number",
	reflect.Float32:    "number",
	reflect.Float64:    "number",
	reflect.Complex64:  "number",
	reflect.Complex128: "number",
	reflect.String:     "string",
	reflect.Interface:  "any",
}

type entry struct {
	order int
	data  string
}

type ByOrder []entry

func (a ByOrder) Len() int           { return len(a) }
func (a ByOrder) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByOrder) Less(i, j int) bool { return a[i].order < a[j].order }

type CustomFn func(*Converter, reflect.Type, string, string, string, int) string

type Converter struct {
	prefix  string
	structs int
	outputs map[string]entry
	custom  map[string]CustomFn
}

func (c *Converter) addSchema(name string, data string) {
	// First check if the object already exists. If it does do not replace. This is needed for second order
	_, ok := c.outputs[name]
	if !ok {
		order := c.structs
		c.outputs[name] = entry{order, data}
		c.structs = order + 1
	}
}

func (c *Converter) Export() string {
	output := strings.Builder{}
	var sorted []entry
	for _, ent := range c.outputs {
		sorted = append(sorted, ent)
	}

	sort.Sort(ByOrder(sorted))

	for _, ent := range sorted {
		output.WriteString(ent.data)
		output.WriteString("\n\n")
	}
	return output.String()
}

func schemaName(prefix, name string) string {
	return fmt.Sprintf("%s%sSchema", prefix, name)
}

func fieldName(input reflect.StructField) string {
	if json := input.Tag.Get("json"); json != "" {
		args := strings.Split(json, ",")
		if len(args[0]) > 0 {
			return args[0]
		}
		// This is also valid:
		// json:",omitempty"
		// so in this case, args[0] will be empty, so fall through to using the
		// raw field name.
	}

	// When Golang marshals a struct to JSON, and it doesn't have any JSON tags
	// that give the fields names, it defaults to just using the field's name.
	return input.Name
}

func typeName(t reflect.Type) string {
	if t.Kind() == reflect.Struct {
		return t.Name()
	}
	if t.Kind() == reflect.Ptr {
		return typeName(t.Elem())
	}
	if t.Kind() == reflect.Slice {
		return typeName(t.Elem())
	}
	if t.Kind() == reflect.Map {
		return typeName(t.Elem())
	}

	return "UNKNOWN"
}

func (c *Converter) convertStructTopLevel(t reflect.Type) string {
	output := strings.Builder{}

	name := t.Name()

	output.WriteString(fmt.Sprintf(
		`export const %s = %s
`,
		schemaName(c.prefix, name), c.convertStruct(t, 0)))

	output.WriteString(fmt.Sprintf(`export type %s%s = z.infer<typeof %s%sSchema>`,
		c.prefix, name, c.prefix, name))

	return output.String()
}

func (c *Converter) convertStruct(input reflect.Type, indent int) string {
	output := strings.Builder{}

	output.WriteString(`z.object({
`)

	fields := input.NumField()
	for i := 0; i < fields; i++ {
		field := input.Field(i)
		optional := isOptional(field)
		nullable := isNullable(field)

		line := c.convertField(field, indent+1, optional, nullable)

		output.WriteString(line)
	}

	output.WriteString(indentation(indent))
	output.WriteString(`})`)

	return output.String()
}

var matchGenericTypeName = regexp.MustCompile(`(.+)\[(.+)]`)

// checking it a reflected type is a generic isn't supported as far as I can see
// so this simple check looks for a `[` character in the type name: `T1[T2]`.
func isGeneric(t reflect.Type) bool {
	return strings.Contains(t.Name(), "[")
}

// gets the full name and if it's a generic type, strips out the [T] part.
func getFullName(t reflect.Type) (string, string) {
	var typename string
	var generic string

	if isGeneric(t) {
		m := matchGenericTypeName.FindAllStringSubmatch(t.Name(), 1)[0]

		typename = m[1]
		generic = m[2]
	} else {
		typename = t.Name()
	}

	return fmt.Sprintf("%s.%s", t.PkgPath(), typename), generic
}

func (c *Converter) handleCustomType(t reflect.Type, name, validate string, indent int) (string, bool) {
	fullName, generic := getFullName(t)

	custom, ok := c.custom[fullName]
	if ok {
		return custom(c, t, name, generic, validate, indent), true
	}

	return "", false
}

func (c *Converter) ConvertType(t reflect.Type, name string, validate string, indent int) string {
	if t.Kind() == reflect.Ptr {
		inner := t.Elem()
		return c.ConvertType(inner, name, validate, indent)
	}

	if custom, ok := c.handleCustomType(t, name, validate, indent); ok {
		return custom
	}

	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		return c.convertSliceAndArray(t, name, validate, indent)
	}

	if t.Kind() == reflect.Struct {
		// Handle nested un-named structs - these are inline.
		if t.Name() == "" {
			return c.convertStruct(t, indent)
		} else if t.Name() == "Time" {
			var validateStr string
			if validate != "" {
				// We compare with both the zero value from go and the zero value that zod coerces to
				if validate == "required" {
					validateStr = ".refine((val) => val.getTime() !== new Date('0001-01-01T00:00:00Z').getTime() && val.getTime() !== new Date(0).getTime(), 'Invalid date')"
				}
			}
			// timestamps are to be coerced to date by zod. JSON.parse only serializes to string
			return "z.coerce.date()" + validateStr
		} else {
			c.addSchema(name, c.convertStructTopLevel(t))
			return schemaName(c.prefix, name)
		}
	}

	if t.Kind() == reflect.Map {
		return c.convertMap(t, name, validate, indent)
	}

	zodType, ok := typeMapping[t.Kind()]
	if !ok {
		panic(fmt.Sprint("cannot handle: ", t.Kind()))
	}

	var validateStr string
	if validate != "" {
		switch zodType {
		case "string":
			validateStr = c.validateString(validate)
			if strings.Contains(validateStr, ".enum(") {
				return "z" + validateStr
			}
		case "number":
			validateStr = c.validateNumber(validate)
		}
	}

	return fmt.Sprintf("z.%s()%s", zodType, validateStr)
}

func (c *Converter) convertField(f reflect.StructField, indent int, optional, nullable bool) string {
	name := fieldName(f)

	// fields named `-` are not exported to JSON so don't export zod types
	if name == "-" {
		return ""
	}

	// because nullability is processed before custom types, this makes sure
	// the custom type has control over nullability.
	fullName, _ := getFullName(f.Type)
	_, isCustom := c.custom[fullName]

	optionalCall := ""
	if optional {
		optionalCall = ".optional()"
	}
	nullableCall := ""
	if nullable && !isCustom {
		nullableCall = ".nullable()"
	}

	return fmt.Sprintf(
		"%s%s: %s%s%s,\n",
		indentation(indent),
		name,
		c.ConvertType(f.Type, typeName(f.Type), f.Tag.Get("validate"), indent),
		optionalCall,
		nullableCall)
}

func (c *Converter) convertSliceAndArray(t reflect.Type, name, validate string, indent int) string {
	if t.Kind() == reflect.Array {
		return fmt.Sprintf(
			"%s.array()%s",
			c.ConvertType(t.Elem(), name, getValidateAfterDive(validate), indent), fmt.Sprintf(".length(%d)", t.Len()))
	}

	var validateStr strings.Builder
	validateCurrent := getValidateCurrent(validate)
	if validateCurrent != "" {
		parts := strings.Split(validateCurrent, ",")

		// eq and ne should be at the end since they output a refine function
		sort.SliceStable(parts, func(i, j int) bool {
			if strings.HasPrefix(parts[i], "ne") {
				return false
			}
			if strings.HasPrefix(parts[j], "ne") {
				return true
			}
			return i < j
		})

		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "dive" {
				break
			} else if part == "required" {
				validateStr.WriteString(".nonempty()")
			} else if strings.HasPrefix(part, "min=") {
				validateStr.WriteString(fmt.Sprintf(".min(%s)", part[4:]))
			} else if strings.HasPrefix(part, "max=") {
				validateStr.WriteString(fmt.Sprintf(".max(%s)", part[4:]))
			} else if strings.HasPrefix(part, "len=") {
				validateStr.WriteString(fmt.Sprintf(".length(%s)", part[4:]))
			} else if strings.HasPrefix(part, "eq=") {
				validateStr.WriteString(fmt.Sprintf(".length(%s)", part[3:]))
			} else if strings.HasPrefix(part, "ne=") {
				validateStr.WriteString(fmt.Sprintf(".refine((val) => val.length !== %s)", part[3:]))
			} else if strings.HasPrefix(part, "gt=") {
				val, err := strconv.Atoi(part[3:])
				if err != nil || val < 0 {
					panic(fmt.Sprintf("invalid gt value: %s", part[3:]))
				}
				validateStr.WriteString(fmt.Sprintf(".min(%d)", val+1))
			} else if strings.HasPrefix(part, "gte=") {
				validateStr.WriteString(fmt.Sprintf(".min(%s)", part[4:]))
			} else if strings.HasPrefix(part, "lt=") {
				val, err := strconv.Atoi(part[3:])
				if err != nil || val <= 0 {
					panic(fmt.Sprintf("invalid lt value: %s", part[3:]))
				}
				validateStr.WriteString(fmt.Sprintf(".max(%d)", val-1))
			} else if strings.HasPrefix(part, "lte=") {
				validateStr.WriteString(fmt.Sprintf(".max(%s)", part[4:]))
			} else {
				panic(fmt.Sprintf("unknown validation: %s", part))
			}
		}
	}

	return fmt.Sprintf(
		"%s.array()%s",
		c.ConvertType(t.Elem(), name, getValidateAfterDive(validate), indent), validateStr.String())
}

func (c *Converter) convertMap(t reflect.Type, name, validate string, indent int) string {
	var validateStr strings.Builder
	if validate != "" {
		parts := strings.Split(validate, ",")

		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "dive" {
				break
			} else if part == "required" {
				validateStr.WriteString(".refine((val) => Object.keys(val).length > 0, 'Empty map')")
			} else if strings.HasPrefix(part, "min=") {
				validateStr.WriteString(fmt.Sprintf(".refine((val) => Object.keys(val).length >= %s, 'Map too small')", part[4:]))
			} else if strings.HasPrefix(part, "max=") {
				validateStr.WriteString(fmt.Sprintf(".refine((val) => Object.keys(val).length <= %s, 'Map too large')", part[4:]))
			} else if strings.HasPrefix(part, "len=") {
				validateStr.WriteString(fmt.Sprintf(".refine((val) => Object.keys(val).length === %s, 'Map wrong size')", part[4:]))
			} else if strings.HasPrefix(part, "eq=") {
				validateStr.WriteString(fmt.Sprintf(".refine((val) => Object.keys(val).length === %s, 'Map wrong size')", part[3:]))
			} else if strings.HasPrefix(part, "ne=") {
				validateStr.WriteString(fmt.Sprintf(".refine((val) => Object.keys(val).length !== %s, 'Map wrong size')", part[3:]))
			} else if strings.HasPrefix(part, "gt=") {
				validateStr.WriteString(fmt.Sprintf(".refine((val) => Object.keys(val).length > %s, 'Map too small')", part[3:]))
			} else if strings.HasPrefix(part, "gte=") {
				validateStr.WriteString(fmt.Sprintf(".refine((val) => Object.keys(val).length >= %s, 'Map too small')", part[4:]))
			} else if strings.HasPrefix(part, "lt=") {
				validateStr.WriteString(fmt.Sprintf(".refine((val) => Object.keys(val).length < %s, 'Map too large')", part[3:]))
			} else if strings.HasPrefix(part, "lte=") {
				validateStr.WriteString(fmt.Sprintf(".refine((val) => Object.keys(val).length <= %s, 'Map too large')", part[4:]))
			} else {
				panic(fmt.Sprintf("unknown validation: %s", part))
			}
		}
	}

	return fmt.Sprintf(`z.record(%s, %s)%s`,
		c.ConvertType(t.Key(), name, getValidateKeys(validate), indent),
		c.ConvertType(t.Elem(), name, getValidateValues(validate), indent),
		validateStr.String())
}

func getValidateAfterDive(validate string) string {
	// select part of validate string after dive, if it exists
	var validateNext string
	if validate != "" {
		parts := strings.Split(validate, ",")
		for i, part := range parts {
			part = strings.TrimSpace(part)
			if part == "dive" {
				validateNext = strings.Join(parts[i+1:], ",")
				break
			}
		}
	}

	return validateNext
}

// These are to be used together directly after the dive tag and tells the validator that anything between 'keys' and 'endkeys' applies to the keys of a map and not the values; think of it like the 'dive' tag, but for map keys instead of values. Multidimensional nesting is also supported, each level you wish to validate will require another 'keys' and 'endkeys' tag. These tags are only valid for maps.
//
// Usage: dive,keys,othertagvalidation(s),endkeys,valuevalidationtags
func getValidateKeys(validate string) string {
	var validateKeys string
	if strings.Contains(validate, "keys") {
		removedSuffix := strings.SplitN(validate, ",endkeys", 2)[0]
		parts := strings.SplitN(removedSuffix, "keys,", 2)
		if len(parts) == 2 {
			validateKeys = parts[1]
		}
	}
	return validateKeys
}

func getValidateValues(validate string) string {
	var validateValues string

	if strings.Contains(validate, "dive,keys") {
		removedPrefix := strings.SplitN(validate, ",endkeys", 2)[1]

		if strings.Contains(removedPrefix, ",dive") {
			validateValues = strings.SplitN(removedPrefix, ",dive", 2)[0]
		} else {
			validateValues = removedPrefix
		}
		validateValues = strings.TrimPrefix(validateValues, ",")
	} else if strings.Contains(validate, "dive") {
		removedPrefix := strings.SplitN(validate, "dive,", 2)[1]
		if strings.Contains(removedPrefix, ",dive") {
			validateValues = strings.SplitN(removedPrefix, ",dive", 2)[0]
		} else {
			validateValues = removedPrefix
		}
	}

	return validateValues
}

func (c *Converter) validateNumber(validate string) string {
	var validateStr strings.Builder
	parts := strings.Split(validate, ",")

	// eq and ne should be at the end since they output a refine function
	sort.SliceStable(parts, func(i, j int) bool {
		if strings.HasPrefix(parts[i], "eq") || strings.HasPrefix(parts[i], "len") ||
			strings.HasPrefix(parts[i], "ne") || strings.HasPrefix(parts[i], "oneof") ||
			strings.HasPrefix(parts[i], "required") {
			return false
		}
		if strings.HasPrefix(parts[j], "eq") || strings.HasPrefix(parts[j], "len") ||
			strings.HasPrefix(parts[j], "ne") || strings.HasPrefix(parts[j], "oneof") ||
			strings.HasPrefix(parts[j], "required") {
			return true
		}
		return i < j
	})

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.ContainsRune(part, '=') {
			idx := strings.Index(part, "=")
			if idx == 0 || idx == len(part)-1 {
				panic(fmt.Sprintf("invalid validation: %s", part))
			}

			valName := part[:idx]
			valValue := part[idx+1:]

			switch valName {
			case "gt":
				validateStr.WriteString(fmt.Sprintf(".gt(%s)", valValue))
			case "gte", "min":
				validateStr.WriteString(fmt.Sprintf(".gte(%s)", valValue))
			case "lt":
				validateStr.WriteString(fmt.Sprintf(".lt(%s)", valValue))
			case "lte", "max":
				validateStr.WriteString(fmt.Sprintf(".lte(%s)", valValue))
			case "eq", "len":
				validateStr.WriteString(fmt.Sprintf(".refine((val) => val === %s)", valValue))
			case "ne":
				validateStr.WriteString(fmt.Sprintf(".refine((val) => val !== %s)", valValue))
			case "oneof":
				vals := strings.Fields(valValue)
				if len(vals) == 0 {
					panic(fmt.Sprintf("invalid oneof validation: %s", part))
				}
				validateStr.WriteString(fmt.Sprintf(".refine((val) => [%s].includes(val))", strings.Join(vals, ", ")))

			default:
				panic(fmt.Sprintf("unknown validation: %s", part))
			}
		} else if part == "required" {
			validateStr.WriteString(".refine((val) => val !== 0)")
		} else {
			panic(fmt.Sprintf("unknown validation: %s", part))
		}
	}

	return validateStr.String()
}

func (c *Converter) validateString(validate string) string {
	var validateStr strings.Builder
	parts := strings.Split(validate, ",")

	// eq and ne should be at the end since they output a refine function
	sort.SliceStable(parts, func(i, j int) bool {
		if strings.HasPrefix(parts[i], "eq") || strings.HasPrefix(parts[i], "ne") {
			return false
		}
		if strings.HasPrefix(parts[j], "eq") || strings.HasPrefix(parts[j], "ne") {
			return true
		}
		return i < j
	})

	for _, part := range parts {
		part = strings.TrimSpace(part)
		// We handle the parts which have = separately
		if strings.ContainsRune(part, '=') {
			idx := strings.Index(part, "=")
			if idx == 0 || idx == len(part)-1 {
				panic(fmt.Sprintf("invalid validation: %s", part))
			}

			valName := part[:idx]
			valValue := part[idx+1:]

			switch valName {
			case "oneof":
				vals := splitParamsRegex.FindAllString(part[6:], -1)
				for i := 0; i < len(vals); i++ {
					vals[i] = strings.Replace(vals[i], "'", "", -1)
				}
				if len(vals) == 0 {
					panic("oneof= must be followed by a list of values")
				}
				// const FishEnum = z.enum(["Salmon", "Tuna", "Trout"]);
				validateStr.WriteString(fmt.Sprintf(".enum([\"%s\"] as const)", strings.Join(vals, "\", \"")))
			case "len":
				validateStr.WriteString(fmt.Sprintf(".length(%s)", valValue))
			case "min":
				validateStr.WriteString(fmt.Sprintf(".min(%s)", valValue))
			case "max":
				validateStr.WriteString(fmt.Sprintf(".max(%s)", valValue))
			case "gt":
				val, err := strconv.Atoi(valValue)
				if err != nil {
					panic("gt= must be followed by a number")
				}
				validateStr.WriteString(fmt.Sprintf(".min(%d)", val+1))
			case "gte":
				validateStr.WriteString(fmt.Sprintf(".min(%s)", valValue))
			case "lt":
				val, err := strconv.Atoi(valValue)
				if err != nil {
					panic("lt= must be followed by a number")
				}
				validateStr.WriteString(fmt.Sprintf(".max(%d)", val-1))
			case "lte":
				validateStr.WriteString(fmt.Sprintf(".max(%s)", valValue))
			case "contains":
				validateStr.WriteString(fmt.Sprintf(".includes(\"%s\")", valValue))
			case "endswith":
				validateStr.WriteString(fmt.Sprintf(".endsWith(\"%s\")", valValue))
			case "startswith":
				validateStr.WriteString(fmt.Sprintf(".startsWith(\"%s\")", valValue))
			case "eq":
				validateStr.WriteString(fmt.Sprintf(".refine((val) => val === \"%s\")", valValue))
			case "ne":
				validateStr.WriteString(fmt.Sprintf(".refine((val) => val !== \"%s\")", valValue))

			default:
				panic(fmt.Sprintf("unknown validation: %s", part))
			}
		} else {
			switch part {
			case "required":
				validateStr.WriteString(".min(1)")
			case "email":
				// email is more readable than copying the regex in regexes.go but could be incompatible
				// Also there is an open issue https://github.com/go-playground/validator/issues/517
				// https://github.com/puellanivis/pedantic-regexps/blob/master/email.go
				// solution is there in the comments but not implemented yet
				validateStr.WriteString(".email()")
			case "url":
				// url is more readable than copying the regex in regexes.go but could be incompatible
				validateStr.WriteString(".url()")
			case "ipv4":
				validateStr.WriteString(".ip({ version: \"v4\" })")
			case "ip4_addr":
				validateStr.WriteString(".ip({ version: \"v4\" })")
			case "ipv6":
				validateStr.WriteString(".ip({ version: \"v6\" })")
			case "ip6_addr":
				validateStr.WriteString(".ip({ version: \"v6\" })")
			case "ip":
				validateStr.WriteString(".ip()")
			case "ip_addr":
				validateStr.WriteString(".ip()")
			case "http_url":
				// url is more readable than copying the regex in regexes.go but could be incompatible
				validateStr.WriteString(".url()")
			case "url_encoded":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", uRLEncodedRegexString))
			case "alpha":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", alphaRegexString))
			case "alphanum":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", alphaNumericRegexString))
			case "alphanumunicode":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", alphaUnicodeNumericRegexString))
			case "alphaunicode":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", alphaUnicodeRegexString))
			case "ascii":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", aSCIIRegexString))
			case "boolean":
				validateStr.WriteString(".enum(['true', 'false'])")
			case "lowercase":
				validateStr.WriteString(".refine((val) => val === val.toLowerCase())")
			case "number":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", numberRegexString))
			case "numeric":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", numericRegexString))
			case "uppercase":
				validateStr.WriteString(".refine((val) => val === val.toUpperCase())")
			case "base64":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", base64RegexString))
			case "mongodb":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", mongodbRegexString))
			case "datetime":
				validateStr.WriteString(".datetime()")
			case "hexadecimal":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", hexadecimalRegexString))
			case "json":
				// TODO: Better error messages with this
				// const literalSchema = z.union([z.string(), z.number(), z.boolean(), z.null()]);
				//type Literal = z.infer<typeof literalSchema>;
				//type Json = Literal | { [key: string]: Json } | Json[];
				//const jsonSchema: z.ZodType<Json> = z.lazy(() =>
				//  z.union([literalSchema, z.array(jsonSchema), z.record(jsonSchema)])
				//);
				//
				//jsonSchema.parse(data);

				validateStr.WriteString(".refine((val) => { try { JSON.parse(val); return true } catch { return false } })")
			case "jwt":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", jWTRegexString))
			case "latitude":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", latitudeRegexString))
			case "longitude":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", longitudeRegexString))
			case "uuid":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", uUIDRegexString))
			case "uuid3":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", uUID3RegexString))
			case "uuid3_rfc4122":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", uUID3RFC4122RegexString))
			case "uuid4":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", uUID4RegexString))
			case "uuid4_rfc4122":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", uUID4RFC4122RegexString))
			case "uuid5":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", uUID5RegexString))
			case "uuid5_rfc4122":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", uUID5RFC4122RegexString))
			case "uuid_rfc4122":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", uUIDRFC4122RegexString))
			case "md4":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", md4RegexString))
			case "md5":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", md5RegexString))
			case "sha256":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", sha256RegexString))
			case "sha384":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", sha384RegexString))
			case "sha512":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", sha512RegexString))

			default:
				panic(fmt.Sprintf("unknown validation: %s", part))
			}
		}
	}

	return validateStr.String()
}

func isNullable(field reflect.StructField) bool {
	validateCurrent := getValidateCurrent(field.Tag.Get("validate"))

	// interfaces are currently exported with "any" type, which already includes "null"
	if isInterface(field) || strings.Contains(validateCurrent, "required") {
		return false
	}
	// pointers can be nil, which are mapped to null in JS/TS.
	if field.Type.Kind() == reflect.Ptr {
		// However, if a pointer field is tagged with "omitempty", it usually cannot be exported as "null"
		// since nil is a pointer's empty value.
		if strings.Contains(field.Tag.Get("json"), "omitempty") {
			// Unless it is a pointer to a slice, a map, a pointer, or an interface
			// because values with those types can themselves be nil and will be exported as "null".
			k := field.Type.Elem().Kind()
			return k == reflect.Ptr || k == reflect.Slice || k == reflect.Map
		}
		return true
	}
	// nil slices and maps are exported as null so these types are usually nullable
	if field.Type.Kind() == reflect.Slice || field.Type.Kind() == reflect.Map {
		for _, part := range strings.Split(validateCurrent, ",") {
			part = strings.TrimSpace(part)
			if strings.ContainsRune(part, '=') {
				idx := strings.Index(part, "=")
				if idx == 0 || idx == len(part)-1 {
					panic(fmt.Sprintf("invalid validation: %s", part))
				}

				valName := part[:idx]
				valValue := part[idx+1:]

				if (valName == "len" || valName == "min" || valName == "eq" || valName == "gte") && valValue != "0" ||
					valName == "ne" && valValue == "0" ||
					valName == "gt" {
					return false
				}

				if (valName == "max" || valName == "lte") && valValue == "0" ||
					valName == "lt" && (valValue == "1" || valValue == "0") ||
					(valName == "eq" || valName == "len") && valValue == "0" {
					return true
				}
			}
		}

		// unless there are also optional in which case they are no longer nullable
		return !strings.Contains(field.Tag.Get("json"), "omitempty")
	}
	return false
}

func getValidateCurrent(validate string) string {
	var validateCurrent string

	if strings.HasPrefix(validate, "dive") {

	} else if strings.Contains(validate, ",dive") {
		validateCurrent = strings.Split(validate, ",dive")[0]
	} else {
		validateCurrent = validate
	}

	return validateCurrent
}

// Checks whether the first non-pointer type is an interface
func isInterface(field reflect.StructField) bool {
	t := field.Type
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Kind() == reflect.Interface
}

func isOptional(field reflect.StructField) bool {
	// Non-pointer struct types and direct or indirect interface types should never be optional().
	// Struct fields that are themselves structs ignore the "omitempty" tag because
	// structs do not have an empty value.
	// Interfaces are currently exported with "any" type, which already includes "undefined"
	if field.Type.Kind() == reflect.Struct || isInterface(field) ||
		strings.Contains(getValidateCurrent(field.Tag.Get("validate")), "required") {
		return false
	}
	// Otherwise, omitempty zero-values are omitted and are mapped to undefined in JS/TS.
	return strings.Contains(field.Tag.Get("json"), "omitempty")
}

func indentation(level int) string {
	return strings.Repeat(" ", level*2)
}
