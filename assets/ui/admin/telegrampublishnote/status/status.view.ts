namespace $.$$ {
	export class $trip2g_admin_telegrampublishnote_status extends $.$trip2g_admin_telegrampublishnote_status {
		override status() {
			const v = this.value() as keyof ReturnType<typeof this.statuses>
			return this.statuses()[v] || v
		}
	}
}
