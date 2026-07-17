namespace $.$$ {
	export class $trip2g_admin_notfoundpath_show extends $.$trip2g_admin_notfoundpath_show {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_admin_notfoundpath_show_show()

			const notFoundPath = res.admin.allNotFoundPaths.nodes.find(n => n.id === this.notfoundpath_id())
			if (!notFoundPath) {
				throw new Error('Not Found Path not found')
			}

			return notFoundPath
		}

		notfoundpath_path(): string {
			return this.data().path
		}

		notfoundpath_total_hits(): string {
			return this.data().totalHits.toString()
		}

		notfoundpath_last_hit_at(): string {
			const m = new $mol_time_moment(this.data().lastHitAt)
			return m.toString('YYYY-MM-DD HH:mm')
		}
	}
}