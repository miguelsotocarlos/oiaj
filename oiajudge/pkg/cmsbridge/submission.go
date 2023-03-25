package cmsbridge

import (
	"database/sql"
	"encoding/json"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/carlosmiguelsoto/oiajudge/pkg/bridge"
	"github.com/carlosmiguelsoto/oiajudge/pkg/store"
)

func GetSubmission(tx store.Transaction, id bridge.Id) (submission *bridge.Submission, err error) {
	submission = &bridge.Submission{Id: id}
	row := tx.QueryRow(`
	SELECT
		task_id,
		user_id,
		timestamp,
		compilation_outcome,
		compilation_text,
		evaluation_outcome,
		score,
		score_details
	FROM submissions
		INNER JOIN participations
			ON participations.id = submissions.participation_id
		LEFT JOIN submission_results
			ON submission_results.submission_id = submissions.id
		WHERE submissions.id = $1
`, id)
	var compilation_outcome sql.NullString
	var compilation_text []string
	var evaluation_outcome sql.NullString
	var score sql.NullFloat64
	var score_details sql.NullString
	err = row.Scan(
		&submission.ProblemId,
		&submission.UserId,
		&submission.Timestamp,
		&compilation_outcome,
		&compilation_text,
		&evaluation_outcome,
		&score,
		&score_details)
	if store.IsNoRows(err) {
		submission.Deleted = true
		return
	}
	if err != nil {
		return
	}

	row = tx.QueryRow(`
			SELECT score_type, score_type_parameters
			FROM tasks
				INNER JOIN datasets ON tasks.id = datasets.task_id
			WHERE tasks.id = $1`,
		submission.ProblemId)
	var score_type string
	var score_type_parameters string
	err = row.Scan(&score_type, &score_type_parameters)
	if err != nil {
		return
	}

	if !compilation_outcome.Valid {
		submission.SubmissionStatus = bridge.COMPILING
		return
	}
	submission.CompilationMessage = strings.Join(compilation_text, "\n")
	if compilation_outcome.String == "fail" {
		submission.SubmissionStatus = bridge.COMPILATION_FAILED
		return
	}
	if !evaluation_outcome.Valid {
		submission.SubmissionStatus = bridge.EVALUATING
	} else {
		submission.SubmissionStatus = bridge.SCORED
	}
	submission.Result = &bridge.SubmissionResult{}

	FillTestcaseResults(tx, submission)

	switch score_type {
	case "Sum":
		FillSubmissionResultsScoreTypeSum(tx, score_details.String, score_type_parameters, submission)
	default:
		FillSubmissionResultsScoreTypeGrouped(tx, score_details.String, score_type_parameters, submission)
	}
	return
}

type Testcase struct {
	Idx string `json:"idx"`
}

func FillSubmissionResultsScoreTypeSum(tx store.Transaction, score_details string, score_type_parameters string, submission *bridge.Submission) (err error) {
	multiplier, err := strconv.ParseFloat(score_type_parameters, 64)
	if err != nil {
		return
	}
	submission.Result.Score.MaxScore = float64(0)
	submission.Result.Score.Score = float64(0)
	for i, t := range submission.Result.Testcases {
		submission.Result.Subtasks = append(submission.Result.Subtasks, bridge.SubtaskResult{
			Subtask: int64(i),
			Score: bridge.Score{
				t.Score.Score * multiplier,
				t.Score.MaxScore * multiplier,
			},
			Testcases: []string{t.Testcase},
		})
		submission.Result.Score.MaxScore += t.Score.MaxScore * multiplier
		submission.Result.Score.Score += t.Score.Score * multiplier
	}
	return
}

func FillSubmissionResultsScoreTypeGrouped(tx store.Transaction, score_details string, score_type_parameters string, submission *bridge.Submission) (err error) {
	type Submission struct {
		Idx           int64   `json:"idx"`
		MaxScore      float64 `json:"max_score"`
		ScoreFraction float64 `json:"score_fraction"`
		Testcases     []Testcase
	}
	var sub []Submission
	log.Printf("UNMARSHALLING: %s", score_details)
	json.Unmarshal([]byte(score_details), &sub)
	submission.Result.Score.MaxScore = float64(0)
	submission.Result.Score.Score = float64(0)
	for _, s := range sub {
		submission.Result.Score.MaxScore += s.MaxScore
		submission.Result.Score.Score += s.MaxScore * s.ScoreFraction
		var ts []string
		for _, t := range s.Testcases {
			ts = append(ts, t.Idx)
		}
		submission.Result.Subtasks = append(submission.Result.Subtasks, bridge.SubtaskResult{
			Subtask: s.Idx,
			Score: bridge.Score{
				MaxScore: s.MaxScore,
				Score:    s.MaxScore * s.ScoreFraction,
			},
			Testcases: ts,
		})
	}
	return
}

func FillTestcaseResults(tx store.Transaction, submission *bridge.Submission) (err error) {
	rows, err := tx.Query(`
	SELECT
		outcome,
		codename,
		text,
		execution_time,
		execution_memory
	FROM evaluations
		INNER JOIN testcases
			ON evaluations.testcase_id = testcases.id
		WHERE submission_id = $1
`, submission.Id)
	if err != nil {
		return
	}
	for rows.Next() {
		var outcome_str sql.NullString
		var codename sql.NullString
		var messages []string
		var execution_time sql.NullFloat64
		var memory_usage sql.NullInt64
		err = rows.Scan(&outcome_str, &codename, &messages, &execution_time, &memory_usage)
		if err != nil {
			return
		}
		outcome, err := parseOutcome(outcome_str)
		if err != nil {
			messages = []string{err.Error()}
		}
		message := strings.Join(messages, "\n")
		submission.Result.Testcases = append(submission.Result.Testcases, bridge.TestcaseResult{
			Testcase: codename.String,
			Message:  message,
			Score: bridge.Score{
				MaxScore: 1.0,
				Score:    outcome,
			},
			ExecutionTime: execution_time.Float64,
			MemoryUsage:   memory_usage.Int64,
		})
	}
	// Sort testcases alphabetically to have a consistent order
	sort.Slice(submission.Result.Testcases, func(i, j int) bool {
		return submission.Result.Testcases[i].Testcase < submission.Result.Testcases[j].Testcase
	})
	return
}