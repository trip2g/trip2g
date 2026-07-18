namespace $.$$ {
	export class $trip2g_admin_frontmatterpatch_update extends $.$trip2g_admin_frontmatterpatch_update {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_admin_frontmatterpatch_update_data({
				id: this.frontmatterpatch_id()
			})

			const patch = res.admin.frontmatterPatch
			if (!patch) {
				throw new Error('Frontmatter Patch not found')
			}

			return patch
		}

		@$mol_mem
		override description(next?: string): string {
			if (next !== undefined) {
				return next
			}
			return this.data().description
		}

		@$mol_mem
		override include_patterns_text(next?: string): string {
			if (next !== undefined) {
				return next
			}
			// Join array patterns into newline-separated text
			return this.data().includePatterns.join('\n')
		}

		@$mol_mem
		override exclude_patterns_text(next?: string): string {
			if (next !== undefined) {
				return next
			}
			const excludePatterns = this.data().excludePatterns
			return excludePatterns ? excludePatterns.join('\n') : ''
		}

		@$mol_mem
		override jsonnet(next?: string): string {
			if (next !== undefined) {
				return next
			}
			return this.data().jsonnet
		}

		@$mol_mem
		override priority(next?: number): number {
			if (next !== undefined) {
				return next
			}
			return this.data().priority
		}

		@$mol_mem
		override enabled(next?: boolean): boolean {
			if (next !== undefined) {
				return next
			}
			return this.data().enabled
		}

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

			const res = $trip2g_admin_frontmatterpatch_update_save({
				input: {
					id: this.frontmatterpatch_id(),
					description: this.description(),
					includePatterns,
					excludePatterns,
					jsonnet: this.jsonnet(),
					priority: this.priority(),
					enabled: this.enabled(),
				},
			})

			if (res.admin.payload.__typename === 'ErrorPayload') {
				this.result(res.admin.payload.message)
				return
			}

			if (res.admin.payload.__typename === 'UpdateFrontmatterPatchPayload') {
				this.result('Frontmatter Patch updated successfully')
				// Navigate back to show page
				this.$.$mol_state_arg.value('action', '')
				return
			}

			this.result('Unexpected response type')
		}
	}
}
