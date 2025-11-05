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

	SendTelegramPublishPost(ctx context.Context, params model.SendTelegramPublishPostParams) error
	UpdateTelegramPublishPost(ctx context.Context, notePathID int64) error

	ConvertNoteViewToTelegramPost(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error)
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
	at, atOk := note.ExtractTelegramPublishAt(p.timeLocation)
	// telegram_publish_tags: string[]
	tags, tagsOk := note.ExtractTelegramPublishTags()

	if !atOk && !tagsOk {
		return nil
	}

	if atOk != tagsOk {
		const msg = "incomplete telegram publish metadata, both telegram_publish_at and telegram_publish_tags must be present"
		note.AddWarning(model.NoteWarningWarning, msg)
		return nil
	}

	// Validate telegram post content
	source := model.TelegramPostSource{
		NoteView: note,
		ChatID:   0, // ChatID=0 skips DB query for sent messages
		Instant:  false,
	}
	post, err := p.env.ConvertNoteViewToTelegramPost(p.ctx, source)

	for _, tag := range tags {
		upsertErr := p.tagIDs.upsert(p.ctx, p.env, tag)
		if upsertErr != nil {
			return fmt.Errorf("failed to upsert telegram publish tag %q: %w", tag, upsertErr)
		}
	}

	// Count errors: post.Warnings + Critical-level warnings from noteView
	errorCount := int64(0)

	// Count conversion errors
	if err != nil {
		errorCount++
	}

	// Count post warnings (content validation errors)
	if post != nil {
		errorCount += int64(len(post.Warnings))
	}

	// Count Critical-level warnings from noteView
	for _, warning := range note.Warnings {
		if warning.Level == model.NoteWarningCritical {
			errorCount++
		}
	}

	noteParams := db.UpsertTelegramPublishNoteParams{
		NotePathID: note.PathID,
		PublishAt:  at.UTC(),
		ErrorCount: errorCount,
	}

	err = p.env.UpsertTelegramPublishNote(p.ctx, noteParams)
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

	// send a  preview immediately
	params := model.SendTelegramPublishPostParams{
		NotePathID:        note.PathID,
		Instant:           true,
		UpdateLinkedPosts: false,
	}
	err = p.env.SendTelegramPublishPost(p.ctx, params)
	if err != nil {
		return fmt.Errorf("failed to SendTelegramPublishPost: %w", err)
	}

	// update existing post if it was already published
	err = p.env.UpdateTelegramPublishPost(p.ctx, note.PathID)
	if err != nil {
		return fmt.Errorf("failed to UpdateTelegramPublishPost: %w", err)
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
