package defaulttemplate

import "trip2g/internal/templateviews"

// MagazineItemSize represents the visual size tier of a magazine card.
type MagazineItemSize int

const (
	MagazineItemFeatured MagazineItemSize = iota // index 0
	MagazineItemSmall                            // index 1-4
	MagazineItemList                             // index 5+
)

// MagazineItem represents a single item in the magazine layout.
type MagazineItem struct {
	Note     *templateviews.Note
	Size     MagazineItemSize
	ImageURL string
}

// MagazineItems returns magazine items sorted by magazine_property meta field, descending.
// Excludes the current note. Only includes notes with the magazine_property set.
func (ctx *Ctx) MagazineItems() []MagazineItem {
	if ctx.Notes == nil {
		return nil
	}

	prop := ctx.MagazineProperty()
	glob := ctx.MagazineIncludeFiles()

	all := ctx.Notes.ByGlob(glob).SortByMeta(prop).Desc().All()

	var items []MagazineItem
	for _, note := range all {
		if ctx.Note != nil && note.Path() == ctx.Note.Path() {
			continue // exclude current note
		}
		if note.M().Get(prop) == nil {
			continue // only include notes with the property set
		}
		size := MagazineItemList
		if len(items) == 0 {
			size = MagazineItemFeatured
		} else if len(items) < 5 {
			size = MagazineItemSmall
		}
		items = append(items, MagazineItem{
			Note:     note,
			Size:     size,
			ImageURL: note.FirstImageURL(),
		})
	}
	return items
}
