namespace $.$$ {
	type Handle = {
		dom: HTMLElement
		getDoc(): string
		setDoc(doc: string): void
		setDark(dark: boolean): void
		destroy(): void
	}

	type Lib = {
		create(opts: {
			onChange(doc: string): void
			onLinkClick(offset: number): boolean
		}): Handle
	}

	export class $trip2g_editor_codemirror extends $.$trip2g_editor_codemirror {
		@$mol_mem
		lib(): Lib {
			return $mol_import.script(this.bundle_uri()).$trip2g_cmeditor
		}

		@$mol_mem
		editor(): Handle {
			return this.lib().create({
				onChange: doc => this.value(doc),
				onLinkClick: offset => this.link_click(offset),
			})
		}

		link_click(offset: number): boolean {
			const link = $trip2g_editor_wikilink(this.value(), offset)
			if (!link) return false
			$mol_wire_async(this).link_open(link)
			return true
		}

		@$mol_mem
		doc_sync(): null {
			this.editor().setDoc(this.value())
			return null
		}

		@$mol_mem
		theme_sync(): null {
			this.editor().setDark(!this.$.$mol_lights())
			return null
		}

		override sub() {
			return [this.editor().dom]
		}

		override auto() {
			this.doc_sync()
			this.theme_sync()
		}

		override destructor() {
			$mol_wire_probe(() => this.editor())?.destroy()
			super.destructor()
		}
	}
}
