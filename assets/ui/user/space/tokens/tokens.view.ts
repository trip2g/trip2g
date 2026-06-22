namespace $.$$ {
	const listRequest = $trip2g_graphql_request(/* GraphQL */ `
		query UserTokensList {
			viewer {
				user {
					tokens {
						id
						name
						tokenPrefix
						scope
						createdAt
						lastUsedAt
						expiresAt
					}
				}
			}
		}
	`)

	const createMutation = $trip2g_graphql_request(/* GraphQL */ `
		mutation CreateUserToken($input: CreateUserTokenInput!) {
			createUserToken(input: $input) {
				... on CreateUserTokenPayload {
					plaintextToken
					token {
						id
						name
						tokenPrefix
						scope
						createdAt
						lastUsedAt
						expiresAt
					}
				}
				... on ErrorPayload {
					message
				}
			}
		}
	`)

	const revokeMutation = $trip2g_graphql_request(/* GraphQL */ `
		mutation RevokeUserToken($input: RevokeUserTokenInput!) {
			revokeUserToken(input: $input) {
				... on RevokeUserTokenPayload {
					token {
						id
					}
				}
				... on ErrorPayload {
					message
				}
			}
		}
	`)

	export class $trip2g_user_space_tokens extends $.$trip2g_user_space_tokens {
		@$mol_mem
		data(reset?: null) {
			const res = listRequest()
			if (!res.viewer.user) {
				return $trip2g_graphql_make_map([])
			}
			return $trip2g_graphql_make_map(res.viewer.user.tokens)
		}

		override token_rows() {
			return this.data().map(key => this.TokenRow(key))
		}

		override page_body() {
			const items: readonly any[] = [ this.TokenList(), this.AddPanel() ]
			return this.plaintext_modal_open()
				? [ ...items, this.Backdrop() ]
				: items
		}

		row(id: any) {
			return this.data().get(id)
		}

		override token_row_content(id: any) {
			return [
				this.TokenHead(id),
				this.TokenMeta(id),
			]
		}

		override token_name(id: any): string {
			return this.row(id).name
		}

		override token_prefix(id: any): string {
			return this.row(id).tokenPrefix
		}

		override token_scope(id: any): string {
			return this.row(id).scope
		}

		override token_created_at(id: any) {
			return this.row(id).createdAt
		}

		override token_last_used_at(id: any) {
			return this.row(id).lastUsedAt
		}

		override token_expires_at(id: any) {
			return this.row(id).expiresAt
		}

		override revoke(id: any) {
			revokeMutation({ input: { id: this.row(id).id } })
			this.data(null)
		}

		@$mol_mem
		override plaintext_token(val?: string): string {
			return val ?? ''
		}

		@$mol_mem
		override plaintext_modal_open(val?: string | null): string | null {
			return val !== undefined ? val : null
		}

		override mcp_url(): string {
			const token = this.plaintext_token()
			if (!token) return ''
			return `${location.origin}/_system/mcp?token=${token}`
		}

		override curl_example(): string {
			const url = this.mcp_url()
			if (!url) return ''
			return `curl -s -X POST '${url}' \\\n  -H 'Content-Type: application/json' \\\n  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'`
		}

		override generate() {
			const name = this.new_name()
			if (!name.trim()) {
				this.generate_result('Token name is required')
				return
			}

			const expiry = this.new_expiry()
			let expiresInDays: number | null = null
			if (expiry === '30d') expiresInDays = 30
			else if (expiry === '90d') expiresInDays = 90
			else if (expiry === '1y') expiresInDays = 365
			else expiresInDays = null

			const res = createMutation({
				input: {
					name,
					expiresInDays,
				},
			})

			const payload = res.createUserToken
			if (!('plaintextToken' in payload)) {
				this.generate_result((payload as any).message ?? 'Unknown error')
				return
			}
			const plaintext = payload.plaintextToken
			this.plaintext_token(plaintext)
			this.plaintext_modal_open('open')
			this.new_name('')
			this.generate_result('')
			this.data(null)
		}

		override close_modal() {
			this.plaintext_modal_open(null)
			this.plaintext_token('')
		}
	}
}
