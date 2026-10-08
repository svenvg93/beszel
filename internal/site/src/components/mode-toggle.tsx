import { t } from "@lingui/core/macro"
import { MoonStarIcon, SunIcon, SunMoonIcon } from "lucide-react"
import { useTheme } from "@/components/theme-provider"
import { Button } from "@/components/ui/button"
import { Tooltip, TooltipContent, TooltipTrigger } from "./ui/tooltip"
import { Trans } from "@lingui/react/macro"
import { cn } from "@/lib/utils"

const themes = ["light", "dark", "system"] as const
const icons = [SunIcon, MoonStarIcon, SunMoonIcon] as const

/** Current theme icon and a function that cycles light -> dark -> system */
export function useThemeCycle() {
	const { theme, setTheme } = useTheme()
	const currentIndex = themes.indexOf(theme)
	return {
		currentIndex,
		Icon: icons[currentIndex],
		cycleTheme: () => setTheme(themes[(currentIndex + 1) % themes.length]),
	}
}

export function ModeToggle() {
	const { currentIndex, Icon, cycleTheme } = useThemeCycle()

	return (
		<Tooltip>
			<TooltipTrigger asChild>
				<Button variant={"ghost"} size="icon" aria-label={t`Switch theme`} onClick={cycleTheme}>
					<Icon
						className={cn(
							"animate-in fade-in spin-in-[-30deg] duration-200",
							currentIndex === 2 ? "size-[1.35rem]" : "size-[1.2rem]"
						)}
					/>
				</Button>
			</TooltipTrigger>
			<TooltipContent>
				<Trans>Switch theme</Trans>
			</TooltipContent>
		</Tooltip>
	)
}
