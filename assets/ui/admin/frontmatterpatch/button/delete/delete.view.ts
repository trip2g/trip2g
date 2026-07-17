namespace $.$$ {
	export class $trip2g_admin_frontmatterpatch_button_delete extends $.$trip2g_admin_frontmatterpatch_button_delete {
		click(e: PointerEvent) {
			e.stopPropagation()
			e.preventDefault()

			const res = $trip2g_admin_frontmatterpatch_button_delete_save({
				input: {
					id: this.frontmatterpatch_id(),
				},
			});

			if (res.admin.payload.__typename === 'ErrorPayload') {
				throw new Error(res.admin.payload.message);
			}

			if (res.admin.payload.__typename === 'DeleteFrontmatterPatchPayload') {
				this.after_success()
			}
		}
	}
}
