package bridge

import "context"

type Id = int64

type Event struct {
	EventId   Id
	ObjectId  Id
	EventType string
	Error     error
}

type Bridge interface {
	HandleEvents(ctx context.Context, handler func(context.Context, Event) error) error
	CreateUser(ctx context.Context, username string) (Id, error)
	GetSubmission(ctx context.Context, submission Id) (*Submission, error)
	GetTask(ctx context.Context, task Id) (*Task, error)
	MakeSubmission(ctx context.Context, uid Id, task_id Id, sources map[string][]byte) error
	GetAttachment(ctx context.Context, tid Id, filename string) ([]byte, error)
}
