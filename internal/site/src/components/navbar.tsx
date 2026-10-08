import { Trans } from "@lingui/react/macro"
import { useStore } from "@nanostores/react"
import { PlusIcon, SearchIcon } from "lucide-react"
import { Fragment, lazy, Suspense, useState } from "react"
import {
	Breadcrumb,
	BreadcrumbItem,
	BreadcrumbLink,
	BreadcrumbList,
	BreadcrumbPage,
	BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { isReadOnlyUser } from "@/lib/api"
import { cn } from "@/lib/utils"
import { $allSystemsById } from "@/lib/stores"
import { AddSystemDialog } from "./add-system"
import { getNavGroups } from "./app-sidebar"
import { ModeToggle } from "./mode-toggle"
import { $router, Link } from "./router"

const CommandPalette = lazy(() => import("./command-palette"))

const isMac = navigator.platform.toUpperCase().indexOf("MAC") >= 0

export default function Navbar() {
	const [addSystemDialogOpen, setAddSystemDialogOpen] = useState(false)
	const [commandPaletteOpen, setCommandPaletteOpen] = useState(false)

	return (
		<header className="flex h-14 md:h-16 shrink-0 items-center gap-2 my-2 min-w-0">
			<Suspense>
				<CommandPalette open={commandPaletteOpen} setOpen={setCommandPaletteOpen} />
			</Suspense>
			<AddSystemDialog open={addSystemDialogOpen} setOpen={setAddSystemDialogOpen} />

			<SidebarTrigger className="-ms-1" />
			<Separator orientation="vertical" className="me-2 h-4" />

			<Breadcrumbs />

			<div className="ms-auto flex items-center">
				<Button
					variant="outline"
					className="hidden lg:block text-sm text-muted-foreground px-4 me-1"
					onClick={() => setCommandPaletteOpen(true)}
				>
					<span className="flex items-center">
						<SearchIcon className="me-1.5 h-4 w-4" />
						<Trans>Search</Trans>
						<span className="flex items-center ms-3.5">
							<Kbd>{isMac ? "⌘" : "Ctrl"}</Kbd>
							<Kbd>K</Kbd>
						</span>
					</span>
				</Button>
				<Button
					variant="ghost"
					size="icon"
					className="lg:hidden"
					aria-label="Search"
					onClick={() => setCommandPaletteOpen(true)}
				>
					<SearchIcon className="h-[1.2rem] w-[1.2rem]" />
				</Button>
				<ModeToggle />
				{!isReadOnlyUser() && (
					<Button variant="outline" className="flex gap-1 ms-2" onClick={() => setAddSystemDialogOpen(true)}>
						<PlusIcon className="h-4 w-4 -ms-1" />
						<span className="hidden sm:inline">
							<Trans>Add System</Trans>
						</span>
					</Button>
				)}
			</div>
		</header>
	)
}

/** Group > page (> system name) trail for the current route, based on the sidebar navigation */
function Breadcrumbs() {
	const page = useStore($router)
	const systemId = page?.route === "system" ? page.params.id : ""
	const system = useStore($allSystemsById, { keys: [systemId] })[systemId]

	let crumbs: { title: React.ReactNode; href?: string }[] = []
	for (const group of getNavGroups()) {
		const item = group.items.find((item) => item.isActive(page))
		if (item) {
			crumbs = [{ title: group.label }, { title: item.title, href: item.href }]
			break
		}
	}
	if (system) {
		crumbs.push({ title: system.name })
	}
	if (!crumbs.length) {
		return null
	}

	return (
		<Breadcrumb className="min-w-0">
			<BreadcrumbList className="flex-nowrap">
				{crumbs.map((crumb, i) => {
					const last = i === crumbs.length - 1
					return (
						<Fragment key={i}>
							{i > 0 && <BreadcrumbSeparator className="hidden md:block" />}
							<BreadcrumbItem className={cn(!last && "hidden md:inline-flex", last && "min-w-0")}>
								{last ? (
									<BreadcrumbPage className="truncate">{crumb.title}</BreadcrumbPage>
								) : crumb.href ? (
									<BreadcrumbLink asChild>
										<Link href={crumb.href}>{crumb.title}</Link>
									</BreadcrumbLink>
								) : (
									crumb.title
								)}
							</BreadcrumbItem>
						</Fragment>
					)
				})}
			</BreadcrumbList>
		</Breadcrumb>
	)
}

const Kbd = ({ children }: { children: React.ReactNode }) => (
	<kbd className="pointer-events-none inline-flex h-5 select-none items-center gap-1 rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground opacity-100">
		{children}
	</kbd>
)
