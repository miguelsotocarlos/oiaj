package oiajudge

import (
	"context"
	"net/http"

	"github.com/carlosmiguelsoto/oiajudge/pkg/bridge"
	"golang.org/x/crypto/bcrypt"
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

type CreateUserQuery struct {
	Username string `json:"username"`
	Password string `json:"password"`
	School   string `json:"school"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}

type CreateUserResponse struct {
	UserId Id    `json:"user_id"`
	Token  Token `json:"token"`
}

func (s *Server) CreateUser(ctx context.Context, q CreateUserQuery) (r CreateUserResponse, err error) {
	cms_uid, err := s.Bridge.CreateUser(ctx, q.Username)
	if err != nil {
		return
	}
	tx, err := s.Db.Tx(ctx)
	if err != nil {
		return
	}
	defer tx.Close(&err)
	password_hash, err := bcrypt.GenerateFromPassword([]byte(q.Password), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	uid, err := CreateUser(*tx, q.Email, q.Username, cms_uid, password_hash)
	if err != nil {
		return
	}
	token, err := CreateUserToken(*tx, uid)
	if err != nil {
		return
	}
	r.UserId = uid
	r.Token = token
	return
}

type UserLoginQuery struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserLoginResponse struct {
	UserId Id    `json:"user_id"`
	Token  Token `json:"token"`
}

func (s *Server) UserLogin(ctx context.Context, q UserLoginQuery) (r UserLoginResponse, err error) {
	tx, err := s.Db.Tx(ctx)
	if err != nil {
		return
	}
	defer tx.Close(&err)
	hash, uid, err := GetUserPasswordAndId(*tx, q.Username)
	err = bcrypt.CompareHashAndPassword(hash, []byte(q.Password))
	if err != nil {
		err = &OiaError{
			HttpCode:      http.StatusUnauthorized,
			InternalError: err,
		}
		return
	}
	token, err := CreateUserToken(*tx, uid)
	if err != nil {
		return
	}
	r.UserId = uid
	r.Token = token
	return
}

type GetUserQuery struct {
	UserId Id `json:"user_id"`
}

func (q GetUserQuery) Uid() Id {
	return q.UserId
}

type GetUserResponse struct {
	Username string  `json:"username"`
	Score    float64 `json:"score"`
}

func (s *Server) GetUser(ctx context.Context, q GetUserQuery) (r GetUserResponse, err error) {
	tx, err := s.Db.Tx(ctx)
	if err != nil {
		return
	}
	defer tx.Close(&err)
	user, err := GetUser(*tx, q.UserId)
	if err != nil {
		return
	}
	r.Username = user.Username
	r.Score = user.Score
	return
}

type GetTasksQuery struct{}

type GetTasksResponse struct {
	Tasks []bridge.Task `json:"tasks"`
}

func (s *Server) GetTasks(ctx context.Context, q GetTasksQuery) (r GetTasksResponse, err error) {
	tx, err := s.Db.Tx(ctx)
	if err != nil {
		return
	}
	defer tx.Close(&err)
	tasks, err := GetTasks(*tx)
	if err != nil {
		return
	}
	r.Tasks = tasks
	return
}

func (s *Server) GetTaskStatement(ctx context.Context, tid Id) (statement []byte, err error) {
	tx, err := s.Db.Tx(ctx)
	if err != nil {
		return
	}
	defer tx.Close(&err)
	statement, err = GetTaskStatement(*tx, tid)
	if err != nil {
		return
	}
	return
}
