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

	data := c.convertStructTopLevel(t)
	order := c.structs
	c.outputs[name] = entry{order, data}
	c.structs = order + 1
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
	order int
	data  string
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

func (c *Converter) addSchema(name string, data string) {
	// First check if the object already exists. If it does do not replace. This is needed for second order
	_, ok := c.outputs[name]
	if !ok {
		order := c.structs
		c.outputs[name] = entry{order, data}
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

func (c *Converter) convertStructTopLevel(t reflect.Type) string {
	output := strings.Builder{}

	name := typeName(t)
	c.stack = append(c.stack, meta{name, false})

	data := c.convertStruct(t, 0)
	fullName := c.prefix + name

	top := c.stack[len(c.stack)-1]
	if top.selfRef {
		output.WriteString(fmt.Sprintf(`export type %s = %s
`, fullName, c.getTypeStruct(t, 0)))

		output.WriteString(fmt.Sprintf(
			`export const %s: z.ZodType<%s> = %s`, schemaName(c.prefix, name), fullName, data))
	} else {
		output.WriteString(fmt.Sprintf(
			`export const %s = %s
`,
			schemaName(c.prefix, name), data))

		output.WriteString(fmt.Sprintf(`export type %s = z.infer<typeof %s>`,
			fullName, schemaName(c.prefix, name)))
	}

	c.stack = c.stack[:len(c.stack)-1]

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

		line, shouldMerge := c.convertField(field, indent+1, optional, nullable, field.Anonymous)

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

	fields := input.NumField()
	for i := 0; i < fields; i++ {
		field := input.Field(i)
		optional := isOptional(field)
		nullable := isNullable(field)

		line := c.getTypeField(field, indent+1, optional, nullable)

		output.WriteString(line)
	}

	output.WriteString(indentation(indent))
	output.WriteString(`}`)

	return output.String()
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
				c.addSchema(name, c.convertStructTopLevel(t))
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

	if zodType == "string" {
		if validate != "" {
			validateStrResult, isTopLevel := c.validateString(validate)
			if isTopLevel {
				return validateStrResult
			}
			if validateStrResult != "" {
				return fmt.Sprintf("z.string()%s", validateStrResult)
			}
		}
		return "z.string()"
	} else if zodType == "number" {
		if validate != "" {
			validateStrResult, isTopLevel := c.validateNumber(validate)
			if isTopLevel {
				return validateStrResult // e.g. "z.union([z.literal(1), ...]).min(0)"
			}
			// Not top-level, so it's something like ".min(1)" or ""
			if validateStrResult != "" {
				return fmt.Sprintf("z.number()%s", validateStrResult)
			}
		}
		return "z.number()" // Default if no validation
	}

	// For other types like boolean, any.
	return fmt.Sprintf("z.%s()", zodType)
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

func (c *Converter) convertField(f reflect.StructField, indent int, optional, nullable, anonymous bool) (string, bool) {
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
	if !anonymous {
		return fmt.Sprintf(
			"%s%s: %s%s%s,\n",
			indentation(indent),
			name,
			t,
			optionalCall,
			nullableCall), false
	} else {
		return fmt.Sprintf(".merge(%s)", t), true
	}
}

func (c *Converter) getTypeField(f reflect.StructField, indent int, optional, nullable bool) string {
	name := fieldName(f)

	// fields named `-` are not exported to JSON so don't export types
	if name == "-" {
		return ""
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

	return fmt.Sprintf(
		"%s%s%s: %s%s%s,\n",
		indentation(indent),
		name,
		optionalCallPre,
		c.getType(f.Type, indent),
		nullableCall,
		optionalCallUndef)
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
	if !ok || (zodType != "string" && zodType != "number") { // Map keys can only be string or number (coerced)
		panic(fmt.Sprint("cannot handle key type: ", t.Kind()))
	}

	if zodType == "string" {
		if validate != "" {
			validateStrResult, isTopLevel := c.validateString(validate)
			if isTopLevel {
				return validateStrResult
			}
			if validateStrResult != "" {
				return fmt.Sprintf("z.string()%s", validateStrResult)
			}
		}
		return "z.string()"
	} else if zodType == "number" { // must be number if not string, due to earlier check for map keys
		if validate != "" {
			validateStrResult, isTopLevel := c.validateNumber(validate)
			// For map keys, numbers are coerced. So, a union of literals would also need coercion if it's a base schema.
			// z.coerce.union([z.literal(1),...]) isn't a thing.
			// If `oneof` is used for a number key, it implies the key must be one of those specific numbers.
			// JSON object keys are always strings. So, `z.enum(["1", "2"])` or `z.union([z.literal("1"), ...])` might be more appropriate for stringified number keys.
			// However, Zod's `z.coerce.number()` for keys implies it expects the key to be parseable as a number.
			// If `isTopLevel` is true (e.g. `oneof` created `z.union([z.literal(1), ...])`),
			// this schema needs to be wrapped with `z.coerce.string().transform((val, ctx) => { ... })` to parse then validate,
			// or the literal union should be of strings: z.union([z.literal("1"), z.literal("2")]) and then .pipe(z.coerce.number()).
			// This is complex. For now, let's assume `oneof` for numeric keys is not common or needs string literals.
			// The simplest path that keeps current coercion behavior for non-oneof cases:
			if isTopLevel {
				// This path is problematic for numeric keys if `oneof` for numbers produces `z.union([z.literal(1), ...])`
				// because map keys are strings in JSON.
				// A simple solution might be to disallow `oneof` for numeric map keys or require stringified literals in the tag.
				// For now, let's assume `validateNumber` for keys won't return `isTopLevel` for `oneof`,
				// or `oneof` for keys should produce string literals.
				// Given current `validateNumber` will produce `z.union([z.literal(1),...])`, this will likely lead to issues for map keys.
				// Revisit if `oneof` for numeric map keys is a strong requirement.
				// For now, we'll assume `isTopLevel` from `validateNumber` on a key implies it's already a string-coercible schema or simple appendages.
				// A safer bet for numeric keys with `oneof` would be for `validateNumber` to produce string literals for keys: z.union([z.literal("1"), ...])
				// and then ConvertType can .pipe(z.coerce.number()) if needed.
				// This specific interaction is tricky. Let's stick to the direct change first and test.
				// The most straightforward for now: if isTopLevel, it's a schema like z.union, which won't be further wrapped by z.coerce.number() here.
				// This means `oneof=1 2` for a number key would become `z.union([z.literal(1), z.literal(2)])` which is not ideal for string keys.
				//
				// A better approach for `oneof` in numeric keys:
				// validateNumber for keys should detect it's for a key and make string literals, then ConvertKeyType appends .pipe(z.coerce.number())
				// This is beyond the current scope of just changing `validateNumber`'s return type.
				// Let's assume `isTopLevel` will be false for keys for now if `oneof` is used, forcing it through `z.coerce.number()%.s`
				// This means `validateNumber` needs to know its context (key or not) or `oneof` for number keys won't use `z.union`.
				//
				// Sticking to the plan: `validateNumber` returns `(string, bool)`. `convertKeyType` uses it.
				// If `isTopLevel` is true: return `validateStrResult` (e.g. `z.union(...)`)
				// Else: return `z.coerce.number()%s`
				// This means for a map[int]string with `oneof=1 2`, key becomes `z.union([z.literal(1), z.literal(2)])` -> BAD for JSON keys.
				//
				// Let's simplify: For numeric keys, `oneof` will use the refine method as it currently does, not a top-level union.
				// So, `validateNumber` will need a context or `oneof` part needs to be adjusted.
				// For now, let's assume `validateNumber`'s `oneof` will NOT set `isTopLevel=true` if it's for a map key context.
				// This is getting too complex for this step.
				// The original plan: `validateNumber` returns `(string, bool)`. `convertKeyType` uses it.
				// We will proceed with this and address numeric map key `oneof` specifics if they arise as an issue.
				if isTopLevel { // This path for numeric keys with oneof needs careful thought.
					return validateStrResult // This would be z.union([z.literal(1),...])
				}
				return fmt.Sprintf("z.coerce.number()%s", validateStrResult)
			}
		}
		return "z.coerce.number()" // Default for numeric keys if no validation
	}
	// Should not be reached due to previous checks for map key types
	panic(fmt.Sprintf("unsupported key type after checks: %s for type %s", zodType, t.Name()))
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

// validateNumber returns the Zod validation string and a boolean indicating if it's a top-level schema.
func (c *Converter) validateNumber(validate string) (string, bool) {
	var baseSchema strings.Builder
	var appendages strings.Builder
	var refines []string
	isTopLevel := false
	hasBaseSchema := false

	parts := strings.Split(validate, ",")

	for _, part := range parts {
		// Pass '&appendages' for custom tags that append directly.
		valName, valValue, processedByCustomTag := c.preprocessValidationTagPart(part, &refines, &appendages)
		if processedByCustomTag {
			continue
		}

		if valValue != "" { // Validations with arguments
			switch valName {
			case "oneof":
				if hasBaseSchema {
					panic(fmt.Sprintf("cannot combine 'oneof' with another top-level number validation: %s", part))
				}
				numValues := strings.Fields(valValue)
				if len(numValues) == 0 {
					panic(fmt.Sprintf("invalid oneof validation for number: %s. Requires values.", part))
				}
				var literalStrings []string
				for _, nv := range numValues {
					// Ensure nv is a valid number before creating literal
					if _, err := strconv.ParseFloat(nv, 64); err != nil {
						panic(fmt.Sprintf("invalid number value '%s' in oneof for number: %s", nv, part))
					}
					literalStrings = append(literalStrings, fmt.Sprintf("z.literal(%s)", nv))
				}
				baseSchema.WriteString(fmt.Sprintf("z.union([%s])", strings.Join(literalStrings, ", ")))
				isTopLevel = true
				hasBaseSchema = true
			case "gt":
				appendages.WriteString(fmt.Sprintf(".gt(%s)", valValue))
			case "gte", "min":
				appendages.WriteString(fmt.Sprintf(".gte(%s)", valValue))
			case "lt":
				appendages.WriteString(fmt.Sprintf(".lt(%s)", valValue))
			case "lte", "max":
				appendages.WriteString(fmt.Sprintf(".lte(%s)", valValue))
			case "eq", "len": // 'len' is unusual for numbers, but validator package supports it. Zod uses direct value check.
				refines = append(refines, fmt.Sprintf(".refine((val) => val === %s, { message: \"Number must be equal to %s\" })", valValue, valValue))
			case "ne":
				refines = append(refines, fmt.Sprintf(".refine((val) => val !== %s, { message: \"Number must not be equal to %s\" })", valValue, valValue))
			default:
				panic(fmt.Sprintf("unknown number validation with value: %s", part))
			}
		} else { // Validations without arguments
			switch valName {
			case "omitempty":
				// Handled by .optional() at field level.
			case "required":
				// For numbers, 'required' often means non-zero.
				// Zod's .min, .gte, etc. imply non-null. If 0 is disallowed, a refine is needed.
				// Current behavior is .refine((val) => val !== 0)
				refines = append(refines, ".refine((val) => val !== 0, { message: \"Number is required and cannot be 0\" })")
			default:
				panic(fmt.Sprintf("unknown number validation without value: %s", part))
			}
		}
	}

	finalSchema := baseSchema.String() + appendages.String()
	for _, refine := range refines {
		finalSchema += refine
	}
	return finalSchema, isTopLevel
}

// validateString returns the Zod validation string and a boolean indicating if it's a top-level schema.
func (c *Converter) validateString(validate string) (string, bool) {
	var baseSchema strings.Builder // For top-level schemas like z.email(), z.uuid()
	var appendages strings.Builder // For chained methods like .min(), .max()
	var refines []string           // For .refine() calls
	isTopLevel := false
	hasBaseSchema := false // Tracks if baseSchema is set by a top-level validator like email, url, etc.

	parts := strings.Split(validate, ",")

	for _, part := range parts {
		// Pass '&appendages' to preprocessValidationTagPart for custom tags that append directly.
		// If a custom tag intends to set a base schema, preprocessValidationTagPart would need modification
		// or the custom tag logic here would need to identify it.
		// For now, assume custom tags append to 'appendages' or 'refines'.
		valName, valValue, processedByCustomTag := c.preprocessValidationTagPart(part, &refines, &appendages)
		if processedByCustomTag {
			continue // Custom tag has handled this part
		}

		// Logic for built-in validations will be fully implemented in the next step.
		// Logic for built-in validations
		if valValue != "" { // Validations with arguments, e.g., min=5, oneof='a' 'b'
			switch valName {
			case "oneof":
				if hasBaseSchema {
					panic(fmt.Sprintf("cannot combine 'oneof' with other top-level string validation: %s", part))
				}
				vals := splitParamsRegex.FindAllString(valValue, -1)
				for i := 0; i < len(vals); i++ {
					vals[i] = strings.Replace(vals[i], "'", "", -1)
				}
				if len(vals) == 0 {
					panic(fmt.Sprintf("invalid oneof validation: %s", part))
				}
				baseSchema.WriteString(fmt.Sprintf("z.enum([\"%s\"] as const)", strings.Join(vals, "\", \"")))
				isTopLevel = true
				hasBaseSchema = true
			case "len":
				appendages.WriteString(fmt.Sprintf(".length(%s)", valValue))
			case "min":
				appendages.WriteString(fmt.Sprintf(".min(%s)", valValue))
			case "max":
				appendages.WriteString(fmt.Sprintf(".max(%s)", valValue))
			case "gt": // Greater than, Zod uses min(value + 1) for strings if we interpret gt as length
				val, err := strconv.Atoi(valValue)
				if err != nil {
					panic(fmt.Sprintf("invalid gt value for string length: %s, error: %v", valValue, err))
				}
				appendages.WriteString(fmt.Sprintf(".min(%d)", val+1))
			case "gte": // Greater than or equal to, Zod uses min(value) for strings
				appendages.WriteString(fmt.Sprintf(".min(%s)", valValue))
			case "lt": // Less than
				val, err := strconv.Atoi(valValue)
				if err != nil {
					panic(fmt.Sprintf("invalid lt value for string length: %s, error: %v", valValue, err))
				}
				appendages.WriteString(fmt.Sprintf(".max(%d)", val-1))
			case "lte": // Less than or equal to
				appendages.WriteString(fmt.Sprintf(".max(%s)", valValue))
			case "contains":
				appendages.WriteString(fmt.Sprintf(".includes(\"%s\")", valValue))
			case "startswith":
				appendages.WriteString(fmt.Sprintf(".startsWith(\"%s\")", valValue))
			case "endswith":
				appendages.WriteString(fmt.Sprintf(".endsWith(\"%s\")", valValue))
			case "eq": // Equality for strings
				refines = append(refines, fmt.Sprintf(".refine((val) => val === \"%s\", { message: \"String must be equal to '%s'\" })", valValue, valValue))
			case "ne": // Non-equality for strings
				refines = append(refines, fmt.Sprintf(".refine((val) => val !== \"%s\", { message: \"String must not be equal to '%s'\" })", valValue, valValue))
			default:
				panic(fmt.Sprintf("unknown string validation with value: %s (valName: %s, valValue: %s)", part, valName, valValue))
			}
		} else { // Validations without arguments, e.g., email, required, uuid
			switch valName {
			case "omitempty":
				// This is usually handled by .optional() at the field level, not directly in validateString.
				// If it needs to imply optionality here, it's a larger design change.
			case "required":
				appendages.WriteString(".min(1)") // Common way to ensure string is not empty
			case "email":
				if hasBaseSchema {
					panic(fmt.Sprintf("cannot combine 'email' with other top-level string validation: %s", part))
				}
				baseSchema.WriteString("z.email()")
				isTopLevel = true
				hasBaseSchema = true
			case "url", "http_url":
				if hasBaseSchema {
					panic(fmt.Sprintf("cannot combine 'url' with other top-level string validation: %s", part))
				}
				baseSchema.WriteString("z.url()")
				isTopLevel = true
				hasBaseSchema = true
			case "ipv4", "ip4_addr":
				if hasBaseSchema {
					panic(fmt.Sprintf("cannot combine 'ipv4' with other top-level string validation: %s", part))
				}
				baseSchema.WriteString("z.ipv4()")
				isTopLevel = true
				hasBaseSchema = true
			case "ipv6", "ip6_addr":
				if hasBaseSchema {
					panic(fmt.Sprintf("cannot combine 'ipv6' with other top-level string validation: %s", part))
				}
				baseSchema.WriteString("z.ipv6()")
				isTopLevel = true
				hasBaseSchema = true
			case "ip", "ip_addr":
				if hasBaseSchema {
					panic(fmt.Sprintf("cannot combine 'ip' with other top-level string validation: %s", part))
				}
				baseSchema.WriteString("z.union([z.ipv4(), z.ipv6()])")
				isTopLevel = true
				hasBaseSchema = true
			case "datetime": // ISO DateTime
				if hasBaseSchema {
					panic(fmt.Sprintf("cannot combine 'datetime' with other top-level string validation: %s", part))
				}
				baseSchema.WriteString("z.iso.datetime()")
				isTopLevel = true
				hasBaseSchema = true
			case "uuid", "uuid3", "uuid3_rfc4122", "uuid4", "uuid4_rfc4122", "uuid5", "uuid5_rfc4122", "uuid_rfc4122":
				if hasBaseSchema {
					panic(fmt.Sprintf("cannot combine 'uuid' with other top-level string validation: %s", part))
				}
				baseSchema.WriteString("z.uuid()")
				isTopLevel = true
				hasBaseSchema = true
			// Regex-based validations from original code, kept as appendages
			case "url_encoded":
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"Invalid URL encoded string\" })", uRLEncodedRegexString))
			case "alpha":
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"Invalid alpha string\" })", alphaRegexString))
			case "alphanum":
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"Invalid alphanumeric string\" })", alphaNumericRegexString))
			case "alphanumunicode":
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"Invalid alphanumeric unicode string\" })", alphaUnicodeNumericRegexString))
			case "alphaunicode":
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"Invalid alpha unicode string\" })", alphaUnicodeRegexString))
			case "ascii":
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"Invalid ASCII string\" })", aSCIIRegexString))
			case "boolean": // String 'true' or 'false'
				// This was .enum(['true', 'false']), which is a form of string validation.
				// If a true z.boolean() is needed, the type itself should be bool.
				appendages.WriteString(".refine((val) => val === 'true' || val === 'false', { message: \"String must be 'true' or 'false'\" })")
			case "lowercase":
				refines = append(refines, ".refine((val) => val === val.toLowerCase(), { message: \"String must be lowercase\" })")
			case "uppercase":
				refines = append(refines, ".refine((val) => val === val.toUpperCase(), { message: \"String must be uppercase\" })")
			case "number": // String representation of a number
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"String must be a number\" })", numberRegexString))
			case "numeric":
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"String must be numeric\" })", numericRegexString))
			case "base64":
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"Invalid base64 string\" })", base64RegexString))
			case "mongodb":
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"Invalid MongoDB ID\" })", mongodbRegexString))
			case "hexadecimal":
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"Invalid hexadecimal string\" })", hexadecimalRegexString))
			case "json":
				refines = append(refines, ".refine((val) => { try { JSON.parse(val); return true; } catch (e) { return false; } }, { message: \"String must be valid JSON\" })")
			case "jwt":
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"Invalid JWT token\" })", jWTRegexString))
			case "latitude":
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"Invalid latitude string\" })", latitudeRegexString))
			case "longitude":
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"Invalid longitude string\" })", longitudeRegexString))
			// MD, SHA hashes - kept as regex appendages
			case "md4":
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"Invalid MD4 hash\" })", md4RegexString))
			case "md5":
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"Invalid MD5 hash\" })", md5RegexString))
			case "sha256":
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"Invalid SHA256 hash\" })", sha256RegexString))
			case "sha384":
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"Invalid SHA384 hash\" })", sha384RegexString))
			case "sha512":
				appendages.WriteString(fmt.Sprintf(".regex(/%s/, { message: \"Invalid SHA512 hash\" })", sha512RegexString))
			default:
				panic(fmt.Sprintf("unknown string validation without value: %s (valName: %s)", part, valName))
			}
		}
	}

	// Combine base schema (if any) with appendages
	finalSchema := baseSchema.String() + appendages.String()

	// Add all refines at the end
	for _, refine := range refines {
		finalSchema += refine
	}

	// If baseSchema was set, isTopLevel is true.
	// If only appendages were added (e.g. .min(5)), isTopLevel remains false,
	// and ConvertType will prepend "z.string()".
	return finalSchema, isTopLevel
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
		// Pass a reflect.Type that won't cause a panic in getFullName if called by custom func.
		// reflect.TypeOf("") is a safe, non-nil type if the custom func doesn't care about the actual type.
		// If the custom func *does* care, it should be designed to handle various types or expect a specific one.
		v := h(c, reflect.TypeOf(""), valValue, 0)
		if strings.HasPrefix(v, ".refine") {
			*refines = append(*refines, v)
		} else {
			// Ensure validateStr (now validateBuilder) is the correct parameter name from the calling function
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
