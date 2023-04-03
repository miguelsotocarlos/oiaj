package oiajudge

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/carlosmiguelsoto/oiajudge/pkg/bridge"
	"github.com/carlosmiguelsoto/oiajudge/pkg/store"
	"github.com/carlosmiguelsoto/oiajudge/pkg/utils"
	"github.com/gorilla/mux"
)

type Server struct {
	Bridge bridge.Bridge
	Config Config
	Db     store.DBClient
}

func WrongJsonInput(expected_type string, err error) *OiaError {
	return &OiaError{
		HttpCode:      http.StatusBadRequest,
		Message:       fmt.Sprintf("could not parse request body as instance of %s", expected_type),
		InternalError: err,
	}
}

type Handler func(w http.ResponseWriter, r *http.Request)
type Authenticator[Q any] func(Q, *http.Request) error
type ApiFunction[Q any, R any] func(context.Context, Q) (R, error)

func Outer[Q any, R any](auth Authenticator[Q], handler ApiFunction[Q, R]) Handler {
	processError := func(w http.ResponseWriter, err error) {
		if err == nil {
			log.Fatalf("ERROR WAS NIL")
		}
		fmt.Printf("ERROR: %s\n\n", err)
		var code int
		var body []byte
		if err2, ok := err.(*OiaError); ok {
			code = err2.HttpCode
			body = []byte(err2.Error())
		} else {
			code = http.StatusInternalServerError
			body = []byte(fmt.Sprintf("Unexpected error: %s", err.Error()))
		}
		w.WriteHeader(code)
		w.Write(body)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")

		var query Q
		err := json.NewDecoder(r.Body).Decode(&query)
		if err != nil {
			processError(w, &OiaError{
				HttpCode:      http.StatusBadRequest,
				Message:       "could not parse json body",
				InternalError: err,
			})
			return
		}
		err = auth(query, r)
		if err != nil {
			processError(w, err)
			return
		}
		resp, err := handler(r.Context(), query)
		if err != nil {
			processError(w, err)
			return
		}
		data, err := json.Marshal(resp)
		if err != nil {
			processError(w, err)
			return
		}
		_, err = w.Write(data)
		if err != nil {
			processError(w, err)
			return
		}
	}
}

func NoAuth[Q any, R any](server *Server, f ApiFunction[Q, R]) Handler {
	return Outer(func(query Q, r *http.Request) error { return nil }, f)
}

func WithUserAuth[Q Authenticatable, R any](server *Server, f ApiFunction[Q, R]) Handler {
	auth := func(query Q, r *http.Request) error {
		uid := query.Uid()
		authHeader := r.Header.Get("Authorization")

		malformedAuthError := &OiaError{
			HttpCode: http.StatusBadRequest,
			Message:  "Authorization header must be of the form `Bearer <token-id>:<token-value>`",
		}
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return malformedAuthError
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		tx, err := server.Db.Tx(r.Context())
		if err != nil {
			return err
		}
		defer tx.Close(&err)
		err = CheckUserToken(*tx, uid, token)
		if err != nil {
			return &OiaError{
				HttpCode:      http.StatusUnauthorized,
				Message:       "Unauthorized",
				InternalError: err,
			}
		}
		return nil
	}
	return Outer(auth, f)
}

func Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func ServeStatement(w http.ResponseWriter, r *http.Request, server *Server) {
	tid_s, ok := mux.Vars(r)["tid"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	tid, err := strconv.ParseInt(tid_s, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	statement, err := server.GetTaskStatement(r.Context(), tid)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Write(statement)
	w.Header().Add("Content-Type", "application/pdf")
}

func (server *Server) MakeServer() http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/{any:.*}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			// Set the CORS headers
			headers := w.Header()
			headers.Set("Access-Control-Allow-Origin", "*")
			headers.Set("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, OPTIONS")
			headers.Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")

			// Return immediately
			return
		}
	}).Methods("OPTIONS")

	r.HandleFunc("/user/create", NoAuth(server, server.CreateUser)).Methods("POST")
	r.HandleFunc("/user/login", NoAuth(server, server.UserLogin)).Methods("POST")
	r.HandleFunc("/user/get", WithUserAuth(server, server.GetUser)).Methods("POST")
	r.HandleFunc("/submissions/get", NoAuth(server, server.GetSubmissions)).Methods("POST")
	r.HandleFunc("/submission/create", WithUserAuth(server, server.MakeSubmission)).Methods("POST")
	r.HandleFunc("/task/get", NoAuth(server, server.GetTasks)).Methods("POST")

	r.HandleFunc("/task/statement/{tid}", func(w http.ResponseWriter, r *http.Request) {
		ServeStatement(w, r, server)
	}).Methods("GET")
	r.HandleFunc("/health", Health).Methods("GET")
	return r
}

//go:embed migrations
var migrations embed.FS

func RunServer(ctx context.Context, bridge bridge.Bridge) error {
	port_string := os.Getenv("OIAJ_SERVER_PORT")
	port, err := strconv.ParseInt(port_string, 10, 64)
	if err != nil {
		return err
	}
	config := Config{
		OiaDbConnectionString: os.Getenv("OIAJ_DB_CONNECTION_STRING"),
		OiaServerPort:         port,
	}

	sql, err := utils.ExtractEmbeddedFsIntoFileMap(migrations, "migrations")
	if err != nil {
		return err
	}
	client, err := store.MakeClientWithInitScript(ctx, config.OiaDbConnectionString, sql, "oiajudge")
	if err != nil {
		return err
	}

	server := &Server{
		Db:     client,
		Bridge: bridge,
		Config: config,
	}

	bridge.HandleEvents(context.Background(), server.HandleEvents)

	handler := server.MakeServer()
	url := fmt.Sprintf(":%d", config.OiaServerPort)
	err = http.ListenAndServe(url, handler)
	return err
}
