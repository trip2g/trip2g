namespace $.$$ {
	const data_request = $trip2g_graphql_request(
		`
			query AdminTelegramAccountDialogs($id: Int64!) {
				admin {
					telegramAccount(id: $id) {
						dialogs {
							id
							username
							title
							type
							publishTags {
								id
							}
							publishInstantTags {
								id
							}
						}
					}
				}
			}
		`
	)

	export class $trip2g_admin_telegramaccount_show_dialogs extends $.$trip2g_admin_telegramaccount_show_dialogs {
		@$mol_mem
		data( reset?: null ) {
			const res = data_request({
				id: String(this.account_id()),
			})

			if (!res.admin.telegramAccount) {
				throw new Error('Telegram account not found')
			}

			return $trip2g_graphql_make_map( res.admin.telegramAccount.dialogs )
		}

		@$mol_mem
		override data_rows() {
			const search = this.search().toLowerCase().trim()
			let keys = this.data().keys()

			if (search.length > 1) {
				keys = keys.filter(id => {
					const row = this.data().get(id)
					return row.username.toLowerCase().includes(search) ||
						row.title.toLowerCase().includes(search)
				})
			}

			return keys.map( id => this.Row( id ) )
		}

		row( id: any ) {
			return this.data().get( id )
		}

		override row_id( id: any ): string {
			return this.row( id ).id
		}

		override row_username( id: any ): string {
			return this.row( id ).username || '-'
		}

		override row_title( id: any ): string {
			return this.row( id ).title || '-'
		}

		override row_publish_tag_ids( id: any ) {
			return this.row(id).publishTags.map( (tag: any) => tag.id )
		}

		override row_publish_instant_tag_ids( id: any ) {
			return this.row(id).publishInstantTags.map( (tag: any) => tag.id )
		}

		override tags(id: any) {
			return this.Tags(id)
		}

		override instant_tags(id: any) {
			return this.InstantTags(id)
		}

		override import(id: any) {
			if (this.row(id).type === 'channel') {
				return this.Import(id)
			}

			return this.Empty()
		}

		override row_default_import_base_path(id: any): string {
			const r = this.row(id)
			return r.username || r.title || String(r.id)
		}
	}
}
