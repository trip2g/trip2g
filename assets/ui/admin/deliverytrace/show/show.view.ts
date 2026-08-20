namespace $.$$ {
	export class $trip2g_admin_deliverytrace_show extends $.$trip2g_admin_deliverytrace_show {
		@$mol_mem
		hops( reset?: null ) {
			return $trip2g_admin_deliverytrace_show_data( { trace: this.trace() } ).admin.deliveryTrace
		}

		// Keyed by "<kind>:<id>" — the same shape the parent link uses, so a step
		// can be addressed by the id its children point at.
		@$mol_mem
		data() {
			return new Map( this.hops().map( hop => [ `${ hop.kind }:${ hop.id }`, hop ] as const ) )
		}

		@$mol_mem
		spreads(): any {
			return Object.fromEntries( [ ...this.data().keys() ].map( key => [ key, this.ShowPage( key ) ] ) )
		}

		override trace_title(): string {
			return this.trace() || super.trace_title()
		}

		hop( id: any ) {
			const hop = this.data().get( id )
			if( !hop ) throw new Error( `Unknown step ${ id }` )
			return hop
		}

		override hop_ref( id: any ): string {
			return id
		}

		override hop_title( id: any ): string {
			return `${ this.hop_kind( id ) } #${ this.hop_id( id ) }`
		}

		override hop_kind( id: any ): string {
			return this.hop( id ).kind
		}

		override hop_id( id: any ): string {
			return this.hop( id ).id.toString()
		}

		override hop_webhook( id: any ): string {
			return this.hop( id ).webhookId.toString()
		}

		override hop_depth( id: any ): string {
			return this.hop( id ).depthReached.toString()
		}

		// The root has no cause; every other step names the delivery whose writes
		// triggered it, in the same "<kind>:<id>" form the chain id uses.
		override hop_parent( id: any ): string {
			const hop = this.hop( id )
			if( !hop.parentKind || !hop.parentId ) return '-'
			return `${ hop.parentKind }:${ hop.parentId }`
		}

		override hop_status( id: any ): string {
			const hop = this.hop( id )
			const response = hop.responseStatus ? ` (${ hop.responseStatus })` : ''
			return `${ hop.status }${ response }`
		}

		override hop_spend( id: any ): string {
			const hop = this.hop( id )
			const parts: string[] = []
			if( hop.tokensUsed ) parts.push( `${ hop.tokensUsed } tok` )
			if( hop.steps ) parts.push( `${ hop.steps } steps` )
			if( hop.durationMs ) parts.push( `${ hop.durationMs } ms` )
			return parts.join( ', ' ) || '-'
		}

		override hop_created_at( id: any ): string {
			return new $mol_time_moment( this.hop( id ).createdAt ).toString( 'YYYY-MM-DD HH:mm:ss' )
		}

		// Notes this step wrote, from the version attribution.
		override hop_writes( id: any ): readonly string[] {
			return this.hop( id ).writes.map( w => w.path )
		}

		// What set this step off: the notes its parent wrote. A root has no parent
		// — a cron tick or a human edit started it, and neither is a note.
		override hop_trigger( id: any ): readonly string[] {
			const hop = this.hop( id )
			if( !hop.parentKind || !hop.parentId ) return []
			const parent = this.data().get( `${ hop.parentKind }:${ hop.parentId }` )
			return parent ? parent.writes.map( w => w.path ) : []
		}
	}
}
