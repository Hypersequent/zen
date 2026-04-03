package zen

import (
	"encoding/json"
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

// Emits legacy Zod v3-compatible schemas instead of the default Zod v4 output.
func WithZodV3() Opt {
	return func(c *Converter) {
		c.zodV3 = true
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

// AddTypeWithName converts a struct type to corresponding zod schema using a custom name
// instead of the struct's type name. Useful for anonymous structs from reflect.StructOf.
func (c *Converter) AddTypeWithName(input interface{}, name string) {
	c.addType(reflect.TypeOf(input), name)
}

// AddType converts a struct type to corresponding zod schema. AddType can be called
// multiple times, followed by Export to get the corresponding zod schemas.
func (c *Converter) AddType(input interface{}) {
	t := reflect.TypeOf(input)
	c.addType(t, typeName(t))
}

func (c *Converter) addType(t reflect.Type, name string) {
	if t.Kind() != reflect.Struct {
		panic("input must be a struct")
	}

	if _, ok := c.outputs[name]; ok {
		return
	}

	data, selfRef := c.convertStructTopLevel(t, name)
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

type stringValidator struct {
	tag string // "email", "ip", "required", "trim", "max", "_custom", etc.
	arg string // "45" for max=45, raw text for _custom
}

type Converter struct {
	prefix      string
	customTypes map[string]CustomFn
	customTags  map[string]CustomFn
	ignoreTags  []string
	zodV3       bool
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

func (c *Converter) convertStructTopLevel(t reflect.Type, name string) (string, bool) {
	output := strings.Builder{}

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

		if field.Anonymous {
			output.WriteString(c.convertEmbeddedFieldSpread(field, indent+1))
		} else {
			output.WriteString(c.convertNamedField(field, indent+1, optional, nullable))
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
	embeddedFields := []string{}
	namedFields := []string{}

	fields := input.NumField()
	for i := 0; i < fields; i++ {
		field := input.Field(i)
		optional := isOptional(field)
		nullable := isNullable(field)

		if field.Anonymous {
			if c.zodV3 {
				line, shouldMerge := c.convertEmbeddedFieldMerge(field, indent+1)
				if shouldMerge {
					merges = append(merges, line)
				} else {
					output.WriteString(line)
				}
			} else {
				embeddedFields = append(embeddedFields, c.convertEmbeddedFieldSpread(field, indent+1))
			}
		} else {
			namedFields = append(namedFields, c.convertNamedField(field, indent+1, optional, nullable))
		}
	}

	// In v4, embedded spreads are written before named fields so that named
	// fields override embedded ones (last key wins in JS object literals).
	// This matches Go's shadowing semantics where the outer struct's field
	// takes precedence over the embedded struct's field.
	if !c.zodV3 {
		for _, line := range embeddedFields {
			output.WriteString(line)
		}
	}
	for _, line := range namedFields {
		output.WriteString(line)
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

	// Collect own (non-anonymous) field names to detect shadowing.
	fields := input.NumField()
	ownFieldNames := map[string]bool{}
	for i := 0; i < fields; i++ {
		f := input.Field(i)
		if !f.Anonymous {
			if name := fieldName(f); name != "-" {
				ownFieldNames[name] = true
			}
		}
	}

	for i := 0; i < fields; i++ {
		field := input.Field(i)
		optional := isOptional(field)
		nullable := isNullable(field)

		line, shouldMerge := c.getTypeField(field, indent+1, optional, nullable)

		if !shouldMerge {
			output.WriteString(line)
		} else {
			// When own fields shadow embedded fields, wrap in Omit<> so the
			// TypeScript intersection doesn't produce conflicting property types.
			embeddedType := field.Type
			if embeddedType.Kind() == reflect.Ptr {
				embeddedType = embeddedType.Elem()
			}
			var shadowedKeys []string
			if embeddedType.Kind() == reflect.Struct {
				for j := 0; j < embeddedType.NumField(); j++ {
					if name := fieldName(embeddedType.Field(j)); name != "-" && ownFieldNames[name] {
						shadowedKeys = append(shadowedKeys, name)
					}
				}
			}
			if len(shadowedKeys) > 0 {
				quoted := make([]string, len(shadowedKeys))
				for k, key := range shadowedKeys {
					quoted[k] = fmt.Sprintf("'%s'", key)
				}
				line = fmt.Sprintf("Omit<%s, %s>", line, strings.Join(quoted, " | "))
			}
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

type convertResult struct {
	text    string
	selfRef bool
}

// ConvertType should be called from custom converter functions.
func (c *Converter) ConvertType(t reflect.Type, validate string, indent int) string {
	return c.convertType(t, validate, indent).text
}

func (c *Converter) convertType(t reflect.Type, validate string, indent int) convertResult {
	if t.Kind() == reflect.Ptr {
		inner := t.Elem()
		validate = strings.TrimPrefix(validate, "omitempty")
		validate = strings.TrimPrefix(validate, ",")
		return c.convertType(inner, validate, indent)
	}

	// Custom types should be handled before maps/slices, as we might have
	// custom types that are maps/slices.
	if custom, ok := c.handleCustomType(t, validate, indent); ok {
		return convertResult{text: custom}
	}

	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		return c.convertSliceAndArray(t, validate, indent)
	}

	if t.Kind() == reflect.Map {
		return convertResult{text: c.convertMap(t, validate, indent)}
	}

	if t.Kind() == reflect.Struct {
		var validateStr strings.Builder
		var refines []string
		var selfRef bool
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
				if c.zodV3 {
					validateStr.WriteString(fmt.Sprintf("z.lazy(() => %s)", schemaName(c.prefix, name)))
				} else {
					selfRef = true
					validateStr.WriteString(schemaName(c.prefix, name))
				}
			} else {
				// throws panic if there is a cycle
				detectCycle(name, c.stack)
				data, sRef := c.convertStructTopLevel(t, name)
				c.addSchema(name, data, sRef)
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

		schema := validateStr.String()
		return convertResult{text: schema, selfRef: selfRef}
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
			return convertResult{text: c.validateString(validate)}
		case "number":
			validateStr = c.validateNumber(validate)
		}
	}

	return convertResult{text: fmt.Sprintf("z.%s()%s", zodType, validateStr)}
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
			return "Date"
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

func (c *Converter) convertNamedField(f reflect.StructField, indent int, optional, nullable bool) string {
	name := fieldName(f)

	// fields named `-` are not exported to JSON so don't export zod types
	if name == "-" {
		return ""
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

	res := c.convertType(f.Type, f.Tag.Get("validate"), indent)

	if res.selfRef && !c.zodV3 {
		return fmt.Sprintf(
			"%sget %s() { return %s%s%s; },\n",
			indentation(indent),
			name,
			res.text,
			optionalCall,
			nullableCall)
	}

	return fmt.Sprintf(
		"%s%s: %s%s%s,\n",
		indentation(indent),
		name,
		res.text,
		optionalCall,
		nullableCall)
}

func (c *Converter) convertEmbeddedFieldMerge(f reflect.StructField, indent int) (string, bool) {
	t := c.convertType(f.Type, f.Tag.Get("validate"), indent).text
	typeName := typeName(f.Type)
	entry, ok := c.outputs[typeName]
	if ok && entry.selfRef {
		// Since we are spreading shape, we won't be able to support any validation tags on the embedded field
		return fmt.Sprintf("%s...%s,\n", indentation(indent), shapeName(c.prefix, typeName)), false
	}

	return fmt.Sprintf(".merge(%s)", t), true
}

func (c *Converter) convertEmbeddedFieldSpread(f reflect.StructField, indent int) string {
	t := c.convertType(f.Type, f.Tag.Get("validate"), indent).text
	typeName := typeName(f.Type)
	entry, ok := c.outputs[typeName]
	if ok && entry.selfRef {
		// Since we are spreading shape, we won't be able to support any validation tags on the embedded field
		return fmt.Sprintf("%s...%s,\n", indentation(indent), shapeName(c.prefix, typeName))
	}

	return fmt.Sprintf("%s...%s.shape,\n", indentation(indent), t)
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

func (c *Converter) convertSliceAndArray(t reflect.Type, validate string, indent int) convertResult {
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

	elemResult := c.convertType(t.Elem(), getValidateAfterDive(validate), indent)
	return convertResult{
		text:    fmt.Sprintf("%s.array()%s", elemResult.text, validateStr.String()),
		selfRef: elemResult.selfRef,
	}
}

func (c *Converter) getTypeSliceAndArray(t reflect.Type, indent int) string {
	return fmt.Sprintf(
		"%s[]",
		c.getType(t.Elem(), indent))
}

func (c *Converter) convertKeyType(t reflect.Type, validate string) string {
	if t.Name() == "Time" {
		// JSON serializes time.Time map keys as RFC3339 strings via TextMarshaler.
		return "z.string()"
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
			return c.validateString(validate)
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

	keySchema := c.convertKeyType(t.Key(), getValidateKeys(validate))
	recordFn := "z.record"
	if !c.zodV3 && isPartialRecordKeySchema(keySchema) {
		recordFn = "z.partialRecord"
	}

	return fmt.Sprintf(`%s(%s, %s)%s`,
		recordFn,
		keySchema,
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

// Tag classification sets for string validators.
var formatTags = map[string]bool{
	"email": true, "url": true, "http_url": true,
	"ipv4": true, "ip4_addr": true, "ipv6": true, "ip6_addr": true,
	"base64": true, "datetime": true, "hexadecimal": true, "jwt": true,
	"uuid": true, "uuid3": true, "uuid3_rfc4122": true,
	"uuid4": true, "uuid4_rfc4122": true,
	"uuid5": true, "uuid5_rfc4122": true,
	"uuid_rfc4122": true,
	"md5":          true, "sha256": true, "sha384": true, "sha512": true,
}

var unionTags = map[string]bool{
	"ip": true, "ip_addr": true,
}

// Tags where generated Zod schemas accepts an empty string
// unless `.min(1)` is added.
var v4AcceptsEmpty = map[string]bool{
	"base64": true, "hexadecimal": true,
}

func (c *Converter) validateString(validate string) string {
	validators := c.parseStringValidators(validate)
	return c.renderStringSchema(validators)
}

var knownStringTags = map[string]bool{
	"required": true, "email": true, "url": true, "http_url": true,
	"ipv4": true, "ip4_addr": true, "ipv6": true, "ip6_addr": true,
	"ip": true, "ip_addr": true,
	"url_encoded": true, "alpha": true, "alphanum": true,
	"alphanumunicode": true, "alphaunicode": true, "ascii": true,
	"lowercase": true, "number": true, "numeric": true, "uppercase": true,
	"base64": true, "mongodb": true, "datetime": true, "hexadecimal": true,
	"json": true, "jwt": true, "latitude": true, "longitude": true,
	"uuid": true, "uuid3": true, "uuid3_rfc4122": true,
	"uuid4": true, "uuid4_rfc4122": true,
	"uuid5": true, "uuid5_rfc4122": true, "uuid_rfc4122": true,
	"md4": true, "md5": true, "sha256": true, "sha384": true, "sha512": true,
	"contains": true, "endswith": true, "startswith": true,
	"eq": true, "ne": true, "len": true, "min": true, "max": true,
	"gt": true, "gte": true, "lt": true, "lte": true,
}

func (c *Converter) parseStringValidators(validate string) []stringValidator {
	var validators []stringValidator
	parts := strings.Split(validate, ",")

	for _, rawPart := range parts {
		valName, valValue, skip := c.parseValidationTagPart(rawPart)
		if skip {
			continue
		}

		if h, ok := c.customTags[valName]; ok {
			v := h(c, reflect.TypeOf(""), valValue, 0)
			validators = append(validators, stringValidator{tag: "_custom", arg: v})
			continue
		}

		switch {
		case valName == "omitempty":
			// skip
		case valName == "oneof" && valValue != "":
			vals := splitParamsRegex.FindAllString(rawPart[len("oneof="):], -1)
			for i := 0; i < len(vals); i++ {
				vals[i] = escapeJSString(strings.ReplaceAll(vals[i], "'", ""))
			}
			enumText := fmt.Sprintf("z.enum([\"%s\"] as const)", strings.Join(vals, "\", \""))
			validators = append(validators, stringValidator{tag: "oneof", arg: enumText})
		case valName == "boolean":
			validators = append(validators, stringValidator{tag: "boolean", arg: "z.enum(['true', 'false'])"})
		case knownStringTags[valName]:
			validators = append(validators, stringValidator{tag: valName, arg: valValue})
		default:
			panic(fmt.Sprintf("unknown validation: %s", rawPart))
		}
	}

	return validators
}

func (c *Converter) renderStringSchema(validators []stringValidator) string {
	// Phase 1: Classify validators
	hasFormat := false
	hasUnion := false
	hasRequired := false
	hasEnum := false
	formatIdx := -1
	formatCount := 0

	for i, v := range validators {
		if formatTags[v.tag] {
			hasFormat = true
			formatCount++
			if formatIdx == -1 {
				formatIdx = i
			}
		}
		if unionTags[v.tag] {
			hasUnion = true
		}
		if v.tag == "required" {
			hasRequired = true
		}
		if v.tag == "oneof" || v.tag == "boolean" {
			hasEnum = true
		}
	}

	// Phase 2: Validate combinations
	if hasFormat && hasUnion {
		panic("cannot combine format validator with union validator (e.g. email + ip)")
	}
	if formatCount > 1 {
		panic("cannot combine multiple format validators (e.g. email + url)")
	}

	// Phase 3: Handle enum — return early
	if hasEnum {
		base := ""
		var chain strings.Builder
		for _, v := range validators {
			if v.tag == "oneof" || v.tag == "boolean" {
				base = v.arg
				break
			}
		}
		for _, v := range validators {
			if v.tag == "oneof" || v.tag == "boolean" {
				continue
			}
			var rendered string
			if c.zodV3 {
				rendered = c.renderV3Chain(v)
			} else {
				rendered = renderChain(v)
			}
			if strings.HasPrefix(rendered, ".refine") {
				chain.WriteString(rendered)
			}
		}
		return base + chain.String()
	}

	// Phase 4: Render v3
	if c.zodV3 {
		// Skip required when a format or union is present — format validators
		// already reject empty strings in both v3 and v4.
		skipRequired := hasFormat || hasUnion
		var chain strings.Builder
		for _, v := range validators {
			if v.tag == "required" && skipRequired {
				continue
			}
			chain.WriteString(c.renderV3Chain(v))
		}
		return "z.string()" + chain.String()
	}

	// Phase 5: Render v4

	// Case 1: Union (ip/ip_addr)
	if hasUnion {
		var armChain strings.Builder
		for _, v := range validators {
			if v.tag == "required" || unionTags[v.tag] {
				continue
			}
			armChain.WriteString(renderChain(v))
		}
		ac := armChain.String()
		return fmt.Sprintf("z.union([z.ipv4()%s, z.ipv6()%s])", ac, ac)
	}

	// Case 2: Format present
	if hasFormat {
		// Check if anything (non-required, non-omitempty) precedes the format
		hasTransformBefore := false
		for i := 0; i < formatIdx; i++ {
			v := validators[i]
			if v.tag != "required" && v.tag != "omitempty" {
				hasTransformBefore = true
				break
			}
		}

		// Determine if required should be kept (base64/hex accept empty in v4)
		keepRequired := hasRequired && v4AcceptsEmpty[validators[formatIdx].tag]

		if hasTransformBefore {
			// Fall back to z.string() + chains (format becomes a chain method via v3 form)
			var chain strings.Builder
			for _, v := range validators {
				if v.tag == "required" && !keepRequired {
					continue
				}
				if formatTags[v.tag] {
					chain.WriteString(c.renderV3Chain(v))
				} else {
					chain.WriteString(renderChain(v))
				}
			}
			return "z.string()" + chain.String()
		}

		// Format as base
		base := c.renderV4FormatBase(validators[formatIdx])
		var chain strings.Builder
		if keepRequired {
			chain.WriteString(".min(1)")
		}
		for i := formatIdx + 1; i < len(validators); i++ {
			v := validators[i]
			if v.tag == "required" && !keepRequired {
				continue
			}
			chain.WriteString(renderChain(v))
		}
		return base + chain.String()
	}

	// Case 3: No format/union — plain string
	var chain strings.Builder
	for _, v := range validators {
		chain.WriteString(renderChain(v))
	}
	return "z.string()" + chain.String()
}

// escapeJSString escapes a string so it can be safely interpolated into a
// JavaScript double-quoted string literal. Uses json.Marshal for complete
// handling of quotes, backslashes, newlines, and control characters, then
// strips the outer quotes.
func escapeJSString(s string) string {
	b, _ := json.Marshal(s)
	// json.Marshal wraps in quotes: "foo" → strip them
	return string(b[1 : len(b)-1])
}

// requireIntArg validates that arg is a valid integer for the given tag name.
// Returns the parsed value. Panics if arg is not a valid integer.
func requireIntArg(tag, arg string) int {
	val, err := strconv.Atoi(arg)
	if err != nil {
		panic(fmt.Sprintf("%s= requires an integer argument, got: %s", tag, arg))
	}
	return val
}

// regexChainMap maps validator tags to their regex pattern strings.
// Used by renderChain and renderV3Chain to generate .regex() calls.
var regexChainMap = map[string]string{
	"url_encoded": uRLEncodedRegexString,
	"alpha":       alphaRegexString,
	"alphanum":    alphaNumericRegexString,
	"ascii":       aSCIIRegexString,
	"number":      numberRegexString,
	"numeric":     numericRegexString,
	"mongodb":     mongodbRegexString,
	"latitude":    latitudeRegexString,
	"longitude":   longitudeRegexString,
	"md4":         md4RegexString,
}

func renderRegex(pattern string) string {
	return fmt.Sprintf(".regex(/%s/)", pattern)
}

// unicodeRegexChainMap is like regexChainMap but for patterns needing the /u flag.
var unicodeRegexChainMap = map[string]string{
	"alphanumunicode": alphaUnicodeNumericRegexString,
	"alphaunicode":    alphaUnicodeRegexString,
}

func renderUnicodeRegex(pattern string) string {
	return fmt.Sprintf(".regex(/%s/u)", pattern)
}

// renderChain is used by both v3 and v4 rendering
func renderChain(v stringValidator) string {
	// Regex-based validators
	if pattern, ok := regexChainMap[v.tag]; ok {
		return renderRegex(pattern)
	}
	if pattern, ok := unicodeRegexChainMap[v.tag]; ok {
		return renderUnicodeRegex(pattern)
	}

	switch v.tag {
	case "required":
		return ".min(1)"
	case "contains":
		return fmt.Sprintf(`.includes("%s")`, escapeJSString(v.arg))
	case "startswith":
		return fmt.Sprintf(`.startsWith("%s")`, escapeJSString(v.arg))
	case "endswith":
		return fmt.Sprintf(`.endsWith("%s")`, escapeJSString(v.arg))
	case "eq":
		return fmt.Sprintf(`.refine((val) => val === "%s")`, escapeJSString(v.arg))
	case "ne":
		return fmt.Sprintf(`.refine((val) => val !== "%s")`, escapeJSString(v.arg))
	case "len":
		requireIntArg("len", v.arg)
		return fmt.Sprintf(".refine((val) => [...val].length === %s, 'String must contain %s character(s)')", v.arg, v.arg)
	case "min":
		requireIntArg("min", v.arg)
		return fmt.Sprintf(".refine((val) => [...val].length >= %s, 'String must contain at least %s character(s)')", v.arg, v.arg)
	case "max":
		requireIntArg("max", v.arg)
		return fmt.Sprintf(".refine((val) => [...val].length <= %s, 'String must contain at most %s character(s)')", v.arg, v.arg)
	case "gt":
		val := requireIntArg("gt", v.arg)
		return fmt.Sprintf(".refine((val) => [...val].length > %d, 'String must contain at least %d character(s)')", val, val+1)
	case "gte":
		requireIntArg("gte", v.arg)
		return fmt.Sprintf(".refine((val) => [...val].length >= %s, 'String must contain at least %s character(s)')", v.arg, v.arg)
	case "lt":
		val := requireIntArg("lt", v.arg)
		return fmt.Sprintf(".refine((val) => [...val].length < %d, 'String must contain at most %d character(s)')", val, val-1)
	case "lte":
		requireIntArg("lte", v.arg)
		return fmt.Sprintf(".refine((val) => [...val].length <= %s, 'String must contain at most %s character(s)')", v.arg, v.arg)
	case "lowercase":
		return ".refine((val) => val === val.toLowerCase())"
	case "uppercase":
		return ".refine((val) => val === val.toUpperCase())"
	case "json":
		return ".refine((val) => { try { JSON.parse(val); return true } catch { return false } })"
	case "_custom":
		return v.arg
	default:
		return ""
	}
}

// v3FormatRegexMap maps format validator tags to their v3 regex pattern strings.
// These tags have v4 top-level builders but fall back to regex in v3.
var v3FormatRegexMap = map[string]string{
	"base64":        base64RegexString,
	"hexadecimal":   hexadecimalRegexString,
	"jwt":           jWTRegexString,
	"uuid":          uUIDRegexString,
	"uuid3":         uUID3RegexString,
	"uuid3_rfc4122": uUID3RFC4122RegexString,
	"uuid4":         uUID4RegexString,
	"uuid4_rfc4122": uUID4RFC4122RegexString,
	"uuid5":         uUID5RegexString,
	"uuid5_rfc4122": uUID5RFC4122RegexString,
	"uuid_rfc4122":  uUIDRFC4122RegexString,
	"md5":           md5RegexString,
	"sha256":        sha256RegexString,
	"sha384":        sha384RegexString,
	"sha512":        sha512RegexString,
}

func (c *Converter) renderV3Chain(v stringValidator) string {
	if s := renderChain(v); s != "" {
		return s
	}

	// v3 format regex fallbacks (these have v4 top-level builders but use regex in v3)
	if pattern, ok := v3FormatRegexMap[v.tag]; ok {
		return renderRegex(pattern)
	}

	switch v.tag {
	case "email":
		return ".email()"
	case "url":
		return ".url()"
	case "ip", "ip_addr":
		return ".ip()"
	case "ipv4", "ip4_addr":
		return `.ip({ version: "v4" })`
	case "ipv6", "ip6_addr":
		return `.ip({ version: "v6" })`
	case "http_url":
		return ".url()"
	case "datetime":
		return ".datetime()"
	default:
		return ""
	}
}

func (c *Converter) renderV4FormatBase(v stringValidator) string {
	switch v.tag {
	case "email":
		return "z.email()"
	case "url":
		return "z.url()"
	case "http_url":
		return "z.httpUrl()"
	case "ipv4", "ip4_addr":
		return "z.ipv4()"
	case "ipv6", "ip6_addr":
		return "z.ipv6()"
	case "base64":
		return "z.base64()"
	case "datetime":
		return "z.iso.datetime()"
	case "hexadecimal":
		return "z.hex()"
	case "jwt":
		return "z.jwt()"
	case "uuid":
		return "z.uuid()"
	case "uuid3", "uuid3_rfc4122":
		return `z.uuid({ version: "v3" })`
	case "uuid4", "uuid4_rfc4122":
		return `z.uuid({ version: "v4" })`
	case "uuid5", "uuid5_rfc4122":
		return `z.uuid({ version: "v5" })`
	case "uuid_rfc4122":
		return "z.uuid()"
	case "md5":
		return `z.hash("md5")`
	case "sha256":
		return `z.hash("sha256")`
	case "sha384":
		return `z.hash("sha384")`
	case "sha512":
		return `z.hash("sha512")`
	default:
		panic(fmt.Sprintf("renderV4FormatBase: unhandled format tag %q", v.tag))
	}
}

func isPartialRecordKeySchema(schema string) bool {
	schema = strings.TrimSpace(schema)
	return strings.HasPrefix(schema, "z.enum(") || strings.HasPrefix(schema, "z.literal(")
}

func (c *Converter) parseValidationTagPart(part string) (string, string, bool) {
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

	return valName, valValue, false
}

func (c *Converter) preprocessValidationTagPart(part string, refines *[]string, validateStr *strings.Builder) (string, string, bool) {
	valName, valValue, done := c.parseValidationTagPart(part)
	if done {
		return "", "", true
	}

	if h, ok := c.customTags[valName]; ok {
		v := h(c, reflect.TypeOf(""), valValue, 0)
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
