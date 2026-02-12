namespace $ {

	export class $trip2g_sse_host extends $mol_object {

		restart_delay() { return 3000 }
		url() { return '/graphql' }
		query() { return '' }
		variables() { return {} as Record<string, unknown> }

		@ $mol_mem
		opened(next?: boolean) { return next ?? false }

		@ $mol_mem
		error_packed(error?: null | [Error]) { return error ?? null }

		error(error?: null | Error) {
			return this.error_packed(error ? [error] : error)?.[0] ?? null
		}

		@ $mol_mem
		ready() {
			this.source()
			return this.opened()
		}

		@ $mol_mem
		error_message() {
			try {
				this.ready()
				return this.error()?.message ?? ''
			} catch (e) {
				if (!$mol_promise_like(e)) return (e as Error).message ?? ''
			}

			return ''
		}

		@ $mol_mem
		data(next?: any) {
			this.source()
			return next ?? null
		}

		@ $mol_mem
		source(reset?: null) {
			const query = this.query().replace(/@exportType\s*(\([^)]*\))?\s*/g, '')
			const abort = new AbortController()

			this.stream(query, abort.signal)

			return {
				destructor: () => abort.abort(),
			}
		}

		protected async stream(query: string, signal: AbortSignal) {
			while (!signal.aborted) {
				try {
					await this.connect(query, signal)
				} catch (err) {
					if (signal.aborted) return
					this.opened(false)
					this.error(err instanceof Error ? err : new Error(String(err)))
				}

				if (signal.aborted) return
				await new Promise(resolve => setTimeout(resolve, this.restart_delay()))
			}
		}

		protected async connect(query: string, signal: AbortSignal) {
			const response = await fetch(this.url(), {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'Accept': 'text/event-stream',
				},
				credentials: 'include',
				body: JSON.stringify({ query, variables: this.variables() }),
				signal,
			})

			if (!response.ok) {
				throw new Error(`SSE: ${response.status} ${response.statusText}`)
			}

			const reader = response.body!.getReader()
			const decoder = new TextDecoder()
			let buffer = ''
			let eventType = ''

			this.opened(true)
			this.error(null)

			try {
				while (true) {
					const { done, value } = await reader.read()
					if (done) break

					buffer += decoder.decode(value, { stream: true })

					const lines = buffer.split('\n')
					buffer = lines.pop()!

					for (const line of lines) {
						if (line.startsWith('event:')) {
							eventType = line.slice(6).trim()
						} else if (line.startsWith('data:')) {
							const payload = line.slice(5).trim()

							if (eventType === 'next') {
								const parsed = JSON.parse(payload)
								if (parsed.errors) {
									this.error(new $trip2g_graphql_error('Subscription error', parsed.errors))
								}
								if (parsed.data) {
									this.data(parsed.data)
								}
							} else if (eventType === 'complete') {
								return
							}

							eventType = ''
						}
					}
				}
			} finally {
				reader.releaseLock()
				this.opened(false)
			}
		}
	}
}
