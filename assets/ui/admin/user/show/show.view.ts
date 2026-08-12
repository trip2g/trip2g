namespace $.$$ {
	export class $trip2g_admin_user_show extends $.$trip2g_admin_user_show {
		action() {
			return this.$.$mol_state_arg.value('action') || 'view';
		}

		override body() {
			if (this.action() === 'update') {
				return [this.UpdateForm()]
			}

			return super.body()
		}

		override tools() {
			const items = [...super.tools()]

			if (!this.data().admin) {
				items.unshift( this.CreateAdminLink() )
			}

			return items
		}

		@$mol_mem
		data() {
			const res = $trip2g_admin_user_show_data({
				id: this.user_id()
			})

			if (!res.admin.user) {
				throw new Error('User not found')
			}

			return res.admin.user
		}

		override user_id_string(): string {
			return this.data().id.toString()
		}

		override user_email(): string {
			return this.data().email || '-'
		}

		override user_hat_email(): string {
			return this.data().email || ''
		}

		override hat_link_allowed(): boolean {
			return this.HatLink().generate_allowed()
		}

		override hat_link_generate() {
			this.HatLink().generate()
		}

		override user_created_at(): string {
			const m = new $mol_time_moment( this.data().createdAt )
			return m.toString( 'YYYY-MM-DD hh:mm:ss' )
		}
	}
}
