namespace $ {
	export const $trip2g_graphql = (s: string) => s

	export class $trip2g_graphql_error extends Error {
		constructor(message: string, public detail?: unknown) {
			super(message)
		}
	}

	class cache extends $mol_object2 {
		@$mol_mem_key
		typename(key: string, val?: object) {
			return val
		}

		markTypenames(value: unknown, reset: boolean): void {
			if (Array.isArray(value)) {
				for (const item of value) this.markTypenames(item, reset);
			} else if (value && typeof value === 'object') {
				const obj = value as Record<string, unknown>;
		
				if (typeof obj.__typename === 'string') {
					this.typename(obj.__typename, reset ? obj : undefined);
				}
		
				for (const key in obj) {
					this.markTypenames(obj[key], reset);
				}
			}
		}
	}

	const cacheInstance = new cache()

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

		const isMutation = !!query.match(/^\s+mutation/);

		cacheInstance.markTypenames(res.data, isMutation)

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
