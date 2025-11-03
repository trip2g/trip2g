package sendpublishpost

import (
	"context"
	"trip2g/internal/jobs"
)

const ID = "backjobs:send_publish_post"

type SendPublishPostEnv interface {
	Env

	jobs.Env
}

type SendPublishPostJob struct {
	env SendPublishPostEnv
}

func New(env SendPublishPostEnv) *SendPublishPostJob {
	task := SendPublishPostJob{env: env}
	jobs.Register(env, ID, Resolve)
	return &task
}

func (t SendPublishPostJob) QueueSendPublishPost(ctx context.Context, notePathID int64, instant bool) error {
	return t.env.EnqueueJob(ctx, ID, Params{NotePathID: notePathID, Instant: instant})
}
