package sendpublishpost

import (
	"context"
	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "send_publish_post"
const QueueID = model.BackgroundTelegramJobQueue
const Priority = 1 // shoud process before updates

type SendPublishPostJob struct {
	enqueue jobs.EnqueueFunc
}

func New(env jobs.Env) *SendPublishPostJob {
	return &SendPublishPostJob{
		enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
	}
}

func (t SendPublishPostJob) EnqueueSendPublishPost(ctx context.Context, notePathID int64, instant bool) error {
	return t.enqueue(ctx, Params{NotePathID: notePathID, Instant: instant})
}
