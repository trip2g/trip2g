package templateviews_test

import (
	"trip2g/internal/model"
	"trip2g/internal/templateviews"
)

func openNVS(nvs *model.NoteViews) *templateviews.NVS {
	return templateviews.NVSWithAccess(templateviews.NewNVS(nvs, "live"), func(*model.NoteView) bool {
		return true
	})
}
