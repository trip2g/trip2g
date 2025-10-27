namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminNoteAsset($id: Int64!) {
			admin {
				noteAsset(id: $id) {
					id
					absolutePath
					fileName
					size
					createdAt
					url
				}
			}
		}
	`)

	export class $trip2g_admin_noteasset_show extends $.$trip2g_admin_noteasset_show {
		@$mol_mem
		data() {
			const res = request({
				id: this.asset_id()
			})

			if (!res.admin.noteAsset) {
				throw new Error(`Note asset with ID ${this.asset_id()} not found`)
			}

			return res.admin.noteAsset
		}

		is_image() {
			const path = this.data().absolutePath
			return /\.(png|jpg|jpeg|gif|webp)$/i.test(path)
		}

		override url(): string {
			return this.data().url
		}

		preview(): $mol_view {
			if (this.is_image()) {
				return this.ImagePreview()
			}

			return this.LinkPreview()
		}
	}
}
