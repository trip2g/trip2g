namespace $.$$ {
	export class $trip2g_user_space_tokens_buttons_revoke extends $.$trip2g_user_space_tokens_buttons_revoke {
		override handle_click() {
			if (this.token_id() === '') {
				throw new Error('token id is not set')
			}

			const res = $trip2g_user_space_tokens_buttons_revoke_revoke({
				input: {
					id: this.token_id(),
				},
			})

			if (res.revokeUserToken.__typename === 'ErrorPayload') {
				throw new Error(res.revokeUserToken.message)
			}

			// The list and the screen both read from the same query, so telling
			// the owner it is gone is enough to make both of them agree.
			this.revoked(null)
		}
	}
}
