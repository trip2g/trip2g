namespace $.$$ {
	export class $trip2g_admin_link extends $.$trip2g_admin_link {
		override sub() {
			const viewer = this.$.$trip2g_auth_viewer.current()

			if (viewer.role === $trip2g_graphql_role.Admin) {
				return super.sub()
			}

			return []
		}
	}
}