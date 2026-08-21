namespace $.$$ {
	export class $trip2g_admin_deliverytrace_log extends $.$trip2g_admin_deliverytrace_log {
		// Fetched here rather than with the chain: a chain has many hops and this
		// runs only for the one whose log someone opened.
		@$mol_mem
		entries() {
			return $trip2g_admin_deliverytrace_log_log( {
				kind: this.kind(),
				deliveryId: this.delivery_id(),
			} ).admin.deliveryLogs
		}

		@$mol_mem
		override entry_rows() {
			return this.entries().map( ( _, index ) => this.Entry( index ) )
		}

		// ts, level and msg are the only fields every agent means the same thing by,
		// so they are the only ones this line is built from.
		override entry_label( index: number ): string {
			const entry = this.entries()[ index ]
			const time = entry.ts
				? new $mol_time_moment( entry.ts ).toString( 'hh:mm:ss' )
				: '--:--:--'
			return `${ time }  ${ entry.level }  ${ entry.msg }`
		}

		// Whatever the agent attached, pretty-printed and not otherwise touched:
		// this screen does not know what an agent's tools are called, and must not
		// start guessing. Unparseable JSON is shown raw rather than swallowed.
		override entry_data( index: number ): string {
			const data = this.entries()[ index ].data
			if( !data ) return ''
			try {
				return JSON.stringify( JSON.parse( data ), null, 2 )
			} catch {
				return data
			}
		}
	}
}
