package cmsbridge

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/carlosmiguelsoto/oiajudge/pkg/bridge"
	"github.com/carlosmiguelsoto/oiajudge/pkg/store"
)

func DeleteNotification(tx store.Transaction, id bridge.Id) (err error) {
	_, err = tx.Exec("DELETE FROM event_queue WHERE id = $1", id)
	return
}

func parseOutcome(outcome_str sql.NullString) (float64, error) {
	if !outcome_str.Valid {
		return 0.0, fmt.Errorf("outcome doesn't exist")
	}
	outcome, err := strconv.ParseFloat(outcome_str.String, 64)
	if err != nil {
		return 0.0, fmt.Errorf("invalid outcome `%s`", outcome_str.String)
	}
	return outcome, nil
}

func CreateUser(config Config, tx store.Transaction, username string) (d bridge.Id, err error) {
	// We don't use the CMS auth system, so we can safely set a dummy password
	_, err = tx.Exec("INSERT INTO users (username, password, first_name, last_name, preferred_languages) VALUES ($1, $2, $3, $4, $5)", username, "plaintext:dummy", "", "", []string{})
	if err != nil {
		return
	}

	row := tx.QueryRow("SELECT id FROM users WHERE username = $1", username)
	err = row.Scan(&d)
	if err != nil {
		return
	}

	_, err = tx.Exec("INSERT INTO participations (user_id, contest_id, hidden, unrestricted, delay_time, extra_time) VALUES ($1, $2, $3, $4, $5, $6)", d, config.CmsContestId, false, false, time.Duration(0), time.Duration(0))
	if err != nil {
		return
	}

	return
}

func GetNotifications(tx store.Transaction) (v []bridge.Event, err error) {
	rows, err := tx.Query("SELECT id, foreign_id, object_type FROM event_queue WHERE seen = false")
	if err != nil {
		return
	}
	for rows.Next() {
		var event bridge.Event
		err = rows.Scan(&event.EventId, &event.ObjectId, &event.EventType)
		if err != nil {
			return
		}
		v = append(v, event)
	}
	_, err = tx.Exec("UPDATE event_queue SET seen = true")
	if err != nil {
		return
	}
	return
}

func GetNotificationChannel(ctx context.Context, db *store.DBClient) (chan bridge.Event, error) {
	events := make(chan bridge.Event)
	notifs, err := db.ListenOn(ctx, "event_queue")
	if err != nil {
		return nil, err
	}
	push_existing_rows := func() {
		tx, err := db.Tx(ctx)
		if err != nil {
			events <- bridge.Event{Error: err}
			return
		}
		evs, err := GetNotifications(*tx)
		tx.Close(&err)
		if err != nil {
			events <- bridge.Event{Error: err}
			return
		}
		for _, ev := range evs {
			events <- ev
		}
	}
	go func() {
		push_existing_rows()
		for {
			_, ok := <-notifs
			if !ok {
				break
			}
			push_existing_rows()
		}
	}()
	return events, nil
}
