namespace $.$$ {
	export class $trip2g_admin_noteview_button_reembed extends $.$trip2g_admin_noteview_button_reembed {
		reembed( next?: null ) {
			if( next === undefined ) return null

			// force: the stored content hash covers title, content and the model,
			// so a chunking change leaves every note looking up to date. The
			// per-chunk hashes still decide what is actually re-sent to the model,
			// and the jobs drain through the global queue at its own concurrency.
			const res = $trip2g_admin_noteview_button_reembed_mutate({ input: { force: true } })

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				this.status( res.admin.payload.message )
				return null
			}

			const { enqueued, upToDate, totalNotes } = res.admin.payload
			this.status( `${ enqueued } / ${ totalNotes } enqueued, ${ upToDate } up to date` )

			return null
		}
	}
}
