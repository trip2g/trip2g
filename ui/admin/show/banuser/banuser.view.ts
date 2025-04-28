namespace $.$$ {
	export class $trip2g_admin_show_banuser extends $.$trip2g_admin_show_banuser {
		@$mol_mem
		reason(next?: string): string {
			return next ?? '';
		}

		ban_user() {
			const reason = this.reason();
			// Здесь должна быть логика мутации для бана пользователя
			console.log(`User banned for reason: ${reason}`);
		}
	}
}
