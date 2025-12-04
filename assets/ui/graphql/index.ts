namespace $ {
	export const $trip2g_graphql = (s: string) => s

	type GraphQLError = {
		message: string
		path: string[]
	}

	export class $trip2g_graphql_error extends Error {
		constructor(message: string, public detail?: GraphQLError[]) {
			if (detail) {
				for (let err of detail) {
					message += `. ${err.message}`
				}
			}

			super(message)
		}
	}

	class reset_query_marker extends $mol_object2 {
		@$mol_mem
		query_marker(val?: number) {
			return val || 0
		}
	}

	class cache extends $mol_object2 {
		@$mol_mem_key
		typename(key: string, val?: number) {
			return val || 0
		}

		is_root(tn: string) {
			return tn === 'Query' || tn === 'Mutation' || tn === 'AdminQuery' || tn === 'AdminMutation'
		}

		markTypenames(value: unknown, reset: boolean): void {
			if (Array.isArray(value)) {
				for (const item of value) this.markTypenames(item, reset)
			} else if (value && typeof value === 'object') {
				const obj = value as Record<string, unknown>

				if (typeof obj.__typename === 'string' && !this.is_root(obj.__typename)) {
					const nv = reset ? this.typename(obj.__typename) + 1 : undefined
					this.typename(obj.__typename, nv)
				}

				for (const key in obj) {
					this.markTypenames(obj[key], reset)
				}
			}
		}
	}

	type RequestOptions = {
		resetCache?: boolean // true by default
	}

	const reset_marker = new reset_query_marker()

	export function $trip2g_graphql_raw_request(query: string) {
		// replace @exportType directives
		query = query.replace(/@exportType\s*(\([^)]*\))?\s*/g, '')

		return (variables?: any, opts?: RequestOptions): any => {
			const res = $.$mol_fetch.json('/graphql', {
				method: 'POST',
				credentials: 'include',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ query, variables }),
			}) as { data?: any; errors?: any[] }

			if (res.errors) {
				throw new $.$trip2g_graphql_error('GraphQL Error', res.errors)
			}

			const isMutation = !!query.match(/^\s+mutation/)

			if (opts?.resetCache !== false) {
				if (isMutation) {
					reset_marker.query_marker(reset_marker.query_marker() + 1)
				} else {
					reset_marker.query_marker()
				}
			}

			return res.data
		}
	}

	export function $trip2g_graphql_make_map<K extends PropertyKey, T extends { id: K }>(rows: T[]) {
		const keys = rows.map(row => row.id.toString())
		const map = new Map<string, T>(rows.map(row => [row.id.toString(), row] as [string, T]))

		return {
			keys: () => keys,
			size: () => keys.length,
			get: (key: string) => {
				const val = map.get(key.replace(/^key/, ''))
				if (!val) {
					throw new Error(`Key ${key} not found`)
				}

				return val
			},
			mapKeys<V>(fn: (key: string) => V): Record<string, V> {
				const out: Record<string, V> = {}
				for (const key of keys) {
					out[`key${key}`] = fn(key)
				}
				return out
			},
			map<V>(fn: (key: string, idx: number) => V, empty?: V): V[] {
				if (map.size === 0 && empty) {
					return [empty]
				}

				const out: V[] = []
				let i = 0;
				for (const key of keys) {
					out.push(fn(key, i++))
				}
				return out
			},
		}
	}

	export function $trip2g_graphql_raw_subscription(query: string, variables?: any): any {
		throw new Error('Not implemented')
	}
}
