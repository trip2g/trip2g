namespace $ {

	$mol_test({

		async 'currentTime subscription delivers data and cleans up'() {

			const host = new $trip2g_sse_host()
			host.query = () => 'subscription { currentTime }'

			// Start connection.
			host.source()

			// Collect events for 3 seconds.
			const events: string[] = []

			await new Promise<void>(resolve => {
				const start = Date.now()
				const check = () => {
					const d = host.data()
					if (d?.currentTime && !events.includes(d.currentTime)) {
						events.push(d.currentTime)
					}
					if (Date.now() - start >= 3000) {
						resolve()
					} else {
						setTimeout(check, 200)
					}
				}
				check()
			})

			// Should have received 2+ distinct time values in 3 seconds.
			$mol_assert_ok(events.length >= 2)
			$mol_assert_ok(host.opened())

			// Cleanup — destructor aborts the stream.
			host.source(null)

			// Give abort a tick to settle.
			await new Promise(resolve => setTimeout(resolve, 100))
			$mol_assert_not(host.opened())
		},

	})
}
