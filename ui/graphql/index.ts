namespace $ {
	export const $trip2g_graphql = (s: string) => s;

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
}
