namespace $.$$ {
	const MINUTE = 60
	const HOUR = 60 * MINUTE
	const DAY = 24 * HOUR

	function pluralize( num: number, singular: string, plural: string ) {
		return num === 1 ? singular : plural
	}

	export class $trip2g_time_remining extends $.$trip2g_time_remining {
		override text(): string {
			const s = this.seconds()

			if( s < MINUTE ) {
				return `${ s } ${ pluralize( s, 'second', 's' ) } left`
			}

			if( s < HOUR ) {
				const minutes = Math.floor( s / MINUTE )
				return `${ minutes } ${ pluralize( minutes, 'minute', 'minutes' ) } left`
			}

			if( s < DAY ) {
				const hours = Math.floor( s / HOUR )
				return `${ hours } ${ pluralize( hours, 'hour', 'hours' ) } left`
			}

			const days = Math.floor( s / DAY )
			return `${ days } ${ pluralize( days, 'day', 'days' ) } left`
		}
	}
}
