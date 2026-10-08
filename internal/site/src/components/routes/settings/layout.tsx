import { t } from "@lingui/core/macro"
import { useLingui } from "@lingui/react/macro"
import { useStore } from "@nanostores/react"
import { getPagePath, redirectPage } from "@nanostores/router"
import { BellIcon, FileSlidersIcon, FingerprintIcon, HeartPulseIcon, SettingsIcon } from "lucide-react"
import { lazy, useEffect } from "react"
import { $router } from "@/components/router.tsx"
import { Card, CardContent } from "@/components/ui/card.tsx"
import { toast } from "@/components/ui/use-toast.ts"
import { saveUserSettings } from "@/lib/api"
import { $userSettings } from "@/lib/stores.ts"
import type { UserSettings } from "@/types"
import { SidebarNav } from "./sidebar-nav.tsx"

const generalSettingsImport = () => import("./general.tsx")
const notificationsSettingsImport = () => import("./notifications.tsx")
const configYamlSettingsImport = () => import("./config-yaml.tsx")
const fingerprintsSettingsImport = () => import("./tokens-fingerprints.tsx")
const heartbeatSettingsImport = () => import("./heartbeat.tsx")

const GeneralSettings = lazy(generalSettingsImport)
const NotificationsSettings = lazy(notificationsSettingsImport)
const ConfigYamlSettings = lazy(configYamlSettingsImport)
const FingerprintsSettings = lazy(fingerprintsSettingsImport)
const HeartbeatSettings = lazy(heartbeatSettingsImport)

export async function saveSettings(newSettings: Partial<UserSettings>) {
	try {
		await saveUserSettings(newSettings)
		toast({
			title: t`Settings saved`,
			description: t`Your user settings have been updated.`,
		})
	} catch (e) {
		console.error("save settings", e)
		toast({
			title: t`Failed to save settings`,
			description: t`Check logs for more details.`,
			variant: "destructive",
		})
	}
}

export default function SettingsLayout() {
	const { t } = useLingui()

	const sidebarNavItems = [
		{
			title: t({ message: `General`, comment: "Context: General settings" }),
			href: getPagePath($router, "settings", { name: "general" }),
			icon: SettingsIcon,
		},
		{
			title: t`Notifications`,
			href: getPagePath($router, "settings", { name: "notifications" }),
			icon: BellIcon,
			preload: notificationsSettingsImport,
		},
		{
			title: t`Tokens & Fingerprints`,
			href: getPagePath($router, "settings", { name: "tokens" }),
			icon: FingerprintIcon,
			noReadOnly: true,
			preload: fingerprintsSettingsImport,
		},
		{
			title: t`Heartbeat`,
			href: getPagePath($router, "settings", { name: "heartbeat" }),
			icon: HeartPulseIcon,
			admin: true,
			preload: heartbeatSettingsImport,
		},
		{
			title: t`YAML Config`,
			href: getPagePath($router, "settings", { name: "config" }),
			icon: FileSlidersIcon,
			admin: true,
			preload: configYamlSettingsImport,
		},
	]

	const page = useStore($router)

	// biome-ignore lint/correctness/useExhaustiveDependencies: no dependencies
	useEffect(() => {
		document.title = `${t`Settings`} / Beszel`
		// @ts-expect-error redirect to account page if no page is specified
		const name = page?.params?.name
		if (!name) {
			redirectPage($router, "settings", { name: "general" })
		} else if (name === "alert-history") {
			// alert history moved to its own page
			redirectPage($router, "alerts")
		}
	}, [])

	return (
		<Card className="pt-5 px-4 pb-8 min-h-96 mb-14 sm:pt-6 sm:px-7">
			<CardContent className="p-0">
				<div className="flex flex-col gap-3.5 md:flex-row md:gap-5 lg:gap-12">
					<aside className="md:hidden">
						<SidebarNav items={sidebarNavItems} />
					</aside>
					<div className="flex-1 min-w-0">
						{/* @ts-ignore */}
						<SettingsContent name={page?.params?.name ?? "general"} />
					</div>
				</div>
			</CardContent>
		</Card>
	)
}

function SettingsContent({ name }: { name: string }) {
	const userSettings = useStore($userSettings)

	switch (name) {
		case "general":
			return <GeneralSettings userSettings={userSettings} />
		case "notifications":
			return <NotificationsSettings userSettings={userSettings} />
		case "config":
			return <ConfigYamlSettings />
		case "tokens":
			return <FingerprintsSettings />
		case "heartbeat":
			return <HeartbeatSettings />
	}
}
