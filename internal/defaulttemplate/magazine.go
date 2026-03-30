package defaulttemplate

import (
	"trip2g/internal/templateviews"

	"github.com/bmatcuk/doublestar/v4"
)

// MagazineItemSize represents the visual size tier of a magazine card.
type MagazineItemSize int

const (
	MagazineItemFeatured MagazineItemSize = iota // index 0
	MagazineItemSmall                            // index 1-4
	MagazineItemList                             // index 5+
)

// MagazineItem represents a single item in the magazine layout.
type MagazineItem struct {
	Note *templateviews.Note
	Size MagazineItemSize
}

// MagazineItems returns magazine items for the current page.
// Sorting: if magazine_sort_property is set, notes with that property come first
// (sorted by its value desc); remaining notes follow sorted by created_at desc.
// Filtering: if magazine_include_property is set, only notes with that property are included.
// Excludes the current note.
func (ctx *Ctx) MagazineItems() []MagazineItem {
	if ctx.Notes == nil {
		return nil
	}

	glob := ctx.MagazineIncludeFiles()
	excludeGlob := ctx.MagazineExcludeFiles()
	sortProp := ctx.MagazineSortProperty()
	includeProp := ctx.MagazineIncludeProperty()

	q := ctx.Notes.ByGlob(glob)
	if sortProp != "" {
		// Notes with the property sort first (desc = higher number first), then the rest by date desc.
		// compareValues returns -1 for nil, so nil sorts last in desc order,
		// and the secondary created_at sort handles their relative order.
		q = q.SortByMeta(sortProp).Desc().SortBy("created_at").Desc().SortBy("path_id").Desc()
	} else {
		q = q.SortBy("created_at").Desc().SortBy("path_id").Desc()
	}

	all := q.All()

	var items []MagazineItem
	for _, note := range all {
		if ctx.Note != nil && note.Path() == ctx.Note.Path() {
			continue
		}
		// Skip system notes (any path component starting with _).
		if note.IsSystem() {
			continue
		}
		if excludeGlob != "" {
			if matched, _ := doublestar.Match(excludeGlob, note.Path()); matched {
				continue
			}
		}
		if includeProp != "" && note.M().Get(includeProp) == nil {
			continue
		}
		size := MagazineItemList
		if len(items) == 0 {
			size = MagazineItemFeatured
		} else if len(items) < 5 {
			size = MagazineItemSmall
		}
		items = append(items, MagazineItem{
			Note: note,
			Size: size,
		})
	}
	return items
}
