namespace $.$$ {
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
			const res = $trip2g_graphql_request(
				`
					mutation AdminCreateTgBotMutation($input: CreateTgBotInput!) {
						admin {
							data: createTgBot(input: $input) {
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
				`,
				{
					input: {
						token: this.token(),
						description: this.description()
					},
				}
			)

			if( res.admin.data.__typename === 'ErrorPayload' ) {
				this.result( res.admin.data.message )
				return
			}

			if( res.admin.data.__typename === 'CreateTgBotPayload' ) {
				this.tgbot_name( res.admin.data.tgBot.name )
				return
			}

			this.result( 'Unexpected response type' )
		}
	}
}