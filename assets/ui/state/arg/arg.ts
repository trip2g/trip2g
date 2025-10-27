namespace $ {
	const TRUE_MARKET = '1'

	export class $trip2g_state_arg extends $mol_object {
		
		@ $mol_mem_key
		static bool_value( key : string , next? : boolean ): boolean {
			if (next !== undefined) {
				this.$.$mol_state_arg.value( key, next ? TRUE_MARKET : null )
			}

			return this.$.$mol_state_arg.value( key ) === TRUE_MARKET
		}
		
	}
}
