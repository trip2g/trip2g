namespace $.$$ {
	export class $trip2g_admin_dashboard_buildinfo extends $.$trip2g_admin_dashboard_buildinfo {
		@$mol_mem
		data( reset?: null ) {
			return $trip2g_admin_dashboard_buildinfo_buildinfo().admin
		}

		override build_git_commit(): string {
			const label = this.data().buildGitCommit || 'unknown'
			return `${super.build_git_commit()}: ${label}`
		}
	}
}
