namespace $.$$ {
	export function $trip2g_moment_toserver(val: $mol_time_moment | null): string | null {
		if (val === null) {
			return null;
		}

		return val.toString('YYYY-MM-DDThh:mm:ss.sssZ');
	}
}