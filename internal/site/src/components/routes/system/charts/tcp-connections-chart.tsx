import { t } from "@lingui/core/macro"
import { MoreHorizontalIcon } from "lucide-react"
import { memo, useMemo, useRef, useState } from "react"
import ChartTimeSelect from "@/components/charts/chart-time-select"
import LineChartDefault, { type DataPoint } from "@/components/charts/line-chart"
import { Button } from "@/components/ui/button"
import { DialogTitle } from "@/components/ui/dialog"
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet"
import { decimalString } from "@/lib/utils"
import type { ChartData, SystemStatsRecord } from "@/types"
import { ChartCard } from "../chart-card"

/** TCP states in display order, with their index in the [established, time_wait, total, close_wait] tuple */
function tcpStates() {
	return [
		{
			label: t({ message: `Established`, comment: "TCP connection state" }),
			color: "hsl(217, 91%, 60%)", // Blue
			index: 0,
		},
		{
			label: t({ message: `Time wait`, comment: "TCP connection state" }),
			color: "hsl(25, 95%, 53%)", // Orange
			index: 1,
		},
		{
			label: t({ message: `Close wait`, comment: "TCP connection state" }),
			color: "hsl(0, 84%, 60%)", // Red
			index: 3,
		},
		{
			label: t({ message: `Total`, comment: "Total TCP sockets in any state" }),
			color: "hsl(271, 81%, 60%)", // Purple
			index: 2,
		},
	]
}

const formatCount = (value: number) => decimalString(value, 0)

export function TcpConnectionsChart({
	chartData,
	grid,
	dataEmpty,
}: {
	chartData: ChartData
	grid: boolean
	dataEmpty: boolean
}) {
	if (!chartData.systemStats?.at(-1)?.stats.tcp) {
		return null
	}
	return (
		<ChartCard
			empty={dataEmpty}
			grid={grid}
			title={t`TCP Connections`}
			description={t`Number of TCP sockets by state`}
			legend={true}
			cornerEl={<TcpConnectionsSheet chartData={chartData} dataEmpty={dataEmpty} grid={grid} />}
		>
			<LineChartDefault
				chartData={chartData}
				contentFormatter={(item) => formatCount(item.value)}
				tickFormatter={formatCount}
				legend={true}
				dataPoints={tcpStates().map(({ label, color, index }) => ({
					label,
					color,
					dataKey: ({ stats }) => stats?.tcp?.[index],
				}))}
			></LineChartDefault>
		</ChartCard>
	)
}

const TcpConnectionsSheet = memo(function TcpConnectionsSheet({
	chartData,
	dataEmpty,
	grid,
}: {
	chartData: ChartData
	dataEmpty: boolean
	grid: boolean
}) {
	const [open, setOpen] = useState(false)
	const hasOpened = useRef(false)
	const latest = chartData.systemStats.at(-1)?.stats?.tcpi
	// busiest interfaces first, with colors spread as in the network sheet
	const interfaces = useMemo(() => {
		const keys = Object.keys(latest ?? {}).sort((a, b) => (latest?.[b]?.[2] ?? 0) - (latest?.[a]?.[2] ?? 0))
		return keys.map((name, i) => ({ name, color: `hsl(${220 + (((i * 360) / keys.length) % 360)}, 70%, 50%)` }))
	}, [latest])
	const showLegend = interfaces.length < 15

	if (open && !hasOpened.current) {
		hasOpened.current = true
	}

	if (!interfaces.length) {
		return null
	}

	return (
		<Sheet open={open} onOpenChange={setOpen}>
			<DialogTitle className="sr-only">{t`TCP connections of public interfaces`}</DialogTitle>
			<SheetTrigger asChild>
				<Button
					title={t`View more`}
					variant="outline"
					size="icon"
					className="shrink-0 max-sm:absolute max-sm:top-0 max-sm:end-0"
				>
					<MoreHorizontalIcon />
				</Button>
			</SheetTrigger>
			{hasOpened.current && (
				<SheetContent aria-describedby={undefined} className="overflow-auto w-200 !max-w-full p-4 sm:p-6">
					<ChartTimeSelect className="w-[calc(100%-2em)] bg-card" agentVersion={chartData.agentVersion} />
					{tcpStates().map(({ label, index }) => (
						<ChartCard
							key={index}
							empty={dataEmpty}
							grid={grid}
							title={label}
							description={t`TCP connections of public interfaces`}
							legend={showLegend}
							className="min-h-auto"
						>
							<LineChartDefault
								chartData={chartData}
								contentFormatter={(item) => formatCount(item.value)}
								tickFormatter={formatCount}
								itemSorter={(a, b) => b.value - a.value}
								legend={showLegend}
								dataPoints={interfaces.map(
									({ name, color }): DataPoint<SystemStatsRecord> => ({
										label: name,
										color,
										// an interface missing from a record had no connections
										dataKey: ({ stats }) => (stats?.tcpi ? (stats.tcpi[name]?.[index] ?? 0) : undefined),
									})
								)}
							/>
						</ChartCard>
					))}
				</SheetContent>
			)}
		</Sheet>
	)
})
