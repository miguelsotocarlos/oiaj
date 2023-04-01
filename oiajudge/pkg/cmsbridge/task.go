package cmsbridge

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/carlosmiguelsoto/oiajudge/pkg/bridge"
	"github.com/carlosmiguelsoto/oiajudge/pkg/store"
)

type OiajTaskEmbeddedData struct {
	Tags       []string
	Multiplier float64
}

func GetTask(tx store.Transaction, taskId bridge.Id) (task *bridge.Task, err error) {
	task = &bridge.Task{}
	row := tx.QueryRow(`
		SELECT name, title, score_type, score_type_parameters, datasets.id, submission_format, datasets.description
		FROM tasks
		INNER JOIN datasets ON datasets.task_id = tasks.id
		WHERE tasks.id = $1
	`, taskId)
	task.Id = taskId
	var score_type string
	var score_parameters string
	var dataset_id bridge.Id
	var description string
	err = row.Scan(&task.Name, &task.Title, &score_type, &score_parameters, &dataset_id, &task.SubmissionFormat, &description)
	if err != nil {
		return
	}

	var embedded_data OiajTaskEmbeddedData
	err = json.Unmarshal([]byte(description), &embedded_data)
	if err != nil {
		log.Printf("Invalid embedded data: %s", description)
		task.Multiplier = 1
		task.Tags = make([]string, 0)
	} else {
		task.Multiplier = embedded_data.Multiplier
		task.Tags = embedded_data.Tags
	}

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
		err = json.Unmarshal([]byte(score_parameters), &params)
		if err != nil {
			return
		}
		task.MaxScore = 0
		for _, params := range params {
			score := params[0]
			switch score := score.(type) {
			case float64:
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
