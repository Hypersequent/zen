import {z} from "zod"
import {describe, expect, it} from 'vitest'

describe("Zod time tests", () => {
	it('TestStructTime', () => {
		const UserSchema = z.object({
			Name: z.string(),
			When: z.coerce.date()
		})

		const user1 = UserSchema.parse({
			Name: "John",
			When: "2021-01-01T00:00:00Z",
		})
		expect(user1).toEqual({
			Name: "John",
			When: new Date("2021-01-01T00:00:00Z"),
		})

		const user2 = UserSchema.parse({
			Name: "John",
			When: 1609459200000,
		})
		expect(user2).toEqual({
			Name: "John",
			When: new Date("2021-01-01T00:00:00Z"),
		})

		const user3 = UserSchema.parse({
			Name: "John",
			When: null,
		})
		expect(user3).toEqual({
			Name: "John",
			When: new Date(0),
		})

		const user4 = UserSchema.parse({
			Name: "John",
			When: "0001-01-01T00:00:00Z"
		})
		expect(user4).toEqual({
			Name: "John",
			When: new Date("0001-01-01T00:00:00Z"),
		})

		const user5 = UserSchema.safeParse({
			Name: "John",
			When: "",
		});
		expect(user5.success).toBe(false)
	})

	it('TestTimeWithRequired', () => {
		const UserSchema = z.object({
			Name: z.string(),
			When: z.coerce.date().refine(
				(val) => val.getTime() !== new Date('0001-01-01T00:00:00Z').getTime() && val.getTime() !== new Date(0).getTime(),
				'Invalid date'
			),
		})

		const user1 = UserSchema.parse({
			Name: "John",
			When: "2021-01-01T00:00:00Z",
		})
		expect(user1).toEqual({
			Name: "John",
			When: new Date("2021-01-01T00:00:00Z"),
		})

		const user2 = UserSchema.parse({
			Name: "John",
			When: 1609459200000,
		})
		expect(user2).toEqual({
			Name: "John",
			When: new Date("2021-01-01T00:00:00Z"),
		})

		const user3 = UserSchema.safeParse({
			Name: "John",
			When: null,
		})
		expect(user3.success).toBe(false)

		const user4 = UserSchema.safeParse({
			Name: "John",
			When: "0001-01-01T00:00:00Z"
		})
		expect(user4.success).toBe(false)

		const user5 = UserSchema.safeParse({
			Name: "John",
			When: "",
		});
		expect(user5.success).toBe(false)
	})
})

describe("Zod test everything validations", () => {
	it('TestEverything', () => {
		const PostSchema = z.object({
			Title: z.string().min(1),
		})
		type Post = z.infer<typeof PostSchema>

		const PostWithMetaDataSchema = z.object({
			Title: z.string().min(1),
			Post: PostSchema,
		})
		type PostWithMetaData = z.infer<typeof PostWithMetaDataSchema>

		const UserSchema = z.object({
			Name: z.string().min(1),
			Nickname: z.string().nullable(),
			Age: z.number().gte(18).refine((val) => val !== 0),
			Height: z.number().gte(1.5).refine((val) => val !== 0),
			OldPostWithMetaData: PostWithMetaDataSchema,
			Tags: z.string().array().nonempty().min(1),
			TagsOptional: z.string().array().optional(),
			TagsOptionalNullable: z.string().array().optional().nullable(),
			Favourites: z.object({
				Name: z.string().min(1),
			}).array().nullable(),
			Posts: PostSchema.array().nonempty(),
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
		type User = z.infer<typeof UserSchema>

		const user1 = UserSchema.parse({
			Name: "John",
			Nickname: null,
			Age: 18,
			Height: 1.5,
			OldPostWithMetaData: {
				Title: "Hello",
				Post: {
					Title: "World",
				},
			},
			Tags: ["a", "b"],
			TagsOptional: ["a", "b"],
			TagsOptionalNullable: ["a", "b"],
			Favourites: null,
			Posts: [
				{
					Title: "Hello",
				},
			],
			Post: {
				Title: "Hello",
			},
			PostOptional: {
				Title: "Hello",
			},
			PostOptionalNullable: {
				Title: "Hello",
			},
			Metadata: null,
			MetadataLength: {
				"Hello": "World",
			},
			MetadataOptional: undefined,
			MetadataOptionalNullable: null,
			ExtendedProps: null,
			ExtendedPropsOptional: undefined,
			ExtendedPropsNullable: null,
			ExtendedPropsOptionalNullable: null,
			ExtendedPropsVeryIndirect: null,
			NewPostWithMetaData: {
				Title: "Hello",
				Post: {
					Title: "World",
				},
			},
			VeryNewPost: {
				Title: "Hello",
			},
			MapWithStruct: {
				"Hello": {
					Title: "World",
					Post: {
						Title: "Hello",
					},
				},
			},
		})
		expect(user1).toEqual({
			Name: "John",
			Nickname: null,
			Age: 18,
			Height: 1.5,
			OldPostWithMetaData: {
				Title: "Hello",
				Post: {
					Title: "World",
				},
			},
			Tags: ["a", "b"],
			TagsOptional: ["a", "b"],
			TagsOptionalNullable: ["a", "b"],
			Favourites: null,
			Posts: [
				{
					Title: "Hello",
				},
			],
			Post: {
				Title: "Hello",
			},
			PostOptional: {
				Title: "Hello",
			},
			PostOptionalNullable: {
				Title: "Hello",
			},
			Metadata: null,
			MetadataLength: {
				"Hello": "World",
			},
			MetadataOptional: undefined,
			MetadataOptionalNullable: null,
			ExtendedProps: null,
			ExtendedPropsOptional: undefined,
			ExtendedPropsNullable: null,
			ExtendedPropsOptionalNullable: null,
			ExtendedPropsVeryIndirect: null,
			NewPostWithMetaData: {
				Title: "Hello",
				Post: {
					Title: "World",
				},
			},
			VeryNewPost: {
				Title: "Hello",
			},
			MapWithStruct: {
				"Hello": {
					Title: "World",
					Post: {
						Title: "Hello",
					},
				},
			},
		})
	})
})

describe("Zod test enum", () => {
	it('TestEnum1', () => {
		const EnumSchema = z.enum(["a", "b"])
		type Enum = z.infer<typeof EnumSchema>

		const enum1 = EnumSchema.parse("a")
		testStringType(enum1)
		// testConstType(enum1)
		// Does not work with const
	})

	it('TestEnum2', () => {
		const EnumSchema = z.enum(["abc", "def"] as const)
		type Enum = z.infer<typeof EnumSchema>

		const enum1 = EnumSchema.parse("abc")
		testStringType(enum1)
		testConstType(enum1)
		// Works with both string and const
	})
})

function testStringType(x: string) {
	console.log(x)
}

function testConstType(x: "abc"|"def") {
	console.log(x)
}
