namespace $.$$ {
	// How much of the end always stays visible. Note paths here end in a hash and
	// an extension, which is the part that tells two of them apart.
	const tailLength = 10

	export class $trip2g_admin_path extends $.$trip2g_admin_path {
		// The head shrinks and clips, the tail never does, so a path that does not
		// fit loses its middle rather than its name — and no measuring is needed,
		// the browser does it at whatever width the column happens to have.
		override head(): string {
			const path = this.path()
			return path.length > tailLength ? path.slice( 0, -tailLength ) : path
		}

		override tail(): string {
			const path = this.path()
			return path.length > tailLength ? path.slice( -tailLength ) : ''
		}
	}
}
