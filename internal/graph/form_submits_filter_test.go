package graph

import (
	"testing"
	"time"

	"trip2g/internal/graph/model"

	"github.com/stretchr/testify/require"
)

func ptrInt64(v int64) *int64                                    { return &v }
func ptrInt32(v int32) *int32                                    { return &v }
func ptrStr(v string) *string                                    { return &v }
func ptrBool(v bool) *bool                                       { return &v }
func ptrStatus(v model.FormSubmitStatus) *model.FormSubmitStatus { return &v }
func ptrTime(t time.Time) *time.Time                             { return &t }

func TestBuildListFormSubmitsParams_NilFilter(t *testing.T) {
	p := buildListFormSubmitsParams(nil)
	require.Equal(t, int64(defaultFormSubmitsLimit), p.Lim)
	require.Equal(t, int64(0), p.Off)
	require.Nil(t, p.NotePathID)
	require.Nil(t, p.FormID)
	require.Nil(t, p.Status)
	require.Nil(t, p.ProcessedFilter)
	require.Nil(t, p.CreatedAtGte)
	require.Nil(t, p.CreatedAtLte)
}

func TestBuildListFormSubmitsParams_ProcessedFalse(t *testing.T) {
	p := buildListFormSubmitsParams(&model.AdminFormSubmitsFilterInput{Processed: ptrBool(false)})
	require.Equal(t, int64(0), p.ProcessedFilter)
}

func TestBuildListFormSubmitsParams_ProcessedTrue(t *testing.T) {
	p := buildListFormSubmitsParams(&model.AdminFormSubmitsFilterInput{Processed: ptrBool(true)})
	require.Equal(t, int64(1), p.ProcessedFilter)
}

func TestBuildListFormSubmitsParams_NotePathID(t *testing.T) {
	p := buildListFormSubmitsParams(&model.AdminFormSubmitsFilterInput{NotePathID: ptrInt64(42)})
	require.Equal(t, int64(42), p.NotePathID)
}

func TestBuildListFormSubmitsParams_LimitClamp(t *testing.T) {
	cases := []struct {
		in  int32
		out int64
	}{
		{in: -5, out: 0},
		{in: 0, out: 0},
		{in: 1, out: 1},
		{in: 200, out: 200},
		{in: 500, out: maxFormSubmitsLimit},
	}
	for _, c := range cases {
		p := buildListFormSubmitsParams(&model.AdminFormSubmitsFilterInput{Limit: ptrInt32(c.in)})
		require.Equal(t, c.out, p.Lim, "limit %d", c.in)
	}
}

func TestBuildListFormSubmitsParams_OffsetClamp(t *testing.T) {
	p := buildListFormSubmitsParams(&model.AdminFormSubmitsFilterInput{Offset: ptrInt32(-10)})
	require.Equal(t, int64(0), p.Off)

	p = buildListFormSubmitsParams(&model.AdminFormSubmitsFilterInput{Offset: ptrInt32(99)})
	require.Equal(t, int64(99), p.Off)
}

func TestBuildListFormSubmitsParams_CreatedAtWindow(t *testing.T) {
	gte := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	lte := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	p := buildListFormSubmitsParams(&model.AdminFormSubmitsFilterInput{
		CreatedAt: &model.AdminFormSubmitsDateFilter{Gte: ptrTime(gte), Lte: ptrTime(lte)},
	})
	require.Equal(t, gte, p.CreatedAtGte)
	require.Equal(t, lte, p.CreatedAtLte)
}

func TestBuildListFormSubmitsParams_Status(t *testing.T) {
	p := buildListFormSubmitsParams(&model.AdminFormSubmitsFilterInput{Status: ptrStatus(model.FormSubmitStatusPending)})
	require.Equal(t, "pending", p.Status)
}

func TestBuildListFormSubmitsParams_FormIDPassthrough(t *testing.T) {
	// Adversarial input must pass through as-is (parameterised by sqlc) — no error, no panic.
	id := "'; DROP TABLE form_submits;--"
	p := buildListFormSubmitsParams(&model.AdminFormSubmitsFilterInput{FormID: ptrStr(id)})
	require.Equal(t, id, p.FormID)
}

func TestBuildCountFormSubmitsParams_MirrorsList(t *testing.T) {
	f := &model.AdminFormSubmitsFilterInput{
		NotePathID: ptrInt64(7),
		FormID:     ptrStr("contact"),
		Status:     ptrStatus(model.FormSubmitStatusVisible),
		Processed:  ptrBool(false),
		Limit:      ptrInt32(10),
		Offset:     ptrInt32(20),
	}
	l := buildListFormSubmitsParams(f)
	c := buildCountFormSubmitsParams(f)
	require.Equal(t, l.NotePathID, c.NotePathID)
	require.Equal(t, l.FormID, c.FormID)
	require.Equal(t, l.Status, c.Status)
	require.Equal(t, l.ProcessedFilter, c.ProcessedFilter)
}
