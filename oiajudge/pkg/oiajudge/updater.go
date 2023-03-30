package oiajudge

import (
	"context"
	"log"
	"math"

	"github.com/carlosmiguelsoto/oiajudge/pkg/bridge"
	"github.com/carlosmiguelsoto/oiajudge/pkg/store"
)

func (s *Server) recalculateUserScoreForTask(tx store.Transaction, uid Id, tid Id) error {
	scores, err := GetAllScores(tx, uid, tid)
	if err != nil {
		return err
	}
	by_subtask := make([]float64, 0)
	for _, v := range scores {
		for len(by_subtask) < len(v) {
			by_subtask = append(by_subtask, 0)
		}
		for i := range v {
			by_subtask[i] = math.Max(by_subtask[i], v[i])
		}
	}
	base_score := float64(0)
	for _, v := range by_subtask {
		base_score += v
	}
	previous_score, err := GetUserTaskScore(tx, uid, tid)
	if err != nil {
		return err
	}
	score, err := SaveUserScore(tx, uid, tid, base_score)
	if err != nil {
		return err
	}
	err = IncrementUserScore(tx, uid, score-previous_score)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) handleSubmission(ctx context.Context, submission_id Id) error {
	submission, err := s.Bridge.GetSubmission(ctx, submission_id)
	if err != nil {
		return err
	}
	tx, err := s.Db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Close(&err)
	err = CreateSubmission(*tx, *submission)
	if err != nil {
		return err
	}
	err = s.recalculateUserScoreForTask(*tx, submission.UserId, submission.ProblemId)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) handleTask(ctx context.Context, task_id Id) error {
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

func (s *Server) HandleEvents(ctx context.Context, event bridge.Event) error {
	switch event.EventType {
	case "submission":
		return s.handleSubmission(ctx, event.ObjectId)
	case "task":
		return s.handleTask(ctx, event.ObjectId)
	default:
		log.Printf("Unkonwn event type %s", event.EventType)
	}
	return nil
}
