import type {
	MonitorCertInfo,
	MonitorStats,
	NetworkMonitorRecord,
	NetworkMonitorStatsRecord,
	RawMonitorStatsRecord,
} from "@/types"
import { toFixedFloat } from "./utils"

/** Derive chart metrics from the counts and response sum stored at every retention tier. */
export function getMonitorStats(record: RawMonitorStatsRecord): MonitorStats {
	return {
		res_avg: record.success_count > 0 ? toFixedFloat(record.res_sum / record.success_count, 2) : 0,
		res_min: record.res_min,
		res_max: record.res_max,
		loss:
			record.total_count > 0
				? toFixedFloat(((record.total_count - record.success_count) / record.total_count) * 100, 2)
				: 0,
	}
}

/**
 * Return the records that have stats for one monitor, with an empty record inserted wherever
 * consecutive records are further apart than expected (e.g. while the agent was disconnected),
 * so charts break the line there instead of drawing across the missing time.
 */
export function withMonitorGaps(
	records: NetworkMonitorStatsRecord[],
	monitor: Pick<NetworkMonitorRecord, "id" | "interval">,
	expectedInterval: number
): NetworkMonitorStatsRecord[] {
	// long-interval monitors only get a record when a new probe completes
	const maxGap = Math.max(expectedInterval, monitor.interval * 1000) * 1.5
	const result: NetworkMonitorStatsRecord[] = []
	let prevTime = 0
	for (const record of records) {
		// skip appendData's gap markers (created: null) and records without this monitor
		if (record.created == null || !record.stats?.[monitor.id]) continue
		if (prevTime && record.created - prevTime > maxGap) {
			result.push({ created: (prevTime + record.created) / 2, stats: {} })
		}
		prevTime = record.created
		result.push(record)
	}
	return result
}

export function getMonitorTarget(monitor: Pick<NetworkMonitorRecord, "target" | "protocol" | "port">) {
	if (monitor.protocol !== "tcp") return monitor.target
	const host = monitor.target.includes(":") && !monitor.target.startsWith("[") ? `[${monitor.target}]` : monitor.target
	return `${host}:${monitor.port}`
}

/** Whole days until the certificate expires; negative once expired. */
export function getCertDaysLeft(cert: Pick<MonitorCertInfo, "expires">, now = Date.now()) {
	return Math.floor((cert.expires - now) / 86_400_000)
}

/** Expiry severity used for certificate colors. */
export function getCertExpiryLevel(daysLeft: number): "ok" | "warning" | "critical" {
	if (daysLeft < 7) return "critical"
	if (daysLeft < 14) return "warning"
	return "ok"
}
