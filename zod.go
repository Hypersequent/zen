package zen

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Opt represents a converter option used to modify its behavior.
type Opt func(*Converter)

// Adds prefix to the generated schema and type names.
func WithPrefix(prefix string) Opt {
	return func(c *Converter) {
		c.prefix = prefix
	}
}

// Adds custom handler/converters for types. The map should be keyed on
// the fully qualified type name (excluding generic type arguments), ie.
// package.typename.
func WithCustomTypes(custom map[string]CustomFn) Opt {
	return func(c *Converter) {
		for k, v := range custom {
			c.customTypes[k] = v
		}
	}
}

// Adds custom handler/converts for tags. The functions should return
// strings like `.regex(/[a-z0-9_]+/)` or `.refine((val) => val !== 0)`
// which can be appended to the generated schema.
func WithCustomTags(custom map[string]CustomFn) Opt {
	return func(c *Converter) {
		for k, v := range custom {
			c.customTags[k] = v
		}
	}
}

// Adds tags which should be ignored. Any unrecognized tag (which is also
// not ignored) results in panic.
func WithIgnoreTags(ignores ...string) Opt {
	return func(c *Converter) {
		c.ignoreTags = append(c.ignoreTags, ignores...)
	}
}

// NewConverterWithOpts initializes and returns a new converter instance.
func NewConverterWithOpts(opts ...Opt) *Converter {
	c := &Converter{
		prefix:      "",
		customTypes: make(map[string]CustomFn),
		customTags:  make(map[string]CustomFn),
		ignoreTags:  []string{},
		outputs:     make(map[string]entry),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Deprecated: NewConverter is deprecated. Use NewConverterWithOpts(WithCustomTypes(customTypes)) instead.
// Example:
//
//	converter := NewConverterWithOpts(WithCustomTypes(customTypes))
func NewConverter(customTypes map[string]CustomFn) Converter {
	return *NewConverterWithOpts(WithCustomTypes(customTypes))
}

// AddType converts a struct type to corresponding zod schema. AddType can be called
// multiple times, followed by Export to get the corresponding zod schemas.
func (c *Converter) AddType(input interface{}) {
	t := reflect.TypeOf(input)

	if t.Kind() != reflect.Struct {
		panic("input must be a struct")
	}

	name := typeName(t)
	if _, ok := c.outputs[name]; ok {
		return
	}

	data, selfRef := c.convertStructTopLevel(t)
	c.addSchema(name, data, selfRef)
}

// Convert returns zod schema corresponding to a struct type. Its a shorthand for
// call to AddType followed by Export. So calling Convert after other calls to
// AddType/Convert/ConvertSlice, returns schemas from those previous calls as well.
// Calling AddType followed by Export might be more appropriate in such scenarios.
func (c *Converter) Convert(input interface{}) string {
	c.AddType(input)

	return c.Export()
}

// ConvertSlice returns zod schemas corresponding to multiple struct types passed
// in the argument. Similar to Convert, calling ConvertSlice after other calls to
// AddType/Convert/ConvertSlice, returns schemas from those previous calls as well.
// Calling AddType followed by Export might be more appropriate in such scenarios.
func (c *Converter) ConvertSlice(inputs []interface{}) string {
	for _, input := range inputs {
		c.AddType(input)
	}

	return c.Export()
}

// StructToZodSchema returns zod schema corresponding to a struct type.
func StructToZodSchema(input interface{}, opts ...Opt) string {
	return NewConverterWithOpts(opts...).Convert(input)
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
	order   int
	data    string
	selfRef bool
}

type byOrder []entry

func (a byOrder) Len() int           { return len(a) }
func (a byOrder) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byOrder) Less(i, j int) bool { return a[i].order < a[j].order }

type CustomFn func(c *Converter, t reflect.Type, validate string, indent int) string

type meta struct {
	name    string
	selfRef bool
}

type Converter struct {
	prefix      string
	customTypes map[string]CustomFn
	customTags  map[string]CustomFn
	ignoreTags  []string
	structs     int
	outputs     map[string]entry
	stack       []meta
}

func (c *Converter) addSchema(name string, data string, selfRef bool) {
	// First check if the object already exists. If it does do not replace. This is needed for second order
	_, ok := c.outputs[name]
	if !ok {
		order := c.structs
		c.outputs[name] = entry{order, data, selfRef}
		c.structs = order + 1
	}
}

// Export returns the zod schemas corresponding to all types that have been
// converted so far.
func (c *Converter) Export() string {
	output := strings.Builder{}
	var sorted []entry
	for _, ent := range c.outputs {
		sorted = append(sorted, ent)
	}

	sort.Sort(byOrder(sorted))

	for _, ent := range sorted {
		output.WriteString(ent.data)
		output.WriteString("\n\n")
	}

	return output.String()
}

func schemaName(prefix, name string) string {
	return fmt.Sprintf("%s%sSchema", prefix, name)
}

func shapeName(prefix, name string) string {
	return schemaName(prefix, name) + "Shape"
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
		return getTypeNameWithGenerics(t.Name())
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

func (c *Converter) convertStructTopLevel(t reflect.Type) (string, bool) {
	output := strings.Builder{}

	name := typeName(t)
	c.stack = append(c.stack, meta{name, false})

	data := c.convertStruct(t, 0)
	fullName := c.prefix + name

	top := c.stack[len(c.stack)-1]
	if top.selfRef {
		shapeName := shapeName(c.prefix, name)

		output.WriteString(fmt.Sprintf(`export type %s = %s
`, fullName, c.getTypeStruct(t, 0)))

		output.WriteString(fmt.Sprintf(`const %s = %s
`, shapeName, c.getStructShape(t, 0)))

		output.WriteString(fmt.Sprintf(
			`export const %s: z.ZodType<%s> = z.object(%s)`, schemaName(c.prefix, name), fullName, shapeName))
	} else {
		output.WriteString(fmt.Sprintf(
			`export const %s = %s
`,
			schemaName(c.prefix, name), data))

		output.WriteString(fmt.Sprintf(`export type %s = z.infer<typeof %s>`,
			fullName, schemaName(c.prefix, name)))
	}

	c.stack = c.stack[:len(c.stack)-1]

	return output.String(), top.selfRef
}

func (c *Converter) getStructShape(input reflect.Type, indent int) string {
	output := strings.Builder{}

	output.WriteString(`{
`)

	fields := input.NumField()
	for i := 0; i < fields; i++ {
		field := input.Field(i)
		optional := isOptional(field)
		nullable := isNullable(field)

		line, shouldMerge := c.convertField(field, indent+1, optional, nullable)

		if !shouldMerge {
			output.WriteString(line)
		} else {
			output.WriteString(fmt.Sprintf("%s...%s.shape,\n", indentation(indent+1), schemaName(c.prefix, typeName(field.Type))))
		}
	}

	output.WriteString(indentation(indent))
	output.WriteString(`}`)

	return output.String()
}

func (c *Converter) convertStruct(input reflect.Type, indent int) string {
	output := strings.Builder{}

	output.WriteString(`z.object({
`)

	merges := []string{}

	fields := input.NumField()
	for i := 0; i < fields; i++ {
		field := input.Field(i)
		optional := isOptional(field)
		nullable := isNullable(field)

		line, shouldMerge := c.convertField(field, indent+1, optional, nullable)

		if !shouldMerge {
			output.WriteString(line)
		} else {
			merges = append(merges, line)
		}
	}

	output.WriteString(indentation(indent))
	output.WriteString(`})`)
	if len(merges) > 0 {
		for _, merge := range merges {
			output.WriteString(merge)
		}
	}

	return output.String()
}

func (c *Converter) getTypeStruct(input reflect.Type, indent int) string {
	output := strings.Builder{}

	output.WriteString(`{
`)

	merges := []string{}

	fields := input.NumField()
	for i := 0; i < fields; i++ {
		field := input.Field(i)
		optional := isOptional(field)
		nullable := isNullable(field)

		line, shouldMerge := c.getTypeField(field, indent+1, optional, nullable)

		if !shouldMerge {
			output.WriteString(line)
		} else {
			merges = append(merges, line)
		}
	}

	output.WriteString(indentation(indent))
	output.WriteString(`}`)

	if len(merges) == 0 {
		return output.String()
	}

	newOutput := strings.Builder{}
	for _, merge := range merges {
		newOutput.WriteString(fmt.Sprintf("%s & ", merge))
	}
	newOutput.WriteString(output.String())
	return newOutput.String()
}

var matchGenericTypeName = regexp.MustCompile(`(.+)\[(.+)]`)

// Checking if a reflected type is a generic isn't supported as far as I can see.
// So this simple check looks for a `[` character in the type name: `T1[T2]`.
func isGeneric(t reflect.Type) bool {
	return strings.Contains(t.Name(), "[")
}

// Gets the full type name (package+type), stripping out generic type arguments.
func getFullName(t reflect.Type) string {
	var typename string

	if isGeneric(t) {
		m := matchGenericTypeName.FindAllStringSubmatch(t.Name(), 1)[0]
		typename = m[1]
	} else {
		typename = t.Name()
	}

	return fmt.Sprintf("%s.%s", t.PkgPath(), typename)
}

func (c *Converter) handleCustomType(t reflect.Type, validate string, indent int) (string, bool) {
	fullName := getFullName(t)

	custom, ok := c.customTypes[fullName]
	if ok {
		return custom(c, t, validate, indent), true
	}

	return "", false
}

// ConvertType should be called from custom converter functions.
func (c *Converter) ConvertType(t reflect.Type, validate string, indent int) string {
	if t.Kind() == reflect.Ptr {
		inner := t.Elem()
		validate = strings.TrimPrefix(validate, "omitempty")
		validate = strings.TrimPrefix(validate, ",")
		return c.ConvertType(inner, validate, indent)
	}

	// Custom types should be handled before maps/slices, as we might have
	// custom types that are maps/slices.
	if custom, ok := c.handleCustomType(t, validate, indent); ok {
		return custom
	}

	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		return c.convertSliceAndArray(t, validate, indent)
	}

	if t.Kind() == reflect.Map {
		return c.convertMap(t, validate, indent)
	}

	if t.Kind() == reflect.Struct {
		var validateStr strings.Builder
		var refines []string
		name := typeName(t)
		parts := strings.Split(validate, ",")

		if name == "" {
			// Handle fields with non-defined types - these are inline.
			validateStr.WriteString(c.convertStruct(t, indent))
		} else if name == "Time" {
			// timestamps are to be coerced to date by zod. JSON.parse only serializes to string
			validateStr.WriteString("z.coerce.date()")
		} else {
			if c.stack[len(c.stack)-1].name == name {
				c.stack[len(c.stack)-1].selfRef = true
				validateStr.WriteString(fmt.Sprintf("z.lazy(() => %s)", schemaName(c.prefix, name)))
			} else {
				// throws panic if there is a cycle
				detectCycle(name, c.stack)
				data, selfRef := c.convertStructTopLevel(t)
				c.addSchema(name, data, selfRef)
				validateStr.WriteString(schemaName(c.prefix, name))
			}
		}

		for _, part := range parts {
			valName, _, done := c.preprocessValidationTagPart(part, &refines, &validateStr)
			if done {
				continue
			}

			switch valName {
			case "required":
				if name == "Time" {
					// We compare with both the zero value from go and the zero value that zod coerces to
					refines = append(refines, ".refine((val) => val.getTime() !== new Date('0001-01-01T00:00:00Z').getTime() && val.getTime() !== new Date(0).getTime(), 'Invalid date')")
				}
			default:
				panic(fmt.Sprintf("unknown validation: %s", part))
			}
		}

		for _, refine := range refines {
			validateStr.WriteString(refine)
		}

		return validateStr.String()
	}

	// boolean, number, string, any
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

func (c *Converter) getType(t reflect.Type, indent int) string {
	if t.Kind() == reflect.Ptr {
		inner := t.Elem()
		return c.getType(inner, indent)
	}

	// TODO: handle types for custom types

	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		return c.getTypeSliceAndArray(t, indent)
	}

	if t.Kind() == reflect.Map {
		return c.getTypeMap(t, indent)
	}

	if t.Kind() == reflect.Struct {
		name := typeName(t)

		if t.Name() == "" {
			// Handle fields with non-defined types - these are inline.
			return c.getTypeStruct(t, indent)
		} else if t.Name() == "Time" {
			return "date"
		} else {
			return c.prefix + name
		}
	}

	zodType, ok := typeMapping[t.Kind()]
	if !ok {
		panic(fmt.Sprint("cannot handle: ", t.Kind()))
	}
	return zodType
}

func (c *Converter) convertField(f reflect.StructField, indent int, optional, nullable bool) (string, bool) {
	name := fieldName(f)

	// fields named `-` are not exported to JSON so don't export zod types
	if name == "-" {
		return "", false
	}

	// because nullability is processed before custom types, this makes sure
	// the custom type has control over nullability.
	fullName := getFullName(f.Type)
	_, isCustom := c.customTypes[fullName]

	optionalCall := ""
	if optional {
		optionalCall = ".optional()"
	}
	nullableCall := ""
	if nullable && !isCustom {
		nullableCall = ".nullable()"
	}

	t := c.ConvertType(f.Type, f.Tag.Get("validate"), indent)
	if !f.Anonymous {
		return fmt.Sprintf(
			"%s%s: %s%s%s,\n",
			indentation(indent),
			name,
			t,
			optionalCall,
			nullableCall), false
	} else {
		typeName := typeName(f.Type)
		entry, ok := c.outputs[typeName]
		if ok && entry.selfRef {
			// Since we are spreading shape, we won't be able to support any validation tags on the embedded field
			return fmt.Sprintf("%s...%s,\n", indentation(indent), shapeName(c.prefix, typeName)), false
		}

		return fmt.Sprintf(".merge(%s)", t), true
	}
}

func (c *Converter) getTypeField(f reflect.StructField, indent int, optional, nullable bool) (string, bool) {
	name := fieldName(f)

	// fields named `-` are not exported to JSON so don't export types
	if name == "-" {
		return "", false
	}

	// because nullability is processed before custom types, this makes sure
	// the custom type has control over nullability.
	fullName := getFullName(f.Type)
	_, isCustom := c.customTypes[fullName]

	optionalCallPre := ""
	optionalCallUndef := ""
	if optional {
		optionalCallPre = "?"
		optionalCallUndef = " | undefined"
	}
	nullableCall := ""
	if nullable && !isCustom {
		nullableCall = " | null"
	}

	if !f.Anonymous {
		return fmt.Sprintf(
			"%s%s%s: %s%s%s,\n",
			indentation(indent),
			name,
			optionalCallPre,
			c.getType(f.Type, indent),
			nullableCall,
			optionalCallUndef), false
	}

	return typeName(f.Type), true
}

func (c *Converter) convertSliceAndArray(t reflect.Type, validate string, indent int) string {
	var validateStr strings.Builder
	var refines []string
	validateCurrent := getValidateCurrent(validate)
	parts := strings.Split(validateCurrent, ",")
	isArray := t.Kind() == reflect.Array

forParts:
	for _, part := range parts {
		valName, valValue, done := c.preprocessValidationTagPart(part, &refines, &validateStr)
		if done {
			continue
		}

		if isArray {
			panic(fmt.Sprintf("unknown validation: %s", part))
		} else {
			if valValue != "" {
				switch valName {
				case "min":
					validateStr.WriteString(fmt.Sprintf(".min(%s)", valValue))
				case "max":
					validateStr.WriteString(fmt.Sprintf(".max(%s)", valValue))
				case "len":
					validateStr.WriteString(fmt.Sprintf(".length(%s)", valValue))
				case "eq":
					validateStr.WriteString(fmt.Sprintf(".length(%s)", valValue))
				case "ne":
					refines = append(refines, fmt.Sprintf(".refine((val) => val.length !== %s)", valValue))
				case "gt":
					val, err := strconv.Atoi(valValue)
					if err != nil || val < 0 {
						panic(fmt.Sprintf("invalid gt value: %s", valValue))
					}
					validateStr.WriteString(fmt.Sprintf(".min(%d)", val+1))
				case "gte":
					validateStr.WriteString(fmt.Sprintf(".min(%s)", valValue))
				case "lt":
					val, err := strconv.Atoi(valValue)
					if err != nil || val <= 0 {
						panic(fmt.Sprintf("invalid lt value: %s", valValue))
					}
					validateStr.WriteString(fmt.Sprintf(".max(%d)", val-1))
				case "lte":
					validateStr.WriteString(fmt.Sprintf(".max(%s)", valValue))

				default:
					panic(fmt.Sprintf("unknown validation: %s", part))
				}
			} else {
				switch valName {
				case "omitempty":
				case "required":
				case "dive":
					break forParts

				default:
					panic(fmt.Sprintf("unknown validation: %s", part))
				}
			}
		}
	}

	if isArray {
		validateStr.WriteString(fmt.Sprintf(".length(%d)", t.Len()))
	}

	for _, refine := range refines {
		validateStr.WriteString(refine)
	}

	return fmt.Sprintf(
		"%s.array()%s",
		c.ConvertType(t.Elem(), getValidateAfterDive(validate), indent), validateStr.String())
}

func (c *Converter) getTypeSliceAndArray(t reflect.Type, indent int) string {
	return fmt.Sprintf(
		"%s[]",
		c.getType(t.Elem(), indent))
}

func (c *Converter) convertKeyType(t reflect.Type, validate string) string {
	if t.Name() == "Time" {
		return "z.coerce.date()"
	}

	// boolean, number, string, any
	zodType, ok := typeMapping[t.Kind()]
	if !ok || (zodType != "string" && zodType != "number") {
		panic(fmt.Sprint("cannot handle key type: ", t.Kind()))
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

	if zodType == "string" {
		return fmt.Sprintf("z.%s()%s", zodType, validateStr)
	}

	// https://pkg.go.dev/encoding/json#Marshal
	// Map values encode as JSON objects. The map's key type must either be a string, an integer type, or implement encoding.TextMarshaler.
	return fmt.Sprintf("z.coerce.%s()%s", zodType, validateStr)
}

func (c *Converter) convertMap(t reflect.Type, validate string, indent int) string {
	var validateStr strings.Builder
	var refines []string
	parts := strings.Split(validate, ",")

forParts:
	for _, part := range parts {
		valName, valValue, done := c.preprocessValidationTagPart(part, &refines, &validateStr)
		if done {
			continue
		}

		if valValue != "" {
			switch valName {
			case "min":
				refines = append(refines, fmt.Sprintf(".refine((val) => Object.keys(val).length >= %s, 'Map too small')", valValue))
			case "max":
				refines = append(refines, fmt.Sprintf(".refine((val) => Object.keys(val).length <= %s, 'Map too large')", valValue))
			case "len":
				refines = append(refines, fmt.Sprintf(".refine((val) => Object.keys(val).length === %s, 'Map wrong size')", valValue))
			case "eq":
				refines = append(refines, fmt.Sprintf(".refine((val) => Object.keys(val).length === %s, 'Map wrong size')", valValue))
			case "ne":
				refines = append(refines, fmt.Sprintf(".refine((val) => Object.keys(val).length !== %s, 'Map wrong size')", valValue))
			case "gt":
				refines = append(refines, fmt.Sprintf(".refine((val) => Object.keys(val).length > %s, 'Map too small')", valValue))
			case "gte":
				refines = append(refines, fmt.Sprintf(".refine((val) => Object.keys(val).length >= %s, 'Map too small')", valValue))
			case "lt":
				refines = append(refines, fmt.Sprintf(".refine((val) => Object.keys(val).length < %s, 'Map too large')", valValue))
			case "lte":
				refines = append(refines, fmt.Sprintf(".refine((val) => Object.keys(val).length <= %s, 'Map too large')", valValue))

			default:
				panic(fmt.Sprintf("unknown validation: %s", part))
			}
		} else {
			switch valName {
			case "omitempty":
			case "required":
			case "dive":
				break forParts

			default:
				panic(fmt.Sprintf("unknown validation: %s", part))
			}
		}
	}

	for _, refine := range refines {
		validateStr.WriteString(refine)
	}

	return fmt.Sprintf(`z.record(%s, %s)%s`,
		c.convertKeyType(t.Key(), getValidateKeys(validate)),
		c.ConvertType(t.Elem(), getValidateValues(validate), indent),
		validateStr.String())
}

func (c *Converter) getTypeMap(t reflect.Type, indent int) string {
	return fmt.Sprintf(`Record<%s, %s>`,
		c.getType(t.Key(), indent),
		c.getType(t.Elem(), indent))
}

// Select part of validate string after dive, if it exists.
func getValidateAfterDive(validate string) string {
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

// These are to be used together directly after the dive tag and tells the validator that anything between
// 'keys' and 'endkeys' applies to the keys of a map and not the values; think of it like the 'dive' tag,
// but for map keys instead of values. Multidimensional nesting is also supported, each level you wish to
// validate will require another 'keys' and 'endkeys' tag. These tags are only valid for maps.
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

func (c *Converter) checkIsIgnored(part string) bool {
	for _, ignore := range c.ignoreTags {
		if part == ignore {
			return true
		}
	}
	return false
}

// not implementing omitempty for numbers and strings
// could support unusual cases like `validate:"omitempty,min=3,max=5"`
func (c *Converter) validateNumber(validate string) string {
	var validateStr strings.Builder
	var refines []string
	parts := strings.Split(validate, ",")

	for _, part := range parts {
		valName, valValue, done := c.preprocessValidationTagPart(part, &refines, &validateStr)
		if done {
			continue
		}

		if valValue != "" {
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
				refines = append(refines, fmt.Sprintf(".refine((val) => val === %s)", valValue))
			case "ne":
				refines = append(refines, fmt.Sprintf(".refine((val) => val !== %s)", valValue))
			case "oneof":
				vals := strings.Fields(valValue)
				if len(vals) == 0 {
					panic(fmt.Sprintf("invalid oneof validation: %s", part))
				}
				refines = append(refines, fmt.Sprintf(".refine((val) => [%s].includes(val))", strings.Join(vals, ", ")))

			default:
				panic(fmt.Sprintf("unknown validation: %s", part))
			}
		} else {
			switch part {
			case "omitempty":
			case "required":
				refines = append(refines, ".refine((val) => val !== 0)")

			default:
				panic(fmt.Sprintf("unknown validation: %s", part))
			}
		}
	}

	for _, refine := range refines {
		validateStr.WriteString(refine)
	}

	return validateStr.String()
}

func (c *Converter) validateString(validate string) string {
	var validateStr strings.Builder
	var refines []string
	parts := strings.Split(validate, ",")

	for _, part := range parts {
		valName, valValue, done := c.preprocessValidationTagPart(part, &refines, &validateStr)
		if done {
			continue
		}

		if valValue != "" {
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
				refines = append(refines, fmt.Sprintf(".refine((val) => [...val].length === %s, 'String must contain %s character(s)')", valValue, valValue))
			case "min":
				refines = append(refines, fmt.Sprintf(".refine((val) => [...val].length >= %s, 'String must contain at least %s character(s)')", valValue, valValue))
			case "max":
				refines = append(refines, fmt.Sprintf(".refine((val) => [...val].length <= %s, 'String must contain at most %s character(s)')", valValue, valValue))
			case "gt":
				val, err := strconv.Atoi(valValue)
				if err != nil {
					panic("gt= must be followed by a number")
				}
				refines = append(refines, fmt.Sprintf(".refine((val) => [...val].length > %d, 'String must contain at least %d character(s)')", val, val+1))
			case "gte":
				refines = append(refines, fmt.Sprintf(".refine((val) => [...val].length >= %s, 'String must contain at least %s character(s)')", valValue, valValue))
			case "lt":
				val, err := strconv.Atoi(valValue)
				if err != nil {
					panic("lt= must be followed by a number")
				}
				refines = append(refines, fmt.Sprintf(".refine((val) => [...val].length < %d, 'String must contain at most %d character(s)')", val, val-1))
			case "lte":
				refines = append(refines, fmt.Sprintf(".refine((val) => [...val].length <= %s, 'String must contain at most %s character(s)')", valValue, valValue))
			case "contains":
				validateStr.WriteString(fmt.Sprintf(".includes(\"%s\")", valValue))
			case "endswith":
				validateStr.WriteString(fmt.Sprintf(".endsWith(\"%s\")", valValue))
			case "startswith":
				validateStr.WriteString(fmt.Sprintf(".startsWith(\"%s\")", valValue))
			case "eq":
				refines = append(refines, fmt.Sprintf(".refine((val) => val === \"%s\")", valValue))
			case "ne":
				refines = append(refines, fmt.Sprintf(".refine((val) => val !== \"%s\")", valValue))

			default:
				panic(fmt.Sprintf("unknown validation: %s", part))
			}
		} else {
			switch part {
			case "omitempty":
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
				refines = append(refines, ".refine((val) => val === val.toLowerCase())")
			case "number":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", numberRegexString))
			case "numeric":
				validateStr.WriteString(fmt.Sprintf(".regex(/%s/)", numericRegexString))
			case "uppercase":
				refines = append(refines, ".refine((val) => val === val.toUpperCase())")
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

				refines = append(refines, ".refine((val) => { try { JSON.parse(val); return true } catch { return false } })")
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

	for _, refine := range refines {
		validateStr.WriteString(refine)
	}

	return validateStr.String()
}

func (c *Converter) preprocessValidationTagPart(part string, refines *[]string, validateStr *strings.Builder) (string, string, bool) {
	part = strings.TrimSpace(part)
	if part == "" {
		return "", "", true
	}

	idx := strings.Index(part, "=")
	if idx == 0 || idx == len(part)-1 {
		panic(fmt.Sprintf("invalid validation: %s", part))
	}

	var valName string
	var valValue string
	if idx == -1 {
		valName = part
	} else {
		valName = part[:idx]
		valValue = part[idx+1:]
	}

	if c.checkIsIgnored(valName) {
		return "", "", true
	}

	if h, ok := c.customTags[valName]; ok {
		v := h(c, reflect.TypeOf(0), valValue, 0)
		if strings.HasPrefix(v, ".refine") {
			*refines = append(*refines, v)
		} else {
			(*validateStr).WriteString(v)
		}
		return "", "", true
	}

	return valName, valValue, false
}

func isNullable(field reflect.StructField) bool {
	validateCurrent := getValidateCurrent(field.Tag.Get("validate"))

	// interfaces are currently exported with "any" type, which already includes "null"
	if isInterface(field) || strings.Contains(validateCurrent, "required") {
		return false
	}

	// If some comparison is present min=1 or max=2 or len=4 etc. then go-validator requires the value
	// to be non-nil unless omitempty is also present
	if strings.Contains(validateCurrent, "=") && !strings.Contains(validateCurrent, "omitempty") {
		return false
	}

	jsonTag := field.Tag.Get("json")

	// pointers can be nil, which are mapped to null in JS/TS.
	if field.Type.Kind() == reflect.Ptr {
		// However, if a pointer field is tagged with "omitempty"/"omitzero", it usually cannot be exported
		// as "null" since nil is a pointer's empty/zero value.
		if strings.Contains(jsonTag, "omitempty") || strings.Contains(jsonTag, "omitzero") {
			// Unless it is a pointer to a slice, a map, a pointer, or an interface
			// because values with those types can themselves be nil and will be exported as "null".
			k := field.Type.Elem().Kind()
			return k == reflect.Ptr || k == reflect.Slice || k == reflect.Map
		}

		return true
	}

	// nil slices and maps are exported as null so these types are usually nullable
	if field.Type.Kind() == reflect.Slice || field.Type.Kind() == reflect.Map {
		// unless there are also optional in which case they are no longer nullable
		return !strings.Contains(jsonTag, "omitempty") && !strings.Contains(jsonTag, "omitzero")
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
	validateCurrent := getValidateCurrent(field.Tag.Get("validate"))

	// Non-pointer struct types and direct or indirect interface types should never be optional().
	// Struct fields that are themselves structs ignore the "omitempty" tag because
	// structs do not have an empty value.
	// Interfaces are currently exported with "any" type, which already includes "undefined"
	if field.Type.Kind() == reflect.Struct || isInterface(field) ||
		strings.Contains(validateCurrent, "required") {
		return false
	}

	// If some comparison is present min=1 or max=2 or len=4 etc. then go-validator requires the value
	// to be non-nil unless omitempty is also present
	if strings.Contains(validateCurrent, "=") && !strings.Contains(validateCurrent, "omitempty") {
		return false
	}

	// Otherwise, omitempty/omitzero zero-values are omitted and are mapped to undefined in JS/TS.
	jsonTag := field.Tag.Get("json")
	return strings.Contains(jsonTag, "omitempty") || strings.Contains(jsonTag, "omitzero")
}

func indentation(level int) string {
	return strings.Repeat(" ", level*2)
}

func detectCycle(name string, stack []meta) {
	var found bool
	var cycle strings.Builder
	for _, m := range stack {
		cycle.WriteString(m.name)
		if m.name == name {
			found = true
			break
		}
		cycle.WriteString(" -> ")
	}

	if found {
		panic(fmt.Sprintf("circular dependency detected: %s", cycle.String()))
	}
}

func getTypeNameWithGenerics(name string) string {
	typeArgsIdx := strings.Index(name, "[")
	if typeArgsIdx == -1 {
		return name
	}

	var sb strings.Builder
	sb.WriteString(name[:typeArgsIdx])

	typeArgs := strings.Split(name[typeArgsIdx+1:len(name)-1], ",")
	for _, arg := range typeArgs {
		sb.WriteString(strings.ToTitle(arg[:1])) // Capitalize first letter
		sb.WriteString(arg[1:])
	}

	return sb.String()
}
