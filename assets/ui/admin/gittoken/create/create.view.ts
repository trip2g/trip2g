namespace $.$$ {
	export class $trip2g_admin_gittoken_create extends $.$trip2g_admin_gittoken_create {
		override body() {
			if (this.git_token() !== '') {
				return [this.GitTokenView()]
			}

			return super.body()
		}

		override submit() {
			const res = $trip2g_admin_gittoken_create_create({
				input: {
					description: this.description(),
					canPull: this.can_pull(),
					canPush: this.can_push(),
				},
			})

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message)
			}

			if (res.admin.data.__typename === 'CreateGitTokenPayload') {
				this.git_token(res.admin.data.value)
				return
			}

			throw new Error('Unexpected response type')
		}
	}
}