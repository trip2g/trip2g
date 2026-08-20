namespace $.$$ {
	export class $trip2g_admin_deliverytrace_show extends $.$trip2g_admin_deliverytrace_show {
		@$mol_mem
		hops() {
			return $trip2g_admin_deliverytrace_show_data( { trace: this.trace() } ).admin.deliveryTrace
		}

		// Keyed by "<kind>:<id>" — the same shape the parent link uses, so a step
		// can be addressed by the id its children point at. The rows arrive masked;
		// the list unmasks them for its own columns and hands the ref itself to the
		// step page.
		@$mol_mem
		data() {
			return new Map( this.hops().map( ref => {
				const hop = $trip2g_admin_deliverytrace_hop_hop_unmask( ref )
				return [ `${ hop.kind }:${ hop.id }`, { ref, hop } ] as const
			} ) )
		}

		@$mol_mem
		spreads(): any {
			return Object.fromEntries( [ ...this.data().keys() ].map( key => [ key, this.ShowPage( key ) ] ) )
		}

		override trace_title(): string {
			return this.trace() || super.trace_title()
		}

		row( id: any ) {
			const row = this.data().get( id )
			if( !row ) throw new Error( `Unknown step ${ id }` )
			return row
		}

		override hop( id: any ) {
			return this.row( id ).ref
		}

		override hop_ref( id: any ): string {
			return id
		}

		override hop_status( id: any ): string {
			const hop = this.row( id ).hop
			const response = hop.responseStatus ? ` (${ hop.responseStatus })` : ''
			return `${ hop.status }${ response }`
		}

		override hop_spend( id: any ): string {
			const hop = this.row( id ).hop
			const parts: string[] = []
			const costs = $trip2g_admin_costs_text( hop.costs )
			if( costs !== '-' ) parts.push( costs )
			if( hop.durationMs ) parts.push( `${ hop.durationMs } ms` )
			return parts.join( ', ' ) || '-'
		}

		// Notes this step wrote, from the version attribution.
		override hop_writes( id: any ): readonly string[] {
			return this.row( id ).hop.writes.map( write => write.path )
		}

		// What set this step off: the notes its parent wrote. A root has no parent
		// — a cron tick or a human edit started it, and neither is a note.
		override hop_trigger( id: any ): readonly string[] {
			const hop = this.row( id ).hop
			if( !hop.parentKind || !hop.parentId ) return []
			const parent = this.data().get( `${ hop.parentKind }:${ hop.parentId }` )
			return parent ? parent.hop.writes.map( write => write.path ) : []
		}
	}
}
