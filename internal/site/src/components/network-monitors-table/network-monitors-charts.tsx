import { Trans } from "@lingui/react/macro"
import { useStore } from "@nanostores/react"
import { atom } from "nanostores"
import { ChevronDownIcon } from "lucide-react"
import { memo, useCallback, useMemo, useState } from "react"
import ChartTimeSelect from "@/components/charts/chart-time-select"
import { LossChart, ResponseChart } from "@/components/routes/system/charts/monitors-charts"
import { Button } from "@/components/ui/button"
import { getMonitorIdentityKey } from "@/lib/monitor-compare"
import { getMonitorLabels, withMonitorGaps } from "@/lib/network-monitor-utils"
import { $allSystemsById, $direction, getUserChartTime } from "@/lib/stores"
import { useProtocolMonitorStats } from "@/lib/use-network-monitors"
import { chartTimeData, cn, parseSemVer } from "@/lib/utils"
import type { ChartOptions, ChartTimes, NetworkMonitorRecord, NetworkMonitorStatsRecord } from "@/types"
import { SystemMultiSelect } from "./monitor-dialog"

type Protocol = NetworkMonitorRecord["protocol"]

const PROTOCOLS: Protocol[] = ["icmp", "tcp", "http", "dns"]
const PROTOCOL_LABELS: Record<Protocol, string> = { icmp: "ICMP", tcp: "TCP", http: "HTTP(S)", dns: "DNS" }
/** All panels share one hover crosshair. */
const SYNC_ID = "network-monitors"
const COLLAPSED_STORAGE_KEY = "besz-monitor-charts-collapsed"

function loadCollapsed(): Set<string> {
	try {
		return new Set(JSON.parse(localStorage.getItem(COLLAPSED_STORAGE_KEY) || "[]"))
	} catch {
		return new Set()
	}
}

/**
 * Dashboard of response and loss charts for every monitor, one section per protocol.
 * `monitors` arrive already filtered by the table's text and status filters.
 */
export default function NetworkMonitorsCharts({ monitors }: { monitors: NetworkMonitorRecord[] }) {
	// Realtime isn't offered (see useProtocolMonitorStats), so fall back to 1h.
	const [chartTimeStore] = useState(() => {
		const defaultTime = getUserChartTime()
		return atom<ChartTimes>(defaultTime === "1m" ? "1h" : defaultTime)
	})
	const chartTime = useStore(chartTimeStore)
	const direction = useStore($direction)
	const [selectedSystemIds, setSelectedSystemIds] = useState<Set<string>>(() => new Set())
	const [collapsed, setCollapsed] = useState(loadCollapsed)

	const toggleSection = useCallback((protocol: Protocol) => {
		setCollapsed((prev) => {
			const next = new Set(prev)
			if (!next.delete(protocol)) next.add(protocol)
			try {
				localStorage.setItem(COLLAPSED_STORAGE_KEY, JSON.stringify([...next]))
			} catch {
				// collapse state just isn't remembered
			}
			return next
		})
	}, [])

	const chartData = useMemo<ChartOptions>(
		() => ({ agentVersion: parseSemVer(), orientation: direction === "rtl" ? "right" : "left", chartTime }),
		[direction, chartTime]
	)

	const systemIdsWithMonitors = useMemo(() => [...new Set(monitors.map((m) => m.system))], [monitors])

	const monitorsByProtocol = useMemo(() => {
		const byProtocol = new Map<Protocol, NetworkMonitorRecord[]>()
		for (const monitor of monitors) {
			if (selectedSystemIds.size && !selectedSystemIds.has(monitor.system)) continue
			const group = byProtocol.get(monitor.protocol) ?? []
			group.push(monitor)
			byProtocol.set(monitor.protocol, group)
		}
		return byProtocol
	}, [monitors, selectedSystemIds])

	const protocols = PROTOCOLS.filter((protocol) => monitorsByProtocol.has(protocol))

	return (
		<div className="grid gap-4">
			<div className="flex flex-wrap items-center gap-2">
				<SystemMultiSelect
					id="monitor-charts-systems"
					className="flex-1 min-w-0 basis-full sm:basis-0 sm:max-w-80 bg-card"
					systemIds={systemIdsWithMonitors}
					selectedSystemIds={selectedSystemIds}
					onChange={setSelectedSystemIds}
					disabled={systemIdsWithMonitors.length < 2}
				/>
				<ChartTimeSelect
					className="bg-card flex-1 min-w-0 basis-full sm:basis-0 sm:max-w-60 sm:ms-auto"
					agentVersion={chartData.agentVersion}
					chartTimeStore={chartTimeStore}
					allowRealtime={false}
				/>
			</div>
			{protocols.length ? (
				protocols.map((protocol) => (
					<ProtocolSection
						key={protocol}
						protocol={protocol}
						monitors={monitorsByProtocol.get(protocol) ?? []}
						chartData={chartData}
						collapsed={collapsed.has(protocol)}
						onToggle={toggleSection}
					/>
				))
			) : (
				<p className="py-12 text-center text-sm text-muted-foreground">
					<Trans>No results.</Trans>
				</p>
			)}
		</div>
	)
}

