import { useStore } from "@nanostores/react"
import { syncByNearestTime } from "@/lib/chart-sync"
import { $chartSync } from "@/lib/stores"

/** Shared recharts syncId for all charts when chart sync is enabled */
const chartSyncId = "beszel"

const syncProps = { syncId: chartSyncId, syncMethod: syncByNearestTime } as const

/** Hides the tooltip box on charts that aren't hovered, leaving only the cursor and active dots */
const crosshairClass = "[&:not(:hover)_.recharts-tooltip-wrapper]:hidden"

/** Returns recharts props and container class for the user's chart sync setting */
export function useChartSync() {
	const mode = useStore($chartSync)
	if (mode === "off") {
		return { mode, syncProps: undefined, className: undefined }
	}
	return { mode, syncProps, className: mode === "crosshair" ? crosshairClass : undefined }
}
