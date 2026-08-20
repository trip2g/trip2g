namespace $ {
	/**
	 * Renders reported costs as text. The unit lives in the key, and how many
	 * units there are is up to whoever ran the delivery, so this formats whatever
	 * arrived instead of naming any of them.
	 */
	export function $trip2g_admin_costs_text( costs: readonly { id: string, value: number }[] ): string {
		if( !costs?.length ) return '-'
		return costs.map( cost => `${ $trip2g_admin_costs_amount( cost.value ) } ${ cost.id }` ).join( ', ' )
	}

	/** Trims the float tail so 5186 does not read as 5186.000001. */
	export function $trip2g_admin_costs_amount( value: number ): string {
		if( Number.isInteger( value ) ) return value.toString()
		return value.toFixed( 4 ).replace( /0+$/, '' ).replace( /\.$/, '' )
	}
}
