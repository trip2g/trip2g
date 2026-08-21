namespace $.$$ {
	export class $trip2g_admin_deliverytrace_hop extends $.$trip2g_admin_deliverytrace_hop {
		// The page takes the delivery as one masked fragment ref and reads it here:
		// the fragment is declared next to this component, so what it renders and
		// what it asks the server for stay in one place.
		@$mol_mem
		data() {
			// The prop is nullable (a page can be constructed before its row exists),
			// so unmask hands back a nullable fragment; every field access below would
			// otherwise have to re-check it.
			const hop = $trip2g_admin_deliverytrace_hop_hop_unmask( this.hop() )
			if( !hop ) throw new Error( 'Step has no delivery data' )
			return hop
		}

		override hop_title(): string {
			return `${ this.kind() } #${ this.delivery_id() }`
		}

		override kind(): string {
			return this.data().kind
		}

		override delivery_id(): string {
			return this.data().id.toString()
		}

		override webhook(): string {
			return this.data().webhookId.toString()
		}

		override depth(): string {
			return this.data().depthReached.toString()
		}

		// The root has no cause; every other step names the delivery whose writes
		// triggered it, in the same "<kind>:<id>" form the chain id uses.
		override parent(): string {
			const hop = this.data()
			if( !hop.parentKind || !hop.parentId ) return '-'
			return `${ hop.parentKind }:${ hop.parentId }`
		}

		override status(): string {
			const hop = this.data()
			const response = hop.responseStatus ? ` (${ hop.responseStatus })` : ''
			return `${ hop.status }${ response }`
		}

		// Duration is trip2g's own measurement; the rest is what the agent reported
		// about its run, in units only it knows.
		override spend(): string {
			const hop = this.data()
			const parts: string[] = []
			const costs = $trip2g_admin_costs_text( hop.costs )
			if( costs !== '-' ) parts.push( costs )
			if( hop.durationMs ) parts.push( `${ hop.durationMs } ms` )
			return parts.join( ', ' ) || '-'
		}

		override created_at(): string {
			return new $mol_time_moment( this.data().createdAt ).toString( 'YYYY-MM-DD hh:mm:ss' )
		}

		// One expander per written note, keyed by the version it stored — opening one
		// pulls that version's bytes, so a step that wrote a dozen notes costs
		// nothing until you ask for a specific one.
		@$mol_mem
		override write_rows() {
			return this.data().writes.map( write => this.Write( Number( write.versionId ) ) )
		}

		write( version_id: number ) {
			const write = this.data().writes.find( item => Number( item.versionId ) === version_id )
			if( !write ) throw new Error( `Unknown version ${ version_id }` )
			return write
		}

		override write_path( version_id: number ): string {
			return this.write( version_id ).path
		}

		override write_version_id( version_id: number ): number {
			return version_id
		}

		// The log is addressed by the delivery itself, not by the chain, so it is
		// fetched only when this step is on screen.
		override log_delivery_id(): number {
			return Number( this.data().id )
		}
	}
}
