namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query SiteSearch($input: SearchInput!) {
			search(input: $input) {
				nodes {
					highlightedTitle
					highlightedContent
					id: url
					score
					matchOrigin
				}
			}
		}
	`)

	const OPEN = 'open'

	export class $trip2g_user_search extends $.$trip2g_user_search {
		@$mol_mem
		static state(next?: boolean) {
			if (next === undefined) {
				return this.$.$mol_state_arg.value('search') === OPEN
			}

			this.$.$mol_state_arg.value('search', next ? OPEN : null)

			return next
		}

		static toggle() {
			return this.state(!this.state())
		}

		override sub() {
			if ($trip2g_user_search.state()) {
				const p = this.Panel()
				new this.$.$mol_after_frame(() => {
					const q = p.Search().Query()
					q.focused(true)
					q.selection([0, q.value().length])
				})
				return [ p ]
			}

			return []
		}
	}

	export class $trip2g_user_search_toggle extends $.$trip2g_user_search_toggle {
		override state(next?: boolean) {
			return $trip2g_user_search.state(next)
		}
	}

	export class $trip2g_user_search_panel extends $.$trip2g_user_search_panel {
		is_too_short() {
			return this.query().trim().length < 3
		}

		@$mol_mem
		data() {
			const query = this.query().trim()

			if (this.is_too_short()) {
				return $trip2g_graphql_make_map([])
			}

			// debounce
			this.$.$mol_wait_timeout( 1000 )

			const res = request({
				input: { query },
			})

			return $trip2g_graphql_make_map(res.search.nodes)
		}

		@$mol_mem
		query(next?: string) {
			return this.$.$mol_state_arg.value('q', next) || ''
		}

		is_vector_only(): boolean {
			const data = this.data()
			if (data.size() === 0) return false
			const origins = data.map((id: any) => data.get(id).matchOrigin)
			return !origins.some((o: string) => o !== 'VECTOR')
		}

		override results() {
			const data = this.data()
			if (this.is_too_short()) {
				return []
			}

			const items: any[] = [this.ResultCount()]
			if (this.is_vector_only()) {
				items.push(this.VectorWarning())
			}
			items.push(...data.map(id => this.ResultItem(id)))
			items.push(this.CloseButton())
			return items
		}

		override close() {
			$trip2g_user_search.state(false)
		}

		override result_count() {
			const size = this.data().size()
			if (size === 0) {
				return this.messages().no_results
			}

			return `${ this.messages().results_count } ${size}`
		}

		override result_title( id: any ) {
			return this.data().get(id).highlightedTitle || ''
		}

		override result_content( id: any ) {
			return this.data().get(id).highlightedContent.map(html => {
				const view = new this.$.$trip2g_user_search_dimmer()
				view.html = () => html
				return view
			})
		}

		override result_link( id: any ): string {
			return this.data().get(id).id
		}

		override result_score( id: any ): string {
			const item = this.data().get(id)
			const data = this.data()
			const scores = data.map( ( itemId: any ) => data.get( itemId ).score )
			const maxScore = Math.max( ...scores )
			const normalized = maxScore > 0 ? Math.round( item.score / maxScore * 10 ) : 0

			const origin = item.matchOrigin
			const badge = origin === 'VECTOR' ? ' V' : origin === 'HYBRID' ? ' M' : ''
			return `${ super.result_score(id) } ${ normalized }/10${ badge }`
		}
	}

	export class $trip2g_user_search_dimmer extends $.$trip2g_user_search_dimmer {
		@$mol_mem
		override haystack(): string {
			if (!this.html()) return ''

			return this.html().replace(/<[^>]+>/g, '')
		}

		@$mol_mem
		override needle(): string {
			if (!this.html()) return ''
			
			// extract all <mark>...</mark> contents as search needles and join by space
			const needles = Array.from(this.html().matchAll(/<mark>(.*?)<\/mark>/g)).map(m => m[1])
			return needles.join(' ')
		}
	}
}