const ProtocolSection = memo(function ProtocolSection({
	protocol,
	monitors,
	chartData,
	collapsed,
	onToggle,
}: {
	protocol: Protocol
	monitors: NetworkMonitorRecord[]
	chartData: ChartOptions
	collapsed: boolean
	onToggle: (protocol: Protocol) => void
}) {
	// Stats cover the whole protocol, so filtering monitors doesn't refetch.
	const monitorStats = useProtocolMonitorStats(protocol, chartData.chartTime, !collapsed)

	// One chart per target, with a line for each system that probes it.
	const targets = useMemo(() => {
		const byTarget = new Map<string, NetworkMonitorRecord[]>()
		for (const monitor of monitors) {
			const key = getMonitorIdentityKey(monitor)
			const group = byTarget.get(key) ?? []
			group.push(monitor)
			byTarget.set(key, group)
		}
		const groups = [...byTarget.entries()].map(([key, group]) => ({ key, monitors: group }))
		// Titles tell apart targets that differ only by DNS server or URL scheme.
		const titles = getMonitorLabels(
			groups.map(({ monitors }) => monitors[0]),
			(_, target) => target
		)
		return groups
			.map((group) => ({ ...group, title: titles.get(group.monitors[0].id) ?? group.monitors[0].target }))
			.sort((a, b) => a.title.localeCompare(b.title, undefined, { numeric: true }))
	}, [monitors])

	return (
		<section className="grid gap-3">
			<div className="flex flex-wrap items-center gap-x-3 gap-y-1">
				<Button
					variant="ghost"
					className="-ms-2 px-2 gap-2 font-semibold"
					aria-expanded={!collapsed}
					onClick={() => onToggle(protocol)}
				>
					<ChevronDownIcon className={cn("size-4 transition-transform", { "-rotate-90": collapsed })} />
					{PROTOCOL_LABELS[protocol]}
					<span className="font-normal text-muted-foreground tabular-nums">{targets.length}</span>
				</Button>
			</div>
			{!collapsed &&
				targets.map(({ key, title, monitors }) => (
					<TargetCharts key={key} title={title} monitors={monitors} monitorStats={monitorStats} chartData={chartData} />
				))}
		</section>
	)
})

/** Response and loss charts for one target, with a line for each system that probes it. */
const TargetCharts = memo(function TargetCharts({
	title,
	monitors,
	monitorStats,
	chartData,
}: {
	title: string
	monitors: NetworkMonitorRecord[]
	monitorStats: NetworkMonitorStatsRecord[]
	chartData: ChartOptions
}) {
	const systems = useStore($allSystemsById)
	const getLabel = useCallback((m: NetworkMonitorRecord) => systems[m.system]?.name ?? m.system, [systems])

	// Keep only this target's records, so a single line isn't broken up by other monitors' timestamps.
	const stats = useMemo(() => {
		if (monitors.length === 1) {
			return withMonitorGaps(monitorStats, monitors[0], chartTimeData[chartData.chartTime].expectedInterval)
		}
		return monitorStats.filter((record) => monitors.some((m) => record.stats?.[m.id] != null))
	}, [monitorStats, monitors, chartData.chartTime])
	const empty = stats.length === 0
	const legend = monitors.length > 1 && monitors.length < 10

	return (
		<div className="grid xl:grid-cols-2 gap-4">
			<ResponseChart
				monitorStats={stats}
				grid
				monitors={monitors}
				chartData={chartData}
				empty={empty}
				getLabel={getLabel}
				showFilter={false}
				legend={legend}
				syncId={SYNC_ID}
				titlePrefix={title}
			/>
			<LossChart
				monitorStats={stats}
				grid
				monitors={monitors}
				chartData={chartData}
				empty={empty}
				getLabel={getLabel}
				showFilter={false}
				legend={legend}
				syncId={SYNC_ID}
				titlePrefix={title}
			/>
		</div>
	)
})
