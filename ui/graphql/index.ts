namespace $ {
	export const $trip2g_graphql = (s: string) => s

	export class $trip2g_graphql_error extends Error {
		constructor(message: string, public detail?: unknown) {
			super(message)
		}
	}

	export function $trip2g_graphql_raw_request(query: string, variables?: any) {
		const res = $.$mol_fetch.json('/graphql', {
			method: 'POST',
			credentials: 'include',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ query, variables }),
		}) as { data?: any; errors?: any[] }

		if (res.errors) {
			throw new $.$trip2g_graphql_error('GraphQL Error', res.errors)
		}

		return res.data
	}

	export function $trip2g_graphql_make_map<K extends PropertyKey, T extends { id: K }>(rows: T[]) {
		const map = new Map<string, T>(rows.map(row => [row.id.toString(), row] as [string, T]))

		return {
			keys: () => map.keys(),
			get: (key: string) => {
				const val = map.get(key)
				if (!val) {
					throw new Error(`Key ${key} not found`)
				}

				return val
			},
			mapKeys<V>(fn: (key: string) => V): Record<string, V> {
				const out: Record<string, V> = {}
				for (const key of map.keys()) {
					out[key] = fn(key)
				}
				return out
			},
		}
	}
}
