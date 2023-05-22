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

