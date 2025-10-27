namespace $.$$ {
	const mutate = $trip2g_graphql_request( /* GraphQL */ `
		mutation AdminBanUser($input: BanUserInput!) {
			admin {
				banUser(input: $input) {
					... on BanUserPayload {
						__typename
						user { id, __typename }
					}
					... on ErrorPayload {
						__typename
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_user_banuser extends $.$trip2g_admin_user_banuser {
		@$mol_mem
		reason( next?: string ): string {
			return next ?? ''
		}

		submit() {
			const res = mutate( {
				input: {
					userId: parseInt( this.$.$mol_state_arg.value( 'ban_id' ) || '0', 10 ),
					reason: this.reason(),
				},
			} )

			if( res.admin.banUser.__typename === 'ErrorPayload' ) {
				this.result( res.admin.banUser.message )
				return
			}

			this.$.$mol_state_arg.value( 'ban_id', null )
		}
	}
}
