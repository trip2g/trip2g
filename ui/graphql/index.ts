namespace $ {
	export const $trip2g_graphql = (s: string) => s

	export class $trip2g_graphql_error extends Error {
		constructor(message: string, public detail?: unknown) {
			for (let err of detail) {
				message += `. ${err.message}`;
			}

			super(message)
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

	const cacheInstance = new cache()

	function inject_typenames(source: string) {
		let out = ''
		let i = 0
		const n = source.length

		let state: 'default' | 'string' | 'blockString' | 'comment' = 'default'
		let quoteChar: '"' | "'" | null = null // для string / blockString
		let blockQuoteLen = 0 // 1 или 3 кавычки

		while (i < n) {
			const c = source[i]
			const next2 = source.slice(i, i + 2)
			const next3 = source.slice(i, i + 3)

			/* ---------- переходы состояний ---------- */
			if (state === 'default') {
				if (c === '#') {
					// начало комментария
					state = 'comment'
					out += c
					i++
					continue
				}
				if (next3 === '"""' || next3 === "'''") {
					// начало блок-строки
					state = 'blockString'
					quoteChar = c as '"' | "'"
					blockQuoteLen = 3
					out += next3
					i += 3
					continue
				}
				if (c === '"' || c === "'") {
					// начало обычной строки
					state = 'string'
					quoteChar = c as '"' | "'"
					blockQuoteLen = 1
					out += c
					i++
					continue
				}
				if (c === '{') {
					// наша целевая точка
					out += c
					i++

					// пропускаем пробелы/переводы строк/запятые
					let j = i
					while (j < n && /[\s,\r\n]/.test(source[j])) j++

					const hasTypename = source.startsWith('__typename', j)
					if (!hasTypename) out += ' __typename'
					continue
				}
			} else if (state === 'comment') {
				out += c
				i++
				if (c === '\n') state = 'default'
				continue
			} else if (state === 'string') {
				out += c
				i++
				if (c === quoteChar && source[i - 2] !== '\\') state = 'default'
				continue
			} else if (state === 'blockString') {
				out += c
				i++
				if (source.slice(i - blockQuoteLen, i) === quoteChar!.repeat(blockQuoteLen)) state = 'default'
				continue
			}

			/* default path */
			out += c
			i++
		}
		return out
	}

	export function $trip2g_graphql_raw_request(query: string, variables?: any) {
		const res = $.$mol_fetch.json('/graphql', {
			method: 'POST',
			credentials: 'include',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ query: inject_typenames(query), variables }),
		}) as { data?: any; errors?: any[] }

		if (res.errors) {
			throw new $.$trip2g_graphql_error('GraphQL Error', res.errors)
		}

		const isMutation = !!query.match(/^\s+mutation/)

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
			map<V>(fn: (key: string) => V, empty?: V): V[] {
				if (map.size === 0 && empty) {
					return [empty]
				}

				const out: V[] = []
				for (const key of map.keys()) {
					out.push(fn(key))
				}
				return out
			},
		}
	}
}
