package defaulttemplate

import (
	"trip2g/internal/model"
	"trip2g/internal/templateviews"
)

func openNVS(nvs *model.NoteViews, defaultVersion string) *templateviews.NVS {
	return templateviews.NVSWithAccess(templateviews.NewNVS(nvs, defaultVersion), func(*model.NoteView) bool {
		return true
	})
}
