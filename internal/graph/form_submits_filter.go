package graph

import (
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

const (
	defaultFormSubmitsLimit = 50
	maxFormSubmitsLimit     = 200
)

func buildCountFormSubmitsParams(f *model.AdminFormSubmitsFilterInput) db.CountFormSubmitsParams {
	p := db.CountFormSubmitsParams{}
	if f == nil {
		return p
	}
	if f.NotePathID != nil {
		p.NotePathID = *f.NotePathID
	}
	if f.FormID != nil {
		p.FormID = *f.FormID
	}
	if f.Status != nil {
		p.Status = string(*f.Status)
	}
	if f.Processed != nil {
		if *f.Processed {
			p.ProcessedFilter = int64(1)
		} else {
			p.ProcessedFilter = int64(0)
		}
	}
	if f.CreatedAt != nil {
		if f.CreatedAt.Gte != nil {
			p.CreatedAtGte = *f.CreatedAt.Gte
		}
		if f.CreatedAt.Lte != nil {
			p.CreatedAtLte = *f.CreatedAt.Lte
		}
	}
	return p
}

func buildListFormSubmitsParams(f *model.AdminFormSubmitsFilterInput) db.ListFormSubmitsParams {
	c := buildCountFormSubmitsParams(f)
	p := db.ListFormSubmitsParams{
		NotePathID:      c.NotePathID,
		FormID:          c.FormID,
		Status:          c.Status,
		ProcessedFilter: c.ProcessedFilter,
		CreatedAtGte:    c.CreatedAtGte,
		CreatedAtLte:    c.CreatedAtLte,
		Lim:             defaultFormSubmitsLimit,
		Off:             0,
	}
	if f == nil {
		return p
	}
	if f.Limit != nil {
		lim := int64(*f.Limit)
		if lim < 0 {
			lim = 0
		}
		if lim > maxFormSubmitsLimit {
			lim = maxFormSubmitsLimit
		}
		p.Lim = lim
	}
	if f.Offset != nil {
		off := int64(*f.Offset)
		if off < 0 {
			off = 0
		}
		p.Off = off
	}
	return p
}
