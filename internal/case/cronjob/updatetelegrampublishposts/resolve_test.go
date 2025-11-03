package updatetelegrampublishposts_test

import (
	"context"
	"errors"
	"testing"
	"trip2g/internal/case/cronjob/updatetelegrampublishposts"
	"trip2g/internal/logger"
)

func TestResolve(t *testing.T) { //nolint:gocognit // test complexity is acceptable
	tests := []struct {
		name             string
		chatIDs          []int64
		listErr          error
		enqueueErrs      map[int64]error
		wantErr          bool
		wantErrContains  string
		wantEnqueueCalls int
		wantSuccessChats int
		wantFailedChats  int
	}{
		{
			name:             "success with multiple chats",
			chatIDs:          []int64{1, 2, 3},
			enqueueErrs:      map[int64]error{},
			wantErr:          false,
			wantEnqueueCalls: 3,
			wantSuccessChats: 3,
			wantFailedChats:  0,
		},
		{
			name:             "no chats",
			chatIDs:          []int64{},
			wantErr:          false,
			wantEnqueueCalls: 0,
			wantSuccessChats: 0,
			wantFailedChats:  0,
		},
		{
			name:            "error listing chats",
			listErr:         errors.New("database error"),
			wantErr:         true,
			wantErrContains: "failed to ListDistinctChatIDsFromSentMessages",
		},
		{
			name:    "partial failure enqueueing",
			chatIDs: []int64{1, 2, 3},
			enqueueErrs: map[int64]error{
				2: errors.New("enqueue error"),
			},
			wantErr:          false, // cronjob continues on individual failures
			wantEnqueueCalls: 3,
			wantSuccessChats: 2,
			wantFailedChats:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enqueueCalls := 0

			env := &EnvMock{
				LoggerFunc: func() logger.Logger {
					return &logger.TestLogger{}
				},
				ListDistinctChatIDsFromSentMessagesFunc: func(ctx context.Context) ([]int64, error) {
					if tt.listErr != nil {
						return nil, tt.listErr
					}
					return tt.chatIDs, nil
				},
				QueueUpdateAllChatTelegramPublishPostsFunc: func(ctx context.Context, chatID int64) error {
					enqueueCalls++
					if err, ok := tt.enqueueErrs[chatID]; ok {
						return err
					}
					return nil
				},
			}

			result, err := updatetelegrampublishposts.Resolve(context.Background(), env)

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

			if enqueueCalls != tt.wantEnqueueCalls {
				t.Errorf("Resolve() enqueue calls = %v, want %v", enqueueCalls, tt.wantEnqueueCalls)
			}

			if !tt.wantErr && result != nil {
				res := result.(updatetelegrampublishposts.Result)
				successCount := 0
				failedCount := 0
				for _, chat := range res.Chats {
					if chat.Error == nil {
						successCount++
					} else {
						failedCount++
					}
				}
				if successCount != tt.wantSuccessChats {
					t.Errorf("Resolve() success chats = %v, want %v", successCount, tt.wantSuccessChats)
				}
				if failedCount != tt.wantFailedChats {
					t.Errorf("Resolve() failed chats = %v, want %v", failedCount, tt.wantFailedChats)
				}
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
