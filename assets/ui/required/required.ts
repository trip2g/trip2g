namespace $ {
	export function $trip2g_required<T>(data: T | null): T {
		if (data === null) throw new Error('No data')
		return data
	}
}
