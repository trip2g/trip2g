namespace $.$$ {
	const submit_request = $trip2g_graphql_request(
		`
			mutation AdminCreateFrontmatterPatchMutation($input: CreateFrontmatterPatchInput!) {
				admin {
					data: createFrontmatterPatch(input: $input) {
						__typename
						... on CreateFrontmatterPatchPayload {
							frontmatterPatch {
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

	export class $trip2g_admin_frontmatterpatch_create extends $.$trip2g_admin_frontmatterpatch_create {
		override description_bid(): string {
			const description = this.description()
			if (!description.trim()) {
				return 'Description is required'
			}
			return ''
		}

		override include_patterns_bid(): string {
			const text = this.include_patterns_text()
			if (!text.trim()) {
				return 'At least one include pattern is required'
			}
			return ''
		}

		override exclude_patterns_bid(): string {
			// Exclude patterns are optional
			return ''
		}

		override jsonnet_bid(): string {
			const jsonnet = this.jsonnet()
			if (!jsonnet.trim()) {
				return 'Jsonnet expression is required'
			}
			return ''
		}

		override priority_bid(): string {
			const priority = this.priority()
			if (priority === null || priority === undefined) {
				return 'Priority is required'
			}
			return ''
		}

		// Parse patterns from textarea (one per line, trim whitespace)
		include_patterns(): string[] {
			const text = this.include_patterns_text()
			return text
				.split('\n')
				.map(line => line.trim())
				.filter(line => line.length > 0)
		}

		exclude_patterns(): string[] {
			const text = this.exclude_patterns_text()
			if (!text.trim()) {
				return []
			}
			return text
				.split('\n')
				.map(line => line.trim())
				.filter(line => line.length > 0)
		}

		submit() {
			const includePatterns = this.include_patterns()
			const excludePatterns = this.exclude_patterns()

			const res = submit_request({
				input: {
					description: this.description(),
					includePatterns,
					excludePatterns,
					jsonnet: this.jsonnet(),
					priority: this.priority() || 0,
					enabled: this.enabled(),
				},
			})

			if (res.admin.data.__typename === 'ErrorPayload') {
				this.result(res.admin.data.message)
				return
			}

			if (res.admin.data.__typename === 'CreateFrontmatterPatchPayload') {
				this.after_success(res.admin.data.frontmatterPatch.id)
				return
			}

			this.result('Unexpected response type')
		}
	}
}
