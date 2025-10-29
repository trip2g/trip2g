package handletgpublishviews

import (
	"context"
	"fmt"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

type Env interface {
	Logger() logger.Logger
	InsertTelegramPublishTags(ctx context.Context, label string) error
	TelegramPublishTagByLabel(ctx context.Context, label string) (db.TelegramPublishTag, error)

	UpsertTelegramPublishNote(ctx context.Context, params db.UpsertTelegramPublishNoteParams) error
	DeleteTelegramPublishNoteTagsByPathID(ctx context.Context, pathID int64) error
	UpsertTelegramPublishNoteTag(ctx context.Context, params db.UpsertTelegramPublishNoteTagParams) error

	TimeLocation() *time.Location
	LatestNoteViews() *model.NoteViews

	SendTelegramPublishPost(ctx context.Context, notePathID int64, instant bool) error
}

type tagIDCache map[string]int64

func Resolve(ctx context.Context, env Env, changedPathIDs []int64) error {
	timeLocation := env.TimeLocation()
	nvs := env.LatestNoteViews()

	p := &processor{
		timeLocation: timeLocation,
		tagIDs:       tagIDCache{},

		nvs: nvs,
		ctx: ctx,
		env: env,
	}

	changedNoteIDs := make(map[int64]struct{})

	for _, id := range changedPathIDs {
		changedNoteIDs[id] = struct{}{}
	}

	for _, note := range nvs.List {
		_, changed := changedNoteIDs[note.PathID]
		if !changed {
			continue
		}

		err := p.process(note)
		if err != nil {
			return fmt.Errorf("failed to process note %q: %w", note.Path, err)
		}
	}

	return nil
}

type processor struct {
	tagIDs       tagIDCache
	timeLocation *time.Location
	nvs          *model.NoteViews
	ctx          context.Context
	env          Env
}

func (p *processor) process(note *model.NoteView) error {
	// telegram_publish_at: 2024-07-02T23:02:00
	at, atOk := extractTime(note, p.timeLocation)
	// telegram_publish_tags: string[]
	tags, tagsOk := extractTags(note)

	if !atOk && !tagsOk {
		return nil
	}

	if atOk != tagsOk {
		const msg = "incomplete telegram publish metadata, both telegram_publish_at and telegram_publish_tags must be present"
		note.AddWarning(model.NoteWarningWarning, msg)
		return nil
	}

	for _, tag := range tags {
		upsertErr := p.tagIDs.upsert(p.ctx, p.env, tag)
		if upsertErr != nil {
			return fmt.Errorf("failed to upsert telegram publish tag %q: %w", tag, upsertErr)
		}
	}

	noteParams := db.UpsertTelegramPublishNoteParams{
		NotePathID: note.PathID,
		PublishAt:  at.UTC(),
	}

	err := p.env.UpsertTelegramPublishNote(p.ctx, noteParams)
	if err != nil {
		return fmt.Errorf("failed to UpsertTelegramPublishNote: %w", err)
	}

	err = p.env.DeleteTelegramPublishNoteTagsByPathID(p.ctx, note.PathID)
	if err != nil {
		return fmt.Errorf("failed to DeleteTelegramPublishNoteTagsByPathID: %w", err)
	}

	for _, tag := range tags {
		tagParams := db.UpsertTelegramPublishNoteTagParams{
			NotePathID: note.PathID,
			TagID:      p.tagIDs[tag],
		}

		err = p.env.UpsertTelegramPublishNoteTag(p.ctx, tagParams)
		if err != nil {
			return fmt.Errorf("failed to UpsertTelegramPublishNoteTag for tag %q: %w", tag, err)
		}
	}

	err = p.env.SendTelegramPublishPost(p.ctx, note.PathID, true)
	if err != nil {
		return fmt.Errorf("failed to SendTelegramPublishPost for note %q: %w", note.Path, err)
	}

	return nil
}

func (c tagIDCache) upsert(ctx context.Context, env Env, label string) error {
	_, exists := c[label]
	if exists {
		return nil
	}

	// on conflict(label) do nothing
	insertErr := env.InsertTelegramPublishTags(ctx, label)
	if insertErr != nil {
		return fmt.Errorf("failed to insert telegram publish tag %q: %w", label, insertErr)
	}

	tag, err := env.TelegramPublishTagByLabel(ctx, label)
	if err != nil {
		return fmt.Errorf("failed to fetch telegram publish tag %q: %w", label, err)
	}

	c[label] = tag.ID

	return nil
}

func extractTime(note *model.NoteView, loc *time.Location) (time.Time, bool) {
	rawAt, ok := note.RawMeta["telegram_publish_at"]
	if !ok {
		return time.Time{}, false
	}

	atStr, ok := rawAt.(string)
	if !ok {
		note.AddWarning(model.NoteWarningWarning, "invalid telegram_publish_at format, expected string")
		return time.Time{}, false
	}

	// parse time with timezone
	at, err := time.Parse(time.RFC3339, atStr)
	if err == nil {
		return at, true
	}

	// parse time without timezone
	at, err = time.ParseInLocation("2006-01-02T15:04:05", atStr, loc)
	if err != nil {
		msg := "failed to parse telegram_publish_at, expected format YYYY-MM-DDTHH:MM:SS (%s)"
		note.AddWarning(model.NoteWarningWarning, msg, err.Error())
		return time.Time{}, false
	}

	return at, true
}

func extractTags(note *model.NoteView) ([]string, bool) {
	rawTags, ok := note.RawMeta["telegram_publish_tags"]
	if !ok {
		return nil, false
	}

	tagsI, ok := rawTags.([]interface{})
	if !ok {
		note.AddWarning(model.NoteWarningWarning, "invalid telegram_publish_tags format, expected []string")
		return nil, false
	}

	var tags []string

	for _, t := range tagsI {
		tagStr, tagOk := t.(string)
		if !tagOk {
			note.AddWarning(model.NoteWarningWarning, "invalid tag in telegram_publish_tags, expected string")
			continue
		}

		tags = append(tags, tagStr)
	}

	return tags, true
}
