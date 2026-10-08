import { useStore } from "@nanostores/react"
import type React from "react"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Separator } from "@/components/ui/separator"
import { isAdmin, isReadOnlyUser } from "@/lib/api"
import { $router, navigate } from "../../router"

interface SidebarNavProps {
	items: {
		href: string
		title: string
		icon?: React.FC<React.SVGProps<SVGSVGElement>>
		admin?: boolean
		noReadOnly?: boolean
		preload?: () => Promise<{ default: React.ComponentType<any> }>
	}[]
}

/** Settings page picker for mobile. On desktop the app sidebar lists the settings pages. */
export function SidebarNav({ items }: SidebarNavProps) {
	const page = useStore($router)

	return (
		<div className="md:hidden">
			<Select onValueChange={navigate} value={page?.path}>
				<SelectTrigger className="w-full my-3.5">
					<SelectValue placeholder="Select page" />
				</SelectTrigger>
				<SelectContent>
					{items.map((item) => {
						if ((item.admin && !isAdmin()) || (item.noReadOnly && isReadOnlyUser())) return null
						return (
							<SelectItem key={item.href} value={item.href}>
								<span className="flex items-center gap-2 truncate">
									{item.icon && <item.icon className="size-4" />}
									<span className="truncate">{item.title}</span>
								</span>
							</SelectItem>
						)
					})}
				</SelectContent>
			</Select>
			<Separator />
		</div>
	)
}
