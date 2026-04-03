/**
 * Runtime test cases for golden file schemas.
 *
 * Each case references a golden file, a schema name exported from it,
 * test input, whether parsing should succeed, and the expected output.
 *
 * Golden files are copied into the Docker test environment by docker-typecheck.sh.
 * The import paths here are relative to the test runner's location in the container.
 */

export interface TestCase {
	/** Description of what this test verifies */
	name: string;
	/** Path to golden file relative to testdata/ */
	golden: string;
	/** Name of the exported schema to test (e.g. "UserSchema") */
	schema: string;
	/** Input to pass to schema.safeParse() */
	input: unknown;
	/** Whether parsing should succeed */
	success: boolean;
	/** Expected output after parsing (only checked if success=true, if not provided expected output will be the same as the input) */
	output?: unknown;
}

export const cases: TestCase[] = [
	// ---------------------------------------------------------------------------
	// SIMPLE STRUCTS
	// ---------------------------------------------------------------------------

	// --- TestStructSimple ---
	{
		name: "simple struct: parses valid object",
		golden: "TestStructSimple.golden",
		schema: "UserSchema",
		input: { Name: "John", Age: 30, Height: 5.9 },
		success: true,
	},
	{
		name: "simple struct: rejects type error (string for Age)",
		golden: "TestStructSimple.golden",
		schema: "UserSchema",
		input: { Name: "John", Age: "thirty", Height: 5.9 },
		success: false,
	},

	// --- TestStructSimplePrefix ---
	{
		name: "simple struct prefix: parses valid BotUser",
		golden: "TestStructSimplePrefix.golden",
		schema: "BotUserSchema",
		input: { Name: "Bot", Age: 1, Height: 3.0 },
		success: true,
	},

	// --- TestStructSimpleWithOmittedField ---
	{
		name: "omitted field: parses valid object (omitted field not in schema)",
		golden: "TestStructSimpleWithOmittedField.golden",
		schema: "UserSchema",
		input: { Name: "John", Age: 30, Height: 5.9 },
		success: true,
	},

	// --- TestStringOptional ---
	{
		name: "string optional: parses with Nickname present",
		golden: "TestStringOptional.golden",
		schema: "UserSchema",
		input: { Name: "John", Nickname: "Johnny" },
		success: true,
	},
	{
		name: "string optional: parses without Nickname (undefined)",
		golden: "TestStringOptional.golden",
		schema: "UserSchema",
		input: { Name: "John" },
		success: true,
	},

	// --- TestStringNullable ---
	{
		name: "string nullable: parses with null Nickname",
		golden: "TestStringNullable.golden",
		schema: "UserSchema",
		input: { Name: "John", Nickname: null },
		success: true,
	},

	// --- TestStringOptionalNotNullable ---
	{
		name: "string optional not nullable: parses with undefined Nickname",
		golden: "TestStringOptionalNotNullable.golden",
		schema: "UserSchema",
		input: { Name: "John" },
		success: true,
	},

	// --- TestStringOptionalNullable ---
	{
		name: "string optional nullable: parses with null Nickname",
		golden: "TestStringOptionalNullable.golden",
		schema: "UserSchema",
		input: { Name: "John", Nickname: null },
		success: true,
	},
	{
		name: "string optional nullable: parses with undefined Nickname",
		golden: "TestStringOptionalNullable.golden",
		schema: "UserSchema",
		input: { Name: "John" },
		success: true,
	},

	// --- TestDuration ---
	{
		name: "duration: parses valid number",
		golden: "TestDuration.golden",
		schema: "UserSchema",
		input: { HowLong: 3600 },
		success: true,
	},

	// --- TestStructTime ---
	{
		name: "time: parses ISO string to Date",
		golden: "TestStructTime.golden",
		schema: "UserSchema",
		input: { Name: "John", When: "2021-01-01T00:00:00Z" },
		success: true,
		output: { Name: "John", When: new Date("2021-01-01T00:00:00Z") },
	},
	{
		name: "time: parses unix timestamp to Date",
		golden: "TestStructTime.golden",
		schema: "UserSchema",
		input: { Name: "John", When: 1609459200000 },
		success: true,
		output: { Name: "John", When: new Date("2021-01-01T00:00:00Z") },
	},
	{
		name: "time: coerces null to epoch Date",
		golden: "TestStructTime.golden",
		schema: "UserSchema",
		input: { Name: "John", When: null },
		success: true,
		output: { Name: "John", When: new Date(0) },
	},
	{
		name: "time: parses zero date string",
		golden: "TestStructTime.golden",
		schema: "UserSchema",
		input: { Name: "John", When: "0001-01-01T00:00:00Z" },
		success: true,
		output: { Name: "John", When: new Date("0001-01-01T00:00:00Z") },
	},
	{
		name: "time: rejects empty string",
		golden: "TestStructTime.golden",
		schema: "UserSchema",
		input: { Name: "John", When: "" },
		success: false,
	},

	// --- TestTimeWithRequired ---
	{
		name: "required time: parses valid date",
		golden: "TestTimeWithRequired.golden",
		schema: "UserSchema",
		input: { When: "2021-01-01T00:00:00Z" },
		success: true,
		output: { When: new Date("2021-01-01T00:00:00Z") },
	},
	{
		name: "required time: parses unix timestamp",
		golden: "TestTimeWithRequired.golden",
		schema: "UserSchema",
		input: { When: 1609459200000 },
		success: true,
		output: { When: new Date("2021-01-01T00:00:00Z") },
	},
	{
		name: "required time: rejects null (zero date)",
		golden: "TestTimeWithRequired.golden",
		schema: "UserSchema",
		input: { When: null },
		success: false,
	},
	{
		name: "required time: rejects zero date string",
		golden: "TestTimeWithRequired.golden",
		schema: "UserSchema",
		input: { When: "0001-01-01T00:00:00Z" },
		success: false,
	},
	{
		name: "required time: rejects empty string",
		golden: "TestTimeWithRequired.golden",
		schema: "UserSchema",
		input: { When: "" },
		success: false,
	},


	// ---------------------------------------------------------------------------
	// ARRAYS
	// ---------------------------------------------------------------------------

	// --- TestStringArray ---
	{
		name: "string array: parses valid array",
		golden: "TestStringArray.golden",
		schema: "UserSchema",
		input: { Tags: ["a", "b", "c"] },
		success: true,
	},
	{
		name: "string array: parses null",
		golden: "TestStringArray.golden",
		schema: "UserSchema",
		input: { Tags: null },
		success: true,
	},

	// --- TestStringArrayNullable ---
	{
		name: "string array nullable: parses valid array",
		golden: "TestStringArrayNullable.golden",
		schema: "UserSchema",
		input: { Name: "John", Tags: ["x"] },
		success: true,
	},

	// --- TestStringNestedArray ---
	{
		name: "nested array: parses valid nested array (inner length 2)",
		golden: "TestStringNestedArray.golden",
		schema: "UserSchema",
		input: {
			TagPairs: [
				["a", "b"],
				["c", "d"],
			],
		},
		success: true,
	},

	// --- TestConvertArray/single ---
	{
		name: "fixed array: parses array of length 10",
		golden: "TestConvertArray/single.golden",
		schema: "ArraySchema",
		input: { Arr: ["a", "b", "c", "d", "e", "f", "g", "h", "i", "j"] },
		success: true,
	},
	{
		name: "fixed array: rejects wrong count",
		golden: "TestConvertArray/single.golden",
		schema: "ArraySchema",
		input: { Arr: ["a", "b"] },
		success: false,
	},

	// --- TestConvertArray/multi ---
	{
		name: "multi-dim array: parses valid 3D array",
		golden: "TestConvertArray/multi.golden",
		schema: "MultiArraySchema",
		input: {
			Arr: Array.from({ length: 10 }, () =>
				Array.from({ length: 20 }, () => Array.from({ length: 30 }, () => "x")),
			),
		},
		success: true,
	},

	// --- TestConvertSlice ---
	{
		name: "convert slice: ZipSchema with valid Foo",
		golden: "TestConvertSlice.golden",
		schema: "ZipSchema",
		input: { Zap: { Bar: "a", Baz: "b", Quz: "c" } },
		success: true,
	},
	{
		name: "convert slice: ZipSchema with null",
		golden: "TestConvertSlice.golden",
		schema: "ZipSchema",
		input: { Zap: null },
		success: true,
	},
	{
		name: "convert slice: WhimSchema with valid Foo",
		golden: "TestConvertSlice.golden",
		schema: "WhimSchema",
		input: { Wham: { Bar: "a", Baz: "b", Quz: "c" } },
		success: true,
	},

	// --- TestStructSlice ---
	{
		name: "struct slice: parses valid array",
		golden: "TestStructSlice.golden",
		schema: "UserSchema",
		input: { Favourites: [{ Name: "Alice" }, { Name: "Bob" }] },
		success: true,
	},
	{
		name: "struct slice: parses null",
		golden: "TestStructSlice.golden",
		schema: "UserSchema",
		input: { Favourites: null },
		success: true,
	},

	// --- TestStructSliceOptional ---
	{
		name: "struct slice optional: parses valid array",
		golden: "TestStructSliceOptional.golden",
		schema: "UserSchema",
		input: { Favourites: [{ Name: "Alice" }] },
		success: true,
	},
	{
		name: "struct slice optional: parses undefined",
		golden: "TestStructSliceOptional.golden",
		schema: "UserSchema",
		input: {},
		success: true,
	},

	// --- TestStructSliceOptionalNullable ---
	{
		name: "struct slice optional nullable: parses valid array",
		golden: "TestStructSliceOptionalNullable.golden",
		schema: "UserSchema",
		input: { Favourites: [{ Name: "Alice" }] },
		success: true,
	},
	{
		name: "struct slice optional nullable: parses null",
		golden: "TestStructSliceOptionalNullable.golden",
		schema: "UserSchema",
		input: { Favourites: null },
		success: true,
	},
	{
		name: "struct slice optional nullable: parses undefined",
		golden: "TestStructSliceOptionalNullable.golden",
		schema: "UserSchema",
		input: {},
		success: true,
	},

	// --- TestSliceFields ---
	{
		name: "slice fields: parses valid object with all fields",
		golden: "TestSliceFields.golden",
		schema: "TestSliceFieldsStructSchema",
		input: {
			NoValidate: [1, 2],
			Required: [1],
			Min: [1],
			OmitEmpty: [1, 2],
			JSONOmitEmpty: [1, 2],
			MinOmitEmpty: [1],
			JSONMinOmitEmpty: [1],
		},
		success: true,
	},

	// --- TestConvertSliceWithValidations ---
	{
		name: "slice validations: requiredSchema accepts array",
		golden: "TestConvertSliceWithValidations.golden",
		schema: "requiredSchema",
		input: { value: ["a"] },
		success: true,
	},
	{
		name: "slice validations: minSchema accepts array with >= 1 item",
		golden: "TestConvertSliceWithValidations.golden",
		schema: "minSchema",
		input: { value: ["a"] },
		success: true,
	},
	{
		name: "slice validations: minSchema rejects empty array",
		golden: "TestConvertSliceWithValidations.golden",
		schema: "minSchema",
		input: { value: [] },
		success: false,
	},
	{
		name: "slice validations: maxSchema accepts array with <= 1 item",
		golden: "TestConvertSliceWithValidations.golden",
		schema: "maxSchema",
		input: { value: ["a"] },
		success: true,
	},
	{
		name: "slice validations: maxSchema rejects array with > 1 item",
		golden: "TestConvertSliceWithValidations.golden",
		schema: "maxSchema",
		input: { value: ["a", "b"] },
		success: false,
	},
	{
		name: "slice validations: lenSchema accepts array of length 1",
		golden: "TestConvertSliceWithValidations.golden",
		schema: "lenSchema",
		input: { value: ["a"] },
		success: true,
	},
	{
		name: "slice validations: lenSchema rejects wrong length",
		golden: "TestConvertSliceWithValidations.golden",
		schema: "lenSchema",
		input: { value: ["a", "b"] },
		success: false,
	},
	{
		name: "slice validations: eqSchema accepts array of length 1",
		golden: "TestConvertSliceWithValidations.golden",
		schema: "eqSchema",
		input: { value: ["a"] },
		success: true,
	},
	{
		name: "slice validations: gtSchema accepts array with >= 2 items",
		golden: "TestConvertSliceWithValidations.golden",
		schema: "gtSchema",
		input: { value: ["a", "b"] },
		success: true,
	},
	{
		name: "slice validations: gtSchema rejects array with < 2 items",
		golden: "TestConvertSliceWithValidations.golden",
		schema: "gtSchema",
		input: { value: ["a"] },
		success: false,
	},
	{
		name: "slice validations: gteSchema accepts array with >= 1 items",
		golden: "TestConvertSliceWithValidations.golden",
		schema: "gteSchema",
		input: { value: ["a"] },
		success: true,
	},
	{
		name: "slice validations: ltSchema accepts empty array",
		golden: "TestConvertSliceWithValidations.golden",
		schema: "ltSchema",
		input: { value: [] },
		success: true,
	},
	{
		name: "slice validations: ltSchema rejects array with >= 1 items",
		golden: "TestConvertSliceWithValidations.golden",
		schema: "ltSchema",
		input: { value: ["a"] },
		success: false,
	},
	{
		name: "slice validations: lteSchema accepts array with <= 1 item",
		golden: "TestConvertSliceWithValidations.golden",
		schema: "lteSchema",
		input: { value: ["a"] },
		success: true,
	},
	{
		name: "slice validations: neSchema accepts non-empty array",
		golden: "TestConvertSliceWithValidations.golden",
		schema: "neSchema",
		input: { value: ["a"] },
		success: true,
	},
	{
		name: "slice validations: neSchema rejects empty array",
		golden: "TestConvertSliceWithValidations.golden",
		schema: "neSchema",
		input: { value: [] },
		success: false,
	},

	// --- TestConvertSliceWithValidations/dive_nested ---
	{
		name: "dive nested: dive1Schema accepts nested array",
		golden: "TestConvertSliceWithValidations/dive_nested.golden",
		schema: "dive1Schema",
		input: { value: [["a", "b"], ["c"]] },
		success: true,
	},
	{
		name: "dive nested: dive2Schema accepts array of arrays with min 1",
		golden: "TestConvertSliceWithValidations/dive_nested.golden",
		schema: "dive2Schema",
		input: { value: [["a"], ["b", "c"]] },
		success: true,
	},

	// --- TestConvertSliceWithValidations/dive_oneof ---
	{
		name: "dive oneof: accepts array of valid enum values",
		golden: "TestConvertSliceWithValidations/dive_oneof.golden",
		schema: "dive_oneofSchema",
		input: { value: ["a", "b", "c"] },
		success: true,
	},
	{
		name: "dive oneof: rejects array with invalid enum value",
		golden: "TestConvertSliceWithValidations/dive_oneof.golden",
		schema: "dive_oneofSchema",
		input: { value: ["a", "d"] },
		success: false,
	},

	// ---------------------------------------------------------------------------
	// MAPS
	// ---------------------------------------------------------------------------

	// --- TestMapStringToString ---
	{
		name: "map string to string: parses valid map",
		golden: "TestMapStringToString.golden",
		schema: "UserSchema",
		input: { Name: "John", Metadata: { key: "val" } },
		success: true,
	},
	{
		name: "map string to string: parses null",
		golden: "TestMapStringToString.golden",
		schema: "UserSchema",
		input: { Name: "John", Metadata: null },
		success: true,
	},

	// --- TestMapStringToInterface ---
	{
		name: "map string to interface: parses valid map with any values",
		golden: "TestMapStringToInterface.golden",
		schema: "UserSchema",
		input: { Name: "John", Metadata: { key: 42, nested: { a: true } } },
		success: true,
	},

	// --- TestMapWithStruct ---
	{
		name: "map with struct: parses valid map",
		golden: "TestMapWithStruct.golden",
		schema: "UserSchema",
		input: { MapWithStruct: { hello: { Title: "World" } } },
		success: true,
	},

	// --- TestMapWithValidations ---
	{
		name: "map validations: requiredSchema accepts map",
		golden: "TestMapWithValidations.golden",
		schema: "requiredSchema",
		input: { value: { a: "b" } },
		success: true,
	},
	{
		name: "map validations: minSchema accepts map with >= 1 keys",
		golden: "TestMapWithValidations.golden",
		schema: "minSchema",
		input: { value: { a: "b" } },
		success: true,
	},
	{
		name: "map validations: minSchema rejects empty map",
		golden: "TestMapWithValidations.golden",
		schema: "minSchema",
		input: { value: {} },
		success: false,
	},
	{
		name: "map validations: maxSchema accepts map with <= 1 keys",
		golden: "TestMapWithValidations.golden",
		schema: "maxSchema",
		input: { value: { a: "b" } },
		success: true,
	},
	{
		name: "map validations: maxSchema rejects map with > 1 keys",
		golden: "TestMapWithValidations.golden",
		schema: "maxSchema",
		input: { value: { a: "b", c: "d" } },
		success: false,
	},
	{
		name: "map validations: lenSchema accepts map with exactly 1 key",
		golden: "TestMapWithValidations.golden",
		schema: "lenSchema",
		input: { value: { a: "b" } },
		success: true,
	},
	{
		name: "map validations: lenSchema rejects map with != 1 keys",
		golden: "TestMapWithValidations.golden",
		schema: "lenSchema",
		input: { value: { a: "b", c: "d" } },
		success: false,
	},
	{
		name: "map validations: eqSchema accepts map with exactly 1 key",
		golden: "TestMapWithValidations.golden",
		schema: "eqSchema",
		input: { value: { a: "b" } },
		success: true,
	},
	{
		name: "map validations: neSchema accepts map with != 1 keys",
		golden: "TestMapWithValidations.golden",
		schema: "neSchema",
		input: { value: { a: "b", c: "d" } },
		success: true,
	},
	{
		name: "map validations: neSchema rejects map with exactly 1 key",
		golden: "TestMapWithValidations.golden",
		schema: "neSchema",
		input: { value: { a: "b" } },
		success: false,
	},
	{
		name: "map validations: gtSchema accepts map with > 1 keys",
		golden: "TestMapWithValidations.golden",
		schema: "gtSchema",
		input: { value: { a: "b", c: "d" } },
		success: true,
	},
	{
		name: "map validations: gtSchema rejects map with <= 1 keys",
		golden: "TestMapWithValidations.golden",
		schema: "gtSchema",
		input: { value: { a: "b" } },
		success: false,
	},
	{
		name: "map validations: gteSchema accepts map with >= 1 keys",
		golden: "TestMapWithValidations.golden",
		schema: "gteSchema",
		input: { value: { a: "b" } },
		success: true,
	},
	{
		name: "map validations: ltSchema accepts empty map",
		golden: "TestMapWithValidations.golden",
		schema: "ltSchema",
		input: { value: {} },
		success: true,
	},
	{
		name: "map validations: ltSchema rejects map with >= 1 keys",
		golden: "TestMapWithValidations.golden",
		schema: "ltSchema",
		input: { value: { a: "b" } },
		success: false,
	},
	{
		name: "map validations: lteSchema accepts map with <= 1 keys",
		golden: "TestMapWithValidations.golden",
		schema: "lteSchema",
		input: { value: { a: "b" } },
		success: true,
	},
	{
		name: "map validations: dive1Schema accepts map with values >= 2 chars",
		golden: "TestMapWithValidations.golden",
		schema: "dive1Schema",
		input: { value: { key: "ab" } },
		success: true,
	},

	// --- TestMapWithValidations/dive_nested ---
	{
		name: "map dive nested: dive2Schema accepts array of maps",
		golden: "TestMapWithValidations/dive_nested.golden",
		schema: "dive2Schema",
		input: { value: [{ aaa: "bbb", ccc: "ddd" }] },
		success: true,
	},
	{
		name: "map dive nested: dive3Schema accepts array of maps with key/value constraints",
		golden: "TestMapWithValidations/dive_nested.golden",
		schema: "dive3Schema",
		input: { value: [{ abc: "abcd", def: "ef" }] },
		success: true,
	},

	// --- TestMapWithNonStringKey/int_key ---
	{
		name: "map int key: parses valid map with coerced number keys",
		golden: "TestMapWithNonStringKey/int_key.golden",
		schema: "Map1Schema",
		input: { Name: "John", Metadata: { "1": "one", "2": "two" } },
		success: true,
	},

	// --- TestMapWithNonStringKey/float_key ---
	{
		name: "map float key: parses valid map with coerced number keys",
		golden: "TestMapWithNonStringKey/float_key.golden",
		schema: "Map3Schema",
		input: { Name: "John", Metadata: { "1.5": "one-half", "2.5": "two-half" } },
		success: true,
	},

	// --- TestMapWithNonStringKey/time_key ---
	{
		name: "map time key: parses valid map with string keys",
		golden: "TestMapWithNonStringKey/time_key.golden",
		schema: "Map2Schema",
		input: { Name: "John", Metadata: { "2021-01-01T00:00:00Z": "new year" } },
		success: true,
	},

	// --- TestNullableWithValidations ---
	{
		name: "nullable with validations: parses full valid object",
		golden: "TestNullableWithValidations.golden",
		schema: "UserSchema",
		input: {
			Name: "John",
			PtrMapOptionalNullable1: null,
			PtrMapOptionalNullable2: null,
			PtrMap1: { a: 1, b: 2, c: 3 },
			PtrMap2: { a: 1, b: 2, c: 3 },
			PtrMapNullable: { a: 1, b: 2, c: 3 },
			MapOptional1: undefined,
			MapOptional2: undefined,
			Map1: { a: 1, b: 2, c: 3 },
			Map2: { a: 1, b: 2, c: 3 },
			MapNullable: { a: 1, b: 2, c: 3 },
			PtrSliceOptionalNullable1: null,
			PtrSliceOptionalNullable2: null,
			PtrSlice1: ["a", "b", "c"],
			PtrSlice2: ["a", "b", "c"],
			PtrSliceNullable: ["a", "b", "c"],
			SliceOptional1: undefined,
			SliceOptional2: undefined,
			Slice1: ["a", "b", "c"],
			Slice2: ["a", "b", "c"],
			SliceNullable: ["a", "b", "c"],
			PtrIntOptional1: undefined,
			PtrIntOptional2: undefined,
			PtrInt1: 3,
			PtrInt2: 3,
			PtrIntNullable: 3,
			PtrStringOptional1: undefined,
			PtrStringOptional2: undefined,
			PtrString1: "abc",
			PtrString2: "abc",
			PtrStringNullable: "abc",
		},
		success: true,
	},

	// ---------------------------------------------------------------------------
	// NESTED V4
	// ---------------------------------------------------------------------------

	// --- TestNestedStruct/v4 ---
	{
		name: "nested struct v4: parses valid object with spread shapes",
		golden: "TestNestedStruct/v4.golden",
		schema: "UserSchema",
		input: { Tags: ["a", "b"], ID: "123", name: "John" },
		success: true,
	},

	// --- TestRecursive1/v4 ---
	{
		name: "recursive1 v4: parses nested children",
		golden: "TestRecursive1/v4.golden",
		schema: "NestedItemSchema",
		input: {
			id: 1,
			title: "Root",
			pos: 0,
			parent_id: 0,
			project_id: 1,
			children: [
				{
					id: 2,
					title: "Child",
					pos: 1,
					parent_id: 1,
					project_id: 1,
					children: null,
				},
			],
		},
		success: true,
	},

	// --- TestRecursive2/v4 ---
	{
		name: "recursive2 v4: parses ParentSchema with nested next",
		golden: "TestRecursive2/v4.golden",
		schema: "ParentSchema",
		input: {
			child: {
				value: 1,
				next: {
					value: 2,
					next: null,
				},
			},
		},
		success: true,
	},

	// --- TestRecursiveEmbeddedStruct/v4 ---
	{
		name: "recursive embedded v4: parses ItemBSchema",
		golden: "TestRecursiveEmbeddedStruct/v4.golden",
		schema: "ItemBSchema",
		input: {
			Name: "root",
			Children: [
				{ Name: "child1", Children: null },
				{ Name: "child2", Children: [{ Name: "grandchild", Children: null }] },
			],
		},
		success: true,
	},

	// --- TestRecursiveEmbeddedWithPointersAndDates/recursive_struct_with_pointer_field_and_date/v4 ---
	{
		name: "recursive with dates v4: parses TreeSchema",
		golden:
			"TestRecursiveEmbeddedWithPointersAndDates/recursive_struct_with_pointer_field_and_date/v4.golden",
		schema: "TreeSchema",
		input: {
			UpdatedAt: "2021-01-01T00:00:00Z",
			Value: "root",
			CreatedAt: "2021-01-01T00:00:00Z",
			Children: [
				{
					Value: "child",
					CreatedAt: "2021-02-01T00:00:00Z",
					Children: null,
				},
			],
		},
		success: true,
		output: {
			UpdatedAt: new Date("2021-01-01T00:00:00Z"),
			Value: "root",
			CreatedAt: new Date("2021-01-01T00:00:00Z"),
			Children: [
				{
					Value: "child",
					CreatedAt: new Date("2021-02-01T00:00:00Z"),
					Children: null,
				},
			],
		},
	},

	// --- TestRecursiveEmbeddedWithPointersAndDates/embedded_struct_with_pointer_to_self_and_date/v4 ---
	{
		name: "embedded self-pointer with dates v4: parses ArticleSchema",
		golden:
			"TestRecursiveEmbeddedWithPointersAndDates/embedded_struct_with_pointer_to_self_and_date/v4.golden",
		schema: "ArticleSchema",
		input: {
			Title: "Article",
			Text: "Hello",
			Timestamp: "2021-01-01T00:00:00Z",
			Reply: {
				Text: "Reply",
				Timestamp: "2021-02-01T00:00:00Z",
				Reply: null,
			},
		},
		success: true,
		output: {
			Title: "Article",
			Text: "Hello",
			Timestamp: new Date("2021-01-01T00:00:00Z"),
			Reply: {
				Text: "Reply",
				Timestamp: new Date("2021-02-01T00:00:00Z"),
				Reply: null,
			},
		},
	},

	// ---------------------------------------------------------------------------
	// STRING VALIDATIONS
	// ---------------------------------------------------------------------------

	// --- eqSchema ---
	{
		name: "string eq: accepts exact match 'hello'",
		golden: "TestStringValidations.golden",
		schema: "eqSchema",
		input: { value: "hello" },
		success: true,
	},
	{
		name: "string eq: rejects non-match",
		golden: "TestStringValidations.golden",
		schema: "eqSchema",
		input: { value: "world" },
		success: false,
	},

	// --- neSchema ---
	{
		name: "string ne: accepts value not 'hello'",
		golden: "TestStringValidations.golden",
		schema: "neSchema",
		input: { value: "world" },
		success: true,
	},
	{
		name: "string ne: rejects 'hello'",
		golden: "TestStringValidations.golden",
		schema: "neSchema",
		input: { value: "hello" },
		success: false,
	},

	// --- oneofSchema ---
	{
		name: "string oneof: accepts 'hello'",
		golden: "TestStringValidations.golden",
		schema: "oneofSchema",
		input: { value: "hello" },
		success: true,
	},
	{
		name: "string oneof: rejects invalid value",
		golden: "TestStringValidations.golden",
		schema: "oneofSchema",
		input: { value: "invalid" },
		success: false,
	},

	// --- lenSchema ---
	{
		name: "string len: accepts string of length 5",
		golden: "TestStringValidations.golden",
		schema: "lenSchema",
		input: { value: "abcde" },
		success: true,
	},
	{
		name: "string len: rejects string of wrong length",
		golden: "TestStringValidations.golden",
		schema: "lenSchema",
		input: { value: "abc" },
		success: false,
	},

	// --- minSchema ---
	{
		name: "string min: accepts string of length >= 5",
		golden: "TestStringValidations.golden",
		schema: "minSchema",
		input: { value: "abcde" },
		success: true,
	},
	{
		name: "string min: rejects string of length < 5",
		golden: "TestStringValidations.golden",
		schema: "minSchema",
		input: { value: "abc" },
		success: false,
	},

	// --- maxSchema ---
	{
		name: "string max: accepts string of length <= 5",
		golden: "TestStringValidations.golden",
		schema: "maxSchema",
		input: { value: "abcde" },
		success: true,
	},
	{
		name: "string max: rejects string of length > 5",
		golden: "TestStringValidations.golden",
		schema: "maxSchema",
		input: { value: "abcdef" },
		success: false,
	},

	// --- containsSchema ---
	{
		name: "string contains: accepts string containing 'hello'",
		golden: "TestStringValidations.golden",
		schema: "containsSchema",
		input: { value: "say hello world" },
		success: true,
	},
	{
		name: "string contains: rejects string not containing 'hello'",
		golden: "TestStringValidations.golden",
		schema: "containsSchema",
		input: { value: "goodbye" },
		success: false,
	},

	// --- startswithSchema ---
	{
		name: "string startswith: accepts string starting with 'hello'",
		golden: "TestStringValidations.golden",
		schema: "startswithSchema",
		input: { value: "hello world" },
		success: true,
	},
	{
		name: "string startswith: rejects string not starting with 'hello'",
		golden: "TestStringValidations.golden",
		schema: "startswithSchema",
		input: { value: "world hello" },
		success: false,
	},

	// --- endswithSchema ---
	{
		name: "string endswith: accepts string ending with 'hello'",
		golden: "TestStringValidations.golden",
		schema: "endswithSchema",
		input: { value: "world hello" },
		success: true,
	},
	{
		name: "string endswith: rejects string not ending with 'hello'",
		golden: "TestStringValidations.golden",
		schema: "endswithSchema",
		input: { value: "hello world" },
		success: false,
	},

	// --- requiredSchema ---
	{
		name: "string required: accepts non-empty string",
		golden: "TestStringValidations.golden",
		schema: "requiredSchema",
		input: { value: "a" },
		success: true,
	},
	{
		name: "string required: rejects empty string",
		golden: "TestStringValidations.golden",
		schema: "requiredSchema",
		input: { value: "" },
		success: false,
	},

	// --- lowercaseSchema ---
	{
		name: "string lowercase: accepts lowercase",
		golden: "TestStringValidations.golden",
		schema: "lowercaseSchema",
		input: { value: "hello" },
		success: true,
	},
	{
		name: "string lowercase: rejects uppercase",
		golden: "TestStringValidations.golden",
		schema: "lowercaseSchema",
		input: { value: "Hello" },
		success: false,
	},

	// --- uppercaseSchema ---
	{
		name: "string uppercase: accepts uppercase",
		golden: "TestStringValidations.golden",
		schema: "uppercaseSchema",
		input: { value: "HELLO" },
		success: true,
	},
	{
		name: "string uppercase: rejects lowercase",
		golden: "TestStringValidations.golden",
		schema: "uppercaseSchema",
		input: { value: "Hello" },
		success: false,
	},

	// --- boolean_validatorSchema ---
	{
		name: "string boolean: accepts 'true'",
		golden: "TestStringValidations.golden",
		schema: "boolean_validatorSchema",
		input: { value: "true" },
		success: true,
	},
	{
		name: "string boolean: rejects 'yes'",
		golden: "TestStringValidations.golden",
		schema: "boolean_validatorSchema",
		input: { value: "yes" },
		success: false,
	},

	// --- json_validatorSchema ---
	{
		name: "string json: accepts valid JSON",
		golden: "TestStringValidations.golden",
		schema: "json_validatorSchema",
		input: { value: '{"key":"value"}' },
		success: true,
	},
	{
		name: "string json: rejects invalid JSON",
		golden: "TestStringValidations.golden",
		schema: "json_validatorSchema",
		input: { value: "{invalid" },
		success: false,
	},

	// --- alphaSchema ---
	{
		name: "string alpha: accepts alpha-only",
		golden: "TestStringValidations.golden",
		schema: "alphaSchema",
		input: { value: "hello" },
		success: true,
	},
	{
		name: "string alpha: rejects non-alpha",
		golden: "TestStringValidations.golden",
		schema: "alphaSchema",
		input: { value: "hello123" },
		success: false,
	},

	// --- number_validatorSchema ---
	{
		name: "string number: accepts digits only",
		golden: "TestStringValidations.golden",
		schema: "number_validatorSchema",
		input: { value: "12345" },
		success: true,
	},
	{
		name: "string number: rejects non-digit",
		golden: "TestStringValidations.golden",
		schema: "number_validatorSchema",
		input: { value: "123abc" },
		success: false,
	},

	// ---------------------------------------------------------------------------
	// NUMBER VALIDATIONS
	// ---------------------------------------------------------------------------

	// --- gte_lteSchema ---
	{
		name: "number gte_lte: accepts 18",
		golden: "TestNumberValidations.golden",
		schema: "gte_lteSchema",
		input: { value: 18 },
		success: true,
	},
	{
		name: "number gte_lte: accepts 60",
		golden: "TestNumberValidations.golden",
		schema: "gte_lteSchema",
		input: { value: 60 },
		success: true,
	},
	{
		name: "number gte_lte: rejects 17",
		golden: "TestNumberValidations.golden",
		schema: "gte_lteSchema",
		input: { value: 17 },
		success: false,
	},
	{
		name: "number gte_lte: rejects 61",
		golden: "TestNumberValidations.golden",
		schema: "gte_lteSchema",
		input: { value: 61 },
		success: false,
	},

	// --- gt_ltSchema ---
	{
		name: "number gt_lt: accepts 19",
		golden: "TestNumberValidations.golden",
		schema: "gt_ltSchema",
		input: { value: 19 },
		success: true,
	},
	{
		name: "number gt_lt: rejects 18 (not >18)",
		golden: "TestNumberValidations.golden",
		schema: "gt_ltSchema",
		input: { value: 18 },
		success: false,
	},
	{
		name: "number gt_lt: rejects 60 (not <60)",
		golden: "TestNumberValidations.golden",
		schema: "gt_ltSchema",
		input: { value: 60 },
		success: false,
	},

	// --- number eqSchema ---
	{
		name: "number eq: accepts 18",
		golden: "TestNumberValidations.golden",
		schema: "eqSchema",
		input: { value: 18 },
		success: true,
	},
	{
		name: "number eq: rejects 19",
		golden: "TestNumberValidations.golden",
		schema: "eqSchema",
		input: { value: 19 },
		success: false,
	},

	// --- number neSchema ---
	{
		name: "number ne: accepts 19",
		golden: "TestNumberValidations.golden",
		schema: "neSchema",
		input: { value: 19 },
		success: true,
	},
	{
		name: "number ne: rejects 18",
		golden: "TestNumberValidations.golden",
		schema: "neSchema",
		input: { value: 18 },
		success: false,
	},

	// --- number oneofSchema ---
	{
		name: "number oneof: accepts 18",
		golden: "TestNumberValidations.golden",
		schema: "oneofSchema",
		input: { value: 18 },
		success: true,
	},
	{
		name: "number oneof: rejects 21",
		golden: "TestNumberValidations.golden",
		schema: "oneofSchema",
		input: { value: 21 },
		success: false,
	},

	// --- number min_maxSchema ---
	{
		name: "number min_max: accepts 30",
		golden: "TestNumberValidations.golden",
		schema: "min_maxSchema",
		input: { value: 30 },
		success: true,
	},

	// --- number lenSchema ---
	{
		name: "number len: accepts 18",
		golden: "TestNumberValidations.golden",
		schema: "lenSchema",
		input: { value: 18 },
		success: true,
	},

	// ---------------------------------------------------------------------------
	// FORMAT VALIDATORS V4
	// ---------------------------------------------------------------------------

	// --- emailSchema ---
	{
		name: "email: accepts valid email",
		golden: "TestFormatValidators/format_only/v4.golden",
		schema: "emailSchema",
		input: { value: "test@example.com" },
		success: true,
	},
	{
		name: "email: rejects invalid email",
		golden: "TestFormatValidators/format_only/v4.golden",
		schema: "emailSchema",
		input: { value: "notanemail" },
		success: false,
	},

	// --- urlSchema ---
	{
		name: "url: accepts valid url",
		golden: "TestFormatValidators/format_only/v4.golden",
		schema: "urlSchema",
		input: { value: "https://example.com" },
		success: true,
	},
	{
		name: "url: rejects invalid url",
		golden: "TestFormatValidators/format_only/v4.golden",
		schema: "urlSchema",
		input: { value: "not a url" },
		success: false,
	},

	// --- ipv4Schema ---
	{
		name: "ipv4: accepts valid ipv4",
		golden: "TestFormatValidators/format_only/v4.golden",
		schema: "ipv4Schema",
		input: { value: "127.0.0.1" },
		success: true,
	},
	{
		name: "ipv4: rejects invalid ipv4",
		golden: "TestFormatValidators/format_only/v4.golden",
		schema: "ipv4Schema",
		input: { value: "999.999.999.999" },
		success: false,
	},

	// --- ipv6Schema ---
	{
		name: "ipv6: accepts valid ipv6",
		golden: "TestFormatValidators/format_only/v4.golden",
		schema: "ipv6Schema",
		input: { value: "::1" },
		success: true,
	},
	{
		name: "ipv6: rejects invalid ipv6",
		golden: "TestFormatValidators/format_only/v4.golden",
		schema: "ipv6Schema",
		input: { value: "not-ipv6" },
		success: false,
	},

	// --- base64Schema ---
	{
		name: "base64: accepts valid base64",
		golden: "TestFormatValidators/format_only/v4.golden",
		schema: "base64Schema",
		input: { value: "SGVsbG8=" },
		success: true,
	},
	{
		name: "base64: rejects invalid base64",
		golden: "TestFormatValidators/format_only/v4.golden",
		schema: "base64Schema",
		input: { value: "not base64!!!" },
		success: false,
	},

	// --- uuid4Schema ---
	{
		name: "uuid4: accepts valid uuid v4",
		golden: "TestFormatValidators/format_only/v4.golden",
		schema: "uuid4Schema",
		input: { value: "550e8400-e29b-41d4-a716-446655440000" },
		success: true,
	},
	{
		name: "uuid4: rejects invalid uuid",
		golden: "TestFormatValidators/format_only/v4.golden",
		schema: "uuid4Schema",
		input: { value: "not-a-uuid" },
		success: false,
	},

	// --- md5Schema ---
	{
		name: "md5: accepts valid md5 hash",
		golden: "TestFormatValidators/format_only/v4.golden",
		schema: "md5Schema",
		input: { value: "d41d8cd98f00b204e9800998ecf8427e" },
		success: true,
	},
	{
		name: "md5: rejects invalid md5",
		golden: "TestFormatValidators/format_only/v4.golden",
		schema: "md5Schema",
		input: { value: "not-a-hash" },
		success: false,
	},

	// ---------------------------------------------------------------------------
	// UNION V4
	// ---------------------------------------------------------------------------

	// --- ipSchema ---
	{
		name: "ip union: accepts valid ipv4",
		golden: "TestFormatValidators/union_only/v4.golden",
		schema: "ipSchema",
		input: { value: "127.0.0.1" },
		success: true,
	},
	{
		name: "ip union: accepts valid ipv6",
		golden: "TestFormatValidators/union_only/v4.golden",
		schema: "ipSchema",
		input: { value: "::1" },
		success: true,
	},
	{
		name: "ip union: rejects invalid ip",
		golden: "TestFormatValidators/union_only/v4.golden",
		schema: "ipSchema",
		input: { value: "notanip" },
		success: false,
	},

	// ---------------------------------------------------------------------------
	// SPECIAL
	// ---------------------------------------------------------------------------

	// --- TestEverything ---
	{
		name: "everything: parses full valid object",
		golden: "TestEverything.golden",
		schema: "UserSchema",
		input: {
			Name: "John",
			Nickname: null,
			Age: 30,
			Height: 5.9,
			OldPostWithMetaData: { Title: "Hello", Post: { Title: "World" } },
			Tags: ["a", "b"],
			TagsOptional: ["x"],
			TagsOptionalNullable: null,
			Favourites: [{ Name: "Alice" }],
			Posts: [{ Title: "Post1" }],
			Post: { Title: "Main" },
			PostOptional: { Title: "Optional" },
			PostOptionalNullable: null,
			Metadata: { key: "val" },
			MetadataOptional: undefined,
			MetadataOptionalNullable: null,
			ExtendedProps: { any: "thing" },
			ExtendedPropsOptional: null,
			ExtendedPropsNullable: null,
			ExtendedPropsOptionalNullable: null,
			ExtendedPropsVeryIndirect: null,
			NewPostWithMetaData: { Title: "New", Post: { Title: "Inner" } },
			VeryNewPost: { Title: "VeryNew" },
			MapWithStruct: { k: { Title: "T", Post: { Title: "P" } } },
		},
		success: true,
	},

	// --- TestEverythingWithValidations ---
	{
		name: "everything with validations: parses full valid object",
		golden: "TestEverythingWithValidations.golden",
		schema: "UserSchema",
		input: {
			Name: "John",
			Nickname: null,
			Age: 18,
			Height: 1.5,
			OldPostWithMetaData: { Title: "Hello", Post: { Title: "World" } },
			Tags: ["a", "b"],
			TagsOptional: ["a", "b"],
			TagsOptionalNullable: ["a", "b"],
			Favourites: null,
			Posts: [{ Title: "Hello" }],
			Post: { Title: "Hello" },
			PostOptional: { Title: "Hello" },
			PostOptionalNullable: { Title: "Hello" },
			Metadata: null,
			MetadataLength: { Hello: "World" },
			MetadataOptional: undefined,
			MetadataOptionalNullable: null,
			ExtendedProps: null,
			ExtendedPropsOptional: undefined,
			ExtendedPropsNullable: null,
			ExtendedPropsOptionalNullable: null,
			ExtendedPropsVeryIndirect: null,
			NewPostWithMetaData: { Title: "Hello", Post: { Title: "World" } },
			VeryNewPost: { Title: "Hello" },
			MapWithStruct: {
				Hello: { Title: "World", Post: { Title: "Hello" } },
			},
		},
		success: true,
	},

	// --- TestGenerics ---
	{
		name: "generics: StringIntPairSchema",
		golden: "TestGenerics.golden",
		schema: "StringIntPairSchema",
		input: { First: "hello", Second: 42 },
		success: true,
	},
	{
		name: "generics: GenericPairIntBoolSchema",
		golden: "TestGenerics.golden",
		schema: "GenericPairIntBoolSchema",
		input: { First: 1, Second: true },
		success: true,
	},
	{
		name: "generics: PairMapStringIntBoolSchema",
		golden: "TestGenerics.golden",
		schema: "PairMapStringIntBoolSchema",
		input: { items: { key: { First: 1, Second: false } } },
		success: true,
	},

	// --- TestInterfaceAny ---
	{
		name: "interface any: accepts any value for Metadata",
		golden: "TestInterfaceAny.golden",
		schema: "UserSchema",
		input: { Name: "John", Metadata: { anything: [1, 2, 3] } },
		success: true,
	},

	// --- TestCustomTag/v4 ---
	{
		name: "custom tag v4: parses SortParamsSchema",
		golden: "TestCustomTag/v4.golden",
		schema: "SortParamsSchema",
		input: { order: "asc", field: "name" },
		success: true,
	},

	// --- TestZodV4Defaults/enum_keyed_maps_become_partial_records ---
	{
		name: "v4 defaults: enum keyed maps become partial records",
		golden: "TestMapWithEnumKey/v4.golden",
		schema: "PayloadSchema",
		input: { Metadata: { draft: "some note" } },
		success: true,
	},
	{
		name: "v4 defaults: enum keyed maps for partial records reject invalid keys",
		golden: "TestMapWithEnumKey/v4.golden",
		schema: "PayloadSchema",
		input: { Metadata: { invalid: "some note" } },
		success: false,
	},

	// --- TestZodV4Defaults/ip_unions_inherit_generic_string_constraints ---
	{
		name: "v4 defaults: ip unions inherit generic string constraints",
		golden:
			"TestZodV4Defaults/ip_unions_inherit_generic_string_constraints.golden",
		schema: "PayloadSchema",
		input: { Address: "127.0.0.1" },
		success: true,
	},

	// --- TestZodV4Defaults/oneof_takes_precedence_over_ip_specialization ---
	{
		name: "v4 defaults: oneof takes precedence over ip specialization",
		golden:
			"TestZodV4Defaults/oneof_takes_precedence_over_ip_specialization.golden",
		schema: "PayloadSchema",
		input: { Address: "127.0.0.1" },
		success: true,
	},

	// --- TestZodV4Defaults/optional_format_with_nullable_pointer/v4 ---
	{
		name: "v4 defaults: optional format with nullable pointer accepts null",
		golden: "TestZodV4Defaults/optional_format_with_nullable_pointer/v4.golden",
		schema: "PayloadSchema",
		input: { email: null },
		success: true,
	},
	{
		name: "v4 defaults: optional format with nullable pointer accepts valid email",
		golden: "TestZodV4Defaults/optional_format_with_nullable_pointer/v4.golden",
		schema: "PayloadSchema",
		input: { email: "test@example.com" },
		success: true,
	},

	// --- TestZodV4Defaults/string_formats_use_zod_v4_builders ---
	{
		name: "v4 defaults: string formats use zod v4 builders",
		golden: "TestZodV4Defaults/string_formats_use_zod_v4_builders.golden",
		schema: "PayloadSchema",
		input: {
			Email: "test@example.com",
			Link: "https://example.com",
			Base64: "SGVsbG8=",
			ID: "550e8400-e29b-41d4-a716-446655440000",
			Checksum: "d41d8cd98f00b204e9800998ecf8427e",
		},
		success: true,
	},

	// --- Custom types ---
	{
		name: "custom type: mapped to string",
		golden: "TestCustomTypes/custom_type_mapped_to_string.golden",
		schema: "UserSchema",
		input: { Name: "John", Money: "123.45" },
		success: true,
	},
	{
		name: "custom type: resolves inner generic type",
		golden: "TestCustomTypes/custom_type_resolves_inner_generic_type.golden",
		schema: "UserSchema",
		input: {
			MaybeName: "John",
			MaybeAge: 30,
			MaybeHeight: 1.8,
			MaybeProfile: { Bio: "Hello" },
		},
		success: true,
	},
	{
		name: "custom type: resolves inner generic with nullish",
		golden: "TestCustomTypes/custom_type_resolves_inner_generic_type.golden",
		schema: "UserSchema",
		input: {
			MaybeName: null,
			MaybeAge: undefined,
			MaybeHeight: null,
			MaybeProfile: undefined,
		},
		success: true,
	},
	{
		name: "custom type: nullable pointer with custom handler",
		golden: "TestCustomTypes/custom_type_with_nullable_control.golden",
		schema: "UserSchema",
		input: { Name: "John", Email: null },
		success: true,
	},
];
