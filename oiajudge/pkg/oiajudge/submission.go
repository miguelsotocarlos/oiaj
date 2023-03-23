package oiajudge

import (
	"context"
	"log"
	"math"

	"github.com/carlosmiguelsoto/oiajudge/pkg/bridge"
	"github.com/carlosmiguelsoto/oiajudge/pkg/store"
)

type MakeSubmissionQuery struct {
	Task Id `json:"task_id"`
	User Id `json:"user_id"`

	// Submissions can have many files, indexed by filename
	Sources map[string][]byte `json:"sources"`
}

func (q MakeSubmissionQuery) Uid() Id {
	return q.User
}

type MakeSubmissionResponse struct {
	Submission Id `json:"submission"`
}

func (s *Server) MakeSubmission(ctx context.Context, q MakeSubmissionQuery) (r MakeSubmissionResponse, err error) {
	err = s.Bridge.MakeSubmission(ctx, q.User, q.Task, q.Sources)
	if err != nil {
		return
	}
	return
}

type GetSubmissionsQuery struct {
	Task Id `json:"task_id"`
	User Id `json:"user_id"`
}

type GetSubmissionsResponse struct {
	Submissions []bridge.Submission `json:"submissions"`
}

func (s *Server) GetSubmissions(ctx context.Context, q GetSubmissionsQuery) (r GetSubmissionsResponse, err error) {
	tx, err := s.Db.Tx(ctx)
	if err != nil {
		return
	}
	defer tx.Close(&err)
	submissions, err := GetSubmissions(*tx, q.User, q.Task)
	if err != nil {
		return
	}
	r.Submissions = submissions
	return
}

func (s *Server) RecalculateUserScoreForTask(tx store.Transaction, uid Id, tid Id) error {
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
	total_score := float64(0)
	for _, v := range by_subtask {
		total_score += v
	}
	previous_score, err := GetUserTaskScore(tx, uid, tid)
	if err != nil {
		return err
	}
	err = SaveUserScore(tx, uid, tid, total_score)
	if err != nil {
		return err
	}
	err = IncrementUserScore(tx, uid, total_score-previous_score)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) HandleSubmission(ctx context.Context, submission_id Id) error {
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
	err = s.RecalculateUserScoreForTask(*tx, submission.UserId, submission.ProblemId)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) HandleEvents(ctx context.Context, event bridge.Event) error {
	switch event.EventType {
	case "submission":
		return s.HandleSubmission(ctx, event.ObjectId)
	case "task":
		return s.HandleTask(ctx, event.ObjectId)
	default:
		log.Printf("Unkonwn event type %s", event.EventType)
	}
	return nil
}
