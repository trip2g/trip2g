namespace $.$$ {

	const sub = $trip2g_graphql_subscription(/* GraphQL */ `
		subscription CurrentTime($format: String) {
			currentTime(format: $format)
		}
	`, { format: '15:04' })

	export class $trip2g_time_current extends $.$trip2g_time_current {
		override sse() {
			return sub
		}

		override current_time(): string {
			return this.sse().data()?.currentTime ?? ''
		}
	}

}
