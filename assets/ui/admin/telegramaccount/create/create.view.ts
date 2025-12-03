namespace $.$$ {
	export class $trip2g_admin_telegramaccount_create extends $.$trip2g_admin_telegramaccount_create {
		override body() {
			switch (this.step()) {
				case 'step0':
					return [this.Step0()]
				case 'step1':
					return [this.Step1()]
			}

			return super.body()
		}

		override to_step_1() {
			this.step('step1')
		}
	}
}
