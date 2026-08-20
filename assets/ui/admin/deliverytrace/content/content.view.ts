namespace $.$$ {
	export class $trip2g_admin_deliverytrace_content extends $.$trip2g_admin_deliverytrace_content {
		// Fetched here rather than with the chain: a chain can write dozens of notes
		// and this runs only for the one whose content someone opened.
		@$mol_mem
		override text(): string {
			const version = $trip2g_admin_deliverytrace_content_data( { versionId: this.version_id() } )
				.admin.noteVersion
			return version?.content ?? ''
		}
	}
}
