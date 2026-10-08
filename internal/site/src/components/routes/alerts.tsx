import { useLingui } from "@lingui/react/macro"
import { useEffect } from "react"
import AlertsHistoryTable from "@/components/alerts-history-table"

export default function Alerts() {
	const { t } = useLingui()

	useEffect(() => {
		document.title = `${t`Alert History`} / Beszel`
	}, [t])

	return <AlertsHistoryTable />
}
