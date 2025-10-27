namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query SiteSearch($input: SearchInput!) {
			search(input: $input) {
				nodes {
					highlightedTitle
					highlightedContent
					id: url
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

		override results() {
			const data = this.data()
			if (this.is_too_short()) {
				return []
			}

			return [
				this.ResultCount(),
				...data.map(id => this.ResultItem(id)),
				this.CloseButton(),
			]
		}

		override close() {
			$trip2g_user_search.state(false)
		}

		override result_count() {
			const size = this.data().size()
			if (size === 0) {
				return 'Результаты не найдены'
			}

			return `Найдено результатов: ${size}`
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
