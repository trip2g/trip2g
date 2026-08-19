namespace $.$$ {
	export class $trip2g_user_space_home extends $.$trip2g_user_space_home {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_user_space_home_list()
			return res.viewer.user ?? null
		}

		override user_email(): string {
			return this.data()?.email ?? '—'
		}

		notes() {
			return this.data()?.favoriteNotes ?? []
		}

		override favorite_rows() {
			if (this.notes().length === 0) {
				return [this.EmptyFavorites()]
			}

			return this.notes().map((_, index) => this.Favorite(index))
		}

		override favorite_title(index: any): string {
			return this.notes()[index].title
		}

		override favorite_uri(index: any): string {
			return this.notes()[index].url
		}
	}
}
