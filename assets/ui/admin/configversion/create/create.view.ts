namespace $.$$ {
	export class $trip2g_admin_configversion_create extends $.$trip2g_admin_configversion_create {
		@$mol_mem
		current() {
			const res = $trip2g_graphql_request( `
				query AdminCreateConfigLatestConfig {
					admin {
						latestConfig {
							showDraftVersions
							defaultLayout
							timezone
						}
					}
				}
			`)

			return res.admin.latestConfig
		}

		override submit() {
			const res = $trip2g_graphql_request(
				`
					mutation AdminCreateConfigVersion($input: CreateConfigVersionInput!) {
						admin {
							data: createConfigVersion(input: $input) {
								... on ErrorPayload {
									message
								}
								... on CreateConfigVersionPayload {
									configVersion {
										id
									}
								}
							}
						}
					}
				`,
				{
					input: {
						showDraftVersions: this.show_draft_versions(),
						defaultLayout: this.default_layout(),
						timezone: this.timezone(),
					},
				}
			)

			if( res.admin.data.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.data.message )
			}

			if( res.admin.data.__typename === 'CreateConfigVersionPayload' ) {
				this.result( 'Config version created successfully' )
				return
			}

			throw new Error( 'Unexpected response type' )
		}

		@$mol_mem
		override show_draft_versions( next?: boolean ): boolean {
			return next || this.current().showDraftVersions
		}

		@$mol_mem
		override default_layout( next?: string ): string {
			return next || this.current().defaultLayout || ''
		}

		@$mol_mem
		override timezone( next?: string ): string {
			return next || this.current().timezone || 'UTC'
		}

		override set_my_timezone() {
			const userTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone
			this.timezone( userTimezone )
		}
	}
}
