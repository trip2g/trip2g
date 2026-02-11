package handlenotewebhooks_test

import (
	"context"
	"sync"

	"trip2g/internal/case/handlenotewebhooks"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

// mockEnv is a manual mock for the Env interface.
type mockEnv struct {
	mu sync.Mutex

	webhooks  []db.ChangeWebhook
	noteViews *model.NoteViews
	logger    logger.Logger

	deliveries []db.InsertWebhookDeliveryParams
	enqueued   []handlenotewebhooks.DeliverChangeWebhookParams

	listWebhooksErr    error
	insertDeliveryErr  error
	enqueueDeliveryErr error
	nextDeliveryID     int64
}

func newMockEnv() *mockEnv {
	return &mockEnv{
		webhooks:       []db.ChangeWebhook{},
		noteViews:      model.NewNoteViews(),
		logger:         &logger.DummyLogger{},
		deliveries:     []db.InsertWebhookDeliveryParams{},
		enqueued:       []handlenotewebhooks.DeliverChangeWebhookParams{},
		nextDeliveryID: 1,
	}
}

func (m *mockEnv) ListEnabledWebhooks(ctx context.Context) ([]db.ChangeWebhook, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.listWebhooksErr != nil {
		return nil, m.listWebhooksErr
	}

	return m.webhooks, nil
}

func (m *mockEnv) InsertWebhookDelivery(ctx context.Context, arg db.InsertWebhookDeliveryParams) (db.ChangeWebhookDelivery, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.insertDeliveryErr != nil {
		return db.ChangeWebhookDelivery{}, m.insertDeliveryErr
	}

	m.deliveries = append(m.deliveries, arg)

	delivery := db.ChangeWebhookDelivery{
		ID:        m.nextDeliveryID,
		WebhookID: arg.WebhookID,
		Attempt:   arg.Attempt,
	}

	m.nextDeliveryID++

	return delivery, nil
}

func (m *mockEnv) LatestNoteViews() *model.NoteViews {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.noteViews
}

func (m *mockEnv) EnqueueDeliverChangeWebhook(ctx context.Context, params handlenotewebhooks.DeliverChangeWebhookParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.enqueueDeliveryErr != nil {
		return m.enqueueDeliveryErr
	}

	m.enqueued = append(m.enqueued, params)

	return nil
}

func (m *mockEnv) Logger() logger.Logger {
	return m.logger
}

// Test helper methods.

func (m *mockEnv) setWebhooks(webhooks []db.ChangeWebhook) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.webhooks = webhooks
}

func (m *mockEnv) addNote(path string, pathID int64, versionID int64, title string, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	note := &model.NoteView{
		Path:      path,
		PathID:    pathID,
		VersionID: versionID,
		Title:     title,
		Content:   []byte(content),
	}

	m.noteViews.PathMap[path] = note
	m.noteViews.Map[path] = note
}

func (m *mockEnv) getEnqueued() []handlenotewebhooks.DeliverChangeWebhookParams {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]handlenotewebhooks.DeliverChangeWebhookParams, len(m.enqueued))
	copy(result, m.enqueued)

	return result
}
