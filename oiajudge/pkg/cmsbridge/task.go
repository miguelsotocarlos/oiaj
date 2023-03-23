package cmsbridge

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/carlosmiguelsoto/oiajudge/pkg/bridge"
	"github.com/carlosmiguelsoto/oiajudge/pkg/store"
)

func GetTask(tx store.Transaction, taskId bridge.Id) (task *bridge.Task, err error) {
	task = &bridge.Task{}
	row := tx.QueryRow(`
		SELECT name, title, score_type, score_type_parameters, datasets.id, submission_format
		FROM tasks
		INNER JOIN datasets ON datasets.task_id = tasks.id
		WHERE tasks.id = $1
	`, taskId)
	task.Id = taskId
	var score_type string
	var score_parameters string
	var dataset_id bridge.Id
	err = row.Scan(&task.Name, &task.Title, &score_type, &score_parameters, &dataset_id, &task.SubmissionFormat)
	if err != nil {
		return
	}

	task.Multiplier = 1
	switch score_type {
	case "Sum":
		var multiplier float64
		multiplier, err = strconv.ParseFloat(score_parameters, 64)
		if err != nil {
			return
		}
		var cases int64
		row = tx.QueryRow(`SELECT count(testcases.id) FROM testcases WHERE dataset_id = $1`, dataset_id)
		err = row.Scan(&cases)
		if err != nil {
			return
		}
		task.MaxScore = multiplier * float64(cases)
	default:
		var params [][]interface{}
		json.Unmarshal([]byte(score_parameters), &params)
		task.MaxScore = 0
		for _, params := range params {
			score := params[0]
			switch score := score.(type) {
			case int:
				task.MaxScore += float64(score)
			default:
				err = fmt.Errorf("invalid score type %s", score)
				return
			}
		}
	}

	rows, err := tx.Query(`
			SELECT pg_largeobject.data
					FROM statements
					INNER JOIN fsobjects ON statements.digest = fsobjects.digest
					INNER JOIN pg_largeobject ON fsobjects.loid = pg_largeobject.loid
					WHERE statements.task_id = $1
					ORDER BY pg_largeobject.pageno ASC;`,
		taskId)
	if err != nil {
		return
	}
	var statement []byte
	for rows.Next() {
		var page []byte
		err = rows.Scan(&page)
		if err != nil {
			return
		}
		statement = append(statement, page...)
	}
	task.Statement = statement
	return
}
