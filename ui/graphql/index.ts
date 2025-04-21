namespace $ {
	export const $trip2g_graphql_request = function<T>(s: string, variables: any) {
		const res = $.$mol_fetch.json('/graphql', {
			method: 'POST',
			credentials: 'include',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify({
				query: s,
				variables,
			}),
		}) as {
			data: T;
		}

		return res.data;
	}
}
