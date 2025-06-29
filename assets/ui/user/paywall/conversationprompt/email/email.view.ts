namespace $.$$ {
	export class $trip2g_user_paywall_conversationprompt_email extends $.$trip2g_user_paywall_conversationprompt_email {
		@$mol_mem
		done( next?: boolean ) {
			return next || false
		}

		override sub() {
			if( this.done() ) {
				return [ this.SuccessView() ]
			}

			return super.sub()
		}

		override request() {
			const res = $trip2g_graphql_request(
				`
					mutation CreateEmailWaitListRequestMutation ($input: CreateEmailWaitListRequestInput!) {
						createEmailWaitListRequest(input: $input) {
							... on ErrorPayload {
								message
							}
							... on CreateEmailWaitListRequestPayload {
								success
							}
						}
					}
				`,
				{
					input: {
						email: this.email(),
						pathId: $trip2g_user_paywall_page.id(),
					}
				}
			)

			if( res?.createEmailWaitListRequest?.__typename === 'ErrorPayload' ) {
				this.result( res.createEmailWaitListRequest.message )
				return
			}

			if( res?.createEmailWaitListRequest?.__typename === 'CreateEmailWaitListRequestPayload' ) {
				this.done(true)
				return
			}

			throw new Error( 'Unexpected response from server' )
		}
	}
}