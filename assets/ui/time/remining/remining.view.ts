namespace $.$$ {
	const MINUTE = 60
	const HOUR = 60 * MINUTE
	const DAY = 24 * HOUR

	function pluralizeRussian( num: number, form1: string, form2: string, form5: string ) {
		const n = Math.abs(num) % 100
		const n1 = n % 10
		
		if (n > 10 && n < 20) return form5
		if (n1 > 1 && n1 < 5) return form2
		if (n1 === 1) return form1
		return form5
	}

	export class $trip2g_time_remining extends $.$trip2g_time_remining {
		override text(): string {
			const s = this.seconds()
			const absS = Math.abs(s)
			const isNegative = s < 0

			if( absS < MINUTE ) {
				const timeText = `${ absS } ${ pluralizeRussian( absS, 'секунда', 'секунды', 'секунд' ) }`
				return isNegative ? `${ timeText } назад` : `осталось ${ timeText }`
			}

			if( absS < HOUR ) {
				const minutes = Math.floor( absS / MINUTE )
				const timeText = `${ minutes } ${ pluralizeRussian( minutes, 'минута', 'минуты', 'минут' ) }`
				return isNegative ? `${ timeText } назад` : `осталось ${ timeText }`
			}

			if( absS < DAY ) {
				const hours = Math.floor( absS / HOUR )
				const timeText = `${ hours } ${ pluralizeRussian( hours, 'час', 'часа', 'часов' ) }`
				return isNegative ? `${ timeText } назад` : `осталось ${ timeText }`
			}

			const days = Math.floor( absS / DAY )
			const timeText = `${ days } ${ pluralizeRussian( days, 'день', 'дня', 'дней' ) }`
			return isNegative ? `${ timeText } назад` : `осталось ${ timeText }`
		}
	}
}
