import { describe, expect, test } from "bun:test"
import { formatSpeedtestInterval, getSpeedtestServerLabel, SPEEDTEST_INTERVALS } from "../src/lib/speedtest-utils"

describe("formatSpeedtestInterval", () => {
	test("uses the largest whole unit", () => {
		expect(SPEEDTEST_INTERVALS.map(formatSpeedtestInterval)).toEqual(["1m", "15m", "30m", "1h", "3h", "6h", "12h", "1d"])
		expect(formatSpeedtestInterval(90)).toBe("90m")
		expect(formatSpeedtestInterval(2880)).toBe("2d")
	})
})

describe("getSpeedtestServerLabel", () => {
	test("combines name, location and pinned server ID", () => {
		expect(getSpeedtestServerLabel({ server_id: 42, server_name: "Example", server_location: "Amsterdam" })).toBe(
			"Example — Amsterdam (#42)"
		)
		expect(getSpeedtestServerLabel({ server_id: 42, server_name: "", server_location: "" })).toBe("#42")
		expect(getSpeedtestServerLabel({ server_id: 0, server_name: "Example", server_location: "" })).toBe("Example")
		expect(getSpeedtestServerLabel({ server_id: 0, server_name: "", server_location: "" })).toBe("")
	})
})
