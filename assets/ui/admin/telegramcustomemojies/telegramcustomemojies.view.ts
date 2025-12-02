namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminListTelegramCustomEmojies {
			admin {
				allTelegramCustomEmojies {
					nodes {
						id
						base64Uri
					}
				}
			}
		}
	`)

	export class $trip2g_admin_telegramcustomemojies extends $.$trip2g_admin_telegramcustomemojies {
		static player(): any {
			return $mol_import.script( 'https://unpkg.com/@lottiefiles/lottie-player@latest/dist/lottie-player.js' )
		}

		@$mol_mem
		data(reset?: null) {
			const res = request()
			return $trip2g_graphql_make_map(res.admin.allTelegramCustomEmojies.nodes)
		}

		override rows() {
			return this.data().map(key => this.Row(key))
		}

		override row_src( id: any ): string {
			console.log($trip2g_admin_telegramcustomemojies.player())
			return this.data().get(id).base64Uri
		}

		override row_label( id: any ): string {
			return `![emoji](tg://emoji?id=${id})`
		}
	}
}
