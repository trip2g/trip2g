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
// Sizing: the first magazine_featured items are MagazineItemFeatured, the next
// magazine_grid items are MagazineItemSmall, everything after that is MagazineItemList.
func (ctx *Ctx) MagazineItems() []MagazineItem {
	if ctx.Notes == nil {
		return nil
	}

	glob := ctx.MagazineIncludeFiles()
	excludeGlob := ctx.MagazineExcludeFiles()
	sortProp := ctx.MagazineSortProperty()
	includeProp := ctx.MagazineIncludeProperty()
	excludeProp := ctx.MagazineExcludeProperty()

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

	var notes []*templateviews.Note
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
		if excludeProp != "" && note.M().Get(excludeProp) != nil {
			continue
		}
		notes = append(notes, note)
	}

	featuredCount, gridCount, _ := magazineTierCounts(len(notes), ctx.MagazineFeaturedCount(), ctx.MagazineGridCount())

	items := make([]MagazineItem, len(notes))
	for i, note := range notes {
		size := MagazineItemList
		switch {
		case i < featuredCount:
			size = MagazineItemFeatured
		case i < featuredCount+gridCount:
			size = MagazineItemSmall
		}
		items[i] = MagazineItem{Note: note, Size: size}
	}
	return items
}

// magazineTierCounts clamps the requested featured/grid tier sizes to the available
// item total. Negative inputs are treated as 0. Returns the featured, grid, and list
// tier sizes, which always sum to total.
func magazineTierCounts(total, featured, grid int) (int, int, int) {
	if featured < 0 {
		featured = 0
	}
	if grid < 0 {
		grid = 0
	}
	if featured > total {
		featured = total
	}
	remaining := total - featured
	if grid > remaining {
		grid = remaining
	}
	return featured, grid, remaining - grid
}
