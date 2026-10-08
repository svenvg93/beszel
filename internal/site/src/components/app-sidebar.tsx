import { Trans } from "@lingui/react/macro"
import { useStore } from "@nanostores/react"
import { getPagePath } from "@nanostores/router"
import {
	AlertOctagonIcon,
	BellIcon,
	ChevronsUpDownIcon,
	CircleArrowUpIcon,
	ContainerIcon,
	DatabaseBackupIcon,
	FileSlidersIcon,
	FingerprintIcon,
	GithubIcon,
	HardDriveIcon,
	HeartPulseIcon,
	LayoutDashboardIcon,
	LogOutIcon,
	LogsIcon,
	NetworkIcon,
	ServerIcon,
	SettingsIcon,
	TagIcon,
	UserIcon,
	UsersIcon,
} from "lucide-react"
import type React from "react"
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuItem,
	DropdownMenuLabel,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
	Sidebar,
	SidebarContent,
	SidebarFooter,
	SidebarGroup,
	SidebarGroupLabel,
	SidebarHeader,
	SidebarMenu,
	SidebarMenuButton,
	SidebarMenuItem,
	SidebarRail,
	useSidebar,
} from "@/components/ui/sidebar"
import { isAdmin, isReadOnlyUser, logOut, pb } from "@/lib/api"
import { $newVersion } from "@/lib/stores"
import { runOnce } from "@/lib/utils"
import { Logo } from "./logo"
import { $router, basePath, Link, prependBasePath } from "./router"

export interface NavItem {
	title: React.ReactNode
	tooltip: string
	href: string
	icon: React.ComponentType<React.SVGProps<SVGSVGElement>>
	/** whether the current page belongs to this item */
	isActive: (page: ReturnType<typeof $router.get>) => boolean
	preload?: () => void
	/** opens outside the app in a new tab */
	external?: boolean
}

export interface NavGroup {
	label: React.ReactNode
	items: NavItem[]
}

const settingsItem = (
	name: string,
	title: React.ReactNode,
	tooltip: string,
	icon: NavItem["icon"],
	preload?: () => void
): NavItem => ({
	title,
	tooltip,
	href: getPagePath($router, "settings", { name }),
	icon,
	isActive: (page) => page?.route === "settings" && page.params.name === name,
	preload,
})

/** Navigation groups shown in the sidebar, also used for the header breadcrumbs */
export function getNavGroups(): NavGroup[] {
	const groups: NavGroup[] = [
		{
			label: <Trans>Systems</Trans>,
			items: [
				{
					title: <Trans>All Systems</Trans>,
					tooltip: "All Systems",
					href: basePath || "/",
					icon: LayoutDashboardIcon,
					isActive: (page) => page?.route === "home" || page?.route === "system",
					preload: runOnce(() => import("@/components/routes/home")),
				},
				{
					title: <Trans>Containers</Trans>,
					tooltip: "Containers",
					href: getPagePath($router, "containers"),
					icon: ContainerIcon,
					isActive: (page) => page?.route === "containers",
				},
				{
					title: <Trans>S.M.A.R.T. Disks</Trans>,
					tooltip: "S.M.A.R.T. Disks",
					href: getPagePath($router, "smart"),
					icon: HardDriveIcon,
					isActive: (page) => page?.route === "smart",
				},
			],
		},
		{
			label: <Trans>Network</Trans>,
			items: [
				{
					title: <Trans>Network Monitors</Trans>,
					tooltip: "Network Monitors",
					href: getPagePath($router, "monitors"),
					icon: NetworkIcon,
					isActive: (page) => page?.route === "monitors",
					preload: runOnce(() => import("@/components/routes/monitors")),
				},
			],
		},
		{
			label: <Trans>Settings</Trans>,
			items: [
				settingsItem(
					"general",
					<Trans comment="Context: General settings">General</Trans>,
					"General",
					SettingsIcon,
					runOnce(() => import("@/components/routes/settings/general"))
				),
				settingsItem("notifications", <Trans>Notifications</Trans>, "Notifications", BellIcon),
				...(isReadOnlyUser()
					? []
					: [settingsItem("tokens", <Trans>Tokens & Fingerprints</Trans>, "Tokens & Fingerprints", FingerprintIcon)]),
				settingsItem("alert-history", <Trans>Alert History</Trans>, "Alert History", AlertOctagonIcon),
				...(isAdmin()
					? [
							settingsItem("heartbeat", <Trans>Heartbeat</Trans>, "Heartbeat", HeartPulseIcon),
							settingsItem("config", <Trans>YAML Config</Trans>, "YAML Config", FileSlidersIcon),
						]
					: []),
			],
		},
	]

	if (isAdmin()) {
		const adminItem = (title: React.ReactNode, tooltip: string, href: string, icon: NavItem["icon"]): NavItem => ({
			title,
			tooltip,
			href: prependBasePath(href),
			icon,
			isActive: () => false,
			external: true,
		})
		groups.push({
			label: <Trans>Admin</Trans>,
			items: [
				adminItem(<Trans>Users</Trans>, "Users", "/_/#/collections?collection=users", UsersIcon),
				adminItem(<Trans>Systems</Trans>, "Systems", "/_/#/collections?collection=systems", ServerIcon),
				adminItem(<Trans>Logs</Trans>, "Logs", "/_/#/logs", LogsIcon),
				adminItem(<Trans>Backups</Trans>, "Backups", "/_/#/settings/backups", DatabaseBackupIcon),
			],
		})
	}

	return groups
}

