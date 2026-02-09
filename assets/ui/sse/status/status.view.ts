namespace $.$$ {
	export class $trip2g_sse_status extends $.$trip2g_sse_status {

		override status() {
			const sse = this.sse()
			if (sse.error_message()) return 'error'
			if (sse.ready()) return 'open'
			return 'connecting'
		}

		override title_formatted() {
			const message = this.status_message()[this.status()] || this.status_message().error
			return super.title().replace('{status}', message)
		}
	}
}
