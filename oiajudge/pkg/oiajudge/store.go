package oiajudge

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/carlosmiguelsoto/oiajudge/pkg/bridge"
	"github.com/carlosmiguelsoto/oiajudge/pkg/store"
)

func CreateUser(tx store.Transaction, email, username string, cms_user_id Id, password_hash []byte) (uid Id, err error) {
	_, err = tx.Exec("INSERT INTO oia_user(id, email, username, password_hash) VALUES ($1, $2, $3, $4)",
		cms_user_id, email, username, password_hash)
	if err != nil {
		return
	}
	uid = cms_user_id
	return
}

func CreateUserToken(tx store.Transaction, uid Id) (t Token, err error) {
	secret := make([]byte, 32)
	_, err = rand.Read(secret)
	if err != nil {
		return
	}
	row := tx.QueryRow("INSERT INTO oia_tokens(user_id, secret) VALUES ($1, $2) RETURNING id",
		uid, secret)
	var id Id
	err = row.Scan(&id)
	if err != nil {
		return
	}
	t = Token(fmt.Sprintf("%d:%s", id, base64.StdEncoding.EncodeToString(secret)))
	return
}

func GetUserPasswordAndId(tx store.Transaction, username string) (hash []byte, id Id, err error) {
	row := tx.QueryRow("SELECT password_hash, id FROM oia_user WHERE username = $1",
		username)
	err = row.Scan(&hash, &id)
	if err != nil {
		return
	}
	return
}

func CheckUserToken(tx store.Transaction, uid Id, token_s string) error {
	v := strings.Split(token_s, ":")
	malformedTokenError := &OiaError{
		HttpCode: http.StatusBadRequest,
		Message:  "malformed token",
	}
	if len(v) != 2 {
		return malformedTokenError
	}
	token_id := v[0]
	token_value := v[1]
	id, err := strconv.ParseInt(token_id, 10, 64)
	if err != nil {
		return malformedTokenError
	}
	value, err := base64.StdEncoding.DecodeString(token_value)
	if err != nil {
		return malformedTokenError
	}

	row := tx.QueryRow("SELECT secret FROM oia_tokens WHERE id = $1 AND user_id = $2", id, uid)
	var secret []byte
	err = row.Scan(&secret)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare(secret, value) == 0 {
		return &OiaError{
			HttpCode: http.StatusForbidden,
			Message:  "invalid credentials",
		}
	}
	return err
}

type DbUser struct {
	Username string
	Score    float64
}

func GetUser(tx store.Transaction, uid Id) (user DbUser, err error) {
	row := tx.QueryRow("SELECT username, score FROM oia_user WHERE id = $1", uid)
	err = row.Scan(&user.Username, &user.Score)
	if err != nil {
		return
	}
	return
}

func LastUserSubmission(tx store.Transaction, uid Id) (t time.Time, err error) {
	row := tx.QueryRow("SELECT last_submission_ms FROM oia_user WHERE id = $1", uid)
	var unixms int64
	err = row.Scan(&unixms)
	if err != nil {
		return
	}
	t = time.UnixMilli(unixms)
	return
}

func SetUserSubmission(tx store.Transaction, uid Id, t time.Time) error {
	_, err := tx.Exec("UPDATE oia_user SET last_submission_ms = $1 WHERE id = $2", t.UnixMilli(), uid)
	if err != nil {
		return err
	}
	return nil
}

