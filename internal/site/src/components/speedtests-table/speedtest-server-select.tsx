import { useCallback, useEffect, useRef, useState } from "react"
import { Trans, useLingui } from "@lingui/react/macro"
import { CheckIcon, ChevronDownIcon, GlobeIcon, LoaderCircleIcon, SearchIcon } from "lucide-react"
import { pb } from "@/lib/api"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"

/** An Ookla server as returned by the hub. Zero ID means automatic server selection. */
export interface SpeedtestServer {
	id: number
	name: string
	location: string
}

export const AUTOMATIC_SERVER: SpeedtestServer = { id: 0, name: "", location: "" }

const SEARCH_DEBOUNCE_MS = 300

/**
 * Searchable single-select for Ookla speedtest servers. Opening it lists the
 * servers closest to the hub; typing searches Ookla's server list.
 */
export function SpeedtestServerSelect({
	id,
	value,
	onChange,
	disabled,
}: {
	id: string
	value: SpeedtestServer
	onChange: (server: SpeedtestServer) => void
	disabled?: boolean
}) {
	const { t } = useLingui()
	const [open, setOpen] = useState(false)
	const [search, setSearch] = useState("")
	const [servers, setServers] = useState<SpeedtestServer[]>([])
	const [loading, setLoading] = useState(false)
	const [error, setError] = useState("")
	const searchRef = useRef<HTMLInputElement>(null)
	const contentRef = useRef<HTMLDivElement>(null)
	const focusSearchOnMount = useCallback((node: HTMLInputElement | null) => {
		searchRef.current = node
		if (!node) return
		// Focus after the menu has completed its own initial focus handling.
		const frame = requestAnimationFrame(() => node.focus())
		return () => cancelAnimationFrame(frame)
	}, [])

	// Fetch servers when opened and after typing pauses. Stale responses are ignored.
	useEffect(() => {
		if (!open) return
		let cancelled = false
		const query = search.trim()
		setLoading(true)
		const timeout = setTimeout(
			() => {
				pb.send<SpeedtestServer[]>("/api/beszel/speedtest/servers", {
					query: query ? { search: query } : {},
					requestKey: null,
				})
					.then((result) => {
						if (cancelled) return
						setServers(result ?? [])
						setError("")
					})
					.catch((err: Error) => {
						if (cancelled) return
						setServers([])
						setError(err?.message || t`Failed to load servers.`)
					})
					.finally(() => {
						if (!cancelled) setLoading(false)
					})
			},
			query ? SEARCH_DEBOUNCE_MS : 0
		)
		return () => {
			cancelled = true
			clearTimeout(timeout)
		}
	}, [open, search, t])

	const isAutomatic = value.id === 0
	const label = isAutomatic ? t`Automatic` : [value.name, value.location].filter(Boolean).join(" — ") || `#${value.id}`

	return (
		<DropdownMenu
			open={open}
			onOpenChange={(nextOpen) => {
				setOpen(nextOpen)
				setSearch("")
			}}
		>
			<DropdownMenuTrigger asChild>
				<Button
					id={id}
					disabled={disabled}
					type="button"
					variant="outline"
					className="relative w-full min-w-0 ps-10 pe-10 justify-start font-normal text-start"
				>
					<GlobeIcon className="size-3.5 absolute start-4 top-1/2 -translate-y-1/2 opacity-85" />
					<span className="truncate">{label}</span>
					<ChevronDownIcon className="size-4 absolute end-4 top-1/2 -translate-y-1/2 opacity-50" />
				</Button>
			</DropdownMenuTrigger>
			<DropdownMenuContent
				ref={contentRef}
				onKeyDown={(event) => {
					if (event.key === "Tab") {
						event.preventDefault()
						searchRef.current?.focus()
					}
				}}
				align="start"
				className="w-[var(--radix-dropdown-menu-trigger-width)] max-h-[min(22rem,var(--radix-dropdown-menu-content-available-height))] flex flex-col overflow-hidden"
			>
				<div className="shrink-0 border-b mb-1">
					<div className="flex items-center gap-2 px-2.5">
						<SearchIcon aria-hidden="true" className="size-4 shrink-0 text-muted-foreground" />
						<Input
							ref={focusSearchOnMount}
							value={search}
							onChange={(event) => setSearch(event.target.value)}
							placeholder={t`Search servers`}
							aria-label={t`Search servers`}
							className="h-10 min-w-0 rounded-none border-0 bg-transparent px-0 shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
							onKeyDown={(event) => {
								if (event.key === "Escape") return
								// Keep menu typeahead and form submission from consuming search input.
								event.stopPropagation()
								if (event.key === "Enter") event.preventDefault()
								if (event.key === "ArrowDown" || event.key === "ArrowUp" || event.key === "Tab") {
									event.preventDefault()
									const items = contentRef.current?.querySelectorAll<HTMLElement>(
										'[role^="menuitem"]:not([data-disabled])'
									)
									const index = event.key === "ArrowUp" || event.shiftKey ? (items?.length ?? 1) - 1 : 0
									items?.[index]?.focus()
								}
							}}
						/>
						{loading && <LoaderCircleIcon className="size-4 shrink-0 animate-spin text-muted-foreground" />}
					</div>
				</div>
				<div className="min-h-0 overflow-y-auto">
					{!search.trim() && (
						<ServerItem
							selected={isAutomatic}
							title={t`Automatic`}
							description={t`Pick the best server for each run`}
							onSelect={() => onChange(AUTOMATIC_SERVER)}
						/>
					)}
					{servers.map((server) => (
						<ServerItem
							key={server.id}
							selected={server.id === value.id}
							title={server.name}
							description={server.location}
							onSelect={() => onChange(server)}
						/>
					))}
					{!loading && !error && servers.length === 0 && (
						<output className="block px-2.5 py-3 text-sm text-muted-foreground">
							<Trans>No servers found.</Trans>
						</output>
					)}
					{error && <p className="px-2.5 py-3 text-sm text-red-500">{error}</p>}
				</div>
			</DropdownMenuContent>
		</DropdownMenu>
	)
}

function ServerItem({
	selected,
	title,
	description,
	onSelect,
}: {
	selected: boolean
	title: string
	description: string
	onSelect: () => void
}) {
	return (
		<DropdownMenuItem className="min-w-0 gap-2.5 py-2 ps-2.5" onSelect={onSelect}>
			<CheckIcon className={cn("size-4 shrink-0", !selected && "invisible")} />
			<div className="grid min-w-0">
				<span className="truncate">{title}</span>
				{description && <span className="truncate text-xs text-muted-foreground">{description}</span>}
			</div>
		</DropdownMenuItem>
	)
}
