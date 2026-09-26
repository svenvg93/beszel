import type { SpeedtestRecord } from "@/types"

/** Speedtest interval presets in minutes. */
// TODO: remove the 1 minute preset before merging; it is for testing only.
export const SPEEDTEST_INTERVALS = [1, 15, 30, 60, 180, 360, 720, 1440] as const

export const DEFAULT_SPEEDTEST_INTERVAL = 360

/** Format an interval in minutes as a short duration, e.g. "30m" or "6h". */
export function formatSpeedtestInterval(minutes: number) {
	if (minutes % 1440 === 0) return `${minutes / 1440}d`
	if (minutes % 60 === 0) return `${minutes / 60}h`
	return `${minutes}m`
}

/** Label for the server a speedtest uses. */
export function getSpeedtestServerLabel(
	speedtest: Pick<SpeedtestRecord, "server_id" | "server_name" | "server_location">
) {
	const name = [speedtest.server_name, speedtest.server_location].filter(Boolean).join(" — ")
	if (speedtest.server_id) return name ? `${name} (#${speedtest.server_id})` : `#${speedtest.server_id}`
	return name
}
