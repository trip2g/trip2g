namespace $.$$ {
	type Hop = {
		kind: string
		id: number
		status: string
		createdAt: string
		startedAt?: string | null
		completedAt?: string | null
		durationMs?: number | null
	}

	const rowHeight = 1.75 // rem, must match the Bar height + gap in the css

	export class $trip2g_admin_deliverytrace_timeline extends $.$trip2g_admin_deliverytrace_timeline {
		hop_list(): readonly Hop[] {
			return this.hops() as readonly Hop[]
		}

		@$mol_mem
		override bars() {
			return this.hop_list().map( ( _, index ) => this.Bar( index ) )
		}

		override style() {
			return {
				...super.style(),
				height: `${ this.hop_list().length * rowHeight + 1 }rem`,
			}
		}

		hop( index: number ): Hop {
			return this.hop_list()[ index ]
		}

		time( value?: string | null ): number {
			return value ? new Date( value ).getTime() : 0
		}

		// A delivery occupies the clock from the moment its row appeared until its
		// run ended. duration_ms is the authoritative run length — completed_at is
		// only second-granular — so the run is measured forward from its start.
		span( index: number ): { from: number, run: number, to: number } {
			const hop = this.hop( index )
			const from = this.time( hop.createdAt )
			const run = this.time( hop.startedAt ) || from
			const duration = hop.durationMs ?? 0
			const to = duration > 0 ? run + duration : ( this.time( hop.completedAt ) || run )
			return { from, run, to }
		}

		@$mol_mem
		bounds(): { start: number, span: number } {
			const spans = this.hop_list().map( ( _, index ) => this.span( index ) )
			if( !spans.length ) return { start: 0, span: 1 }
			const start = Math.min( ...spans.map( s => s.from ) )
			const end = Math.max( ...spans.map( s => s.to ) )
			return { start, span: Math.max( 1, end - start ) }
		}

		percent( value: number ): string {
			return `${ ( value * 100 ).toFixed( 2 ) }%`
		}

		override bar_style( index: number ) {
			const { start, span } = this.bounds()
			const { from, to } = this.span( index )
			return {
				left: this.percent( ( from - start ) / span ),
				width: this.percent( Math.max( to - from, 1 ) / span ),
				top: `${ index * rowHeight + 0.5 }rem`,
			}
		}

		// The solid part is the run; whatever precedes it inside the bar is the
		// wait in the queue.
		override bar_run_style( index: number ) {
			const { from, run, to } = this.span( index )
			const total = Math.max( to - from, 1 )
			return { width: this.percent( ( to - run ) / total ) }
		}

		override bar_label( index: number ): string {
			const hop = this.hop( index )
			return `${ hop.kind } #${ hop.id }`
		}

		override bar_hint( index: number ): string {
			const hop = this.hop( index )
			const { from, run, to } = this.span( index )
			const wait = Math.max( 0, run - from )
			return `${ hop.kind } #${ hop.id } — ${ hop.status }, waited ${ wait } ms, ran ${ to - run } ms`
		}

		override bar_arg( index: number ) {
			const hop = this.hop( index )
			return { hop: `${ hop.kind }:${ hop.id }` }
		}
	}
}
