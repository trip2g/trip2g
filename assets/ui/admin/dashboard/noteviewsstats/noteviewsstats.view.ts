namespace $.$$ {
	export class $trip2g_admin_dashboard_noteviewsstats_ruler extends $.$trip2g_admin_dashboard_noteviewsstats_ruler {
		// start the gridlines where the labels end so they do not strike the numbers
		override curve() {
			return super.curve().replace( /M 0 /g, `M ${ this.gap().x.min } ` )
		}
	}

	export class $trip2g_admin_dashboard_noteviewsstats extends $.$trip2g_admin_dashboard_noteviewsstats {
		@$mol_mem
		data( reset?: null ) {
			return $trip2g_admin_dashboard_noteviewsstats_stats().admin.noteViewsStats
		}

		override day_indexes() {
			return this.data().days.map( ( _, i ) => i )
		}

		override day_views() {
			return this.data().days.map( d => d.views )
		}

		override day_labels() {
			return this.data().days.map( d => `${ d.day }: ${ d.views }` )
		}

		override day_axis_labels() {
			const lang = this.$.$mol_locale.lang()
			return this.data().days.map( d => new Date( d.day ).toLocaleDateString( lang, { day: 'numeric', month: 'short', timeZone: 'UTC' } ) )
		}

		override note_rows() {
			const rows = this.data().topNotes.map( ( _, i ) => this.Note_row( i ) )
			return rows.length ? [ this.Notes_head(), ...rows ] : [ this.Empty() ]
		}

		override reader_rows() {
			const rows = this.data().topReaders.map( ( _, i ) => this.Reader_row( i ) )
			return rows.length ? [ this.Readers_head(), ...rows ] : [ this.Empty() ]
		}

		override source_rows() {
			const rows = this.data().bySource.map( ( _, i ) => this.Source_row( i ) )
			return rows.length ? [ this.Sources_head(), ...rows ] : [ this.Empty() ]
		}

		override Note_cell( index: number ) {
			return this.data().topNotes[ index ].note ? this.Note_link( index ) : this.Note_plain( index )
		}

		override note_id( index: number ) {
			return this.data().topNotes[ index ].note?.id ?? ''
		}

		override note_title( index: number ) {
			const row = this.data().topNotes[ index ]
			return row.note?.title || row.path
		}

		override note_views( index: number ) {
			return String( this.data().topNotes[ index ].views )
		}

		override note_readers( index: number ) {
			return String( this.data().topNotes[ index ].readers )
		}

		override reader_id( index: number ) {
			return String( this.data().topReaders[ index ].userId )
		}

		override reader_title( index: number ) {
			const row = this.data().topReaders[ index ]
			return row.email || `${ this.reader_fallback() } #${ row.userId }`
		}

		override reader_views( index: number ) {
			return String( this.data().topReaders[ index ].views )
		}

		override source_name( index: number ) {
			return this.data().bySource[ index ].createdVia
		}

		override source_views( index: number ) {
			return String( this.data().bySource[ index ].views )
		}

		override source_readers( index: number ) {
			return String( this.data().bySource[ index ].readers )
		}
	}
}
