package oiajudge

import (
	"context"
)

func (s *Server) HandleTask(ctx context.Context, task_id Id) error {
	task, err := s.Bridge.GetTask(ctx, task_id)
	if err != nil {
		return err
	}
	tx, err := s.Db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Close(&err)
	err = SaveTask(*tx, *task)
	if err != nil {
		return err
	}
	return nil
}
