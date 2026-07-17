namespace $.$$ {
	export class $trip2g_admin_subgraph_show extends $.$trip2g_admin_subgraph_show {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_admin_subgraph_show_data({ id: this.subgraph_id() })

			if (!res.admin.subgraph) {
				throw new Error('Subgraph not found')
			}

			return res.admin.subgraph
		}

		subgraph_name(): string {
			return this.data().name
		}

		@$mol_mem
		subgraph_color(next?: string): string {
			if (next !== undefined) {
				return next
			}

			return this.data().color || ''
		}

		@$mol_mem
		override subgraph_hidden(next?: boolean): boolean {
			if (next !== undefined) {
				return next
			}

			return this.data().hidden
		}

		@$mol_mem
		override subgraph_require_signin(next?: boolean): boolean {
			if (next !== undefined) {
				return next
			}

			return this.data().requireSignin
		}

		submit() {
			const res = $trip2g_admin_subgraph_show_save({
				input: {
					id: this.subgraph_id(),
					color: this.subgraph_color(),
					hidden: this.subgraph_hidden(),
					requireSignin: this.subgraph_require_signin(),
				},
			})

			if (res.admin.payload.__typename === 'ErrorPayload') {
				throw new Error(res.admin.payload.message)
			}

			if (res.admin.payload.__typename === 'UpdateSubgraphPayload') {
				this.subgraph_color(res.admin.payload.subgraph.color || '')
				// this.on_save(res.admin.payload);
				return
			}

			throw new Error('Unknown response type')
		}
	}
}
