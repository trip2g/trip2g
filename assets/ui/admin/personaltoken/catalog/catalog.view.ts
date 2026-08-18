namespace $.$$ {
	export class $trip2g_admin_personaltoken_catalog extends $.$trip2g_admin_personaltoken_catalog {
		// No user_id in the arg means every token issued by everyone; with one,
		// the same screen narrows to that user without a second page existing.
		@$mol_mem
		override user_id(): number {
			return Number(this.$.$mol_state_arg.value('user_id') ?? 0)
		}

		@$mol_mem
		data(reset?: null) {
			const userId = this.user_id()
			const res = $trip2g_admin_personaltoken_catalog_list({
				filter: userId > 0 ? { userId } : {},
			})

			return $trip2g_graphql_make_map(res.admin.personalTokens.nodes)
		}

		@$mol_mem
		spreads(): any {
			return {
				add: this.AddForm(),
				...this.data().mapKeys(key => this.Content(key)),
			}
		}

		@$mol_mem
		override spread_ids_filtered() {
			return this.spread_ids().filter(id => id !== 'add')
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

		override row_owner(id: any): string {
			return this.row(id).user?.email ?? '-'
		}

		override row_last_used_at(id: any): string {
			return this.row(id).lastUsedAt ?? ''
		}

		override row_expires_at(id: any): string {
			return this.row(id).expiresAt ?? ''
		}

		override row_revoked_at(id: any): string {
			return this.row(id).revokedAt ?? ''
		}
	}
}
