namespace $.$$ {
	export class $trip2g_editor_content extends $.$trip2g_editor_content {
		override file_title(): string {
			const path = this.path()
			return path ? path.split('/').at(-1) ?? path : 'Editor'
		}

		has_content(): boolean {
			return this.content().length > 0
		}

		@$mol_mem
		override content_blob(): Blob {
			return new Blob([this.content()], { type: 'text/markdown' })
		}

		override download_name(): string {
			const path = this.path()
			const name = path ? path.split('/').at(-1) ?? 'note.md' : 'note.md'
			return name.endsWith('.md') ? name : `${name}.md`
		}
	}
}
