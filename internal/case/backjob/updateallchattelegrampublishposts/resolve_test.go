package updateallchattelegrampublishposts_test

import (
	"context"
	"errors"
	"testing"
	"trip2g/internal/case/backjob/updateallchattelegrampublishposts"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

func TestResolve(t *testing.T) { //nolint:gocognit // test complexity is acceptable
	tests := []struct {
		name             string
		params           updateallchattelegrampublishposts.Params
		sentMessages     []db.ListTelegramPublishSentMessagesByChatIDRow
		listErr          error
		noteViews        map[int64]*model.NoteView
		updateErrs       map[int64]error
		wantErr          bool
		wantErrContains  string
		updateCallsCount int
	}{
		{
			name: "success with multiple notes",
			params: updateallchattelegrampublishposts.Params{
				ChatID: 1,
			},
			sentMessages: []db.ListTelegramPublishSentMessagesByChatIDRow{
				{ChatID: 1, MessageID: 100, NotePathID: 10, NotePath: "/note1", TelegramChatID: -1001},
				{ChatID: 1, MessageID: 101, NotePathID: 10, NotePath: "/note1", TelegramChatID: -1001},
				{ChatID: 1, MessageID: 102, NotePathID: 20, NotePath: "/note2", TelegramChatID: -1001},
			},
			noteViews: map[int64]*model.NoteView{
				10: {PathID: 10, Path: "/note1"},
				20: {PathID: 20, Path: "/note2"},
			},
			updateErrs:       map[int64]error{},
			wantErr:          false,
			updateCallsCount: 3, // 2 messages for note 10 + 1 message for note 20
		},
		{
			name: "no sent messages",
			params: updateallchattelegrampublishposts.Params{
				ChatID: 1,
			},
			sentMessages:     []db.ListTelegramPublishSentMessagesByChatIDRow{},
			wantErr:          false,
			updateCallsCount: 0,
		},
		{
			name: "error listing sent messages",
			params: updateallchattelegrampublishposts.Params{
				ChatID: 1,
			},
			listErr:          errors.New("database error"),
			wantErr:          true,
			wantErrContains:  "failed to list sent messages",
			updateCallsCount: 0,
		},
		{
			name: "error updating note",
			params: updateallchattelegrampublishposts.Params{
				ChatID: 1,
			},
			sentMessages: []db.ListTelegramPublishSentMessagesByChatIDRow{
				{ChatID: 1, MessageID: 100, NotePathID: 10, NotePath: "/note1", TelegramChatID: -1001},
			},
			noteViews: map[int64]*model.NoteView{
				10: {PathID: 10, Path: "/note1"},
			},
			updateErrs: map[int64]error{
				10: errors.New("update error"),
			},
			wantErr:          true,
			wantErrContains:  "failed to update telegram publish post",
			updateCallsCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalls := 0

			env := &EnvMock{
				LoggerFunc: func() logger.Logger {
					return &logger.TestLogger{}
				},
				ListTelegramPublishSentMessagesByChatIDFunc: func(ctx context.Context, chatID int64) ([]db.ListTelegramPublishSentMessagesByChatIDRow, error) {
					if tt.listErr != nil {
						return nil, tt.listErr
					}
					return tt.sentMessages, nil
				},
				LatestNoteViewsFunc: func() *model.NoteViews {
					nvList := make([]*model.NoteView, 0, len(tt.noteViews))
					nvMap := make(map[string]*model.NoteView)
					for id, nv := range tt.noteViews {
						nvList = append(nvList, nv)
						nvMap[nv.Path] = nv
						_ = id
					}
					return &model.NoteViews{
						Map:       nvMap,
						List:      nvList,
						Subgraphs: map[string]*model.NoteSubgraph{},
						Version:   "test",
					}
				},
				ListTelegramPublishSentMessagesByNotePathIDFunc: func(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error) {
					// Return messages matching this notePathID
					var result []db.ListTelegramPublishSentMessagesByNotePathIDRow
					for _, msg := range tt.sentMessages {
						if msg.NotePathID == notePathID {
							result = append(result, db.ListTelegramPublishSentMessagesByNotePathIDRow{
								ChatID:      msg.ChatID,
								MessageID:   msg.MessageID,
								TelegramID:  msg.TelegramChatID,
								ContentHash: "oldhash",
							})
						}
					}
					return result, nil
				},
				ConvertNoteViewToTelegramPostFunc: func(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error) {
					return &model.TelegramPost{Content: "test"}, nil
				},
				QueueUpdateTelegramPostFunc: func(ctx context.Context, params model.TelegramUpdatePostParams) error {
					updateCalls++
					if err, ok := tt.updateErrs[params.NotePathID]; ok {
						return err
					}
					return nil
				},
			}

			err := updateallchattelegrampublishposts.Resolve(context.Background(), env, tt.params)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Resolve() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.wantErrContains != "" && !contains(err.Error(), tt.wantErrContains) {
					t.Errorf("Resolve() error = %v, want error containing %v", err, tt.wantErrContains)
				}
			} else if err != nil {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
			}

			if updateCalls != tt.updateCallsCount {
				t.Errorf("Resolve() update calls = %v, want %v", updateCalls, tt.updateCallsCount)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
