import { useEffect } from "react"
import SmartTable from "@/components/routes/system/smart-table"

export default function Smart() {
	useEffect(() => {
		document.title = `S.M.A.R.T. / Beszel`
	}, [])

	return (
		<>
			<SmartTable />
		</>
	)
}
