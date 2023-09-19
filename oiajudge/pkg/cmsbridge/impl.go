package cmsbridge

import (
	"context"
	"log"

	"github.com/carlosmiguelsoto/oiajudge/pkg/bridge"
)

func (b *CmsBridge) GetTask(ctx context.Context, submission bridge.Id) (res *bridge.Task, err error) {
	tx, err := b.Db.Tx(ctx)
	if err != nil {
		return
	}
	defer tx.Close(&err)
	res, err = GetTask(*tx, submission)
	if err != nil {
		return
	}
	return
}

func (b *CmsBridge) HandleEvents(ctx context.Context, handler func(context.Context, bridge.Event) error) error {
	channel, err := GetNotificationChannel(ctx, &b.Db)
	if err != nil {
		return err
	}
	go func() {
		for {
			event, ok := <-channel
			if !ok {
				break
			}
			if event.Error != nil {
				log.Printf("HandleEvents(): got error %s. Ignoring", event.Error)
				continue
			}
			retries := 10
			for i := 0; i < retries; i += 1 {
				err := handler(ctx, event)
				if err == nil {
					break
				}
				if i < retries-1 {
					log.Printf("HandleEvents(): got error %s. Retrying (%d/%d)", err, i+1, retries)
				} else {
					log.Printf("HandleEvents(): got error %s. Retried %d times. Ignoring", err, retries)
				}
			}
			tx, err := b.Db.Tx(ctx)
			if err != nil {
				continue
			}
			err = DeleteNotification(*tx, event.EventId)
			tx.Close(&err)
		}
	}()
	return nil
}

func (b *CmsBridge) CreateUser(ctx context.Context, username string) (uid bridge.Id, err error) {
	tx, err := b.Db.Tx(ctx)
	if err != nil {
		return
	}
	defer tx.Close(&err)
	uid, err = CreateUser(b.Config, *tx, username)
	if err != nil {
		return
	}
	return
}

func (b *CmsBridge) GetSubmission(ctx context.Context, submission bridge.Id) (res *bridge.Submission, err error) {
	tx, err := b.Db.Tx(ctx)
	if err != nil {
		return
	}
	defer tx.Close(&err)
	res, err = GetSubmission(*tx, submission)
	if err != nil {
		return
	}
	return
}

type SubmitQuery struct {
	Uid      bridge.Id         `json:"user_id"`
	Task     bridge.Id         `json:"task_id"`
	Files    map[string][]byte `json:"files"`
	Language string            `json:"language"`
}

func (b *CmsBridge) MakeSubmission(ctx context.Context, uid bridge.Id, task_id bridge.Id, sources map[string][]byte) (err error) {
	tx, err := b.Db.Tx(ctx)
	if err != nil {
		return
	}
	defer tx.Close(&err)
	err = MakeSubmission(*tx, b.Config.CmsContestId, uid, task_id, sources)
	if err != nil {
		return
	}
	return
}

func (b *CmsBridge) GetAttachment(ctx context.Context, tid bridge.Id, filename string) (attachment []byte, err error) {
	tx, err := b.Db.Tx(ctx)
	if err != nil {
		return
	}
	defer tx.Close(&err)
	attachment, err = GetAttachment(*tx, tid, filename)
	if err != nil {
		return
	}
	return
}
