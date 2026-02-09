namespace $.$$ {

	// const current_time_host = $trip2g_graphql_subscription(/* GraphQL */ `
	// 	subscription CurrentTime($format: String) {
	// 		currentTime(format: $format)
	// 	}
	// `, { format: '15:04:05' })

	export class $trip2g_time_current extends $.$trip2g_time_current {

		// override sse() {
		// 	return current_time_host
		// }

		override current_time(): string {
			return 'hello'
			// return this.sse().data()?.currentTime ?? ''
		}
	}
}
