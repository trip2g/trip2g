namespace $.$$ {
	export class $trip2g_admin_usersubgraphaccesses_show extends $.$trip2g_admin_usersubgraphaccesses_show {
		@$mol_mem
		all_data(reset?: null) {
			const res = $trip2g_admin_usersubgraphaccesses_show_data({ id: this.access_id() })

			return res.admin
		}

		data() {
			const data = this.all_data()
			if (!data.userSubgraphAccess) {
				throw new Error('UserSubgraphAccess not found')
			}

			return data.userSubgraphAccess
		}

		override body() {
			if (this.data().revoke) {
				return [this.Revoked(), this.Form()]
			}

			return [this.Form(), this.Revoke_form()]
		}

		@$mol_mem
		expires_at_moment(next?: any) {
			if (next === undefined) {
				const raw = this.data().expiresAt

				if (raw) {
					return new $mol_time_moment(raw)
				}

				return null
			}

			if (next) {
				next = new $mol_time_moment().merge(next)
			}

			return next
		}

		@$mol_mem
		subgraph_id(next?: number): number {
			return next === undefined ? this.data().subgraphId : next
		}

		submit() {
			const res = $trip2g_admin_usersubgraphaccesses_show_save({
				input: {
					id: this.access_id(),
					expiresAt: $trip2g_moment_toserver(this.expires_at_moment()),
					subgraphId: this.subgraph_id(),
				},
			})

			if (res.admin.payload.__typename === 'ErrorPayload') {
				this.result(res.admin.payload.message)
			}
		}

		override revoke() {
			const res = $trip2g_admin_usersubgraphaccesses_show_revoke({
				input: {
					id: this.access_id(),
					reason: this.revoke_reason(),
				},
			})

			if (res.admin.payload.__typename === 'ErrorPayload') {
				this.revoke_result(res.admin.payload.message)
			}
		}

		revoke_data() {
			const revoke = this.data().revoke
			if (!revoke) {
				throw new Error('Access is not revoked')
			}

			return revoke
		}

		override revoked_at(): string {
			return new $mol_time_moment(this.revoke_data().createdAt).toString('YYYY-MM-DD hh:mm')
		}

		override revoked_by(): string {
			return this.revoke_data().by.email || `#${this.revoke_data().by.id}`
		}

		override revoked_reason(): string {
			return this.revoke_data().reason || ''
		}
	}
}
