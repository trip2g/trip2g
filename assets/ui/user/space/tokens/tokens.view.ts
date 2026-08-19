namespace $.$$ {
	export class $trip2g_user_space_tokens extends $.$trip2g_user_space_tokens {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_user_space_tokens_list()
			if (!res.viewer.user) {
				return $trip2g_graphql_make_map([])
			}

			return $trip2g_graphql_make_map(res.viewer.user.tokens)
		}

		override token_rows() {
			if (this.data().size() === 0) {
				return [this.Empty()]
			}

			return this.data().map(key => this.Row(key))
		}

		row(id: any) {
			return this.data().get(id)
		}

		override row_id(id: any): string {
			return this.row(id).id
		}

		override row_name(id: any): string {
			return this.row(id).name
		}

		override row_prefix(id: any): string {
			return this.row(id).tokenPrefix
		}

		// A token has one date worth showing: when it runs out, or when it was
		// revoked if that already happened. The two never matter at once, and a
		// revoked token's expiry is a date that will never arrive.
		override row_state_title(id: any): string {
			return this.row(id).revokedAt ? this.state_label_revoked() : this.state_label_expires()
		}

		override row_state_at(id: any) {
			return this.row(id).revokedAt ?? this.row(id).expiresAt
		}
	}
}
