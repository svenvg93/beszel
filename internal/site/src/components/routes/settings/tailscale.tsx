import { t } from "@lingui/core/macro"
import { Trans } from "@lingui/react/macro"
import { redirectPage } from "@nanostores/router"
import { ExternalLinkIcon, TriangleAlertIcon } from "lucide-react"
import { useEffect, useState } from "react"
import { $router } from "@/components/router"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { toast } from "@/components/ui/use-toast"
import { isAdmin, pb } from "@/lib/api"
import { cn } from "@/lib/utils"

interface TailscaleStatus {
	enabled: boolean
	funnel?: boolean
	backendState?: string
	authURL?: string
	version?: string
	tailnet?: string
	url?: string
	ips?: string[]
	health?: string[]
}

const refreshInterval = 60_000

export default function TailscaleSettings() {
	const [status, setStatus] = useState<TailscaleStatus | null>(null)
	const [isLoading, setIsLoading] = useState(true)

	if (!isAdmin()) {
		redirectPage($router, "settings", { name: "general" })
	}

	useEffect(() => {
		let stopped = false
		async function fetchStatus() {
			try {
				const res = await pb.send<TailscaleStatus>("/api/beszel/tailscale-status", { requestKey: null })
				if (!stopped) setStatus(res)
			} catch (error: unknown) {
				toast({
					title: t`Error`,
					description: (error as Error).message,
					variant: "destructive",
				})
			} finally {
				setIsLoading(false)
			}
		}
		fetchStatus()
		const interval = setInterval(fetchStatus, refreshInterval)
		return () => {
			stopped = true
			clearInterval(interval)
		}
	}, [])

	return (
		<div>
			<div>
				<h3 className="text-xl font-medium mb-2">Tailscale</h3>
				<p className="text-sm text-muted-foreground leading-relaxed">
					<Trans>Serve the hub on your tailnet with automatic HTTPS, without a separate Tailscale client.</Trans>
				</p>
			</div>
			<Separator className="my-4" />

			{status?.enabled ? <EnabledState status={status} /> : <NotEnabledState isLoading={isLoading} />}
		</div>
	)
}

function EnabledState({ status }: { status: TailscaleStatus }) {
	const running = status.backendState === "Running"

	return (
		<div className="space-y-5">
			<div className="flex flex-wrap items-center gap-2">
				<Badge variant={running ? "success" : status.backendState === "NeedsLogin" ? "warning" : "secondary"}>
					{status.backendState}
				</Badge>
			</div>

			{status.authURL && (
				<div className="rounded-md border border-yellow-500/40 bg-yellow-500/10 px-3 py-2.5 text-sm">
					<Trans>This node needs to be authorized.</Trans>{" "}
					<a href={status.authURL} target="_blank" rel="noopener noreferrer" className="font-medium underline">
						<Trans>Log in to Tailscale</Trans>
					</a>
				</div>
			)}

			{status.health && status.health.length > 0 && (
				<div className="grid gap-1.5">
					{status.health.map((msg) => (
						<p key={msg} className="flex gap-2 text-sm text-muted-foreground">
							<TriangleAlertIcon className="size-4 shrink-0 mt-0.5 text-yellow-500" />
							{msg}
						</p>
					))}
				</div>
			)}

			<div className="grid gap-4 sm:grid-cols-2">
				<div>
					<p className="text-sm font-medium mb-0.5">URL</p>
					{status.url ? (
						<a
							href={status.url}
							target="_blank"
							rel="noopener noreferrer"
							className="text-sm text-muted-foreground font-mono break-all hover:underline inline-flex items-center gap-1"
						>
							{status.url}
							<ExternalLinkIcon className="size-3.5 shrink-0" />
						</a>
					) : (
						<p className="text-sm text-muted-foreground">-</p>
					)}
				</div>
				<div>
					<p className="text-sm font-medium mb-1">Funnel</p>
					<Badge variant={status.funnel ? "success" : "secondary"}>
						{status.funnel ? <Trans>Enabled</Trans> : <Trans>Disabled</Trans>}
					</Badge>
				</div>
				<ConfigItem label={t`Tailnet`} value={status.tailnet || "-"} />
				<ConfigItem label={t`Tailscale IPs`} value={status.ips?.join(", ") || "-"} mono />
				<ConfigItem label={t`Version`} value={status.version?.split("-")[0] || "-"} />
			</div>
		</div>
	)
}

function NotEnabledState({ isLoading }: { isLoading?: boolean }) {
	return (
		<div className={cn("grid gap-4", isLoading && "animate-pulse")}>
			<div>
				<p className="text-sm text-muted-foreground leading-relaxed mb-3">
					<Trans>Set the following environment variables on your Beszel hub to enable Tailscale:</Trans>
				</p>
				<div className="grid gap-2.5">
					<EnvVarItem name="TS_HOSTNAME" description={t`Hostname on your tailnet (required)`} example="beszel" />
					<EnvVarItem
						name="TS_AUTHKEY"
						description={t`Auth key for the first login. Without it, a login URL is shown here and in the logs.`}
						example="tskey-auth-xxxx"
					/>
					<EnvVarItem
						name="TS_FUNNEL"
						description={t`Also expose the hub to the public internet with Tailscale Funnel`}
						example="true"
					/>
				</div>
			</div>
			<p className="text-sm text-muted-foreground leading-relaxed">
				<Trans>After setting the environment variables, restart your Beszel hub for changes to take effect.</Trans>
			</p>
		</div>
	)
}

function ConfigItem({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
	return (
		<div>
			<p className="text-sm font-medium mb-0.5">{label}</p>
			<p className={cn("text-sm text-muted-foreground break-all", mono && "font-mono")}>{value}</p>
		</div>
	)
}

function EnvVarItem({ name, description, example }: { name: string; description: string; example: string }) {
	return (
		<div className="bg-muted/50 rounded-md px-3 py-2.5 grid gap-1.5">
			<code className="text-sm font-mono text-primary font-medium leading-tight">{name}</code>
			<p className="text-sm text-muted-foreground">{description}</p>
			<p className="text-xs text-muted-foreground">
				<Trans>Example:</Trans> <code className="font-mono">{example}</code>
			</p>
		</div>
	)
}
