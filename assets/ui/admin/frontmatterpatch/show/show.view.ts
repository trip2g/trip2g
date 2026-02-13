namespace $.$$ {
	const data_request = $trip2g_graphql_request(
		`
			query FrontmatterPatch($id: Int64!) {
				admin {
					frontmatterPatch(id: $id) {
						id
						includePatterns
						excludePatterns
						jsonnet
						priority
						description
						enabled
						createdAt
						updatedAt
						createdBy {
							id
							email
						}
					}
				}
			}
		`
	)

	export class $trip2g_admin_frontmatterpatch_show extends $.$trip2g_admin_frontmatterpatch_show {
		action() {
			return this.$.$mol_state_arg.value( 'action' ) || 'view'
		}

		@$mol_mem
		data( reset?: null ) {
			const res = data_request({
				id: this.frontmatterpatch_id()
			})

			const patch = res.admin.frontmatterPatch
			if( !patch ) {
				throw new Error( 'Frontmatter Patch not found' )
			}

			return patch
		}

		override body() {
			if( this.action() === 'update' ) {
				return [ this.UpdateForm() ]
			}

			return [ this.PatchDetails() ]
		}

		patch_id(): string {
			return this.data().id.toString()
		}

		patch_description(): string {
			return this.data().description || '-'
		}

		patch_priority(): string {
			return this.data().priority.toString()
		}

		patch_enabled(): string {
			return this.data().enabled ? 'Yes' : 'No'
		}

		patch_include_patterns(): string {
			const patterns = this.data().includePatterns
			if (!patterns || patterns.length === 0) return '-'
			return patterns.join('\n')
		}

		patch_exclude_patterns(): string {
			const patterns = this.data().excludePatterns
			if (!patterns || patterns.length === 0) return '-'
			return patterns.join('\n')
		}

		patch_jsonnet(): string {
			return this.data().jsonnet
		}

		patch_created_at(): string {
			const m = new $mol_time_moment( this.data().createdAt )
			return m.toString( 'YYYY-MM-DD HH:mm' )
		}

		patch_created_by(): string {
			const createdBy = this.data().createdBy
			return createdBy ? createdBy.email : '-'
		}

		patch_updated_at(): string {
			const m = new $mol_time_moment( this.data().updatedAt )
			return m.toString( 'YYYY-MM-DD HH:mm' )
		}
	}
}
