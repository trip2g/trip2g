namespace $ {

	/**
	 * Request seam for *.codellm.graphql wrappers: same convention as
	 * $trip2g_gql_request (shared invalidation marker), different endpoint.
	 * The codellm subgraph schema lives at
	 * cmd/codellm/internal/codellm/codellmgql/schema.graphqls; its generated
	 * schema types (schema.graphql.ts in this module) are separate from the
	 * main-schema types (assets/ui/gql/schema.graphql.ts).
	 */
	export function $trip2g_codellm_gql_request(query: string, variables?: object, opts?: $trip2g_gql_opts): unknown {
		return $trip2g_gql_post('/_system/codellm/graphql', query, variables, opts)
	}

	/** Opaque fragment ref for codellm-subgraph fragments (see $trip2g_gql_ref). */
	export type $trip2g_codellm_gql_ref<Frag> = $trip2g_gql_ref<Frag>

}
