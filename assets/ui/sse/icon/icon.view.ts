namespace $.$$ {
	export class $trip2g_sse_icon extends $.$trip2g_sse_icon {
		override path() {
			return this.status_icon()[this.status()]?.path() ?? super.path()
		}
	}
}
