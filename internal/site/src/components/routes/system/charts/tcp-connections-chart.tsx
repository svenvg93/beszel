import { t } from "@lingui/core/macro"
import LineChartDefault from "@/components/charts/line-chart"
import { decimalString } from "@/lib/utils"
import type { ChartData } from "@/types"
import { ChartCard } from "../chart-card"

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
		>
			<LineChartDefault
				chartData={chartData}
				contentFormatter={(item) => decimalString(item.value, 0)}
				tickFormatter={(value) => decimalString(value, 0)}
				legend={true}
				dataPoints={[
					{
						label: t({ message: `Established`, comment: "TCP connection state" }),
						color: "hsl(217, 91%, 60%)", // Blue
						dataKey: ({ stats }) => stats?.tcp?.[0],
					},
					{
						label: t({ message: `Time wait`, comment: "TCP connection state" }),
						color: "hsl(25, 95%, 53%)", // Orange
						dataKey: ({ stats }) => stats?.tcp?.[1],
					},
					{
						label: t({ message: `Close wait`, comment: "TCP connection state" }),
						color: "hsl(0, 84%, 60%)", // Red
						dataKey: ({ stats }) => stats?.tcp?.[3],
					},
					{
						label: t({ message: `Total`, comment: "Total TCP sockets in any state" }),
						color: "hsl(271, 81%, 60%)", // Purple
						dataKey: ({ stats }) => stats?.tcp?.[2],
					},
				]}
			></LineChartDefault>
		</ChartCard>
	)
}
