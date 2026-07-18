namespace $.$$ {
	function basename(path: string): string {
		return path.split('/').at(-1) ?? path
	}

	export class $trip2g_editor_navigator extends $.$trip2g_editor_navigator {
		@$mol_mem
		override paths(): string[] {
			const res = $trip2g_editor_navigator_list()
			return res.notePaths.map(p => p.value)
		}

		override searching(): boolean {
			return this.query().trim().length > 0
		}

		@$mol_mem
		override ids_tags() {
			const map: Record<string, string[]> = {}
			for (const path of $trip2g_editor_navigator_filter(this.paths(), this.query())) {
				map[path] = [path]
			}
			return map
		}

		item_title(id: readonly string[]): string {
			return basename(id.at(-1) ?? '')
		}

		item_click(id: readonly string[], next?: Event): null {
			if (next !== undefined) {
				this.path(id.at(-1) ?? '')
			}
			return null
		}
	}

	export class $trip2g_editor_navigator_tree extends $.$trip2g_editor_navigator_tree {
		ids_tags_map(): Record<string, string[]> {
			return this.ids_tags() as Record<string, string[]>
		}

		@$mol_mem
		override ids(): string[] {
			return $trip2g_editor_navigator_ids(this.ids_tags_map(), this.path().join('/'))
		}

		@$mol_mem
		override tags(): string[] {
			return $trip2g_editor_navigator_subfolders(
				this.ids(),
				this.ids_tags_map(),
				this.path().join('/'),
				$mol_compare_text(),
			)
		}

		@$mol_mem
		override item_list() {
			const path = this.path()
			const grouped = new Set(this.tags().flatMap(tag => this.Tag_tree(tag).ids()))
			const cmp = $mol_compare_text()
			return this.ids()
				.filter(id => !grouped.has(id))
				.sort((a, b) => cmp(basename(a), basename(b)))
				.map(id => this.Item([...path, id]))
		}

		build_dir_opened_key(path: string): string {
			return `trip2g_editor:navigator:dir_opened:${path}`
		}

		override tag_expanded(id: readonly string[], next?: boolean): boolean {
			const prefix = this.path().join('/')
			const seg = id.join('/')
			const dir = prefix ? `${prefix}/${seg}` : seg
			const key = this.build_dir_opened_key(dir)
			if (next !== undefined) {
				this.$.$mol_state_local.value(key, next)
				return next
			}
			if (this.force_expanded()) return true
			return this.$.$mol_state_local.value(key) ?? false
		}

		@$mol_mem_key
		override Tag_tree(id: any) {
			const obj = new this.$.$trip2g_editor_navigator_tree()
			obj.ids_tags = () => this.ids_tags()
			obj.path = () => this.tag_path(id)
			obj.force_expanded = () => this.force_expanded()
			obj.Item = (key: any) => this.Item(key)
			obj.item_title = (key: any) => this.item_title(key)
			obj.tag_name = (key: any) => this.tag_name(key)
			return obj
		}
	}
}
