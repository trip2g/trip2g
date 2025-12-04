namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminCreateTgBotMutation($input: CreateTgBotInput!) {
			admin {
				payload: createTgBot(input: $input) {
					__typename
					... on CreateTgBotPayload {
						tgBot {
							id
							name
						}
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_tgbot_create extends $.$trip2g_admin_tgbot_create {
		override body() {
			if( this.tgbot_name() !== '' ) {
				return [ this.TgBotView() ]
			}

			return super.body()
		}

		override token_bid(): string {
			if (this.token().trim() === '') {
				return 'Token is required'
			}

			return ''
		}

		override description_bid(): string {
			if (this.description().trim() === '') {
				return 'Description is required'
			}

			return ''
		}

		override submit() {
			const res = mutate({
				input: {
					token: this.token(),
					description: this.description()
				},
			})

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				this.result( res.admin.payload.message )
				return
			}

			if( res.admin.payload.__typename === 'CreateTgBotPayload' ) {
				this.tgbot_name( res.admin.payload.tgBot.name )
				return
			}

			this.result( 'Unexpected response type' )
		}
	}
}