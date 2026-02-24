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

	function formatEntry(label: string, current: number, limit: number): string {
		const curr = current.toFixed(2)
		if (limit === 0) return `${label}: ${curr} MB`
		return `${label}: ${curr} MB / ${limit.toFixed(2)} MB`
	}

	export class $trip2g_admin_dashboard_storageusage extends $.$trip2g_admin_dashboard_storageusage {
		@$mol_mem
		data(reset?: null) {
			return request().admin.storageUsage
		}

		override db_label(): string {
			const d = this.data().db
			return formatEntry(super.db_label(), d.current, d.limit)
		}

		override assets_label(): string {
			const d = this.data().assets
			return formatEntry(super.assets_label(), d.current, d.limit)
		}
	}
}
