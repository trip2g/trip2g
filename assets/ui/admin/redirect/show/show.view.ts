namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */`
		query AdminShowRedirect($id: Int64!) {
			admin {
				redirect(id: $id) {
					id
					createdAt
					pattern
					ignoreCase
					isRegex
					target
					createdBy {
						id
						email
					}
				}
			}
		}
	`)

	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminDeleteRedirectMutation($input: DeleteRedirectInput!) {
			admin {
				payload: deleteRedirect(input: $input) {
					__typename
					... on DeleteRedirectPayload {
						id
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_redirect_show extends $.$trip2g_admin_redirect_show {
		action() {
			return this.$.$mol_state_arg.value('action') || 'view';
		}

		@$mol_mem
		data(reset?: null) {
			const res = request({ id: this.redirect_id() })

			if (!res.admin.redirect) {
				throw new Error('Redirect not found')
			}

			return res.admin.redirect
		}

		override body() {
			if (this.action() === 'update') {
				return [this.UpdateForm()]
			}

			return [this.RedirectDetails(), this.DeleteResult()]
		}

		redirect_pattern(): string {
			return this.data().pattern
		}

		redirect_target(): string {
			return this.data().target
		}

		redirect_type(): string {
			return this.data().isRegex ? 'Regex Pattern' : 'Simple Pattern'
		}

		redirect_case(): string {
			return this.data().ignoreCase ? 'Case Insensitive' : 'Case Sensitive'
		}

		redirect_created_at(): string {
			const m = new $mol_time_moment(this.data().createdAt)
			return m.toString('YYYY-MM-DD HH:mm')
		}

		delete() {
			const res = mutate({
				input: {
					id: this.redirect_id()
				},
			})

			if (res.admin.payload.__typename === 'ErrorPayload') {
				this.delete_result(res.admin.payload.message)
				return
			}

			if (res.admin.payload.__typename === 'DeleteRedirectPayload') {
				this.delete_result('Redirect deleted successfully')
				// Navigate back to catalog
				this.$.$mol_state_arg.value('id', '')
				return
			}

			this.delete_result('Unexpected response type')
		}
	}
}