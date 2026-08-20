namespace $.$$ {
	export class $trip2g_admin_deliverytrace_show extends $.$trip2g_admin_deliverytrace_show {
		@$mol_mem
		hops( reset?: null ) {
			return $trip2g_admin_deliverytrace_show_data( { trace: this.trace() } ).admin.deliveryTrace
		}

		override trace_title(): string {
			return this.trace() || super.trace_title()
		}

		override hop_rows() {
			return this.hops().map( ( _, index ) => this.HopRow( index ) )
		}

		hop( index: any ) {
			return this.hops()[ index ]
		}

		override hop_kind( index: any ): string {
			return this.hop( index ).kind
		}

		override hop_id( index: any ): string {
			return this.hop( index ).id.toString()
		}

		override hop_webhook( index: any ): string {
			return this.hop( index ).webhookId.toString()
		}

		override hop_depth( index: any ): string {
			return this.hop( index ).depthReached.toString()
		}

		// The root has no cause; every other hop names the delivery whose writes
		// triggered it, in the same "<kind>:<id>" form the trace id uses.
		override hop_parent( index: any ): string {
			const hop = this.hop( index )
			if( !hop.parentKind || !hop.parentId ) return '-'
			return `${ hop.parentKind }:${ hop.parentId }`
		}

		override hop_status( index: any ): string {
			const hop = this.hop( index )
			const response = hop.responseStatus ? ` (${ hop.responseStatus })` : ''
			return `${ hop.status }${ response }`
		}

		override hop_spend( index: any ): string {
			const hop = this.hop( index )
			const parts: string[] = []
			if( hop.tokensUsed ) parts.push( `${ hop.tokensUsed } tok` )
			if( hop.steps ) parts.push( `${ hop.steps } steps` )
			if( hop.durationMs ) parts.push( `${ hop.durationMs } ms` )
			return parts.join( ', ' ) || '-'
		}

		override hop_created_at( index: any ): string {
			return new $mol_time_moment( this.hop( index ).createdAt ).toString( 'YYYY-MM-DD HH:mm:ss' )
		}
	}
}