export function AppSidebar(props: React.ComponentProps<typeof Sidebar>) {
	const page = useStore($router)
	const { isMobile, setOpenMobile } = useSidebar()

	const groups = getNavGroups()

	// close the mobile sheet after navigating
	const onNavigate = () => isMobile && setOpenMobile(false)

	return (
		<Sidebar collapsible="icon" {...props}>
			<SidebarHeader>
				<SidebarMenu>
					<SidebarMenuItem>
						<SidebarMenuButton size="lg" asChild>
							<Link href={basePath || "/"} aria-label="Home" className="group/logo" onClick={onNavigate}>
								<img
									src={prependBasePath("/static/icon.svg")}
									alt=""
									className="hidden size-5 shrink-0 group-data-[collapsible=icon]:block"
								/>
								<Logo className="h-5! w-auto! ms-1 fill-foreground group-data-[collapsible=icon]:hidden" />
							</Link>
						</SidebarMenuButton>
					</SidebarMenuItem>
				</SidebarMenu>
			</SidebarHeader>
			<SidebarContent>
				{groups.map((group, i) => (
					<SidebarGroup key={i}>
						<SidebarGroupLabel>{group.label}</SidebarGroupLabel>
						<SidebarMenu>
							{group.items.map((item) => (
								<SidebarMenuItem key={item.href}>
									<SidebarMenuButton asChild tooltip={item.tooltip} isActive={item.isActive(page)}>
										{item.external ? (
											<a href={item.href} target="_blank" rel="noopener">
												<item.icon />
												<span>{item.title}</span>
											</a>
										) : (
											<Link href={item.href} onMouseEnter={item.preload} onClick={onNavigate}>
												<item.icon />
												<span>{item.title}</span>
											</Link>
										)}
									</SidebarMenuButton>
								</SidebarMenuItem>
							))}
						</SidebarMenu>
					</SidebarGroup>
				))}
			</SidebarContent>
			<SidebarFooter>
				<NavRepoLinks />
				<NavUser />
			</SidebarFooter>
			<SidebarRail />
		</Sidebar>
	)
}

function NavRepoLinks() {
	const newVersion = useStore($newVersion)
	const version = `Beszel ${globalThis.BESZEL.HUB_VERSION}`

	return (
		<SidebarMenu>
			{newVersion?.v && (
				<SidebarMenuItem>
					<SidebarMenuButton asChild size="sm" tooltip={`${newVersion.v} available`}>
						<a href={newVersion.url} target="_blank" rel="noopener" className="text-yellow-500! hover:text-yellow-400!">
							<CircleArrowUpIcon />
							<span>
								<Trans context="New version available">{newVersion.v} available</Trans>
							</span>
						</a>
					</SidebarMenuButton>
				</SidebarMenuItem>
			)}
			<SidebarMenuItem>
				<SidebarMenuButton asChild size="sm" tooltip="GitHub" className="text-muted-foreground">
					<a href="https://github.com/henrygd/beszel" target="_blank" rel="noopener">
						<GithubIcon />
						<span>GitHub</span>
					</a>
				</SidebarMenuButton>
			</SidebarMenuItem>
			<SidebarMenuItem>
				<SidebarMenuButton asChild size="sm" tooltip={version} className="text-muted-foreground">
					<a href="https://github.com/henrygd/beszel/releases" target="_blank" rel="noopener">
						<TagIcon />
						<span>{version}</span>
					</a>
				</SidebarMenuButton>
			</SidebarMenuItem>
		</SidebarMenu>
	)
}

function NavUser() {
	const { isMobile } = useSidebar()
	const email = pb.authStore.record?.email

	return (
		<SidebarMenu>
			<SidebarMenuItem>
				<DropdownMenu>
					<DropdownMenuTrigger asChild>
						<SidebarMenuButton
							size="lg"
							aria-label="User Actions"
							className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
						>
							<div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-accent text-sidebar-accent-foreground">
								<UserIcon className="size-4" />
							</div>
							<span className="flex-1 truncate text-start text-sm">{email}</span>
							<ChevronsUpDownIcon className="ms-auto size-4" />
						</SidebarMenuButton>
					</DropdownMenuTrigger>
					<DropdownMenuContent
						className="w-(--radix-dropdown-menu-trigger-width) min-w-56"
						side={isMobile ? "bottom" : "right"}
						align="end"
						sideOffset={4}
					>
						<DropdownMenuLabel className="truncate font-normal">{email}</DropdownMenuLabel>
						<DropdownMenuSeparator />
						<DropdownMenuGroup>
							<DropdownMenuItem onSelect={logOut}>
								<LogOutIcon className="me-2.5 size-4" />
								<Trans>Log Out</Trans>
							</DropdownMenuItem>
						</DropdownMenuGroup>
					</DropdownMenuContent>
				</DropdownMenu>
			</SidebarMenuItem>
		</SidebarMenu>
	)
}
