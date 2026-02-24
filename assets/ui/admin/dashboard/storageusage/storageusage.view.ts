namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminStorageUsage {
			admin {
				storageUsage {
					db {
						current(format: MB)
						limit(format: MB)
					}
					assets {
						current(format: MB)
						limit(format: MB)
					}
				}
			}
		}
	`)

	export class $trip2g_admin_dashboard_storageusage extends $.$trip2g_admin_dashboard_storageusage {
		@$mol_mem
		data(reset?: null) {
			return request().admin.storageUsage
		}

		override db_label(): string {
			const d = this.data().db
			return `${super.db_label()}: ${d.current.toFixed(2)} MB / ${d.limit.toFixed(2)} MB`
		}

		override db_portion(): number {
			const d = this.data().db
			if (d.limit === 0) return 0
			return Math.min(d.current / d.limit, 1)
		}

		override assets_label(): string {
			const d = this.data().assets
			return `${super.assets_label()}: ${d.current.toFixed(2)} MB / ${d.limit.toFixed(2)} MB`
		}

		override assets_portion(): number {
			const d = this.data().assets
			if (d.limit === 0) return 0
			return Math.min(d.current / d.limit, 1)
		}
	}
}