func CreateSubmission(tx store.Transaction, submission bridge.Submission) error {
	if submission.Deleted {
		_, err := tx.Exec("DELETE FROM oia_submissions WHERE id = $1", submission.Id)
		if err != nil {
			return err
		}
		return nil
	}

	data, err := json.Marshal(submission)
	if err != nil {
		return err
	}

	subtask_details := make([]float64, 0)
	if submission.Result != nil {
		for _, subtask := range submission.Result.Subtasks {
			subtask_details = append(subtask_details, subtask.Score.Score)
		}
	}
	subtask_details_data, err := json.Marshal(subtask_details)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO oia_submissions(id, user_id, task_id, details, subtask_details)
		VALUES ($4, $1, $2, $3, $5)
		ON CONFLICT(id) DO UPDATE SET
			details=EXCLUDED.details,
			subtask_details=EXCLUDED.subtask_details`,
		submission.UserId, submission.ProblemId, data, submission.Id, subtask_details_data)
	if err != nil {
		return err
	}
	return nil
}

func GetAllScores(tx store.Transaction, uid Id, tid Id) (v [][]float64, err error) {
	rows, err := tx.Query("SELECT subtask_details from oia_submissions WHERE user_id = $1 AND task_id = $2", uid, tid)
	if err != nil {
		return
	}
	for rows.Next() {
		var json_arr string
		err = rows.Scan(&json_arr)
		if err != nil {
			return
		}
		subtask_scores := make([]float64, 0)
		err = json.Unmarshal([]byte(json_arr), &subtask_scores)
		if err != nil {
			return
		}
		v = append(v, subtask_scores)
	}
	return
}

func SaveUserScore(tx store.Transaction, uid Id, tid Id, score float64) (float64, error) {
	row := tx.QueryRow("SELECT multiplier FROM oia_task WHERE oia_task.id = $1", tid)
	var multiplier float64
	err := row.Scan(&multiplier)
	if store.IsNoRows(err) {
		log.Panicf("Processing submission with unkown task. Assumming multiplier = 1.")
		multiplier = 1
	}
	_, err = tx.Exec(`
		INSERT INTO oia_task_score(user_id, task_id, score, base_score)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, task_id)
		DO UPDATE SET
			score=EXCLUDED.score,
			base_score=EXCLUDED.base_score
		`, uid, tid, score*multiplier, score)
	if err != nil {
		return 0, err
	}
	return score * multiplier, nil
}

func GetUserTaskScore(tx store.Transaction, uid, tid Id) (float64, error) {
	row := tx.QueryRow("SELECT score FROM oia_task_score WHERE user_id = $1 AND task_id = $2", uid, tid)
	var score float64
	err := row.Scan(&score)
	if store.IsNoRows(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return score, nil
}

func IncrementUserScore(tx store.Transaction, uid Id, delta float64) error {
	_, err := tx.Exec("UPDATE oia_user SET score = score + $1 WHERE id = $2", delta, uid)
	if err != nil {
		return err
	}
	return nil
}

func SaveTask(tx store.Transaction, task bridge.Task) (err error) {
	_, err = tx.Exec(`
		INSERT INTO oia_task(id, title, name, statement, max_score, multiplier, submission_format, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT(id) DO UPDATE SET
			title = EXCLUDED.title,
			name = EXCLUDED.name,
			statement = EXCLUDED.statement,
			max_score = EXCLUDED.max_score,
			multiplier = EXCLUDED.multiplier,
			tags = EXCLUDED.tags,
			submission_format = EXCLUDED.submission_format;`,
		task.Id, task.Title, task.Name, task.Statement, task.MaxScore, task.Multiplier, task.SubmissionFormat, task.Tags)
	if err != nil {
		return
	}
	return
}

func GetSubmissions(tx store.Transaction, uid Id, tid Id) ([]bridge.Submission, error) {
	rows, err := tx.Query("SELECT details FROM oia_submissions WHERE user_id=$1 AND task_id=$2", uid, tid)
	if err != nil {
		return nil, err
	}
	res := make([]bridge.Submission, 0)
	for rows.Next() {
		var details string
		err = rows.Scan(&details)
		if err != nil {
			return nil, err
		}

		var submission bridge.Submission
		err = json.Unmarshal([]byte(details), &submission)
		if err != nil {
			return nil, err
		}

		res = append(res, submission)
	}
	return res, nil
}

func GetSubmission(tx store.Transaction, sid Id) (submission bridge.Submission, err error) {
	rows := tx.QueryRow("SELECT details FROM oia_submissions WHERE id=$1", sid)

	var details string
	err = rows.Scan(&details)
	if err != nil {
		return
	}

	err = json.Unmarshal([]byte(details), &submission)
	if err != nil {
		return
	}

	return
}

func GetTasks(tx store.Transaction) (tasks []bridge.Task, err error) {
	row, err := tx.Query("SELECT id, name, title, max_score, multiplier, submission_format, tags FROM oia_task")
	if err != nil {
		return
	}
	for row.Next() {
		var task bridge.Task
		err = row.Scan(&task.Id, &task.Name, &task.Title, &task.MaxScore, &task.Multiplier, &task.SubmissionFormat, &task.Tags)
		if err != nil {
			return
		}
		tasks = append(tasks, task)
	}
	return
}

func GetSingleTask(tx store.Transaction, tid Id) (task bridge.Task, err error) {
	row := tx.QueryRow("SELECT id, name, title, max_score, multiplier, submission_format, tags FROM oia_task WHERE id = $1", tid)
	err = row.Scan(&task.Id, &task.Name, &task.Title, &task.MaxScore, &task.Multiplier, &task.SubmissionFormat, &task.Tags)
	if err != nil {
		return
	}
	return
}

func GetTaskStatement(tx store.Transaction, tid Id) (statement []byte, err error) {
	row := tx.QueryRow("SELECT statement FROM oia_task WHERE id = $1", tid)
	err = row.Scan(&statement)
	if err != nil {
		return
	}
	return
}
