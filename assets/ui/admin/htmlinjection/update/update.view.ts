namespace $.$$ {
	const data_request = $trip2g_graphql_request(
		`
			query AdminUpdateDataHtmlInjection($id: Int64!) {
				admin {
					htmlInjection(id: $id) {
						id
						createdAt
						activeFrom
						activeTo
						description
						position
						placement
						content
					}
				}
			}
		`
	)

	const submit_request = $trip2g_graphql_request(
		`
			mutation AdminUpdateHtmlInjection($input: UpdateHtmlInjectionInput!) {
				admin {
					data: updateHtmlInjection(input: $input) {
						... on UpdateHtmlInjectionPayload {
							htmlInjection {
								id
							}
						}
						... on ErrorPayload {
							message
						}
					}
				}
			}
		`
	)

	export class $trip2g_admin_htmlinjection_update extends $.$trip2g_admin_htmlinjection_update {
		@$mol_mem
		data(reset?: null) {
			const res = data_request({
				id: this.htmlinjection_id()
			})

			const injection = res.admin.htmlInjection
			if (!injection) {
				throw new Error('Html Injection not found')
			}

			return injection
		}

		@$mol_mem
		override description(next?: string): string {
			if (next !== undefined) {
				return next
			}
			return this.data().description || ''
		}

		@$mol_mem
		override placement(next?: string): string {
			if (next !== undefined) {
				return next
			}
			return this.data().placement
		}

		@$mol_mem
		override position(next?: number): number {
			if (next !== undefined) {
				return next
			}
			return this.data().position
		}

		@$mol_mem
		override content(next?: string): string {
			if (next !== undefined) {
				return next
			}
			return this.data().content
		}

		@$mol_mem
		override active_from_moment(next?: $mol_time_moment | null): $mol_time_moment | null {
			if (next !== undefined) {
				if (next) {
					next = new $mol_time_moment().merge(next)
				}
				return next
			}

			const activeFrom = this.data().activeFrom
			return activeFrom ? new $mol_time_moment(activeFrom) : null
		}

		@$mol_mem
		override active_to_moment(next?: $mol_time_moment | null): $mol_time_moment | null {
			if (next !== undefined) {
				if (next) {
					next = new $mol_time_moment().merge(next)
				}
				return next
			}

			const activeTo = this.data().activeTo
			return activeTo ? new $mol_time_moment(activeTo) : null
		}

		override description_bid(): string {
			// Description is optional
			return ''
		}

		override placement_bid(): string {
			const placement = this.placement()
			if (!placement) {
				return 'Placement is required'
			}
			return ''
		}

		override position_bid(): string {
			const position = this.position()
			if (position === null || position === undefined) {
				return 'Position is required'
			}
			if (position < 0) {
				return 'Position must be non-negative'
			}
			return ''
		}

		override content_bid(): string {
			const content = this.content()
			if (!content.trim()) {
				return 'Content is required'
			}
			return ''
		}

		submit() {
			const res = submit_request({
				input: {
					id: this.htmlinjection_id(),
					description: this.description(),
					placement: this.placement(),
					position: this.position(),
					content: this.content(),
					activeFrom: $trip2g_moment_toserver(this.active_from_moment()),
					activeTo: $trip2g_moment_toserver(this.active_to_moment())
				},
			})

			if (res.admin.data.__typename === 'ErrorPayload') {
				this.result(res.admin.data.message)
				return
			}

			if (res.admin.data.__typename === 'UpdateHtmlInjectionPayload') {
				this.result('Html Injection updated successfully')
				// Navigate back to show page
				this.$.$mol_state_arg.value('action', '')
				return
			}

			this.result('Unexpected response type')
		}
	}
}
