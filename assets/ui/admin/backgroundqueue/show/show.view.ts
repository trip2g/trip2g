namespace $.$$ {
	const data_request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminBackgroundQueue($id: String!) {
			admin {
				backgroundQueue(id: $id) {
					id
					pendingCount
					retryCount
					stopped
					jobs @exportType(name: "Job", single: true) {
						id
						name
						params
						retryCount
					}
				}
			}
		}
	`)

	export class $trip2g_admin_backgroundqueue_show extends $.$trip2g_admin_backgroundqueue_show {
		@$mol_mem
		data() {
			const res = data_request( { id: this.queue_id() } )
			const data = res.admin.backgroundQueue
			if( !data ) throw new Error( 'Queue not found' )
			return data
		}

		override queue_name(): string {
			return this.data().id
		}

		override queue_name_value(): string {
			return this.queue_id()
		}

		override status_text(): string {
			return this.data().stopped ? 'Stopped' : 'Running'
		}

		override pending_count_text(): string {
			return this.data().pendingCount.toString()
		}

		override retry_count_text(): string {
			return this.data().retryCount.toString()
		}

		override tools() {
			const items: $mol_view[] = [
				this.ClearButton(),
			]

			if( this.data().stopped ) {
				items.push( this.StartButton() )
			} else {
				items.push( this.StopButton() )
			}

			return items
		}

		override job_rows() {
			return this.data().jobs.map( job => this.Job( job.id ) )
		}

		override job_data(id: any) {
			return this.data().jobs.find( job => job.id === id )!
		}
	}

	const truncate = ( str: string, length: number )=> {
		return str.length > length ? str.slice( 0, length ) + '...' : str
	}

	export class $trip2g_admin_backgroundqueue_show_job extends $.$trip2g_admin_backgroundqueue_show_job {
		override data() {
			return super.data()!
		}

		override job_id(): string {
			return truncate(this.data().id, 10)
		}

		override job_name(): string {
			return this.data().name
		}

		override job_params(): string {
			// return JSON.stringify(JSON.parse(this.data().params), null, 2)
			return truncate(this.data().params, 30)
		}
	}
}
