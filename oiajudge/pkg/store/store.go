package store

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBClient struct {
	pool *pgxpool.Pool
}

type Transaction struct {
	Ctx    context.Context
	Tx     pgx.Tx
	Conn   *pgxpool.Conn
	Cancel bool
}

func (db *DBClient) Tx(ctx context.Context) (*Transaction, error) {
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		conn.Release()
		return nil, err
	}
	return &Transaction{Ctx: ctx, Tx: tx, Conn: conn}, nil
}

func (db *DBClient) ListenOn(ctx context.Context, channel string) (chan string, error) {
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	_, err = conn.Exec(ctx, "LISTEN event_queue")
	if err != nil {
		return nil, err
	}
	notifications := make(chan string)
	go func() {
		for {
			notification, err := conn.Conn().WaitForNotification(ctx)
			if err != nil {
				break
			}
			notifications <- notification.Payload
		}
		conn.Release()
	}()
	return notifications, nil
}

func (tx *Transaction) Close(func_error *error) (err error) {
	if tx == nil {
		return nil
	}

	if tx.Cancel || (func_error != nil && *func_error != nil) {
		err = tx.Tx.Rollback(tx.Ctx)
	} else {
		err = tx.Tx.Commit(tx.Ctx)
		if func_error != nil {
			*func_error = err
		}
	}
	if err != nil {
		return err
	}
	tx.Conn.Release()
	return
}

func (tx *Transaction) Query(query string, args ...any) (pgx.Rows, error) {
	res, err := tx.Tx.Query(tx.Ctx, query, args...)
	if err != nil {
		return res, fmt.Errorf("Query(): error during query %s: %s", query, err)
	}
	return res, nil
}

type CustomRow struct {
	inner pgx.Row
	query string
}

type CustomError struct {
	InnerError error
	Query      string
}

func (c *CustomError) Error() string {
	return fmt.Sprintf("QueryRow(): error during query %s: %s", c.Query, c.InnerError)
}

func IsNoRows(err error) bool {
	switch err := err.(type) {
	case *CustomError:
		return IsNoRows(err.InnerError)
	default:
		return err == pgx.ErrNoRows
	}
}

func (r CustomRow) Scan(vals ...interface{}) error {
	err := r.inner.Scan(vals...)
	if err != nil {
		return &CustomError{Query: r.query, InnerError: err}
	}
	return nil
}

func (tx *Transaction) QueryRow(query string, args ...any) CustomRow {
	return CustomRow{inner: tx.Tx.QueryRow(tx.Ctx, query, args...), query: query}
}

func (tx *Transaction) Exec(query string, args ...any) (pgconn.CommandTag, error) {
	res, err := tx.Tx.Exec(tx.Ctx, query, args...)
	if err != nil {
		return res, fmt.Errorf("Exec(): error during query %s: %s", query, err)
	}
	return res, nil
}

func MakeClientWithInitScript(ctx context.Context, url string, init_sql map[string]string, appname string) (client DBClient, err error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return
	}
	client.pool = pool

	tx, err := client.Tx(ctx)
	if err != nil {
		return
	}
	defer tx.Close(&err)
	keys := make([]string, 0, len(init_sql))
	for k := range init_sql {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	tx.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s_migrations (version TEXT PRIMARY KEY)", appname))

	for _, k := range keys {
		sql := init_sql[k]
		row := tx.QueryRow(fmt.Sprintf("SELECT version FROM %s_migrations WHERE version = $1", appname), k)
		var version string
		err = row.Scan(&version)
		if IsNoRows(err) {
			parts := strings.Split(sql, ";;")
			for _, part := range parts {
				log.Printf("executing sql: %s", part)
				_, err = tx.Exec(part)
				if err != nil {
					log.Printf("error executing sql: %s", err)
					return
				}
			}
			_, err = tx.Exec(fmt.Sprintf("INSERT INTO %s_migrations(version) VALUES ($1)", appname), k)
			if err != nil {
				return
			}
		} else if err != nil {
			return
		}
	}

	return
}
